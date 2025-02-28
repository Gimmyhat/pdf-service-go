# PDF Service

Сервис для генерации PDF документов из DOCX шаблонов с использованием Gotenberg.

## Особенности

- Конвертация DOCX в PDF с помощью Gotenberg
- Поддержка шаблонов DOCX с использованием Python (docxtpl)
- Мониторинг с Prometheus и Grafana
- Трейсинг с OpenTelemetry и Jaeger
- Circuit Breaker и Retry механизмы
- Поддержка Kubernetes (Helm charts)
- Docker Compose для локальной разработки

## Требования

- Go 1.22+
- Docker
- Python 3.9+ (PyPy)
- Kubernetes (для production)
- Helm 3.x (для production)

## Быстрый старт

1. Клонируйте репозиторий:
```bash
git clone https://github.com/gimmyhat/pdf-service-go.git
cd pdf-service-go
```

2. Создайте новую версию и запустите сервисы:
```bash
make new-version
make deploy-local
```

3. Проверьте работу сервиса:
```bash
curl -X POST -H "Content-Type: application/json" --data-binary "@example.json" http://localhost:8080/api/v1/docx -o result.pdf
```

## Разработка

Основные команды:

```bash
# Сборка
make build

# Тесты
make test

# Линтер
make lint

# Локальный запуск
make dev

# Запуск в Docker
make run-local
```

## Деплой

### Локальное окружение

```bash
# Создание новой версии
make new-version

# Деплой локально
make deploy-local
```

### Kubernetes

```bash
# Деплой в test
make deploy ENV=test

# Деплой в production
make deploy ENV=prod
```

## Мониторинг

- Grafana: http://localhost:3000
- Prometheus: http://localhost:9090
- Jaeger UI: http://localhost:16686

## Документация

Дополнительная документация:
- [Команды (COMMANDS.md)](COMMANDS.md)
- [План развития (ROADMAP.md)](ROADMAP.md)

## Лицензия

MIT
