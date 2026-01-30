# Phase 5: 自动化引擎

**目标**: 自然语言 → 自动化脚本

**预计时间**: 见 PLAN.md

---

## 概述

本阶段将实现 FlowMind 的核心功能，详细说明请参考 [PLAN.md](../../PLAN.md)。

## 主要任务

设计自动化 DSL (JSON Schema)
实现 AI 脚本生成器
实现沙箱执行环境
实现任务调度器 (cron)
实现常用步骤 (Git/Slack/Shell)
设计自动化编辑器 UI

## 验收标准

- 详见 PLAN.md

## 关键技术点

- 详见 PLAN.md

## 下一步

完成后请进入下一个 Phase。

---

**相关文档**：

**前置文档**（上下阶段）:
- [系统架构总览](../architecture/00-system-architecture.md) - 理解整体架构
- [Phase 4: 知识管理](./05-phase4-knowledge.md) - 实现剪藏和知识图谱
- [开发环境搭建](./01-development-setup.md) - 配置开发环境

**本阶段详细架构**:
- [自动化引擎详解](../architecture/05-automation-engine.md) - 脚本生成和沙箱执行
- [AI 服务详解](../architecture/04-ai-service.md) - AI 驱动的脚本生成

**后续阶段**（下阶段）:
- [Phase 6: 可视化](./07-phase6-visualization.md) - 实现仪表板和可视化
