.PHONY: build run test clean tidy proto

# 变量
APP_NAME := idea-arena-server
BUILD_DIR := ./bin

# 编译
build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server/

# 开发模式运行
run:
	go run ./cmd/server/ -conf ./configs/

# 运行测试
test:
	go test -v ./...

# 清理构建产物
clean:
	rm -rf $(BUILD_DIR)

# 整理依赖
tidy:
	go mod tidy

# 代码检查
lint:
	golangci-lint run ./...

# 数据库迁移（GORM AutoMigrate，启动时自动执行）
migrate:
	@echo "Migration is automatic on startup via GORM AutoMigrate"

# Docker 构建
docker:
	docker build -t $(APP_NAME):latest .
