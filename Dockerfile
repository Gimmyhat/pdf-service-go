FROM golang:1.21-alpine AS builder

WORKDIR /app

# Устанавливаем git
RUN apk add --no-cache git

COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/api/main.go

FROM pypy:3.9-slim

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
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

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

EXPOSE 8080

CMD ["./main"] 