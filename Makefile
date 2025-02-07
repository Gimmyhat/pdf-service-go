.PHONY: build deploy-test deploy-prod deploy-all update-template-test update-template-prod help

# Автоматическая генерация версии в формате YYMMDD.HHMM
VERSION ?= $(shell powershell -Command "if (Test-Path current_version.txt) { Get-Content current_version.txt } else { Get-Date -Format 'yy.MM.dd.HHmm' }")
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

# Проверка наличия образа в Docker Hub
check-image:
	@echo "Checking if image $(IMAGE_NAME):$(VERSION) exists..."
	@docker pull $(IMAGE_NAME):$(VERSION) > /dev/null 2>&1 || (echo "Error: Image $(IMAGE_NAME):$(VERSION) not found in Docker Hub. Run 'make build' first." && exit 1)

# Получение последней версии из файла
get-version:
	@powershell -Command "if (-not (Test-Path current_version.txt)) { Write-Host 'Error: No version found. Run ''make build'' first.'; exit 1 }"
	@echo "Current version: $(VERSION)"

deploy-test: check-image
	@echo "Deploying version $(VERSION) to test cluster..."
	set "KUBECONFIG=$(TEST_KUBECONFIG)" && \
	kubectl apply -f k8s/configmap.yaml && \
	kubectl apply -f k8s/templates-configmap-filled.yaml && \
	kubectl apply -f k8s/gotenberg-deployment.yaml && \
	powershell -Command "(Get-Content k8s/pdf-service-deployment.yaml) -replace '$(IMAGE_NAME):.*', '$(IMAGE_NAME):$(VERSION)' | kubectl apply -f -" && \
	kubectl apply -f k8s/hpa.yaml && \
	kubectl rollout restart deployment/pdf-service -n print-serv && \
	kubectl rollout status deployment/pdf-service -n print-serv

deploy-prod: check-image
	@echo "Deploying version $(VERSION) to production cluster..."
	set "KUBECONFIG=$(PROD_KUBECONFIG)" && \
	kubectl apply -f k8s/configmap.yaml && \
	kubectl apply -f k8s/templates-configmap-filled.yaml && \
	kubectl apply -f k8s/gotenberg-deployment.yaml && \
	powershell -Command "(Get-Content k8s/pdf-service-deployment.yaml) -replace '$(IMAGE_NAME):.*', '$(IMAGE_NAME):$(VERSION)' | kubectl apply -f -" && \
	kubectl apply -f k8s/hpa.yaml && \
	kubectl rollout restart deployment/pdf-service -n print-serv && \
	kubectl rollout status deployment/pdf-service -n print-serv

deploy-all: deploy-test deploy-prod

update-template:
	@echo "Updating template ConfigMap..."
	powershell -Command "[Convert]::ToBase64String([IO.File]::ReadAllBytes('internal/domain/pdf/templates/template.docx'))" > template.base64
	powershell -Command "$$content = Get-Content k8s/templates-configmap.yaml -Raw; $$base64 = Get-Content template.base64; $$content -replace '{{ .base64Content }}',$$base64 | Set-Content k8s/templates-configmap-filled.yaml"
	del template.base64

update-template-test: update-template
	@echo "Updating template in test cluster..."
	set "KUBECONFIG=$(TEST_KUBECONFIG)" && \
	kubectl apply -f k8s/templates-configmap-filled.yaml && \
	kubectl rollout restart deployment/pdf-service -n print-serv

update-template-prod: update-template
	@echo "Updating template in production cluster..."
	set "KUBECONFIG=$(PROD_KUBECONFIG)" && \
	kubectl apply -f k8s/templates-configmap-filled.yaml && \
	kubectl rollout restart deployment/pdf-service -n print-serv

check-test:
	@echo "Checking test cluster status..."
	set "KUBECONFIG=$(TEST_KUBECONFIG)" && \
	kubectl get pods -n print-serv && \
	kubectl get deploy -n print-serv && \
	kubectl get hpa -n print-serv

check-prod:
	@echo "Checking production cluster status..."
	set "KUBECONFIG=$(PROD_KUBECONFIG)" && \
	kubectl get pods -n print-serv && \
	kubectl get deploy -n print-serv && \
	kubectl get hpa -n print-serv

help:
	@echo "Available targets:"
	@echo "  build               - Build and push Docker image (auto-versioned)"
	@echo "  get-version        - Show current version"
	@echo "  deploy-test         - Deploy to test cluster"
	@echo "  deploy-prod         - Deploy to production cluster"
	@echo "  deploy-all          - Deploy to both clusters"
	@echo "  update-template-test - Update template in test cluster"
	@echo "  update-template-prod - Update template in production cluster"
	@echo "  check-test          - Check test cluster status"
	@echo "  check-prod          - Check production cluster status"
	@echo ""
	@echo "Examples:"
	@echo "  make build                    # Build with auto-generated version"
	@echo "  make build VERSION=1.2.3      # Build with specific version"
	@echo "  make get-version             # Show current version"
	@echo "  make deploy-test              # Deploy latest build to test"
	@echo "  make deploy-all VERSION=1.2.3 # Deploy specific version to all clusters" 