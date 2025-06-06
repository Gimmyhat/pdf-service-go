package docxgen

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"pdf-service-go/internal/pkg/cache"
	"pdf-service-go/internal/pkg/circuitbreaker"
	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/retry"
	"pdf-service-go/internal/pkg/tracing"

	"bufio"
	"bytes"
	"encoding/hex"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"

	"go.opentelemetry.io/otel/attribute"
)

var (
	docxGenerationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "docx_generation_duration_seconds",
			Help:    "Duration of DOCX generation process",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 20, 30},
		},
		[]string{"status", "python_implementation"},
	)

	docxGenerationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "docx_generation_errors_total",
			Help: "Total number of DOCX generation errors",
		},
		[]string{"type", "python_implementation"},
	)

	docxGenerationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "docx_generation_total",
			Help: "Total number of DOCX generation attempts",
		},
		[]string{"status", "python_implementation"},
	)

	docxFileSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "docx_file_size_bytes",
			Help:    "Size of generated DOCX files",
			Buckets: prometheus.ExponentialBuckets(1024, 2, 10), // от 1KB до 1MB
		},
		[]string{"status", "python_implementation"},
	)
)

// Config содержит настройки для генератора DOCX
type Config struct {
	ScriptPath       string
	FailureThreshold int
	ResetTimeout     time.Duration
	HalfOpenMaxCalls int
	SuccessThreshold int
	PodName          string
	Namespace        string
	CacheTTL         time.Duration
	// Retry конфигурация
	RetryMaxAttempts   int
	RetryInitialDelay  time.Duration
	RetryMaxDelay      time.Duration
	RetryBackoffFactor float64
}

// Generator представляет генератор DOCX файлов с Circuit Breaker
type Generator struct {
	config       Config
	cb           *circuitbreaker.CircuitBreaker
	cache        *cache.Cache
	tempManager  *TempManager
	gotenbergURL string
	retrier      *retry.Retrier
}

// getEnvWithDefault возвращает значение переменной окружения или значение по умолчанию
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntWithDefault возвращает целочисленное значение переменной окружения или значение по умолчанию
func getEnvIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvDurationWithDefault возвращает значение длительности из переменной окружения или значение по умолчанию
func getEnvDurationWithDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// NewGenerator создает новый экземпляр генератора DOCX
func NewGenerator(scriptPath string) *Generator {
	config := Config{
		ScriptPath:       scriptPath,
		FailureThreshold: getEnvIntWithDefault("DOCX_CIRCUIT_BREAKER_FAILURE_THRESHOLD", 3),
		ResetTimeout:     getEnvDurationWithDefault("DOCX_CIRCUIT_BREAKER_RESET_TIMEOUT", 5*time.Second),
		HalfOpenMaxCalls: getEnvIntWithDefault("DOCX_CIRCUIT_BREAKER_HALF_OPEN_MAX_CALLS", 2),
		SuccessThreshold: getEnvIntWithDefault("DOCX_CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 2),
		PodName:          os.Getenv("POD_NAME"),
		Namespace:        os.Getenv("POD_NAMESPACE"),
		CacheTTL:         getEnvDurationWithDefault("DOCX_TEMPLATE_CACHE_TTL", 5*time.Minute),
		// Retry конфигурация
		RetryMaxAttempts:   getEnvIntWithDefault("DOCX_RETRY_MAX_ATTEMPTS", 3),
		RetryInitialDelay:  getEnvDurationWithDefault("DOCX_RETRY_INITIAL_DELAY", 100*time.Millisecond),
		RetryMaxDelay:      getEnvDurationWithDefault("DOCX_RETRY_MAX_DELAY", 2*time.Second),
		RetryBackoffFactor: float64(getEnvIntWithDefault("DOCX_RETRY_BACKOFF_FACTOR", 2)),
	}

	tempManager, err := NewTempManager(TempManagerConfig{
		Dir:           os.TempDir(),
		CleanupPeriod: getEnvDurationWithDefault("DOCX_TEMP_CLEANUP_INTERVAL", 1*time.Minute),
		MaxDirSize:    1024 * 1024 * 1024, // 1GB по умолчанию
	})
	if err != nil {
		logger.Error("Failed to create temp manager", zap.Error(err))
	}

	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Name:             "docx-generator",
		FailureThreshold: config.FailureThreshold,
		ResetTimeout:     config.ResetTimeout,
		HalfOpenMaxCalls: config.HalfOpenMaxCalls,
		SuccessThreshold: config.SuccessThreshold,
		PodName:          config.PodName,
		Namespace:        config.Namespace,
	})

	// Создаем retrier с конфигурацией
	retrier := retry.New(
		"docx-generator",
		logger.Log,
		retry.WithMaxAttempts(config.RetryMaxAttempts),
		retry.WithInitialDelay(config.RetryInitialDelay),
		retry.WithMaxDelay(config.RetryMaxDelay),
		retry.WithBackoffFactor(config.RetryBackoffFactor),
	)

	return &Generator{
		config:       config,
		cb:           cb,
		cache:        cache.NewCache(config.CacheTTL),
		tempManager:  tempManager,
		gotenbergURL: getEnvWithDefault("GOTENBERG_API_URL", "http://nas-pdf-service-gotenberg:3000"),
		retrier:      retrier,
	}
}

