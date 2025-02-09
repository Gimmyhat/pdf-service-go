package gotenberg

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClientWithRetry_ConvertDocxToPDF(t *testing.T) {
	// Создаем временный DOCX файл для тестов
	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "test.docx")
	content := []byte("test content")
	if err := os.WriteFile(docxPath, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Создаем клиент с неправильным URL для тестирования retry
	client := NewClientWithRetry("http://invalid-url")

	// Проверяем, что все попытки завершатся ошибкой
	start := time.Now()
	_, err := client.ConvertDocxToPDF(docxPath)
	duration := time.Since(start)

	// Проверяем, что была ошибка
	if err == nil {
		t.Error("Expected error from invalid URL")
	}

	// Проверяем, что время выполнения соответствует ожидаемому количеству retry
	expectedMinDuration := 300 * time.Millisecond // 100ms + 200ms для первых двух retry
	if duration < expectedMinDuration {
		t.Errorf("Expected duration >= %v, got %v", expectedMinDuration, duration)
	}
}

func TestClientWithRetry_Integration(t *testing.T) {
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
	client := NewClientWithRetry(gotenbergURL)

	// Проверяем успешную конвертацию
	pdf, err := client.ConvertDocxToPDF(docxPath)
	if err != nil {
		t.Errorf("Failed to convert DOCX to PDF: %v", err)
	}
	if len(pdf) == 0 {
		t.Error("Expected non-empty PDF content")
	}
}
