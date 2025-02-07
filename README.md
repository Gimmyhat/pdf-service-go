# PDF Service Go

Сервис для генерации PDF документов на основе DOCX шаблонов.

## Возможности

- Генерация DOCX документов из шаблонов с использованием Python docxtpl
- Конвертация DOCX в PDF с помощью Gotenberg
- Поддержка циклов и условий в шаблонах
- Работа в Docker контейнерах

## Требования

- Docker
- Docker Compose

## Установка и запуск

1. Клонируйте репозиторий:
```bash
git clone https://github.com/Gimmyhat/pdf-service-go.git
cd pdf-service-go
```

2. Создайте шаблон DOCX в директории `internal/domain/pdf/templates/template.docx`

3. Запустите сервисы:
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
