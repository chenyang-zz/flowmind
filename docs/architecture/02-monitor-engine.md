# 监控引擎 (Monitor Engine)

监控引擎是 FlowMind 的基础组件，负责捕获用户的所有关键操作事件，为模式分析和 AI 理解提供数据源。

---

## 设计目标

1. **全面监控**：捕获键盘、剪贴板、应用切换、文件系统等事件
2. **高性能**：低资源占用，不影响系统性能
3. **隐私保护**：仅记录必要信息，用户可控
4. **跨平台潜力**：抽象层设计，便于未来移植

---

## 架构设计

### 组件关系

```
Monitor Engine (监控引擎)
    ├─→ Keyboard Monitor (键盘监控)
    │    ├─→ 平台层: 捕获所有键盘输入
    │    └─→ 业务层: 添加上下文，发布到事件总线
    ├─→ Hotkey Manager (快捷键管理器)
    │    └─→ 订阅键盘事件，匹配特定组合键，触发回调
    ├─→ Clipboard Monitor (剪贴板监听)
    ├─→ Application Monitor (应用切换监听)
    └─→ FileSystem Monitor (文件系统监听)
         ↓
    Event Bus (事件总线 - 发布/订阅模式)
         ↓
    ┌─────────────┬─────────────┬─────────────┐
    ↓             ↓             ↓             ↓
 Analyzer       Storage       Frontend      AI Service
```

### 分层架构

监控引擎采用分层架构设计，将平台相关逻辑与业务逻辑分离：

#### 1. 平台层 (Platform Layer)
- **职责**：与操作系统交互，捕获底层事件
- **位置**：`internal/platform/`
- **特点**：
  - 使用 CGO 调用系统 API
  - 处理平台特定的数据格式
  - 无业务逻辑，仅做数据转换

#### 2. 业务层 (Business Layer)
- **职责**：处理业务逻辑，添加上下文信息
- **位置**：`internal/monitor/`
- **特点**：
  - 订阅平台层事件
  - 获取当前应用上下文
  - 发布业务事件到事件总线

#### 3. 功能层 (Feature Layer)
- **职责**：实现特定功能，如快捷键匹配
- **特点**：
  - 订阅业务层事件
  - 实现功能逻辑
  - 发布自定义事件

### 数据流向

```
用户操作 (键盘输入)
    ↓
平台层键盘监控 (CGEventTap)
    ↓ (platform.KeyboardEvent)
业务层键盘监控 (添加上下文)
    ↓ (events.Event → EventBus)
功能层快捷键管理器 (订阅 + 匹配)
    ↓ (匹配成功)
触发回调 → 发布自定义事件
    ↓
前端/其他模块订阅事件 → 响应操作
```

### 核心接口

```go
// pkg/events/events.go
package events

// EventType 事件类型
type EventType string

const (
    // EventTypeKeyboard 键盘事件（所有键盘输入）
    EventTypeKeyboard EventType = "keyboard"

    // EventTypeClipboard 剪贴板事件
    EventTypeClipboard EventType = "clipboard"

    // EventTypeAppSwitch 应用切换事件
    EventTypeAppSwitch EventType = "app_switch"

    // EventTypeFileSystem 文件系统事件
    EventTypeFileSystem EventType = "file_system"
)

// Event 统一事件结构
type Event struct {
    // ID 事件唯一标识
    ID string

    // Type 事件类型
    Type EventType

    // Timestamp 事件时间戳
    Timestamp time.Time

    // Data 事件数据（键值对）
    Data map[string]interface{}

    // Context 事件上下文（当前应用等信息）
    Context *EventContext
}

// EventContext 事件上下文信息
type EventContext struct {
    // Application 当前应用名称
    Application string

    // BundleID 应用 Bundle ID
    BundleID string

    // WindowTitle 窗口标题
    WindowTitle string

    // FilePath 文件路径（可选）
    FilePath string

    // Selection 选中文本（可选）
    Selection string
}

// EventBus 事件总线接口
type EventBus interface {
    // Publish 发布事件
    Publish(topic string, event Event) error

    // Subscribe 订阅事件，返回订阅 ID
    Subscribe(topic string, handler EventHandler) string

    // Unsubscribe 取消订阅
    Unsubscribe(subscriptionID string)
}

// EventHandler 事件处理函数
type EventHandler func(event Event) error
```

---

## 键盘监控

键盘监控负责捕获用户的所有键盘输入，为行为分析和快捷键功能提供数据源。

