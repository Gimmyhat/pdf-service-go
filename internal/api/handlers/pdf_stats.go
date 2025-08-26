package handlers

import (
	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/statistics"
	"time"
)

// AddStatisticsTracking добавляет отслеживание статистики к PDFHandler
func (h *PDFHandler) AddStatisticsTracking() {
	// Не кэшируем инстанс навсегда: берём лениво на первый вызов
	h.stats = statistics.GetInstance()
}

// TrackDocxGeneration отслеживает статистику генерации DOCX
func (h *PDFHandler) TrackDocxGeneration(duration time.Duration, hasError bool) {
	if h.stats == nil {
		h.stats = statistics.GetInstance()
	}
	if h.stats != nil {
		if err := h.stats.TrackDocx(duration, hasError); err != nil {
			logger.Warn("TrackDocx failed", logger.Field("error", err))
		}
	}
}

// TrackPDFFile отслеживает статистику PDF файла
func (h *PDFHandler) TrackPDFFile(size int64) {
	if h.stats == nil {
		h.stats = statistics.GetInstance()
	}
	if h.stats != nil {
		if err := h.stats.TrackPDF(size); err != nil {
			logger.Warn("TrackPDF failed", logger.Field("error", err))
		}
	}
}

// TrackGotenbergRequest отслеживает статистику запроса к Gotenberg
func (h *PDFHandler) TrackGotenbergRequest(duration time.Duration, hasError bool) {
	if h.stats == nil {
		h.stats = statistics.GetInstance()
	}
	if h.stats != nil {
		if err := h.stats.TrackGotenberg(duration, hasError); err != nil {
			logger.Warn("TrackGotenberg failed", logger.Field("error", err))
		}
	}
}
