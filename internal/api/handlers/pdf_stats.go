package handlers

import (
	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/statistics"
	"time"

	"go.uber.org/zap"
)

// AddStatisticsTracking добавляет отслеживание статистики к PDFHandler
func (h *PDFHandler) AddStatisticsTracking() {
	h.stats = statistics.GetInstance()
}

// TrackDocxGeneration отслеживает статистику генерации DOCX
func (h *PDFHandler) TrackDocxGeneration(duration time.Duration, hasError bool) {
	if h.stats != nil {
		if err := h.stats.TrackDocx(duration, hasError); err != nil {
			logger.Log.Error("Failed to track DOCX metrics", zap.Error(err))
		}
	}
}

// TrackPDFFile отслеживает статистику PDF файла
func (h *PDFHandler) TrackPDFFile(size int64) {
	if h.stats != nil {
		if err := h.stats.TrackPDF(size); err != nil {
			logger.Log.Error("Failed to track PDF metrics", zap.Error(err))
		}
	}
}

// TrackGotenbergRequest отслеживает статистику запроса к Gotenberg
func (h *PDFHandler) TrackGotenbergRequest(duration time.Duration, hasError bool) {
	if h.stats != nil {
		if err := h.stats.TrackGotenberg(duration, hasError); err != nil {
			logger.Log.Error("Failed to track Gotenberg metrics", zap.Error(err))
		}
	}
}
