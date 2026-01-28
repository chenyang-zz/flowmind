# FlowMind 项目开发规范

## 📝 Git 提交规范

### ⚠️ 重要规则

**所有 Git 提交信息必须使用中文。**

### 提交格式

```
<type>(<scope>): <subject>

<body>
```

### Type 类型

- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式
- `refactor`: 重构
- `perf`: 性能优化
- `test`: 测试
- `chore`: 构建/工具

### 示例

```bash
# 新功能
git commit -m "feat(events): 实现事件总线系统

- 支持发布-订阅模式
- 支持通配符订阅
- 添加中间件支持"

# Bug 修复
git commit -m "fix(events): 修复取消订阅时的数据竞争问题"

# 文档
git commit -m "docs: 添加 Monitor Engine 实现计划"
```
