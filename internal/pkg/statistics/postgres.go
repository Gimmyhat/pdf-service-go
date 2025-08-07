package statistics

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// PostgresDB представляет интерфейс для работы с PostgreSQL
type PostgresDB struct {
	connStr string
	db      *sql.DB
}

// NewPostgresDB создает новое подключение к PostgreSQL
func NewPostgresDB(host, port, dbname, user, password string) (*PostgresDB, error) {
	connStr := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		host, port, dbname, user, password)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	postgresDB := &PostgresDB{
		connStr: connStr,
		db:      db,
	}

	// Инициализируем схему базы данных
	if err := postgresDB.InitSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return postgresDB, nil
}

// InitSchema инициализирует схему базы данных
func (p *PostgresDB) InitSchema() error {
	_, err := p.db.Exec(`
		CREATE TABLE IF NOT EXISTS request_logs (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			path TEXT NOT NULL,
			method TEXT NOT NULL,
			duration_ns BIGINT NOT NULL,
			success BOOLEAN NOT NULL
		);

		CREATE TABLE IF NOT EXISTS docx_logs (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			duration_ns BIGINT NOT NULL,
			has_error BOOLEAN NOT NULL
		);

		CREATE TABLE IF NOT EXISTS gotenberg_logs (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			duration_ns BIGINT NOT NULL,
			has_error BOOLEAN NOT NULL
		);

		CREATE TABLE IF NOT EXISTS pdf_logs (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			size_bytes BIGINT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS error_logs (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			request_id TEXT,
			trace_id TEXT,
			span_id TEXT,
			error_type TEXT NOT NULL,
			component TEXT NOT NULL,
			message TEXT NOT NULL,
			stack_trace TEXT,
			request_details JSONB,
			client_ip TEXT,
			user_agent TEXT,
			http_method TEXT,
			http_path TEXT,
			http_status INTEGER,
			duration_ns BIGINT,
			severity TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_request_logs_timestamp ON request_logs(timestamp);
		CREATE INDEX IF NOT EXISTS idx_docx_logs_timestamp ON docx_logs(timestamp);
		CREATE INDEX IF NOT EXISTS idx_gotenberg_logs_timestamp ON gotenberg_logs(timestamp);
		CREATE INDEX IF NOT EXISTS idx_pdf_logs_timestamp ON pdf_logs(timestamp);
		CREATE INDEX IF NOT EXISTS idx_error_logs_timestamp ON error_logs(timestamp);
		CREATE INDEX IF NOT EXISTS idx_error_logs_type ON error_logs(error_type);
		CREATE INDEX IF NOT EXISTS idx_error_logs_component ON error_logs(component);
		CREATE INDEX IF NOT EXISTS idx_error_logs_severity ON error_logs(severity);
	`)

	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// LogRequest записывает информацию о запросе
func (p *PostgresDB) LogRequest(timestamp time.Time, path, method string, duration time.Duration, success bool) error {
	_, err := p.db.Exec(
		"INSERT INTO request_logs (timestamp, path, method, duration_ns, success) VALUES ($1, $2, $3, $4, $5)",
		timestamp.UTC(), path, method, duration.Nanoseconds(), success,
	)
	return err
}

// LogDocx записывает информацию о генерации DOCX
func (p *PostgresDB) LogDocx(timestamp time.Time, duration time.Duration, hasError bool) error {
	_, err := p.db.Exec(
		"INSERT INTO docx_logs (timestamp, duration_ns, has_error) VALUES ($1, $2, $3)",
		timestamp.UTC(), duration.Nanoseconds(), hasError,
	)
	return err
}

// LogGotenberg записывает информацию о запросе к Gotenberg
func (p *PostgresDB) LogGotenberg(timestamp time.Time, duration time.Duration, hasError bool) error {
	_, err := p.db.Exec(
		"INSERT INTO gotenberg_logs (timestamp, duration_ns, has_error) VALUES ($1, $2, $3)",
		timestamp.UTC(), duration.Nanoseconds(), hasError,
	)
	return err
}

// LogPDF записывает информацию о PDF файле
func (p *PostgresDB) LogPDF(timestamp time.Time, size int64) error {
	_, err := p.db.Exec(
		"INSERT INTO pdf_logs (timestamp, size_bytes) VALUES ($1, $2)",
		timestamp.UTC(), size,
	)
	return err
}

// GetStatistics возвращает статистику за указанный период
func (p *PostgresDB) GetStatistics(since time.Time) (*Stats, error) {
	// Отладочный вывод только для нулевой даты или необычных случаев
	if since.IsZero() {
		fmt.Printf("GetStatistics: using no time filter (all data)\n")
	}

	stats := &Stats{
		Requests: RequestStats{
			RequestsByDay:  make(map[time.Weekday]uint64),
			RequestsByHour: make(map[int]uint64),
		},
		Docx:      DocxStats{},
		Gotenberg: GotenbergStats{},
		PDF:       PDFStats{},
	}

	// Проверяем, что время не нулевое
	var whereClause string
	var params []interface{}

	if since.IsZero() {
		whereClause = ""
		params = []interface{}{}
	} else {
		whereClause = "WHERE timestamp >= $1"
		params = []interface{}{since.UTC()}
	}

	// Сначала проверим наличие данных в таблице
	var totalCount int64
	checkQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM request_logs
		%s
	`, whereClause)

	checkErr := p.db.QueryRow(checkQuery, params...).Scan(&totalCount)
	if checkErr != nil {
		return nil, checkErr
	}

	// Проверим временной диапазон данных
	var minTs, maxTs sql.NullTime
	rangeQuery := fmt.Sprintf(`
		SELECT MIN(timestamp), MAX(timestamp)
		FROM request_logs
		%s
	`, whereClause)

	rangeErr := p.db.QueryRow(rangeQuery, params...).Scan(&minTs, &maxTs)
	if rangeErr != nil {
		return nil, rangeErr
	}

	// Запрашиваем статистику запросов
	requestQuery := fmt.Sprintf(`
		SELECT 
			COALESCE(COUNT(*), 0) as total,
			COALESCE(SUM(CASE WHEN success = true THEN 1 ELSE 0 END), 0) as success,
			COALESCE(SUM(CASE WHEN success = false THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(CAST(AVG(CAST(duration_ns AS FLOAT)) AS BIGINT), 0) as avg_duration,
			COALESCE(MIN(duration_ns), 0) as min_duration,
			COALESCE(MAX(duration_ns), 0) as max_duration,
			MAX(timestamp) as last_updated
		FROM request_logs
		%s
	`, whereClause)

	var row *sql.Row
	if len(params) > 0 {
		row = p.db.QueryRow(requestQuery, params...)
	} else {
		row = p.db.QueryRow(requestQuery)
	}

	var avgDuration, minDuration, maxDuration int64
	var lastUpdated sql.NullTime
	if err := row.Scan(
		&stats.Requests.TotalRequests,
		&stats.Requests.SuccessRequests,
		&stats.Requests.FailedRequests,
		&avgDuration,
		&minDuration,
		&maxDuration,
		&lastUpdated,
	); err != nil {
		return nil, fmt.Errorf("error scanning request stats: %w", err)
	}

	// Запрашиваем статистику DOCX
	docxQuery := fmt.Sprintf(`
		SELECT 
			COALESCE(COUNT(*), 0) as total,
			COALESCE(SUM(CASE WHEN has_error = true THEN 1 ELSE 0 END), 0) as errors,
			COALESCE(CAST(AVG(CAST(duration_ns AS FLOAT)) AS BIGINT), 0) as avg_duration,
			COALESCE(MIN(duration_ns), 0) as min_duration,
			COALESCE(MAX(duration_ns), 0) as max_duration,
			MAX(timestamp) as last_generation
		FROM docx_logs
		%s
	`, whereClause)

	if len(params) > 0 {
		row = p.db.QueryRow(docxQuery, params...)
	} else {
		row = p.db.QueryRow(docxQuery)
	}

	var docxAvgDuration, docxMinDuration, docxMaxDuration int64
	var lastGeneration sql.NullTime
	if err := row.Scan(
		&stats.Docx.TotalGenerations,
		&stats.Docx.ErrorGenerations,
		&docxAvgDuration,
		&docxMinDuration,
		&docxMaxDuration,
		&lastGeneration,
	); err != nil {
		return nil, fmt.Errorf("error scanning docx stats: %w", err)
	}

	// Запрашиваем статистику Gotenberg
	gotenbergQuery := fmt.Sprintf(`
		SELECT 
			COALESCE(COUNT(*), 0) as total,
			COALESCE(SUM(CASE WHEN has_error = true THEN 1 ELSE 0 END), 0) as errors,
			COALESCE(CAST(AVG(CAST(duration_ns AS FLOAT)) AS BIGINT), 0) as avg_duration,
			COALESCE(MIN(duration_ns), 0) as min_duration,
			COALESCE(MAX(duration_ns), 0) as max_duration,
			MAX(timestamp) as last_request
		FROM gotenberg_logs
		%s
	`, whereClause)

	if len(params) > 0 {
		row = p.db.QueryRow(gotenbergQuery, params...)
	} else {
		row = p.db.QueryRow(gotenbergQuery)
	}

	var gotenbergAvgDuration, gotenbergMinDuration, gotenbergMaxDuration int64
	var lastRequest sql.NullTime
	if err := row.Scan(
		&stats.Gotenberg.TotalRequests,
		&stats.Gotenberg.ErrorRequests,
		&gotenbergAvgDuration,
		&gotenbergMinDuration,
		&gotenbergMaxDuration,
		&lastRequest,
	); err != nil {
		return nil, fmt.Errorf("error scanning gotenberg stats: %w", err)
	}

	// Запрашиваем статистику PDF
	pdfQuery := fmt.Sprintf(`
		SELECT 
			COALESCE(COUNT(*), 0) as total,
			COALESCE(CAST(AVG(CAST(size_bytes AS FLOAT)) AS BIGINT), 0) as avg_size,
			COALESCE(MIN(size_bytes), 0) as min_size,
			COALESCE(MAX(size_bytes), 0) as max_size,
			MAX(timestamp) as last_processed
		FROM pdf_logs
		%s
	`, whereClause)

	if len(params) > 0 {
		row = p.db.QueryRow(pdfQuery, params...)
	} else {
		row = p.db.QueryRow(pdfQuery)
	}

	var avgSize, minSize, maxSize int64
	var lastProcessed sql.NullTime
	if err := row.Scan(
		&stats.PDF.TotalFiles,
		&avgSize,
		&minSize,
		&maxSize,
		&lastProcessed,
	); err != nil {
		return nil, fmt.Errorf("error scanning pdf stats: %w", err)
	}

	// Устанавливаем значения длительностей и размеров
	if stats.Requests.TotalRequests > 0 {
		if avgDuration > 0 {
			stats.Requests.TotalDuration = time.Duration(avgDuration) * time.Duration(stats.Requests.TotalRequests)
		}
		if minDuration > 0 {
			stats.Requests.MinDuration = time.Duration(minDuration)
		}
		if maxDuration > 0 {
			stats.Requests.MaxDuration = time.Duration(maxDuration)
		}
		if lastUpdated.Valid {
			stats.Requests.LastUpdated = lastUpdated.Time
		}
	}

	if stats.Docx.TotalGenerations > 0 {
		if docxAvgDuration > 0 {
			stats.Docx.TotalDuration = time.Duration(docxAvgDuration) * time.Duration(stats.Docx.TotalGenerations)
		}
		if docxMinDuration > 0 {
			stats.Docx.MinDuration = time.Duration(docxMinDuration)
		}
		if docxMaxDuration > 0 {
			stats.Docx.MaxDuration = time.Duration(docxMaxDuration)
		}
		if lastGeneration.Valid {
			stats.Docx.LastGenerationTime = lastGeneration.Time
		}
	}

	if stats.Gotenberg.TotalRequests > 0 {
		if gotenbergAvgDuration > 0 {
			stats.Gotenberg.TotalDuration = time.Duration(gotenbergAvgDuration) * time.Duration(stats.Gotenberg.TotalRequests)
		}
		if gotenbergMinDuration > 0 {
			stats.Gotenberg.MinDuration = time.Duration(gotenbergMinDuration)
		}
		if gotenbergMaxDuration > 0 {
			stats.Gotenberg.MaxDuration = time.Duration(gotenbergMaxDuration)
		}
		if lastRequest.Valid {
			stats.Gotenberg.LastRequestTime = lastRequest.Time
		}
	}

	if stats.PDF.TotalFiles > 0 {
		if avgSize > 0 {
			stats.PDF.TotalSize = avgSize * int64(stats.PDF.TotalFiles)
			stats.PDF.AverageSize = float64(avgSize)
		}
		if minSize > 0 {
			stats.PDF.MinSize = minSize
		}
		if maxSize > 0 {
			stats.PDF.MaxSize = maxSize
		}
		if lastProcessed.Valid {
			stats.PDF.LastProcessedTime = lastProcessed.Time
		}
	}

	// Запрашиваем распределение по дням недели
	dayQuery := fmt.Sprintf(`
		WITH day_counts AS (
			SELECT 
				EXTRACT(DOW FROM timestamp) as day_number,
				COUNT(*) as count,
				MAX(timestamp) as utc_time
			FROM request_logs
			%s
			GROUP BY EXTRACT(DOW FROM timestamp)
		)
		SELECT 
			day_number,
			count,
			utc_time
		FROM day_counts
		ORDER BY day_number
	`, whereClause)

	var dayRows *sql.Rows
	var dayErr error
	if len(params) > 0 {
		dayRows, dayErr = p.db.Query(dayQuery, params...)
	} else {
		dayRows, dayErr = p.db.Query(dayQuery)
	}
	if dayErr != nil {
		return nil, fmt.Errorf("error querying days: %w", dayErr)
	}
	defer dayRows.Close()

	for dayRows.Next() {
		var dayNumber float64
		var count uint64
		var utcTime time.Time
		if err := dayRows.Scan(&dayNumber, &count, &utcTime); err != nil {
			return nil, fmt.Errorf("error scanning day row: %w", err)
		}

		day := time.Weekday(int(dayNumber))
		stats.Requests.RequestsByDay[day] = count
	}

	// Запрашиваем распределение по часам
	hourQuery := fmt.Sprintf(`
		WITH hour_counts AS (
			SELECT 
				EXTRACT(HOUR FROM timestamp) as hour,
				COUNT(*) as count
			FROM request_logs
			%s
			GROUP BY EXTRACT(HOUR FROM timestamp)
		)
		SELECT hour, count
		FROM hour_counts
		ORDER BY hour
	`, whereClause)

	var hourRows *sql.Rows
	var hourErr error
	if len(params) > 0 {
		hourRows, hourErr = p.db.Query(hourQuery, params...)
	} else {
		hourRows, hourErr = p.db.Query(hourQuery)
	}
	if hourErr != nil {
		return nil, hourErr
	}
	defer hourRows.Close()

	for hourRows.Next() {
		var hourFloat float64
		var count uint64
		if err := hourRows.Scan(&hourFloat, &count); err != nil {
			return nil, err
		}
		hour := int(hourFloat)
		stats.Requests.RequestsByHour[hour] = count
	}

	return stats, nil
}

// Close закрывает соединение с базой данных
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// LogError записывает детальную информацию об ошибке
func (p *PostgresDB) LogError(errorDetails *ErrorDetails) error {
	requestDetailsJSON, err := json.Marshal(errorDetails.RequestDetails)
	if err != nil {
		requestDetailsJSON = []byte("{}")
	}

	_, err = p.db.Exec(`
		INSERT INTO error_logs (
			timestamp, request_id, trace_id, span_id, error_type, component, 
			message, stack_trace, request_details, client_ip, user_agent, 
			http_method, http_path, http_status, duration_ns, severity
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		errorDetails.Timestamp.UTC(),
		errorDetails.RequestID,
		errorDetails.TraceID,
		errorDetails.SpanID,
		errorDetails.ErrorType,
		errorDetails.Component,
		errorDetails.Message,
		errorDetails.StackTrace,
		requestDetailsJSON,
		errorDetails.ClientIP,
		errorDetails.UserAgent,
		errorDetails.HTTPMethod,
		errorDetails.HTTPPath,
		errorDetails.HTTPStatus,
		errorDetails.Duration.Nanoseconds(),
		errorDetails.Severity,
	)
	return err
}

// GetRecentErrors возвращает последние ошибки (гибридный подход)
func (p *PostgresDB) GetRecentErrors(limit int, since time.Time) ([]ErrorDetails, error) {
	var allErrors []ErrorDetails

	// === ПОЛУЧАЕМ ERROR TRACKING ОШИБКИ (новая система) ===
	trackingErrors, err := p.getTrackingErrors(limit, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracking errors: %w", err)
	}
	allErrors = append(allErrors, trackingErrors...)

	// === ПОЛУЧАЕМ СТАТИСТИЧЕСКИЕ ОШИБКИ (историческая система) ===
	statisticalErrors, err := p.getStatisticalErrors(limit, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistical errors: %w", err)
	}
	allErrors = append(allErrors, statisticalErrors...)

	// === СОРТИРУЕМ ПО ВРЕМЕНИ И ОГРАНИЧИВАЕМ ===
	// Сортируем по убыванию времени (новые сначала)
	for i := 0; i < len(allErrors)-1; i++ {
		for j := i + 1; j < len(allErrors); j++ {
			if allErrors[i].Timestamp.Before(allErrors[j].Timestamp) {
				allErrors[i], allErrors[j] = allErrors[j], allErrors[i]
			}
		}
	}

	// Ограничиваем количество
	if len(allErrors) > limit {
		allErrors = allErrors[:limit]
	}

	return allErrors, nil
}

// GetErrorPatterns возвращает паттерны ошибок для анализа
func (p *PostgresDB) GetErrorPatterns(since time.Time) ([]ErrorPattern, error) {
	rows, err := p.db.Query(`
		SELECT 
			error_type,
			component,
			COUNT(*) as count,
			MAX(timestamp) as last_occurred,
			AVG(duration_ns) as avg_duration_ns
		FROM error_logs
		WHERE timestamp >= $1
		GROUP BY error_type, component
		ORDER BY count DESC
	`, since.UTC())

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []ErrorPattern
	for rows.Next() {
		var p ErrorPattern
		var avgDurationNs int64

		err := rows.Scan(
			&p.ErrorType, &p.Component, &p.Count,
			&p.LastOccured, &avgDurationNs,
		)
		if err != nil {
			return nil, err
		}

		p.AvgDuration = time.Duration(avgDurationNs)

		// Определяем частоту
		hours := time.Since(since).Hours()
		if hours > 0 {
			frequency := float64(p.Count) / hours
			if frequency >= 1 {
				p.Frequency = fmt.Sprintf("%.1f/hour", frequency)
			} else {
				p.Frequency = fmt.Sprintf("%.1f/day", frequency*24)
			}
		}

		// Простое определение тренда (можно улучшить)
		if p.Count > 10 {
			p.Trend = "high"
		} else if p.Count > 5 {
			p.Trend = "medium"
		} else {
			p.Trend = "low"
		}

		patterns = append(patterns, p)
	}

	return patterns, nil
}

// GetErrorCounts возвращает количество ошибок за разные периоды (гибридный подход)
func (p *PostgresDB) GetErrorCounts() (total, last24h, lastHour int64, err error) {
	now := time.Now()
	last24Hours := now.Add(-24 * time.Hour)
	lastHourTime := now.Add(-1 * time.Hour)

	// === ERROR TRACKING ОШИБКИ (новая система) ===
	var trackingTotal, tracking24h, trackingHour int64

	// Общее количество из error_logs
	err = p.db.QueryRow("SELECT COUNT(*) FROM error_logs").Scan(&trackingTotal)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to count tracking errors: %w", err)
	}

	// За последние 24 часа из error_logs
	err = p.db.QueryRow("SELECT COUNT(*) FROM error_logs WHERE timestamp >= $1", last24Hours.UTC()).Scan(&tracking24h)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to count tracking errors 24h: %w", err)
	}

	// За последний час из error_logs
	err = p.db.QueryRow("SELECT COUNT(*) FROM error_logs WHERE timestamp >= $1", lastHourTime.UTC()).Scan(&trackingHour)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to count tracking errors 1h: %w", err)
	}

	// === СТАТИСТИЧЕСКИЕ ОШИБКИ (историческая система) ===
	var statTotal, stat24h, statHour int64

	// Общее количество статистических ошибок
	err = p.db.QueryRow(`
		SELECT 
			COALESCE((SELECT COUNT(*) FROM request_logs WHERE success = false), 0) +
			COALESCE((SELECT COUNT(*) FROM docx_logs WHERE has_error = true), 0) +
			COALESCE((SELECT COUNT(*) FROM gotenberg_logs WHERE has_error = true), 0)
	`).Scan(&statTotal)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to count statistical errors: %w", err)
	}

	// За последние 24 часа статистических ошибок
	err = p.db.QueryRow(`
		SELECT 
			COALESCE((SELECT COUNT(*) FROM request_logs WHERE success = false AND timestamp >= $1), 0) +
			COALESCE((SELECT COUNT(*) FROM docx_logs WHERE has_error = true AND timestamp >= $1), 0) +
			COALESCE((SELECT COUNT(*) FROM gotenberg_logs WHERE has_error = true AND timestamp >= $1), 0)
	`, last24Hours.UTC()).Scan(&stat24h)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to count statistical errors 24h: %w", err)
	}

	// За последний час статистических ошибок
	err = p.db.QueryRow(`
		SELECT 
			COALESCE((SELECT COUNT(*) FROM request_logs WHERE success = false AND timestamp >= $1), 0) +
			COALESCE((SELECT COUNT(*) FROM docx_logs WHERE has_error = true AND timestamp >= $1), 0) +
			COALESCE((SELECT COUNT(*) FROM gotenberg_logs WHERE has_error = true AND timestamp >= $1), 0)
	`, lastHourTime.UTC()).Scan(&statHour)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to count statistical errors 1h: %w", err)
	}

	// === ОБЪЕДИНЯЕМ РЕЗУЛЬТАТЫ ===
	total = trackingTotal + statTotal
	last24h = tracking24h + stat24h
	lastHour = trackingHour + statHour

	return total, last24h, lastHour, nil
}

