package docxgen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"pdf-service-go/internal/pkg/tracing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
	dir           string
	cleanupPeriod time.Duration
	maxDirSize    int64        // максимальный размер временной директории
	files         sync.Map     // map[string]*FileInfo
	mu            sync.RWMutex // для атомарного обновления размера директории
	currentSize   int64        // текущий размер всех файлов
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
		Dir:           os.TempDir(),
		CleanupPeriod: 30 * time.Second,
		MaxDirSize:    1024 * 1024 * 1024, // 1GB
	}
}

// NewTempManager создает новый менеджер временных файлов
func NewTempManager(config TempManagerConfig) (*TempManager, error) {
	if err := os.MkdirAll(config.Dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	tm := &TempManager{
		dir:           config.Dir,
		cleanupPeriod: config.CleanupPeriod,
		maxDirSize:    config.MaxDirSize,
	}

	// Инициализация текущего размера директории
	if err := tm.updateDirSize(); err != nil {
		return nil, fmt.Errorf("failed to calculate initial directory size: %w", err)
	}

	go tm.startCleanup()
	return tm, nil
}

// updateDirSize обновляет информацию о текущем размере директории
func (tm *TempManager) updateDirSize() error {
	var totalSize int64
	err := filepath.Walk(tm.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return err
	}

	tm.mu.Lock()
	tm.currentSize = totalSize
	tm.mu.Unlock()

	tempDirSize.Set(float64(totalSize))
	return nil
}

// CreateTemp создает временный файл
func (tm *TempManager) CreateTemp(ctx context.Context, pattern string) (*os.File, error) {
	ctx, span := tracing.StartSpan(ctx, "TempManager.CreateTemp")
	defer span.End()

	span.SetAttributes(
		attribute.String("temp.dir", tm.dir),
		attribute.String("temp.pattern", pattern),
	)

	// Проверяем, не превышен ли лимит размера директории
	tm.mu.RLock()
	if tm.currentSize >= tm.maxDirSize {
		tm.mu.RUnlock()
		err := fmt.Errorf("temp directory size limit exceeded: %d >= %d", tm.currentSize, tm.maxDirSize)
		tempFileErrors.WithLabelValues("size_limit").Inc()
		tracing.RecordError(ctx, err)
		return nil, err
	}
	tm.mu.RUnlock()

	// Генерируем уникальный префикс на основе времени и случайности
	prefix := fmt.Sprintf("%d-%x-", time.Now().UnixNano(), time.Now().Nanosecond())

	// Добавляем префикс к шаблону
	if pattern == "" {
		pattern = "tmp-*"
	}
	fullPattern := prefix + pattern

	file, err := os.CreateTemp(tm.dir, fullPattern)
	if err != nil {
		tempFileErrors.WithLabelValues("create").Inc()
		tracing.RecordError(ctx, err)
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	// Создаем информацию о файле
	fileInfo := &FileInfo{
		Path:      file.Name(),
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		InUse:     true,
	}

	// Сохраняем информацию о файле
	tm.files.Store(file.Name(), fileInfo)

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

	threshold := time.Now().Add(-tm.cleanupPeriod)
	var deletedCount int
	var deletedSize int64

	// Собираем список файлов для удаления
	var filesToDelete []*FileInfo
	tm.files.Range(func(key, value interface{}) bool {
		fileInfo := value.(*FileInfo)

		// Проверяем, не используется ли файл
		if fileInfo.InUse {
			return true
		}

		// Проверяем время последнего использования
		if fileInfo.LastUsed.Before(threshold) {
			filesToDelete = append(filesToDelete, fileInfo)
		}
		return true
	})

	// Сортируем файлы по времени последнего использования (самые старые в начале)
	sort.Slice(filesToDelete, func(i, j int) bool {
		return filesToDelete[i].LastUsed.Before(filesToDelete[j].LastUsed)
	})

	// Удаляем файлы
	for _, fileInfo := range filesToDelete {
		// Проверяем существование файла
		if _, err := os.Stat(fileInfo.Path); os.IsNotExist(err) {
			tm.files.Delete(fileInfo.Path)
			continue
		}

		age := time.Since(fileInfo.CreatedAt).Seconds()
		if err := os.Remove(fileInfo.Path); err != nil {
			tempFileErrors.WithLabelValues("cleanup").Inc()
			tracing.RecordError(ctx, fmt.Errorf("failed to remove file %s: %w", fileInfo.Path, err))
			continue
		}

		deletedSize += fileInfo.Size
		deletedCount++

		// Обновляем метрики
		tempFileAge.WithLabelValues("deleted").Observe(age)
		tempFileCount.Dec()

		// Удаляем информацию о файле
		tm.files.Delete(fileInfo.Path)
	}

	// Атомарно обновляем общий размер директории
	if deletedSize > 0 {
		tm.mu.Lock()
		tm.currentSize -= deletedSize
		tm.mu.Unlock()
		tempDirSize.Set(float64(tm.currentSize))
	}

	span.SetAttributes(
		attribute.Int("temp.files.deleted", deletedCount),
		attribute.Int64("temp.bytes.deleted", deletedSize),
	)
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
			// Логируем ошибку, но продолжаем работу
			span := trace.SpanFromContext(ctx)
			span.RecordError(err)
		}

		// Принудительная очистка при превышении лимита
		tm.mu.RLock()
		if tm.currentSize >= tm.maxDirSize {
			tm.mu.RUnlock()
			if err := tm.ForceCleanup(ctx); err != nil {
				span := trace.SpanFromContext(ctx)
				span.RecordError(err)
			}
		} else {
			tm.mu.RUnlock()
		}
	}
}

