package connpool

import (
	"context"
	"errors"
	"sync"
	"time"

	"pdf-service-go/internal/pkg/metrics"

	"go.uber.org/zap"
)

var (
	ErrPoolClosed      = errors.New("connection pool is closed")
	ErrPoolExhausted   = errors.New("connection pool exhausted")
	ErrConnectionStale = errors.New("connection is stale")
)

// Config содержит настройки пула соединений
type Config struct {
	// MinConns минимальное количество соединений в пуле
	MinConns int
	// MaxConns максимальное количество соединений в пуле
	MaxConns int
	// MaxIdleTime максимальное время простоя соединения
	MaxIdleTime time.Duration
	// MaxLifetime максимальное время жизни соединения
	MaxLifetime time.Duration
	// DialTimeout таймаут на создание соединения
	DialTimeout time.Duration
	// IdleTimeout таймаут на получение соединения из пула
	IdleTimeout time.Duration
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() Config {
	return Config{
		MinConns:    5,
		MaxConns:    50,
		MaxIdleTime: 5 * time.Minute,
		MaxLifetime: 30 * time.Minute,
		DialTimeout: 5 * time.Second,
		IdleTimeout: 30 * time.Second,
	}
}

// Connection представляет соединение в пуле
type Connection struct {
	conn       interface{}
	createdAt  time.Time
	lastUsedAt time.Time
	inUse      bool
	closeOnce  sync.Once
	closeFunc  func() error
}

// GetConn возвращает соединение
func (c *Connection) GetConn() interface{} {
	return c.conn
}

// Pool представляет пул соединений
type Pool struct {
	config     Config
	logger     *zap.Logger
	mu         sync.RWMutex
	conns      []*Connection
	closed     bool
	dialFunc   func(context.Context) (interface{}, func() error, error)
	cleanupCtx context.Context
	cancel     context.CancelFunc
}

// NewPool создает новый пул соединений
func NewPool(config Config, logger *zap.Logger, dialFunc func(context.Context) (interface{}, func() error, error)) *Pool {
	ctx, cancel := context.WithCancel(context.Background())

	p := &Pool{
		config:     config,
		logger:     logger,
		dialFunc:   dialFunc,
		cleanupCtx: ctx,
		cancel:     cancel,
	}

	// Создаем минимальное количество соединений
	for i := 0; i < config.MinConns; i++ {
		if err := p.createConnection(); err != nil {
			logger.Error("failed to create initial connection", zap.Error(err))
		}
	}

	// Запускаем горутину для очистки старых соединений
	go p.cleanup()

	return p
}

// Get получает соединение из пула
func (p *Pool) Get(ctx context.Context) (*Connection, error) {
	start := time.Now()
	defer func() {
		metrics.ConnectionPoolGetDuration.Observe(time.Since(start).Seconds())
	}()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, ErrPoolClosed
	}

	// Ищем свободное соединение
	for _, conn := range p.conns {
		if !conn.inUse && !p.isStale(conn) {
			conn.inUse = true
			conn.lastUsedAt = time.Now()
			metrics.ConnectionPoolActiveConnections.Set(float64(p.activeConnections()))
			return conn, nil
		}
	}

	// Если есть место для нового соединения, создаем его
	if len(p.conns) < p.config.MaxConns {
		conn, err := p.createConnectionLocked()
		if err != nil {
			metrics.ConnectionPoolErrors.WithLabelValues("create").Inc()
			return nil, err
		}
		metrics.ConnectionPoolActiveConnections.Set(float64(p.activeConnections()))
		return conn, nil
	}

	// Ждем освобождения соединения
	timer := time.NewTimer(p.config.IdleTimeout)
	defer timer.Stop()

	p.mu.Unlock()
	select {
	case <-ctx.Done():
		p.mu.Lock()
		metrics.ConnectionPoolErrors.WithLabelValues("timeout").Inc()
		return nil, ctx.Err()
	case <-timer.C:
		p.mu.Lock()
		metrics.ConnectionPoolErrors.WithLabelValues("exhausted").Inc()
		return nil, ErrPoolExhausted
	}
}

