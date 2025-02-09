# Parse command line arguments
param(
    [Parameter(Mandatory=$true)]
    [ValidateSet('test', 'prod')]
    [string]$Environment,
    
    [Parameter(Mandatory=$false)]
    [switch]$Force,
    
    [Parameter(Mandatory=$false)]
    [switch]$SkipBackup
)

# Set UTF8 encoding
$OutputEncoding = [Console]::OutputEncoding = [Text.Encoding]::UTF8

# Bypass execution policy
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass -Force

# Enable all error output
$ErrorActionPreference = "Stop"

function Backup-Template {
    param (
        [string]$Path
    )
    $timestamp = Get-Date -Format "yyMMdd_HHmmss"
    $backupDir = "backups/templates"
    
    if (-not (Test-Path $backupDir)) {
        New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
    }
    
    $backupPath = "${backupDir}/template_${timestamp}.docx"
    Copy-Item -Path $Path -Destination $backupPath
    Write-Host "Template backed up to: $backupPath"
}

function Test-KubeConnection {
    param (
        [string]$Context
    )
    try {
        $result = kubectl config get-contexts $Context 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Context '$Context' not found in kubeconfig"
        }
        
        $result = kubectl get namespace print-serv 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Namespace 'print-serv' not found or no access"
        }
        
        return $true
    }
    catch {
        Write-Error "Kubernetes connection test failed: ${_}"
        return $false
    }
}

try {
    $envDisplay = if ($Environment -eq 'prod') { "PRODUCTION" } else { "TEST" }
    Write-Host "Starting template update in $envDisplay environment..."
    
    # Check if running in production without force flag
    if ($Environment -eq 'prod' -and -not $Force) {
        $confirmation = Read-Host "You are about to update the template in PRODUCTION environment. Are you sure? (y/N)"
        if ($confirmation -ne 'y') {
            Write-Host "Operation cancelled by user"
            exit 0
        }
    }

    # Check template file
    $templatePath = "internal/domain/pdf/templates/template.docx"
    if (-not (Test-Path $templatePath)) {
        throw "Template file not found: $templatePath"
    }
    
    # Create backup unless skipped
    if (-not $SkipBackup) {
        Backup-Template -Path $templatePath
    }

    # Set environment-specific kubeconfig
    $kubeconfigPath = if ($Environment -eq 'prod') {
        "$HOME/.kube/config_prod"
    } else {
        "$HOME/.kube/config"
    }
    
    if (-not (Test-Path $kubeconfigPath)) {
        throw "Kubeconfig not found: $kubeconfigPath"
    }
    
    $env:KUBECONFIG = $kubeconfigPath
    Write-Host "Using kubeconfig: $env:KUBECONFIG"

    # Set environment-specific context
    $context = if ($Environment -eq 'prod') {
        "efgi-irk-prod"
    } else {
        "efgi-irk-test"
    }
    
    # Test Kubernetes connection
    if (-not (Test-KubeConnection -Context $context)) {
        throw "Failed to connect to Kubernetes cluster"
    }
    
    kubectl config use-context $context
    Write-Host "Using context: $context"

    # Check current deployment status
    $deploymentStatus = kubectl get deployment nas-pdf-service -n print-serv -o json | ConvertFrom-Json
    if ($deploymentStatus.status.availableReplicas -lt 1) {
        Write-Warning "Warning: Current deployment has no available replicas!"
        if (-not $Force) {
            $confirmation = Read-Host "Do you want to continue? (y/N)"
            if ($confirmation -ne 'y') {
                Write-Host "Operation cancelled by user"
                exit 0
            }
        }
    }

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
  annotations:
    last-updated: "$(Get-Date -Format "yyyy-MM-dd HH:mm:ss")"
    updated-by: "$env:USERNAME"
binaryData:
  template.docx: $base64Content
"@

    # Save to file
    $outputPath = "k8s/templates-configmap-filled.yaml"
    Write-Host "Saving configuration to file: $outputPath"
    $yaml | Set-Content -Path $outputPath -NoNewline

    # Apply configuration to cluster
    Write-Host "Applying configuration to $envDisplay cluster..."
    kubectl apply -f $outputPath

    # Restart pods with confirmation in production
    if ($Environment -eq 'prod' -and -not $Force) {
        $confirmation = Read-Host "Template has been updated. Do you want to restart the pods now? (y/N)"
        if ($confirmation -ne 'y') {
            Write-Host "Template updated but pods were not restarted. Run 'kubectl rollout restart deployment/nas-pdf-service -n print-serv' manually when ready."
            exit 0
        }
    }

    Write-Host "Restarting pods in $envDisplay..."
    kubectl rollout restart deployment/nas-pdf-service -n print-serv

    Write-Host "Waiting for pods to be ready..."
    kubectl rollout status deployment/nas-pdf-service -n print-serv

    Write-Host "Template update in $envDisplay completed successfully!"
    
    # Show quick status
    Write-Host "`nCurrent deployment status:"
    kubectl get pods -n print-serv -l app=nas-pdf-service
}
catch {
    Write-Error "Error updating template in $envDisplay`: $($_.Exception.Message)"
    exit 1
} 