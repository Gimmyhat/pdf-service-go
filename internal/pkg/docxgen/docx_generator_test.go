package docxgen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDocxGenerator_Generate(t *testing.T) {
	// Путь к тестовому шаблону
	templatePath := "../../domain/pdf/templates/template.docx"

	// Путь к тестовому JSON файлу
	jsonPath := "../../../test/exm2.json"

	// Создаем временную директорию для выходного файла
	tempDir, err := os.MkdirTemp("", "docx_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Путь к выходному файлу
	outputPath := filepath.Join(tempDir, "output.docx")

	// Читаем тестовые данные
	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	// Создаем генератор
	generator := NewDocxGenerator(templatePath)

	// Генерируем документ
	err = generator.Generate(jsonData, outputPath)
	if err != nil {
		t.Fatalf("Failed to generate document: %v", err)
	}

	// Проверяем, что файл создан
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}
}
