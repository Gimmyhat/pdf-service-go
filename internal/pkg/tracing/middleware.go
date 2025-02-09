package tracing

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware добавляет трейсинг к HTTP-запросам
func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем текущий контекст и пропагатор
		ctx := r.Context()
		propagator := otel.GetTextMapPropagator()

		// Извлекаем контекст трейсинга из заголовков запроса
		ctx = propagator.Extract(ctx, propagation.HeaderCarrier(r.Header))

		// Создаем новый спан
		tracer := otel.Tracer("http.server")
		spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		ctx, span := tracer.Start(ctx, spanName)
		defer span.End()

		// Добавляем системную информацию
		hostname, _ := os.Hostname()
		span.SetAttributes(
			semconv.HostNameKey.String(hostname),
			attribute.String("runtime.os", runtime.GOOS),
			attribute.String("runtime.arch", runtime.GOARCH),
			attribute.Int("runtime.num_cpu", runtime.NumCPU()),
			attribute.Int64("runtime.num_goroutines", int64(runtime.NumGoroutine())),
		)

		// Добавляем информацию о запросе
		span.SetAttributes(
			semconv.HTTPMethodKey.String(r.Method),
			semconv.HTTPURLKey.String(r.URL.String()),
			semconv.HTTPTargetKey.String(r.URL.Path),
			semconv.HTTPHostKey.String(r.Host),
			semconv.HTTPSchemeKey.String(r.URL.Scheme),
			semconv.HTTPUserAgentKey.String(r.UserAgent()),
			semconv.HTTPClientIPKey.String(getClientIP(r)),
			attribute.String("http.request_id", r.Header.Get("X-Request-ID")),
			attribute.String("http.correlation_id", r.Header.Get("X-Correlation-ID")),
		)

		// Добавляем заголовки запроса (исключая чувствительные данные)
		for k, v := range r.Header {
			k = strings.ToLower(k)
			if !isSensitiveHeader(k) {
				span.SetAttributes(attribute.String("http.header."+k, strings.Join(v, ",")))
			}
		}

		// Добавляем обработку паники с записью стека
		defer func() {
			if err := recover(); err != nil {
				stack := make([]byte, 4096)
				stack = stack[:runtime.Stack(stack, false)]
				span.SetStatus(codes.Error, fmt.Sprintf("panic: %v", err))
				span.SetAttributes(
					attribute.String("error.type", "panic"),
					attribute.String("error.message", fmt.Sprintf("%v", err)),
					attribute.String("error.stack", string(stack)),
				)
				span.RecordError(fmt.Errorf("panic: %v", err))
				panic(err)
			}
		}()

		// Добавляем метрики времени
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w}

		// Выполняем следующий обработчик
		next.ServeHTTP(rw, r.WithContext(ctx))

		// Добавляем информацию о результате
		duration := time.Since(start)
		span.SetAttributes(
			semconv.HTTPStatusCodeKey.Int(rw.statusCode),
			attribute.Float64("http.duration_ms", float64(duration.Milliseconds())),
			attribute.Int64("http.response_size", rw.size),
		)

		// Устанавливаем статус спана
		if rw.statusCode >= 500 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d: Server Error", rw.statusCode))
		} else if rw.statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d: Client Error", rw.statusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// Добавляем события для важных статус кодов
		if rw.statusCode >= 400 {
			span.AddEvent("http.error", trace.WithAttributes(
				attribute.Int("http.status_code", rw.statusCode),
				attribute.String("http.status_text", http.StatusText(rw.statusCode)),
			))
		}
	})
}

// responseWriter оборачивает http.ResponseWriter для отслеживания статуса ответа и размера
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.size += int64(n)
	return n, err
}

// getClientIP получает реальный IP-адрес клиента
func getClientIP(r *http.Request) string {
	// Проверяем заголовки X-Forwarded-For и X-Real-IP
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		ips := strings.Split(ip, ",")
		return strings.TrimSpace(ips[0])
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	// Получаем IP из RemoteAddr
	ip := r.RemoteAddr
	if i := strings.LastIndex(ip, ":"); i != -1 {
		ip = ip[:i]
	}
	return ip
}

// isSensitiveHeader проверяет, является ли заголовок чувствительным
func isSensitiveHeader(header string) bool {
	sensitiveHeaders := map[string]bool{
		"authorization":   true,
		"proxy-authorize": true,
		"cookie":          true,
		"set-cookie":      true,
		"x-api-key":       true,
		"api-key":         true,
		"password":        true,
		"token":           true,
	}
	return sensitiveHeaders[header]
}

// GinTracingMiddleware адаптер для использования трейсинга с Gin
func GinTracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		handler := TracingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
		}))
		handler.ServeHTTP(c.Writer, c.Request)
	}
}
