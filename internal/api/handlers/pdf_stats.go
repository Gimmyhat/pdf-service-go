package handlers

import (
	"pdf-service-go/internal/pkg/statistics"
	"time"
)

// AddStatisticsTracking добавляет отслеживание статистики к PDFHandler
func (h *PDFHandler) AddStatisticsTracking() {
	h.stats = statistics.GetInstance()
}

// TrackDocxGeneration отслеживает статистику генерации DOCX
func (h *PDFHandler) TrackDocxGeneration(duration time.Duration, hasError bool) {
	if h.stats != nil {
		h.stats.TrackDocx(duration, hasError)
	}
}

// TrackPDFFile отслеживает статистику PDF файла
func (h *PDFHandler) TrackPDFFile(size int64) {
	if h.stats != nil {
		h.stats.TrackPDF(size)
	}
}

// TrackGotenbergRequest отслеживает статистику запроса к Gotenberg
func (h *PDFHandler) TrackGotenbergRequest(duration time.Duration, hasError bool) {
	if h.stats != nil {
		h.stats.TrackGotenberg(duration, hasError)
	}
}
