#Requires -Version 5.0
using namespace System.Management.Automation
$global:ErrorActionPreference = 'Stop'

# Обход политики выполнения для текущего скрипта
$policy = [Microsoft.PowerShell.ExecutionPolicy]::Bypass
$scope = [Microsoft.PowerShell.ExecutionPolicyScope]::Process
[Microsoft.PowerShell.Security.PSPolicy]::SetExecutionPolicy($policy, $scope)

param(
    [string]$Version = ""
)

# Проверяем наличие Docker
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "Docker не установлен"
}

# Если версия не указана, генерируем её на основе даты и времени
if (-not $Version) {
    $Version = Get-Date -Format "yyyy.MM.dd.HHmm"
}

Write-Host "Building Docker image with tag: $Version"

# Собираем образ
docker build -t "gimmyhat/pdf-service-go:$Version" .
if ($LASTEXITCODE -ne 0) {
    throw "Ошибка при сборке Docker образа"
}

Write-Host "Pushing Docker image to Docker Hub"

# Отправляем образ
docker push "gimmyhat/pdf-service-go:$Version"
if ($LASTEXITCODE -ne 0) {
    throw "Ошибка при отправке Docker образа"
}

# Создаем файл с текущей версией
$Version | Out-File -FilePath "current_version.txt"

Write-Host "Successfully built and pushed Docker image with tag: $Version" 