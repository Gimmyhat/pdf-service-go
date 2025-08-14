.PHONY: build test lint tidy clean \
        docker-build docker-push docker-push-latest dockerhub-push dockerhub-push-latest new-version new-version-hub build-local \
        check-env get-version status logs \
        check-storage check-test check-prod check-grafana check-prometheus check-jaeger test-error-system \
        deploy deploy-local deploy-storage force-update check-mirror get-service-url \
        dev run-local port-forward-grafana port-forward-prometheus port-forward-jaeger \
        clear-stats \
        helm-repos helm-deps helm-template helm-lint helm-deploy helm-uninstall helm-status helm-history helm-rollback \
        update-template update-mirror show-mirror-usage \
        migrate-db

# Основные переменные
APP_NAME = pdf-service-go
DOCKER_REPO = gimmyhat
# Реестры для push/pull (поддержка разделения RW/RO)
REGISTRY_PUSH ?= registry-irk-rw.devops.rgf.local
REGISTRY_PULL ?= registry.devops.rgf.local

# Профиль источника образов (mirror | devops | nexus)
# Используется для быстрого переключения всех образов без правки YAML
REGISTRY_PROFILE ?= devops
ifeq ($(REGISTRY_PROFILE),mirror)
  REGISTRY_PULL := dh-mirror.gitverse.ru
else ifeq ($(REGISTRY_PROFILE),devops)
  REGISTRY_PULL := registry.devops.rgf.local
else ifeq ($(REGISTRY_PROFILE),nexus)
  REGISTRY_PULL := registry-irk-rw.devops.rgf.local
endif
# Обратная совместимость (устарело): DOCKER_MIRROR/DOCKER_IMAGE
DOCKER_MIRROR = $(REGISTRY_PULL)
DOCKER_HUB_IMAGE = $(DOCKER_REPO)/$(APP_NAME)
DOCKER_IMAGE = $(DOCKER_MIRROR)/$(DOCKER_REPO)/$(APP_NAME)
# Явные имена образов
DOCKER_IMAGE_PUSH = $(REGISTRY_PUSH)/$(DOCKER_REPO)/$(APP_NAME)
DOCKER_IMAGE_PULL = $(REGISTRY_PULL)/$(DOCKER_REPO)/$(APP_NAME)

# Образы сервисов (формируются из REGISTRY_PULL)
IMG_PDF := $(REGISTRY_PULL)/$(DOCKER_REPO)/$(APP_NAME)
IMG_GOTENBERG := $(REGISTRY_PULL)/gotenberg/gotenberg
IMG_PROMETHEUS := $(REGISTRY_PULL)/prom/prometheus
IMG_GRAFANA := $(REGISTRY_PULL)/grafana/grafana
IMG_JAEGER := $(REGISTRY_PULL)/jaegertracing/all-in-one
IMG_POSTGRES := $(REGISTRY_PULL)/library/postgres

# Теги образов (закреплены для воспроизводимости)
TAG_GOTENBERG ?= 7.10
TAG_PROMETHEUS ?= v2.51.2
TAG_GRAFANA ?= 11.1.3
TAG_JAEGER ?= 1.54
TAG_POSTGRES ?= 15-alpine
NAMESPACE = print-serv

# Автоматическая генерация версии в формате YY.MM.DD.HHMM
NEW_VERSION := $(shell powershell -Command "Get-Date -Format 'yy.MM.dd.HHmm'")
VERSION ?= latest

# Kubernetes контексты
TEST_CONTEXT = efgi-test
PROD_CONTEXT = efgi-prod
ENV ?= test
CONTEXT = $(if $(filter prod,$(ENV)),$(PROD_CONTEXT),$(TEST_CONTEXT))

# ============================================================================
# Базовые команды для разработки
# ============================================================================

build:
	go build -o main cmd/api/main.go

test:
	@echo "Running tests..."
	go test -v ./...

lint:
	golangci-lint run

tidy:
	go mod tidy

clean:
	powershell -Command "if (Test-Path main) { Remove-Item main }"
	powershell -Command "Remove-Item -ErrorAction SilentlyContinue *.pdf"
	powershell -Command "Remove-Item -ErrorAction SilentlyContinue *.docx"

# ============================================================================
# Docker команды
# ============================================================================

docker-build:
	docker build -t $(DOCKER_IMAGE_PUSH):$(VERSION) .

