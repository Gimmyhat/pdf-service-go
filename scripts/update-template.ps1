# Set UTF8 encoding
$OutputEncoding = [Console]::OutputEncoding = [Text.Encoding]::UTF8

# Bypass execution policy
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass -Force

# Enable all error output
$ErrorActionPreference = "Stop"

try {
    Write-Host "Starting template update in TEST environment..."

    # Check template file
    $templatePath = "internal/domain/pdf/templates/template.docx"
    if (-not (Test-Path $templatePath)) {
        throw "Template file not found: $templatePath"
    }

    # Set test kubeconfig
    $env:KUBECONFIG = "$HOME/.kube/config"
    Write-Host "Using test kubeconfig: $env:KUBECONFIG"

    # Encode to base64
    Write-Host "Encoding template to base64..."
    $base64Content = [Convert]::ToBase64String([IO.File]::ReadAllBytes($templatePath))

    # Create YAML
    Write-Host "Creating YAML configuration..."
    $yaml = @"
apiVersion: v1
kind: ConfigMap
metadata:
  name: nas-pdf-service-templates
  namespace: print-serv
binaryData:
  template.docx: $base64Content
"@

    # Save to file
    $outputPath = "k8s/templates-configmap-filled.yaml"
    Write-Host "Saving configuration to file: $outputPath"
    $yaml | Set-Content -Path $outputPath -NoNewline

    # Apply configuration to cluster
    Write-Host "Applying configuration to TEST cluster..."
    kubectl apply -f $outputPath

    # Restart pods
    Write-Host "Restarting pods in TEST..."
    kubectl rollout restart deployment/nas-pdf-service -n print-serv

    Write-Host "Waiting for pods to be ready..."
    kubectl rollout status deployment/nas-pdf-service -n print-serv

    Write-Host "Template update in TEST completed successfully!"
}
catch {
    Write-Error "Error updating template in TEST: $_"
    exit 1
} 