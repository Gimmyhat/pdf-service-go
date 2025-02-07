package api

import (
	"pdf-service-go/internal/api/handlers"
)

type Handlers struct {
	PDF *handlers.PDFHandler
}

func NewHandlers(pdfHandler *handlers.PDFHandler) *Handlers {
	return &Handlers{
		PDF: pdfHandler,
	}
}
