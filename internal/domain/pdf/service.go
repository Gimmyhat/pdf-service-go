package pdf

import (
	"context"

	"pdf-service-go/internal/pkg/circuitbreaker"
)

// Service определяет интерфейс для работы с PDF
type Service interface {
	// GenerateDocx генерирует PDF документ из шаблона DOCX
	GenerateDocx(ctx context.Context, req *DocxRequest) ([]byte, error)

	// GetCircuitBreakerState возвращает текущее состояние Circuit Breaker
	GetCircuitBreakerState() circuitbreaker.State

	// IsCircuitBreakerHealthy возвращает true, если Circuit Breaker в здоровом состоянии
	IsCircuitBreakerHealthy() bool

	// GetDocxGeneratorState возвращает текущее состояние DOCX генератора
	GetDocxGeneratorState() circuitbreaker.State

	// IsDocxGeneratorHealthy возвращает true, если DOCX генератор в здоровом состоянии
	IsDocxGeneratorHealthy() bool
}