// getTrackingErrors получает ошибки из error_logs (новая система)
func (p *PostgresDB) getTrackingErrors(limit int, since time.Time) ([]ErrorDetails, error) {
	rows, err := p.db.Query(`
		SELECT 
			id, timestamp, request_id, trace_id, span_id, error_type, component,
			message, stack_trace, request_details, client_ip, user_agent,
			http_method, http_path, http_status, duration_ns, severity
		FROM error_logs
		WHERE timestamp >= $1
		ORDER BY timestamp DESC
		LIMIT $2
	`, since.UTC(), limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var errors []ErrorDetails
	for rows.Next() {
		var e ErrorDetails
		var requestDetailsJSON []byte
		var durationNs int64

		err := rows.Scan(
			&e.ID, &e.Timestamp, &e.RequestID, &e.TraceID, &e.SpanID,
			&e.ErrorType, &e.Component, &e.Message, &e.StackTrace,
			&requestDetailsJSON, &e.ClientIP, &e.UserAgent,
			&e.HTTPMethod, &e.HTTPPath, &e.HTTPStatus, &durationNs, &e.Severity,
		)
		if err != nil {
			return nil, err
		}

		e.Duration = time.Duration(durationNs)
		if len(requestDetailsJSON) > 0 {
			json.Unmarshal(requestDetailsJSON, &e.RequestDetails)
		}

		errors = append(errors, e)
	}

	return errors, nil
}

// getStatisticalErrors получает ошибки из статистических таблиц (историческая система)
func (p *PostgresDB) getStatisticalErrors(limit int, since time.Time) ([]ErrorDetails, error) {
	var errors []ErrorDetails

	// === ОШИБКИ ИЗ REQUEST_LOGS ===
	requestErrors, err := p.getRequestErrors(limit/3, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get request errors: %w", err)
	}
	errors = append(errors, requestErrors...)

	// === ОШИБКИ ИЗ DOCX_LOGS ===
	docxErrors, err := p.getDocxErrors(limit/3, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get docx errors: %w", err)
	}
	errors = append(errors, docxErrors...)

	// === ОШИБКИ ИЗ GOTENBERG_LOGS ===
	gotenbergErrors, err := p.getGotenbergErrors(limit/3, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get gotenberg errors: %w", err)
	}
	errors = append(errors, gotenbergErrors...)

	return errors, nil
}

// getRequestErrors получает ошибки из request_logs
func (p *PostgresDB) getRequestErrors(limit int, since time.Time) ([]ErrorDetails, error) {
	rows, err := p.db.Query(`
		SELECT 
			id, timestamp, path, method, duration_ns
		FROM request_logs
		WHERE success = false AND timestamp >= $1
		ORDER BY timestamp DESC
		LIMIT $2
	`, since.UTC(), limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var errors []ErrorDetails
	for rows.Next() {
		var e ErrorDetails
		var id int64
		var durationNs int64

		err := rows.Scan(&id, &e.Timestamp, &e.HTTPPath, &e.HTTPMethod, &durationNs)
		if err != nil {
			return nil, err
		}

		// Генерируем уникальный Request ID на базе ID записи
		e.RequestID = fmt.Sprintf("req_%d_%d", id, e.Timestamp.Unix())

		e.Component = "api"
		e.ErrorType = "request_failure"
		e.Duration = time.Duration(durationNs)

		// Более информативные сообщения на основе длительности
		if e.Duration > 60*time.Second {
			e.Severity = "high"
			e.Message = fmt.Sprintf("Таймаут HTTP запроса: %s %s (%.2f сек)",
				e.HTTPMethod, e.HTTPPath, e.Duration.Seconds())
			e.HTTPStatus = 504 // Gateway Timeout
		} else if e.Duration > 10*time.Second {
			e.Severity = "medium"
			e.Message = fmt.Sprintf("Медленный HTTP запрос с ошибкой: %s %s (%.2f сек)",
				e.HTTPMethod, e.HTTPPath, e.Duration.Seconds())
			e.HTTPStatus = 500
		} else {
			e.Severity = "medium"
			e.Message = fmt.Sprintf("Быстрая ошибка HTTP запроса: %s %s (возможна ошибка валидации или подключения)",
				e.HTTPMethod, e.HTTPPath)
			e.HTTPStatus = 400
		}

		e.RequestDetails = map[string]interface{}{
			"source":            "request_logs",
			"type":              "statistical_error",
			"duration_seconds":  e.Duration.Seconds(),
			"duration_category": getDurationCategory(e.Duration),
			"likely_cause":      getLikelyCause(e.HTTPPath, e.Duration),
		}

		errors = append(errors, e)
	}

	return errors, nil
}

// getDocxErrors получает ошибки из docx_logs
func (p *PostgresDB) getDocxErrors(limit int, since time.Time) ([]ErrorDetails, error) {
	rows, err := p.db.Query(`
		SELECT 
			id, timestamp, duration_ns
		FROM docx_logs
		WHERE has_error = true AND timestamp >= $1
		ORDER BY timestamp DESC
		LIMIT $2
	`, since.UTC(), limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var errors []ErrorDetails
	for rows.Next() {
		var e ErrorDetails
		var id int64
		var durationNs int64

		err := rows.Scan(&id, &e.Timestamp, &durationNs)
		if err != nil {
			return nil, err
		}

		// Генерируем уникальный Request ID на базе ID записи
		e.RequestID = fmt.Sprintf("docx_%d_%d", id, e.Timestamp.Unix())

		e.Component = "docx"
		e.ErrorType = "generation_failure"
		e.Duration = time.Duration(durationNs)
		e.HTTPPath = "/api/v1/docx"
		e.HTTPMethod = "POST"

		// Анализ ошибки на основе длительности
		if e.Duration < 1*time.Second {
			e.Severity = "critical"
			e.Message = "Критическая ошибка DOCX: мгновенный сбой (проблема с Python скриптом или файловой системой)"
			e.HTTPStatus = 500
		} else if e.Duration < 5*time.Second {
			e.Severity = "high"
			e.Message = fmt.Sprintf("Ошибка генерации DOCX: быстрый сбой (%.2f сек) - возможна проблема с шаблоном или данными", e.Duration.Seconds())
			e.HTTPStatus = 422 // Unprocessable Entity
		} else if e.Duration > 30*time.Second {
			e.Severity = "high"
			e.Message = fmt.Sprintf("Таймаут генерации DOCX: процесс завис (%.2f сек) - возможна проблема с Python или большим объемом данных", e.Duration.Seconds())
			e.HTTPStatus = 504 // Gateway Timeout
		} else {
			e.Severity = "high"
			e.Message = fmt.Sprintf("Ошибка генерации DOCX: стандартный сбой (%.2f сек) - проблема в логике обработки", e.Duration.Seconds())
			e.HTTPStatus = 500
		}

		e.RequestDetails = map[string]interface{}{
			"source":            "docx_logs",
			"type":              "statistical_error",
			"duration_seconds":  e.Duration.Seconds(),
			"duration_category": getDurationCategory(e.Duration),
			"likely_cause":      getDocxErrorCause(e.Duration),
			"troubleshooting":   getDocxTroubleshooting(e.Duration),
		}

		errors = append(errors, e)
	}

	return errors, nil
}

// getGotenbergErrors получает ошибки из gotenberg_logs
func (p *PostgresDB) getGotenbergErrors(limit int, since time.Time) ([]ErrorDetails, error) {
	rows, err := p.db.Query(`
		SELECT 
			id, timestamp, duration_ns
		FROM gotenberg_logs
		WHERE has_error = true AND timestamp >= $1
		ORDER BY timestamp DESC
		LIMIT $2
	`, since.UTC(), limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var errors []ErrorDetails
	for rows.Next() {
		var e ErrorDetails
		var id int64
		var durationNs int64

		err := rows.Scan(&id, &e.Timestamp, &durationNs)
		if err != nil {
			return nil, err
		}

		// Генерируем уникальный Request ID на базе ID записи
		e.RequestID = fmt.Sprintf("gotenberg_%d_%d", id, e.Timestamp.Unix())

		e.Component = "gotenberg"
		e.ErrorType = "conversion_failure"
		e.Duration = time.Duration(durationNs)
		e.HTTPPath = "/convert"
		e.HTTPMethod = "POST"

		// Анализ ошибки Gotenberg на основе длительности
		if e.Duration < 2*time.Second {
			e.Severity = "critical"
			e.Message = "Критическая ошибка Gotenberg: мгновенный сбой (сервис недоступен или некорректный запрос)"
			e.HTTPStatus = 503 // Service Unavailable
		} else if e.Duration > 120*time.Second {
			e.Severity = "high"
			e.Message = fmt.Sprintf("Таймаут Gotenberg: конвертация зависла (%.0f сек) - возможна проблема с LibreOffice или большим файлом", e.Duration.Seconds())
			e.HTTPStatus = 504 // Gateway Timeout
		} else if e.Duration > 60*time.Second {
			e.Severity = "medium"
			e.Message = fmt.Sprintf("Медленная ошибка Gotenberg: долгая конвертация (%.0f сек) - возможно сложный документ", e.Duration.Seconds())
			e.HTTPStatus = 500
		} else {
			e.Severity = "high"
			e.Message = fmt.Sprintf("Ошибка конвертации Gotenberg: стандартный сбой (%.2f сек) - проблема с форматом или содержимым документа", e.Duration.Seconds())
			e.HTTPStatus = 422 // Unprocessable Entity
		}

		e.RequestDetails = map[string]interface{}{
			"source":            "gotenberg_logs",
			"type":              "statistical_error",
			"duration_seconds":  e.Duration.Seconds(),
			"duration_category": getDurationCategory(e.Duration),
			"likely_cause":      getGotenbergErrorCause(e.Duration),
			"troubleshooting":   getGotenbergTroubleshooting(e.Duration),
		}

		errors = append(errors, e)
	}

	return errors, nil
}
