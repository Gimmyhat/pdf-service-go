# Команды для работы с PDF Service

## 🐳 Docker команды

### Локальная разработка
```bash
# Запуск всех сервисов
docker-compose up --build

# Запуск в фоновом режиме
docker-compose up -d --build

# Остановка сервисов
docker-compose down

# Просмотр логов
docker-compose logs -f
docker-compose logs -f pdf-service
docker-compose logs -f gotenberg
```

## 🚀 Команды деплоя

### Сборка и публикация образа
```bash
# Сборка с автоматической версией (YYYY.MM.DD.HHMM)
make build

# Сборка с конкретной версией
make build VERSION=1.2.3
```

### Деплой в кластеры
```bash
# Деплой в тестовый кластер
make deploy-test
make deploy-test VERSION=1.2.3

# Деплой в продакшен
make deploy-prod
make deploy-prod VERSION=1.2.3

# Деплой во все кластеры
make deploy-all
make deploy-all VERSION=1.2.3
```

### Проверка статуса
```bash
# Проверка тестового кластера
make check-test

# Проверка продакшена
make check-prod
```

### Обновление шаблона
После любых изменений в файле шаблона `internal/domain/pdf/templates/template.docx` необходимо обновить ConfigMap в кластере. Есть два способа это сделать:

```bash
# Способ 1: Через make команду (может работать нестабильно)
make update-template-test  # для тестового кластера
make update-template-prod  # для продакшена

# Способ 2: Напрямую через PowerShell скрипты (рекомендуемый способ)
powershell -ExecutionPolicy Bypass -File scripts/update-template.ps1     # для тестового кластера
powershell -ExecutionPolicy Bypass -File scripts/update-template-prod.ps1  # для продакшена
```

## 📋 Типовые сценарии

### Полный цикл обновления
```bash
# 1. Сборка и публикация нового образа (версия генерируется автоматически)
make build

# 2. Деплой в тестовый кластер
make deploy-test

# 3. Проверка статуса в тестовом кластере
make check-test

# 4. После проверки на тесте, деплой в продакшен
make deploy-prod

# 5. Проверка статуса в продакшене
make check-prod
```

### Обновление с указанием версии
```bash
# Сборка с конкретной версией
make build VERSION=1.2.3

# Деплой конкретной версии
make deploy-test VERSION=1.2.3
make deploy-prod VERSION=1.2.3
```

## 🔍 Kubernetes команды

### Просмотр состояния
```bash
# Проверка подов
kubectl get pods -n print-serv
kubectl get pods -n print-serv -l app=pdf-service
kubectl get pods -n print-serv -l app=gotenberg

# Проверка сервисов
kubectl get svc -n print-serv

# Проверка деплойментов
kubectl get deploy -n print-serv

# Проверка HPA
kubectl get hpa -n print-serv
```

### Логи и отладка
```bash
# Просмотр логов pdf-service
kubectl logs -n print-serv -l app=pdf-service -f

# Просмотр логов gotenberg
kubectl logs -n print-serv -l app=gotenberg -f

# Описание пода
kubectl describe pod -n print-serv -l app=pdf-service

# Проверка ConfigMap
kubectl get configmap -n print-serv pdf-service-templates -o yaml
```

### Масштабирование
```bash
# Ручное масштабирование
kubectl scale deployment -n print-serv pdf-service --replicas=3
kubectl scale deployment -n print-serv gotenberg --replicas=3
```

### Перезапуск подов
```bash
# Перезапуск pdf-service
kubectl rollout restart deployment -n print-serv pdf-service

# Перезапуск gotenberg
kubectl rollout restart deployment -n print-serv gotenberg
```

## 📝 Тестирование API

### Отправка тестового запроса
```bash
# Генерация PDF из JSON
curl -X POST \
  -H "Content-Type: application/json" \
  --data-binary @test.json \
  http://localhost:8080/generate-pdf \
  -o result.pdf

# Проверка здоровья сервиса
curl http://localhost:8080/health
```

## 📊 Мониторинг

### Метрики Prometheus
```bash
# Просмотр метрик сервиса
curl http://localhost:8080/metrics
```

## ❓ Помощь

### Просмотр доступных make команд
```bash
make help
``` 