// Generate генерирует DOCX файл из шаблона и данных
func (g *Generator) Generate(ctx context.Context, templatePath, dataPath, outputPath string) error {
	start := time.Now()
	pythonImpl := "python"
	if os.Getenv("PYTHON_IMPLEMENTATION") == "pypy3" {
		pythonImpl = "pypy3"
	}
	docxGenerationTotal.WithLabelValues("started", pythonImpl).Inc()

	// Пытаемся получить шаблон из кэша
	templateName := filepath.Base(templatePath)
	template, err := g.cache.Get(ctx, templateName)
	if err != nil {
		// Если шаблона нет в кэше, читаем его и сохраняем
		template, err = os.ReadFile(templatePath)
		if err != nil {
			logger.Error("Failed to read template",
				zap.Error(err),
				zap.String("template", templatePath),
			)
			return err
		}
		g.cache.Set(templateName, template)
	}

	// Создаем временную директорию для выходного файла
	tempDir, err := os.MkdirTemp("", "docx_gen_")
	if err != nil {
		logger.Error("Failed to create temp directory",
			zap.Error(err))
		return err
	}
	defer os.RemoveAll(tempDir)

	// Создаем временный файл для шаблона с буферизированной записью
	tempPath := filepath.Join(tempDir, templateName)
	templateFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Error("Failed to create template file",
			zap.Error(err))
		return err
	}
	bufWriter := bufio.NewWriter(templateFile)
	if _, err = bufWriter.Write(template); err != nil {
		templateFile.Close()
		logger.Error("Failed to write template",
			zap.Error(err))
		return err
	}
	if err = bufWriter.Flush(); err != nil {
		templateFile.Close()
		logger.Error("Failed to flush template buffer",
			zap.Error(err))
		return err
	}
	templateFile.Close()

	// Создаем временный файл для выходного DOCX
	outputDocx := filepath.Join(tempDir, fmt.Sprintf("output_%s.docx",
		hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))))

	// Оборачиваем выполнение Python-скрипта в retry механизм
	err = g.retrier.Do(ctx, func(ctx context.Context) error {
		return g.cb.Execute(ctx, func() error {
			// Определяем, какую версию Python использовать
			pythonCmd := "python"
			if os.Getenv("PYTHON_IMPLEMENTATION") == "pypy3" {
				pythonCmd = "pypy3"
			}

			// Устанавливаем переменную окружения для параллельной обработки
			cmd := exec.CommandContext(ctx, pythonCmd, g.config.ScriptPath, tempPath, dataPath, outputDocx)
			output, cmdErr := cmd.CombinedOutput()
			if cmdErr != nil {
				logger.Error("Failed to execute Python script",
					zap.Error(cmdErr),
					zap.String("output", string(output)),
				)
				docxGenerationErrors.WithLabelValues("python_error", pythonImpl).Inc()
				return cmdErr
			}

			// Обновляем метрики
			duration := time.Since(start).Seconds()
			docxGenerationDuration.WithLabelValues("success", pythonImpl).Observe(duration)
			docxGenerationTotal.WithLabelValues("success", pythonImpl).Inc()

			if fi, statErr := os.Stat(outputPath); statErr == nil {
				docxFileSize.WithLabelValues("success", pythonImpl).Observe(float64(fi.Size()))
			}

			return nil
		})
	})

	if err != nil {
		docxGenerationTotal.WithLabelValues("failed", pythonImpl).Inc()
		return err
	}

	return nil
}

