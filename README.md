# PDF Service Go

Высокопроизводительный сервис для генерации PDF документов из DOCX шаблонов с детальным отслеживанием ошибок и полной автоматизацией развертывания.

## ⭐ Ключевые особенности

### Основная функциональность
- **PDF Generation**: Конвертация DOCX в PDF с помощью Gotenberg
- **Template Processing**: Поддержка DOCX шаблонов с использованием Python (docxtpl)
- **REST API**: Полнофункциональный API с валидацией и обработкой ошибок

### 🆕 Система отслеживания ошибок и анализ запросов
- **Веб-интерфейс**: Единый дашборд `/dashboard` (Ошибки, Статистика, Архив)
- **Детальная аналитика**: Графики, фильтрация, поиск по ошибкам
- **API для ошибок**: `/api/v1/errors` для программного доступа к данным
 - **API для запросов**: `/api/v1/requests/*` (детали, тело запроса, ошибки, аналитика, архив)
- **Централизованное логирование**: Автоматический сбор ошибок из всех компонентов

### 🚀 Полная автоматизация
- **One-command deployment**: `make new-version && make deploy ENV=prod`
- **Автоматическое определение URL**: `make get-service-url ENV=prod`
- **Централизованная конфигурация**: Легкая смена Docker зеркал и настроек
- **Комплексное тестирование**: `make test-error-system ENV=prod`

### 📊 Мониторинг и наблюдаемость
- **Prometheus + Grafana**: Метрики производительности и бизнес-показатели
- **OpenTelemetry + Jaeger**: Distributed tracing для анализа производительности
- **Structured Logging**: JSON логи с контекстными полями
- **Health Checks**: Kubernetes probes для мониторинга состояния
 - **Архив**: хранение последних N запросов с артефактами (JSON и PDF) и очистка

### 🛡️ Надежность
- **Circuit Breaker**: Защита от каскадных сбоев
- **Retry механизмы**: Устойчивость к временным сбоям
- **Kubernetes**: Автоматическое восстановление и масштабирование
- **PostgreSQL**: Надежное хранение статистики и логов ошибок

## Требования

- Go 1.22+
- Docker
- Python 3.9+ (PyPy)
- Kubernetes (для production)
- Helm 3.x (для production)

## 🚀 Быстрый старт

### Локальная разработка

1. **Клонируйте репозиторий:**
```bash
git clone https://github.com/gimmyhat/pdf-service-go.git
cd pdf-service-go
```

2. **Создайте и запустите сервисы:**
```bash
make new-version      # Создает новую версию образа
make deploy-local     # Запускает все сервисы в Docker Compose
```

3. **Проверьте работу:**
```bash
# Генерация PDF
curl -X POST -H "Content-Type: application/json" \
  --data-binary "@test_data.json" \
  http://localhost:8080/api/v1/docx -o result.pdf

# Генерация на проде
curl -X POST -H "Content-Type: application/json" --data-binary "@test_data.json" http://172.27.239.2:31005/api/v1/docx -o result.pdf

# Проверка системы отслеживания ошибок
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/errors/stats
```

4. **Откройте веб-интерфейсы:**
- **Система ошибок**: http://localhost:8080/errors
- **Grafana**: http://localhost:3000
- **Prometheus**: http://localhost:9090  
- **Jaeger**: http://localhost:16686

### Развертывание в Kubernetes

1. **Создайте новую версию:**
```bash
make new-version
```

2. **Разверните в тестовом окружении:**
```bash
make deploy ENV=test
make test-error-system ENV=test  # Проверка всех endpoints
```

3. **Разверните в продакшне:**
```bash
make deploy ENV=prod
make get-service-url ENV=prod    # Получить URL сервиса
```

## 🔧 Основные команды

### Разработка
```bash
make build          # Сборка Go приложения
make test           # Запуск тестов
make lint           # Статический анализ кода
make dev            # Локальный запуск (без Docker)
make run-local      # Запуск в Docker Compose
```

