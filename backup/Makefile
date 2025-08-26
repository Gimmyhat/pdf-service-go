.PHONY: build test lint tidy clean \
        docker-build docker-push new-version build-local \
        check-env get-version status logs \
        check-storage check-test check-prod check-grafana check-prometheus check-jaeger \
        deploy deploy-local deploy-storage \
        dev run-local port-forward-grafana port-forward-prometheus port-forward-jaeger \
        clear-stats \
        helm-repos helm-deps helm-template helm-lint helm-deploy helm-uninstall helm-status helm-history helm-rollback

# Основные переменные
APP_NAME = pdf-service-go
DOCKER_REPO = gimmyhat
DOCKER_IMAGE = $(DOCKER_REPO)/$(APP_NAME)
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
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

docker-push:
	docker push $(DOCKER_IMAGE):$(VERSION)

build-local:
	@echo "Building Docker image for local development..."
	docker-compose build
	@echo "Successfully built local development image"

# Создание нового образа с новой версией
new-version:
	@echo "Building new version: $(NEW_VERSION)"
	$(MAKE) docker-build VERSION=$(NEW_VERSION)
	$(MAKE) docker-push VERSION=$(NEW_VERSION)
	@echo "$(NEW_VERSION)" > current_version.txt
	@echo "New version $(NEW_VERSION) has been built and pushed"

# Получение текущей версии
get-version:
	@if [ -f current_version.txt ]; then \
		echo "Current version: $$(cat current_version.txt)"; \
	else \
		echo "Current version: latest (no current_version.txt found)"; \
	fi

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

# Универсальная команда деплоя
deploy: check-env
	@echo "Checking PostgreSQL ConfigMap..."
	@if ! kubectl get configmap nas-pdf-service-postgres-config -n $(NAMESPACE) > /dev/null 2>&1; then \
		echo "Creating PostgreSQL ConfigMap..."; \
		kubectl create configmap nas-pdf-service-postgres-config \
			--from-literal=POSTGRES_DB=pdf_service \
			--from-literal=POSTGRES_USER=pdf_service \
			--from-literal=POSTGRES_PASSWORD=pdf_service_pass \
			-n $(NAMESPACE); \
	fi
	@echo "Checking template ConfigMap..."
	@if ! kubectl get configmap nas-pdf-service-templates -n $(NAMESPACE) > /dev/null 2>&1; then \
		echo "Creating template ConfigMap..."; \
		kubectl create configmap nas-pdf-service-templates \
			--from-file=template.docx=internal/domain/pdf/templates/template.docx \
			-n $(NAMESPACE); \
	fi
	@echo "Deploying PostgreSQL..."
	kubectl config use-context $(CONTEXT)
	kubectl apply -f k8s/nas-pdf-service-postgres-deployment.yaml -n $(NAMESPACE)
	@echo "Waiting for PostgreSQL to be ready..."
	@kubectl wait --for=condition=ready pod -l app=nas-pdf-service-postgres -n $(NAMESPACE) --timeout=180s || echo "Warning: PostgreSQL pod not ready in time. Continuing anyway..."
	@echo "Deploying main service..."
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
	if [ "$(ENV)" = "prod" ]; then \
		read -p "Are you sure you want to deploy to production? (y/N) " confirm; \
		if [ "$$confirm" != "y" ]; then \
			echo "Deployment cancelled."; \
			exit 1; \
		fi; \
	fi; \
	kubectl config use-context $(CONTEXT); \
	echo "Applying all configurations..."; \
	kubectl apply -f k8s/nas-pdf-service-configmap.yaml -n $(NAMESPACE); \
	kubectl apply -f k8s/nas-pdf-service-templates-configmap-filled.yaml -n $(NAMESPACE); \
	kubectl apply -f k8s/nas-pdf-service-storage.yaml -n $(NAMESPACE); \
	kubectl apply -f k8s/nas-pdf-service-gotenberg-deployment.yaml -n $(NAMESPACE); \
	kubectl apply -f k8s/nas-pdf-service-prometheus-deployment.yaml -n $(NAMESPACE); \
	kubectl apply -f k8s/nas-pdf-service-deployment.yaml -n $(NAMESPACE); \
	kubectl apply -f k8s/nas-pdf-service-hpa.yaml -n $(NAMESPACE); \
	echo "Updating deployment image..."; \
	kubectl set image deployment/nas-pdf-service nas-pdf-service=$(DOCKER_IMAGE):$$DEPLOY_VERSION -n $(NAMESPACE); \
	echo "Restarting deployments..."; \
	kubectl rollout restart deployment/nas-pdf-service -n $(NAMESPACE); \
	kubectl rollout restart deployment/nas-pdf-service-gotenberg -n $(NAMESPACE); \
	kubectl rollout restart deployment/nas-pdf-service-prometheus -n $(NAMESPACE); \
	echo "Waiting for rollouts to complete..."; \
	kubectl rollout status deployment/nas-pdf-service -n $(NAMESPACE); \
	kubectl rollout status deployment/nas-pdf-service-gotenberg -n $(NAMESPACE); \
	kubectl rollout status deployment/nas-pdf-service-prometheus -n $(NAMESPACE); \
	echo "Deployment to $(ENV) completed successfully"; \
	echo "Use 'make status ENV=$(ENV)' to check deployment status"; \
	echo "Use 'make logs ENV=$(ENV)' to view logs"

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