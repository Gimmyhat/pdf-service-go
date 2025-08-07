package statistics

import (
	"encoding/json"
	"fmt"
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
			request_log_id, docx_log_id, gotenberg_log_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
		ON CONFLICT (request_id) DO UPDATE SET
			timestamp = EXCLUDED.timestamp,
			success = EXCLUDED.success,
			http_status = EXCLUDED.http_status,
			duration_ns = EXCLUDED.duration_ns,
			error_category = EXCLUDED.error_category
	`

	_, err = p.db.Exec(query,
		detail.RequestID, detail.Timestamp, detail.Method, detail.Path,
		detail.ClientIP, detail.UserAgent, headersJSON, detail.BodyText,
		detail.BodySizeBytes, detail.Success, detail.HTTPStatus, detail.DurationNs,
		detail.ContentType, detail.HasSensitiveData, detail.ErrorCategory,
		detail.RequestLogID, detail.DocxLogID, detail.GotenbergLogID,
	)

	return err
}

// GetRequestDetail получает детальную информацию о запросе по request_id
func (p *PostgresDB) GetRequestDetail(requestID string) (*RequestDetail, error) {
	query := `
		SELECT 
			id, request_id, timestamp, method, path, client_ip, user_agent,
			headers, body_text, body_size_bytes, success, http_status, duration_ns,
			content_type, has_sensitive_data, error_category,
			request_log_id, docx_log_id, gotenberg_log_id
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
			headers, body_text, body_size_bytes, success, http_status, duration_ns,
			content_type, has_sensitive_data, error_category,
			request_log_id, docx_log_id, gotenberg_log_id
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
		var headersJSON []byte

		err := rows.Scan(
			&detail.ID, &detail.RequestID, &detail.Timestamp, &detail.Method,
			&detail.Path, &detail.ClientIP, &detail.UserAgent, &headersJSON,
			&detail.BodyText, &detail.BodySizeBytes, &detail.Success,
			&detail.HTTPStatus, &detail.DurationNs, &detail.ContentType,
			&detail.HasSensitiveData, &detail.ErrorCategory,
			&detail.RequestLogID, &detail.DocxLogID, &detail.GotenbergLogID,
		)
		if err != nil {
			return nil, err
		}

		if len(headersJSON) > 0 {
			if err := json.Unmarshal(headersJSON, &detail.Headers); err != nil {
				return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
			}
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
			request_log_id, docx_log_id, gotenberg_log_id
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
		var headersJSON []byte

		err := rows.Scan(
			&detail.ID, &detail.RequestID, &detail.Timestamp, &detail.Method,
			&detail.Path, &detail.ClientIP, &detail.UserAgent, &headersJSON,
			&detail.BodyText, &detail.BodySizeBytes, &detail.Success,
			&detail.HTTPStatus, &detail.DurationNs, &detail.ContentType,
			&detail.HasSensitiveData, &detail.ErrorCategory,
			&detail.RequestLogID, &detail.DocxLogID, &detail.GotenbergLogID,
		)
		if err != nil {
			return nil, err
		}

		if len(headersJSON) > 0 {
			if err := json.Unmarshal(headersJSON, &detail.Headers); err != nil {
				return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
			}
		}

		details = append(details, detail)
	}

	return details, nil
}
