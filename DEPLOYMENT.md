# 🚀 Deployment Guide - PDF Service

## 📋 Обзор

Этот документ описывает как использовать автоматизированную систему развертывания PDF Service с системой отслеживания ошибок.

## ⚙️ Профили реестров и переменные

Makefile поддерживает разделение push/pull и профили реестров «как у pim»:

```makefile
# Push (RW) и Pull (RO) реестры
REGISTRY_PUSH ?= registry-irk-rw.devops.rgf.local   # RW Nexus (для push)
REGISTRY_PULL ?= registry.devops.rgf.local          # RO DevOps (для pull)

# Профили: mirror | devops | nexus
REGISTRY_PROFILE ?= devops  # по умолчанию используем pull из registry.devops.rgf.local

# Исходные образы (формируются от REGISTRY_PULL)
IMG_PDF := $(REGISTRY_PULL)/gimmyhat/pdf-service-go
IMG_GOTENBERG := $(REGISTRY_PULL)/gotenberg/gotenberg
IMG_PROMETHEUS := $(REGISTRY_PULL)/prom/prometheus
IMG_GRAFANA := $(REGISTRY_PULL)/grafana/grafana
IMG_JAEGER := $(REGISTRY_PULL)/jaegertracing/all-in-one
IMG_POSTGRES := $(REGISTRY_PULL)/library/postgres
```

**Что это даёт:**

- Лёгкое переключение источника образов одной переменной `REGISTRY_PROFILE`
- Push всегда в RW `registry-irk-rw.devops.rgf.local`, pull в кластере из `registry.devops.rgf.local`
- Единообразно с namespace `pim`, где используется `registry.devops.rgf.local/...:tag`

## 🔧 Основные команды

### 1. Создание новой версии (уникальный тег)

```bash
make new-version
```

**Что происходит:**

- Генерируется уникальный тег `YY.MM.DD.HHMM`
- Собирается Docker образ и пушится в RW `registry-irk-rw.devops.rgf.local`
- Создайте и запушьте такой же тег в Docker Hub (если нужен pull через dh‑mirror)
- Сохраняется версия в `current_version.txt`

### 2. Развертывание в тестовой среде (pull из registry.devops.rgf.local)

```bash
make deploy ENV=test REGISTRY_PROFILE=devops VERSION=<YY.MM.DD.HHMM>
```

### 3. Развертывание в продакшне (как у pim)

```bash
make deploy ENV=prod REGISTRY_PROFILE=devops VERSION=<YY.MM.DD.HHMM>
```

**Требует подтверждения!**

### 4. Проверка статуса

```bash
make status ENV=prod
make logs ENV=prod
```

## 🛠️ Дополнительные команды

### Проверка системы отслеживания ошибок

```bash
make test-error-system ENV=prod
make test-error-system ENV=test
```

**Проверяет:**

- ✅ `/health` endpoint
- ✅ `/errors` веб-интерфейс  
- ✅ `/api/v1/errors/stats` API
- ✅ Генерацию тестовых ошибок

### Принудительное обновление (если зеркало не синхронизировалось)

```bash
make force-update ENV=prod VERSION=25.08.06.1019
make force-update ENV=test  # Использует current_version.txt
```

### Проверка синхронизации зеркала

```bash
make check-mirror ENV=prod
```

### Получение текущей версии

```bash
make get-version
```

### Получение URL сервиса

```bash
make get-service-url ENV=prod
make get-service-url ENV=test
```

### Управление Docker зеркалом

```bash
# Показать файлы использующие зеркало
make show-mirror-usage

# Обновить зеркало во всех файлах (если изменилось)
make update-mirror NEW_MIRROR=new-mirror.example.com
```

## 🔄 Схема работы с Docker Registry (по аналогии с pim)

1. Push: локально/CI → RW `registry-irk-rw.devops.rgf.local/<org>/<image>:<tag>`
2. Pull: кластеры → RO `registry.devops.rgf.local/<org>/<image>:<tag>`

Примечания:

- В большинстве кластеров pull из `registry.devops.rgf.local` не требует `imagePullSecret` (read‑only), так как ноды уже доверяют CA
- Избегаем `latest` — используем только уникальные теги `YY.MM.DD.HHMM`

## 🚨 Решение проблем

### Проблема: "ImagePullBackOff" после развертывания

**Причины и решения:**

- Тег отсутствует в `registry.devops.rgf.local` → Попросить админа реплицировать тег из RW в RO
- TLS доверие нод (редко) → Админ устанавливает CA на ноды (как уже сделано для pim)
- Секрет не нужен для RO, но не помешает для RW операций: `make create-nexus-pull-secret`

### Подготовка GitHub Actions

Добавьте секреты репозитория:

- `NEXUS_USERNAME` = `tech_irk`
- `NEXUS_PASSWORD` = пароль от Nexus

### Проблема: Старая версия в кластере

**Причина:** Кластер кэширует образы

**Решение:**

```bash
# Принудительное обновление с конкретной версией
make force-update ENV=prod VERSION=25.08.06.1019
```

### Проблема: "Access denied" при push

**Причина:** Попытка push в зеркало вместо Docker Hub

**Решение:** Makefile исправлен! Теперь push идет в Docker Hub.

## 📊 Мониторинг после развертывания

### Endpoints для проверки

**Автоматическое определение:**

```bash
make get-service-url ENV=prod  # Покажет актуальный URL
make get-service-url ENV=test
```

> 💡 Используйте `make get-service-url ENV=<test|prod>` чтобы получить актуальный URL сервиса. Не полагайтесь на фиксированные IP.

- `/health` - Статус сервиса
- `/stats` - Общая статистика
- `/errors` - **Новый!** Дашборд анализа ошибок
- `/api/v1/errors` - API детальных ошибок
- `/api/v1/errors/stats` - API статистики ошибок
- `/test-error` - Генерация тестовых ошибок

## 🎯 Типичный рабочий процесс

1. **Разработка изменений**
2. **Тестирование локально:**

   ```bash
   make build
   make dev
   ```

3. **Создание новой версии:**

   ```bash
   make new-version
   ```

4. **Развертывание в тест:**

   ```bash
   make deploy ENV=test
   make test-error-system ENV=test
   ```

5. **Развертывание в продакшн:**

   ```bash
   make deploy ENV=prod
   make test-error-system ENV=prod
   ```

6. **Мониторинг:**

   ```bash
   make status ENV=prod
   make logs ENV=prod
   ```

## ✅ Проверка успешного развертывания

После развертывания проверьте:

1. **Статус подов:** `make status ENV=prod`
2. **Логи запуска:** `make logs ENV=prod`
3. **Endpoints:** `make test-error-system ENV=prod`
4. **Веб-интерфейс:** Откройте `/errors` в браузере

**Успешное развертывание:** Все endpoints отвечают ✅, логи показывают новые маршруты с `errors_api` и `errors_ui`.

## 🔐 Требования

- Docker с авторизацией в Docker Hub
- kubectl с настроенными контекстами `efgi-test` и `efgi-prod`  
- PowerShell (для Windows команд)
- Доступ к кластерам Kubernetes
