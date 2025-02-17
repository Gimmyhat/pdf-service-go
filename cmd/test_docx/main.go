package main

import (
	"fmt"
	"os"
	"path/filepath"
	"pdf-service-go/internal/pkg/docxgen"
)

func main() {
	// Путь к шаблону
	templatePath := "internal/domain/pdf/templates/template.docx"

	// Путь к тестовому JSON файлу
	jsonPath := "test/exm2.json"

	// Путь к выходному файлу
	outputPath := "test_output.docx"

	// Читаем тестовые данные
	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		fmt.Printf("Failed to read test data: %v\n", err)
		return
	}

	// Создаем генератор
	generator := docxgen.NewDocxGenerator(templatePath)

	// Генерируем документ
	err = generator.Generate(jsonData, outputPath)
	if err != nil {
		fmt.Printf("Failed to generate document: %v\n", err)
		return
	}

	// Получаем абсолютный путь к созданному файлу
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		return
	}

	fmt.Printf("Document successfully generated: %s\n", absPath)
}
