# Команды для работы с проектом

## Разработка

### Базовые команды
```bash
# Сборка проекта
make build

# Запуск тестов
make test

# Проверка кода линтером
make lint

# Обновление зависимостей
make tidy

# Очистка временных файлов
make clean
```

### Локальная разработка
```bash
# Запуск сервиса локально
make dev

# Сборка и запуск всех сервисов через Docker Compose
make run-local

# Сборка для локальной разработки
make build-local

# Деплой локально через docker-compose
make deploy-local
```

## Docker

### Работа с образами
```bash
# Сборка Docker образа
make docker-build VERSION=<версия>

# Отправка образа в Docker Hub
make docker-push VERSION=<версия>

# Создание нового образа с автоматической версией (YY.MM.DD.HHMM)
make new-version

# Проверка текущей версии
make get-version
```

## Kubernetes

### Команды деплоя
```bash
# Универсальная команда деплоя
make deploy ENV=test     # деплой в тестовый кластер
make deploy ENV=prod     # деплой в продакшен кластер

# Деплой с конкретной версией
make deploy ENV=test VERSION=<версия>
make deploy ENV=prod VERSION=<версия>

# Деплой хранилища (PostgreSQL)
make deploy-storage ENV=test
make deploy-storage ENV=prod
```

Команда `deploy` автоматически:
1. Проверяет и создает ConfigMap `nas-pdf-service-postgres-config` с настройками PostgreSQL
2. Проверяет и создает ConfigMap `nas-pdf-service-templates` с DOCX шаблоном
3. Разворачивает PostgreSQL и ждет его готовности
4. Деплоит основной сервис с указанной версией

### Мониторинг и проверка статуса
```bash
# Проверка статуса
make status ENV=test
make status ENV=prod

# Просмотр логов сервиса
make logs ENV=test
make logs ENV=prod

# Проверка хранилища
make check-storage ENV=test
make check-storage ENV=prod

# Проверка окружений
make check-test    # проверка тестового окружения
make check-prod    # проверка продакшн окружения

# Проверка компонентов мониторинга
make check-grafana
make check-prometheus
make check-jaeger
```

### Проброс портов для мониторинга
```bash
# Grafana (http://localhost:3000)
make port-forward-grafana

# Prometheus (http://localhost:9090)
make port-forward-prometheus

# Jaeger UI (http://localhost:16686)
make port-forward-jaeger
```

## Переменные окружения

Основные переменные, которые можно переопределить при запуске команд:

- `VERSION` - версия образа (по умолчанию latest)
- `ENV` - окружение (test/prod, по умолчанию test)
- `NAMESPACE` - пространство имен в Kubernetes (по умолчанию print-serv)

Пример использования:
```bash
make deploy VERSION=1.2.3 ENV=test
```

## Статистика

### Управление статистикой
```bash
# Очистка статистики в тестовом окружении
make clear-stats ENV=test

# Очистка статистики в продакшен окружении (требует подтверждения)
make clear-stats ENV=prod
```

Команда `clear-stats`:
- Очищает все таблицы статистики (request_logs, docx_logs, gotenberg_logs, pdf_logs)
- Требует указания окружения через параметр ENV
- Для продакшен окружения запрашивает подтверждение
- Использует существующую проверку окружения через `check-env`
- Выводит информативные сообщения о процессе очистки

### Периоды статистики
В веб-интерфейсе доступны следующие периоды для просмотра статистики:
- За 15 минут
- За 1 час
- За 5 часов
- За 24 часа
- За неделю
- За месяц
- За все время