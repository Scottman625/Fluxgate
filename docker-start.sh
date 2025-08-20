#!/bin/bash

echo "🚀 啟動 Queue System 服務..."

# 停止並清理現有容器
echo "清理現有容器..."
docker-compose down

# 建構並啟動所有服務
echo "建構並啟動服務..."
docker-compose up --build -d

# 等待服務啟動
echo "等待服務啟動..."
sleep 20

# 檢查服務狀態
echo "檢查服務狀態..."
docker-compose ps

echo ""
echo "✅ 服務已啟動！"
echo ""
echo "📊 可用的服務："
echo "  - Queue System API: http://localhost:8080"
echo "  - Queue Dashboard:   http://localhost:8080"
echo "  - Prometheus:        http://localhost:9091"
echo "  - Grafana:           http://localhost:3000 (admin/admin)"
echo ""
echo "🔍 查看日誌:"
echo "  docker-compose logs -f queue-server"
echo ""
echo "🛑 停止服務:"
echo "  docker-compose down"
