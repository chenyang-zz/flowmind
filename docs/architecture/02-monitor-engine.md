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
Monitor Engine
    ├─→ Keyboard Monitor (键盘监听)
    ├─→ Clipboard Monitor (剪贴板监听)
    ├─→ Application Monitor (应用切换监听)
    └─→ FileSystem Monitor (文件系统监听)
         ↓
    Event Bus (事件总线)
         ↓
    ┌─────────────┬─────────────┬─────────────┐
    ↓             ↓             ↓             ↓
 Analyzer       Storage       Frontend      AI Service
```

### 核心接口

```go
// internal/monitor/monitor.go
package monitor

type Monitor interface {
    // Start 启动监控
    Start() error

    // Stop 停止监控
    Stop() error

    // Events 返回事件通道
    Events() <-chan Event

    // IsRunning 检查运行状态
    IsRunning() bool
}

// Event 统一事件结构
type Event struct {
    ID        string                 `json:"id"`
    Type      EventType              `json:"type"`
    Timestamp time.Time              `json:"timestamp"`
    Data      map[string]interface{} `json:"data"`
    Context   *EventContext          `json:"context"`
}

type EventType string

const (
    EventTypeKeyboard   EventType = "keyboard"
    EventTypeClipboard  EventType = "clipboard"
    EventTypeAppSwitch  EventType = "app_switch"
    EventTypeFileSystem EventType = "file_system"
)

// EventContext 事件上下文信息
type EventContext struct {
    Application string    `json:"application"`
    BundleID    string    `json:"bundle_id"`
    WindowTitle string    `json:"window_title"`
    FilePath    string    `json:"file_path,omitempty"`
    Selection   string    `json:"selection,omitempty"`
}
```

---

## 键盘监控

### macOS 实现 (CGO)

```go
// internal/monitor/keyboard_darwin.go
package monitor

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework Cocoa

#include <CoreFoundation/CoreFoundation.h>
#include <CoreGraphics/CoreGraphics.h>
#include <Cocoa/Cocoa.h>

CGEventTapEventMask eventMask = (1 << kCGEventKeyDown) | (1 << kCGEventFlagsChanged);

CGEventTapCallBack callback(CGEventTapProxy proxy, CGEventType type,
                            CGEventRef event, void *refcon) {
    // 获取按键代码
    CGKeyCode keycode = (CGKeyCode)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);

    // 获取修饰键标志
    CGEventFlags flags = CGEventGetFlags(event);

    // 回调到 Go
    goKeyboardCallback(keycode, flags);

    return event;
}

void goKeyboardCallback(int keyCode, int flags);
*/
import "C"
import (
    "unsafe"
)

type KeyboardMonitor struct {
    eventTap C.CGEventTapRef
    eventChan chan<- Event
    stopChan  chan struct{}
}

func NewKeyboardMonitor(eventChan chan<- Event) *KeyboardMonitor {
    return &KeyboardMonitor{
        eventChan: eventChan,
        stopChan:  make(chan struct{}),
    }
}

//export goKeyboardCallback
func goKeyboardCallback(keyCode, flags C.int) {
    // 全局回调处理
    defaultKeyboardMonitor.handleCallback(int(keyCode), uint64(flags))
}

func (km *KeyboardMonitor) handleCallback(keyCode int, flags uint64) {
    // 构造事件
    event := Event{
        ID:        generateEventID(),
        Type:      EventTypeKeyboard,
        Timestamp: time.Now(),
        Data: map[string]interface{}{
            "keycode":   keyCode,
            "modifiers": flags,
        },
        Context: km.getCurrentContext(),
    }

    // 发送到事件总线
    select {
    case km.eventChan <- event:
    default:
        log.Warn("Keyboard event channel full, dropping event")
    }
}