### 键盘监控 vs 快捷键监控

监控引擎包含两个不同层面的键盘处理功能：

| 特性 | 键盘监控 | 快捷键监控 |
|------|-------------------|---------------------|
| **监控范围** | 所有键盘输入 | 仅注册的组合键 |
| **数据用途** | 行为分析、工作流生成 | 功能触发、操作快捷键 |
| **事件来源** | 直接从操作系统 | 从事件总线订阅 |
| **处理方式** | 记录并发布到事件总线 | 匹配后触发回调函数 |

- **键盘监控**：全面捕获所有键盘输入，用于记录用户行为、生成工作流、提供 AI 分析数据
- **快捷键监控**：监听特定的组合键（如 Cmd+Shift+M），触发预定义的功能（如打开 AI 面板）

### 架构分层

键盘监控采用三层架构设计：

```
┌─────────────────────────────────────────────────┐
│           业务层 (Business Layer)               │
│  internal/monitor/keyboard.go                   │
│  - 处理平台事件                                  │
│  - 添加上下文信息                                │
│  - 发布到事件总线                                │
│  - 管理快捷键管理器                              │
└─────────────────────────────────────────────────┘
                     ↑ 订阅
┌─────────────────────────────────────────────────┐
│          平台层 (Platform Layer)                │
│  internal/platform/keyboard_darwin.go           │
│  - 使用 CGEventTap 捕获键盘事件                  │
│  - CGO 调用 CoreGraphics API                    │
│  - 回调到业务层                                  │
└─────────────────────────────────────────────────┘
                     ↑ 捕获
┌─────────────────────────────────────────────────┐
│             操作系统 (macOS)                     │
│  CoreGraphics Framework                          │
│  CGEventTap                                      │
└─────────────────────────────────────────────────┘
```

### 业务层实现

```go
// internal/monitor/keyboard.go
package monitor

// KeyboardMonitor 键盘监控器（业务层）
//
// 负责键盘输入的监控和事件处理。本监控器采用分层架构：
//   - 业务层（本结构体）：处理事件、添加上下文、发布到事件总线
//   - 平台层（platform字段）：与操作系统交互，捕获底层键盘事件
//
// 工作流程：
//   1. 平台层捕获原始键盘事件
//   2. 业务层接收平台事件
//   3. 获取当前应用上下文信息
//   4. 构造业务事件并附加上下文
//   5. 发布到事件总线供其他模块消费
type KeyboardMonitor struct {
    // platform 平台层键盘监控器，负责与操作系统交互
    platform platform.KeyboardMonitor

    // eventBus 事件总线，用于发布键盘事件
    eventBus *events.EventBus

    // contextMgr 上下文管理器，用于获取当前应用信息
    contextMgr platform.ContextProvider

    // hotkeyManager 快捷键管理器，用于快捷键注册和匹配
    hotkeyManager *HotkeyManager

    // isRunning 监控器运行状态标志
    isRunning bool

    // mu 读写锁，保护并发访问
    mu sync.RWMutex
}

// NewKeyboardMonitor 创建键盘监控器
//
// 创建一个新的键盘监控器实例，并初始化其依赖的平台层组件、上下文管理器和快捷键管理器。
//
// Parameters:
//   - eventBus: 事件总线实例，用于发布键盘事件
//
// Returns: Monitor - 新创建的键盘监控器实例（返回接口类型）
func NewKeyboardMonitor(eventBus *events.EventBus) Monitor {
    return &KeyboardMonitor{
        platform:      platform.NewKeyboardMonitor(),
        eventBus:      eventBus,
        contextMgr:    platform.NewContextProvider(),
        hotkeyManager: NewHotkeyManager(eventBus),
    }
}

// Start 启动键盘监控
//
// 启动平台层的键盘监控器，并注册事件回调函数。
// 如果监控器已经在运行，则幂等地返回成功。
//
// Returns: error - 启动失败时返回错误（如缺少系统权限）
func (km *KeyboardMonitor) Start() error {
    km.mu.Lock()
    defer km.mu.Unlock()

    if km.isRunning {
        return nil // 已经在运行
    }

    logger.Info("启动键盘监控器")

    // 启动平台层监控器，并传入回调函数
    if err := km.platform.Start(km.handlePlatformEvent); err != nil {
        logger.Error("启动平台层键盘监控器失败", zap.Error(err))
        return err
    }

    // 启动快捷键管理器
    if km.hotkeyManager != nil {
        if err := km.hotkeyManager.Start(); err != nil {
            // 快捷键管理器启动失败，不影响主监控器
            logger.Warn("快捷键管理器启动失败，但不影响键盘监控")
        } else {
            // 注册预定义的快捷键
            registerPresetHotkeys(km.hotkeyManager, km.eventBus)
            logger.Info("预定义快捷键已注册")
        }
    }

    km.isRunning = true
    logger.Info("键盘监控器启动成功")
    return nil
}

// handlePlatformEvent 处理平台层传来的原始键盘事件
//
// 作为平台层监控器的回调函数，接收原始键盘事件并将其转换为业务事件。
// 处理流程：
//   1. 从上下文管理器获取当前应用信息
//   2. 提取键盘事件的关键信息（按键码、修饰键）
//   3. 构造业务键盘事件
//   4. 附加上下文信息（当前应用）
//   5. 发布到事件总线
//
// Parameters:
//   - event: 平台层的原始键盘事件
func (km *KeyboardMonitor) handlePlatformEvent(event platform.KeyboardEvent) {
    logger.Debug("捕获键盘事件",
        zap.Int("keycode", event.KeyCode),
        zap.Uint64("modifiers", event.Modifiers),
    )

    // 1. 获取上下文
    context := km.contextMgr.GetContext()

    // 2. 构造业务事件数据
    data := map[string]interface{}{
        "keycode":   event.KeyCode,
        "modifiers": event.Modifiers,
    }

    // 3. 创建业务事件
    businessEvent := events.NewEvent(events.EventTypeKeyboard, data)
    businessEvent.WithContext(context)

    // 4. 发布到事件总线
    if err := km.eventBus.Publish(string(events.EventTypeKeyboard), *businessEvent); err != nil {
        logger.Error("发布键盘事件失败", zap.Error(err))
    }
}

// GetHotkeyManager 获取快捷键管理器
//
// 返回键盘监控器管理的快捷键管理器实例，可用于注册和取消注册快捷键。
//
// Returns: *HotkeyManager - 快捷键管理器实例，可能为 nil（在监控器未初始化时）
func (km *KeyboardMonitor) GetHotkeyManager() *HotkeyManager {
    km.mu.RLock()
    defer km.mu.RUnlock()
    return km.hotkeyManager
}
```

