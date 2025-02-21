package gotenberg

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClientWithPool_ConvertDocxToPDF(t *testing.T) {
	// Создаем временный DOCX файл для тестов
	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "test.docx")
	content := []byte("test content")
	if err := os.WriteFile(docxPath, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Создаем клиент с неправильным URL для тестирования ошибок
	client := NewClientWithPool("http://invalid-url")
	defer client.Close()

	// Проверяем, что конвертация завершится ошибкой
	_, err := client.ConvertDocxToPDF(docxPath)
	if err == nil {
		t.Error("Expected error from invalid URL")
	}

	// Проверяем статистику пула
	stats := client.Stats()
	if stats.TotalConnections == 0 {
		t.Error("Expected at least one connection in pool")
	}
}

func TestClientWithPool_Integration(t *testing.T) {
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
	client := NewClientWithPool(gotenbergURL)
	defer client.Close()

	// Проверяем здоровье сервиса
	if err := client.HealthCheck(); err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Проверяем успешную конвертацию
	pdf, err := client.ConvertDocxToPDF(docxPath)
	if err != nil {
		t.Errorf("Failed to convert DOCX to PDF: %v", err)
	}
	if len(pdf) == 0 {
		t.Error("Expected non-empty PDF content")
	}

	// Проверяем статистику пула
	stats := client.Stats()
	if stats.TotalConnections < 1 {
		t.Error("Expected at least one connection in pool")
	}
	if stats.ActiveConnections > stats.TotalConnections {
		t.Error("Active connections cannot exceed total connections")
	}
}

func TestClientWithPool_Concurrency(t *testing.T) {
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
	client := NewClientWithPool(gotenbergURL)
	defer client.Close()

	// Запускаем несколько параллельных конвертаций
	concurrency := 5
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			pdf, err := client.ConvertDocxToPDF(docxPath)
			if err != nil {
				t.Errorf("Concurrent conversion failed: %v", err)
			}
			if len(pdf) == 0 {
				t.Error("Expected non-empty PDF content")
			}
			done <- true
		}()
	}

	// Ждем завершения всех горутин с таймаутом
	timeout := time.After(30 * time.Second)
	for i := 0; i < concurrency; i++ {
		select {
		case <-done:
			continue
		case <-timeout:
			t.Fatal("Test timed out")
		}
	}

	// Проверяем статистику пула после параллельных запросов
	stats := client.Stats()
	if stats.TotalConnections < 1 {
		t.Error("Expected at least one connection in pool")
	}
	if stats.ActiveConnections != 0 {
		t.Errorf("Expected no active connections after test, got %d", stats.ActiveConnections)
	}
}