// ForceCleanup выполняет принудительную очистку при превышении лимита размера
func (tm *TempManager) ForceCleanup(ctx context.Context) error {
	ctx, span := tracing.StartSpan(ctx, "TempManager.ForceCleanup")
	defer span.End()

	// Собираем все неиспользуемые файлы
	var filesToDelete []*FileInfo
	tm.files.Range(func(key, value interface{}) bool {
		fileInfo := value.(*FileInfo)
		if !fileInfo.InUse {
			filesToDelete = append(filesToDelete, fileInfo)
		}
		return true
	})

	// Сортируем файлы по размеру (самые большие в начале)
	sort.Slice(filesToDelete, func(i, j int) bool {
		return filesToDelete[i].Size > filesToDelete[j].Size
	})

	var deletedSize int64
	var deletedCount int

	// Удаляем файлы, пока размер директории не станет меньше лимита
	for _, fileInfo := range filesToDelete {
		tm.mu.RLock()
		if tm.currentSize < tm.maxDirSize {
			tm.mu.RUnlock()
			break
		}
		tm.mu.RUnlock()

		if err := os.Remove(fileInfo.Path); err != nil {
			if !os.IsNotExist(err) {
				tempFileErrors.WithLabelValues("force_cleanup").Inc()
				tracing.RecordError(ctx, fmt.Errorf("failed to remove file %s: %w", fileInfo.Path, err))
			}
			continue
		}

		deletedSize += fileInfo.Size
		deletedCount++

		// Обновляем метрики
		tempFileAge.WithLabelValues("force_deleted").Observe(time.Since(fileInfo.CreatedAt).Seconds())
		tempFileCount.Dec()

		// Удаляем информацию о файле
		tm.files.Delete(fileInfo.Path)
	}

	// Атомарно обновляем общий размер директории
	if deletedSize > 0 {
		tm.mu.Lock()
		tm.currentSize -= deletedSize
		tm.mu.Unlock()
		tempDirSize.Set(float64(tm.currentSize))
	}

	span.SetAttributes(
		attribute.Int("temp.files.force_deleted", deletedCount),
		attribute.Int64("temp.bytes.force_deleted", deletedSize),
	)
	span.AddEvent("Force cleanup completed")

	return nil
}

// MarkFileInUse помечает файл как используемый
func (tm *TempManager) MarkFileInUse(path string) {
	if info, ok := tm.files.Load(path); ok {
		fileInfo := info.(*FileInfo)
		fileInfo.LastUsed = time.Now()
		fileInfo.InUse = true
		tm.files.Store(path, fileInfo)
	}
}

// MarkFileNotInUse помечает файл как неиспользуемый
func (tm *TempManager) MarkFileNotInUse(path string) {
	if info, ok := tm.files.Load(path); ok {
		fileInfo := info.(*FileInfo)
		fileInfo.InUse = false
		tm.files.Store(path, fileInfo)
	}
}

// UpdateFileSize обновляет размер файла
func (tm *TempManager) UpdateFileSize(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if fileInfoInterface, ok := tm.files.Load(path); ok {
		fileInfo := fileInfoInterface.(*FileInfo)
		oldSize := fileInfo.Size
		fileInfo.Size = info.Size()

		// Атомарно обновляем общий размер директории
		tm.mu.Lock()
		tm.currentSize = tm.currentSize - oldSize + info.Size()
		tm.mu.Unlock()

		tempDirSize.Set(float64(tm.currentSize))
		tempFileSize.WithLabelValues("update").Observe(float64(info.Size()))

		// Сохраняем обновленную информацию
		tm.files.Store(path, fileInfo)
	}

	return nil
}
