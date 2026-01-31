# Phase 3: AI 助手面板

**目标**: 实现实时 AI 助手面板，全局快捷键唤起 + 上下文感知对话

**预计时间**: 14-18 天

---

## 📋 概述

本阶段将实现 FlowMind 的**交互核心**——一个类似 Raycast/Alfred 的全局 AI 助手面板：

1. **全局快捷键** - 使用 `Cmd+Shift+M` 等快捷键唤起面板
2. **全局面板 UI** - 半透明、居中下、美观的对话界面
3. **上下文感知** - 通过 macOS Accessibility API 获取当前应用、窗口、选中文本
4. **流式对话** - 集成 AI 接口，支持实时流式响应
5. **代码注入** - 将 AI 生成的内容插入到当前应用光标位置

### ⚠️ 重要说明

**本文档中的代码实现仅作为参考和思路启发**，实际编码时需要：

1. **独立思考** - 不要机械照搬文档代码，要理解设计意图
2. **架构评估** - 根据实际需求选择合适的架构和设计模式
3. **技术选型** - 验证技术选型是否合理，是否有更优方案
4. **现有代码复用** - 检查 `internal/infrastructure/ai/` 中已实现的代码，优先复用
5. **渐进式实现** - 从最小可行方案开始，避免过度设计
6. **性能和可维护性** - 关注代码质量、测试覆盖和文档完善

**文档中的代码示例仅用于说明概念和思路，实际实现可能完全不同。**

### 核心体验

```
你在 VS Code 中写代码
    ↓
按 Cmd+Shift+M
    ↓
半透明面板从屏幕中下位置浮现
    ↓
AI: 我注意到你在写 useEffect，需要帮助吗？
    [1] 生成清理函数
    [2] 检查依赖项
    [3] 查看最佳实践
    [4] 自定义问题...
    ↓
选择 1 → AI 流式生成代码 → 自动插入到光标位置 → 面板消失
```

### 系统架构

按照 [系统架构规范](../architecture/00-system-architecture.md)，Phase 3 采用**三层架构**（当前阶段无需 Service 层）：

```
┌─────────────────────────────────────────────────────────┐
│                    React 19 前端层                       │
│  ┌───────────────────────────────────────────────────┐ │
│  │  AI Panel 组件 (React 19 + Tailwind 4)             │ │
│  │  - Panel.tsx (主面板)                              │ │
│  │  - MessageList.tsx (消息列表)                      │ │
│  │  - SuggestionBar.tsx (建议栏)                      │ │
│  │  - 流式渲染 + 乐观更新                              │ │
│  └───────────────────────────────────────────────────┘ │
│                        ▲                               │
│                        │ Wails Bindings               │
│                        │ (方法调用 + 事件推送)          │
│                        ▼                               │
│  ┌───────────────────────────────────────────────────┐ │
│  │  App 层 (internal/app/)                            │ │
│  │  - panel.go (PanelManager)                        │ │
│  │  - 快捷键注册 → 显示面板 → 上下文采集 → 注入结果    │ │
│  └───────────────────────────────────────────────────┘ │
│                        ▼                               │
│  ┌───────────────────────────────────────────────────┐ │
│  │  Domain 层 (internal/domain/)                      │ │
│  │  ┌─────────────────────────────────────────────┐ │ │
│  │  │ 监控领域 (monitor)                            │ │ │
│  │  │  - HotkeyManager (全局快捷键) ✅ 已实现       │ │ │
│  │  │  - ClipboardMonitor (剪贴板监控) ✅ 已实现    │ │ │
│  │  └─────────────────────────────────────────────┘ │ │
│  │                                                     │ │
│  │  ┌─────────────────────────────────────────────┐ │ │
│  │  │ AI 领域 (ai) - 待实现                         │ │ │
│  │  │  - AI 业务逻辑封装                            │ │ │
│  │  │  - 提示词模板管理                             │ │ │
│  │  │  - 对话历史管理                               │ │ │
│  │  └─────────────────────────────────────────────┘ │ │
│  └───────────────────────────────────────────────────┘ │
│                        ▼                               │
│  ┌───────────────────────────────────────────────────┐ │
│  │  Infrastructure 层 (internal/infrastructure/)      │ │
│  │  ┌─────────────────────────────────────────────┐ │ │
│  │  │ 平台层 (platform)                             │ │ │
│  │  │  - context.go (ContextProvider 接口)          │ │ │
│  │  │  - context_darwin.go (macOS 实现)             │ │ │
│  │  │    • GetFrontmostApp()                       │ │ │
│  │  │    • GetBundleID()                           │ │ │
│  │  │    • GetFocusedWindowTitle()                 │ │ │
│  │  │    • GetSelectedText() (新增)                │ │ │
│  │  │  - NSPanel (macOS 原生面板)                   │ │ │
│  │  │    • 毛玻璃效果 (NSVisualEffectView)          │ │ │
│  │  │    • 屏幕中下位置显示                         │ │ │
│  │  │    • WKWebView 加载 React UI                  │ │ │
│  │  └─────────────────────────────────────────────┘ │ │
│  │                                                     │ │
│  │  ┌─────────────────────────────────────────────┐ │ │
│  │  │ AI 框架集成 (ai) - 使用 eino 框架           │ │ │
│  │  │  - client.go (统一 AI 接口)                  │ │ │
│  │  │  - factory.go (AI 客户端工厂)                │ │ │
│  │  │  - claude_client.go (Claude 实现)            │ │ │
│  │  │  - zhipu_client.go (智谱 AI 实现)            │ │ │
│  │  │  - prompts.go (提示词模板)                   │ │ │
│  │  └─────────────────────────────────────────────┘ │ │
│  │                                                     │ │
│  │  ┌─────────────────────────────────────────────┐ │ │
│  │  │ 代码注入 (injector) - 新增                    │ │ │
│  │  │  - AppleScript 方式（精准注入）               │ │ │
│  │  │  - 键盘模拟降级方案（兼容性）                  │ │ │
│  │  │  - 剪贴板管理（避免干扰）                     │ │ │
│  │  └─────────────────────────────────────────────┘ │ │
│  └───────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### 数据流

#### 1. 面板唤起流程

```
用户按下 Cmd+Shift+M
    ↓
HotkeyManager.Register() 触发回调
    ↓ (传入 EventContext)
PanelManager.onHotkeyTriggered()
    ↓ 调用
ContextProvider.GetContext()
    ├─ GetFrontmostApp()     → "VS Code"
    ├─ GetBundleID()         → "com.microsoft.VSCode"
    ├─ GetFocusedWindowTitle() → "main.go - flowmind"
    └─ GetSelectedText()     → "useEffect(() => {...})"
    ↓
PanelManager.showPanelWithContext()
    ├─ 创建/显示 NSPanel (屏幕中下位置)
    ├─ WKWebView 加载 React UI
    └─ 通过 bridge 注入上下文
    ↓ (runtime.EventsEmit)
Frontend: Panel.tsx 接收上下文
    └─ 显示 AI 面板，附带应用信息
```

#### 2. AI 对话流程

```
Frontend: 用户输入问题
    ↓ (Wails 方法调用)
App: PanelManager.AskAI()
    ├─ 组装上下文 (应用 + 选中文本)
    └─ 调用 AI 业务逻辑
    ↓
Domain AI (internal/domain/ai/)
    ├─ 提示词模板管理
    ├─ 对话历史管理
    └─ 调用 Infrastructure AI Client
    ↓
Infrastructure AI Client (internal/infrastructure/ai/)
    ├─ 使用 eino 框架
    ├─ 工厂模式选择 AI 提供商 (Claude/智谱/自定义)
    ├─ 发送请求 + 流式响应
    └─ 返回流
    ↓ (runtime.EventsEmit 流式事件)
Frontend: MessageList.tsx
    ├─ 实时渲染 AI 响应
    └─ 用户看到流式文本
```

#### 3. 代码注入流程

```
Frontend: 用户选择建议 [1] 生成清理函数
    ↓ (Wails 方法调用)
App: PanelManager.InjectContent()
    ↓ 调用
Injector.Inject(generatedCode)
    ├─ 尝试 AppleScript 注入 (精准)
    │   └─ 成功 → 返回 nil
    └─ 失败 → 降级到键盘模拟
    ↓
内容插入到当前应用光标位置
    ↓
PanelManager.hidePanel()
    └─ 面板消失
```

### 模块依赖关系

```
internal/app/
├── panel.go              # PanelManager (App 层协调者)
│   ├─→ domain.HotkeyManager      (快捷键)
│   ├─→ platform.ContextProvider  (上下文)
│   ├─→ NSPanel                   (macOS 面板)
│   ├─→ domain.AIManager          (AI 业务逻辑)
│   │   └─→ infrastructure.AIClient (AI 客户端)
│   └─→ injector.Injector         (代码注入)
│
└── methods.go            # 导出给前端的方法

internal/domain/
├── monitor/
│   └── hotkey.go          # HotkeyManager ✅ 已实现
│
└── ai/                    # AI 业务逻辑 (待实现)
    ├── manager.go         # AIManager (业务封装)
    ├── prompt.go          # 提示词模板管理
    └── conversation.go    # 对话历史管理

internal/infrastructure/
├── ai/                    # AI 框架集成 (使用 eino)
│   ├── client.go          # 统一 AI 接口
│   ├── factory.go         # AI 客户端工厂 ✅ 已实现
│   ├── claude_client.go   # Claude 实现 ✅ 已实现
│   ├── zhipu_client.go    # 智谱 AI 实现 ✅ 已实现
│   ├── prompts.go         # 提示词工具 ✅ 已实现
│   └── prompts_test.go    # 提示词测试 ✅ 已实现
│
├── platform/
│   ├── context.go         # ContextProvider 接口
│   ├── context_darwin.go  # macOS 实现
│   ├── context_stub.go    # 跨平台存根
│   └── panel_darwin.m     # NSPanel 封装 (新增)
│
└── injector/
    ├── injector.go        # 注入器接口
    ├── apple_script.go    # AppleScript 方式
    └── keyboard.go        # 键盘模拟方式

