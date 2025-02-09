package docxgen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
)

// TempManager управляет временными файлами
type TempManager struct {
	dir           string
	cleanupPeriod time.Duration
}

// NewTempManager создает новый менеджер временных файлов
func NewTempManager(dir string, cleanupPeriod time.Duration) (*TempManager, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	tm := &TempManager{
		dir:           dir,
		cleanupPeriod: cleanupPeriod,
	}

	go tm.startCleanup()
	return tm, nil
}

// CreateTemp создает временный файл
func (tm *TempManager) CreateTemp(ctx context.Context, pattern string) (*os.File, error) {
	ctx, span := tracing.StartSpan(ctx, "TempManager.CreateTemp")
	defer span.End()

	span.SetAttributes(
		attribute.String("temp.dir", tm.dir),
		attribute.String("temp.pattern", pattern),
	)

	file, err := os.CreateTemp(tm.dir, pattern)
	if err != nil {
		tempFileErrors.WithLabelValues("create").Inc()
		tracing.RecordError(ctx, err)
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

	threshold := time.Now().Add(-tm.cleanupPeriod)
	var deletedCount int

	err := filepath.Walk(tm.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && info.ModTime().Before(threshold) {
			age := time.Since(info.ModTime()).Seconds()
			if err := os.Remove(path); err != nil {
				tempFileErrors.WithLabelValues("cleanup").Inc()
				tracing.RecordError(ctx, fmt.Errorf("failed to remove file %s: %w", path, err))
				return err
			}
			tempFileAge.WithLabelValues("deleted").Observe(age)
			tempFileCount.Dec()
			deletedCount++
		}
		return nil
	})

	if err != nil {
		tempFileErrors.WithLabelValues("cleanup").Inc()
		tracing.RecordError(ctx, err)
		return fmt.Errorf("cleanup failed: %w", err)
	}

	span.SetAttributes(attribute.Int("temp.files.deleted", deletedCount))
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
	}
}
