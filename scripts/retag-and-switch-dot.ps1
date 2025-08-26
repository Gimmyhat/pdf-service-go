#Requires -Version 5.0
param(
  [Parameter(Mandatory=$true)][string]$Tag,
  [string]$Namespace = "print-serv",
  [string]$Context   = "efgi-test",
  [string]$RW        = "registry-irk-rw.devops.rgf.local",
  [string]$RO        = "registry.devops.rgf.local",
  [string]$SrcRepo   = "gimmyhat/pdf-service-go",
  [string]$DotRepo   = "rgf.nas.pdf-service-go"
)

$ErrorActionPreference = "Stop"

function Wait-PodRunning([string]$ns, [string]$name, [int]$timeoutSec = 120) {
  $deadline = (Get-Date).AddSeconds($timeoutSec)
  while ((Get-Date) -lt $deadline) {
    $phase = kubectl -n $ns get pod $name -o jsonpath='{.status.phase}' 2>$null
    if ($phase -eq "Running") { return $true }
    Start-Sleep -Seconds 3
  }
  return $false
}

Write-Host "Using context: $Context"
kubectl config use-context $Context | Out-Null

Write-Host ("Pull from RW: {0}/{1}:{2}" -f $RW, $SrcRepo, $Tag)
docker pull ("{0}/{1}:{2}" -f $RW, $SrcRepo, $Tag)

Write-Host ("Retag to RW dot: {0}/{1}:{2}" -f $RW, $DotRepo, $Tag)
docker tag ("{0}/{1}:{2}" -f $RW, $SrcRepo, $Tag) ("{0}/{1}:{2}" -f $RW, $DotRepo, $Tag)

Write-Host ("Push RW dot: {0}/{1}:{2}" -f $RW, $DotRepo, $Tag)
docker push ("{0}/{1}:{2}" -f $RW, $DotRepo, $Tag)

$testPod = "ro-replica-test"
Write-Host ("Start RO pull test: {0}/{1}:{2}" -f $RO, $DotRepo, $Tag)
kubectl -n $Namespace delete pod $testPod --ignore-not-found | Out-Null

# Создаём временный Pod-манифест, чтобы избежать проблем с разбором аргументов
$img = ("{0}/{1}:{2}" -f $RO, $DotRepo, $Tag)
$tmp = New-TemporaryFile
@"
apiVersion: v1
kind: Pod
metadata:
  name: $testPod
  namespace: $Namespace
spec:
  containers:
    - name: tester
      image: $img
      imagePullPolicy: Always
      command: ["/bin/sh","-c","sleep 3600"]
  restartPolicy: Never
"@ | Set-Content -Path $tmp.FullName -Encoding UTF8

kubectl apply -f $tmp.FullName | Out-Null
Start-Sleep -Seconds 5

$events = kubectl -n $Namespace describe pod $testPod | Select-String -Pattern "Image:", "Failed to pull", "ErrImagePull", "Back-off", "unauthorized", "not found", "manifest unknown", "x509"
if ($events) { $events | ForEach-Object { $_.Line } }

if (-not (Wait-PodRunning -ns $Namespace -name $testPod -timeoutSec 120)) {
  Write-Host "RO pull failed or no replication yet. Deployment is not changed."
  exit 2
}

Write-Host ("RO pull OK. Switching deployment to {0}/{1}:{2}" -f $RO, $DotRepo, $Tag)
kubectl -n $Namespace set image deployment/nas-pdf-service nas-pdf-service=("{0}/{1}:{2}" -f $RO, $DotRepo, $Tag)
kubectl -n $Namespace rollout status deployment/nas-pdf-service
kubectl -n $Namespace describe deploy nas-pdf-service | Select-String -Pattern "Image:"
Write-Host "Done."