frontend/src/
├── components/Panel/
│   ├── Panel.tsx          # 主面板组件
│   ├── MessageList.tsx    # 消息列表
│   └── SuggestionBar.tsx  # 建议栏
│
└── lib/
    └── bridge.ts          # Wails 通信桥接
```

---

## 🚀 实施步骤

### Step 1: 注册全局快捷键 (0.5 天)

**基于现有代码**:
- ✅ `HotkeyManager` 已实现 (`internal/domain/monitor/hotkey.go`)
- ✅ `PermissionChecker` 已实现 (`internal/infrastructure/platform/permission.go`)

**任务清单**:

> 💡 **提示**：下面的代码示例仅供参考，实际实现时需要：
> - 检查现有代码是否已经实现了这些功能
> - 根据实际需求调整代码结构
> - 确保符合项目的代码规范和架构设计

- [x] 快捷键管理器已实现
- [x] 权限检查器已实现
- [ ] 检查辅助功能权限
- [ ] 注册 AI 助手面板快捷键（`Cmd+Shift+M`）
- [ ] 实现快捷键回调函数
- [ ] 测试快捷键触发

**实现代码**:

```go
// internal/app/panel.go
package app

import (
    "github.com/chenyang-zz/flowmind/internal/domain/monitor"
    "github.com/chenyang-zz/flowmind/pkg/events"
    "github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
    "github.com/chenyang-zz/flowmind/internal/infrastructure/platform"
    "go.uber.org/zap"
)

type PanelManager struct {
    hotkeyManager  *monitor.HotkeyManager
    permissionChecker platform.PermissionChecker  // 新增：权限检查器
    panel         *NSPanel // TODO: Step 4 实现
}

func NewPanelManager(hotkeyManager *monitor.HotkeyManager) *PanelManager {
    pm := &PanelManager{
        hotkeyManager:  hotkeyManager,
        permissionChecker: platform.NewPermissionChecker(),  // 初始化权限检查器
    }

    // 检查权限
    if !pm.checkPermissions() {
        logger.Warn("辅助功能权限未授予，部分功能可能无法正常工作")
    }

    // 注册全局快捷键
    pm.registerHotkeys()

    return pm
}

// checkPermissions 检查系统权限
// Returns: bool - 是否已授予所有必需权限
func (pm *PanelManager) checkPermissions() bool {
    // 检查辅助功能权限（快捷键、上下文获取都需要）
    status := pm.permissionChecker.CheckPermission(platform.PermissionAccessibility)

    if status == platform.PermissionStatusDenied {
        logger.Error("缺少辅助功能权限",
            zap.String("permission", "accessibility"),
        )

        // 尝试请求权限
        err := pm.permissionChecker.RequestPermission(platform.PermissionAccessibility)
        if err != nil {
            logger.Error("请求辅助功能权限失败", zap.Error(err))

            // 打开系统设置引导用户手动授权
            _ = pm.permissionChecker.OpenSystemSettings(platform.PermissionAccessibility)
        }

        return false
    }

    logger.Info("辅助功能权限检查通过")
    return true
}

// registerHotkeys 注册 AI 助手面板快捷键
func (pm *PanelManager) registerHotkeys() {
    // 注册 Cmd+Shift+M 唤起面板
    _, err := pm.hotkeyManager.Register("Cmd+Shift+M", pm.onHotkeyTriggered)
    if err != nil {
        logger.Error("注册快捷键失败",
            zap.String("hotkey", "Cmd+Shift+M"),
            zap.Error(err),
        )
        return
    }

    logger.Info("AI 助手面板快捷键注册成功",
        zap.String("hotkey", "Cmd+Shift+M"),
    )
}

// onHotkeyTriggered 快捷键触发回调
func (pm *PanelManager) onHotkeyTriggered(reg *monitor.HotkeyRegistration, ctx *events.EventContext) {
    logger.Info("AI 助手面板快捷键被触发",
        zap.String("hotkey", reg.Hotkey.String()),
        zap.String("current_app", ctx.Application),
        zap.String("current_window", ctx.WindowTitle),
    )

    // TODO: Step 4 - 显示 NSPanel
    // pm.panel.ShowWithContext(ctx)

    // 临时：打印触发信息
    logger.Info("TODO: 显示 AI 面板",
        zap.String("application", ctx.Application),
        zap.String("window", ctx.WindowTitle),
    )
}
```

**验证标准**:
- [ ] 应用启动时自动检查辅助功能权限
- [ ] 权限未授予时能正确请求权限
- [ ] 能打开系统设置引导用户手动授权
- [ ] 按 `Cmd+Shift+M` 能触发回调
- [ ] 回调中能获取当前应用上下文
- [ ] 日志输出正确

---

### Step 2: 上下文感知 (0.5 天)

**基于现有代码**: 项目已实现 `ContextProvider` (`internal/infrastructure/platform/context_darwin.go`)

> 💡 **提示**：下面的代码示例仅供参考，实际实现时需要：
> - 检查 `ContextProvider` 的现有实现
> - 评估是否真的需要添加 `GetSelectedText()` 方法
> - 考虑 macOS Accessibility API 的限制和兼容性问题
> - 确保权限检查和错误处理完善

**任务清单**:
- [x] 上下文提供者已实现
- [x] 支持获取应用名称、Bundle ID、窗口标题
- [ ] **添加 GetSelectedText() 方法** - 获取用户选中的文本
- [ ] 集成到面板管理器
- [ ] 测试上下文获取准确性

**现有功能**:

```go
// internal/infrastructure/platform/context.go
type ContextProvider interface {
    GetFrontmostApp() string      // "VS Code"
    GetBundleID() string            // "com.microsoft.VSCode"
    GetFocusedWindowTitle() string  // "main.go - flowmind"
    GetContext() *events.EventContext
}

// pkg/events/event.go - EventContext 已包含 Selection 字段
type EventContext struct {
    Application  string `json:"application,omitempty"`
    BundleID     string `json:"bundle_id,omitempty"`
    WindowTitle  string `json:"window_title,omitempty"`
    FilePath     string `json:"file_path,omitempty"`
    Selection    string `json:"selection,omitempty"`  // ✅ 已存在
}

// 使用示例
contextMgr := platform.NewContextProvider()
ctx := contextMgr.GetContext()

fmt.Println("应用:", ctx.Application)      // "VS Code"
fmt.Println("Bundle ID:", ctx.BundleID)     // "com.microsoft.VSCode"
fmt.Println("窗口:", ctx.WindowTitle)       // "main.go - flowmind"
```

**需要添加的功能 - GetSelectedText()**:

按照系统架构规范，需要修改以下文件：

#### 1. 更新接口 (`internal/infrastructure/platform/context.go`)

```go
// ContextProvider 接口添加新方法
type ContextProvider interface {
    GetFrontmostApp() string
    GetBundleID() string
    GetFocusedWindowTitle() string

    // GetSelectedText 获取用户当前选中的文本
    // 使用 macOS Accessibility API 获取焦点 UI 元素的选中文本
    // 注意：需要辅助功能权限，且某些应用可能不支持
    // Returns: 当前选中的文本内容，如无选中或获取失败则返回空字符串
    GetSelectedText() string

    GetContext() *events.EventContext
}
```

#### 2. macOS 实现 (`internal/infrastructure/platform/context_darwin.go`)

```go
/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices

#import <Cocoa/Cocoa.h>
#import <ApplicationServices/ApplicationServices.h>

// getSelectedText 获取当前选中的文本
// 使用 Accessibility API 获取焦点 UI 元素的选中文本属性
// Returns: 新分配的 C 字符串，调用者需要使用 free() 释放
char* getSelectedText() {
    // 获取最前端应用
    NSRunningApplication *app = [NSWorkspace sharedWorkspace].frontmostApplication;
    if (app == nil) {
        return strdup("");
    }

    // 创建应用的 AXUIElement
    AXUIElementRef appElement = AXUIElementCreateApplication([app processIdentifier]);
    if (appElement == nil) {
        return strdup("");
    }

    // 获取焦点 UI 元素
    AXUIElementRef focusedElement = NULL;
    AXError err = AXUIElementCopyAttributeValue(appElement,
                                                 kAXFocusedUIElementAttribute,
                                                 (CFTypeRef*)&focusedElement);
    if (err != kAXErrorSuccess || focusedElement == NULL) {
        CFRelease(appElement);
        return strdup("");
    }

    // 获取选中文本
    CFStringRef selectedText = NULL;
    err = AXUIElementCopyAttributeValue(focusedElement,
                                         kAXSelectedTextAttribute,
                                         (CFTypeRef*)&selectedText);

    if (err != kAXErrorSuccess || selectedText == NULL) {
        if (focusedElement != NULL) {
            CFRelease(focusedElement);
        }
        CFRelease(appElement);
        return strdup("");
    }

    // 转换为 C 字符串
    NSString *nsText = (__bridge NSString*)selectedText;
    const char* cText = [nsText UTF8String];
    char* result = strdup(cText);

    // 清理资源
    if (focusedElement != NULL) {
        CFRelease(focusedElement);
    }
    CFRelease(appElement);

    return result;
}
*/
import "C"
import "unsafe"

// GetSelectedText 获取选中文本
func (dm *DarwinContextManager) GetSelectedText() string {
    cStr := C.getSelectedText()
    defer C.free(unsafe.Pointer(cStr))
    return C.GoString(cStr)
}

// GetContext 更新实现，包含选中文本
func (dm *DarwinContextManager) GetContext() *events.EventContext {
    return &events.EventContext{
        Application:  dm.GetFrontmostApp(),
        BundleID:     dm.GetBundleID(),
        WindowTitle:  dm.GetFocusedWindowTitle(),
        Selection:    dm.GetSelectedText(),  // 新增
    }
}
```

#### 3. 存根实现 (`internal/infrastructure/platform/context_stub.go`)

```go
// GetSelectedText 获取选中文本（非 macOS 实现）
func (sm *StubContextManager) GetSelectedText() string {
    return ""
}

// GetContext 更新实现
func (sm *StubContextManager) GetContext() *events.EventContext {
    return &events.EventContext{
        Application:  "",
        BundleID:     "",
        WindowTitle:  "",
        Selection:    "",  // 新增
    }
}
```

**集成到面板**:

```go
// internal/app/panel.go