docker-push:
	@echo "Pushing to Nexus ($(DOCKER_IMAGE_PUSH):$(VERSION))..."
	docker push $(DOCKER_IMAGE_PUSH):$(VERSION)
	@echo "Successfully pushed to Nexus registry $(DOCKER_MIRROR)"

build-local:
	@echo "Building Docker image for local development..."
	docker-compose build
	@echo "Successfully built local development image"

# Создание нового образа с новой версией
new-version:
	@echo "Building new version: $(NEW_VERSION)"
	$(MAKE) docker-build VERSION=$(NEW_VERSION)
	$(MAKE) docker-push VERSION=$(NEW_VERSION)
	$(MAKE) docker-push-latest VERSION=$(NEW_VERSION)
	@powershell -Command "Set-Content -Path 'current_version.txt' -Value '$(NEW_VERSION)' -NoNewline"
	@echo "New version $(NEW_VERSION) has been built and pushed"
	@echo "Both versioned and latest tags have been updated"

# То же самое, но с дополнительной публикацией в Docker Hub
new-version-hub:
	@echo "Building new version (with Docker Hub): $(NEW_VERSION)"
	$(MAKE) docker-build VERSION=$(NEW_VERSION)
	$(MAKE) docker-push VERSION=$(NEW_VERSION)
	$(MAKE) docker-push-latest VERSION=$(NEW_VERSION)
	$(MAKE) dockerhub-push VERSION=$(NEW_VERSION)
	$(MAKE) dockerhub-push-latest VERSION=$(NEW_VERSION)
	@powershell -Command "Set-Content -Path 'current_version.txt' -Value '$(NEW_VERSION)' -NoNewline"
	@echo "New version $(NEW_VERSION) has been built and pushed to Nexus and Docker Hub"

# Push образа как latest (для обновления зеркала)
docker-push-latest:
	@echo "Tagging and pushing as latest to Nexus..."
	docker tag $(DOCKER_IMAGE_PUSH):$(VERSION) $(DOCKER_IMAGE_PUSH):latest
	docker push $(DOCKER_IMAGE_PUSH):latest
	@echo "Latest tag updated in Nexus"

# Push образа в Docker Hub (опционально, по уникальным тегам)
dockerhub-push:
	@echo "Tagging and pushing to Docker Hub ($(DOCKER_HUB_IMAGE):$(VERSION))..."
	docker tag $(DOCKER_IMAGE_PUSH):$(VERSION) $(DOCKER_HUB_IMAGE):$(VERSION)
	docker push $(DOCKER_HUB_IMAGE):$(VERSION)
	@echo "Successfully pushed to Docker Hub"

dockerhub-push-latest:
	@echo "Tagging and pushing latest to Docker Hub..."
	docker tag $(DOCKER_IMAGE_PUSH):$(VERSION) $(DOCKER_HUB_IMAGE):latest
	docker push $(DOCKER_HUB_IMAGE):latest
	@echo "Latest tag updated in Docker Hub"

# Получение текущей версии
get-version:
	@powershell -Command "if (Test-Path current_version.txt) { Write-Host \"Current version: $$(Get-Content current_version.txt)\" } else { Write-Host \"Current version: latest (no current_version.txt found)\" }"

# ============================================================================
# Kubernetes команды
# ============================================================================

# Проверка переменных окружения
check-env:
	@powershell -command "if (-not '$(ENV)') { Write-Error 'Error: ENV is not set. Use ENV=test or ENV=prod'; exit 1 }"
	@powershell -command "if ('$(ENV)' -ne 'test' -and '$(ENV)' -ne 'prod') { Write-Error 'Error: ENV must be either ''test'' or ''prod'''; exit 1 }"
	@echo "Environment check passed: ENV=$(ENV)"

# Деплой хранилища
deploy-storage: check-env
	@echo "Deploying storage for $(ENV) environment..."
	kubectl config use-context $(CONTEXT)
	kubectl apply -f k8s/nas-pdf-service-storage.yaml -n $(NAMESPACE)
	kubectl apply -f k8s/nas-pdf-service-postgres-deployment.yaml -n $(NAMESPACE)
	@echo "Storage deployment completed"

