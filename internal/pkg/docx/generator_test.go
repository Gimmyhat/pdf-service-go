package docx

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDocxGenerator_Generate(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir, err := os.MkdirTemp("", "docx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Создаем тестовый шаблон
	templatePath := filepath.Join(tempDir, "template.docx")
	if err := createTestTemplate(templatePath); err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	// Создаем генератор
	generator := NewGenerator(templatePath)

	// Тестовые данные
	now := time.Now()
	doc := &Document{
		ID:            "test-id",
		Operation:     "test-operation",
		CreationDate:  now,
		ApplicantType: "ORGANIZATION",
		OrganizationInfo: &Organization{
			Name:    "Test Org",
			Address: "Test Address",
			Agent:   "Test Agent",
		},
		RegistryItems: []RegistryItem{
			{
				ID:              "item-1",
				InformationDate: now,
				Description:     "Test Item",
			},
		},
		PurposeOfGeoInfoAccess: "Test Purpose",
	}

	// Путь для выходного файла
	outputPath := filepath.Join(tempDir, "output.docx")

	// Генерируем документ
	if err := generator.Generate(doc, outputPath); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Проверяем, что файл создан
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Проверяем содержимое файла
	if err := validateGeneratedFile(outputPath, doc); err != nil {
		t.Errorf("Generated file validation failed: %v", err)
	}
}

func createTestTemplate(path string) error {
	// Создаем ZIP архив
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Добавляем document.xml с плейсхолдерами
	writer, err := zipWriter.Create("word/document.xml")
	if err != nil {
		return err
	}

	content := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
    <w:body>
        <w:p>
            <w:r>
                <w:t>Дата создания: {{creationDate}}</w:t>
            </w:r>
        </w:p>
        <w:p>
            <w:r>
                <w:t>Заявитель: {{applicant_info}}</w:t>
            </w:r>
        </w:p>
        <w:p>
            <w:r>
                <w:t>Цель: {{purposeOfGeoInfoAccess}}</w:t>
            </w:r>
        </w:p>
    </w:body>
</w:document>`

	if _, err := writer.Write([]byte(content)); err != nil {
		return err
	}

	return nil
}

func validateGeneratedFile(path string, doc *Document) error {
	// Открываем сгенерированный файл
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Ищем document.xml
	var documentXML *zip.File
	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			documentXML = file
			break
		}
	}

	if documentXML == nil {
		return nil
	}

	// Читаем содержимое
	rc, err := documentXML.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	// Проверяем наличие замененных плейсхолдеров
	contentStr := string(content)
	expectedDate := doc.CreationDate.Format("02.01.2006")
	if !contains(contentStr, expectedDate) {
		return nil
	}

	if !contains(contentStr, doc.OrganizationInfo.Name) {
		return nil
	}

	if !contains(contentStr, doc.PurposeOfGeoInfoAccess) {
		return nil
	}

	return nil
}
