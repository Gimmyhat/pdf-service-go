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
	stats := &Stats{
		Requests: RequestStats{
			RequestsByDay:  make(map[time.Weekday]uint64),
			RequestsByHour: make(map[int]uint64),
		},
		Docx:      DocxStats{},
		Gotenberg: GotenbergStats{},
		PDF:       PDFStats{},
	}

	// Запрашиваем статистику запросов
	row := p.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN success = true THEN 1 ELSE 0 END) as success,
			SUM(CASE WHEN success = false THEN 1 ELSE 0 END) as failed,
			CAST(AVG(CAST(duration_ns AS FLOAT)) AS BIGINT) as avg_duration,
			MIN(duration_ns) as min_duration,
			MAX(duration_ns) as max_duration,
			MAX(timestamp) as last_updated
		FROM request_logs
		WHERE timestamp >= $1
	`, since.UTC())

	var avgDuration, minDuration, maxDuration sql.NullInt64
	var lastUpdated sql.NullTime
	if err := row.Scan(
		&stats.Requests.TotalRequests,
		&stats.Requests.SuccessRequests,
		&stats.Requests.FailedRequests,
		&avgDuration,
		&minDuration,
		&maxDuration,
		&lastUpdated,
	); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Запрашиваем статистику DOCX
	row = p.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN has_error = true THEN 1 ELSE 0 END) as errors,
			CAST(AVG(CAST(duration_ns AS FLOAT)) AS BIGINT) as avg_duration,
			MIN(duration_ns) as min_duration,
			MAX(duration_ns) as max_duration,
			MAX(timestamp) as last_generation
		FROM docx_logs
		WHERE timestamp >= $1
	`, since.UTC())

	var docxAvgDuration, docxMinDuration, docxMaxDuration sql.NullInt64
	var lastGeneration sql.NullTime
	if err := row.Scan(
		&stats.Docx.TotalGenerations,
		&stats.Docx.ErrorGenerations,
		&docxAvgDuration,
		&docxMinDuration,
		&docxMaxDuration,
		&lastGeneration,
	); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Запрашиваем статистику Gotenberg
	row = p.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN has_error = true THEN 1 ELSE 0 END) as errors,
			CAST(AVG(CAST(duration_ns AS FLOAT)) AS BIGINT) as avg_duration,
			MIN(duration_ns) as min_duration,
			MAX(duration_ns) as max_duration,
			MAX(timestamp) as last_request
		FROM gotenberg_logs
		WHERE timestamp >= $1
	`, since.UTC())

	var gotenbergAvgDuration, gotenbergMinDuration, gotenbergMaxDuration sql.NullInt64
	var lastRequest sql.NullTime
	if err := row.Scan(
		&stats.Gotenberg.TotalRequests,
		&stats.Gotenberg.ErrorRequests,
		&gotenbergAvgDuration,
		&gotenbergMinDuration,
		&gotenbergMaxDuration,
		&lastRequest,
	); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Запрашиваем статистику PDF
	row = p.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			CAST(AVG(CAST(size_bytes AS FLOAT)) AS BIGINT) as avg_size,
			MIN(size_bytes) as min_size,
			MAX(size_bytes) as max_size,
			MAX(timestamp) as last_processed
		FROM pdf_logs
		WHERE timestamp >= $1
	`, since.UTC())

	var avgSize, minSize, maxSize sql.NullInt64
	var lastProcessed sql.NullTime
	if err := row.Scan(
		&stats.PDF.TotalFiles,
		&avgSize,
		&minSize,
		&maxSize,
		&lastProcessed,
	); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Устанавливаем значения длительностей и размеров
	if stats.Requests.TotalRequests > 0 {
		if avgDuration.Valid {
			stats.Requests.TotalDuration = time.Duration(avgDuration.Int64) * time.Duration(stats.Requests.TotalRequests)
		}
		if minDuration.Valid {
			stats.Requests.MinDuration = time.Duration(minDuration.Int64)
		}
		if maxDuration.Valid {
			stats.Requests.MaxDuration = time.Duration(maxDuration.Int64)
		}
		if lastUpdated.Valid {
			stats.Requests.LastUpdated = lastUpdated.Time
		}
	}

	if stats.Docx.TotalGenerations > 0 {
		if docxAvgDuration.Valid {
			stats.Docx.TotalDuration = time.Duration(docxAvgDuration.Int64) * time.Duration(stats.Docx.TotalGenerations)
		}
		if docxMinDuration.Valid {
			stats.Docx.MinDuration = time.Duration(docxMinDuration.Int64)
		}
		if docxMaxDuration.Valid {
			stats.Docx.MaxDuration = time.Duration(docxMaxDuration.Int64)
		}
		if lastGeneration.Valid {
			stats.Docx.LastGenerationTime = lastGeneration.Time
		}
	}

	if stats.Gotenberg.TotalRequests > 0 {
		if gotenbergAvgDuration.Valid {
			stats.Gotenberg.TotalDuration = time.Duration(gotenbergAvgDuration.Int64) * time.Duration(stats.Gotenberg.TotalRequests)
		}
		if gotenbergMinDuration.Valid {
			stats.Gotenberg.MinDuration = time.Duration(gotenbergMinDuration.Int64)
		}
		if gotenbergMaxDuration.Valid {
			stats.Gotenberg.MaxDuration = time.Duration(gotenbergMaxDuration.Int64)
		}
		if lastRequest.Valid {
			stats.Gotenberg.LastRequestTime = lastRequest.Time
		}
	}

	if stats.PDF.TotalFiles > 0 {
		if avgSize.Valid {
			stats.PDF.TotalSize = avgSize.Int64 * int64(stats.PDF.TotalFiles)
			stats.PDF.AverageSize = float64(avgSize.Int64)
		}
		if minSize.Valid {
			stats.PDF.MinSize = minSize.Int64
		}
		if maxSize.Valid {
			stats.PDF.MaxSize = maxSize.Int64
		}
		if lastProcessed.Valid {
			stats.PDF.LastProcessedTime = lastProcessed.Time
		}
	}

	// Запрашиваем распределение по дням недели
	rows, err := p.db.Query(`
		SELECT 
			EXTRACT(DOW FROM timestamp) as day_of_week,
			COUNT(*) as count
		FROM request_logs
		WHERE timestamp >= $1
		GROUP BY day_of_week
	`, since.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var dayFloat float64
		var count uint64
		if err := rows.Scan(&dayFloat, &count); err != nil {
			return nil, err
		}
		// PostgreSQL возвращает 0 для воскресенья, 1-6 для пн-сб
		// Go использует 0 для воскресенья, 1-6 для пн-сб, так что преобразование не требуется
		day := time.Weekday(int(dayFloat))
		stats.Requests.RequestsByDay[day] = count
	}

	// Запрашиваем распределение по часам
	rows, err = p.db.Query(`
		SELECT 
			EXTRACT(HOUR FROM timestamp) as hour,
			COUNT(*) as count
		FROM request_logs
		WHERE timestamp >= $1
		GROUP BY hour
	`, since.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var hourFloat float64
		var count uint64
		if err := rows.Scan(&hourFloat, &count); err != nil {
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
