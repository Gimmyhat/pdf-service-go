.PHONY: build deploy-test deploy-prod deploy-all update-template-test update-template-prod help build-local deploy-grafana deploy-prometheus deploy-jaeger deploy-monitoring

# Автоматическая генерация версии в формате YYMMDD.HHMM
NEW_VERSION := $(shell powershell -Command "Get-Date -Format 'yy.MM.dd.HHmm'")
VERSION ?= $(if $(USE_NEW_VERSION),$(NEW_VERSION),$(shell powershell -Command "if (Test-Path current_version.txt) { Get-Content current_version.txt } else { '$(NEW_VERSION)' }"))
IMAGE_NAME = gimmyhat/pdf-service-go

# Пути к конфигам (измените на свои)
TEST_KUBECONFIG = $(HOME)/.kube/config
PROD_KUBECONFIG = $(HOME)/.kube/config_prod
TEST_CONTEXT = efgi-irk-test
PROD_CONTEXT = efgi-irk-prod

# Переменные окружения
ENV ?= test
CONTEXT = $(if $(filter prod,$(ENV)),$(PROD_CONTEXT),$(TEST_CONTEXT))

# Функция для повторных попытков kubectl команд
define retry_kubectl
	@for i in 1 2 3; do \
		$(1) && break || \
		if [ $$i -lt 3 ]; then \
			echo "Command failed, retrying in 5 seconds... (Attempt $$i of 3)"; \
			sleep 5; \
		else \
			exit 1; \
		fi; \
	done
endef

# Проверка подключения к кластеру
check-cluster-connection:
	@echo "Checking cluster connection..."
	@kubectl cluster-info > /dev/null || (echo "Error: Cannot connect to cluster" && exit 1)
	@kubectl get namespace print-serv > /dev/null || (echo "Error: Cannot access print-serv namespace" && exit 1)
	@echo "Cluster connection OK"

build:
	@echo "Building and pushing Docker image with version: $(VERSION)..."
	docker build -t $(IMAGE_NAME):$(VERSION) .
	docker push $(IMAGE_NAME):$(VERSION)
	@powershell -Command "Set-Content -Path current_version.txt -Value '$(VERSION)'"
	@echo "Successfully built and pushed $(IMAGE_NAME):$(VERSION)"

# Сборка с новой версией
build-new: USE_NEW_VERSION=1
build-new:
	@echo "Building with new version: $(NEW_VERSION)..."
	@$(MAKE) build VERSION=$(NEW_VERSION)

build-local:
	@echo "Building Docker image for local development..."
	docker-compose build
	@echo "Successfully built local development image"

# Проверка наличия образа в Docker Hub
check-image:
	@echo "Checking if image $(IMAGE_NAME):$(VERSION) exists..."
	@docker pull $(IMAGE_NAME):$(VERSION) > /dev/null 2>&1 || (echo "Error: Image $(IMAGE_NAME):$(VERSION) not found in Docker Hub. Run 'make build' first." && exit 1)

# Получение последней версии из файла
get-version:
	@powershell -Command "if (-not (Test-Path current_version.txt)) { Write-Host 'Error: No version found. Run ''make build'' first.'; exit 1 }"
	@echo "Current version: $(VERSION)"

deploy-grafana: check-image
	@echo "Deploying Grafana to cluster..."
	kubectl apply -f k8s/grafana-deployment.yaml
	kubectl apply -f k8s/grafana-datasources.yaml
	kubectl apply -f k8s/grafana-dashboards.yaml
	kubectl rollout restart deployment/nas-grafana -n print-serv
	kubectl rollout status deployment/nas-grafana -n print-serv

deploy-prometheus:
	@echo "Deploying Prometheus to cluster..."
	kubectl apply -f k8s/prometheus-deployment.yaml
	kubectl rollout restart deployment/nas-prometheus -n print-serv
	kubectl rollout status deployment/nas-prometheus -n print-serv

deploy-jaeger:
	@echo "Deploying Jaeger to cluster..."
	kubectl apply -f k8s/jaeger-deployment.yaml
	kubectl rollout restart deployment/nas-jaeger -n print-serv
	kubectl rollout status deployment/nas-jaeger -n print-serv

