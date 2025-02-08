# Система кэширования шаблонов DOCX

## Обзор

Система кэширования шаблонов DOCX реализована для оптимизации производительности сервиса путем сохранения часто используемых шаблонов в памяти. Это позволяет избежать повторного чтения файлов с диска и уменьшить время отклика сервиса.

## Архитектура

### Компоненты

1. **Cache** - основной компонент кэширования
   - Использует `sync.Map` для потокобезопасного хранения данных
   - Поддерживает TTL (Time To Live) для каждого элемента
   - Автоматическая очистка устаревших элементов

2. **Метрики** - система мониторинга кэша
   - Отслеживание попаданий/промахов
   - Мониторинг размера кэша
   - Подсчет количества элементов

3. **Интеграция с генератором DOCX**
   - Автоматическое кэширование шаблонов при первом использовании
   - Использование временных файлов для работы с кэшированными данными

## Конфигурация

### Переменные окружения

- `DOCX_TEMPLATE_CACHE_TTL` - время жизни кэшированных шаблонов
  - Формат: длительность в Go (например, "5m", "1h")
  - По умолчанию: 5 минут
  - Рекомендуемые значения: от 1 минуты до 1 часа

### Пример конфигурации

```yaml
env:
  - name: DOCX_TEMPLATE_CACHE_TTL
    value: "10m"
```

## Метрики

### Prometheus метрики

1. `template_cache_hits_total`
   - Тип: Counter
   - Метки: template
   - Описание: Количество успешных обращений к кэшу

2. `template_cache_misses_total`
   - Тип: Counter
   - Метки: template
   - Описание: Количество промахов кэша

3. `template_cache_size_bytes`
   - Тип: Gauge
   - Метки: template
   - Описание: Размер кэшированных данных в байтах

4. `template_cache_items_total`
   - Тип: Gauge
   - Описание: Общее количество элементов в кэше

### Grafana Dashboard

Рекомендуемые панели для мониторинга:

1. Hit Rate
   ```promql
   sum(rate(template_cache_hits_total[5m])) / 
   (sum(rate(template_cache_hits_total[5m])) + sum(rate(template_cache_misses_total[5m])))
   ```

2. Cache Size
   ```promql
   sum(template_cache_size_bytes) by (template)
   ```

3. Items Count
   ```promql
   template_cache_items_total
   ```

## Примеры использования

### Базовое использование

При стандартной конфигурации кэширование работает автоматически:

```go
generator := docxgen.NewGenerator("scripts/generate_docx.py")

// Первый запрос - шаблон будет загружен с диска и закэширован
err := generator.Generate(ctx, "templates/contract.docx", "data.json", "output1.pdf")

// Второй запрос - шаблон будет взят из кэша
err = generator.Generate(ctx, "templates/contract.docx", "data2.json", "output2.pdf")
```

### Конфигурация через Docker Compose

```yaml
version: '3.8'
services:
  pdf-service:
    environment:
      - DOCX_TEMPLATE_CACHE_TTL=10m
      - LOG_LEVEL=info
```

### Конфигурация через Kubernetes ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: pdf-service-config
data:
  DOCX_TEMPLATE_CACHE_TTL: "10m"
```

### Мониторинг через curl

Проверка метрик кэша:
```bash
curl http://localhost:8080/metrics | grep template_cache
```

Пример вывода:
```
template_cache_hits_total{template="contract.docx"} 45
template_cache_misses_total{template="contract.docx"} 5
template_cache_size_bytes{template="contract.docx"} 52428
template_cache_items_total 1
```

## Примеры метрик и алертов

### Grafana Dashboard

Создайте новый дашборд с следующими панелями:

1. **Cache Hit Rate (Panel 1)**
   ```promql
   # Процент попаданий в кэш за последний час
   sum(increase(template_cache_hits_total[1h])) / 
   (sum(increase(template_cache_hits_total[1h])) + sum(increase(template_cache_misses_total[1h]))) * 100
   ```

2. **Cache Size Trend (Panel 2)**
   ```promql
   # Тренд размера кэша по шаблонам
   sum(template_cache_size_bytes) by (template)
   ```

3. **Cache Operations (Panel 3)**
   ```promql
   # Операции кэша в минуту
   rate(template_cache_hits_total[5m])
   rate(template_cache_misses_total[5m])
   ```

4. **Memory Impact (Panel 4)**
   ```promql
   # Процент памяти, используемой кэшем
   sum(template_cache_size_bytes) / 
   sum(container_memory_working_set_bytes{container="pdf-service"}) * 100
   ```

### Дополнительные алерты

1. **Высокая частота промахов кэша**
   ```yaml
   - alert: PDFServiceHighCacheMissRate
     expr: |
       sum(rate(template_cache_misses_total[5m])) /
       (sum(rate(template_cache_hits_total[5m])) + sum(rate(template_cache_misses_total[5m]))) > 0.4
     for: 10m
     labels:
       severity: warning
     annotations:
       summary: "Высокая частота промахов кэша"
       description: "Более 40% запросов не находят шаблон в кэше"
   ```

2. **Нестабильный размер кэша**
   ```yaml
   - alert: PDFServiceUnstableCacheSize
     expr: |
       abs(
         (sum(template_cache_size_bytes) - 
         sum(template_cache_size_bytes offset 5m)) /
         sum(template_cache_size_bytes offset 5m)
       ) > 0.3
     for: 15m
     labels:
       severity: warning
     annotations:
       summary: "Нестабильный размер кэша"
       description: "Размер кэша изменяется более чем на 30% за 5 минут"
   ```

## Рекомендации по настройке

### Оптимальные значения TTL

1. **Для частых обновлений шаблонов**
   ```yaml
   DOCX_TEMPLATE_CACHE_TTL: "5m"  # Короткий TTL для быстрого обновления
   ```

2. **Для стабильных шаблонов**
   ```yaml
   DOCX_TEMPLATE_CACHE_TTL: "1h"  # Длинный TTL для лучшей производительности
   ```

3. **Для балансировки**
   ```yaml
   DOCX_TEMPLATE_CACHE_TTL: "15m"  # Средний TTL для баланса
   ```

### Настройка ресурсов

Рекомендуемые настройки ресурсов с учетом кэша:

```yaml
resources:
  requests:
    memory: "512Mi"  # Увеличено для кэша
    cpu: "300m"
  limits:
    memory: "1Gi"    # Увеличено для кэша
    cpu: "600m"
```

## Тестирование

Система кэширования покрыта юнит-тестами:

1. Базовые операции
   - Добавление и получение значений
   - Удаление значений
   - Проверка TTL

2. Параллельный доступ
   - Тестирование конкурентных операций
   - Проверка потокобезопасности

3. Очистка кэша
   - Тестирование автоматической очистки
   - Проверка корректности удаления устаревших элементов 