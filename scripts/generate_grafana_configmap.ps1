# Создаем заголовок ConfigMap
@"
apiVersion: v1
kind: ConfigMap
metadata:
  name: nas-grafana-dashboards
  namespace: print-serv
  labels:
    grafana_dashboard: "true"
data:
  nas-pdf-service-dashboard.json: |
"@ | Set-Content k8s/grafana-dashboards.yaml

# Добавляем содержимое JSON файла с правильным отступом
Get-Content k8s/grafana-dashboard.json | ForEach-Object { "    $_" } | Add-Content k8s/grafana-dashboards.yaml 