func (km *KeyboardMonitor) Start() error {
    // 创建事件 tap
    km.eventTap = C.CGEventTapCreate(
        C.kCGSessionEventTap,
        C.kCGHeadInsertEventTap,
        C.kCGEventTapOptionDefault,
        C.eventMask,
        C.CGEventTapCallBack(C.callback),
        unsafe.Pointer(nil),
    )

    if km.eventTap == nil {
        return fmt.Errorf("failed to create event tap")
    }

    // 启用 tap
    C.CGEventTapEnable(km.eventTap, 1)

    // 创建 run loop source
    src := C.CFMachPortCreateRunLoopSource(nil, C.CFMachPortRef(km.eventTap), 0)

    // 添加到当前 run loop
    rl := C.CFRunLoopGetCurrent()
    C.CFRunLoopAddSource(rl, src, C.kCFRunLoopCommonModes)

    // 保存引用
    defaultKeyboardMonitor = km

    return nil
}

func (km *KeyboardMonitor) Stop() error {
    if km.eventTap != nil {
        C.CGEventTapDisable(km.eventTap)
        C.CFRelease(C.CFTypeRef(km.eventTap))
        km.eventTap = nil
    }
    close(km.stopChan)
    return nil
}

func (km *KeyboardMonitor) getCurrentContext() *EventContext {
    // 获取当前应用信息
    return &EventContext{
        Application: getCurrentAppName(),
        BundleID:    getCurrentBundleID(),
        WindowTitle: getCurrentWindowTitle(),
    }
}

var defaultKeyboardMonitor *KeyboardMonitor
```

### 快捷键检测

```go
// internal/monitor/hotkey.go
type HotkeyManager struct {
    registeredHotkeys map[string]*Hotkey
    eventChan         chan<- Event
}

type Hotkey struct {
    Key      string // "cmd+shift+m"
    Callback func()
    enabled  bool
}

func (hm *HotkeyManager) RegisterHotkey(key string, callback func()) error {
    hotkey := &Hotkey{
        Key:      key,
        Callback: callback,
        enabled:  true,
    }

    // 解析快捷键
    keyCode, modifiers := parseHotkey(key)

    // 注册系统快捷键
    // 使用 Carbon framework RegisterEventHotKey

    hm.registeredHotkeys[key] = hotkey
    return nil
}

func parseHotkey(key string) (keyCode int, modifiers uint) {
    // "cmd+shift+m" → keyCode=46, modifiers=cmd|shift
    // 实现解析逻辑
    return 0, 0
}
```

---

## 剪贴板监控

### 实现

```go
// internal/monitor/clipboard_darwin.go
package monitor

import (
    "C"
    "unsafe"
)

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
    eventChan     chan<- Event
    stopChan      chan struct{}
    lastChangeCount int
    pollInterval  time.Duration
}

func NewClipboardMonitor(eventChan chan<- Event) *ClipboardMonitor {
    return &ClipboardMonitor{
        eventChan:    eventChan,
        stopChan:     make(chan struct{}),
        pollInterval: 500 * time.Millisecond, // 每 500ms 检查一次
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
                event := Event{
                    ID:        generateEventID(),
                    Type:      EventTypeClipboard,
                    Timestamp: time.Now(),
                    Data: map[string]interface{}{
                        "content":   content,
                        "length":    len(content),
                    },
                    Context: cm.getCurrentContext(),
                }

                cm.eventChan <- event
            }

        case <-cm.stopChan:
            return
        }
    }
}

