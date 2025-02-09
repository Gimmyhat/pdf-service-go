# Трейсинг в PDF Service

## Обзор

PDF Service использует OpenTelemetry для распределенного трейсинга. Это позволяет отслеживать путь запросов через различные компоненты системы и собирать метрики производительности.

## Компоненты

- **OpenTelemetry SDK**: Основная библиотека для инструментирования кода
- **Jaeger**: Система для сбора и визуализации трейсов
- **OTLP Exporter**: Экспортер для отправки трейсов в Jaeger
- **Prometheus**: Система для сбора метрик трейсинга
- **Grafana**: Визуализация метрик и алертинг

## Конфигурация

### Основные настройки

```env
# Основные параметры
OTEL_EXPORTER_OTLP_ENDPOINT=jaeger:4317
OTEL_SERVICE_NAME=nas-pdf-service
OTEL_SERVICE_VERSION=${VERSION}
OTEL_ENVIRONMENT=development

# Настройки протокола
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
OTEL_EXPORTER_OTLP_TIMEOUT=30s

# Сэмплирование
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=1.0  # 100% для dev, 0.1 для prod
```

### Лимиты и батчинг

```env
# Лимиты спанов
OTEL_ATTRIBUTE_VALUE_LENGTH_LIMIT=4096
OTEL_SPAN_ATTRIBUTE_COUNT_LIMIT=128
OTEL_SPAN_EVENT_COUNT_LIMIT=128

# Настройки батчинга
OTEL_BATCH_TIMEOUT=5
OTEL_MAX_EXPORT_BATCH_SIZE=512
OTEL_MAX_QUEUE_SIZE=2048
```

## Основные спаны

### HTTP Middleware

Каждый HTTP запрос создает спан со следующими атрибутами:

```go
// Базовая информация
http.method      // Метод запроса
http.url         // Полный URL
http.target      // Путь запроса
http.host        // Хост
http.user_agent  // User-Agent
http.client_ip   // IP клиента
http.status_code // Код ответа

// Идентификаторы
http.request_id     // X-Request-ID
http.correlation_id // X-Correlation-ID

// Метрики
http.duration_ms   // Длительность запроса
http.response_size // Размер ответа

// Системная информация
runtime.os            // Операционная система
runtime.arch         // Архитектура
runtime.num_cpu      // Количество CPU
runtime.num_goroutines // Количество горутин
```

### Обработка ошибок

При возникновении ошибок добавляется следующая информация:

```go
// Для ошибок 4xx/5xx
error.type    // Тип ошибки
error.message // Сообщение об ошибке
error.stack   // Стек вызовов (для паник)

// События
http.error    // Событие с деталями ошибки
```

## Интеграция с Kubernetes

Автоматически собирается информация о Kubernetes:

```go
k8s.pod.name    // Имя пода
k8s.namespace   // Namespace
k8s.cluster     // Имя кластера
cloud.provider  // Провайдер (kubernetes)
```

## Мониторинг

### Метрики Prometheus

Основные метрики:

```
# Экспорт трейсов
otel_trace_span_events_total          // Общее количество событий
otel_trace_span_duration_milliseconds // Длительность спанов
otel_trace_span_queue_size           // Размер очереди
otel_trace_span_drops_total          // Количество отброшенных спанов

# HTTP метрики
http_request_duration_seconds        // Длительность запросов
http_request_size_bytes             // Размер запросов
http_response_size_bytes            // Размер ответов
http_requests_total                 // Общее количество запросов
```

### Grafana дашборды

1. **Общий обзор**:
   - Количество запросов в секунду
   - Средняя латентность
   - Количество ошибок
   - Размер очереди трейсов

2. **Детальная статистика**:
   - Распределение латентности по эндпоинтам
   - Топ медленных запросов
   - Количество спанов по типам
   - Размер спанов

3. **Ошибки и алерты**:
   - Количество ошибок по типам
   - Количество отброшенных спанов
   - История алертов
   - Статус экспортера

## Лучшие практики

1. **Именование спанов**:
   - Используйте формат `component.operation`
   - Добавляйте контекст в имя: `http.get.users`

2. **Атрибуты**:
   - Используйте стандартные имена из OpenTelemetry
   - Не добавляйте чувствительные данные
   - Следите за размером атрибутов

3. **Сэмплирование**:
   - Development: 100% (`1.0`)
   - Production: 10% (`0.1`)
   - Важные операции: 100%

4. **Обработка ошибок**:
   - Всегда записывайте стек ошибки
   - Добавляйте контекст ошибки
   - Используйте события для деталей

## Отладка

### Частые проблемы

1. **Потеря трейсов**:
   - Проверьте подключение к Jaeger
   - Проверьте настройки сэмплирования
   - Проверьте размер очереди

2. **Высокая латентность**:
   - Уменьшите размер батча
   - Увеличьте частоту отправки
   - Проверьте ресурсы Jaeger

3. **Большой размер трейсов**:
   - Уменьшите количество атрибутов
   - Фильтруйте ненужные спаны
   - Оптимизируйте размер событий

### Команды для диагностики

```bash
# Проверка статуса Jaeger
kubectl get pods -n print-serv -l app=nas-jaeger

# Логи Jaeger
kubectl logs -n print-serv -l app=nas-jaeger

# Метрики Prometheus
curl -s localhost:8080/metrics | grep otel_trace

# Проверка конфигурации
kubectl describe configmap nas-pdf-service-config -n print-serv
```

## Безопасность

1. **Чувствительные данные**:
   - Не записывайте токены и пароли
   - Фильтруйте персональные данные
   - Используйте маскирование

2. **Доступ к Jaeger UI**:
   - Ограничьте доступ по IP
   - Используйте аутентификацию
   - Логируйте доступ

3. **Хранение данных**:
   - Настройте retention period
   - Очищайте старые данные
   - Используйте сжатие

## Масштабирование

1. **Высокая нагрузка**:
   - Увеличьте размер батча
   - Уменьшите частоту сэмплирования
   - Масштабируйте Jaeger

2. **Оптимизация ресурсов**:
   - Мониторьте использование памяти
   - Настройте лимиты спанов
   - Оптимизируйте размер данных 