### 平台层实现

```go
// internal/platform/keyboard_darwin.go
package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework Cocoa

#include <CoreFoundation/CoreFoundation.h>
#include <CoreGraphics/CoreGraphics.h>

// CGEventTap 回调函数
CGEventRef callback(CGEventTapProxy proxy, CGEventType type,
                   CGEventRef event, void *refcon) {
    // 只处理按键按下事件
    if (type == kCGEventKeyDown) {
        // 获取按键代码
        CGKeyCode keycode = (CGKeyCode)CGEventGetIntegerValueField(
            event, kCGKeyboardEventKeycode
        );

        // 获取修饰键标志
        CGEventFlags flags = CGEventGetFlags(event);

        // 回调到 Go
        goKeyboardCallback(keyCode, (uint64_t)flags);
    }

    return event;
}

// Go 回调函数声明
void goKeyboardCallback(int keyCode, uint64_t flags);
*/
import "C"
import (
    "fmt"
    "unsafe"
)

// KeyboardMonitor 平台层键盘监控器
type KeyboardMonitor struct {
    eventTap  C.CGEventTapRef
    callback  KeyboardCallback
    stopChan  chan struct{}
    isRunning bool
}

// KeyboardCallback 键盘事件回调函数类型
type KeyboardCallback func(event KeyboardEvent)

// KeyboardEvent 键盘事件
type KeyboardEvent struct {
    KeyCode   int
    Modifiers uint64
}

// NewKeyboardMonitor 创建平台层键盘监控器
func NewKeyboardMonitor() *KeyboardMonitor {
    return &KeyboardMonitor{
        stopChan: make(chan struct{}),
    }
}

// Start 启动监控器
func (km *KeyboardMonitor) Start(callback KeyboardCallback) error {
    km.callback = callback

    // 创建事件 tap
    eventMask := C.CGEventMaskBit(C.kCGEventKeyDown)

    km.eventTap = C.CGEventTapCreate(
        C.kCGSessionEventTap,
        C.kCGHeadInsertEventTap,
        C.kCGEventTapOptionDefault,
        eventMask,
        C.CGEventTapCallBack(C.callback),
        unsafe.Pointer(nil),
    )

    if km.eventTap == nil {
        return fmt.Errorf("创建事件 tap 失败，可能缺少辅助功能权限")
    }

    // 启用 tap
    C.CGEventTapEnable(km.eventTap, true)

    // 添加到 run loop
    src := C.CFMachPortCreateRunLoopSource(nil, C.CFMachPortRef(km.eventTap), 0)
    rl := C.CFRunLoopGetCurrent()
    C.CFRunLoopAddSource(rl, src, C.kCFRunLoopCommonModes)

    km.isRunning = true
    return nil
}

//export goKeyboardCallback
func goKeyboardCallback(keyCode C.int, flags C.uint64_t) {
    // 通过全局变量或其他方式调用回调
    // 这部分需要根据实际实现调整
}

// Stop 停止监控器
func (km *KeyboardMonitor) Stop() error {
    if !km.isRunning {
        return nil
    }

    if km.eventTap != nil {
        C.CGEventTapDisable(km.eventTap)
        C.CFRelease(C.CFTypeRef(km.eventTap))
        km.eventTap = nil
    }

    km.isRunning = false
    close(km.stopChan)
    return nil
}

// IsRunning 检查运行状态
func (km *KeyboardMonitor) IsRunning() bool {
    return km.isRunning
}
```

