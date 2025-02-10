package docxgen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/tracing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var (
	tempFileCreations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "docx_temp_file_creations_total",
			Help: "Total number of temporary files created",
		},
		[]string{"type"},
	)

	tempFileErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "docx_temp_file_errors_total",
			Help: "Total number of temporary file errors",
		},
		[]string{"operation"},
	)

	tempFileCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "docx_temp_files_current",
			Help: "Current number of active temporary files",
		},
	)

	tempFileAge = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "docx_temp_file_age_seconds",
			Help:    "Age of temporary files when removed",
			Buckets: []float64{1, 5, 10, 30, 60, 300, 600},
		},
		[]string{"status"},
	)

	// Новые метрики
	tempDirSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "docx_temp_dir_size_bytes",
			Help: "Current size of temporary directory in bytes",
		},
	)

	tempFileSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "docx_temp_file_size_bytes",
			Help:    "Size of temporary files",
			Buckets: prometheus.ExponentialBuckets(1024, 2, 10), // от 1KB до 1GB
		},
		[]string{"type"},
	)

	tempMemoryUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "docx_temp_memory_usage_bytes",
			Help: "Current memory usage by temporary files",
		},
	)

	tempMemoryLimit = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "docx_temp_memory_limit_bytes",
			Help: "Memory limit for temporary files",
		},
	)
)

// FileInfo содержит информацию о временном файле
type FileInfo struct {
	Path      string
	Size      int64
	CreatedAt time.Time
	LastUsed  time.Time
	InUse     bool
}

// TempManager управляет временными файлами
type TempManager struct {
	storageManager *StorageManager
	cleanupPeriod  time.Duration
}

// TempManagerConfig содержит настройки для TempManager
type TempManagerConfig struct {
	Dir           string
	CleanupPeriod time.Duration
	MaxDirSize    int64 // в байтах
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() TempManagerConfig {
	return TempManagerConfig{
		Dir:           "/tmp",            // В k8s это будет примонтированный emptyDir с medium: Memory
		CleanupPeriod: 15 * time.Second,  // Уменьшаем период очистки для memory-based storage
		MaxDirSize:    512 * 1024 * 1024, // 512MB - соответствует лимиту в k8s
	}
}

// NewTempManager создает новый менеджер временных файлов
func NewTempManager(config TempManagerConfig) (*TempManager, error) {
	// Создаем конфигурации для primary и fallback хранилищ
	primaryConfig := StorageConfig{
		Dir:           filepath.Join(config.Dir, "memory"),
		CleanupPeriod: config.CleanupPeriod,
		MaxSize:       config.MaxDirSize,
	}

	tempMemoryLimit.Set(float64(config.MaxDirSize))

	fallbackConfig := StorageConfig{
		Dir:           filepath.Join(config.Dir, "disk"),
		CleanupPeriod: config.CleanupPeriod,
		MaxSize:       config.MaxDirSize * 2, // Для fallback даем больше места
	}

	// Создаем StorageManager
	storageManager, err := NewStorageManager(primaryConfig, fallbackConfig, logger.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage manager: %w", err)
	}

	tm := &TempManager{
		storageManager: storageManager,
		cleanupPeriod:  config.CleanupPeriod,
	}

	go tm.startCleanup()
	return tm, nil
}

// CreateTemp создает временный файл
func (tm *TempManager) CreateTemp(ctx context.Context, pattern string) (*os.File, error) {
	ctx, span := tracing.StartSpan(ctx, "TempManager.CreateTemp")
	defer span.End()

	span.SetAttributes(attribute.String("temp.pattern", pattern))

	file, err := tm.storageManager.CreateTemp(ctx, pattern)
	if err != nil {
		tracing.RecordError(ctx, err)
		tempFileErrors.WithLabelValues("create").Inc()
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	tempFileCreations.WithLabelValues("temp").Inc()
	tempFileCount.Inc()

	span.SetAttributes(attribute.String("temp.file", file.Name()))
	span.AddEvent("Temporary file created")

	return file, nil
}

// Cleanup удаляет старые временные файлы
func (tm *TempManager) Cleanup(ctx context.Context) error {
	ctx, span := tracing.StartSpan(ctx, "TempManager.Cleanup")
	defer span.End()

	if err := tm.storageManager.primary.Cleanup(ctx); err != nil {
		tracing.RecordError(ctx, err)
		tempFileAge.WithLabelValues("error").Observe(float64(tm.cleanupPeriod.Seconds()))
		return fmt.Errorf("failed to cleanup primary storage: %w", err)
	}

	if err := tm.storageManager.fallback.Cleanup(ctx); err != nil {
		tracing.RecordError(ctx, err)
		tempFileAge.WithLabelValues("error").Observe(float64(tm.cleanupPeriod.Seconds()))
		return fmt.Errorf("failed to cleanup fallback storage: %w", err)
	}

	tempFileAge.WithLabelValues("success").Observe(float64(tm.cleanupPeriod.Seconds()))
	span.AddEvent("Cleanup completed")
	return nil
}

// startCleanup запускает периодическую очистку
func (tm *TempManager) startCleanup() {
	ticker := time.NewTicker(tm.cleanupPeriod)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		if err := tm.Cleanup(ctx); err != nil {
			logger.Log.Error("Failed to cleanup temporary files", zap.Error(err))
		}
	}
}

// ForceCleanup выполняет принудительную очистку
func (tm *TempManager) ForceCleanup(ctx context.Context) error {
	ctx, span := tracing.StartSpan(ctx, "TempManager.ForceCleanup")
	defer span.End()

	if err := tm.storageManager.primary.Cleanup(ctx); err != nil {
		tracing.RecordError(ctx, err)
		return fmt.Errorf("failed to force cleanup primary storage: %w", err)
	}

	if err := tm.storageManager.fallback.Cleanup(ctx); err != nil {
		tracing.RecordError(ctx, err)
		return fmt.Errorf("failed to force cleanup fallback storage: %w", err)
	}

	span.AddEvent("Force cleanup completed")
	return nil
}

// MarkFileInUse помечает файл как используемый
func (tm *TempManager) MarkFileInUse(path string) {
	// Эта функциональность теперь обрабатывается в StorageManager
	// Метод оставлен для обратной совместимости
	logger.Log.Debug("MarkFileInUse is deprecated, file usage is managed by StorageManager")
}

// MarkFileNotInUse помечает файл как неиспользуемый
func (tm *TempManager) MarkFileNotInUse(path string) {
	// Эта функциональность теперь обрабатывается в StorageManager
	// Метод оставлен для обратной совместимости
	logger.Log.Debug("MarkFileNotInUse is deprecated, file usage is managed by StorageManager")
}

// UpdateFileSize обновляет размер файла
func (tm *TempManager) UpdateFileSize(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	tempFileSize.WithLabelValues("update").Observe(float64(info.Size()))
	return nil
}

// GetStorageStatus возвращает информацию о состоянии хранилища
func (tm *TempManager) GetStorageStatus() (primarySize, fallbackSize int64) {
	primarySize = tm.storageManager.primary.GetSize()
	fallbackSize = tm.storageManager.fallback.GetSize()

	// Обновляем метрики
	tempMemoryUsage.Set(float64(primarySize))
	tempDirSize.Set(float64(fallbackSize))

	return primarySize, fallbackSize
}

// IsMemoryStorage проверяет, является ли путь memory-based хранилищем
func (tm *TempManager) IsMemoryStorage(path string) bool {
	return isMemoryBasedFS(path)
}
