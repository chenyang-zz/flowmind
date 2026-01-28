# 开发环境搭建

本文档介绍如何搭建 FlowMind 的开发环境。

---

## 系统要求

### 硬件要求
- **CPU**: Apple Silicon (M1/M2/M3) 或 Intel x86_64
- **内存**: 最低 8GB，推荐 16GB+
- **存储**: 最低 10GB 可用空间

### 软件要求
- **操作系统**: macOS 13.0+ (Ventura 或更高)
- **Go**: 1.21+
- **Node.js**: 18+ (推荐 20 LTS)
- **Wails**: v2.8+
- **Ollama**: 最新版（可选，用于本地 AI）

---

## 快速开始

### 1. 安装依赖

```bash
# 安装 Homebrew（如果未安装）
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# 安装 Go
brew install go

# 安装 Node.js
brew install node

# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 安装 Ollama（可选）
brew install ollama
ollama serve
```

### 2. 克隆项目

```bash
git clone https://github.com/yourusername/flowmind.git
cd flowmind
```

### 3. 安装 Go 依赖

```bash
go mod download
```

### 4. 安装前端依赖

```bash
cd frontend
pnpm install
cd ..
```

### 5. 配置环境变量

```bash
# 复制配置模板
cp config/dev.example.yaml config/dev.yaml

# 编辑配置文件
nano config/dev.yaml
```

### 6. 运行开发服务器

```bash
# 开发模式运行
wails dev
```

应用将在开发模式下启动，支持热重载。

---

## 详细配置

### Go 环境配置

```bash
# 设置 GOPATH（如果未设置）
echo 'export GOPATH=$HOME/go' >> ~/.zshrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.zshrc
source ~/.zshrc

# 验证安装
go version
```

### Wails 配置

```bash
# 初始化 Wails 项目（如果需要）
wails init -n flowmind -t react

# 安装 Wails 依赖
go get github.com/wailsapp/wails/v2@latest
```

### Ollama 配置

```bash
# 启动 Ollama 服务
ollama serve

# 下载模型
ollama pull llama3.2
ollama pull nomic-embed-text

# 验证
ollama list
```

### Claude API 配置

```bash
# 设置 API Key
export CLAUDE_API_KEY="sk-ant-..."

# 或保存在 ~/.zshrc
echo 'export CLAUDE_API_KEY="sk-ant-..."' >> ~/.zshrc
```

---

## IDE 配置

### VS Code

推荐安装以下扩展：

```json
{
  "recommendations": [
    "golang.go",
    "dbaeumer.vscode-eslint",
    "bradlc.vscode-tailwindcss",
    "wailsstudio.wails-vscode",
    "ms-vscode.makefile-tools"
  ]
}
```

配置 `.vscode/settings.json`:

```json
{
  "go.toolsManagement.autoUpdate": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "[go]": {
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
      "source.organizeImports": "always"
    }
  }
}
```

### GoLand

1. 打开项目目录
2. GoLand 会自动识别 Go 项目
3. 配置运行配置：
   - **Run kind**: File
   - **Files**: main.go
   - **Working directory**: 项目根目录

---

## 项目结构

```
flowmind/
├── main.go                       # 入口文件（根目录）
├── internal/                     # 私有代码
│   ├── app/                      # App 层
│   ├── domain/                   # 领域层
│   │   ├── monitor/
│   │   ├── analyzer/
│   │   ├── ai/
│   │   ├── automation/
│   │   └── knowledge/
│   └── infrastructure/           # 基础设施层
│       ├── config/
│       ├── storage/
│       └── platform/
├── pkg/                          # 公共代码
│   └── events/
├── frontend/                     # 前端代码
│   ├── src/
│   ├── package.json
│   └── wailsjs/
├── configs/                      # 配置文件
│   └── default.yaml
├── wails.json                    # Wails 配置
├── go.mod
├── go.sum
└── README.md
```

---

## 开发命令

### 构建

```bash
# 开发构建
wails build

# 生产构建
wails build -clean

# 指定平台
wails build -platform darwin/arm64
```

### 运行

```bash
# 开发模式
wails dev

# 直接运行 Go 后端（用于调试）
go run main.go
```

### 测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/monitor

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 代码检查

```bash
# 格式化代码
go fmt ./...

# 运行 linter
golangci-lint run

# 检查依赖漏洞
go mod verify
```

---

## 故障排除

### Wails 构建失败

```bash
# 清理缓存
wails build -clean

# 重新安装依赖
go mod tidy
go mod download

# 重新安装 Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### CGO 错误

```bash
# 确保 Xcode Command Line Tools 已安装
xcode-select --install

# 验证安装
xcode-select -p
```

### 权限问题

应用启动时可能需要授权以下权限：

1. **辅助功能**: 系统设置 > 隐私与安全性 > 辅助功能
2. **完全磁盘访问**: 系统设置 > 隐私与安全性 > 完全磁盘访问
3. **通知**: 系统设置 > 通知 > FlowMind

---

## 调试技巧

### Go 后端调试

```bash
# 使用 Delve
dlv debug main.go

# VS Code 调试配置
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Package",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}",
      "env": {
        "CLAUDE_API_KEY": "${env:CLAUDE_API_KEY}"
      }
    }
  ]
}
```

### 前端调试

开发模式下，前端会自动启用热重载和 React DevTools。

### 日志查看

```bash
# 查看应用日志
tail -f ~/Library/Logs/FlowMind/flowmind.log

# 调整日志级别
export LOG_LEVEL=debug
```

---

## 性能分析

```bash
# 启用 pprof
export DEBUG=true

# 访问性能分析端点
open http://localhost:6060/debug/pprof/
```

---

## 下一步

环境搭建完成后，可以开始：

1. 阅读 [系统架构](../architecture/01-system-architecture.md)
2. 查看 [Phase 1: 基础监控](./02-phase1-monitoring.md)
3. 运行示例代码

---

**相关文档**：
- [Phase 1 实施](./02-phase1-monitoring.md)
- [系统架构](../architecture/01-system-architecture.md)
- [API 设计](../design/02-api-design.md)