// Put возвращает соединение в пул
func (p *Pool) Put(conn *Connection) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		conn.closeOnce.Do(func() {
			if err := conn.closeFunc(); err != nil {
				p.logger.Error("failed to close connection", zap.Error(err))
			}
		})
		return
	}

	conn.inUse = false
	conn.lastUsedAt = time.Now()
	metrics.ConnectionPoolActiveConnections.Set(float64(p.activeConnections()))
}

// Close закрывает пул и все соединения
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	p.cancel()

	var lastErr error
	for _, conn := range p.conns {
		conn.closeOnce.Do(func() {
			if err := conn.closeFunc(); err != nil {
				lastErr = err
				p.logger.Error("failed to close connection", zap.Error(err))
			}
		})
	}

	p.conns = nil
	metrics.ConnectionPoolActiveConnections.Set(0)
	return lastErr
}

// Stats возвращает статистику пула
func (p *Pool) Stats() Stats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	active := 0
	for _, conn := range p.conns {
		if conn.inUse {
			active++
		}
	}

	return Stats{
		TotalConnections:  len(p.conns),
		ActiveConnections: active,
		IdleConnections:   len(p.conns) - active,
	}
}

// Stats содержит статистику пула
type Stats struct {
	TotalConnections  int
	ActiveConnections int
	IdleConnections   int
}

// createConnection создает новое соединение
func (p *Pool) createConnection() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, err := p.createConnectionLocked()
	return err
}

// createConnectionLocked создает новое соединение (должен вызываться с блокировкой)
func (p *Pool) createConnectionLocked() (*Connection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.config.DialTimeout)
	defer cancel()

	conn, closeFunc, err := p.dialFunc(ctx)
	if err != nil {
		return nil, err
	}

	c := &Connection{
		conn:       conn,
		createdAt:  time.Now(),
		lastUsedAt: time.Now(),
		inUse:      true,
		closeFunc:  closeFunc,
	}

	p.conns = append(p.conns, c)
	metrics.ConnectionPoolTotalConnections.Set(float64(len(p.conns)))
	return c, nil
}

// cleanup периодически очищает старые соединения
func (p *Pool) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.cleanupCtx.Done():
			return
		case <-ticker.C:
			p.removeStaleConnections()
		}
	}
}

// removeStaleConnections удаляет старые соединения
func (p *Pool) removeStaleConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()

	var remaining []*Connection
	for _, conn := range p.conns {
		if !conn.inUse && p.isStale(conn) {
			conn.closeOnce.Do(func() {
				if err := conn.closeFunc(); err != nil {
					p.logger.Error("failed to close stale connection", zap.Error(err))
				}
			})
			metrics.ConnectionPoolRemovedConnections.WithLabelValues("stale").Inc()
			continue
		}
		remaining = append(remaining, conn)
	}

	// Проверяем, нужно ли создать новые соединения
	if len(remaining) < p.config.MinConns {
		needed := p.config.MinConns - len(remaining)
		for i := 0; i < needed; i++ {
			if conn, err := p.createConnectionLocked(); err != nil {
				p.logger.Error("failed to create replacement connection", zap.Error(err))
			} else {
				conn.inUse = false
				metrics.ConnectionPoolCreatedConnections.Inc()
			}
		}
	}

	p.conns = remaining
	metrics.ConnectionPoolTotalConnections.Set(float64(len(p.conns)))
}

// isStale проверяет, является ли соединение устаревшим
func (p *Pool) isStale(conn *Connection) bool {
	now := time.Now()
	return now.Sub(conn.createdAt) > p.config.MaxLifetime ||
		now.Sub(conn.lastUsedAt) > p.config.MaxIdleTime
}

// activeConnections возвращает количество активных соединений
func (p *Pool) activeConnections() int {
	active := 0
	for _, conn := range p.conns {
		if conn.inUse {
			active++
		}
	}
	return active
}