import "github.com/chenyang-zz/flowmind/internal/infrastructure/platform"

type PanelManager struct {
    hotkeyManager *monitor.HotkeyManager
    contextMgr    platform.ContextProvider
    panel         *NSPanel
}

func NewPanelManager(hotkeyManager *monitor.HotkeyManager) *PanelManager {
    pm := &PanelManager{
        hotkeyManager: hotkeyManager,
        contextMgr:    platform.NewContextProvider(),
    }
    pm.registerHotkeys()
    return pm
}

// getCurrentContext 获取当前应用上下文
func (pm *PanelManager) getCurrentContext() *events.EventContext {
    ctx := pm.contextMgr.GetContext()

    logger.Debug("获取当前上下文",
        zap.String("application", ctx.Application),
        zap.String("bundle_id", ctx.BundleID),
        zap.String("window_title", ctx.WindowTitle),
        zap.String("selection", ctx.Selection),  // 新增日志
    )

    return ctx
}

// 示例：将上下文传递给前端
func (pm *PanelManager) showPanelWithContext() {
    ctx := pm.getCurrentContext()

    // 将上下文注入到面板
    // 包含：应用名、窗口标题、选中文本
    runtime.EventsEmit(pm.ctx, "panel:show", map[string]interface{}{
        "application":  ctx.Application,
        "window_title": ctx.WindowTitle,
        "selection":    ctx.Selection,  // 用户选中的文本
    })
}
```

**注意事项**:
1. ✅ EventContext.Selection 字段已存在，无需修改
2. 需要**辅助功能权限**才能获取选中文本（使用 `PermissionChecker` 检查）
3. 某些应用可能不支持 Accessibility API，会返回空字符串
4. 获取选中文本可能有延迟，建议异步处理
5. 遵循系统架构规范，代码放在 `internal/infrastructure/platform/` 目录

**权限检查**:

```go
// 在调用 GetSelectedText() 前检查权限
func (pm *PanelManager) getCurrentContext() *events.EventContext {
    ctx := pm.contextMgr.GetContext()

    // 如果选中文本为空，可能是因为权限不足
    if ctx.Selection == "" {
        status := pm.permissionChecker.CheckPermission(platform.PermissionAccessibility)
        if status != platform.PermissionStatusGranted {
            logger.Warn("获取选中文本失败：缺少辅助功能权限")
        }
    }

    return ctx
}
```

**验证标准**:
- [ ] 能准确获取当前应用名称
- [ ] 能获取应用 Bundle ID
- [ ] 能获取窗口标题
- [ ] 能获取用户选中的文本（在支持的应用中）
- [ ] 性能：调用耗时 < 100ms
- [ ] 不支持的应用能优雅降级（返回空字符串）

---

### Step 3: 代码注入功能 (2 天)

**基于现有代码**: 剪贴板监控已实现 (`internal/domain/monitor/clipboard.go`)

> 💡 **提示**：下面的代码示例仅供参考，实际实现时需要：
> - 评估代码注入的最佳实现方式（AppleScript vs 键盘模拟）
> - 考虑不同应用的兼容性问题
> - 确保不会干扰用户的剪贴板内容
> - 测试多种应用场景（VS Code、Terminal、浏览器等）

**任务清单**:
- [ ] 实现 AppleScript 注入方式
- [ ] 实现键盘模拟降级方案
- [ ] 集成剪贴板管理（避免干扰）
- [ ] 测试多种应用

**文件结构**:
```
internal/app/injector/
├── injector.go                 # 代码注入器
├── apple_script.go             # AppleScript 方式
└── keyboard.go                 # 键盘模拟方式
```

**核心实现**:

```go
// internal/app/injector/injector.go
package injector

import (
    "os/exec"
    "strings"
    "github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
    "go.uber.org/zap"
)

type Injector struct {
    // 剪贴板管理器引用（用于保存/恢复）
    // clipboard monitor.ClipboardMonitor
}

func NewInjector() *Injector {
    return &Injector{}
}

// Inject 注入内容到当前应用
func (inj *Injector) Inject(content string) error {
    // 1. 尝试 AppleScript 方式（精确）
    err := inj.injectViaAppleScript(content)
    if err == nil {
        logger.Info("内容注入成功（AppleScript）")
        return nil
    }

    logger.Warn("AppleScript 注入失败，尝试键盘模拟", zap.Error(err))

    // 2. 降级到键盘模拟
    err = inj.injectViaKeyboard(content)
    if err != nil {
        logger.Error("键盘模拟注入失败", zap.Error(err))
        return err
    }

    logger.Info("内容注入成功（键盘模拟）")
    return nil
}

// escapeForAppleScript 转义 AppleScript 特殊字符
func escapeForAppleScript(s string) string {
    replacements := map[string]string{
        "\\": "\\\\",
        "\"": "\\\"",
        "'": "\\'",
        "\n": "\\n",
        "\r": "\\r",
        "\t": "\\t",
    }

    result := s
    for old, new := range replacements {
        result = strings.ReplaceAll(result, old, new)
    }
    return result
}
```

---

**AppleScript 实现**:

```go
// internal/app/injector/apple_script.go
package injector

import (
    "fmt"
    "os/exec"
)

// injectViaAppleScript 使用 AppleScript 注入内容
func (inj *Injector) injectViaAppleScript(content string) error {
    // 转义内容
    escaped := escapeForAppleScript(content)

    // 构造 AppleScript
    script := fmt.Sprintf(`
        tell application "System Events"
            keystroke "%s"
        end tell
    `, escaped)

    // 执行 AppleScript
    cmd := exec.Command("osascript", "-e", script)
    output, err := cmd.CombinedOutput()

    if err != nil {
        return fmt.Errorf("AppleScript 执行失败: %w, output: %s", err, string(output))
    }

    return nil
}
```

---

**键盘模拟实现** (使用 robotgo 或类似库):

```go
// internal/app/injector/keyboard.go
package injector

import (
    "fmt"

    // TODO: 添加键盘模拟库依赖
    // "github.com/go-vgo/robotgo"
)

// injectViaKeyboard 使用键盘模拟注入内容
// 通过剪贴板 + Cmd+V 实现
func (inj *Injector) injectViaKeyboard(content string) error {
    // 1. 将内容写入剪贴板
    // err := clipboard.WriteAll(content)
    // if err != nil {
    //     return fmt.Errorf("写入剪贴板失败: %w", err)
    // }

    // 2. 模拟 Cmd+V 粘贴
    // robotgo.KeyTap("v", "command")

    // TODO: 实现键盘模拟
    return fmt.Errorf("键盘模拟暂未实现")
}
```

**验证标准**:
- [ ] 能在 VS Code 中插入代码
- [ ] 能在 JetBrains IDEs 中插入代码
- [ ] 能在终端中插入命令
- [ ] 不干扰用户剪贴板

---

### Step 4: macOS 原生面板 (3-4 天)

#### Day 1: macOS NSPanel 实现

**设计决策**: 使用 macOS 原生 **NSPanel** 而非 Wails 窗口

> 💡 **提示**：下面的代码示例仅供参考，实际实现时需要：
> - 评估是否真的需要 NSPanel，还是使用 Wails 窗口更简单
> - 考虑 CGO 代码的维护成本和性能影响
> - 检查是否有更现代化的方案（如 Tauri、Electron 等）
> - 确保前端和后端通信的高效性

**优势**:
- ✅ 完全原生的 macOS 外观和行为
- ✅ 系统级毛玻璃效果（`NSVisualEffectView`）
- ✅ 更好的性能和动画流畅度
- ✅ 自动适配暗色/亮色模式
- ✅ 原生阴影和圆角
- ✅ 不会抢夺焦点（设置为非激活面板）

**文件结构**:
```
internal/panel/
├── panel_darwin.go             # macOS 原生实现
├── panel_darwin.m              # Objective-C 实现
└── manager.go                  # 面板管理器
```

---

**核心实现**:

```go
// internal/panel/panel_darwin.go
// +build darwin

package panel

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework WebKit

#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>

// NSPanel 包装器
@interface FlowMindPanel : NSPanel {
    WKWebView *webView;
    NSVisualEffectView *effectView;
}
@property (nonatomic, strong) NSViewController *viewController;
- (instancetype)init;
- (void)loadHTML:(NSString *)html;
- (void)show;
- (void)hide;
- (BOOL)isVisible;
@end

@implementation FlowMindPanel

- (instancetype)init {
    // 获取主屏幕
    NSScreen *screen = [NSScreen mainScreen];
    NSRect screenRect = [screen visibleFrame];

    // 计算面板尺寸和位置（屏幕中下位置，居中）
    CGFloat panelWidth = 700;
    CGFloat panelHeight = 500;
    CGFloat x = (screenRect.size.width - panelWidth) / 2;
    CGFloat y = screenRect.size.height * 0.25;  // 从屏幕底部 25% 位置开始
    NSRect frame = NSMakeRect(x, y, panelWidth, panelHeight);

    // 创建 NSPanel
    self = [super initWithContentRect:frame
                          styleMask:NSWindowStyleMaskTitled |
                                  NSWindowStyleMaskFullSizeContentView
                            backing:NSBackingStoreBuffered
                              defer:NO];

    if (!self) {
        return nil;
    }

    // 配置面板属性
    [self setFloatingPanel:YES];              // 悬浮在其他窗口之上
    [self setLevel:NSFloatingWindowLevel];    // 窗口层级
    [self setHidesOnDeactivate:NO];           // 失去焦点时不隐藏
    [self setWorksWhenModal:YES];             // 模态窗口时也可用
    [self setCollectionBehavior:NSWindowCollectionBehaviorMoveToActiveSpace];
    [self setTitle:@"FlowMind Assistant"];
    [self setTitleVisibility:NO];             // 隐藏标题

    // 移除标题栏
    [self setStyleMask:NSWindowStyleMaskBorderless];

    // 设置圆角和阴影
    [self setOpaque:NO];
    [self setBackgroundColor:[NSColor clearColor]];

    // 创建毛玻璃效果视图
    effectView = [[NSVisualEffectView alloc] initWithFrame:[[self contentView] frame]];
    [effectView setMaterial:NSVisualEffectMaterialMenu];       // 菜单材质
    [effectView setBlendingMode:NSVisualEffectBlendingModeBehindWindow];
    [effectView setState:NSVisualEffectStateActive];
    [self setContentView:effectView];

    // 配置圆角
    [[effectView layer] setCornerRadius:12];
    [[effectView layer] setMasksToBounds:YES];

    // 配置阴影
    [self setHasShadow:YES];
    [self setShadow:[[NSShadow alloc] init]];
    [self.shadow setShadowColor:[NSColor colorWithDeviceWhite:0.0 alpha:0.3]];
    [self.shadow setShadowOffset:NSMakeSize(0, -10)];
    [self.shadow setShadowBlurRadius:30];

    // 创建 WebView（用于渲染前端 UI）
    WKWebViewConfiguration *config = [[WKWebViewConfiguration alloc] init];
    webView = [[WKWebView alloc] initWithFrame:[effectView frame]
                                  configuration:config];
    [webView setTranslatesAutoresizingMaskIntoConstraints:NO];

    // 设置 WebView 透明背景
    [webView setValue:@NO forKey:@"drawsBackground"];

    // 添加到毛玻璃视图
    [effectView addSubview:webView];

    // 布局约束
    NSDictionary *views = NSDictionaryOfVariableBindings(webView);
    [effectView addConstraints:[NSLayoutConstraint constraintsWithVisualFormat:@"H:|[webView]|"
                                                                      options:0
                                                                      metrics:nil
                                                                        views:views]];
    [effectView addConstraints:[NSLayoutConstraint constraintsWithVisualFormat:@"V:|[webView]|"
                                                                      options:0
                                                                      metrics:nil
                                                                        views:views]];

    return self;
}