### 键盘事件数据

每个键盘事件包含以下信息：

```go
// 事件数据示例
{
    "keycode": 46,          // 按键代码（macOS 虚拟键码）
    "modifiers": 0x120000   // 修饰键标志位（Command + Shift）
}

// 修饰键标志位（位掩码）
const (
    ModifierCommand  = 0x100000  // Command 键 (⌘)
    ModifierShift    = 0x20000   // Shift 键 (⇧)
    ModifierControl  = 0x10000   // Control 键 (⌃)
    ModifierOption   = 0x80000   // Option 键 (⌥)
)

// 常见键码（部分）
0 = A,    1 = S,    2 = D,    3 = F,    4 = H,    5 = G,    6 = Z,    7 = X
8 = C,    9 = V,   10 = B,   11 = ,?   12 = Q,   13 = W,   14 = E,   15 = R
16 = Y,   17 = T,   18 = 1,   19 = 2,   20 = 3,   21 = 4,   22 = 6,   23 = 5
...
46 = M,   48 = Tab, 49 = Space, 51 = Delete, 53 = Escape
...
```

---

## 快捷键监控

快捷键监控负责匹配特定的组合键，触发预定义的功能回调。

### 功能特性

- **快捷键注册**：支持动态注册和取消注册快捷键
- **多种格式支持**：支持 "Cmd+Shift+M" 等多种快捷键格式
- **并发安全**：使用读写锁保护并发访问
- **回调隔离**：每个回调在独立的 goroutine 中执行，避免阻塞
- **Panic 恢复**：自动捕获回调中的 panic，防止程序崩溃
- **动态启用/禁用**：支持运行时启用或禁用快捷键

### 核心组件

```go
// internal/monitor/hotkey.go

// Hotkey 快捷键定义
//
// Hotkey 表示一个具体的快捷键组合，包含按键代码和修饰键状态。
// 支持字符串格式解析，如 "Cmd+Shift+A"。
type Hotkey struct {
    // KeyCode 按键代码，对应 macOS 虚拟键码
    KeyCode int

    // Modifiers 修饰键标志位组合
    // 使用位运算组合多个修饰键，如 ModifierCommand | ModifierShift
    Modifiers uint64

    // StringRepresentation 快捷键的字符串表示
    // 用于调试和日志记录，格式如 "Cmd+Shift+A"
    StringRepresentation string
}

// HotkeyRegistration 快捷键注册信息
//
// 表示一个已注册的快捷键及其处理逻辑。
type HotkeyRegistration struct {
    // ID 注册唯一标识符，用于取消注册
    ID string

    // Hotkey 快捷键定义
    Hotkey *Hotkey

    // Callback 快捷键触发时的回调函数
    // 在独立的 goroutine 中执行，避免阻塞事件处理
    Callback HotkeyCallback

    // Enabled 快捷键是否启用
    // 可以通过 SetEnabled 动态切换
    Enabled bool
}

// HotkeyCallback 快捷键回调函数类型
//
// 当快捷键被触发时调用此函数。
// 回调函数在独立的 goroutine 中执行，避免阻塞事件处理流程。
//
// Parameters:
//   - registration: 快捷键注册信息，包含 ID 和快捷键定义
//   - context: 事件上下文，包含当前应用等信息
type HotkeyCallback func(registration *HotkeyRegistration, context *events.EventContext)

// HotkeyManager 快捷键管理器
//
// 负责快捷键的注册、匹配和生命周期管理。
//
// 工作流程：
//   1. 通过 Register 方法注册快捷键
//   2. 从事件总线订阅键盘事件
//   3. 收到键盘事件时，遍历所有已注册的快捷键进行匹配
//   4. 匹配成功时触发回调函数
type HotkeyManager struct {
    // registrations 已注册的快捷键映射
    registrations map[string][]*HotkeyRegistration

    // keyCodeMap 快捷键索引（加速匹配）
    // key: keycode + modifiers 组合（使用位运算构造）
    // value: 注册信息切片的引用
    keyCodeMap map[uint64][]*HotkeyRegistration

    // eventBus 事件总线，用于订阅键盘事件
    eventBus *events.EventBus

    // subscription 事件总线的订阅 ID
    subscription string

    // mu 读写锁，保护并发访问
    mu sync.RWMutex

    // isRunning 管理器运行状态标志
    isRunning bool
}
```

