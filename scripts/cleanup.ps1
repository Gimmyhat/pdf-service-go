# Скрипт очистки проекта от неиспользуемых файлов и директорий

# Временные файлы и артефакты сборки
Remove-Item -Path "pdf-service.exe" -Force -ErrorAction SilentlyContinue
Remove-Item -Path "test-docx.exe" -Force -ErrorAction SilentlyContinue
Remove-Item -Path "output*.docx" -Force -ErrorAction SilentlyContinue
Remove-Item -Path "result.pdf" -Force -ErrorAction SilentlyContinue
Remove-Item -Path "scripts/output.docx" -Force -ErrorAction SilentlyContinue
Remove-Item -Path "scripts/obj" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path "scripts/bin" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path "venv" -Recurse -Force -ErrorAction SilentlyContinue

# Устаревшие директории
Remove-Item -Path "deploy" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path "grafana" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path "backups" -Recurse -Force -ErrorAction SilentlyContinue

# Устаревшие файлы
Remove-Item -Path "update-configmap.ps1" -Force -ErrorAction SilentlyContinue
Remove-Item -Path "run_integration_tests.ps1" -Force -ErrorAction SilentlyContinue
Remove-Item -Path "test.env" -Force -ErrorAction SilentlyContinue
Remove-Item -Path "scripts/deploy.ps1" -Force -ErrorAction SilentlyContinue
Remove-Item -Path "scripts/build-and-push.ps1" -Force -ErrorAction SilentlyContinue
Remove-Item -Path "scripts/generate_grafana_configmap.sh" -Force -ErrorAction SilentlyContinue

# Создаем директорию test и перемещаем тестовые файлы
New-Item -Path "test" -ItemType Directory -Force
Move-Item -Path "test*.json" -Destination "test/" -Force -ErrorAction SilentlyContinue

Write-Host "Cleanup completed successfully" 