# Проверка хранилища
check-storage: check-env
	@echo "Checking storage in $(ENV) environment..."
	kubectl get pvc nas-pdf-service-stats-pvc -n $(NAMESPACE)
	kubectl get pods -n $(NAMESPACE) -l app=nas-pdf-service -o jsonpath='{.items[*].spec.volumes[?(@.persistentVolumeClaim.claimName=="nas-pdf-service-stats-pvc")].persistentVolumeClaim.claimName}'

# Проверка тестового окружения
check-test: check-storage
	@echo "Checking test cluster ($(TEST_CONTEXT)) status..."
	kubectl config use-context $(TEST_CONTEXT)
	kubectl get pods -n $(NAMESPACE) -l "app in (nas-pdf-service,nas-pdf-service-gotenberg,nas-pdf-service-prometheus,nas-grafana,nas-jaeger)"
	kubectl get deploy -n $(NAMESPACE) -l "app in (nas-pdf-service,nas-pdf-service-gotenberg,nas-pdf-service-prometheus,nas-grafana,nas-jaeger)"
	kubectl get hpa -n $(NAMESPACE)

# Проверка продакшн окружения
check-prod: check-storage
	@echo "Checking production cluster ($(PROD_CONTEXT)) status..."
	kubectl config use-context $(PROD_CONTEXT)
	kubectl get pods -n $(NAMESPACE) -l "app in (nas-pdf-service,nas-pdf-service-gotenberg,nas-pdf-service-prometheus,nas-grafana,nas-jaeger)"
	kubectl get deploy -n $(NAMESPACE) -l "app in (nas-pdf-service,nas-pdf-service-gotenberg,nas-pdf-service-prometheus,nas-grafana,nas-jaeger)"
	kubectl get hpa -n $(NAMESPACE)

# ============================================================================
# Команды деплоя
# ============================================================================

# Универсальная команда деплоя (по умолчанию Helm)
deploy: check-env
	@echo "Using Helm deploy (default). For legacy kubectl path use 'make deploy-raw' (see DEPRECATIONS.md)"
	$(MAKE) helm-deploy

