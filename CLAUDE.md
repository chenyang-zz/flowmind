# FlowMind 项目开发规范

## 📝 Git 提交规范

**提交信息必须使用中文**

### 格式
```
<type>(<scope>): <subject>
```

### Type 类型
`feat`(新功能) | `fix`(修复) | `docs`(文档) | `style`(格式) | `refactor`(重构) | `perf`(性能) | `test`(测试) | `chore`(构建)

### 示例
```bash
git commit -m "feat(events): 实现事件总线系统

- 支持发布-订阅模式
- 支持通配符订阅
- 添加中间件支持"
```

## 🧪 测试规范

**必须使用 `testify` 库**

```go
import (
    "github.com/stretchr/testify/assert"  // 失败后继续
    "github.com/stretchr/testify/require" // 失败后立即停止
)
```

- **assert**: 非关键条件验证
- **require**: 关键条件验证

### 测试函数注释
```go
// TestFunctionName 测试函数功能的简要说明
//
// 详细说明测试的目的、测试场景和预期行为
func TestFunctionName(t *testing.T) {
    // 测试代码
}
```

**测试函数必须说明测试的是什么功能和预期行为**

## 💬 代码注释规范

**所有代码必须有详细的中文注释**

### Go 注释
```go
// FunctionName 函数简要说明
// Parameters: param1-说明, param2-说明
// Returns: Type-返回值, error-错误
func FunctionName(param1 Type1, param2 Type2) (Type, error) {}

// StructName 结构体简要说明
type StructName struct {
    // Field1 字段说明
    Field1 Type1
}
```

### TypeScript 注释
```typescript
/**
 * 函数简要说明
 * @param param1 - 参数说明
 * @returns 返回值说明
 */
function functionName(param1: Type1): ReturnType {}
```

### 必须注释的场景
1. ✅ 所有函数/方法：功能、参数、返回值
2. ✅ 所有结构体/接口/类：用途
3. ✅ 所有组件：功能和 props
4. ✅ 复杂逻辑：算法或流程
5. ✅ 常量和配置：含义和用途

### 注释要求
- 使用中文，清晰准确
- 不注释显而易见的代码
- 保持注释与代码同步

**记住：没有注释 = 没有完成！**