- (void)loadHTML:(NSString *)html {
    [webView loadHTMLString:html baseURL:[[NSBundle mainBundle] resourceURL]];
}

- (void)show {
    [self makeKeyAndOrderFront:nil];

    // 添加出现动画
    [self setAlphaValue:0.0];
    [NSAnimationContext runAnimationGroup:^(NSAnimationContext *context) {
        [context setDuration:0.2];
        [self.animator setAlphaValue:1.0];
    } completionHandler:^{
        // 动画完成
    }];
}

- (void)hide {
    // 添加消失动画
    [NSAnimationContext runAnimationGroup:^(NSAnimationContext *context) {
        [context setDuration:0.15];
        [self.animator setAlphaValue:0.0];
    } completionHandler:^{
        [self orderOut:nil];
        [self setAlphaValue:1.0]; // 重置透明度
    }];
}

- (BOOL)isVisible {
    return [self isVisible];
}

@end
*/
import "C"
import (
    "unsafe"
)

// Panel macOS 原生面板
type Panel struct {
    panel unsafe.Pointer // *C.FlowMindPanel
}

// NewPanel 创建新的 macOS 原生面板
func NewPanel() (*Panel, error) {
    // 调用 Objective-C 初始化
    panel := C.FlowMindPanel_alloc()
    panel = C.FlowMindPanel_init(panel)

    if panel == nil {
        return nil, fmt.Errorf("failed to create NSPanel")
    }

    return &Panel{panel: panel}, nil
}

// Show 显示面板（带动画）
func (p *Panel) Show() error {
    C.FlowMindPanel_show(p.panel)
    return nil
}

// Hide 隐藏面板（带动画）
func (p *Panel) Hide() error {
    C.FlowMindPanel_hide(p.panel)
    return nil
}

// IsVisible 检查面板是否可见
func (p *Panel) IsVisible() bool {
    return C.FlowMindPanel_isVisible(p.panel) != 0
}

// LoadHTML 加载 HTML 内容到 WebView
func (p *Panel) LoadHTML(html string) error {
    cHTML := C.CString(html)
    defer C.free(unsafe.Pointer(cHTML))

    cNSString := C.CStringWithUTF8String(cHTML)
    C.FlowMindPanel_loadHTML(p.panel, cNSString)

    return nil
}

// Toggle 切换显示/隐藏
func (p *Panel) Toggle() error {
    if p.IsVisible() {
        return p.Hide()
    }
    return p.Show()
}
```

---

**面板管理器**:

```go
// internal/panel/manager.go
package panel

import (
    "embed"
    "io/fs"
    "sync"
)

var (
    //go:embed frontend/dist
    frontendFS embed.FS
)

type Manager struct {
    panel     *Panel
    isVisible bool
    mu        sync.RWMutex
}

func NewManager() (*Manager, error) {
    panel, err := NewPanel()
    if err != nil {
        return nil, err
    }

    mgr := &Manager{
        panel: panel,
    }

    // 加载前端 HTML
    if err := mgr.loadFrontend(); err != nil {
        return nil, err
    }

    return mgr, nil
}

// loadFrontend 加载前端资源
func (m *Manager) loadFrontend() error {
    // 从嵌入的文件系统读取 index.html
    distFS, err := fs.Sub(frontendFS, "frontend/dist")
    if err != nil {
        return err
    }

    indexHTML, err := fs.ReadFile(distFS, "index.html")
    if err != nil {
        return err
    }

    // 加载到面板
    return m.panel.LoadHTML(string(indexHTML))
}

// Show 显示面板
func (m *Manager) Show() error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if err := m.panel.Show(); err != nil {
        return err
    }

    m.isVisible = true
    return nil
}

// Hide 隐藏面板
func (m *Manager) Hide() error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if err := m.panel.Hide(); err != nil {
        return err
    }

    m.isVisible = false
    return nil
}

// Toggle 切换显示/隐藏
func (m *Manager) Toggle() error {
    m.mu.RLock()
    visible := m.isVisible
    m.mu.RUnlock()

    if visible {
        return m.Hide()
    }
    return m.Show()
}

// IsVisible 检查面板是否可见
func (m *Manager) IsVisible() bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.isVisible
}
```

---

**样式定制**:

```go
// NSVisualEffectMaterial 材质选项：
// - NSVisualEffectMaterialMenu          // 菜单（推荐）
// - NSVisualEffectMaterialSidebar      // 侧边栏
// - NSVisualEffectMaterialHeaderView   // 标题栏
// - NSVisualEffectMaterialPopover      // 气泡
// - NSVisualEffectMaterialModalWindow  // 模态窗口

// 根据系统外观自动调整
func (p *Panel) adaptToSystemAppearance() {
    // 检查当前系统外观（暗色/亮色）
    // NSAppearance.name == NSAppearanceNameDarkAqua
}
```

**验证标准**:
- [ ] 面板使用系统原生毛玻璃效果
- [ ] 出现/消失动画流畅（60fps）
- [ ] 面板显示在屏幕中下位置（居中）
- [ ] 不会抢夺焦点（Floating Panel）
- [ ] 自动适配系统外观变化

---

#### Day 2-3: 前端 UI 实现

> 💡 **提示**：下面的代码示例仅供参考，实际实现时需要：
> - 选择合适的前端框架（React 19 vs Vue vs Svelte）
> - 评估是否真的需要复杂的前端框架，还是使用原生 HTML/JS 更简单
> - 考虑与 Wails 的集成方式
> - 确保代码的可维护性和性能

**架构说明**:

```
NSPanel (macOS 原生窗口)
  ↓
WKWebView (渲染引擎)
  ↓
HTML/CSS/JavaScript (前端内容)
```

**关键变化**:
- ❌ 不再需要实现背景、毛玻璃、圆角、阴影（由 NSPanel 处理）
- ✅ 只需关注内容布局和交互
- ✅ 通过 `WKScriptMessageHandler` 与 Go 通信

**文件结构**:
```
frontend/
├── src/
│   ├── components/
│   │   ├── Panel.vue               # 面板容器（简化版）
│   │   ├── MessageList.vue         # 消息列表
│   │   ├── TypingIndicator.vue     # 输入指示器
│   │   ├── SuggestionButtons.vue   # 建议按钮
│   │   └── CodeBlock.vue           # 代码块
│   ├── styles/
│   │   └── panel.scss              # 内容样式（无需容器样式）
│   ├── bridge.ts                   # WKWebView 通信桥接
│   └── main.ts                     # 入口
└── index.html                      # 纯 HTML 模板
```

**核心组件**: `Panel.tsx` (React 19 + Tailwind 4)

```tsx
// frontend/src/components/Panel.tsx
import { useState, useEffect, useRef } from 'react'
import { bridge } from '../lib/bridge'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  typing?: boolean
}

interface Context {
  application: string
  windowTitle: string
  appIcon: string
}

