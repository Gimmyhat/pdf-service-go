# 🚀 Deployment Guide - PDF Service

## 📋 Обзор

Этот документ описывает как использовать автоматизированную систему развертывания PDF Service с системой отслеживания ошибок.

## ⚙️ Переменные конфигурации

Makefile использует следующие ключевые переменные:

```makefile
APP_NAME = pdf-service-go                    # Имя приложения
DOCKER_REPO = gimmyhat                       # Docker Hub репозиторий  
DOCKER_MIRROR = dh-mirror.gitverse.ru        # Российское зеркало
DOCKER_HUB_IMAGE = gimmyhat/pdf-service-go   # Полное имя в Docker Hub
DOCKER_IMAGE = dh-mirror.gitverse.ru/gimmyhat/pdf-service-go  # Полное имя в зеркале
NAMESPACE = print-serv                       # Kubernetes namespace

# Kubernetes контексты
TEST_CONTEXT = efgi-test                     # Контекст тестового кластера
PROD_CONTEXT = efgi-prod                     # Контекст продакшн кластера
```

**Преимущества централизованных переменных:**
- Одно место для изменения зеркала (`DOCKER_MIRROR`)
- Автоматическое формирование полных имен образов
- Легкая смена репозитория или namespace

## 🔧 Основные команды

### 1. Создание новой версии
```bash
make new-version
```
**Что происходит:**
- Генерируется версия в формате `YY.MM.DD.HHMM`
- Собирается Docker образ
- Пушится в Docker Hub (gimmyhat/pdf-service-go)
- Обновляется тег `latest`
- Сохраняется версия в `current_version.txt`

### 2. Развертывание в тестовой среде
```bash
make deploy ENV=test
```

### 3. Развертывание в продакшне
```bash
make deploy ENV=prod
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

## 🔄 Схема работы с Docker Registry

1. **Локальная разработка** → **Docker Hub** (`gimmyhat/pdf-service-go`)
2. **Docker Hub** → **Российское зеркало** (`dh-mirror.gitverse.ru/gimmyhat/pdf-service-go`) 
3. **Kubernetes кластер** ← **Российское зеркало**

### Почему так?
- Из России нет прямого доступа к Docker Hub в кластерах
- У нас есть push доступ в Docker Hub с локальной машины
- Зеркало автоматически синхронизируется (15-30 минут)

## 🚨 Решение проблем

### Проблема: "ImagePullBackOff" после развертывания
**Причина:** Зеркало еще не синхронизировалось с Docker Hub

**Решение:**
```bash
# Подождать 15-30 минут или принудительно обновить
make force-update ENV=prod
```

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

### Endpoints для проверки:

**Автоматическое определение:**
```bash
make get-service-url ENV=prod  # Покажет актуальный URL
make get-service-url ENV=test
```

**Прямые ссылки:**
- **Тест:** `http://172.27.239.30:31005`  
- **Прод:** `http://172.27.239.2:31005`

> 💡 **Совет:** Используйте `make get-service-url` чтобы не зависеть от конкретных IP адресов

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