# Legacy kubectl deploy (сохранён для совместимости)
deploy-raw: check-env
	@echo "[DEPRECATED] Raw kubectl deploy path. Prefer 'make helm-deploy' (see DEPRECATIONS.md)"
	@echo "Applying PostgreSQL ConfigMap (idempotent)..."
	kubectl create configmap nas-pdf-service-postgres-config \
		--from-literal=POSTGRES_DB=pdf_service \
		--from-literal=POSTGRES_USER=pdf_service \
		--from-literal=POSTGRES_PASSWORD=pdf_service_pass \
		-n $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	@echo "Applying template ConfigMap (idempotent)..."
	kubectl create configmap nas-pdf-service-templates \
		--from-file=template.docx=internal/domain/pdf/templates/template.docx \
		-n $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	@echo "Deploying PostgreSQL..."
	kubectl config use-context $(CONTEXT)
	kubectl apply -f k8s/nas-pdf-service-postgres-deployment.yaml -n $(NAMESPACE)
	@echo "Waiting for PostgreSQL to be ready..."
	@kubectl wait --for=condition=ready pod -l app=nas-pdf-service-postgres -n $(NAMESPACE) --timeout=180s || echo "Warning: PostgreSQL pod not ready in time. Continuing anyway..."
	@echo "Applying DB migrations (init.sql) to existing database..."
	@powershell -Command "$${POD} = (kubectl get pods -n $(NAMESPACE) -l app=nas-pdf-service-postgres -o jsonpath='{.items[0].metadata.name}'); if (-not $${POD}) { Write-Host 'Postgres pod not found'; exit 1 }; kubectl exec -n $(NAMESPACE) $${POD} -- psql -U pdf_service -d pdf_service -f /docker-entrypoint-initdb.d/init.sql | Out-Null; Write-Host 'DB migrations applied'"
	@echo "Deploying main service..."
	@powershell -Command "$$DEPLOY_VERSION='$(VERSION)'; if ([string]::IsNullOrEmpty('$(VERSION)') -or '$(VERSION)' -eq 'latest') { $$DEPLOY_VERSION = 'latest'; Write-Host 'Using latest tag' } else { $$DEPLOY_VERSION = '$(VERSION)'.Trim(); Write-Host \"Using specified version: $$DEPLOY_VERSION\" }; if ('$(ENV)' -eq 'prod') { if ($$env:AUTO_APPROVE -eq 'y') { Write-Host 'AUTO_APPROVE=y detected: skipping interactive confirmation for production deploy.' } else { $$confirm = Read-Host -Prompt 'Are you sure you want to deploy to production? (y/N)'; if ($$confirm -ne 'y') { Write-Host 'Deployment cancelled.'; exit 1 } } }; kubectl config use-context $(CONTEXT); Write-Host 'Applying all configurations...'; kubectl apply -f k8s/nas-pdf-service-configmap.yaml -n $(NAMESPACE); kubectl create configmap nas-pdf-service-templates --from-file=template.docx=internal/domain/pdf/templates/template.docx -n $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -; kubectl apply -f k8s/nas-pdf-service-storage.yaml -n $(NAMESPACE); kubectl apply -f k8s/nas-pdf-service-gotenberg-deployment.yaml -n $(NAMESPACE); kubectl apply -f k8s/nas-pdf-service-prometheus-deployment.yaml -n $(NAMESPACE); kubectl apply -f k8s/nas-pdf-service-deployment.yaml -n $(NAMESPACE); kubectl apply -f k8s/nas-pdf-service-hpa.yaml -n $(NAMESPACE); Write-Host 'Updating deployment images...'; kubectl set image deployment/nas-pdf-service nas-pdf-service=$(IMG_PDF):$$DEPLOY_VERSION -n $(NAMESPACE); kubectl set image deployment/nas-pdf-service-gotenberg nas-pdf-service-gotenberg=$(IMG_GOTENBERG):$(TAG_GOTENBERG) -n $(NAMESPACE); kubectl set image deployment/nas-pdf-service-prometheus nas-pdf-service-prometheus=$(IMG_PROMETHEUS):$(TAG_PROMETHEUS) -n $(NAMESPACE); Write-Host 'Restarting deployments...'; kubectl rollout restart deployment/nas-pdf-service -n $(NAMESPACE); kubectl rollout restart deployment/nas-pdf-service-gotenberg -n $(NAMESPACE); kubectl rollout restart deployment/nas-pdf-service-prometheus -n $(NAMESPACE); Write-Host 'Waiting for rollouts to complete...'; kubectl rollout status deployment/nas-pdf-service -n $(NAMESPACE); kubectl rollout status deployment/nas-pdf-service-gotenberg -n $(NAMESPACE); kubectl rollout status deployment/nas-pdf-service-prometheus -n $(NAMESPACE); Write-Host 'Deployment to $(ENV) completed successfully'; Write-Host \"Use 'make status ENV=$(ENV)' to check deployment status\"; Write-Host \"Use 'make logs ENV=$(ENV)' to view logs\""

# Деплой локально через docker-compose
deploy-local:
	@echo "Deploying services locally..."
	@DEPLOY_VERSION="$(VERSION)"; \
	if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "latest" ]; then \
		if [ -f current_version.txt ]; then \
			DEPLOY_VERSION=$$(cat current_version.txt); \
			echo "Using version from current_version.txt: $$DEPLOY_VERSION"; \
		else \
			echo "Error: No version specified and no current_version.txt found. Please run 'make new-version' first or specify VERSION."; \
			exit 1; \
		fi; \
	else \
		echo "Using specified version: $$DEPLOY_VERSION"; \
	fi; \
	VERSION=$$DEPLOY_VERSION docker-compose down; \
	VERSION=$$DEPLOY_VERSION docker-compose up -d; \
	echo "Started services with version $(VERSION)"
	echo "Checking services status..."
	docker-compose ps

# ============================================================================
# Мониторинг и проверка статуса
# ============================================================================

status: check-env
	kubectl config use-context $(CONTEXT)
	kubectl get pods,services,deployments -n $(NAMESPACE)

logs: check-env
	kubectl config use-context $(CONTEXT)
	kubectl logs -f deployment/nas-pdf-service -n $(NAMESPACE)

check-grafana:
	@echo "Checking Grafana status..."
	kubectl get pods -n $(NAMESPACE) -l app=nas-grafana
	kubectl get svc -n $(NAMESPACE) nas-grafana

check-prometheus:
	@echo "Checking Prometheus status..."
	kubectl get pods -n $(NAMESPACE) -l app=nas-prometheus
	kubectl get svc -n $(NAMESPACE) nas-prometheus

check-jaeger:
	@echo "Checking Jaeger status..."
	kubectl get pods -n $(NAMESPACE) -l app=nas-jaeger
	kubectl get svc -n $(NAMESPACE) nas-jaeger