### Управление версиями и развертывание
```bash
make new-version              # Создание новой версии (YY.MM.DD.HHMM)
make get-version             # Показать текущую версию
make deploy ENV=test         # Развертывание в тестовом окружении
make deploy ENV=prod         # Развертывание в продакшне (с подтверждением)
make force-update ENV=prod   # Принудительное обновление при проблемах
```

### 🆕 Система отслеживания ошибок
```bash
make test-error-system ENV=prod    # Комплексная проверка системы ошибок
make get-service-url ENV=prod      # Получить URL сервиса автоматически
```

### Управление конфигурацией
```bash
make show-mirror-usage                        # Показать файлы использующие Docker зеркало
make update-mirror NEW_MIRROR=new.mirror.ru  # Сменить зеркало во всех файлах
make check-mirror ENV=prod                    # Проверить синхронизацию зеркала
```

### Мониторинг и диагностика
```bash
make status ENV=prod           # Статус подов и сервисов
make logs ENV=prod             # Логи основного сервиса
make check-grafana             # Проверка Grafana
make check-prometheus          # Проверка Prometheus
make check-jaeger              # Проверка Jaeger
make clear-stats ENV=test      # Очистка статистики (с подтверждением для prod)
```

## 📊 Мониторинг и интерфейсы

### Локальная разработка
- **Система ошибок**: http://localhost:8080/errors
- **API документация**: http://localhost:8080/health
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Jaeger UI**: http://localhost:16686

### Продакшн (получите актуальные URL командой `make get-service-url ENV=prod`)
- **Тестовый кластер**: http://172.27.239.30:31005
- **Продакшн кластер**: http://172.27.239.2:31005

### 🆕 Полезные endpoints
- Веб: `/dashboard` (Обзор/Статистика/Ошибки/Архив)
- Ошибки: `GET /api/v1/errors`, `GET /api/v1/errors/stats`, `GET /api/v1/errors/:id`
- Запросы: `GET /api/v1/requests/recent`, `POST /api/v1/requests/cleanup`, `GET /api/v1/requests/:id`, `GET /api/v1/requests/:id/body`
- Тестовые: `GET /test-error`, `GET /test-timeout`

## 📚 Документация

### Основная документация
- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Подробное руководство по развертыванию и автоматизации
- **[COMMANDS.md](COMMANDS.md)** - Справочник по всем командам Makefile
- **[ROADMAP.md](ROADMAP.md)** - План развития проекта

### Memory Bank (для разработчиков)
- **[Project Brief](memory-bank/projectbrief.md)** - Цели и требования проекта
- **[Product Context](memory-bank/productContext.md)** - Проблемы и решения
- **[System Patterns](memory-bank/systemPatterns.md)** - Архитектурные паттерны
- **[Tech Context](memory-bank/techContext.md)** - Технологический стек
- **[Active Context](memory-bank/activeContext.md)** - Текущий фокус работы
- **[Progress](memory-bank/progress.md)** - Статус готовности функций

## 🎯 Статус проекта

**Готовность к продакшну: 95%** ✅

- ✅ **Core functionality**: PDF генерация работает стабильно
- ✅ **Error tracking**: Полная система отслеживания ошибок реализована
- ✅ **Automation**: Полная автоматизация развертывания
- ✅ **Monitoring**: Prometheus + Grafana + Jaeger настроены
- 🟡 **Security**: Базовая безопасность (требуется auth)
- 🟡 **Documentation**: Comprehensive (требуется API docs)

## 🚀 Быстрые ссылки

**Для разработчиков:**
- Начать разработку: `make new-version && make deploy-local`
- Запустить тесты: `make test && make lint`

**Для DevOps:**  
- Развернуть в тест: `make deploy ENV=test && make test-error-system ENV=test`
- Развернуть в прод: `make deploy ENV=prod && make get-service-url ENV=prod`

**Для мониторинга:**
- Система ошибок: `/errors` endpoint
- Метрики: Grafana dashboard
- Трейсы: Jaeger UI

## 📄 Лицензия

MIT License
