# FlowMind 文档中心

欢迎来到 FlowMind 文档中心！本文档库包含项目的完整设计文档、API 参考、实施指南等。

## 📚 文档目录

### 项目概述
- [项目简介](../README.md) - 项目愿景和快速开始
- [核心功能详解](./features/01-overview.md) - 5 大核心功能详细说明
- [差异化优势](./features/02-competitive-advantage.md) - 与现有工具的对比

### 架构设计
- [系统架构](./architecture/01-system-architecture.md) - 整体架构和分层设计
- [监控引擎](./architecture/02-monitor-engine.md) - 系统监控和事件捕获
- [分析引擎](./architecture/03-analyzer-engine.md) - 模式识别和序列挖掘
- [AI 服务](./architecture/04-ai-service.md) - Claude/Ollama 集成
- [自动化引擎](./architecture/05-automation-engine.md) - 脚本生成和执行
- [存储层](./architecture/06-storage-layer.md) - 数据库和向量存储

### 设计文档
- [数据库设计](./design/01-database-design.md) - SQLite schema 和数据模型
- [API 设计](./design/02-api-design.md) - RESTful API 和 WebSocket
- [事件系统](./design/03-event-system.md) - 事件总线设计
- [配置系统](./design/04-config-system.md) - 配置管理和默认值
- [安全设计](./design/05-security-design.md) - 权限和沙箱隔离

### 实施指南
- [开发环境搭建](./implementation/01-development-setup.md) - 环境配置和工具安装
- [Phase 1: 基础监控](./implementation/02-phase1-monitoring.md) - 监控引擎实现
- [Phase 2: 模式识别](./implementation/03-phase2-patterns.md) - 模式挖掘实现
- [Phase 3: AI 助手](./implementation/04-phase3-assistant.md) - AI 面板实现
- [Phase 4: 知识管理](./implementation/05-phase4-knowledge.md) - 剪藏和知识图谱
- [Phase 5: 自动化](./implementation/06-phase5-automation.md) - 自动化引擎实现
- [Phase 6: 可视化](./implementation/07-phase6-visualization.md) - 仪表板实现
- [Phase 7: 优化](./implementation/08-phase7-optimization.md) - 性能优化和打磨

### API 参考
- [Go API 参考](./api/go-api.md) - 后端 API 文档
- [前端 API 参考](./api/frontend-api.md) - Wails 绑定 API
- [事件 API](./api/event-api.md) - 事件系统 API

### 附录
- [常见问题](./faq.md) - FAQ
- [贡献指南](./contributing.md) - 如何贡献代码
- [许可证](./license.md) - MIT License

## 🚀 快速导航

**新开发者？** 从 [项目简介](../README.md) 开始

**想了解架构？** 查看 [系统架构](./architecture/01-system-architecture.md)

**准备开始编码？** 阅读 [开发环境搭建](./implementation/01-development-setup.md)

**寻找 API？** 浏览 [API 参考](./api/)

## 📊 项目进度

- [ ] Phase 1: 基础监控 (2-3 周)
- [ ] Phase 2: 模式识别 (2-3 周)
- [ ] Phase 3: AI 助手面板 (2 周)
- [ ] Phase 4: 智能剪藏与知识图谱 (3 周)
- [ ] Phase 5: 自动化引擎 (3 周)
- [ ] Phase 6: 可视化仪表板 (2 周)
- [ ] Phase 7: 打磨与优化 (2 周)

## 💡 核心概念

FlowMind 是一个 **AI 驱动的工作流智能体**，它通过以下核心技术提供价值：

1. **系统监控** - 捕获用户操作事件
2. **模式挖掘** - 识别重复性操作模式
3. **AI 理解** - Claude/Ollama 理解用户意图
4. **智能自动化** - 生成并执行脚本
5. **知识管理** - 向量搜索和知识图谱

## 🔗 相关资源

- [Wails 官方文档](https://wails.io/docs/introduction)
- [Claude API 文档](https://docs.anthropic.com/claude/reference)
- [ClaudeHeraldGo 项目](https://github.com/chenyang-zz/claude-herald-go)

## 📞 联系方式

- 作者：SheepZhao
- 项目地址：/Users/sheepzhao/WorkSpace/flowmind

---

**最后更新：** 2026-01-28
