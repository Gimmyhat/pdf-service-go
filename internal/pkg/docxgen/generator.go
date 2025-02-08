package docxgen

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"time"

	"pdf-service-go/internal/pkg/circuitbreaker"
	"pdf-service-go/internal/pkg/logger"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var (
	docxGenerationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "docx_generation_duration_seconds",
			Help:    "Duration of DOCX generation process",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 20, 30},
		},
		[]string{"status"},
	)

	docxGenerationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "docx_generation_errors_total",
			Help: "Total number of DOCX generation errors",
		},
		[]string{"type"},
	)

	docxGenerationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "docx_generation_total",
			Help: "Total number of DOCX generation attempts",
		},
		[]string{"status"},
	)

	docxFileSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "docx_file_size_bytes",
			Help:    "Size of generated DOCX files",
			Buckets: prometheus.ExponentialBuckets(1024, 2, 10), // от 1KB до 1MB
		},
		[]string{"status"},
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
}

// Generator представляет генератор DOCX файлов с Circuit Breaker
type Generator struct {
	config Config
	cb     *circuitbreaker.CircuitBreaker
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

	return &Generator{
		config: config,
		cb:     cb,
	}
}

// Generate генерирует DOCX файл из шаблона и данных
func (g *Generator) Generate(ctx context.Context, templatePath, dataPath, outputPath string) error {
	start := time.Now()
	docxGenerationTotal.WithLabelValues("started").Inc()

	err := g.cb.Execute(ctx, func() error {
		cmd := exec.CommandContext(ctx, "python", g.config.ScriptPath, templatePath, dataPath, outputPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			logger.Error("Failed to generate DOCX",
				zap.Error(err),
				zap.String("output", string(output)),
				zap.String("template", templatePath),
				zap.String("data", dataPath),
				zap.String("output_path", outputPath),
			)
			docxGenerationErrors.WithLabelValues("python_error").Inc()
			return err
		}
		return nil
	})

	duration := time.Since(start).Seconds()
	docxGenerationDuration.WithLabelValues(g.getStatus(err)).Observe(duration)

	if err != nil {
		docxGenerationTotal.WithLabelValues("error").Inc()
		return err
	}

	// Записываем размер сгенерированного файла
	if fileInfo, err := os.Stat(outputPath); err == nil {
		docxFileSize.WithLabelValues("success").Observe(float64(fileInfo.Size()))
	}

	docxGenerationTotal.WithLabelValues("success").Inc()
	return nil
}

// getStatus возвращает статус операции для метрик
func (g *Generator) getStatus(err error) string {
	if err == nil {
		return "success"
	}
	return "error"
}

// State возвращает текущее состояние Circuit Breaker
func (g *Generator) State() circuitbreaker.State {
	return g.cb.State()
}

// IsHealthy возвращает true, если Circuit Breaker в здоровом состоянии
func (g *Generator) IsHealthy() bool {
	return g.cb.IsHealthy()
}
