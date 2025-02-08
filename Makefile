.PHONY: build deploy-test deploy-prod deploy-all update-template-test update-template-prod help build-local deploy-grafana deploy-prometheus

# Автоматическая генерация версии в формате YYMMDD.HHMM
NEW_VERSION := $(shell powershell -Command "Get-Date -Format 'yy.MM.dd.HHmm'")
VERSION ?= $(if $(USE_NEW_VERSION),$(NEW_VERSION),$(shell powershell -Command "if (Test-Path current_version.txt) { Get-Content current_version.txt } else { '$(NEW_VERSION)' }"))
IMAGE_NAME = gimmyhat/pdf-service-go

# Пути к конфигам (измените на свои)
TEST_KUBECONFIG = $(HOME)/.kube/config
PROD_KUBECONFIG = $(HOME)/.kube/config_prod

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

deploy-test: check-image deploy-grafana
	@echo "Deploying version $(VERSION) to test cluster..."
	set "KUBECONFIG=$(TEST_KUBECONFIG)" && \
	kubectl apply -f k8s/configmap.yaml && \
	kubectl apply -f k8s/templates-configmap-filled.yaml && \
	kubectl apply -f k8s/gotenberg-deployment.yaml && \
	powershell -Command "(Get-Content k8s/nas-pdf-service-deployment.yaml) -replace '$(IMAGE_NAME):.*', '$(IMAGE_NAME):$(VERSION)' | kubectl apply -f -" && \
	kubectl apply -f k8s/hpa.yaml && \
	kubectl rollout restart deployment/nas-pdf-service -n print-serv && \
	kubectl rollout status deployment/nas-pdf-service -n print-serv

deploy-prod: check-image
	@echo "Deploying version $(VERSION) to production cluster..."
	set "KUBECONFIG=$(PROD_KUBECONFIG)" && \
	kubectl apply -f k8s/configmap.yaml && \
	kubectl apply -f k8s/templates-configmap-filled.yaml && \
	kubectl apply -f k8s/gotenberg-deployment.yaml && \
	powershell -Command "(Get-Content k8s/nas-pdf-service-deployment.yaml) -replace '$(IMAGE_NAME):.*', '$(IMAGE_NAME):$(VERSION)' | kubectl apply -f -" && \
	kubectl apply -f k8s/hpa.yaml && \
	kubectl rollout restart deployment/nas-pdf-service -n print-serv && \
	kubectl rollout status deployment/nas-pdf-service -n print-serv

deploy-all: deploy-test deploy-prod

update-template:
	@echo "Updating template ConfigMap..."
	powershell -ExecutionPolicy Bypass -Command "$$base64 = [Convert]::ToBase64String([IO.File]::ReadAllBytes('internal/domain/pdf/templates/template.docx')); $$yaml = 'apiVersion: v1', 'kind: ConfigMap', 'metadata:', '  name: nas-pdf-service-templates', '  namespace: print-serv', 'binaryData:', '  template.docx: ' + $$base64; Set-Content -Path k8s/templates-configmap-filled.yaml -Value $$yaml"

update-template-test: update-template
	@echo "Updating template in test cluster..."
	set "KUBECONFIG=$(TEST_KUBECONFIG)" && \
	kubectl apply -f k8s/templates-configmap-filled.yaml && \
	kubectl rollout restart deployment/nas-pdf-service -n print-serv

update-template-prod: update-template
	@echo "Updating template in production cluster..."
	set "KUBECONFIG=$(PROD_KUBECONFIG)" && \
	kubectl apply -f k8s/templates-configmap-filled.yaml && \
	kubectl rollout restart deployment/nas-pdf-service -n print-serv

check-test:
	@echo "Checking test cluster status..."
	set "KUBECONFIG=$(TEST_KUBECONFIG)" && \
	kubectl get pods -n print-serv -l "app in (nas-pdf-service,nas-gotenberg)" && \
	kubectl get deploy -n print-serv -l "app in (nas-pdf-service,nas-gotenberg)" && \
	kubectl get hpa -n print-serv

check-prod:
	@echo "Checking production cluster status..."
	set "KUBECONFIG=$(PROD_KUBECONFIG)" && \
	kubectl get pods -n print-serv -l "app in (nas-pdf-service,nas-gotenberg)" && \
	kubectl get deploy -n print-serv -l "app in (nas-pdf-service,nas-gotenberg)" && \
	kubectl get hpa -n print-serv

check-grafana:
	@echo "Checking Grafana status..."
	kubectl get pods -n print-serv -l app=nas-grafana
	kubectl get svc -n print-serv nas-grafana

check-prometheus:
	@echo "Checking Prometheus status..."
	kubectl get pods -n print-serv -l app=nas-prometheus
	kubectl get svc -n print-serv nas-prometheus

port-forward-grafana:
	@echo "Setting up port forward for Grafana..."
	kubectl port-forward -n print-serv svc/nas-grafana 3000:3000

port-forward-prometheus:
	@echo "Setting up port forward for Prometheus..."
	kubectl port-forward -n print-serv svc/nas-prometheus 9090:9090

deploy-monitoring: deploy-prometheus deploy-grafana

help:
	@echo "Available targets:"
	@echo "  build               - Build and push Docker image (using existing version if available)"
	@echo "  build-new          - Build and push Docker image (always generate new version)"
	@echo "  build-local        - Build Docker image for local development"
	@echo "  get-version        - Show current version"
	@echo "  deploy-test        - Deploy to test cluster (includes Grafana)"
	@echo "  deploy-prod        - Deploy to production cluster"
	@echo "  deploy-all         - Deploy to both clusters"
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
	@echo ""
	@echo "Examples:"
	@echo "  make build                    # Build with existing or auto-generated version"
	@echo "  make build-new               # Build with new auto-generated version"
	@echo "  make build VERSION=1.2.3     # Build with specific version"
	@echo "  make get-version            # Show current version"
	@echo "  make deploy-test             # Deploy latest build to test"
	@echo "  make deploy-all VERSION=1.2.3 # Deploy specific version to all clusters"
	@echo "  make port-forward-grafana    # Access Grafana UI at http://localhost:3000"
	@echo "  make port-forward-prometheus # Access Prometheus UI at http://localhost:9090" 