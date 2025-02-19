FROM golang:1.22-bullseye AS builder

WORKDIR /app

# Устанавливаем git и SQLite
RUN apt-get update && apt-get install -y \
    git \
    sqlite3 \
    libsqlite3-dev \
    gcc \
    && rm -rf /var/lib/apt/lists/*

COPY . .
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o main cmd/api/main.go

FROM pypy:3.9-slim-bullseye

WORKDIR /app

# Устанавливаем системные зависимости
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    gcc \
    python3-dev \
    libxml2-dev \
    libxslt-dev \
    libjpeg-dev \
    zlib1g-dev \
    libfreetype-dev \
    liblcms2-dev \
    libopenjp2-7-dev \
    libtiff-dev \
    tk-dev \
    tcl-dev \
    libharfbuzz-dev \
    libfribidi-dev \
    sqlite3 \
    libsqlite3-0 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Создаем директорию для базы данных
RUN mkdir -p /app/data && chown -R nobody:nogroup /app/data

# Копируем requirements-pypy.txt и устанавливаем зависимости Python
COPY --from=builder /app/requirements-pypy.txt .
RUN pypy3 -m pip install --no-cache-dir -r requirements-pypy.txt

# Копируем собранное Go приложение и необходимые файлы
COPY --from=builder /app/main .
COPY --from=builder /app/scripts ./scripts
COPY --from=builder /app/internal/domain/pdf/templates ./internal/domain/pdf/templates
COPY --from=builder /app/internal/static ./internal/static

# Настройка переменных окружения
ENV GIN_MODE=release \
    LOG_LEVEL=info \
    PYTHON_IMPLEMENTATION=pypy3

# Делаем исполняемый файл запускаемым
RUN chmod +x /app/main

# Переключаемся на непривилегированного пользователя
USER nobody

EXPOSE 8080

CMD ["./main"] 