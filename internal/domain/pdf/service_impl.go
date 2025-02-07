package pdf

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"pdf-service-go/internal/pkg/gotenberg"
)

type ServiceImpl struct {
	gotenbergClient *gotenberg.Client
}

func NewService(gotenbergURL string) Service {
	return &ServiceImpl{
		gotenbergClient: gotenberg.NewClient(gotenbergURL),
	}
}

// ... existing code ...

func (s *ServiceImpl) GenerateDocx(ctx context.Context, req *DocxRequest) ([]byte, error) {
	// Проверяем наличие шаблона
	templatePath := filepath.Join("internal/domain/pdf/templates", "template.docx")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return nil, ErrTemplateNotFound
	}

	// Создаем временные файлы
	tempDir := os.TempDir()
	dataPath := filepath.Join(tempDir, fmt.Sprintf("%s.json", req.ID))
	docxPath := filepath.Join(tempDir, fmt.Sprintf("%s.docx", req.ID))
	defer os.Remove(dataPath)
	defer os.Remove(docxPath)

	// Сохраняем данные во временный JSON файл
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}
	err = os.WriteFile(dataPath, data, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write data file: %w", err)
	}

	// Запускаем Python-скрипт для генерации DOCX
	scriptPath := "scripts/generate_docx.py"
	cmd := exec.CommandContext(ctx, "python", scriptPath, templatePath, dataPath, docxPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to generate document: %s: %w", string(output), err)
	}

	// Конвертируем DOCX в PDF через Gotenberg
	pdfContent, err := s.gotenbergClient.ConvertDocxToPDF(docxPath)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to PDF: %w", err)
	}

	return pdfContent, nil
}
