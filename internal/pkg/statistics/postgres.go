package statistics

import (
	"database/sql"
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

		CREATE INDEX IF NOT EXISTS idx_request_logs_timestamp ON request_logs(timestamp);
		CREATE INDEX IF NOT EXISTS idx_docx_logs_timestamp ON docx_logs(timestamp);
		CREATE INDEX IF NOT EXISTS idx_gotenberg_logs_timestamp ON gotenberg_logs(timestamp);
		CREATE INDEX IF NOT EXISTS idx_pdf_logs_timestamp ON pdf_logs(timestamp);
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