// State возвращает текущее состояние Circuit Breaker
func (g *Generator) State() circuitbreaker.State {
	return g.cb.State()
}

// IsHealthy возвращает true, если Circuit Breaker в здоровом состоянии
func (g *Generator) IsHealthy() bool {
	return g.cb.IsHealthy()
}

// GeneratePDF генерирует PDF из DOCX шаблона
func (g *Generator) GeneratePDF(ctx context.Context, templateName string, data interface{}) ([]byte, error) {
	ctx, span := tracing.StartSpan(ctx, "GeneratePDF")
	defer span.End()

	span.SetAttributes(attribute.String("template.name", templateName))

	// Получаем шаблон из кэша
	ctx, cacheSpan := tracing.StartSpan(ctx, "GetTemplateFromCache")
	template, err := g.cache.Get(ctx, templateName)
	cacheSpan.End()

	if err != nil {
		tracing.RecordError(ctx, err)
		return nil, fmt.Errorf("failed to get template from cache: %w", err)
	}

	// Создаем временный файл для заполненного шаблона
	ctx, tempSpan := tracing.StartSpan(ctx, "CreateTempFile")
	filledTemplate, err := g.tempManager.CreateTemp(ctx, fmt.Sprintf("filled-%d-%x-*.docx", time.Now().UnixNano(), time.Now().Nanosecond()))
	tempSpan.End()

	if err != nil {
		tracing.RecordError(ctx, err)
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer filledTemplate.Close()

	// Заполняем шаблон данными
	ctx, fillSpan := tracing.StartSpan(ctx, "FillTemplate")
	templateReader := bytes.NewReader(template)
	err = g.fillTemplate(templateReader, data, filledTemplate)
	fillSpan.End()

	if err != nil {
		tracing.RecordError(ctx, err)
		return nil, fmt.Errorf("failed to fill template: %w", err)
	}

	// Конвертируем DOCX в PDF
	ctx, convertSpan := tracing.StartSpan(ctx, "ConvertToPDF")
	pdf, err := g.convertToPDF(ctx, filledTemplate.Name())
	convertSpan.End()

	if err != nil {
		tracing.RecordError(ctx, err)
		return nil, fmt.Errorf("failed to convert to PDF: %w", err)
	}

	return pdf, nil
}

// fillTemplate заполняет шаблон данными
func (g *Generator) fillTemplate(template io.Reader, _ interface{}, output io.Writer) error {
	// Временная реализация
	_, err := io.Copy(output, template)
	return err
}

// convertToPDF конвертирует DOCX файл в PDF
func (g *Generator) convertToPDF(_ context.Context, _ string) ([]byte, error) {
	// Временная реализация
	return nil, fmt.Errorf("not implemented")
}

// GetTempManager возвращает менеджер временных файлов
func (g *Generator) GetTempManager() *TempManager {
	return g.tempManager
}
