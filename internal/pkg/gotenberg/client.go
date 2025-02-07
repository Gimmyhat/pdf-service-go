package gotenberg

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type Client struct {
	baseURL string
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
	}
}

func (c *Client) ConvertDocxToPDF(docxPath string) ([]byte, error) {
	// Создаем буфер для multipart формы
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Открываем файл DOCX
	file, err := os.Open(docxPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DOCX file: %w", err)
	}
	defer file.Close()

	// Создаем часть формы для файла
	part, err := writer.CreateFormFile("files", filepath.Base(docxPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// Копируем содержимое файла в форму
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Закрываем writer
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Создаем запрос к Gotenberg
	req, err := http.NewRequest("POST", c.baseURL+"/forms/libreoffice/convert", body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Отправляем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("conversion failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Читаем PDF из ответа
	pdfContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return pdfContent, nil
}
