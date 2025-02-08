package docxgen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"pdf-service-go/internal/pkg/logger"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var (
	tempFileManager = &TempFileManager{
		activeFiles: make(map[string]time.Time),
		mutex:       &sync.RWMutex{},
	}

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
)

// TempFileManager управляет временными файлами
type TempFileManager struct {
	activeFiles map[string]time.Time
	mutex       *sync.RWMutex
}

// CreateTempFile создает временный файл с уникальным именем
func (tm *TempFileManager) CreateTempFile(prefix string, data []byte) (string, error) {
	// Генерируем уникальный идентификатор
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	uniqueID := hex.EncodeToString(randomBytes)

	// Создаем имя файла с временной меткой
	timestamp := time.Now().UnixNano()
	fileName := fmt.Sprintf("%s_%s_%d", prefix, uniqueID, timestamp)
	tempPath := filepath.Join(os.TempDir(), fileName)

	// Проверяем и создаем временную директорию если нужно
	if err := os.MkdirAll(os.TempDir(), 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Записываем данные во временный файл
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	// Регистрируем файл в менеджере
	tm.mutex.Lock()
	tm.activeFiles[tempPath] = time.Now()
	tempFileCount.Set(float64(len(tm.activeFiles)))
	tm.mutex.Unlock()

	tempFileCreations.WithLabelValues(prefix).Inc()
	logger.Debug("Created temporary file",
		zap.String("path", tempPath),
		zap.Int("size", len(data)))

	return tempPath, nil
}

// RemoveTempFile безопасно удаляет временный файл
func (tm *TempFileManager) RemoveTempFile(path string) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if createdAt, exists := tm.activeFiles[path]; exists {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			tempFileErrors.WithLabelValues("remove").Inc()
			return fmt.Errorf("failed to remove temp file: %w", err)
		}
		age := time.Since(createdAt).Seconds()
		tempFileAge.WithLabelValues("removed").Observe(age)
		delete(tm.activeFiles, path)
		tempFileCount.Set(float64(len(tm.activeFiles)))
		logger.Debug("Removed temporary file",
			zap.String("path", path),
			zap.Float64("age_seconds", age))
	}
	return nil
}

// Cleanup удаляет все устаревшие временные файлы
func (tm *TempFileManager) Cleanup(maxAge time.Duration) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	now := time.Now()
	for path, createdAt := range tm.activeFiles {
		age := now.Sub(createdAt)
		if age > maxAge {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				tempFileErrors.WithLabelValues("cleanup").Inc()
				logger.Error("Failed to remove old temp file",
					zap.String("path", path),
					zap.Error(err))
			} else {
				tempFileAge.WithLabelValues("expired").Observe(age.Seconds())
				delete(tm.activeFiles, path)
				logger.Debug("Cleaned up old temporary file",
					zap.String("path", path),
					zap.Float64("age_seconds", age.Seconds()))
			}
		}
	}
	tempFileCount.Set(float64(len(tm.activeFiles)))
}

// StartCleanupRoutine запускает периодическую очистку временных файлов
func (tm *TempFileManager) StartCleanupRoutine(interval, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			tm.Cleanup(maxAge)
		}
	}()
}
