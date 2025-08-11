package statistics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// === МЕТОДЫ ДЛЯ РАБОТЫ С ДЕТАЛЬНЫМИ ЗАПРОСАМИ ===

// SaveRequestDetail сохраняет детальную информацию о запросе
func (p *PostgresDB) SaveRequestDetail(detail *RequestDetail) error {
	headersJSON, err := json.Marshal(detail.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	query := `
        INSERT INTO request_details (
            request_id, timestamp, method, path, client_ip, user_agent,
            headers, body_text, body_size_bytes, success, http_status, duration_ns,
            content_type, has_sensitive_data, error_category,
            request_log_id, docx_log_id, gotenberg_log_id,
            request_file_path, result_file_path, result_size_bytes
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
        )
        ON CONFLICT (request_id) DO UPDATE SET
            timestamp = EXCLUDED.timestamp,
            success = EXCLUDED.success,
            http_status = EXCLUDED.http_status,
            duration_ns = EXCLUDED.duration_ns,
            error_category = EXCLUDED.error_category,
            request_file_path = COALESCE(EXCLUDED.request_file_path, request_details.request_file_path),
            result_file_path = COALESCE(EXCLUDED.result_file_path, request_details.result_file_path),
            result_size_bytes = COALESCE(EXCLUDED.result_size_bytes, request_details.result_size_bytes)
    `

	_, err = p.db.Exec(query,
		detail.RequestID, detail.Timestamp, detail.Method, detail.Path,
		detail.ClientIP, detail.UserAgent, headersJSON, detail.BodyText,
		detail.BodySizeBytes, detail.Success, detail.HTTPStatus, detail.DurationNs,
		detail.ContentType, detail.HasSensitiveData, detail.ErrorCategory,
		detail.RequestLogID, detail.DocxLogID, detail.GotenbergLogID,
		detail.RequestFilePath, detail.ResultFilePath, detail.ResultSizeBytes,
	)

	return err
}

// UpdateResultFileInfo обновляет путь и размер результата для запроса
func (p *PostgresDB) UpdateResultFileInfo(requestID string, resultPath string, resultSize int64) error {
	query := `
        UPDATE request_details
        SET result_file_path = $1, result_size_bytes = $2
        WHERE request_id = $3
    `
	_, err := p.db.Exec(query, resultPath, resultSize, requestID)
	return err
}

// GetRequestDetail получает детальную информацию о запросе по request_id
func (p *PostgresDB) GetRequestDetail(requestID string) (*RequestDetail, error) {
	query := `
		SELECT 
            id, request_id, timestamp, method, path, client_ip, user_agent,
            headers, body_text, body_size_bytes, success, http_status, duration_ns,
            content_type, has_sensitive_data, error_category,
            request_log_id, docx_log_id, gotenberg_log_id,
            request_file_path, result_file_path, result_size_bytes
		FROM request_details
		WHERE request_id = $1
	`

	var detail RequestDetail
	var headersJSON []byte

	err := p.db.QueryRow(query, requestID).Scan(
		&detail.ID, &detail.RequestID, &detail.Timestamp, &detail.Method,
		&detail.Path, &detail.ClientIP, &detail.UserAgent, &headersJSON,
		&detail.BodyText, &detail.BodySizeBytes, &detail.Success,
		&detail.HTTPStatus, &detail.DurationNs, &detail.ContentType,
		&detail.HasSensitiveData, &detail.ErrorCategory,
		&detail.RequestLogID, &detail.DocxLogID, &detail.GotenbergLogID,
		&detail.RequestFilePath, &detail.ResultFilePath, &detail.ResultSizeBytes,
	)

	if err != nil {
		return nil, err
	}

	if len(headersJSON) > 0 {
		if err := json.Unmarshal(headersJSON, &detail.Headers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
		}
	}

	return &detail, nil
}

// GetRequestDetailsByError получает детальную информацию о запросах с ошибками
func (p *PostgresDB) GetRequestDetailsByError(limit int, since time.Time) ([]RequestDetail, error) {
	query := `
        SELECT 
            id, request_id, timestamp, method, path, client_ip, user_agent,
            body_size_bytes, success, http_status, duration_ns,
            error_category,
            request_file_path, result_file_path, result_size_bytes
        FROM request_details
        WHERE success = false AND timestamp >= $1
        ORDER BY timestamp DESC
        LIMIT $2
    `

	rows, err := p.db.Query(query, since.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var details []RequestDetail
	for rows.Next() {
		var detail RequestDetail

		err := rows.Scan(
			&detail.ID, &detail.RequestID, &detail.Timestamp, &detail.Method,
			&detail.Path, &detail.ClientIP, &detail.UserAgent,
			&detail.BodySizeBytes, &detail.Success,
			&detail.HTTPStatus, &detail.DurationNs,
			&detail.ErrorCategory,
			&detail.RequestFilePath, &detail.ResultFilePath, &detail.ResultSizeBytes,
		)
		if err != nil {
			return nil, err
		}

		details = append(details, detail)
	}

	return details, nil
}

// CleanupOldRequestDetails удаляет старые записи детальных запросов
func (p *PostgresDB) CleanupOldRequestDetails(retentionDays int) error {
	query := `
		DELETE FROM request_details 
		WHERE timestamp < NOW() - INTERVAL '%d days'
	`

	_, err := p.db.Exec(fmt.Sprintf(query, retentionDays))
	return err
}

// GetRequestDetailsByPattern получает запросы по паттерну ошибок
func (p *PostgresDB) GetRequestDetailsByPattern(errorCategory string, limit int, since time.Time) ([]RequestDetail, error) {
	query := `
		SELECT 
            id, request_id, timestamp, method, path, client_ip, user_agent,
            headers, body_text, body_size_bytes, success, http_status, duration_ns,
            content_type, has_sensitive_data, error_category,
            request_log_id, docx_log_id, gotenberg_log_id,
            request_file_path, result_file_path, result_size_bytes
		FROM request_details
		WHERE error_category = $1 AND timestamp >= $2
		ORDER BY timestamp DESC
		LIMIT $3
	`

	rows, err := p.db.Query(query, errorCategory, since.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var details []RequestDetail
	for rows.Next() {
		var detail RequestDetail

		err := rows.Scan(
			&detail.ID, &detail.RequestID, &detail.Timestamp, &detail.Method,
			&detail.Path, &detail.ClientIP, &detail.UserAgent,
			&detail.BodySizeBytes, &detail.Success,
			&detail.HTTPStatus, &detail.DurationNs,
			&detail.ErrorCategory,
			&detail.RequestFilePath, &detail.ResultFilePath, &detail.ResultSizeBytes,
		)
		if err != nil {
			return nil, err
		}

		details = append(details, detail)
	}

	return details, nil
}

// GetRecentRequestsCtx возвращает последние запросы (успешные и с ошибками) с поддержкой контекста
func (p *PostgresDB) GetRecentRequestsCtx(ctx context.Context, limit int) ([]RequestDetail, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	// Лёгкая выборка без тяжёлых полей (headers, body_text, content_type и т.п.)
	query := `
        SELECT 
            id, request_id, timestamp, method, path, client_ip, user_agent,
            body_size_bytes, success, http_status, duration_ns,
            request_file_path, result_file_path, result_size_bytes
        FROM request_details
        WHERE path IN ('/api/v1/docx', '/generate-pdf')
        ORDER BY timestamp DESC
        LIMIT $1
    `

	rows, err := p.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var details []RequestDetail
	for rows.Next() {
		var detail RequestDetail
		if err := rows.Scan(
			&detail.ID, &detail.RequestID, &detail.Timestamp, &detail.Method,
			&detail.Path, &detail.ClientIP, &detail.UserAgent,
			&detail.BodySizeBytes, &detail.Success, &detail.HTTPStatus, &detail.DurationNs,
			&detail.RequestFilePath, &detail.ResultFilePath, &detail.ResultSizeBytes,
		); err != nil {
			return nil, err
		}
		details = append(details, detail)
	}
	return details, nil
}

// GetRecentRequests — обёртка без контекста для обратной совместимости
func (p *PostgresDB) GetRecentRequests(limit int) ([]RequestDetail, error) {
	return p.GetRecentRequestsCtx(context.TODO(), limit)
}

// GetRecentRequestsWithPaginationCtx возвращает последние запросы с пагинацией и признаком наличия следующей страницы (без COUNT(*))
func (p *PostgresDB) GetRecentRequestsWithPaginationCtx(ctx context.Context, limit, offset int) ([]RequestDetail, bool, error) {
	if limit <= 0 || limit > 1000 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}

	// Лёгкая выборка с пагинацией без тяжёлых полей
	// Берём на одну запись больше, чтобы определить hasMore без дорогого COUNT(*)
	query := `
		SELECT 
			id, request_id, timestamp, method, path, client_ip, user_agent,
			body_size_bytes, success, http_status, duration_ns,
			request_file_path, result_file_path, result_size_bytes
		FROM request_details
		WHERE path IN ('/api/v1/docx', '/generate-pdf')
		ORDER BY timestamp DESC
        LIMIT $1 OFFSET $2
	`

	// Используем limitPlusOne для детекции hasMore
	limitPlusOne := limit + 1

	rows, err := p.db.QueryContext(ctx, query, limitPlusOne, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var details []RequestDetail
	for rows.Next() {
		var detail RequestDetail
		if err := rows.Scan(
			&detail.ID, &detail.RequestID, &detail.Timestamp, &detail.Method,
			&detail.Path, &detail.ClientIP, &detail.UserAgent,
			&detail.BodySizeBytes, &detail.Success, &detail.HTTPStatus, &detail.DurationNs,
			&detail.RequestFilePath, &detail.ResultFilePath, &detail.ResultSizeBytes,
		); err != nil {
			return nil, false, err
		}
		details = append(details, detail)
	}

	hasMore := false
	if len(details) > limit {
		hasMore = true
		details = details[:limit]
	}

	return details, hasMore, nil
}

// CleanupOldRequestArtifactsKeepLast удаляет записи и файлы, оставляя только последние keep записей
func (p *PostgresDB) CleanupOldRequestArtifactsKeepLast(keep int) error {
	if keep <= 0 {
		keep = 100
	}
	// Находим пороговую метку времени (timestamp) на позиции keep
	var cutoff time.Time
	err := p.db.QueryRow(`SELECT timestamp FROM request_details ORDER BY timestamp DESC OFFSET $1 LIMIT 1`, keep).Scan(&cutoff)
	if err != nil {
		// Если записей меньше keep, просто выходим без ошибки
		return nil
	}

	// Получаем пути файлов для удаления
	rows, err := p.db.Query(`SELECT request_file_path, result_file_path FROM request_details WHERE timestamp < $1`, cutoff)
	if err != nil {
		return err
	}
	defer rows.Close()

	var reqPaths []string
	var resPaths []string
	for rows.Next() {
		var reqPath, resPath *string
		if err := rows.Scan(&reqPath, &resPath); err != nil {
			return err
		}
		if reqPath != nil && *reqPath != "" {
			reqPaths = append(reqPaths, *reqPath)
		}
		if resPath != nil && *resPath != "" {
			resPaths = append(resPaths, *resPath)
		}
	}

	// Удаляем файлы на диске (игнорируем ошибки удаления)
	for _, pth := range append(reqPaths, resPaths...) {
		_ = os.Remove(pth)
	}

	// Удаляем старые записи
	_, err = p.db.Exec(`DELETE FROM request_details WHERE timestamp < $1`, cutoff)
	return err
}