### 快捷键注册

```go
// Register 注册快捷键
//
// 注册一个新的快捷键及其回调函数。如果快捷键已被注册，会添加到该快捷键的回调列表中。
// 同一快捷键可以有多个回调，触发时会按注册顺序依次调用。
//
// Parameters:
//   - hotkeyStr: 快捷键字符串，格式如 "Cmd+Shift+A"
//   - callback: 快捷键触发时的回调函数
//
// Returns:
//   - string: 注册 ID，用于 Unregister 取消注册
//   - error: 快捷键格式错误或注册失败时返回错误
//
// 示例：
//   id, err := manager.Register("Cmd+Shift+A", func(reg *HotkeyRegistration, ctx *events.EventContext) {
//       fmt.Println("快捷键触发！当前应用：", ctx.Application)
//   })
func (hm *HotkeyManager) Register(hotkeyStr string, callback HotkeyCallback) (string, error) {
    // 1. 解析快捷键
    hotkey, err := NewHotkey(hotkeyStr)
    if err != nil {
        logger.Warn("解析快捷键失败",
            zap.String("hotkey", hotkeyStr),
            zap.Error(err),
        )
        return "", err
    }

    hm.mu.Lock()
    defer hm.mu.Unlock()

    // 2. 规范化快捷键字符串（用于索引和存储）
    normalizedKey := strings.ToLower(strings.ReplaceAll(hotkeyStr, " ", ""))
    hotkey.StringRepresentation = normalizedKey

    // 3. 创建注册信息
    reg := &HotkeyRegistration{
        ID:       fmt.Sprintf("hotkey-%d-%d", hotkey.KeyCode, hotkey.Modifiers),
        Hotkey:   hotkey,
        Callback: callback,
        Enabled:  true,
    }

    // 4. 添加到注册映射
    hm.registrations[normalizedKey] = append(hm.registrations[normalizedKey], reg)

    // 5. 添加到快速查找索引
    lookupKey := hm.buildLookupKey(hotkey.KeyCode, hotkey.Modifiers)
    hm.keyCodeMap[lookupKey] = append(hm.keyCodeMap[lookupKey], reg)

    logger.Info("注册快捷键",
        zap.String("hotkey", hotkeyStr),
        zap.String("id", reg.ID),
        zap.Int("keycode", hotkey.KeyCode),
        zap.Uint64("modifiers", hotkey.Modifiers),
    )

    return reg.ID, nil
}
```

### 快捷键匹配

