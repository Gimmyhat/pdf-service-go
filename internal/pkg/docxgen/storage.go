package docxgen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var (
	storageOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "docx_storage_operations_total",
			Help: "Total number of storage operations",
		},
		[]string{"storage_type", "operation", "status"},
	)

	storageSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "docx_storage_size_bytes",
			Help: "Current size of storage in bytes",
		},
		[]string{"storage_type"},
	)

	storageFallbackTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "docx_storage_fallback_total",
			Help: "Total number of fallbacks to disk storage",
		},
	)
)

// StorageType определяет тип хранилища
type StorageType string

const (
	MemoryStorage StorageType = "memory"
	DiskStorage   StorageType = "disk"
)

// Storage интерфейс для хранилища временных файлов
type Storage interface {
	CreateTemp(ctx context.Context, pattern string) (*os.File, error)
	Cleanup(ctx context.Context) error
	GetSize() int64
}

// StorageConfig конфигурация для хранилища
type StorageConfig struct {
	Dir           string
	CleanupPeriod time.Duration
	MaxSize       int64
}

// StorageManager управляет primary и fallback хранилищами
type StorageManager struct {
	primary  Storage
	fallback Storage
	logger   *zap.Logger
	maxSize  int64
	mu       sync.RWMutex
}

// NewStorageManager создает новый менеджер хранилища
func NewStorageManager(primaryConfig, fallbackConfig StorageConfig, logger *zap.Logger) (*StorageManager, error) {
	primary, err := newMemoryStorage(primaryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create primary storage: %w", err)
	}

	fallback, err := newDiskStorage(fallbackConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create fallback storage: %w", err)
	}

	return &StorageManager{
		primary:  primary,
		fallback: fallback,
		logger:   logger,
		maxSize:  primaryConfig.MaxSize,
	}, nil
}

// CreateTemp создает временный файл, используя fallback при необходимости
func (sm *StorageManager) CreateTemp(ctx context.Context, pattern string) (*os.File, error) {
	sm.mu.RLock()
	primarySize := sm.primary.GetSize()
	sm.mu.RUnlock()

	// Пробуем создать в primary storage
	if primarySize < sm.maxSize {
		file, err := sm.primary.CreateTemp(ctx, pattern)
		if err == nil {
			return file, nil
		}
		sm.logger.Warn("Failed to create temp file in primary storage, falling back to disk",
			zap.Error(err))
		storageFallbackTotal.Inc()
	}

	// Используем fallback storage
	return sm.fallback.CreateTemp(ctx, pattern)
}

// memoryStorage реализует хранение в памяти
type memoryStorage struct {
	dir         string
	maxSize     int64
	currentSize int64
	files       map[string]time.Time
	mu          sync.RWMutex
}

func newMemoryStorage(config StorageConfig) (*memoryStorage, error) {
	dir := filepath.Clean(config.Dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create memory storage directory: %w", err)
	}

	return &memoryStorage{
		dir:     dir,
		maxSize: config.MaxSize,
		files:   make(map[string]time.Time),
	}, nil
}

func (ms *memoryStorage) CreateTemp(ctx context.Context, pattern string) (*os.File, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	file, err := os.CreateTemp(ms.dir, pattern)
	if err != nil {
		storageOperations.WithLabelValues("memory", "create", "error").Inc()
		return nil, err
	}

	ms.files[file.Name()] = time.Now()
	info, err := file.Stat()
	if err == nil {
		ms.currentSize += info.Size()
		storageSize.WithLabelValues("memory").Set(float64(ms.currentSize))
	}
	storageOperations.WithLabelValues("memory", "create", "success").Inc()
	return file, nil
}

func (ms *memoryStorage) Cleanup(ctx context.Context) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	for path, created := range ms.files {
		if now.Sub(created) > 15*time.Minute {
			if info, err := os.Stat(path); err == nil {
				ms.currentSize -= info.Size()
			}
			if err := os.Remove(path); err != nil {
				storageOperations.WithLabelValues("memory", "cleanup", "error").Inc()
				continue
			}
			delete(ms.files, path)
			storageOperations.WithLabelValues("memory", "cleanup", "success").Inc()
		}
	}
	storageSize.WithLabelValues("memory").Set(float64(ms.currentSize))
	return nil
}

func (ms *memoryStorage) GetSize() int64 {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.currentSize
}

// diskStorage реализует хранение на диске
type diskStorage struct {
	dir         string
	maxSize     int64
	currentSize int64
	files       map[string]time.Time
	mu          sync.RWMutex
}

func newDiskStorage(config StorageConfig) (*diskStorage, error) {
	dir := filepath.Clean(config.Dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create disk storage directory: %w", err)
	}

	return &diskStorage{
		dir:     dir,
		maxSize: config.MaxSize,
		files:   make(map[string]time.Time),
	}, nil
}

func (ds *diskStorage) CreateTemp(ctx context.Context, pattern string) (*os.File, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	file, err := os.CreateTemp(ds.dir, pattern)
	if err != nil {
		storageOperations.WithLabelValues("disk", "create", "error").Inc()
		return nil, err
	}

	ds.files[file.Name()] = time.Now()
	info, err := file.Stat()
	if err == nil {
		ds.currentSize += info.Size()
		storageSize.WithLabelValues("disk").Set(float64(ds.currentSize))
	}
	storageOperations.WithLabelValues("disk", "create", "success").Inc()
	return file, nil
}

func (ds *diskStorage) Cleanup(ctx context.Context) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	now := time.Now()
	for path, created := range ds.files {
		if now.Sub(created) > 30*time.Minute {
			if info, err := os.Stat(path); err == nil {
				ds.currentSize -= info.Size()
			}
			if err := os.Remove(path); err != nil {
				storageOperations.WithLabelValues("disk", "cleanup", "error").Inc()
				continue
			}
			delete(ds.files, path)
			storageOperations.WithLabelValues("disk", "cleanup", "success").Inc()
		}
	}
	storageSize.WithLabelValues("disk").Set(float64(ds.currentSize))
	return nil
}

func (ds *diskStorage) GetSize() int64 {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.currentSize
}
