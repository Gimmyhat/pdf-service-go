package gotenberg

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"pdf-service-go/internal/pkg/metrics"
)

type Client struct {
	baseURL string
	client  *http.Client
	handler interface {
		TrackGotenbergRequest(duration time.Duration, hasError bool)
	}
}

func NewClient(baseURL string) *Client {
	transport := &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		MaxConnsPerHost:     100,
		ForceAttemptHTTP2:   true,
		WriteBufferSize:     64 * 1024,
		ReadBufferSize:      64 * 1024,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return &Client{
		baseURL: baseURL,
		client:  client,
	}
}

// SetHandler устанавливает обработчик для сбора статистики
func (c *Client) SetHandler(handler interface {
	TrackGotenbergRequest(duration time.Duration, hasError bool)
}) {
	c.handler = handler
}

// GetHandler возвращает обработчик статистики
func (c *Client) GetHandler() (interface {
	TrackGotenbergRequest(duration time.Duration, hasError bool)
}, bool) {
	if c.handler == nil {
		return nil, false
	}
	return c.handler, true
}

func (c *Client) ConvertDocxToPDF(docxPath string) ([]byte, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		metrics.GotenbergRequestDuration.WithLabelValues("convert").Observe(duration.Seconds())
		if c.handler != nil {
			c.handler.TrackGotenbergRequest(duration, false)
		}
	}()

	// Создаем буфер для multipart формы с оптимизированным размером
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Открываем файл DOCX с буферизированным чтением
	file, err := os.OpenFile(docxPath, os.O_RDONLY, 0)
	if err != nil {
		metrics.GotenbergRequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to open DOCX file: %w", err)
	}
	defer file.Close()

	// Создаем часть формы для файла
	part, err := writer.CreateFormFile("files", filepath.Base(docxPath))
	if err != nil {
		metrics.GotenbergRequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// Используем буферизированное копирование для улучшения производительности
	copyBuf := make([]byte, 64*1024)
	if _, err := io.CopyBuffer(part, file, copyBuf); err != nil {
		metrics.GotenbergRequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Закрываем writer
	if err := writer.Close(); err != nil {
		metrics.GotenbergRequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Создаем запрос к Gotenberg с оптимизированными заголовками
	req, err := http.NewRequest("POST", c.baseURL+"/forms/libreoffice/convert", body)
	if err != nil {
		metrics.GotenbergRequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Connection", "keep-alive")

	// Отправляем запрос
	resp, err := c.client.Do(req)
	if err != nil {
		metrics.GotenbergRequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		metrics.GotenbergRequestsTotal.WithLabelValues(strconv.Itoa(resp.StatusCode)).Inc()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("conversion failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Читаем PDF из ответа с буферизацией
	responseBuf := new(bytes.Buffer)
	if _, err := io.Copy(responseBuf, resp.Body); err != nil {
		metrics.GotenbergRequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	metrics.GotenbergRequestsTotal.WithLabelValues("success").Inc()
	return responseBuf.Bytes(), nil
}