```go
// handleKeyboardEvent 处理键盘事件
//
// 订阅键盘事件的回调函数，负责：
//   1. 从事件中提取 keycode 和 modifiers
//   2. 构造快速查找键
//   3. 查找匹配的快捷键注册
//   4. 触发所有匹配的快捷键回调
//
// Parameters:
//   - event: 键盘事件
//
// Returns:
//   - error: 处理失败时返回错误
func (hm *HotkeyManager) handleKeyboardEvent(event events.Event) error {
    // 1. 提取 keycode 和 modifiers
    keycode, ok := event.Data["keycode"].(int)
    if !ok {
        return nil // 无效事件，忽略
    }

    modifiers, ok := event.Data["modifiers"].(uint64)
    if !ok {
        return nil // 无效事件，忽略
    }

    // 2. 应用修饰键掩码，过滤掉不相关的状态位（如 Caps Lock、Numeric Pad 等）
    maskedModifiers := modifiers & ModifierMask

    // 3. 构造快速查找键（keycode + masked modifiers 组合）
    lookupKey := hm.buildLookupKey(keycode, maskedModifiers)

    // 4. 查找匹配的注册
    hm.mu.RLock()
    registrations := hm.keyCodeMap[lookupKey]
    hm.mu.RUnlock()

    if len(registrations) == 0 {
        return nil // 没有匹配的快捷键
    }

    // 5. 触发所有匹配的快捷键回调
    for _, reg := range registrations {
        if !reg.Enabled {
            continue // 跳过禁用的快捷键
        }

        logger.Info("快捷键被触发",
            zap.String("hotkey", reg.Hotkey.String()),
            zap.String("id", reg.ID),
        )

        // 在独立的 goroutine 中执行回调，避免阻塞
        go func(r *HotkeyRegistration) {
            defer func() {
                if rec := recover(); rec != nil {
                    // 捕获回调中的 panic，防止崩溃
                    logger.Error("快捷键回调 panic",
                        zap.String("id", r.ID),
                        zap.Any("panic", rec),
                    )
                }
            }()
            r.Callback(r, event.Context)
        }(reg)
    }

    return nil
}
```

### 预定义快捷键

FlowMind 提供了以下预定义快捷键：

```go
// internal/monitor/hotkey_presets.go

// 预定义快捷键常量
const (
    // HotkeyAIAssistant AI 助手面板快捷键
    // 功能：打开/关闭 AI 助手面板，提供智能建议和自动化选项
    HotkeyAIAssistant = "Cmd+Shift+M"

    // HotkeyAutomationSuggestions 自动化建议快捷键
    // 功能：显示当前操作的自动化建议列表
    HotkeyAutomationSuggestions = "Cmd+Shift+A"

    // HotkeyKeybindings 快捷键列表快捷键
    // 功能：显示所有可用快捷键及其功能
    HotkeyKeybindings = "Cmd+Shift+K"

    // HotkeyToggleMonitoring 暂停/恢复监控快捷键
    // 功能：暂停或恢复工作流监控（隐私保护）
    HotkeyToggleMonitoring = "Cmd+Shift+P"

    // HotkeyShowStatus 显示状态快捷键
    // 功能：显示 FlowMind 当前状态和统计信息
    HotkeyShowStatus = "Cmd+Shift+H"
)
```

每个快捷键触发时会发布相应的事件到事件总线：

| 快捷键 | 事件类型 | 功能描述 |
|--------|---------|---------|
| Cmd+Shift+M | `hotkey.toggle_ai` | 打开/关闭 AI 助手面板 |
| Cmd+Shift+A | `hotkey.show_suggestions` | 显示自动化建议 |
| Cmd+Shift+K | `hotkey.show_keybindings` | 显示快捷键列表 |
| Cmd+Shift+P | `hotkey.toggle_monitoring` | 暂停/恢复监控 |
| Cmd+Shift+H | `hotkey.show_status` | 显示状态信息 |

前端或其他模块可以订阅这些事件来响应快捷键操作。

---

## 剪贴板监控

剪贴板监控负责检测剪贴板内容的变化，记录用户复制的内容。

### 实现

```go
// internal/monitor/clipboard_darwin.go
package monitor

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#include <Cocoa/Cocoa.h>

NSString* getClipboardContent() {
    NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
    return [pasteboard stringForType:NSPasteboardTypeString];
}

int getClipboardChangeCount() {
    NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
    return [pasteboard changeCount];
}
*/
import "C"

type ClipboardMonitor struct {
    eventBus        *events.EventBus
    stopChan        chan struct{}
    lastChangeCount int
    pollInterval    time.Duration
    contextMgr      platform.ContextProvider
}

func NewClipboardMonitor(eventBus *events.EventBus) *ClipboardMonitor {
    return &ClipboardMonitor{
        eventBus:     eventBus,
        stopChan:     make(chan struct{}),
        pollInterval: 500 * time.Millisecond, // 每 500ms 检查一次
        contextMgr:   platform.NewContextProvider(),
    }
}

func (cm *ClipboardMonitor) Start() error {
    // 获取初始 change count
    cm.lastChangeCount = int(C.getClipboardChangeCount())

    // 启动轮询
    go cm.poll()

    return nil
}

func (cm *ClipboardMonitor) poll() {
    ticker := time.NewTicker(cm.pollInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            currentCount := int(C.getClipboardChangeCount())

            if currentCount != cm.lastChangeCount {
                cm.lastChangeCount = currentCount

                // 获取剪贴板内容
                content := C.GoString(C.getClipboardContent())

                // 发送事件
                event := events.NewEvent(events.EventTypeClipboard, map[string]interface{}{
                    "content": content,
                    "length":  len(content),
                })
                event.WithContext(cm.contextMgr.GetContext())

                cm.eventBus.Publish(string(events.EventTypeClipboard), *event)
            }

        case <-cm.stopChan:
            return
        }
    }
}

func (cm *ClipboardMonitor) Stop() error {
    close(cm.stopChan)
    return nil
}
```

