#Requires -Version 5.0
using namespace System.Management.Automation
$global:ErrorActionPreference = 'Stop'

# Обход политики выполнения для текущего скрипта
$policy = [Microsoft.PowerShell.ExecutionPolicy]::Bypass
$scope = [Microsoft.PowerShell.ExecutionPolicyScope]::Process
[Microsoft.PowerShell.Security.PSPolicy]::SetExecutionPolicy($policy, $scope)

param(
    [Parameter(Mandatory=$true)]
    [ValidateSet("efgi-test", "efgi-prod")]
    [string]$Cluster,
    
    [string]$Version = ""
)

# Проверяем наличие kubectl
if (-not (Get-Command kubectl -ErrorAction SilentlyContinue)) {
    throw "kubectl не установлен"
}

# Если версия не указана, пытаемся прочитать её из файла
if (-not $Version -and (Test-Path "current_version.txt")) {
    $Version = Get-Content "current_version.txt"
}

# Если версию не удалось получить, используем latest
if (-not $Version) {
    $Version = "latest"
}

Write-Host "Deploying to cluster: $Cluster with version: $Version"

# Обновляем версию образа в deployment.yaml
$deploymentFile = "k8s/deployment.yaml"
$content = Get-Content $deploymentFile -Raw
$content = $content -replace "gimmyhat/pdf-service-go:.*", "gimmyhat/pdf-service-go:$Version"
$content | Set-Content $deploymentFile

Write-Host "Updated image version in deployment.yaml"

# Применяем конфигурацию
Write-Host "Applying Kubernetes configurations..."

# Применяем все конфигурации с нужным контекстом
kubectl --context $Cluster apply -f k8s/deployment.yaml
if ($LASTEXITCODE -ne 0) { throw "Ошибка при применении deployment" }

kubectl --context $Cluster apply -f k8s/service.yaml
if ($LASTEXITCODE -ne 0) { throw "Ошибка при применении service" }

kubectl --context $Cluster apply -f k8s/hpa.yaml
if ($LASTEXITCODE -ne 0) { throw "Ошибка при применении HPA" }

# Обновляем ConfigMap с шаблоном
Write-Host "Updating template ConfigMap..."
./update-configmap.ps1
kubectl --context $Cluster apply -f k8s/templates-configmap-filled.yaml
if ($LASTEXITCODE -ne 0) { throw "Ошибка при обновлении ConfigMap" }

# Перезапускаем поды
Write-Host "Restarting pods..."
kubectl --context $Cluster rollout restart deployment/pdf-service -n print-serv
if ($LASTEXITCODE -ne 0) { throw "Ошибка при перезапуске подов" }

Write-Host "Waiting for rollout to complete..."
kubectl --context $Cluster rollout status deployment/pdf-service -n print-serv
if ($LASTEXITCODE -ne 0) { throw "Ошибка при ожидании обновления подов" }

Write-Host "Deployment to $Cluster completed successfully!" 