# Проверка endpoints системы отслеживания ошибок
test-error-system: check-env
	@echo "Testing error tracking system in $(ENV) environment..."
	kubectl config use-context $(CONTEXT)
	@powershell -Command "Write-Host 'Getting service info...'; try { $$serviceInfo = kubectl get svc nas-pdf-service -n $(NAMESPACE) -o jsonpath='{.spec.ports[0].nodePort}'; $$nodeIP = if ('$(ENV)' -eq 'prod') { '172.27.239.2' } else { '172.27.239.30' }; $$url = \"http://$${nodeIP}:$${serviceInfo}\"; Write-Host \"Service URL: $$url\"; Write-Host 'Testing /health...'; try { $$response = Invoke-RestMethod -Uri \"$$url/health\" -Method GET -TimeoutSec 10; Write-Host \"✅ Health: OK\" } catch { Write-Host \"❌ Health: Failed - $$($$($$_.Exception.Message))\" }; Write-Host 'Testing /errors...'; try { $$response = Invoke-WebRequest -Uri \"$$url/errors\" -Method GET -TimeoutSec 10; if ($$response.StatusCode -eq 200) { Write-Host \"✅ Errors UI: OK\" } else { Write-Host \"❌ Errors UI: Status $$($$($$response.StatusCode))\" } } catch { Write-Host \"❌ Errors UI: Failed - $$($$($$_.Exception.Message))\" }; Write-Host 'Testing /api/v1/errors/stats...'; try { $$response = Invoke-RestMethod -Uri \"$$url/api/v1/errors/stats\" -Method GET -TimeoutSec 10; Write-Host \"✅ Errors API: OK\" } catch { Write-Host \"❌ Errors API: Failed - $$($$($$_.Exception.Message))\" }; Write-Host 'Generating test error...'; try { $$response = Invoke-WebRequest -Uri \"$$url/test-error\" -Method GET -TimeoutSec 10; Write-Host \"✅ Test error generated\" } catch { Write-Host \"❌ Test error failed - $$($$($$_.Exception.Message))\" }; Write-Host 'Testing completed.' } catch { Write-Host \"❌ Failed to get service info: $$($$($$_.Exception.Message))\" }"

# ============================================================================
# Локальная разработка
# ============================================================================

dev: build
	./main

run-local:
	docker-compose up --build -d

port-forward-grafana:
	@echo "Setting up port forward for Grafana..."
	kubectl port-forward -n print-serv svc/nas-grafana 3000:3000

port-forward-prometheus:
	@echo "Setting up port forward for Prometheus..."
	kubectl port-forward -n print-serv svc/nas-prometheus 9090:9090

port-forward-jaeger:
	@echo "Setting up port forward for Jaeger UI..."
	kubectl port-forward -n print-serv svc/nas-jaeger 16686:16686

# ============================================================================
# Команды для работы со статистикой
# ============================================================================

# Очистка статистики
clear-stats: check-env
	@echo "Clearing statistics for $(ENV) environment..."
	@if [ "$(ENV)" = "prod" ]; then \
		read -p "Are you sure you want to clear PRODUCTION statistics? (y/N) " confirm; \
		if [ "$$confirm" != "y" ]; then \
			echo "Operation cancelled."; \
			exit 1; \
		fi; \
	fi
	kubectl config use-context $(CONTEXT)
	@echo "Getting PostgreSQL pod name..."
	@POSTGRES_POD=$$(kubectl get pods -n $(NAMESPACE) -l app=nas-pdf-service-postgres -o jsonpath='{.items[0].metadata.name}') && \
	echo "Clearing statistics tables..." && \
	kubectl exec -n $(NAMESPACE) $$POSTGRES_POD -- psql -U pdf_service -d pdf_service -c "TRUNCATE TABLE request_logs, docx_logs, gotenberg_logs, pdf_logs;"
	@echo "Statistics cleared successfully for $(ENV) environment"

# ============================================================================
# Helm команды
# ============================================================================

# Добавление репозиториев Helm
helm-repos:
	helm repo add bitnami https://raw.githubusercontent.com/bitnami/charts/archive-full-index/bitnami
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	helm repo add grafana https://grafana.github.io/helm-charts
	helm repo add jaegertracing https://jaegertracing.github.io/helm-charts
	helm repo update

