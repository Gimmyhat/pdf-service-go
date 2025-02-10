# Проверяем наличие Chocolatey
if (!(Get-Command choco.exe -ErrorAction SilentlyContinue)) {
    Write-Host "Installing Chocolatey..."
    Set-ExecutionPolicy Bypass -Scope Process -Force
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
    Invoke-Expression ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
}

# Устанавливаем PyPy
Write-Host "Installing PyPy..."
choco install pypy3 -y

# Обновляем PATH
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")

# Устанавливаем зависимости через pip
Write-Host "Installing dependencies..."
pypy3 -m pip install --upgrade pip
pypy3 -m pip install -r requirements-pypy.txt

Write-Host "PyPy setup completed!" 