package statistics

import (
	"fmt"
	"sync"
	"time"
)

var (
	instance *Statistics
	once     sync.Once
)

// GetInstance возвращает синглтон Statistics
func GetInstance() *Statistics {
	once.Do(func() {
		instance = &Statistics{
			Requests: RequestStats{
				RequestsByDay:  make(map[time.Weekday]uint64),
				RequestsByHour: make(map[int]uint64),
			},
		}
	})
	return instance
}

// TrackRequest регистрирует новый запрос
func (s *Statistics) TrackRequest(duration time.Duration, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.Requests.TotalRequests++
	if success {
		s.Requests.SuccessRequests++
	} else {
		s.Requests.FailedRequests++
	}

	// Обновляем длительность
	s.Requests.TotalDuration += duration
	if s.Requests.MinDuration == 0 || duration < s.Requests.MinDuration {
		s.Requests.MinDuration = duration
	}
	if duration > s.Requests.MaxDuration {
		s.Requests.MaxDuration = duration
	}

	// Обновляем распределение по дням недели и часам
	s.Requests.RequestsByDay[now.Weekday()]++
	s.Requests.RequestsByHour[now.Hour()]++
	s.Requests.LastUpdated = now
}

// TrackDocxGeneration регистрирует генерацию DOCX
func (s *Statistics) TrackDocxGeneration(duration time.Duration, hasError bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Docx.TotalGenerations++
	if hasError {
		s.Docx.ErrorGenerations++
	}

	s.Docx.TotalDuration += duration
	if s.Docx.MinDuration == 0 || duration < s.Docx.MinDuration {
		s.Docx.MinDuration = duration
	}
	if duration > s.Docx.MaxDuration {
		s.Docx.MaxDuration = duration
	}
	s.Docx.LastGenerationTime = time.Now()
}

// TrackGotenbergRequest регистрирует запрос к Gotenberg
func (s *Statistics) TrackGotenbergRequest(duration time.Duration, hasError bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Gotenberg.TotalRequests++
	if hasError {
		s.Gotenberg.ErrorRequests++
	}

	s.Gotenberg.TotalDuration += duration
	if s.Gotenberg.MinDuration == 0 || duration < s.Gotenberg.MinDuration {
		s.Gotenberg.MinDuration = duration
	}
	if duration > s.Gotenberg.MaxDuration {
		s.Gotenberg.MaxDuration = duration
	}
	s.Gotenberg.LastRequestTime = time.Now()
}

// TrackPDFFile регистрирует информацию о PDF файле
func (s *Statistics) TrackPDFFile(size int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.PDF.TotalFiles++
	s.PDF.TotalSize += size
	if s.PDF.MinSize == 0 || size < s.PDF.MinSize {
		s.PDF.MinSize = size
	}
	if size > s.PDF.MaxSize {
		s.PDF.MaxSize = size
	}
	s.PDF.AverageSize = float64(s.PDF.TotalSize) / float64(s.PDF.TotalFiles)
	s.PDF.LastProcessedTime = time.Now()
}

// GetStatistics возвращает текущую статистику в формате для API
func (s *Statistics) GetStatistics() StatisticsResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var response StatisticsResponse

	// Заполняем статистику запросов
	response.Requests.Total = s.Requests.TotalRequests
	response.Requests.Success = s.Requests.SuccessRequests
	response.Requests.Failed = s.Requests.FailedRequests

	if s.Requests.TotalRequests > 0 {
		avgDuration := s.Requests.TotalDuration / time.Duration(s.Requests.TotalRequests)
		response.Requests.AverageDuration = avgDuration.String()
	}
	response.Requests.MinDuration = s.Requests.MinDuration.String()
	response.Requests.MaxDuration = s.Requests.MaxDuration.String()

	// Конвертируем дни недели в строки
	response.Requests.ByDayOfWeek = make(map[string]uint64)
	for day, count := range s.Requests.RequestsByDay {
		response.Requests.ByDayOfWeek[day.String()] = count
	}

	// Конвертируем часы в строки
	response.Requests.ByHourOfDay = make(map[string]uint64)
	for hour, count := range s.Requests.RequestsByHour {
		response.Requests.ByHourOfDay[fmt.Sprintf("%02d:00", hour)] = count
	}

	// Заполняем статистику DOCX
	response.Docx.TotalGenerations = s.Docx.TotalGenerations
	response.Docx.ErrorGenerations = s.Docx.ErrorGenerations
	if s.Docx.TotalGenerations > 0 {
		avgDuration := s.Docx.TotalDuration / time.Duration(s.Docx.TotalGenerations)
		response.Docx.AverageDuration = avgDuration.String()
	}
	response.Docx.MinDuration = s.Docx.MinDuration.String()
	response.Docx.MaxDuration = s.Docx.MaxDuration.String()

	// Заполняем статистику Gotenberg
	response.Gotenberg.TotalRequests = s.Gotenberg.TotalRequests
	response.Gotenberg.ErrorRequests = s.Gotenberg.ErrorRequests
	if s.Gotenberg.TotalRequests > 0 {
		avgDuration := s.Gotenberg.TotalDuration / time.Duration(s.Gotenberg.TotalRequests)
		response.Gotenberg.AverageDuration = avgDuration.String()
	}
	response.Gotenberg.MinDuration = s.Gotenberg.MinDuration.String()
	response.Gotenberg.MaxDuration = s.Gotenberg.MaxDuration.String()

	// Заполняем статистику PDF
	response.PDF.TotalFiles = s.PDF.TotalFiles
	response.PDF.TotalSize = formatBytes(s.PDF.TotalSize)
	response.PDF.MinSize = formatBytes(s.PDF.MinSize)
	response.PDF.MaxSize = formatBytes(s.PDF.MaxSize)
	response.PDF.AverageSize = formatBytes(int64(s.PDF.AverageSize))

	response.LastUpdated = s.Requests.LastUpdated

	return response
}

// formatBytes форматирует размер в байтах в человекочитаемый формат
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