# Установка зависимостей Helm
helm-deps: helm-repos
	cd helm/pdf-service && helm dependency update

# Проверка шаблонов Helm
helm-template: helm-deps
	helm template pdf-service helm/pdf-service --values helm/pdf-service/values-$(ENV).yaml

# Проверка синтаксиса Helm
helm-lint: helm-deps
	helm lint helm/pdf-service --values helm/pdf-service/values-$(ENV).yaml

# Установка/обновление через Helm
helm-deploy: check-env helm-deps
	@echo "Deploying to $(ENV) environment..."
	@if [ "$(ENV)" = "prod" ]; then \
		powershell -Command $$confirm = Read-Host -Prompt "Are you sure you want to deploy to production? (y/N)"; \
		if ($$confirm -ne "y") { \
			echo "Deployment cancelled."; \
			exit 1; \
		} \
	fi
	helm upgrade --install pdf-service helm/pdf-service \
		--namespace $(NAMESPACE) \
		--values helm/pdf-service/values-$(ENV).yaml \
		--set image.tag=$(VERSION)

# Удаление через Helm
helm-uninstall: check-env
	@echo "Uninstalling from $(ENV) environment..."
	@if [ "$(ENV)" = "prod" ]; then \
		powershell -Command $$confirm = Read-Host -Prompt "Are you sure you want to uninstall from production? (y/N)"; \
		if ($$confirm -ne "y") { \
			echo "Uninstall cancelled."; \
			exit 1; \
		} \
	fi
	helm uninstall pdf-service --namespace $(NAMESPACE)

# Получение статуса Helm релиза
helm-status: check-env
	helm status pdf-service --namespace $(NAMESPACE)

# Получение истории релизов Helm
helm-history: check-env
	helm history pdf-service --namespace $(NAMESPACE)

# Откат к предыдущей версии
helm-rollback: check-env
	@echo "Rolling back to previous version in $(ENV) environment..."
	@if [ "$(ENV)" = "prod" ]; then \
		powershell -Command $$confirm = Read-Host -Prompt "Are you sure you want to rollback production? (y/N)"; \
		if ($$confirm -ne "y") { \
			echo "Rollback cancelled."; \
			exit 1; \
		} \
	fi
	helm rollback pdf-service --namespace $(NAMESPACE)

# ============================================================================
# Утилиты для шаблонов
# ============================================================================

# Обновление шаблона в Kubernetes
update-template:
	@echo "Updating template in Kubernetes..."
	kubectl create configmap nas-pdf-service-templates --from-file=template.docx=internal/domain/pdf/templates/template.docx -n $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	@echo "Restarting pods to apply new template..."
	kubectl rollout restart deployment nas-pdf-service -n $(NAMESPACE)
	kubectl rollout status deployment nas-pdf-service -n $(NAMESPACE)
	@echo "Template updated successfully"

# Обновление зеркала во всех файлах
update-mirror:
	@echo "Current mirror: $(DOCKER_MIRROR)"
	@powershell -Command "if ('$(NEW_MIRROR)' -eq '') { Write-Host 'Error: Please specify NEW_MIRROR variable'; Write-Host 'Example: make update-mirror NEW_MIRROR=registry.example.com'; exit 1 }; Write-Host 'Updating mirror from $(DOCKER_MIRROR) to $(NEW_MIRROR)...'; $$files = @('k8s/*.yaml', 'helm/pdf-service/values.yaml', '.github/workflows/*.yaml'); foreach ($$pattern in $$files) { Get-ChildItem $$pattern -ErrorAction SilentlyContinue | ForEach-Object { $$content = Get-Content $$_.FullName -Raw; if ($$content -match '$(DOCKER_MIRROR)') { Write-Host \"Updating: $$($_.Name)\"; $$newContent = $$content -replace '$(DOCKER_MIRROR)', '$(NEW_MIRROR)'; Set-Content -Path $$_.FullName -Value $$newContent -NoNewline } } }; Write-Host 'Updating Makefile...'; $$makefileContent = Get-Content 'Makefile' -Raw; $$newMakefileContent = $$makefileContent -replace 'REGISTRY_PULL ?= $(REGISTRY_PULL)', \"REGISTRY_PULL ?= $(NEW_MIRROR)\"; Set-Content -Path 'Makefile' -Value $$newMakefileContent -NoNewline; Write-Host 'Mirror update completed. New mirror: $(NEW_MIRROR)'"

