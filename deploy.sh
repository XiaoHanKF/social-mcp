#!/bin/bash

# 部署脚本：编译并部署到 Docker 容器

set -e  # 遇到错误立即退出

echo "🔨 开始编译 Linux 版本..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o xhs-mcp-server .

echo "📦 复制到 Docker 容器..."
docker cp xhs-mcp-server xiaohongshu-mcp:/app/app

echo "🔄 重启容器..."
docker restart xiaohongshu-mcp

echo "🧹 清理本地文件..."
rm xhs-mcp-server

echo "✅ 部署完成！"
