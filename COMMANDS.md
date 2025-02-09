# Команды для работы с PDF Service

## 🐳 Docker команды

### Локальная разработка
```bash
# Сборка для локальной разработки (без пуша в registry)
make build-local

# Запуск всех сервисов
docker-compose up

# Запуск в фоновом режиме
docker-compose up -d

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
# Сборка с существующей версией (из current_version.txt) или новой, если файл не существует
make build

# Сборка с новой версией (всегда генерирует новую версию в формате YY.MM.DD.HHMM)
make build-new

# Сборка с конкретной версией
make build VERSION=1.2.3

# Просмотр текущей версии
make get-version
```

### Деплой в кластеры
```bash
# Деплой в тестовый кластер (использует версию из current_version.txt)
make deploy-test

# Деплой в тестовый кластер конкретной версии
make deploy-test VERSION=1.2.3

# Деплой в продакшен (использует версию из current_version.txt)
make deploy-prod

# Деплой в продакшен конкретной версии
make deploy-prod VERSION=1.2.3

# Деплой во все кластеры (использует версию из current_version.txt)
make deploy-all

# Деплой во все кластеры конкретной версии
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
После любых изменений в файле шаблона `internal/domain/pdf/templates/template.docx` необходимо обновить ConfigMap в кластере:

```bash
# Обновление шаблона в тестовом кластере
make update-template-test

# Обновление шаблона в продакшене
make update-template-prod

# Дополнительные опции при использовании скрипта напрямую:
# Обновление без подтверждений (полезно для CI/CD)
powershell -ExecutionPolicy Bypass -File scripts/update-template-unified.ps1 -Environment test -Force

# Обновление без создания резервной копии
powershell -ExecutionPolicy Bypass -File scripts/update-template-unified.ps1 -Environment test -SkipBackup

# Обновление в продакшене с подтверждением
powershell -ExecutionPolicy Bypass -File scripts/update-template-unified.ps1 -Environment prod
```

Особенности работы с шаблонами:
1. При обновлении создается резервная копия в папке `backups/templates`
2. В тестовом окружении обновление происходит автоматически
3. В продакшене запрашивается подтверждение на:
   - Обновление шаблона
   - Перезапуск подов
4. Шаблон автоматически обновляется при деплое (`make deploy-test` или `make deploy-prod`)
5. В ConfigMap сохраняется информация о времени обновления и пользователе

## 📋 Типовые сценарии

### Полный цикл обновления
```bash
# 1. Сборка и публикация нового образа с новой версией
make build-new

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
kubectl get pods -n print-serv -l app=nas-pdf-service
kubectl get pods -n print-serv -l app=nas-gotenberg

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
kubectl logs -n print-serv -l app=nas-pdf-service -f

# Просмотр логов gotenberg
kubectl logs -n print-serv -l app=nas-gotenberg -f

# Описание пода
kubectl describe pod -n print-serv -l app=nas-pdf-service

# Проверка ConfigMap
kubectl get configmap -n print-serv nas-pdf-service-templates -o yaml
```

### Масштабирование
```bash
# Ручное масштабирование
kubectl scale deployment -n print-serv nas-pdf-service --replicas=3
kubectl scale deployment -n print-serv nas-gotenberg --replicas=3
```

### Перезапуск подов
```bash
# Перезапуск pdf-service
kubectl rollout restart deployment -n print-serv nas-pdf-service

# Перезапуск gotenberg
kubectl rollout restart deployment -n print-serv nas-gotenberg
```

## 📝 Тестирование API

### Отправка тестового запроса
```bash
# Генерация PDF из JSON (новый endpoint)
curl -X POST \
  -H "Content-Type: application/json" \
  --data-binary @test.json \
  http://localhost:8080/api/v1/docx \
  -o result.pdf

# Генерация PDF из JSON (старый endpoint, поддерживается для обратной совместимости)
curl -X POST \
  -H "Content-Type: application/json" \
  --data-binary @test.json \
  http://localhost:8080/generate-pdf \
  -o result.pdf

# Проверка здоровья сервиса
curl http://localhost:8080/health
```

## 📊 Мониторинг

### Развертывание мониторинга
```bash
# Развертывание всего стека мониторинга (Prometheus + Grafana)
make deploy-monitoring

# Развертывание отдельных компонентов
make deploy-prometheus  # только Prometheus
make deploy-grafana    # только Grafana
```

### Проверка статуса
```bash
# Проверка статуса Prometheus
make check-prometheus

# Проверка статуса Grafana
make check-grafana
```

### Доступ к UI
```bash
# Доступ к Grafana UI (http://localhost:3000)
make port-forward-grafana

# Доступ к Prometheus UI (http://localhost:9090)
make port-forward-prometheus

# Примечание: port-forward нужно запускать в отдельных терминалах
# Для остановки port-forward используйте Ctrl+C
```

### Метрики Prometheus
```bash
# Просмотр метрик сервиса напрямую
curl http://localhost:8080/metrics

# Основные метрики:
# - pdf_generation_total: количество сгенерированных PDF (по статусу)
# - pdf_generation_duration_seconds: длительность генерации PDF
# - pdf_file_size_bytes: размер сгенерированных файлов
# - gotenberg_requests_total: количество запросов к Gotenberg
# - gotenberg_request_duration_seconds: длительность запросов к Gotenberg
# - http_requests_total: общее количество HTTP запросов
# - http_request_duration_seconds: длительность HTTP запросов
```

### Grafana
```bash
# Доступ к UI
make port-forward-grafana
# URL: http://localhost:3000
# Логин: admin
# Пароль: admin (рекомендуется сменить при первом входе)
```

#### Основные дашборды

1. **NAS PDF Service Dashboard**
   - Количество успешных/неуспешных генераций PDF
   - Длительность генерации PDF (95-й и 50-й перцентили)
   - Размер генерируемых файлов
   - Длительность запросов к Gotenberg

#### Алерты
Grafana настроена для мониторинга следующих ситуаций:
- Высокий процент ошибок (>5% за 5 минут)
- Длительное время генерации PDF (>30 секунд)
- Большой размер генерируемых файлов (>10MB)
- Проблемы с Gotenberg сервисом

### Типовые сценарии мониторинга

#### Первичная настройка мониторинга
```bash
# 1. Развертывание стека мониторинга
make deploy-monitoring

# 2. Проверка статуса компонентов
make check-prometheus
make check-grafana

# 3. Доступ к UI (в разных терминалах)
make port-forward-prometheus  # Терминал 1
make port-forward-grafana    # Терминал 2
```

#### Обновление мониторинга
```bash
# Обновление всего стека
make deploy-monitoring

# Обновление отдельных компонентов
make deploy-prometheus
make deploy-grafana
```

## ❓ Помощь

### Просмотр доступных make команд
```bash
make help
``` 