deploy-monitoring: deploy-prometheus deploy-grafana deploy-jaeger
	@echo "All monitoring services deployed"

# Проверка корректности ENV
check-env:
	@if ! [ "$(ENV)" = "test" ] && ! [ "$(ENV)" = "prod" ]; then \
		echo "Error: ENV must be either 'test' or 'prod'"; \
		exit 1; \
	fi

# Универсальная команда деплоя
deploy: check-image check-cluster-connection check-env
	@echo "Deploying version $(VERSION) to $(ENV) cluster ($(CONTEXT))..."
	$(call retry_kubectl,kubectl config use-context $(CONTEXT))
	@echo "Deploying monitoring services..."
	$(call retry_kubectl,kubectl apply -f k8s/prometheus-deployment.yaml)
	$(call retry_kubectl,kubectl apply -f k8s/grafana-deployment.yaml)
	$(call retry_kubectl,kubectl apply -f k8s/grafana-datasources.yaml)
	$(call retry_kubectl,kubectl apply -f k8s/grafana-dashboards.yaml)
	$(call retry_kubectl,kubectl apply -f k8s/jaeger-deployment.yaml)
	@echo "Waiting for monitoring services..."
	$(call retry_kubectl,kubectl rollout status deployment/nas-prometheus -n print-serv)
	$(call retry_kubectl,kubectl rollout status deployment/nas-grafana -n print-serv)
	$(call retry_kubectl,kubectl rollout status deployment/nas-jaeger -n print-serv)
	@echo "Deploying application services..."
	$(call retry_kubectl,kubectl apply -f k8s/configmap.yaml)
	$(call retry_kubectl,kubectl apply -f k8s/gotenberg-deployment.yaml)
	$(call retry_kubectl,kubectl rollout status deployment/nas-gotenberg -n print-serv)
	@echo "Updating template and deploying PDF service..."
	powershell -ExecutionPolicy Bypass -File scripts/update-template-unified.ps1 -Environment $(ENV) -Force -SkipRestart
	@powershell -Command "(Get-Content k8s/nas-pdf-service-deployment.yaml) -replace '$(IMAGE_NAME):.*', '$(IMAGE_NAME):$(VERSION)'" | kubectl apply -f -
	$(call retry_kubectl,kubectl apply -f k8s/hpa.yaml)
	$(call retry_kubectl,kubectl rollout status deployment/nas-pdf-service -n print-serv)
	@echo "Deployment to $(ENV) cluster completed successfully"

# Алиасы для обратной совместимости
deploy-test: 
	@$(MAKE) deploy ENV=test

deploy-prod: 
	@$(MAKE) deploy ENV=prod

deploy-all: check-image
	@echo "Starting deployment of version $(VERSION) to all clusters..."
	@$(MAKE) deploy ENV=test
	@echo "\nTest deployment completed. Checking test cluster status..."
	@kubectl config use-context $(TEST_CONTEXT)
	@kubectl get pods -n print-serv -l app=nas-pdf-service
	@echo "\nWaiting for confirmation before production deployment..."
	@powershell -Command "$$confirmation = Read-Host 'Do you want to proceed with production deployment? (y/N)'; if ($$confirmation -ne 'y') { exit 1 }"
	@echo "\nProceeding with production deployment..."
	@$(MAKE) deploy ENV=prod
	@echo "\nDeployment to all clusters completed successfully!"

update-template-test:
	@echo "Updating template in test cluster..."
	powershell -ExecutionPolicy Bypass -File scripts/update-template-unified.ps1 -Environment test

update-template-prod:
	@echo "Updating template in production cluster..."
	powershell -ExecutionPolicy Bypass -File scripts/update-template-unified.ps1 -Environment prod

