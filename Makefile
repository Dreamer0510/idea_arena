.PHONY: dev dev-backend dev-frontend install build up down logs clean

# 开发模式
dev-backend:
	cd backend && go run ./cmd/server/ -conf ./configs/

dev-frontend:
	cd frontend && npm run dev

# 安装依赖
install:
	cd backend && go mod tidy
	cd frontend && npm install

# Docker 构建 & 启动
up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

# 清理（包括数据卷）
clean:
	docker compose down -v
	cd backend && rm -rf bin/
	cd frontend && rm -rf .next/ node_modules/
