package pdf

import (
	"context"
	"pdf-service-go/internal/pkg/metrics"
	"time"
)

type Service interface {
	// ... existing code ...
	GenerateDocx(ctx context.Context, req *DocxRequest) ([]byte, error)
}

type service struct {
	gotenbergURL string
}

func NewService(gotenbergURL string) Service {
	return &service{
		gotenbergURL: gotenbergURL,
	}
}

func (s *service) GenerateDocx(ctx context.Context, req *DocxRequest) ([]byte, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.PDFGenerationDuration.WithLabelValues(req.TemplateName).Observe(duration)
	}()

	// Увеличиваем счетчик начала генерации
	metrics.PDFGenerationTotal.WithLabelValues("started").Inc()

	// Начало запроса к Gotenberg
	gotenbergStart := time.Now()

	// Здесь должен быть код генерации PDF
	// ... existing code ...

	// После получения ответа от Gotenberg
	gotenbergDuration := time.Since(gotenbergStart).Seconds()
	metrics.GotenbergRequestDuration.WithLabelValues("convert").Observe(gotenbergDuration)

	// После успешной генерации
	if pdfData, err := generatePDF(ctx, req); err != nil {
		metrics.PDFGenerationTotal.WithLabelValues("error").Inc()
		metrics.GotenbergRequestsTotal.WithLabelValues("error").Inc()
		return nil, err
	} else {
		metrics.PDFGenerationTotal.WithLabelValues("success").Inc()
		metrics.GotenbergRequestsTotal.WithLabelValues("success").Inc()
		metrics.PDFFileSizeBytes.WithLabelValues(req.TemplateName).Observe(float64(len(pdfData)))
		return pdfData, nil
	}
}
