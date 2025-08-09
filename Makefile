# Tron区块链监控系统 Makefile

# 变量定义
APP_NAME := tron-monitor
BUILD_DIR := build
DOCKER_IMAGE := tron-monitor
DOCKER_TAG := latest

# Go相关变量
GO := go
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# 默认目标
.PHONY: all
all: build

# 构建应用程序
.PHONY: build
build:
	@echo "构建 $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(APP_NAME) .
	@echo "构建完成: $(BUILD_DIR)/$(APP_NAME)"

# 构建不同平台的二进制文件
.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: build-linux
build-linux:
	@echo "构建 Linux 版本..."
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .

.PHONY: build-darwin
build-darwin:
	@echo "构建 Darwin 版本..."
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 .

.PHONY: build-windows
build-windows:
	@echo "构建 Windows 版本..."
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .

# 运行应用程序
.PHONY: run
run: build
	@echo "运行 $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME)

# 运行开发模式
.PHONY: dev
dev:
	@echo "运行开发模式..."
	$(GO) run main.go

# 测试
.PHONY: test
test:
	@echo "运行测试..."
	$(GO) test -v ./...

# 测试覆盖率
.PHONY: test-coverage
test-coverage:
	@echo "运行测试覆盖率..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 清理
.PHONY: clean
clean:
	@echo "清理构建文件..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "清理完成"

# 格式化代码
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	$(GO) fmt ./...

# 代码检查
.PHONY: lint
lint:
	@echo "代码检查..."
	$(GO) vet ./...

# 下载依赖
.PHONY: deps
deps:
	@echo "下载依赖..."
	$(GO) mod download
	$(GO) mod tidy

# Docker相关命令
.PHONY: docker-build
docker-build:
	@echo "构建Docker镜像..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.PHONY: docker-run
docker-run: docker-build
	@echo "运行Docker容器..."
	docker run -d --name $(APP_NAME) -p 8080:8080 -p 6379:6379 $(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-stop
docker-stop:
	@echo "停止Docker容器..."
	docker stop $(APP_NAME) || true
	docker rm $(APP_NAME) || true

.PHONY: docker-clean
docker-clean:
	@echo "清理Docker镜像..."
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) || true

# Docker Compose命令
.PHONY: compose-up
compose-up:
	@echo "启动Docker Compose服务..."
	docker-compose up -d

.PHONY: compose-down
compose-down:
	@echo "停止Docker Compose服务..."
	docker-compose down

.PHONY: compose-logs
compose-logs:
	@echo "查看Docker Compose日志..."
	docker-compose logs -f

.PHONY: compose-restart
compose-restart:
	@echo "重启Docker Compose服务..."
	docker-compose restart

# 安装
.PHONY: install
install: build
	@echo "安装 $(APP_NAME)..."
	sudo cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/
	@echo "安装完成"

# 卸载
.PHONY: uninstall
uninstall:
	@echo "卸载 $(APP_NAME)..."
	sudo rm -f /usr/local/bin/$(APP_NAME)
	@echo "卸载完成"

# 帮助信息
.PHONY: help
help:
	@echo "Tron区块链监控系统 Makefile"
	@echo ""
	@echo "可用命令:"
	@echo "  build          - 构建应用程序"
	@echo "  build-all      - 构建所有平台的二进制文件"
	@echo "  run            - 运行应用程序"
	@echo "  dev            - 开发模式运行"
	@echo "  test           - 运行测试"
	@echo "  test-coverage  - 运行测试覆盖率"
	@echo "  clean          - 清理构建文件"
	@echo "  fmt            - 格式化代码"
	@echo "  lint           - 代码检查"
	@echo "  deps           - 下载依赖"
	@echo "  install        - 安装应用程序"
	@echo "  uninstall      - 卸载应用程序"
	@echo ""
	@echo "Docker命令:"
	@echo "  docker-build   - 构建Docker镜像"
	@echo "  docker-run     - 运行Docker容器"
	@echo "  docker-stop    - 停止Docker容器"
	@echo "  docker-clean   - 清理Docker镜像"
	@echo ""
	@echo "Docker Compose命令:"
	@echo "  compose-up     - 启动Docker Compose服务"
	@echo "  compose-down   - 停止Docker Compose服务"
	@echo "  compose-logs   - 查看Docker Compose日志"
	@echo "  compose-restart- 重启Docker Compose服务"
	@echo ""
	@echo "  help           - 显示此帮助信息"
