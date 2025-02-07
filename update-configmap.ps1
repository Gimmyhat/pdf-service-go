#Requires -Version 5.0
using namespace System.Management.Automation
$global:ErrorActionPreference = 'Stop'

# Обход политики выполнения для текущего скрипта
$policy = [Microsoft.PowerShell.ExecutionPolicy]::Bypass
$scope = [Microsoft.PowerShell.ExecutionPolicyScope]::Process
[Microsoft.PowerShell.Security.PSPolicy]::SetExecutionPolicy($policy, $scope)

$base64Content = [Convert]::ToBase64String([IO.File]::ReadAllBytes("internal/domain/pdf/templates/template.docx"))
$configMapTemplate = Get-Content -Raw k8s/templates-configmap.yaml
$configMap = $configMapTemplate -replace '{{ \.base64Content }}', $base64Content
$configMap | Out-File -Encoding utf8 k8s/templates-configmap-filled.yaml 