export function Panel() {
  const [messages, setMessages] = useState<Message[]>([])
  const [userInput, setUserInput] = useState('')
  const [showSuggestions, setShowSuggestions] = useState(true)
  const [suggestions, setSuggestions] = useState<string[]>([])
  const [context, setContext] = useState<Context>({
    application: '',
    windowTitle: '',
    appIcon: ''
  })

  const messagesContainerRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  // 组件挂载时注册消息处理器
  useEffect(() => {
    // 监听来自 Go 的消息
    const unsubscribeContext = bridge.on('context', (ctx: Context) => {
      setContext(ctx)
      generateSuggestions(ctx)
    })

    const unsubscribeChunk = bridge.on('ai:chunk', (chunk: string) => {
      handleAIChunk(chunk)
    })

    const unsubscribeComplete = bridge.on('ai:complete', () => {
      setMessages(prev => {
        const lastMsg = prev[prev.length - 1]
        if (lastMsg?.role === 'assistant') {
          return [
            ...prev.slice(0, -1),
            { ...lastMsg, typing: false }
          ]
        }
        return prev
      })
    })

    // 请求初始上下文
    bridge.send('getContext')

    return () => {
      unsubscribeContext()
      unsubscribeChunk()
      unsubscribeComplete()
    }
  }, [])

  // 生成上下文感知的建议
  const generateSuggestions = (ctx: Context) => {
    let newSuggestions: string[] = []

    if (ctx.application === 'VS Code') {
      if (ctx.selection) {
        newSuggestions = ['解释代码', '优化性能', '添加注释', '查找 bug']
      } else {
        newSuggestions = ['生成模板', '最佳实践', '搜索文档']
      }
    } else if (ctx.application === 'Terminal') {
      newSuggestions = ['解释命令', '生成命令', '查看历史']
    } else {
      newSuggestions = ['总结任务', '提供帮助']
    }

    setSuggestions(newSuggestions)
  }

  // 发送消息
  const sendMessage = () => {
    const text = userInput.trim()
    if (!text) return

    // 添加用户消息
    setMessages(prev => [
      ...prev,
      {
        id: Date.now().toString(),
        role: 'user',
        content: text
      }
    ])

    setUserInput('')
    setShowSuggestions(false)

    // 发送到 Go 后端
    bridge.send('sendMessage', { message: text })

    // 滚动到底部
    scrollToBottom()
  }

  // 处理 AI 流式响应
  const handleAIChunk = (chunk: string) => {
    setMessages(prev => {
      const lastMsg = prev[prev.length - 1]

      if (lastMsg?.role === 'assistant' && lastMsg.typing) {
        // 追加到现有消息
        return [
          ...prev.slice(0, -1),
          { ...lastMsg, content: lastMsg.content + chunk }
        ]
      } else {
        // 创建新消息
        return [
          ...prev,
          {
            id: Date.now().toString(),
            role: 'assistant',
            content: chunk,
            typing: true
          }
        ]
      }
    })

    scrollToBottom()
  }

  // 选择建议
  const selectSuggestion = (index: number) => {
    setUserInput(suggestions[index])
    textareaRef.current?.focus()
    setShowSuggestions(false)
  }

  // 关闭面板
  const closePanel = () => {
    bridge.send('closePanel')
  }

  // 渲染 Markdown
  const renderMarkdown = (content: string) => {
    const html = marked(content) as string
    return DOMPurify.sanitize(html)
  }

  // 滚动到底部
  const scrollToBottom = () => {
    setTimeout(() => {
      if (messagesContainerRef.current) {
        messagesContainerRef.current.scrollTop =
          messagesContainerRef.current.scrollHeight
      }
    }, 0)
  }

  // 键盘事件处理
  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey && !e.metaKey) {
      e.preventDefault()
      sendMessage()
    } else if (e.key === 'Escape') {
      closePanel()
    } else if (e.key === 'Enter' && e.metaKey) {
      e.preventDefault()
      sendMessage()
    }
  }

  return (
    <div className="flex flex-col h-[500px]">
      {/* 头部：显示上下文信息 */}
      <div className="flex items-center gap-2.5 px-4 py-3 border-b border-white/10">
        <div className="w-7 h-7 rounded-md flex-shrink-0">
          <img
            src={context.appIcon}
            alt={context.application}
            className="w-full h-full rounded-md"
          />
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-sm font-semibold text-white">
            {context.application}
          </div>
          <div className="text-xs text-white/60 truncate">
            {context.windowTitle}
          </div>
        </div>
        <button
          onClick={closePanel}
          className="w-6 h-6 flex items-center justify-center rounded text-white/60 hover:text-white hover:bg-white/10 transition-colors"
        >
          <svg width="14" height="14" viewBox="0 0 14 14" fill="currentColor">
            <path d="M14 1.4L12.6 0 7 5.6 1.4 0 0 1.4 5.6 7 0 12.6 1.4 14 7 8.4 12.6 14 14 12.6 8.4 7z"/>
          </svg>
        </button>
      </div>

      {/* 消息列表 */}
      <div
        ref={messagesContainerRef}
        className="flex-1 overflow-y-auto px-4 py-4 space-y-3"
      >
        {messages.map((msg) => (
          <div
            key={msg.id}
            className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
          >
            <div
              className={`max-w-[80%] px-3.5 py-2.5 text-sm leading-relaxed break-words ${
                msg.role === 'user'
                  ? 'bg-[#667eea] text-white rounded-xl rounded-tr-none'
                  : 'bg-white/8 rounded-xl rounded-tl-none'
              }`}
            >
              <div
                dangerouslySetInnerHTML={{
                  __html: renderMarkdown(msg.content)
                }}
                className="[&_pre]:bg-black/30 [&_pre]:rounded-lg [&_pre]:p-2.5 [&_pre]:my-2 [&_pre]:overflow-x-auto [&_pre]:text-[13px] [&_code]:font-mono"
              />
            </div>
          </div>
        ))}
      </div>

      {/* 快捷建议 */}
      {showSuggestions && (
        <div className="flex gap-2 px-4 pb-3 flex-wrap">
          {suggestions.map((suggestion, index) => (
            <button
              key={index}
              onClick={() => selectSuggestion(index)}
              className="flex items-center gap-2 px-3 py-2 bg-white/6 border border-white/10 rounded-lg text-white text-sm hover:bg-white/10 transition-colors"
            >
              <span className="min-w-[18px] h-[18px] flex items-center justify-center bg-white/15 rounded text-[11px] font-semibold">
                {index + 1}
              </span>
              {suggestion}
            </button>
          ))}
        </div>
      )}

      {/* 输入框 */}
      <div className="flex items-end gap-2 px-4 py-3 border-t border-white/10">
        <textarea
          ref={textareaRef}
          value={userInput}
          onChange={(e) => setUserInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="问我任何问题... (Enter 发送, Esc 关闭)"
          rows={1}
          className="flex-1 bg-white/6 border border-white/10 rounded-lg px-3 py-2.5 text-white text-sm leading-tight resize-none outline-none focus:border-[#667eea] placeholder:text-white/40 transition-colors"
        />
        <button
          onClick={sendMessage}
          disabled={!userInput.trim()}
          className="w-8 h-8 flex items-center justify-center bg-[#667eea] rounded-lg text-white disabled:opacity-40 disabled:cursor-not-allowed hover:scale-105 active:scale-95 transition-all"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
            <path d="M15.85 2.15L9 15l-2.5-3.5L2 9l13.85-6.85z"/>
          </svg>
        </button>
      </div>
    </div>
  )
}
```

---

**WKWebView 通信桥接**: `lib/bridge.ts`

```typescript
// frontend/src/lib/bridge.ts
import { useRef } from 'react'

type MessageHandler = (data: any) => void
type Unsubscribe = () => void

class WKWebViewBridge {
  private handlers: Map<string, Set<MessageHandler>> = new Map()
  private messageQueue: any[] = []

  constructor() {
    // 检测是否在 WKWebView 中运行
    if (this.isWKWebView()) {
      this.setupMessageHandler()
    }
  }

  // 检测是否在 WKWebView 中
  private isWKWebView(): boolean {
    return typeof (window as any).webkit !== 'undefined' &&
           typeof (window as any).webkit.messageHandlers !== 'undefined'
  }

  // 设置消息处理器
  private setupMessageHandler() {
    // 全局消息处理函数
    ;(window as any).flowmindHandleMessage = (data: any) => {
      this.handleMessage(data)
    }
  }

  // 处理收到的消息
  private handleMessage(data: any) {
    const { type, payload } = data

    const handlers = this.handlers.get(type)
    if (handlers) {
      handlers.forEach(handler => {
        try {
          handler(payload)
        } catch (error) {
          console.error(`Error in handler for ${type}:`, error)
        }
      })
    }
  }

  // 发送消息到 Go
  send(type: string, payload?: any) {
    const message = { type, payload }

    if (this.isWKWebView()) {
      const webkit = (window as any).webkit
      if (webkit?.messageHandlers?.flowmind) {
        webkit.messageHandlers.flowmind.postMessage(message)
      }
    } else {
      // 开发环境：保存到队列（用于调试）
      this.messageQueue.push(message)
      console.log('[Bridge] Sent:', message)
    }
  }

  // 注册消息监听器（返回取消订阅函数）
  on(type: string, handler: MessageHandler): Unsubscribe {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, new Set())
    }
    this.handlers.get(type)!.add(handler)

    // 返回取消订阅函数
    return () => {
      this.off(type, handler)
    }
  }

  // 移除监听器
  off(type: string, handler: MessageHandler) {
    const handlers = this.handlers.get(type)
    if (handlers) {
      handlers.delete(handler)
    }
  }

  // 一次性监听器
  once(type: string, handler: MessageHandler): Unsubscribe {
    const wrappedHandler: MessageHandler = (data) => {
      handler(data)
      this.off(type, wrappedHandler)
    }
    return this.on(type, wrappedHandler)
  }

  // 获取消息队列（用于调试）
  getMessageQueue() {
    return this.messageQueue
  }
}

// 创建全局单例
export const bridge = new WKWebViewBridge()

// React Hook：使用桥接
export function useBridge() {
  return bridge
}

// React Hook：监听消息
export function useBridgeMessage(type: string, handler: MessageHandler, deps: any[] = []) {
  const handlerRef = useRef(handler)

  // 保持 handler 引用最新
  handlerRef.current = handler

  // 使用 useEffect 注册监听器
  React.useEffect(() => {
    const unsubscribe = bridge.on(type, (data) => {
      handlerRef.current(data)
    })

    return unsubscribe
  }, [type, ...deps])
}

// 开发环境：暴露到 window（用于调试）
if (import.meta.env.DEV) {
  ;(window as any).bridge = bridge
  ;(window as any).getMessageQueue = () => bridge.getMessageQueue()
}
```

---

**Go 端配置 WKWebView 消息处理**:

```objc
// 在 panel_darwin.m 的 FlowMindPanel init 方法中添加

// 配置消息处理器
WKWebViewConfiguration *config = [[WKWebViewConfiguration alloc] init];
[config.userContentController addScriptMessageHandler:self name:@"flowmind"];

// 实现消息处理器协议
@interface FlowMindPanel () <WKScriptMessageHandler>
@end

@implementation FlowMindPanel

// 接收来自 JavaScript 的消息
- (void)userContentController:(WKUserContentController *)userContentController
      didReceiveScriptMessage:(WKScriptMessage *)message {
    if ([message.name isEqualToString:@"flowmind"]) {
        NSDictionary *data = (NSDictionary *)message.body;
        NSString *type = data[@"type"];
        id payload = data[@"payload"];

        // 调用 Go 处理函数
        handleWebMessage(type, payload);  // CGO 导出的 Go 函数
    }
}

@end

