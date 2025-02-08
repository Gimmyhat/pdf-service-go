# Circuit Breaker в PDF Service

## Обзор
Circuit Breaker реализован для защиты сервиса от каскадных отказов при проблемах с Gotenberg. Он автоматически отслеживает состояние Gotenberg и предотвращает перегрузку системы при сбоях.

## Состояния Circuit Breaker

1. **Closed (Закрыт)**
   - Нормальное рабочее состояние
   - Все запросы проходят к Gotenberg
   - Отслеживаются ошибки

2. **Open (Открыт)**
   - Активируется при превышении порога ошибок
   - Запросы блокируются
   - Автоматическое восстановление через заданный timeout

3. **Half-Open (Полуоткрыт)**
   - Тестовое состояние после timeout
   - Пропускается ограниченное число запросов
   - При успехе возврат в Closed, при ошибке в Open

## Конфигурация

Настройка через переменные окружения:

```yaml
CIRCUIT_BREAKER_FAILURE_THRESHOLD: "5"     # Порог ошибок для перехода в Open
CIRCUIT_BREAKER_RESET_TIMEOUT: "10s"       # Время до перехода в Half-Open
CIRCUIT_BREAKER_HALF_OPEN_MAX_CALLS: "2"   # Макс. запросов в Half-Open
CIRCUIT_BREAKER_SUCCESS_THRESHOLD: "2"      # Успешных запросов для возврата в Closed
```

## Мониторинг

### Метрики Prometheus

1. **circuit_breaker_state**
   - Текущее состояние (0: Closed, 1: Open, 2: Half-Open)
   - Labels: name, pod_name, namespace

2. **circuit_breaker_failures_total**
   - Счетчик ошибок
   - Labels: name, pod_name, namespace

3. **circuit_breaker_requests_total**
   - Общее количество запросов
   - Labels: name, pod_name, namespace, status

4. **circuit_breaker_pod_health**
   - Состояние здоровья (0: Unhealthy, 1: Healthy)
   - Labels: name, pod_name, namespace

5. **circuit_breaker_recovery_duration_seconds**
   - Время восстановления из Open в Closed
   - Labels: name, pod_name, namespace

### Алерты

1. **CircuitBreakerOpen**
   - Срабатывает когда Circuit Breaker открыт > 5 минут
   - Severity: warning

2. **HighCircuitBreakerFailureRate**
   - Высокий уровень ошибок за 5 минут
   - Severity: warning

3. **CircuitBreakerUnhealthy**
   - Circuit Breaker в нездоровом состоянии
   - Severity: critical

4. **CircuitBreakerSlowRecovery**
   - Долгое время восстановления
   - Severity: warning

### Health Check

Endpoint `/health` возвращает:
```json
{
  "status": "healthy|unhealthy",
  "timestamp": "2024-02-08T12:34:56Z",
  "details": {
    "circuit_breaker": {
      "status": "healthy|unhealthy",
      "state": "Closed|Open|HalfOpen"
    }
  }
}
```

## Использование в коде

```go
// Создание клиента с Circuit Breaker
client := gotenberg.NewClientWithCircuitBreaker(gotenbergURL)

// Использование
pdfContent, err := client.ConvertDocxToPDF(docxPath)
if err != nil {
    if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
        // Обработка случая, когда Circuit Breaker открыт
    }
    // Обработка других ошибок
}
``` 