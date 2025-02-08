FROM golang:1.21-alpine AS builder

WORKDIR /app

# Устанавливаем git
RUN apk add --no-cache git

COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/api/main.go

FROM python:3.9-alpine

WORKDIR /app

# Устанавливаем системные зависимости
RUN apk add --no-cache \
    gcc \
    musl-dev \
    python3-dev \
    libxml2-dev \
    libxslt-dev \
    jpeg-dev \
    zlib-dev \
    freetype-dev \
    lcms2-dev \
    openjpeg-dev \
    tiff-dev \
    tk-dev \
    tcl-dev \
    harfbuzz-dev \
    fribidi-dev

# Устанавливаем зависимости Python
RUN pip install --no-cache-dir docxtpl python-docx lxml Pillow

# Копируем собранное Go приложение
COPY --from=builder /app/main .
COPY --from=builder /app/scripts ./scripts
COPY --from=builder /app/internal/domain/pdf/templates ./internal/domain/pdf/templates

# Настройка переменных окружения
ENV GIN_MODE=release \
    LOG_LEVEL=info

EXPOSE 8080

CMD ["./main"] 