// 发送消息到 JavaScript
void sendToJavaScript(NSString *type, id payload) {
    NSDictionary *data = @{
        @"type": type,
        @"payload": payload ?: [NSNull null]
    };

    NSData *jsonData = [NSJSONSerialization dataWithJSONObject:data
                                                       options:0
                                                         error:nil];
    NSString *jsonString = [[NSString alloc] initWithData:jsonData
                                                 encoding:NSUTF8StringEncoding];

    NSString *script = [NSString stringWithFormat:
        @"window.webkit.messageHandlers.flowmind.postMessage(%@)",
        jsonString
    ];

    [webView evaluateJavaScript:script completionHandler:nil];
}
```

<style scoped lang="scss">
.ai-panel {
  position: fixed;
  bottom: 25%;  // 屏幕中下位置
  left: 50%;
  transform: translateX(-50%) translateY(100px);
  width: 800px;
  max-height: 70vh;
  background: rgba(30, 30, 30, 0.95);
  backdrop-filter: blur(20px);
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  opacity: 0;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  pointer-events: none;
  &.panel-visible {
    transform: translateX(-50%) translateY(0);
    opacity: 1;
    pointer-events: auto;
  }
}

.panel-header {
  display: flex;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);

  .app-icon {
    width: 32px;
    height: 32px;
    border-radius: 8px;
    margin-right: 12px;

    img {
      width: 100%;
      height: 100%;
      border-radius: 8px;
    }
  }

  .app-info {
    flex: 1;

    .app-name {
      font-weight: 600;
      font-size: 14px;
      color: #fff;
    }
    
    .window-title {
      font-size: 12px;
      color: rgba(255, 255, 255, 0.6);
      margin-top: 2px;
    }
  }

  .close-btn {
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.2s;

    &:hover {
      background: rgba(255, 255, 255, 0.1);
    }
  }
}

.messages {
  max-height: 400px;
  overflow-y: auto;
  padding: 20px;

  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.2);
    border-radius: 3px;
  }
}

.suggestions {
  display: flex;
  gap: 8px;
  padding: 0 20px 16px;
}

.input-area {
  display: flex;
  align-items: flex-end;
  padding: 16px 20px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);

  textarea {
    flex: 1;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    padding: 12px;
    color: #fff;
    font-size: 14px;
    resize: none;
    outline: none;
    max-height: 120px;

    &:focus {
      border-color: rgba(255, 255, 255, 0.3);
    }
  }

  .send-btn {
    width: 36px;
    height: 36px;
    margin-left: 12px;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    border: none;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: transform 0.2s;

    &:hover {
      transform: scale(1.05);
    }
    
    &:active {
      transform: scale(0.95);
    }
  }
}
</style>
```

---

#### Day 4: 动画和交互优化

**任务清单**:
- [ ] 实现面板滑入/滑出动画
- [ ] 实现消息淡入效果
- [ ] 实现打字机效果（AI 响应）
- [ ] 实现代码块高亮

**示例：打字机效果**

```typescript
// utils/typewriter.ts
export function useTypewriter(text: Ref<string>, speed = 20) {
  const displayedText = ref('')
  let currentIndex = 0

  watch(text, (newText) => {
    displayedText.value = ''
    currentIndex = 0

    const interval = setInterval(() => {
      if (currentIndex < newText.length) {
        displayedText.value += newText[currentIndex]
        currentIndex++
      } else {
        clearInterval(interval)
      }
    }, speed)

    onUnmounted(() => clearInterval(interval))
  })

  return displayedText
}
```

---

### Step 5: 后端集成 (2 天)

> 💡 **提示**：下面的代码示例仅供参考，实际实现时需要：
> - 复用 `internal/infrastructure/ai/` 中已有的 AI 客户端实现
> - 评估是否真的需要 Domain 层的 AIManager，还是直接使用 Infrastructure 层的客户端
> - 确保对话历史和提示词管理的实现方式合理
> - 考虑错误处理和超时机制

#### 文件结构
```
internal/panel/
├── manager.go                  # 面板管理器
├── service.go                  # Wails 服务
└── bridge.go                   # 前后端桥接
```

#### Manager 实现

```go
// internal/panel/manager.go
package panel

import (
    "context"
    "flowmind/internal/ai"
    "flowmind/internal/context"
    "flowmind/internal/hotkey"
    "flowmind/internal/injector"
)

type Manager struct {
    hotkeyMgr  *hotkey.Manager
    aiService  *ai.AIService
    injector   *injector.Injector
    window     *PanelWindow
    ctx        *context.Context

    // 对话历史
    messages   []Message
}

type Message struct {
    ID      string `json:"id"`
    Role    string `json:"role"` // "user" | "assistant"
    Content string `json:"content"`
}

func NewManager(aiService *ai.AIService) (*Manager, error) {
    mgr := &Manager{
        aiService: aiService,
        messages:  make([]Message, 0),
    }

    // 初始化快捷键管理器
    mgr.hotkeyMgr = hotkey.NewManager()

    // 注册全局快捷键
    err := mgr.hotkeyMgr.Register(
        "m",
        []string{"cmd", "shift"},
        mgr.onHotkeyTriggered,
    )
    if err != nil {
        return nil, err
    }

    // 初始化面板窗口
    window, err := NewPanelWindow()
    if err != nil {
        return nil, err
    }
    mgr.window = window

    return mgr, nil
}

// onHotkeyTriggered 快捷键触发回调
func (mgr *Manager) onHotkeyTriggered() {
    // 获取当前上下文
    ctx, err := context.GetContext()
    if err != nil {
        logger.Error("获取上下文失败", zap.Error(err))
        return
    }
    mgr.ctx = ctx

    // 显示面板
    err = mgr.window.Show()
    if err != nil {
        logger.Error("显示面板失败", zap.Error(err))
    }
}

// SendMessage 发送消息（前端调用）
func (mgr *Manager) SendMessage(userMessage string) error {
    // 添加用户消息
    mgr.messages = append(mgr.messages, Message{
        ID:      generateID(),
        Role:    "user",
        Content: userMessage,
    })

    // 构建上下文感知的提示词
    prompt := mgr.buildPrompt(userMessage)

    // 流式响应
    err := mgr.aiService.Stream(prompt, func(chunk string) error {
        // 将 chunk 发送到前端
        EventsEmit("ai:chunk", chunk)
        return nil
    })

    if err != nil {
        return err
    }

    return nil
}

// buildPrompt 构建上下文感知的提示词
func (mgr *Manager) buildPrompt(userMessage string) string {
    ctx := mgr.ctx

    prompt := fmt.Sprintf(`当前上下文:
- 应用: %s (%s)
- 窗口: %s
- 文件: %s
- 选中文本: %s

用户问题: %s

请根据当前上下文回答。如果需要生成代码或文本，仅输出内容，不要解释。`,
        ctx.Application,
        ctx.BundleID,
        ctx.WindowTitle,
        ctx.FilePath,
        ctx.Selection,
        userMessage,
    )

    return prompt
}

// InjectContent 注入内容到当前应用
func (mgr *Manager) InjectContent(content string) error {
    // 保存当前剪贴板
    err := mgr.injector.SaveClipboard()
    if err != nil {
        return err
    }

    // 注入内容
    err = mgr.injector.Inject(content)
    if err != nil {
        return err
    }

    // 恢复剪贴板
    return mgr.injector.RestoreClipboard()
}

// Start 启动管理器
func (mgr *Manager) Start() error {
    return mgr.hotkeyMgr.Start()
}

// Stop 停止管理器
func (mgr *Manager) Stop() error {
    return mgr.hotkeyMgr.Stop()
}
```

---

### Step 6-8: 测试、部署、优化

详见以下章节。

---

## ✅ 验收标准

### 功能验证

- [ ] **快捷键**: 按 `Cmd+Shift+M` 能唤起面板
- [ ] **上下文**: AI 知道当前应用、文件、选中文本
- [ ] **对话**: 能与 AI 进行流畅对话
- [ ] **流式响应**: AI 响应实时显示，非阻塞
- [ ] **代码注入**: 能将 AI 生成的内容插入到当前应用
- [ ] **快捷建议**: 根据上下文提供智能建议

### 性能验证

- [ ] 面板唤起延迟 < 200ms
- [ ] AI 响应首字节时间 < 1s
- [ ] 内存占用 < 100MB
- [ ] CPU 使用 < 5% (空闲时)

### 用户体验验证

- [ ] 面板动画流畅（60fps）
- [ ] 快捷键不与其他应用冲突
- [ ] 代码插入不干扰用户剪贴板
- [ ] 支持键盘快捷键（数字键选择建议、Esc 关闭）

### 质量验证

- [ ] 单元测试覆盖率 ≥ 80%
- [ ] 核心模块覆盖率 ≥ 90%
- [ ] 所有中文注释完整
- [ ] 文档完整

---

## 📊 成功指标

1. **响应速度**: 面板唤起 + AI 响应首字 < 1.5s
2. **准确性**: 上下文获取准确率 > 95%
3. **可用性**: 代码注入成功率 > 90%（支持的主流应用）
4. **稳定性**: 连续使用 1 天无崩溃

---

## 🔧 关键技术点

### 0. AI 框架选型 - 为什么选择 eino？

**eino** 是一个 Go 语言开发的 LLM 应用开发框架，由 IDL (字节跳动) 开源，专为生产环境设计。

**核心优势**：
- ✅ **统一接口**：提供统一的 LLM 调用接口，轻松切换不同 AI 提供商
- ✅ **流式支持**：原生支持流式响应，性能优异
- ✅ **生产就绪**：字节跳动内部生产验证，稳定可靠
- ✅ **社区活跃**：持续维护，文档完善
- ✅ **类型安全**：充分利用 Go 的类型系统，减少运行时错误

**项目中的实现**：

```
internal/infrastructure/ai/  # AI 框架集成层
├── client.go               # 基于 eino 的统一 AI 接口
├── factory.go              # 工厂模式创建 AI 客户端
├── claude_client.go        # Claude API 实现
├── zhipu_client.go         # 智谱 AI 实现
├── prompts.go              # 提示词模板管理
└── prompts_test.go         # 单元测试
```

**使用示例**：

