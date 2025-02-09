package tracing

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config содержит настройки для трейсинга
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	CollectorURL   string
	SamplingRate   float64 // Частота сэмплирования (0.0 - 1.0)
	BatchTimeout   int     // Таймаут для батча в секундах
	MaxExportBatch int     // Максимальный размер батча
	MaxQueueSize   int     // Максимальный размер очереди
	AttributeLimit int     // Лимит атрибутов на спан
	EventLimit     int     // Лимит событий на спан
	SpanLimit      int     // Лимит дочерних спанов
}

// InitTracer инициализирует глобальный трейсер
func InitTracer(cfg Config) (func(context.Context) error, error) {
	ctx := context.Background()

	// Устанавливаем дефолтные значения
	if cfg.SamplingRate == 0 {
		cfg.SamplingRate = 1.0
	}
	if cfg.BatchTimeout == 0 {
		cfg.BatchTimeout = 5
	}
	if cfg.MaxExportBatch == 0 {
		cfg.MaxExportBatch = 512
	}
	if cfg.MaxQueueSize == 0 {
		cfg.MaxQueueSize = 2048
	}
	if cfg.AttributeLimit == 0 {
		cfg.AttributeLimit = 128
	}
	if cfg.EventLimit == 0 {
		cfg.EventLimit = 128
	}
	if cfg.SpanLimit == 0 {
		cfg.SpanLimit = 128
	}

	// Создаем клиент для отправки трейсов с таймаутом и ретраями
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(cfg.CollectorURL),
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		otlptracegrpc.WithTimeout(30 * time.Second),
		otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: 1 * time.Second,
			MaxInterval:     5 * time.Second,
			MaxElapsedTime:  30 * time.Second,
		}),
	}

	client := otlptracegrpc.NewClient(opts...)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Получаем hostname и pod name для ресурса
	hostname, _ := os.Hostname()
	podName := os.Getenv("KUBERNETES_POD_NAME")
	if podName == "" {
		podName = hostname
	}

	// Создаем расширенный ресурс с информацией о сервисе и окружении
	baseResource := resource.NewWithAttributes(
		semconv.SchemaURL,
		// Информация о сервисе
		semconv.ServiceNameKey.String(cfg.ServiceName),
		semconv.ServiceVersionKey.String(cfg.ServiceVersion),
		semconv.ServiceInstanceIDKey.String(podName),

		// Информация об окружении
		semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		semconv.HostNameKey.String(hostname),

		// Информация о runtime
		semconv.TelemetrySDKLanguageKey.String("go"),
		semconv.TelemetrySDKVersionKey.String(runtime.Version()),
		attribute.String("runtime.os", runtime.GOOS),
		attribute.String("runtime.arch", runtime.GOARCH),

		// Информация о Kubernetes
		attribute.String("k8s.pod.name", podName),
		attribute.String("k8s.namespace", os.Getenv("KUBERNETES_NAMESPACE")),
		attribute.String("k8s.cluster", os.Getenv("KUBERNETES_CLUSTER")),
	)

	cloudResource, err := cloudDetector()
	if err != nil {
		return nil, fmt.Errorf("failed to detect cloud provider: %w", err)
	}

	res, err := resource.Merge(baseResource, cloudResource)
	if err != nil {
		return nil, fmt.Errorf("failed to merge resources: %w", err)
	}

	// Создаем провайдер трейсинга с расширенными настройками
	tracerProvider := sdktrace.NewTracerProvider(
		// Настройки экспортера
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(time.Duration(cfg.BatchTimeout)*time.Second),
			sdktrace.WithMaxExportBatchSize(cfg.MaxExportBatch),
			sdktrace.WithMaxQueueSize(cfg.MaxQueueSize),
		),

		// Добавляем ресурс
		sdktrace.WithResource(res),

		// Настройки сэмплирования
		sdktrace.WithSampler(sdktrace.ParentBased(
			sdktrace.TraceIDRatioBased(cfg.SamplingRate),
		)),

		// Лимиты на спаны
		sdktrace.WithSpanLimits(sdktrace.SpanLimits{
			AttributeCountLimit:         cfg.AttributeLimit,
			EventCountLimit:             cfg.EventLimit,
			LinkCountLimit:              cfg.SpanLimit,
			AttributePerEventCountLimit: cfg.AttributeLimit,
			AttributePerLinkCountLimit:  cfg.AttributeLimit,
		}),
	)

	// Устанавливаем глобальный провайдер
	otel.SetTracerProvider(tracerProvider)

	// Устанавливаем глобальный пропагатор с поддержкой различных форматов
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
		&jaegerPropagator{}, // Добавляем поддержку Jaeger формата
	))

	// Возвращаем функцию для корректного завершения работы
	return func(ctx context.Context) error {
		// Даем время на отправку всех спанов
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		if err := tracerProvider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown tracer provider: %w", err)
		}
		return nil
	}, nil
}

// cloudDetector возвращает ресурс с информацией о провайдере
func cloudDetector() (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		attribute.String("cloud.provider", "unknown"),
	}

	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		attrs = []attribute.KeyValue{
			attribute.String("cloud.provider", "kubernetes"),
		}
	}

	r := resource.NewWithAttributes(
		semconv.SchemaURL,
		attrs...,
	)
	return r, nil
}

// jaegerPropagator реализует поддержку формата Jaeger
type jaegerPropagator struct{}

func (p *jaegerPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	// Реализация инъекции заголовков Jaeger
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		carrier.Set("uber-trace-id", fmt.Sprintf("%s:%s:0:1",
			span.SpanContext().TraceID().String(),
			span.SpanContext().SpanID().String(),
		))
	}
}

func (p *jaegerPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	// Реализация извлечения заголовков Jaeger
	if h := carrier.Get("uber-trace-id"); h != "" {
		if parts := strings.Split(h, ":"); len(parts) == 4 {
			if traceID, err := trace.TraceIDFromHex(parts[0]); err == nil {
				if spanID, err := trace.SpanIDFromHex(parts[1]); err == nil {
					spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
						TraceID: traceID,
						SpanID:  spanID,
						Remote:  true,
					})
					return trace.ContextWithSpanContext(ctx, spanCtx)
				}
			}
		}
	}
	return ctx
}

func (p *jaegerPropagator) Fields() []string {
	return []string{"uber-trace-id"}
}
