package gotenberg

import (
	"context"
	"net/http"
	"time"

	"pdf-service-go/internal/pkg/connpool"
	"pdf-service-go/internal/pkg/logger"
)

// ClientWithPool представляет клиент Gotenberg с пулом соединений
type ClientWithPool struct {
	baseURL string
	pool    *connpool.Pool
}

// NewClientWithPool создает нового клиента с пулом соединений
func NewClientWithPool(baseURL string) *ClientWithPool {
	config := connpool.DefaultConfig()

	// Настраиваем конфигурацию пула из переменных окружения
	config.MinConns = getEnvIntWithDefault("GOTENBERG_POOL_MIN_CONNS", config.MinConns)
	config.MaxConns = getEnvIntWithDefault("GOTENBERG_POOL_MAX_CONNS", config.MaxConns)
	config.MaxIdleTime = getEnvDurationWithDefault("GOTENBERG_POOL_MAX_IDLE_TIME", config.MaxIdleTime)
	config.MaxLifetime = getEnvDurationWithDefault("GOTENBERG_POOL_MAX_LIFETIME", config.MaxLifetime)
	config.DialTimeout = getEnvDurationWithDefault("GOTENBERG_POOL_DIAL_TIMEOUT", config.DialTimeout)
	config.IdleTimeout = getEnvDurationWithDefault("GOTENBERG_POOL_IDLE_TIMEOUT", config.IdleTimeout)

	// Создаем функцию для создания новых HTTP клиентов
	dialFunc := func(ctx context.Context) (interface{}, func() error, error) {
		transport := &http.Transport{
			MaxIdleConns:        1,
			MaxIdleConnsPerHost: 1,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
			ForceAttemptHTTP2:   true,
			WriteBufferSize:     64 * 1024,
			ReadBufferSize:      64 * 1024,
		}

		client := &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		}

		return client, func() error {
			transport.CloseIdleConnections()
			return nil
		}, nil
	}

	pool := connpool.NewPool(config, logger.Log, dialFunc)

	return &ClientWithPool{
		baseURL: baseURL,
		pool:    pool,
	}
}

// ConvertDocxToPDF конвертирует DOCX в PDF используя соединение из пула
func (c *ClientWithPool) ConvertDocxToPDF(docxPath string) ([]byte, error) {
	// Получаем соединение из пула
	conn, err := c.pool.Get(context.Background())
	if err != nil {
		return nil, err
	}
	defer c.pool.Put(conn)

	// Используем базовый клиент для конвертации
	client := &Client{
		baseURL: c.baseURL,
		client:  conn.GetConn().(*http.Client),
	}

	return client.ConvertDocxToPDF(docxPath)
}

// HealthCheck выполняет проверку здоровья сервиса
func (c *ClientWithPool) HealthCheck() error {
	conn, err := c.pool.Get(context.Background())
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	client := &Client{
		baseURL: c.baseURL,
		client:  conn.GetConn().(*http.Client),
	}

	return client.HealthCheck(true)
}

// Close закрывает пул соединений
func (c *ClientWithPool) Close() error {
	return c.pool.Close()
}

// Stats возвращает статистику пула соединений
func (c *ClientWithPool) Stats() connpool.Stats {
	return c.pool.Stats()
}

func (c *ClientWithPool) newClient(conn *connpool.Connection) *Client {
	return &Client{
		baseURL: c.baseURL,
		client:  conn.GetConn().(*http.Client),
	}
}
