# Проверяем аргументы
param(
    [Parameter(Mandatory=$true)]
    [string]$TemplatePath,
    
    [Parameter(Mandatory=$true)]
    [string]$DataPath,
    
    [Parameter(Mandatory=$true)]
    [string]$OutputPath
)

# Проверяем наличие PyPy
if (!(Get-Command pypy3.exe -ErrorAction SilentlyContinue)) {
    Write-Host "PyPy not found. Please run setup-pypy.ps1 first."
    exit 1
}

# Замеряем время выполнения
$startTime = Get-Date

# Запускаем скрипт через PyPy
pypy3 scripts/generate_docx.py $TemplatePath $DataPath $OutputPath

# Выводим время выполнения
$endTime = Get-Date
$duration = $endTime - $startTime
Write-Host "Execution time: $($duration.TotalSeconds) seconds" 