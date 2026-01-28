# FlowMind 代码规范

## 📝 强制规则

### ⭐ **所有代码必须有详细的注释**

**这是项目的核心规则，必须严格遵守！**

---

## Go 代码注释规范

### 函数注释

```go
// FunctionName 函数功能的简要说明
//
// 详细说明函数的作用、实现逻辑和注意事项
//
// Parameters:
//   - param1: 参数1的说明
//   - param2: 参数2的说明
//
// Returns:
//   - Type: 返回值说明
//   - error: 错误信息
func FunctionName(param1 Type1, param2 Type2) (Type, error) {
    // 实现代码
}
```

### 结构体注释

```go
// StructName 结构体功能的简要说明
//
// 详细说明结构体的用途和各字段含义
type StructName struct {
    // Field1 字段1的说明
    Field1 Type1

    // Field2 字段2的说明
    Field2 Type2
}
```

### 接口注释

```go
// InterfaceName 接口功能的简要说明
//
// 详细说明接口的用途和实现要求
type InterfaceName interface {
    // Method1 方法1的说明
    Method1(param Type) error

    // Method2 方法2的说明
    Method2() Type
}
```

### 重要逻辑注释

```go
// 步骤1: 获取数据
data := getData()

// 步骤2: 处理数据
processed := processData(data)

// 步骤3: 保存结果
saveResult(processed)
```

---

## TypeScript 代码注释规范

### 函数注释

```typescript
/**
 * functionName 函数功能的简要说明
 *
 * 详细说明函数的作用、实现逻辑和注意事项
 *
 * @param param1 - 参数1的说明
 * @param param2 - 参数2的说明
 * @returns 返回值说明
 *
 * @example
 * ```ts
 * const result = functionName(param1, param2);
 * ```
 */
function functionName(param1: Type1, param2: Type2): ReturnType {
  // 实现
}
```

### 组件注释

```tsx
/**
 * ComponentName 组件功能的简要说明
 *
 * 详细说明组件的用途、状态和props
 *
 * @param props - 组件属性
 * @returns {JSX.Element} 渲染的 JSX 元素
 */
function ComponentName(props: Props): JSX.Element {
  // 实现
}
```

### 接口/类型注释

```typescript
/**
 * InterfaceName 接口功能的简要说明
 *
 * 详细说明接口的用途
 */
interface InterfaceName {
  /** 属性1的说明 */
  property1: Type1;

  /** 属性2的说明 */
  property2: Type2;
}
```

### Hook 注释

```typescript
/**
 * useHookName Hook 功能的简要说明
 *
 * 详细说明 Hook 的用途和使用方法
 *
 * @param param - 参数说明
 * @returns 返回值说明
 *
 * @example
 * ```tsx
 * const value = useHookName(param);
 * ```
 */
export function useHookName(param: ParamType): ReturnType {
  // 实现
}
```

---

## Less/CSS 注释规范

```less
/**
 * 样式块功能的简要说明
 *
 * 详细说明样式的作用和使用场景
 */

// 单行注释：说明某个样式的用途
.class-name {
  property: value;
}

/**
 * MixinName 混入功能的简要说明
 *
 * @param param - 参数说明
 */
.mixinName(@param) {
  property: @param;
}
```

---

## 注释原则

### 必须添加注释的情况

1. ✅ **所有函数**：说明功能、参数、返回值
2. ✅ **所有结构体/接口/类**：说明用途
3. ✅ **所有组件**：说明功能和 props
4. ✅ **所有 Hook**：说明用途和使用方法
5. ✅ **复杂逻辑**：说明算法或处理流程
6. ✅ **重要的业务逻辑**：说明为什么这样处理
7. ✅ **常量和配置**：说明含义和用途

### 注释要求

- 使用 **中文** 编写注释
- 清晰、准确、完整
- 不要注释显而易见的代码
- 保持注释与代码同步

### 示例

❌ **不好的注释**：
```go
// 设置值
x = 1
```

✅ **好的注释**：
```go
// 设置默认重试次数为1，避免无限重试
retryCount = 1
```

---

## 违规处理

- PR 审查时，没有注释的代码将被拒绝
- 必须补全注释后才能合并

---

## 总结

**记住：没有注释 = 没有完成！**

每写一行代码，都要问自己：*"这段代码的注释写了吗？"*
