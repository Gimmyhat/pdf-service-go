#!/bin/bash

# Создаем заголовок ConfigMap
cat << EOF > k8s/grafana-dashboards.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nas-grafana-dashboards
  namespace: print-serv
  labels:
    grafana_dashboard: "true"
data:
  nas-pdf-service-dashboard.json: |
EOF

# Добавляем содержимое JSON файла с правильным отступом
sed 's/^/    /' k8s/grafana-dashboard.json >> k8s/grafana-dashboards.yaml 