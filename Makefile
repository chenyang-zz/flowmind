.PHONY: dev build clean test run deps help

help:
	@echo "FlowMind - AI 工作流智能体"
	@echo ""
	@echo "可用命令:"
	@echo "  make dev      - 启动开发环境"
	@echo "  make build    - 构建应用"
	@echo "  make test     - 运行测试"
	@echo "  make run      - 运行已构建的应用"
	@echo "  make clean    - 清理构建文件"
	@echo "  make deps     - 安装依赖"

dev:
	@echo "启动开发环境..."
	wails dev

build:
	@echo "构建应用..."
	wails build

test:
	@echo "运行测试..."
	go test ./...
	cd frontend && pnpm test

run: build
	@echo "运行应用..."
	open build/bin/FlowMind.app

clean:
	@echo "清理构建文件..."
	rm -rf frontend/dist
	rm -rf build/bin
	rm -rf build/darwin

deps:
	@echo "安装依赖..."
	go mod download
	go mod tidy
	cd frontend && pnpm install

# 生成 Wails 绑定
generate:
	@echo "生成 Wails 绑定..."
	wails generate module

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...
	cd frontend && pnpm prettier --write "src/**/*.{ts,tsx}"

# Lint
lint:
	@echo "检查代码..."
	go vet ./...
	cd frontend && pnpm lint