check-test:
	@echo "Checking test cluster ($(TEST_CONTEXT)) status..."
	kubectl config use-context $(TEST_CONTEXT)
	kubectl get pods -n print-serv -l "app in (nas-pdf-service,nas-gotenberg,nas-prometheus,nas-grafana,nas-jaeger)"
	kubectl get deploy -n print-serv -l "app in (nas-pdf-service,nas-gotenberg,nas-prometheus,nas-grafana,nas-jaeger)"
	kubectl get hpa -n print-serv

check-prod:
	@echo "Checking production cluster ($(PROD_CONTEXT)) status..."
	kubectl config use-context $(PROD_CONTEXT)
	kubectl get pods -n print-serv -l "app in (nas-pdf-service,nas-gotenberg,nas-prometheus,nas-grafana,nas-jaeger)"
	kubectl get deploy -n print-serv -l "app in (nas-pdf-service,nas-gotenberg,nas-prometheus,nas-grafana,nas-jaeger)"
	kubectl get hpa -n print-serv

check-grafana:
	@echo "Checking Grafana status..."
	kubectl get pods -n print-serv -l app=nas-grafana
	kubectl get svc -n print-serv nas-grafana

check-prometheus:
	@echo "Checking Prometheus status..."
	kubectl get pods -n print-serv -l app=nas-prometheus
	kubectl get svc -n print-serv nas-prometheus

check-jaeger:
	@echo "Checking Jaeger status..."
	kubectl get pods -n print-serv -l app=nas-jaeger
	kubectl get svc -n print-serv nas-jaeger

port-forward-grafana:
	@echo "Setting up port forward for Grafana..."
	kubectl port-forward -n print-serv svc/nas-grafana 3000:3000

port-forward-prometheus:
	@echo "Setting up port forward for Prometheus..."
	kubectl port-forward -n print-serv svc/nas-prometheus 9090:9090

port-forward-jaeger:
	@echo "Setting up port forward for Jaeger UI..."
	kubectl port-forward -n print-serv svc/nas-jaeger 16686:16686

help:
	@echo "Available targets:"
	@echo "  build               - Build and push Docker image (using existing version if available)"
	@echo "  build-new          - Build and push Docker image (always generate new version)"
	@echo "  build-local        - Build Docker image for local development"
	@echo "  get-version        - Show current version"
	@echo "  deploy             - Deploy to specified cluster (use ENV=test or ENV=prod)"
	@echo "  deploy-test        - Deploy to test cluster (alias for deploy ENV=test)"
	@echo "  deploy-prod        - Deploy to production cluster (alias for deploy ENV=prod)"
	@echo "  deploy-all         - Deploy to both clusters sequentially"
	@echo "  deploy-grafana     - Deploy Grafana separately"
	@echo "  deploy-prometheus  - Deploy Prometheus separately"
	@echo "  deploy-monitoring  - Deploy both Grafana and Prometheus"
	@echo "  update-template-test - Update template in test cluster"
	@echo "  update-template-prod - Update template in production cluster"
	@echo "  check-test         - Check test cluster status"
	@echo "  check-prod         - Check production cluster status"
	@echo "  check-grafana      - Check Grafana status"
	@echo "  check-prometheus   - Check Prometheus status"
	@echo "  port-forward-grafana - Set up port forwarding for Grafana UI"
	@echo "  port-forward-prometheus - Set up port forwarding for Prometheus UI"
	@echo "  deploy-jaeger      - Deploy Jaeger separately"
	@echo "  check-jaeger       - Check Jaeger status"
	@echo "  port-forward-jaeger - Set up port forwarding for Jaeger UI"
	@echo ""
	@echo "Examples:"
	@echo "  make build                    # Build with existing or auto-generated version"
	@echo "  make build-new               # Build with new auto-generated version"
	@echo "  make build VERSION=1.2.3     # Build with specific version"
	@echo "  make deploy ENV=test         # Deploy to test cluster"
	@echo "  make deploy ENV=prod         # Deploy to production cluster"
	@echo "  make deploy-all VERSION=1.2.3 # Deploy specific version to all clusters"
	@echo "  make port-forward-grafana    # Access Grafana UI at http://localhost:3000"
	@echo "  make port-forward-prometheus # Access Prometheus UI at http://localhost:9090" 