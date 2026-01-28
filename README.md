# FlowMind - AI 工作流智能体

> 一个主动的 AI 工作流伴侣，通过监控学习你的工作模式，主动发现问题并提供智能自动化建议。

## 项目愿景

FlowMind 不是一个简单的自动化工具，而是一个**主动的 AI 工作流伴侣**，它：

- 🧠 **理解你的工作模式**：通过监控学习你的习惯和偏好
- 💡 **主动发现问题**：AI 识别低效环节并主动建议优化
- 🤖 **智能执行自动化**：生成并执行自动化脚本，无需编程
- 📊 **可视化工作流**：让你清晰地看到自己的时间都去哪了
- 🧩 **跨应用联动**：让不同应用智能协作

## 核心功能

### 1. 智能工作流发现引擎 ⭐
- AI 自动识别重复性操作模式
- 主动建议自动化方案
- 一键生成并执行脚本

### 2. 实时 AI 助手面板 ⭐⭐
- 全局快捷键唤起（Cmd+Shift+M）
- 上下文感知，AI 理解当前工作状态
- 智能代码注入和建议

### 3. 智能剪藏与知识图谱 ⭐⭐⭐
- AI 自动分类、打标签、生成摘要
- 向量化存储和语义搜索
- 知识图谱建立关联，智能推荐

### 4. 工作流可视化仪表板 ⭐⭐
- 时间分布图表
- 工作流流程图（Sankey）
- AI 洞察和优化建议

### 5. 智能任务自动化引擎 ⭐⭐⭐
- 自然语言描述需求
- AI 生成自动化脚本
- 沙箱安全执行

## 技术栈

- **前端**：Wails v2 + React 19 + Vite + TailwindCSS + Zustand
- **后端**：Go 1.21+ + Clean Architecture (App/Service/Domain/Infrastructure)
- **AI**：Claude API + Ollama (本地)
- **存储**：SQLite + BBolt + Chromem-go (向量)
- **监控**：macOS 原生 API (CGEvent, NSPasteboard, NSWorkspace)
- **架构**：事件驱动 + 依赖注入 (Wire)

## 项目结构

```
flowmind/
├── main.go                 # Wails 入口（根目录）
├── internal/
│   ├── app/                # App 层（前后端桥梁）
│   ├── services/           # 服务层（业务编排）
│   ├── domain/             # 领域层（核心业务）
│   │   ├── monitor/        # 监控领域
│   │   ├── analyzer/       # 分析领域
│   │   ├── ai/             # AI 领域
│   │   ├── automation/     # 自动化领域
│   │   └── knowledge/      # 知识管理
│   └── infrastructure/     # 基础设施层
│       ├── storage/        # 存储实现
│       ├── platform/       # 平台相关代码
│       └── config/         # 配置管理
├── frontend/               # React 19 前端
│   └── src/
│       ├── components/     # UI 组件
│       ├── stores/         # Zustand 状态管理
│       ├── hooks/          # React Hooks
│       └── lib/            # 工具库
├── pkg/                    # 公共包
├── docs/                   # 详细文档
└── README.md               # 本文件
```

**详细架构请查看**：[系统架构文档](./docs/architecture/01-system-architecture.md)

## 实施进度

- [ ] Phase 1: 基础监控 (2-3 周)
- [ ] Phase 2: 模式识别 (2-3 周)
- [ ] Phase 3: AI 助手面板 (2 周)
- [ ] Phase 4: 智能剪藏与知识图谱 (3 周)
- [ ] Phase 5: 自动化引擎 (3 周)
- [ ] Phase 6: 可视化仪表板 (2 周)
- [ ] Phase 7: 打磨与优化 (2 周)

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+
- Wails v2.8+
- macOS 14+ (当前仅支持 macOS)

### 快速开始

```bash
# 1. 克隆项目
cd /Users/sheepzhao/WorkSpace/flowmind

# 2. 安装依赖
go mod download
cd frontend && npm install

# 3. 运行开发环境
wails dev
```

## 核心特性

### 1. 清晰分层架构
- **App 层**：前后端通信桥梁
- **Service 层**：业务流程编排
- **Domain 层**：核心业务逻辑
- **Infrastructure 层**：技术实现

### 2. 现代化前端
- **React 19**：最新特性（Compiler、Actions、useOptimistic）
- **TypeScript**：类型安全
- **TailwindCSS + Less**：样式管理
- **Zustand**：轻量级状态管理
- **Wails 绑定**：自动生成的类型安全 API

### 3. 严格代码规范
- **所有代码必须有详细注释**：这是项目的强制要求
- **类型安全**：Go + TypeScript
- **代码审查**：确保代码质量

## 文档

- [系统架构](./docs/architecture/01-system-architecture.md) - 完整的架构设计
- [监控引擎](./docs/architecture/02-monitor-engine.md) - 事件捕获机制
- [分析引擎](./docs/architecture/03-analyzer-engine.md) - 模式识别算法
- [AI 服务](./docs/architecture/04-ai-service.md) - AI 集成
- [自动化引擎](./docs/architecture/05-automation-engine.md) - 脚本执行
- [存储层](./docs/architecture/06-storage-layer.md) - 数据持久化

## 许可证

MIT License

## 作者

SheepZhao

---

**查看 [PLAN.md](./PLAN.md) 了解详细设计方案**