### 隐私保护

```go
// internal/monitor/clipboard_filter.go
type ClipboardFilter struct {
    sensitivePatterns []string
    ignoredApps       []string
}

func (cf *ClipboardFilter) ShouldRecord(content string, app string) bool {
    // 检查敏感应用（密码管理器）
    for _, ignored := range cf.ignoredApps {
        if app == ignored {
            return false
        }
    }

    // 检查敏感内容模式（密码、信用卡号等）
    for _, pattern := range cf.sensitivePatterns {
        if matched, _ := regexp.MatchString(pattern, content); matched {
            return false
        }
    }

    // 检查内容长度（避免记录大文件）
    if len(content) > 10000 {
        return false
    }

    return true
}
```

---

## 应用切换监控

应用切换监控负责检测当前活动应用的变化，记录应用使用情况。

### 实现

```go
// internal/monitor/application_darwin.go
package monitor

type ApplicationMonitor struct {
    eventBus    *events.EventBus
    stopChan     chan struct{}
    currentApp   string
    contextMgr   platform.ContextProvider
    pollInterval time.Duration
}

func NewApplicationMonitor(eventBus *events.EventBus) *ApplicationMonitor {
    return &ApplicationMonitor{
        eventBus:     eventBus,
        stopChan:     make(chan struct{}),
        pollInterval: 1 * time.Second,
        contextMgr:   platform.NewContextProvider(),
    }
}

func (am *ApplicationMonitor) Start() error {
    context := am.contextMgr.GetContext()
    am.currentApp = context.Application

    // 启动轮询
    go am.poll()

    return nil
}

func (am *ApplicationMonitor) poll() {
    ticker := time.NewTicker(am.pollInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            newContext := am.contextMgr.GetContext()

            if newContext.Application != am.currentApp {
                // 应用切换事件
                event := events.NewEvent(events.EventTypeAppSwitch, map[string]interface{}{
                    "from": am.currentApp,
                    "to":   newContext.Application,
                })
                event.WithContext(newContext)

                am.eventBus.Publish(string(events.EventTypeAppSwitch), *event)

                am.currentApp = newContext.Application
            }

        case <-am.stopChan:
            return
        }
    }
}

func (am *ApplicationMonitor) Stop() error {
    close(am.stopChan)
    return nil
}
```

### 应用时长统计

```go
// internal/monitor/app_tracker.go
type AppTracker struct {
    sessions  map[string]*AppSession
    eventBus  *events.EventBus
    mu        sync.RWMutex
}

type AppSession struct {
    AppName  string
    BundleID string
    Start    time.Time
    End      time.Time
}

func (at *AppTracker) OnAppSwitch(from, to string, bundleID string) {
    at.mu.Lock()
    defer at.mu.Unlock()

    // 结束上一个应用会话
    if session, exists := at.sessions[from]; exists {
        session.End = time.Now()

        // 发送会话事件
        event := events.NewEvent("app_session", map[string]interface{}{
            "app":       session.AppName,
            "bundle_id": session.BundleID,
            "start":     session.Start,
            "end":       session.End,
            "duration":  session.End.Sub(session.Start).Seconds(),
        })

        at.eventBus.Publish("app_session", *event)
    }

    // 开始新应用会话
    at.sessions[to] = &AppSession{
        AppName:  to,
        BundleID: bundleID,
        Start:    time.Now(),
    }
}
```

---

## 权限管理

### 辅助功能权限

