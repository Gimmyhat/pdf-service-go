package gotenberg

import (
	"os"
	"path/filepath"
	"testing"

	"pdf-service-go/internal/pkg/circuitbreaker"
)

func TestClientWithCircuitBreaker_ConvertDocxToPDF(t *testing.T) {
	// Создаем временный DOCX файл для тестов
	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "test.docx")
	content := []byte("test content")
	if err := os.WriteFile(docxPath, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Создаем клиент с неправильным URL для тестирования ошибок
	client := NewClientWithCircuitBreaker("http://invalid-url")
	handler := &mockStatsHandler{}
	client.SetHandler(handler)

	// Проверяем начальное состояние
	if state := client.State(); state != circuitbreaker.StateClosed {
		t.Errorf("Expected initial state to be Closed, got %v", state)
	}

	// Вызываем ошибки до срабатывания Circuit Breaker
	for i := 0; i < 6; i++ {
		_, err := client.ConvertDocxToPDF(docxPath)
		if err == nil {
			t.Error("Expected error from invalid URL")
		}
		if handler.isHealthCheck {
			t.Error("Expected regular request, got health check")
		}
	}

	// Проверяем, что Circuit Breaker открылся
	if state := client.State(); state != circuitbreaker.StateOpen {
		t.Errorf("Expected state to be Open after failures, got %v", state)
	}
}

func TestClientWithCircuitBreaker_Integration(t *testing.T) {
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
	client := NewClientWithCircuitBreaker(gotenbergURL)
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
