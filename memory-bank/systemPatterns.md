# System Patterns - PDF Service Go

## Архитектурные паттерны

### Микросервисная архитектура
- **nas-pdf-service**: Основной API сервис (Go)
- **nas-pdf-service-gotenberg**: Сервис конвертации (Gotenberg)
- **nas-pdf-service-postgres**: База данных для статистики и ошибок
- **nas-pdf-service-prometheus**: Сбор метрик
- **nas-grafana**: Визуализация метрик
- **nas-jaeger**: Трейсинг запросов

### Паттерны надежности
- **Circuit Breaker**: Защита от каскадных сбоев при обращении к Gotenberg
- **Retry with Exponential Backoff**: Повторные попытки при временных сбоях
- **Timeout Management**: Настраиваемые таймауты для разных операций
- **Health Checks**: Kubernetes probes для мониторинга состояния
 - **Lazy init for DB**: Ленивое получение `Statistics`/`PostgresDB` в хендлерах для корректной работы при поздней инициализации

### Паттерны наблюдаемости
- **Structured Logging**: JSON логи с контекстными полями (Zap)
- **Metrics Collection**: Prometheus метрики для бизнес и технических показателей
- **Distributed Tracing**: OpenTelemetry трейсы для end-to-end видимости
- **Error Tracking**: Централизованная система отслеживания ошибок

## Ключевые компоненты

### API Layer (`internal/api/`)
- **server.go**: Gin router с middleware, роутинг веб-интерфейсов
- **handlers/**: HTTP обработчики для разных endpoints
- **middleware**: Логирование, трейсинг, восстановление после паники

### Web Interface (`internal/static/`)
- **dashboard.html**: Единый дашборд объединяющий статистику и ошибки
- **index.html**: Детальная статистика (legacy, сохранена для совместимости)
- **errors.html**: Анализ ошибок (legacy, сохранена для совместимости)
- **js/dashboard.js**: Объединенная логика для единого дашборда; модалка деталей запроса парсит `{ request_detail: ... }`, поля защищены от `undefined`

### Business Logic (`internal/domain/`)
- **pdf/**: Логика генерации PDF
- **templates/**: Управление DOCX шаблонами
- **validation**: Валидация входных данных

### Infrastructure (`internal/pkg/`)
- **statistics/**: Работа с PostgreSQL, система отслеживания ошибок. Архив выбирается с фильтром `path IN ('/api/v1/docx','/generate-pdf')`; добавлены индексы `path` и `(path, timestamp)`
  - Добавлен `InitializeOrRetry` вместо единоразового `Initialize` для безопасной инициализации с ретраями
- **errortracker/**: Централизованное логирование ошибок
- **config/**: Управление конфигурацией через environment variables

## Паттерны развертывания

### Docker Registry Strategy
```
Local Development → Docker Hub → Russian Mirror → Kubernetes Cluster
```

### Environment Management
- **Makefile variables**: Централизованная конфигурация
- **Kubernetes contexts**: Разделение test/prod окружений
- **ConfigMaps**: Внешняя конфигурация для шаблонов и настроек

### Automated Deployment Pipeline
1. `make new-version` - Сборка и push образа
2. `make deploy ENV=prod` - Развертывание в кластере
3. `make test-error-system ENV=prod` - Проверка работоспособности

## Паттерны данных

### Error Tracking Schema
```sql
error_logs (
    timestamp, request_id, trace_id, span_id,
    error_type, component, message, stack_trace,
    request_details JSONB, http_context,
    severity
)
```

### Statistics Aggregation
- Временные метрики по запросам
- Группировка ошибок по типам и компонентам
- Трендовый анализ производительности
 - Архив запросов: хранение путей к файлам на диске и очистка «оставить N»

## Паттерны интеграции

### Service Communication
- **HTTP REST**: Синхронная коммуникация между сервисами
- **Health Endpoints**: Стандартизированные проверки состояния
- **Graceful Shutdown**: Корректное завершение при обновлениях
- **Request Capture Middleware**: Автоматический перехват и логирование всех HTTP запросов

### External Dependencies
- **Gotenberg API**: Изоляция через interface и mock для тестов
- **PostgreSQL**: Repository pattern с абстракцией БД, оптимизированным для NFS storage
- **File System**: Управление временными файлами с cleanup

### Web Interface Patterns
- **Unified Dashboard**: Единый интерфейс `/dashboard` с табами для разных типов данных
- **Modal Windows**: Детальный просмотр данных без перехода на новые страницы
- **Real-time Data Fetching**: Асинхронная загрузка данных через REST API
- **Hybrid Data Integration**: Комбинирование данных из разных источников (старые статистические таблицы + новая система детального логирования)
- **Progressive Enhancement**: Базовая функциональность + дополнительные возможности (копирование, форматирование JSON)