```go
// 1. 创建 AI 客户端（工厂模式）
import "flowmind/internal/infrastructure/ai"

client, err := ai.NewClient(ai.ProviderClaude, ai.Config{
    APIKey: "sk-ant-xxx",
    Model:  "claude-3-5-sonnet-20241022",
})
if err != nil {
    return err
}

// 2. 发送流式请求
ctx := context.Background()
messages := []ai.Message{
    {Role: "user", Content: "你好"},
}

err = client.Stream(ctx, messages, func(chunk string) error {
    // 处理流式响应
    fmt.Println(chunk)
    return nil
})
```

**已有的实现**：
- ✅ `client.go` - 统一的 AI 接口定义
- ✅ `factory.go` - AI 客户端工厂，支持动态切换提供商
- ✅ `claude_client.go` - Claude API 完整实现（包含流式响应）
- ✅ `zhipu_client.go` - 智谱 AI 完整实现
- ✅ `prompts.go` - 提示词模板管理工具
- ✅ 完整的单元测试（覆盖率 > 80%）

**分层设计**：

```
┌─────────────────────────────────────┐
│  Domain Layer (AI 业务逻辑)          │
│  - AIManager                         │
│  - 提示词模板管理                     │
│  - 对话历史管理                       │
└──────────────┬──────────────────────┘
               │ 调用
               ▼
┌─────────────────────────────────────┐
│  Infrastructure Layer (AI 框架)      │
│  - 使用 eino 框架                    │
│  - 统一 AI 接口                      │
│  - 多提供商支持 (Claude/智谱)        │
└─────────────────────────────────────┘
```

**依赖添加**：

```bash
# go.mod（已添加）
require (
    github.com/cloudwego/eino v0.7.28
    github.com/cloudwego/eino-ext/components/model/claude v0.1.15
)
```

