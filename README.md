# PDF Service Go

Сервис для генерации PDF документов на основе DOCX шаблонов.

## Возможности

- Генерация DOCX документов из шаблонов с использованием Python docxtpl
- Конвертация DOCX в PDF с помощью Gotenberg
- Поддержка циклов и условий в шаблонах
- Работа в Docker контейнерах
- Кэширование шаблонов DOCX для оптимизации производительности
- Метрики и мониторинг через Prometheus

## Требования

- Docker
- Docker Compose

## Установка и запуск

1. Клонируйте репозиторий:
```bash
git clone https://github.com/Gimmyhat/pdf-service-go.git
cd pdf-service-go
```

2. Переключитесь на нужную ветку:
   - `main` - стабильная версия
   - `dev` - ветка разработки
```bash
git checkout dev  # для версии в разработке
```

3. Создайте шаблон DOCX в директории `internal/domain/pdf/templates/template.docx`

4. Запустите сервисы:
```bash
docker-compose up --build
```

## Использование

Отправьте POST запрос на `/api/v1/docx` с JSON данными:

```bash
curl -X POST --data-binary @test.json \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/docx \
  -o result.pdf
```

Пример JSON данных (`test.json`):
```json
{
    "operation": "CREATE",
    "id": "ЕФГИ-701-25",
    "email": "test@example.com",
    "phone": "1234567890",
    "applicantType": "INDIVIDUAL",
    "individualInfo": {
        "esia": "1001670968",
        "name": "Иванов Иван Иванович",
        "addressDocument": null
    },
    "purposeOfGeoInfoAccess": "Учебные цели",
    "purposeOfGeoInfoAccessDictionary": {
        "value": "Учебные цели"
    },
    "registryItems": [
        {
            "id": 1243002,
            "name": "Тестовый документ",
            "informationDate": null,
            "invNumber": "2871",
            "note": null
        }
    ],
    "creationDate": "2025-01-29T10:08:39.725+03:00",
    "geoInfoStorageOrganization": {
        "code": "2",
        "value": "ФГБУ \"Организация\""
    }
}
```

## Структура проекта

- `cmd/api` - точка входа приложения
- `internal/api` - HTTP сервер и обработчики
- `internal/domain/pdf` - бизнес-логика и модели
- `internal/pkg/gotenberg` - клиент для работы с Gotenberg
- `scripts` - Python скрипты для работы с DOCX
- `docker` - файлы для сборки Docker образов

## Разработка

Проект использует GitFlow для управления версиями:

- `main` - основная ветка, содержит стабильную версию
- `dev` - ветка разработки, содержит последние изменения
- Для новых функций создавайте ветки `feature/*` от `dev`
- Для исправления ошибок создавайте ветки `bugfix/*` от `dev`
- Для срочных исправлений в production создавайте ветки `hotfix/*` от `main`

### Процесс разработки

1. Создайте новую ветку от `dev`:
```bash
git checkout dev
git pull
git checkout -b feature/my-feature  # или bugfix/my-fix
```

2. Внесите изменения и закоммитьте их:
```bash
git add .
git commit -m "Описание изменений"
```

3. Отправьте изменения в репозиторий:
```bash
git push -u origin feature/my-feature
```

4. Создайте Pull Request в ветку `dev`

## Конфигурация

Сервис настраивается через переменные окружения:

### Основные настройки
- `PORT` - порт для HTTP сервера (по умолчанию: 8080)
- `LOG_LEVEL` - уровень логирования (по умолчанию: info)

### Настройки кэширования
- `DOCX_TEMPLATE_CACHE_TTL` - время жизни кэшированных шаблонов (по умолчанию: 5m)

### Circuit Breaker
- `DOCX_CIRCUIT_BREAKER_FAILURE_THRESHOLD` - порог ошибок (по умолчанию: 3)
- `DOCX_CIRCUIT_BREAKER_RESET_TIMEOUT` - таймаут сброса (по умолчанию: 5s)
- `DOCX_CIRCUIT_BREAKER_HALF_OPEN_MAX_CALLS` - максимум вызовов в полуоткрытом состоянии (по умолчанию: 2)
- `DOCX_CIRCUIT_BREAKER_SUCCESS_THRESHOLD` - порог успешных вызовов (по умолчанию: 2)

## Метрики

Сервис предоставляет следующие метрики Prometheus:

### Метрики кэширования
- `template_cache_hits_total` - количество попаданий в кэш шаблонов
- `template_cache_misses_total` - количество промахов кэша шаблонов
- `template_cache_size_bytes` - размер кэшированных шаблонов
- `template_cache_items_total` - общее количество элементов в кэше

### Метрики генерации документов
- `docx_generation_duration_seconds` - время генерации DOCX
- `docx_generation_errors_total` - количество ошибок генерации
- `docx_generation_total` - общее количество попыток генерации
- `docx_file_size_bytes` - размер сгенерированных файлов
