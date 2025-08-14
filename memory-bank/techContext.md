# Tech Context - PDF Service Go

## Технологический стек

### Backend

- **Go 1.22+**: Основной язык разработки
- **Gin**: HTTP web framework
- **Zap**: Структурированное логирование
- **OpenTelemetry**: Трейсинг и метрики
- **PostgreSQL**: База данных для статистики и ошибок
- **Docker**: Контейнеризация

### Template Processing

- **Python 3.9+ (PyPy)**: Обработка DOCX шаблонов
- **docxtpl**: Библиотека для работы с DOCX шаблонами
- **Gotenberg**: Конвертация DOCX в PDF

### Infrastructure

- **Kubernetes**: Оркестрация контейнеров
- **Helm**: Управление Kubernetes манифестами
- **Docker Compose**: Локальная разработка

### Monitoring & Observability

- **Prometheus**: Сбор метрик
- **Grafana**: Визуализация метрик
- **Jaeger**: Distributed tracing
- **Custom Error Tracking**: Веб-интерфейс для анализа ошибок

### Development Tools

- **Make**: Автоматизация сборки и развертывания
- **golangci-lint**: Статический анализ Go кода
- **Git**: Версионирование
- **PowerShell**: Скрипты автоматизации (Windows)

## Конфигурация окружений

### Переменные окружения

```bash
# Gotenberg integration
GOTENBERG_API_URL=http://nas-pdf-service-gotenberg:3000
GOTENBERG_CLIENT_TIMEOUT=300s

# Database
POSTGRES_HOST=nas-pdf-service-postgres
POSTGRES_PORT=5432
POSTGRES_DB=pdf_service
POSTGRES_USER=pdf_service
POSTGRES_PASSWORD=pdf_service_pass

# Monitoring
PROMETHEUS_URL=http://nas-pdf-service-prometheus:9090
JAEGER_ENDPOINT=http://nas-jaeger:14268/api/traces

# Service timeouts
REQUEST_TIMEOUT=180s
```

### Docker Registry Configuration

```makefile
DOCKER_MIRROR = registry-irk-rw.devops.rgf.local
DOCKER_HUB_IMAGE = gimmyhat/pdf-service-go  # legacy
DOCKER_IMAGE = registry-irk-rw.devops.rgf.local/gimmyhat/pdf-service-go
```

### Kubernetes Contexts

- **efgi-test**: Тестовое окружение (172.27.239.30)
- **efgi-prod**: Продакшн окружение (172.27.239.2)

## Архитектурные ограничения

### Сетевые ограничения

- Нет прямого доступа к Docker Hub из кластеров
- Используется внутренний реестр Nexus `registry-irk-rw.devops.rgf.local`
- Образы тянутся по TLS, доступ обеспечивается через `imagePullSecret`

### Ресурсные ограничения

- Memory limits для подов в Kubernetes
- CPU limits для предотвращения resource starvation
- Disk space для временных файлов и логов

### Безопасность

- Service accounts в Kubernetes с ограниченными правами
- Secrets для чувствительных данных (пароли БД)
- Network policies для изоляции сервисов

## Зависимости и версии

### Go Dependencies

```go
github.com/gin-gonic/gin v1.10.0
github.com/lib/pq v1.10.9
go.uber.org/zap v1.27.0
go.opentelemetry.io/otel v1.28.0
```

### Python Dependencies

```
docxtpl==0.16.7
python-docx==0.8.11
Jinja2==3.1.4
```

### Infrastructure Versions

- **Kubernetes**: 1.25+
- **Helm**: 3.x
- **PostgreSQL**: 15-alpine
- **Prometheus**: latest
- **Grafana**: latest
- **Jaeger**: 1.54
- **Gotenberg**: 7.10

## Development Setup

### Local Development

```bash
make build          # Сборка Go приложения
make dev            # Запуск локально
make run-local      # Запуск в Docker Compose
```

### Testing

```bash
make test           # Unit тесты
make lint           # Статический анализ
```

### Deployment

```bash
make new-version    # Создание новой версии
make deploy ENV=test # Развертывание в тест
make deploy ENV=prod # Развертывание в продакшн
```
