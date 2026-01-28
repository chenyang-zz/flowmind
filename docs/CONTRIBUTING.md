# 贡献指南

感谢你对 FlowMind 的贡献兴趣！

---

## 如何贡献

### 报告 Bug

在 [GitHub Issues](https://github.com/yourusername/flowmind/issues) 中提交问题，请包含：
- 清晰的标题
- 详细的描述
- 复现步骤
- 预期 vs 实际行为
- 环境信息（OS 版本、应用版本）
- 日志和截图

### 提交功能建议

1. 先检查是否有类似的建议
2. 详细描述功能需求和使用场景
3. 说明为什么这个功能重要

### 贡献代码

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交改动 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

---

## 代码规范

### Go 代码

- 遵循 [Effective Go](https://go.dev/doc/effective_go)
- 使用 `gofmt` 格式化
- 运行 `golangci-lint` 检查
- 添加单元测试

### 前端代码

- 遵循 ESLint 配置
- 使用 Prettier 格式化
- 组件命名：PascalCase
- 文件命名：kebab-case

### 提交信息

遵循 [Conventional Commits](https://www.conventionalcommits.org/)：

```
feat: add pattern mining algorithm
fix: resolve clipboard monitoring issue
docs: update API documentation
test: add unit tests for event bus
```

---

## Pull Request 流程

1. **描述改动**
   - 为什么需要这个改动
   - 解决了什么问题
   - 如何测试

2. **代码审查**
   - 至少一个维护者审查
   - 解决所有审查意见
   - 确保 CI 通过

3. **合并**
   - Squash and merge
   - 自动删除分支

---

## 开发指南

详见 [开发环境搭建](./implementation/01-development-setup.md)。

---

## 行为准则

- 尊重所有贡献者
- 欢迎不同观点
- 建设性讨论
- 关注问题本身

---

## 获取帮助

- GitHub Issues: 技术问题
- Discussions: 一般讨论
- Email: sheepzhao@example.com

---

**相关文档**：
- [开发环境搭建](./implementation/01-development-setup.md)
- [API 设计](./design/02-api-design.md)
