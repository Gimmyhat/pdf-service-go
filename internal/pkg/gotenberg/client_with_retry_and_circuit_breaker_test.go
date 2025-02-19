package gotenberg

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"pdf-service-go/internal/pkg/circuitbreaker"
)

func TestClientWithRetryAndCircuitBreaker_ConvertDocxToPDF(t *testing.T) {
	// Создаем временный DOCX файл для тестов
	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "test.docx")
	content := []byte("test content")
	if err := os.WriteFile(docxPath, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Создаем клиент с неправильным URL для тестирования retry и circuit breaker
	client := NewClientWithRetryAndCircuitBreaker("http://invalid-url")
	handler := &mockStatsHandler{}
	client.SetHandler(handler)

	// Проверяем начальное состояние
	if state := client.State(); state != circuitbreaker.StateClosed {
		t.Errorf("Expected initial state to be Closed, got %v", state)
	}

	// Проверяем, что все попытки завершатся ошибкой
	start := time.Now()
	for i := 0; i < 6; i++ {
		_, err := client.ConvertDocxToPDF(docxPath)
		if err == nil {
			t.Error("Expected error from invalid URL")
		}
		if handler.isHealthCheck {
			t.Error("Expected regular request, got health check")
		}
	}
	duration := time.Since(start)

	// Проверяем, что Circuit Breaker открылся
	if state := client.State(); state != circuitbreaker.StateOpen {
		t.Errorf("Expected state to be Open after failures, got %v", state)
	}

	// Проверяем, что время выполнения соответствует ожидаемому количеству retry
	expectedMinDuration := 300 * time.Millisecond
	if duration < expectedMinDuration {
		t.Errorf("Expected duration >= %v, got %v", expectedMinDuration, duration)
	}

	// Проверяем быстрый отказ при открытом Circuit Breaker
	start = time.Now()
	_, err := client.ConvertDocxToPDF(docxPath)
	fastFailDuration := time.Since(start)

	if err == nil {
		t.Error("Expected error from open circuit breaker")
	}

	// Проверяем, что отказ произошел быстро (без retry)
	if fastFailDuration > 50*time.Millisecond {
		t.Errorf("Expected fast failure with open circuit breaker, but took %v", fastFailDuration)
	}
}

func TestClientWithRetryAndCircuitBreaker_Integration(t *testing.T) {
	// Пропускаем тест, если переменная окружения не установлена
	gotenbergURL := os.Getenv("GOTENBERG_URL")
	if gotenbergURL == "" {
		t.Skip("GOTENBERG_URL not set")
	}

	// Создаем временный DOCX файл для тестов
	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "test.docx")
	content := []byte("test content")
	if err := os.WriteFile(docxPath, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Создаем клиент с реальным URL
	client := NewClientWithRetryAndCircuitBreaker(gotenbergURL)
	handler := &mockStatsHandler{}
	client.SetHandler(handler)

	// Проверяем успешную конвертацию
	pdf, err := client.ConvertDocxToPDF(docxPath)
	if err != nil {
		t.Errorf("Failed to convert DOCX to PDF: %v", err)
	}
	if len(pdf) == 0 {
		t.Error("Expected non-empty PDF content")
	}
	if handler.isHealthCheck {
		t.Error("Expected regular request, got health check")
	}

	// Проверяем, что Circuit Breaker остался закрытым
	if state := client.State(); state != circuitbreaker.StateClosed {
		t.Errorf("Expected state to remain Closed after success, got %v", state)
	}
}
