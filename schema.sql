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