# Показать все файлы использующие зеркало
show-mirror-usage:
	@echo "Files using mirror $(DOCKER_MIRROR):"
	@powershell -Command "$$files = @('k8s/*.yaml', 'helm/pdf-service/values.yaml', '.github/workflows/*.yaml', 'Makefile'); foreach ($$pattern in $$files) { Get-ChildItem $$pattern -ErrorAction SilentlyContinue | ForEach-Object { $$content = Get-Content $$_.FullName -Raw; if ($$content -match '$(DOCKER_MIRROR)') { Write-Host \"  $$($$($$_.Name))\" } } }"

# Принудительное обновление deployment (когда зеркало не синхронизировалось)
force-update: check-env
	@echo "Force updating deployment in $(ENV) environment..."
	@powershell -Command "$$DEPLOY_VERSION='$(VERSION)'; if ([string]::IsNullOrEmpty('$(VERSION)') -or '$(VERSION)' -eq 'latest') { if (Test-Path current_version.txt) { $$DEPLOY_VERSION = (Get-Content current_version.txt).Trim(); Write-Host \"Using version from current_version.txt: $$DEPLOY_VERSION\" } else { Write-Host 'Error: No version specified and no current_version.txt found.'; exit 1 } } else { $$DEPLOY_VERSION = '$(VERSION)'.Trim(); Write-Host \"Using specified version: $$DEPLOY_VERSION\" }; kubectl config use-context $(CONTEXT); Write-Host 'Force updating image...'; kubectl set image deployment/nas-pdf-service nas-pdf-service=$(DOCKER_IMAGE):$$DEPLOY_VERSION -n $(NAMESPACE); Write-Host 'Restarting deployment...'; kubectl rollout restart deployment/nas-pdf-service -n $(NAMESPACE); Write-Host 'Waiting for rollout...'; kubectl rollout status deployment/nas-pdf-service -n $(NAMESPACE); Write-Host 'Force update completed for $(ENV)'"

# Проверка синхронизации зеркала
check-mirror: check-env
	@echo "Checking if mirror is synchronized..."
	kubectl config use-context $(CONTEXT)
	@powershell -Command "$$DEPLOY_VERSION='$(VERSION)'; if ([string]::IsNullOrEmpty('$(VERSION)') -or '$(VERSION)' -eq 'latest') { if (Test-Path current_version.txt) { $$DEPLOY_VERSION = (Get-Content current_version.txt).Trim() } else { $$DEPLOY_VERSION = 'latest' } }; Write-Host \"Profile: $(REGISTRY_PROFILE)\"; Write-Host \"Images: PDF=$(IMG_PDF):$$DEPLOY_VERSION, GOTENBERG=$(IMG_GOTENBERG):$(TAG_GOTENBERG), PROM=$(IMG_PROMETHEUS):$(TAG_PROMETHEUS), GRAFANA=$(IMG_GRAFANA):$(TAG_GRAFANA), JAEGER=$(IMG_JAEGER):$(TAG_JAEGER), POSTGRES=$(IMG_POSTGRES):$(TAG_POSTGRES)\"; $$pod = kubectl get pods -n $(NAMESPACE) -l app=nas-pdf-service -o jsonpath='{.items[0].metadata.name}'; if ($$pod) { Write-Host \"Current pod: $$pod\"; kubectl describe pod $$pod -n $(NAMESPACE) | findstr 'Image:' } else { Write-Host 'No pods found' }"

# =========================================================================
# Аутентификация и секреты для Nexus
# =========================================================================

# Вход в реестр Nexus (используйте переменные окружения NEXUS_USERNAME/NEXUS_PASSWORD)
docker-login-nexus:
	@powershell -Command "if ([string]::IsNullOrEmpty('$(NEXUS_USERNAME)') -or [string]::IsNullOrEmpty('$(NEXUS_PASSWORD)')) { Write-Host 'Error: please provide NEXUS_USERNAME and NEXUS_PASSWORD'; Write-Host 'Example: NEXUS_USERNAME=tech_irk NEXUS_PASSWORD=*** make docker-login-nexus'; exit 1 }; cmd /c \"echo $(NEXUS_PASSWORD)^| docker login $(REGISTRY_PUSH) -u $(NEXUS_USERNAME) --password-stdin\""

