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

## Версионирование и деплой

### Управление версиями
```bash
# Создание нового образа с автоматической версией (YY.MM.DD.HHMM)
# Собирает и отправляет образ в Docker Hub для использования как в Kubernetes, так и локально
make new-version

# Проверка текущей версии
make get-version
```

### Деплой
```bash
# Деплой в локальное окружение (использует образ из Docker Hub)
make deploy-local

# Деплой в тестовый кластер
make deploy ENV=test

# Деплой в продакшен кластер
make deploy ENV=prod

# Деплой с конкретной версией
make deploy ENV=test VERSION=<версия>
make deploy ENV=prod VERSION=<версия>

# Обновление шаблона в Kubernetes
make update-template
# Для обновления шаблона в другом окружении
make update-template ENV=prod
```

### Деплой хранилища
```bash
# Деплой хранилища в Kubernetes
make deploy-storage ENV=test
make deploy-storage ENV=prod
```

## Docker

### Работа с образами
```bash
# Сборка Docker образа
make docker-build VERSION=<версия>

# Отправка образа в Docker Hub
make docker-push VERSION=<версия>

# Сборка для локальной разработки
make build-local
```

## Локальная разработка

### Запуск сервисов
```bash
# Запуск сервиса локально (без Docker)
make dev

# Запуск всех сервисов через Docker Compose
make run-local
```

## Kubernetes

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

- `VERSION` - версия образа (по умолчанию берется из current_version.txt или latest)
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

### Периоды статистики
В веб-интерфейсе доступны следующие периоды для просмотра статистики:
- За 15 минут
- За 1 час
- За 5 часов
- За 24 часа
- За неделю
- За месяц
- За все время