```go
// internal/monitor/permissions.go
import "github.com/go-vgo/robotgo"

func CheckAccessibilityPermission() bool {
    // 检查是否有辅助功能权限
    trusted := robotgo.HasAccessibility()

    if !trusted {
        // 提示用户授予权限
        promptUserForPermission()
    }

    return trusted
}

func promptUserForPermission() {
    alert := &dialogs.Alert{
        Title:   "需要辅助功能权限",
        Message: "FlowMind 需要辅助功能权限来监听你的操作。\n\n" +
            "请前往 系统设置 > 隐私与安全性 > 辅助功能，\n" +
            "勾选 FlowMind。",
        Buttons: []string{"打开系统设置", "稍后"},
    }

    if alert.Show() == 0 {
        // 打开系统设置
        exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility").Run()
    }
}
```

---

## 性能优化

### 事件过滤

```go
// internal/monitor/filter.go
type EventFilter struct {
    minInterval time.Duration
    lastEvent   map[string]time.Time
}

func (ef *EventFilter) ShouldPass(event events.Event) bool {
    key := string(event.Type) + event.Context.Application

    if lastTime, exists := ef.lastEvent[key]; exists {
        if time.Since(lastTime) < ef.minInterval {
            return false // 过滤掉过快的事件
        }
    }

    ef.lastEvent[key] = time.Now()
    return true
}
```

### 批量处理

```go
// internal/monitor/batch.go
type EventBatcher struct {
    events    []events.Event
    batchSize int
    timeout   time.Duration
    output    chan<- []events.Event
    timer     *time.Timer
}

func (eb *EventBatcher) Add(event events.Event) {
    eb.events = append(eb.events, event)

    if len(eb.events) >= eb.batchSize {
        eb.flush()
    } else {
        eb.resetTimer()
    }
}

func (eb *EventBatcher) flush() {
    if len(eb.events) == 0 {
        return
    }

    eb.output <- eb.events
    eb.events = make([]events.Event, 0, eb.batchSize)
    eb.timer.Stop()
}

func (eb *EventBatcher) resetTimer() {
    if eb.timer != nil {
        eb.timer.Stop()
    }

    eb.timer = time.AfterFunc(eb.timeout, eb.flush)
}
```

---

## 测试

### 单元测试

```go
// internal/monitor/keyboard_test.go
package monitor

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// TestKeyboardMonitor_StartStop 测试键盘监控器的启动和停止
//
// 验证监控器能够正常启动和停止，并且状态标志正确设置。
func TestKeyboardMonitor_StartStop(t *testing.T) {
    eventBus := events.NewEventBus()
    monitor := NewKeyboardMonitor(eventBus)

    // 测试启动
    err := monitor.Start()
    require.NoError(t, err, "启动监控器应该成功")
    assert.True(t, monitor.IsRunning(), "监控器应该处于运行状态")

    // 测试幂等性（再次启动）
    err = monitor.Start()
    assert.NoError(t, err, "重复启动应该返回成功")

    // 测试停止
    err = monitor.Stop()
    require.NoError(t, err, "停止监控器应该成功")
    assert.False(t, monitor.IsRunning(), "监控器应该处于停止状态")
}

// TestHotkeyManager_Register 测试快捷键注册
//
// 验证快捷键能够正确注册，并且能够被成功匹配。
func TestHotkeyManager_Register(t *testing.T) {
    eventBus := events.NewEventBus()
    manager := NewHotkeyManager(eventBus)

    // 启动管理器
    err := manager.Start()
    require.NoError(t, err)
    defer manager.Stop()

    // 注册快捷键
    triggered := false
    id, err := manager.Register("Cmd+Shift+A", func(reg *HotkeyRegistration, ctx *events.EventContext) {
        triggered = true
    })

    require.NoError(t, err, "注册快捷键应该成功")
    assert.NotEmpty(t, id, "注册 ID 不应该为空")
    assert.True(t, manager.IsRegistered("Cmd+Shift+A"), "快捷键应该被注册")

    // 模拟键盘事件
    event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
        "keycode":   0,  // A 键
        "modifiers": 0x120000,  // Cmd + Shift
    })

    manager.handleKeyboardEvent(event)

    // 等待回调执行
    time.Sleep(100 * time.Millisecond)
    assert.True(t, triggered, "快捷键回调应该被触发")
}
```

---

## 相关文档

- [系统架构](./01-system-architecture.md)
- [分析引擎](./03-analyzer-engine.md)
- [实施指南 Phase 1](../implementation/02-phase1-monitoring.md)
