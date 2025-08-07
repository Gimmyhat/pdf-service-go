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

-- Новая таблица для детального логирования запросов
CREATE TABLE IF NOT EXISTS request_details (
    id SERIAL PRIMARY KEY,
    request_id TEXT UNIQUE NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- HTTP детали
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    client_ip TEXT,
    user_agent TEXT,
    
    -- Содержимое запроса
    headers JSONB,
    body_text TEXT,
    body_size_bytes BIGINT,
    
    -- Статус обработки
    success BOOLEAN NOT NULL,
    http_status INTEGER,
    duration_ns BIGINT,
    
    -- Метаданные для анализа
    content_type TEXT,
    has_sensitive_data BOOLEAN DEFAULT false,
    error_category TEXT,
    
    -- Связи с другими таблицами
    request_log_id INTEGER REFERENCES request_logs(id),
    docx_log_id INTEGER REFERENCES docx_logs(id),
    gotenberg_log_id INTEGER REFERENCES gotenberg_logs(id)
);

CREATE INDEX IF NOT EXISTS idx_request_logs_timestamp ON request_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_docx_logs_timestamp ON docx_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_gotenberg_logs_timestamp ON gotenberg_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_pdf_logs_timestamp ON pdf_logs(timestamp);

-- Индексы для новой таблицы
CREATE INDEX IF NOT EXISTS idx_request_details_timestamp ON request_details(timestamp);
CREATE INDEX IF NOT EXISTS idx_request_details_request_id ON request_details(request_id);
CREATE INDEX IF NOT EXISTS idx_request_details_success ON request_details(success);
CREATE INDEX IF NOT EXISTS idx_request_details_error_category ON request_details(error_category); 