func (cm *ClipboardMonitor) getCurrentContext() *EventContext {
    return &EventContext{
        Application: getCurrentAppName(),
        BundleID:    getCurrentBundleID(),
        WindowTitle: getCurrentWindowTitle(),
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

### 实现

```go
// internal/monitor/application_darwin.go
package monitor

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#include <Cocoa/Cocoa.h>

NSString* getCurrentApp() {
    NSRunningApplication *app = [NSWorkspace sharedWorkspace].frontmostApplication;
    return [app localizedName];
}

NSString* getBundleID() {
    NSRunningApplication *app = [NSWorkspace sharedWorkspace].frontmostApplication;
    return [app bundleIdentifier];
}

NSString* getCurrentWindow() {
    AXUIElementRef app = AXUIElementCreateApplication(
        [[NSWorkspace sharedWorkspace] frontmostApplication].processID
    );

    AXUIElementRef window = NULL;
    AXUIElementCopyAttributeValue(app, kAXFocusedWindowAttribute, (CFTypeRef*)&window);

    if (window) {
        CFStringRef title = NULL;
        AXUIElementCopyAttributeValue(window, kAXTitleAttribute, (CFTypeRef*)&title);

        if (title) {
            NSString *nsTitle = (__bridge NSString*)title;
            return nsTitle;
        }
    }

    return @"";
}
*/
import "C"

type ApplicationMonitor struct {
    eventChan    chan<- Event
    stopChan     chan struct{}
    currentApp   string
    currentTimer *time.Timer
    pollInterval time.Duration
}

func NewApplicationMonitor(eventChan chan<- Event) *ApplicationMonitor {
    return &ApplicationMonitor{
        eventChan:    eventChan,
        stopChan:     make(chan struct{}),
        pollInterval: 1 * time.Second,
    }
}

func (am *ApplicationMonitor) Start() error {
    am.currentApp = am.getCurrentApp()

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
            newApp := am.getCurrentApp()

            if newApp != am.currentApp {
                // 应用切换事件
                event := Event{
                    ID:        generateEventID(),
                    Type:      EventTypeAppSwitch,
                    Timestamp: time.Now(),
                    Data: map[string]interface{}{
                        "from": am.currentApp,
                        "to":   newApp,
                    },
                    Context: &EventContext{
                        Application: newApp,
                        BundleID:    am.getBundleID(),
                        WindowTitle: am.getCurrentWindow(),
                    },
                }

                am.eventChan <- event

                am.currentApp = newApp
            }

        case <-am.stopChan:
            return
        }
    }
}

func (am *ApplicationMonitor) getCurrentApp() string {
    return C.GoString(C.getCurrentApp())
}

func (am *ApplicationMonitor) getBundleID() string {
    return C.GoString(C.getBundleID())
}

func (am *ApplicationMonitor) getCurrentWindow() string {
    return C.GoString(C.getCurrentWindow())
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
    sessions      map[string]*AppSession
    eventChan     chan<- Event
}

type AppSession struct {
    AppName  string
    BundleID string
    Start    time.Time
    End      time.Time
}

func (at *AppTracker) OnAppSwitch(from, to string) {
    // 结束上一个应用会话
    if session, exists := at.sessions[from]; exists {
        session.End = time.Now()

        // 发送会话事件
        event := Event{
            Type: EventTypeAppSession,
            Data: map[string]interface{}{
                "app":        session.AppName,
                "bundle_id":  session.BundleID,
                "start":      session.Start,
                "end":        session.End,
                "duration":   session.End.Sub(session.Start).Seconds(),
            },
        }

        at.eventChan <- event
    }

    // 开始新应用会话
    at.sessions[to] = &AppSession{
        AppName: to,
        BundleID: getCurrentBundleID(),
        Start:   time.Now(),
    }
}
```

---

## 文件系统监控

### 实现

```go
// internal/monitor/filesystem_darwin.go
package monitor

import "github.com/fsnotify/fsnotify"

type FileSystemMonitor struct {
    eventChan chan<- Event
    stopChan  chan struct{}
    watcher   *fsnotify.Watcher
    watchedPaths map[string]bool
}

func NewFileSystemMonitor(eventChan chan<- Event) *FileSystemMonitor {
    return &FileSystemMonitor{
        eventChan:    eventChan,
        stopChan:     make(chan struct{}),
        watchedPaths: make(map[string]bool),
    }
}

func (fsm *FileSystemMonitor) Start() error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }

    fsm.watcher = watcher

    // 监控用户目录
    homeDir, _ := os.UserHomeDir()
    fsm.watchPath(homeDir + "/Downloads")
    fsm.watchPath(homeDir + "/Documents")
    fsm.watchPath(homeDir + "/Desktop")

    go fsm.processEvents()

    return nil
}

func (fsm *FileSystemMonitor) watchPath(path string) error {
    if err := fsm.watcher.Add(path); err != nil {
        return err
    }

    fsm.watchedPaths[path] = true
    return nil
}

func (fsm *FileSystemMonitor) processEvents() {
    for {
        select {
        case event, ok := <-fsm.watcher.Events:
            if !ok {
                return
            }

            // 构造文件系统事件
            fsEvent := Event{
                ID:        generateEventID(),
                Type:      EventTypeFileSystem,
                Timestamp: time.Now(),
                Data: map[string]interface{}{
                    "path":     event.Name,
                    "op":       event.Op.String(),
                    "is_create": event.Op&fsnotify.Create == fsnotify.Create,
                    "is_write":  event.Op&fsnotify.Write == fsnotify.Write,
                    "is_remove": event.Op&fsnotify.Remove == fsnotify.Remove,
                    "is_rename": event.Op&fsnotify.Rename == fsnotify.Rename,
                },
            }

            fsm.eventChan <- fsEvent

        case err, ok := <-fsm.watcher.Errors:
            if !ok {
                return
            }
            log.Error("File system watcher error:", err)

        case <-fsm.stopChan:
            return
        }
    }
}

func (fsm *FileSystemMonitor) Stop() error {
    close(fsm.stopChan)
    return fsm.watcher.Close()
}
```

---

## 主监控器

### 整合所有监控器

```go
// internal/monitor/engine.go
package monitor

type Engine struct {
    keyboard   *KeyboardMonitor
    clipboard  *ClipboardMonitor
    app        *ApplicationMonitor
    filesystem *FileSystemMonitor
    eventChan  chan Event
    stopChan   chan struct{}
}

func NewEngine() *Engine {
    eventChan := make(chan Event, 1000)

    return &Engine{
        keyboard:   NewKeyboardMonitor(eventChan),
        clipboard:  NewClipboardMonitor(eventChan),
        app:        NewApplicationMonitor(eventChan),
        filesystem: NewFileSystemMonitor(eventChan),
        eventChan:  eventChan,
        stopChan:   make(chan struct{}),
    }
}

func (e *Engine) Start() error {
    log.Info("Starting monitor engine")

    // 启动所有监控器
    if err := e.keyboard.Start(); err != nil {
        return fmt.Errorf("keyboard monitor: %w", err)
    }

    if err := e.clipboard.Start(); err != nil {
        return fmt.Errorf("clipboard monitor: %w", err)
    }

    if err := e.app.Start(); err != nil {
        return fmt.Errorf("application monitor: %w", err)
    }

    if err := e.filesystem.Start(); err != nil {
        return fmt.Errorf("file system monitor: %w", err)
    }

    log.Info("Monitor engine started")

    return nil
}

func (e *Engine) Stop() error {
    log.Info("Stopping monitor engine")

    e.keyboard.Stop()
    e.clipboard.Stop()
    e.app.Stop()
    e.filesystem.Stop()

    close(e.stopChan)
    close(e.eventChan)

    log.Info("Monitor engine stopped")

    return nil
}

func (e *Engine) Events() <-chan Event {
    return e.eventChan
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

func (ef *EventFilter) ShouldPass(event Event) bool {
    key := event.Type + event.Context.Application

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
    events    []Event
    batchSize int
    timeout   time.Duration
    output    chan<- []Event
    timer     *time.Timer
}

func (eb *EventBatcher) Add(event Event) {
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
    eb.events = make([]Event, 0, eb.batchSize)
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
// internal/monitor/monitor_test.go
func TestKeyboardMonitor(t *testing.T) {
    eventChan := make(chan Event, 100)
    monitor := NewKeyboardMonitor(eventChan)

    err := monitor.Start()
    assert.NoError(t, err)

    defer monitor.Stop()

    // 模拟按键事件
    // ...

    select {
    case event := <-eventChan:
        assert.Equal(t, EventTypeKeyboard, event.Type)
    case <-time.After(5 * time.Second):
        t.Fatal("No event received")
    }
}
```

---

**相关文档**：
- [系统架构](./01-system-architecture.md)
- [分析引擎](./03-analyzer-engine.md)
- [实施指南 Phase 1](../implementation/02-phase1-monitoring.md)