# Вход в pull‑реестр (DevOps), если требуются push-права или проверка доступа
docker-login-devops:
	@powershell -Command "if ([string]::IsNullOrEmpty('$(NEXUS_USERNAME)') -or [string]::IsNullOrEmpty('$(NEXUS_PASSWORD)')) { Write-Host 'Error: please provide NEXUS_USERNAME and NEXUS_PASSWORD'; Write-Host 'Example: NEXUS_USERNAME=tech_irk NEXUS_PASSWORD=*** make docker-login-devops'; exit 1 }; cmd /c \"echo $(NEXUS_PASSWORD)^| docker login $(REGISTRY_PULL) -u $(NEXUS_USERNAME) --password-stdin\""

# Создание/обновление imagePullSecret в namespace $(NAMESPACE)
create-nexus-pull-secret: check-env
	@powershell -Command "if ([string]::IsNullOrEmpty('$(NEXUS_USERNAME)') -or [string]::IsNullOrEmpty('$(NEXUS_PASSWORD)')) { Write-Host 'Error: please provide NEXUS_USERNAME and NEXUS_PASSWORD'; Write-Host 'Example: NEXUS_USERNAME=tech_irk NEXUS_PASSWORD=*** make create-nexus-pull-secret ENV=test'; exit 1 }; kubectl config use-context $(CONTEXT); kubectl create secret docker-registry registry-irk-rw --docker-server=$(DOCKER_MIRROR) --docker-username=$(NEXUS_USERNAME) --docker-password=$(NEXUS_PASSWORD) -n $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -"

# Получение URL сервиса
get-service-url: check-env
	@echo "Getting service URL for $(ENV) environment..."
	kubectl config use-context $(CONTEXT)
	@powershell -Command "try { $$serviceInfo = kubectl get svc nas-pdf-service -n $(NAMESPACE) -o jsonpath='{.spec.ports[0].nodePort}'; Write-Host \"NodePort: $$serviceInfo\"; Write-Host \"Service type: NodePort\"; Write-Host \"\"; Write-Host \"To access the service:\"; Write-Host \"  Replace <NODE_IP> with the actual cluster node IP\"; Write-Host \"  URL: http://<NODE_IP>:$$serviceInfo\"; Write-Host \"\"; Write-Host \"Endpoints:\"; Write-Host \"  Health:     http://<NODE_IP>:$$serviceInfo/health\"; Write-Host \"  Stats:      http://<NODE_IP>:$$serviceInfo/stats\"; Write-Host \"  Errors UI:  http://<NODE_IP>:$$serviceInfo/errors\"; Write-Host \"  Errors API: http://<NODE_IP>:$$serviceInfo/api/v1/errors\"; Write-Host \"  Test Error: http://<NODE_IP>:$$serviceInfo/test-error\"; Write-Host \"\"; Write-Host \"Known IPs:\"; Write-Host \"  Test cluster:  172.27.239.30\"; Write-Host \"  Prod cluster:  172.27.239.2\" } catch { Write-Host \"❌ Failed to get service info: $$($_.Exception.Message)\" }" 

# Явная цель для запуска миграций вручную при необходимости
migrate-db: check-env
	@echo "Applying DB migrations (init.sql) to $(ENV) environment..."
	@powershell -Command "$${POD} = (kubectl get pods -n $(NAMESPACE) -l app=nas-pdf-service-postgres -o jsonpath='{.items[0].metadata.name}'); if (-not $${POD}) { Write-Host 'Postgres pod not found'; exit 1 }; kubectl exec -n $(NAMESPACE) $${POD} -- psql -U pdf_service -d pdf_service -f /docker-entrypoint-initdb.d/init.sql | Out-Null; Write-Host 'DB migrations applied'"

# =========================================================================
# Установка CA для Nexus реестра на ноды (через DaemonSet)
# =========================================================================

# Установить DaemonSet, который положит ca.crt и hosts.toml на каждую ноду
install-registry-ca:
	kubectl apply -f k8s/registry-ca-installer.yaml
	@echo "Applied registry CA installer DaemonSet in kube-system."
	@echo "Next on each node: systemctl restart containerd (или docker), затем 'make uninstall-registry-ca'"

# Удалить DaemonSet
uninstall-registry-ca:
	kubectl delete -f k8s/registry-ca-installer.yaml --ignore-not-found
	@echo "Removed registry CA installer DaemonSet."