**参考资料**：
- [eino 官方文档](https://github.com/cloudwego/eino)
- [Claude API 文档](https://docs.anthropic.com/)
- [智谱 AI API 文档](https://open.bigmodel.cn/)

---

### 0.1 模型灵活切换方案

**设计目标**：支持在运行时灵活切换不同的 AI 模型和提供商，无需重启应用。

#### 支持的切换方式

**1. 配置文件切换**（静态，需要重启）

修改 `configs/default.yaml` 或 `~/.flowmind/config.yaml`：

```yaml
ai:
  # 默认使用的提供商
  default_provider: "zhipu"  # 可选: claude, zhipu, openai, ollama

  # 是否启用自动回退（主模型失败时自动切换备用模型）
  auto_fallback: true

  # 模型池配置
  models:
    # 启用的模型列表
    enabled:
      - "claude-3-5-sonnet"
      - "claude-3-haiku"
      - "glm-4"
      - "llama3.2"

    # 模型优先级（从高到低，用于自动回退）
    priority:
      - "glm-4"
      - "claude-3-5-sonnet"
      - "claude-3-haiku"
      - "llama3.2"

    # 模型用途标签（用于智能选择）
    usage_tags:
      "glm-4": ["chinese", "translation", "default", "code"]
      "claude-3-5-sonnet": ["code", "analysis"]
      "claude-3-haiku": ["chat", "quick"]
      "llama3.2": ["local", "privacy"]

  # Claude 配置
  claude:
    enabled: true
    api_key: "${CLAUDE_API_KEY}"
    model: "claude-3-5-sonnet-20241022"
    max_tokens: 4096
    temperature: 0.7
    base_url: ""  # 可选：自定义 API 端点
    timeout: 30

  # 智谱 AI 配置
  zhipu:
    enabled: true
    api_key: "${ZHIPU_API_KEY}"
    model: "glm-4"
    max_tokens: 4096
    temperature: 0.7
    timeout: 30

  # OpenAI 配置
  openai:
    enabled: false  # 暂未实现
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4o"
    max_tokens: 4096
    temperature: 0.7
    base_url: ""
    timeout: 30

  # Ollama 配置
  ollama:
    enabled: false  # 暂未实现
    base_url: "http://localhost:11434"
    model: "llama3.2"
    max_tokens: 4096
    temperature: 0.7
    timeout: 30
```

**2. 环境变量切换**（静态，需要重启）

```bash
# 设置默认提供商为智谱 AI
export AI_PROVIDER=zhipu

# 设置智谱 API Key
export ZHIPU_API_KEY=your_zhipu_key

# 也可以设置其他提供商的 API Key（作为备用）
export CLAUDE_API_KEY=sk-ant-xxx

# 启动应用
./flowmind
```

**3. 运行时动态切换**（动态，无需重启）

```go
import "github.com/chenyang-zz/flowmind/internal/infrastructure/ai"

// 方式 1：直接切换提供商（例如切换到 Claude）
newModel, err := ai.SwitchProvider("claude", "your_api_key", "claude-3-5-sonnet-20241022")
if err != nil {
    return err
}

// 方式 2：使用配置切换
config := &ai.AIConfig{
    Provider: "zhipu",  // 使用智谱
    APIKey:   "your_zhipu_key",
    Model:    "glm-4",
}
newModel, err := ai.NewAIModel(config)
```

#### 模型管理器设计（建议实现）

```go
// internal/infrastructure/ai/manager.go

// ModelManager 模型管理器
type ModelManager struct {
    // 当前活跃的模型
    currentModel AIModel

    // 模型池（所有已加载的模型）
    models map[string]AIModel

    // 配置
    config *config.AIConfig

    // 使用统计
    stats map[string]*ModelStats
}

// ModelStats 模型使用统计
type ModelStats struct {
    // 总调用次数
    TotalCalls int64

    // 成功次数
    SuccessCalls int64

    // 失败次数
    FailureCalls int64

    // 平均响应时间（毫秒）
    AvgLatencyMs int64

    // 总 Token 消耗
    TotalTokens int64

    // 最后使用时间
    LastUsedAt time.Time
}

// SwitchModel 切换到指定模型
func (m *ModelManager) SwitchModel(provider string) error {
    // 1. 检查模型是否在模型池中
    if _, exists := m.models[provider]; !exists {
        // 创建新模型
        model, err := m.createModel(provider)
        if err != nil {
            return fmt.Errorf("创建模型失败: %w", err)
        }
        m.models[provider] = model
    }

    // 2. 切换当前模型
    m.currentModel = m.models[provider]

    // 3. 记录切换事件
    logger.Info("模型已切换",
        zap.String("provider", provider),
        zap.String("model", m.currentModel.GetType().String()))

    return nil
}

// GetCurrentModel 获取当前模型
func (m *ModelManager) GetCurrentModel() AIModel {
    return m.currentModel
}

// GetModelStats 获取模型使用统计
func (m *ModelManager) GetModelStats(provider string) (*ModelStats, error) {
    stats, exists := m.stats[provider]
    if !exists {
        return nil, fmt.Errorf("模型不存在: %s", provider)
    }
    return stats, nil
}

// GetAllStats 获取所有模型的统计信息
func (m *ModelManager) GetAllStats() map[string]*ModelStats {
    return m.stats
}

// AutoSelectBestModel 根据统计自动选择最佳模型
func (m *ModelManager) AutoSelectBestModel() string {
    var bestProvider string
    var bestScore float64

    for provider, stats := range m.stats {
        // 计算综合评分
        // 评分 = 成功率权重 + 速度权重 - 成本权重
        successRate := float64(stats.SuccessCalls) / float64(stats.TotalCalls)
        score := successRate*0.6 + (1.0/float64(stats.AvgLatencyMs))*0.4

        if score > bestScore {
            bestScore = score
            bestProvider = provider
        }
    }

    return bestProvider
}
```

#### 智能回退机制

```go
// ChatWithFallback 智能回退的对话实现
func (m *ModelManager) ChatWithFallback(ctx context.Context, messages []Message) (*ChatResponse, error) {
    // 尝试使用当前模型
    response, err := m.currentModel.Chat(ctx, messages)
    if err == nil {
        return response, nil
    }

    // 记录失败
    logger.Warn("主模型调用失败，尝试回退",
        zap.String("provider", m.currentModel.GetType().String()),
        zap.Error(err))

    // 根据优先级尝试备用模型
    for _, provider := range m.config.Models.Priority {
        // 跳过当前失败的模型
        if provider == m.currentModel.GetType().String() {
            continue
        }

        // 检查模型是否启用
        if !m.isModelEnabled(provider) {
            continue
        }

        // 尝试使用备用模型
        backupModel, exists := m.models[provider]
        if !exists {
            // 创建备用模型
            var err error
            backupModel, err = m.createModel(provider)
            if err != nil {
                logger.Error("创建备用模型失败",
                    zap.String("provider", provider),
                    zap.Error(err))
                continue
            }
            m.models[provider] = backupModel
        }

        // 尝试调用
        response, err := backupModel.Chat(ctx, messages)
        if err == nil {
            logger.Info("回退成功",
                zap.String("backup_provider", provider))

            // 切换到备用模型
            m.currentModel = backupModel
            return response, nil
        }

        logger.Warn("备用模型也失败了",
            zap.String("backup_provider", provider),
            zap.Error(err))
    }

    return nil, fmt.Errorf("所有模型都失败了")
}
```

#### 前端集成示例

```typescript
// frontend/src/lib/ai.ts

// 可用的模型列表（按优先级排序）
export const AVAILABLE_MODELS = [
  {
    id: 'glm-4',
    name: '智谱 GLM-4',
    provider: 'zhipu',
    tags: ['chinese', 'translation', 'default', 'code'],
    icon: '🇨🇳',
    isDefault: true
  },
  {
    id: 'claude-3-5-sonnet',
    name: 'Claude 3.5 Sonnet',
    provider: 'claude',
    tags: ['code', 'analysis', 'complex'],
    icon: '🧠'
  },
  {
    id: 'claude-3-haiku',
    name: 'Claude 3 Haiku',
    provider: 'claude',
    tags: ['chat', 'quick', 'summary'],
    icon: '⚡'
  },
  {
    id: 'llama3.2',
    name: 'Llama 3.2',
    provider: 'ollama',
    tags: ['local', 'privacy', 'offline'],
    icon: '🦙'
  }
]

// 切换模型
export async function switchModel(modelId: string) {
  await window.api.switchAIModel(modelId)
}

// 获取当前模型
export async function getCurrentModel(): Promise<string> {
  return await window.api.getCurrentAIModel()
}

// 获取模型统计
export async function getModelStats(): Promise<ModelStats[]> {
  return await window.api.getModelStats()
}
```

#### 配置示例：完整的模型切换配置

```yaml
# ~/.flowmind/config.yaml

ai:
  default_provider: "zhipu"  # 默认使用智谱 GLM-4
  auto_fallback: true

  models:
    enabled:
      - "glm-4"                # 默认模型，中文优化
      - "claude-3-5-sonnet"    # 复杂代码任务
      - "claude-3-haiku"       # 快速对话
      - "llama3.2"             # 本地隐私任务

    priority:
      - "glm-4"
      - "claude-3-5-sonnet"
      - "claude-3-haiku"
      - "llama3.2"

    usage_tags:
      "glm-4": ["chinese", "translation", "default", "code", "writing"]
      "claude-3-5-sonnet": ["code", "analysis", "complex"]
      "claude-3-haiku": ["chat", "quick", "summary"]
      "llama3.2": ["local", "privacy", "offline"]

  claude:
    enabled: true
    api_key: "${CLAUDE_API_KEY}"
    model: "claude-3-5-sonnet-20241022"
    max_tokens: 4096
    temperature: 0.7
    timeout: 30

  zhipu:
    enabled: true
    api_key: "${ZHIPU_API_KEY}"
    model: "glm-4"
    max_tokens: 4096
    temperature: 0.7
    timeout: 30
```

#### 使用场景示例

**场景 1：根据任务类型自动选择模型**

```go
func (m *ModelManager) SelectModelForTask(taskType string) AIModel {
    // 根据配置的 usage_tags 选择最合适的模型
    for provider, tags := range m.config.Models.UsageTags {
        for _, tag := range tags {
            if tag == taskType {
                return m.models[provider]
            }
        }
    }

    // 默认返回第一个启用的模型
    return m.currentModel
}

// 使用示例
model := manager.SelectModelForTask("chinese")
response, err := model.Chat(ctx, messages)
```

**场景 2：成本优化（优先使用本地模型）**

```go
func (m *ModelManager) GetCostEffectiveModel() AIModel {
    // 优先级：Ollama（免费）> Claude Haiku（便宜）> Claude Sonnet（贵）
    preferredOrder := []string{"ollama", "claude-3-haiku", "claude-3-5-sonnet"}

    for _, provider := range preferredOrder {
        if model, exists := m.models[provider]; exists {
            return model
        }
    }

    return m.currentModel
}
```

**场景 3：速度优化（快速任务使用轻量模型）**

```go
func (m *ModelManager) GetFastModel() AIModel {
    // 快速任务使用 Haiku 或 Llama 3.2
    fastModels := []string{"claude-3-haiku", "llama3.2"}

    for _, provider := range fastModels {
        if model, exists := m.models[provider]; exists {
            return model
        }
    }

    return m.currentModel
}
```

---

### 1. macOS 权限配置

**基于现有代码**: 项目已实现完整的权限管理系统 (`internal/infrastructure/platform/permission.go`)

**Info.plist 配置**:
```xml
<key>NSAccessibilityUsageDescription</key>
<string>FlowMind 需要辅助功能权限来获取应用上下文和插入内容</string>

<key>NSAppleEventsUsageDescription</key>
<string>需要控制其他应用来实现代码注入</string>
```

**权限检查** (使用已有的 PermissionChecker):

```go
// internal/app/panel.go
import "github.com/chenyang-zz/flowmind/internal/infrastructure/platform"

type PanelManager struct {
    permissionChecker platform.PermissionChecker
    // ... 其他字段
}

func (pm *PanelManager) checkAndRequestPermissions() error {
    // 检查辅助功能权限
    status := pm.permissionChecker.CheckPermission(platform.PermissionAccessibility)

    if status != platform.PermissionStatusGranted {
        logger.Warn("辅助功能权限未授予")

        // 1. 尝试请求权限（会弹出系统对话框）
        err := pm.permissionChecker.RequestPermission(platform.PermissionAccessibility)
        if err != nil {
            logger.Error("请求权限失败", zap.Error(err))

            // 2. 如果失败，打开系统设置引导用户手动授权
            err = pm.permissionChecker.OpenSystemSettings(platform.PermissionAccessibility)
            if err != nil {
                return fmt.Errorf("打开系统设置失败: %w", err)
            }

            return fmt.Errorf("请在系统设置中手动授予权限")
        }
    }

    logger.Info("权限检查通过")
    return nil
}
```

**权限状态说明**:

| 状态 | 说明 | 处理方式 |
|------|------|----------|
| `PermissionStatusGranted` | 权限已授予 | 正常使用功能 |
| `PermissionStatusDenied` | 权限被拒绝 | 请求权限 → 打开系统设置 |
| `PermissionStatusUnknown` | 权限状态未知 | 重新检查权限 |

**支持的权限类型**:

```go
// 已实现的权限
PermissionAccessibility  // ✅ 辅助功能权限（快捷键、上下文获取、代码注入）

// 预留的权限（待实现）
PermissionScreenCapture  // 屏幕录制权限（未来功能）
PermissionFiles          // 文件访问权限（未来功能）
```

---

### 2. Wails 前后端通信

**Go → JavaScript**:
```go
// 发送事件到前端
runtime.EventsEmit(ctx, "ai:chunk", chunk)
```

```typescript
// 前端监听
import { EventsOn } from '../../wailsjs/runtime'

EventsOn("ai:chunk", (chunk: string) => {
  onAIStreamChunk(chunk)
})
```

**JavaScript → Go**:
```typescript
import { SendMessage } from '../../wailsjs/go/main/App'

await SendMessage(userInput)
```

---

### 3. 流式响应优化

**后端**:
```go
func (s *AIService) Stream(prompt string, handler StreamHandler) error {
    // 实现 SSE 流
}
```

**前端**:
```typescript
let buffer = ''

EventsOn("ai:chunk", (chunk: string) => {
  buffer += chunk

  // 解码 markdown
  const html = marked(buffer)
  messageElement.innerHTML = html
})
```

---

### 4. 代码注入降级策略

```go
func (inj *Injector) Inject(content string) error {
    // 1. 尝试 AppleScript
    err := injectViaAppleScript(content)
    if err == nil {
        return nil
    }

    logger.Warn("AppleScript 失败，降级到键盘模拟", zap.Error(err))

    // 2. 降级到键盘模拟
    return injectViaKeyboard(content)
}
```

---

## 📖 如何使用本文档

### ⚠️ 重要提醒

本文档是一个**设计指南和思路参考**，**不是实施手册**。在实际开发时，请务必：

#### 1. 独立思考和验证

- ❌ **不要**机械地复制文档中的代码
- ✅ **要**理解设计意图，根据实际情况调整
- ✅ **要**验证文档中的假设是否正确
- ✅ **要**质疑文档中的技术选择

#### 2. 优先检查现有代码

在开始实现之前，先检查：
- `internal/infrastructure/ai/` - AI 客户端已实现
- `internal/domain/monitor/` - 监控功能已实现
- `internal/infrastructure/platform/` - 平台相关代码已实现

**问题**：这些现有代码能否直接使用？是否需要修改？

#### 3. 从最小可行方案开始

- ❌ **不要**一次性实现所有功能
- ✅ **要**先实现核心功能，验证可行性
- ✅ **要**在验证成功后再扩展功能

**示例**：
- 第一步：先实现快捷键唤起空面板
- 第二步：添加上下文获取
- 第三步：集成 AI 对话
- 第四步：添加代码注入

#### 4. 技术选型要谨慎

文档中提到的技术栈仅供参考：

| 技术 | 文档建议 | 实际选择需要考虑 |
|------|---------|----------------|
| NSPanel | macOS 原生面板 | 是否有更简单的方案？CGO 维护成本？ |
| React 19 | 前端框架 | 是否真的需要？原生 JS 是否够用？ |
| eino | AI 框架 | 是否过度设计？直接调用 API 是否更简单？ |
| AppleScript | 代码注入 | 兼容性如何？是否有更好的方案？ |

#### 5. 参考现有代码的实现风格

在编写新代码时，参考：
- 现有代码的目录结构
- 现有代码的命名规范
- 现有代码的错误处理方式
- 现有代码的测试风格

#### 6. 文档与现实的差距

文档中的代码可能存在：
- 🐛 语法错误
- 🐛 逻辑错误
- 🐛 过时的 API 调用
- 🐛 不完整的实现

**记住**：文档是静态的，代码是动态的。文档更新可能滞后于代码变化。

---

### 推荐的实施流程

```
1. 阅读本文档，理解设计意图
   ↓
2. 检查现有代码，评估可复用性
   ↓
3. 思考技术选型，质疑文档假设
   ↓
4. 设计最小可行方案（MVP）
   ↓
5. 实现 MVP，验证核心功能
   ↓
6. 根据验证结果，调整设计
   ↓
7. 逐步扩展功能，完善实现
   ↓
8. 编写测试，确保代码质量
   ↓
9. 更新文档，记录实际实现
```

---

### 文档更新原则

当你发现文档与实际实现不符时：
1. **先实现正确的代码**
2. **再更新文档以反映实际**
3. **在文档中添加注释说明修改原因**

---

## 🔗 相关文档

**前置文档**（上下阶段）:
- [系统架构总览](../architecture/00-system-architecture.md) - 理解整体架构
- [Phase 1: 基础监控](./02-phase1-monitoring.md) - 实现事件监控
- [Phase 2: 模式识别](./03-phase2-patterns.md) - 实现模式挖掘
- [开发环境搭建](./01-development-setup.md) - 配置开发环境

**本阶段详细架构**:
- [AI 框架集成](../architecture/04-ai-service.md) - 使用 eino 框架的统一 AI 接口
- [监控引擎详解](../architecture/02-monitor-engine.md) - 事件监控和上下文获取
- [eino 框架文档](https://github.com/cloudwego/eino) - 官方文档和 API 参考

**后续阶段**（下阶段）:
- [Phase 4: 知识管理](./05-phase4-knowledge.md) - 实现剪藏和知识图谱

---

**最后更新**: 2026-01-31
