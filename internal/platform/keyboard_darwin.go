//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework Cocoa

#include <CoreFoundation/CoreFoundation.h>
#include <CoreGraphics/CoreGraphics.h>
#include <Cocoa/Cocoa.h>

// goKeyboardCallback Go 层的回调函数声明
// 此函数由 C 层调用，将键盘事件传递到 Go 层
// Parameters: keyCode - 按键代码, flags - 修饰键标志
void goKeyboardCallback(int keyCode, int flags);

// callback CGEventTap 回调函数（static 避免符号冲突）
// 这是 Core Graphics Event Tap 的回调函数，当有键盘事件发生时被调用
// Parameters:
//   proxy - 事件 tap 代理
//   type - 事件类型（按下、释放、修饰键变化等）
//   event - 事件对象
//   refcon - 用户数据（未使用）
// Returns: 原始事件对象（允许事件继续传递）
static CGEventRef callback(CGEventTapProxy proxy, CGEventType type,
                   CGEventRef event, void *refcon) {
    // 只处理键盘按下和修饰键变化事件
    if (type == kCGEventKeyDown || type == kCGEventFlagsChanged) {
        // 获取按键代码
        CGKeyCode keycode = (CGKeyCode)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);

        // 获取修饰键标志（Command, Shift, Control, Option 等）
        CGEventFlags flags = CGEventGetFlags(event);

        // 回调到 Go 层处理
        goKeyboardCallback((int)keycode, (int)flags);
    }

    return event;
}

// createEventTap 创建事件 tap 的辅助函数（static 避免符号冲突）
// Event Tap 允许应用程序拦截和处理系统级的事件
// 此函数创建一个键盘事件 tap，并将其添加到当前 run loop
// Returns: Event tap 的 CFMachPortRef 句柄，失败返回 NULL
static void* createEventTap() {
    // 设置事件掩码：监听键盘按下和修饰键变化事件
    CGEventMask eventMask = (1 << kCGEventKeyDown) | (1 << kCGEventFlagsChanged);

    // 创建事件 tap
    // kCGSessionEventTap: 在会话级别监听事件
    // kCGHeadInsertEventTap: 在事件队列头部插入
    // kCGEventTapOptionDefault: 使用默认选项
    CFMachPortRef tap = CGEventTapCreate(
        kCGSessionEventTap,
        kCGHeadInsertEventTap,
        kCGEventTapOptionDefault,
        eventMask,
        callback,
        NULL
    );

    if (tap == NULL) {
        return NULL;
    }

    // 启用 tap
    CGEventTapEnable(tap, true);

    // 创建 run loop source，将 tap 集成到 run loop 中
    CFRunLoopSourceRef src = CFMachPortCreateRunLoopSource(NULL, tap, 0);

    // 添加到当前 thread 的 run loop
    CFRunLoopAddSource(CFRunLoopGetCurrent(), src, kCFRunLoopCommonModes);

    // 释放 source（run loop 会保留它）
    CFRelease(src);

    return tap;
}

// destroyEventTap 销毁事件 tap 的辅助函数（static 避免符号冲突）
// 停用并释放事件 tap 相关的资源
// Parameters: tap - 要销毁的 event tap 句柄
static void destroyEventTap(void* tap) {
    if (tap != NULL) {
        CFMachPortRef eventTap = (CFMachPortRef)tap;
        CGEventTapEnable(eventTap, false);
        CFRelease(eventTap);
    }
}
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"
)

// DarwinKeyboardMonitor macOS 平台的键盘监控器实现
//
// DarwinKeyboardMonitor 使用 Core Graphics Event Tap API 来捕获系统级的键盘事件。
// 需要用户在系统偏好设置中授予辅助功能权限才能正常工作。
type DarwinKeyboardMonitor struct {
	// eventTap C 层的 CFMachPortRef 句柄，用于管理事件 tap
	eventTap unsafe.Pointer
	// callback 用户注册的键盘事件回调函数
	callback KeyboardCallback
	// isRunning 监控器运行状态标志
	isRunning bool
	// mu 读写锁，保护并发访问
	mu sync.RWMutex
	// stopChan 停止信号通道
	stopChan chan struct{}
}

// 全局键盘监控器实例（用于 C 回调）
//
// 由于 C 函数无法直接调用 Go 方法，需要维护一个全局实例引用。
// monitorMutex 用于保护此全局变量的并发访问。
var (
	defaultKeyboardMonitor *DarwinKeyboardMonitor
	monitorMutex           sync.Mutex
)

// NewKeyboardMonitor 创建键盘监控器
//
// 在 macOS 平台上，此函数返回 DarwinKeyboardMonitor 实例。
// Returns: KeyboardMonitor 接口的 macOS 实现
func NewKeyboardMonitor() KeyboardMonitor {
	return &DarwinKeyboardMonitor{
		stopChan: make(chan struct{}),
	}
}

//export goKeyboardCallback
// goKeyboardCallback C 到 Go 的桥接函数
//
// 这是一个导出函数，由 C 层的 CGEventTap 回调调用。
// 它将键盘事件异步转发给当前活跃的监控器实例处理。
// Parameters: keyCode - 按键代码, flags - 修饰键标志
func goKeyboardCallback(keyCode, flags C.int) {
	monitorMutex.Lock()
	if defaultKeyboardMonitor != nil {
		// 异步处理回调，避免阻塞 C 层的 event loop
		go defaultKeyboardMonitor.handleCallback(int(keyCode), uint64(flags))
	}
	monitorMutex.Unlock()
}

// handleCallback 处理键盘事件回调
//
// 此方法在 goroutine 中执行，检查监控器状态后调用用户注册的回调函数。
// Parameters: keyCode - 按键代码, flags - 修饰键标志
func (km *DarwinKeyboardMonitor) handleCallback(keyCode int, flags uint64) {
	km.mu.RLock()
	if !km.isRunning {
		km.mu.RUnlock()
		return
	}
	km.mu.RUnlock()

	// 调用上层注册的回调函数
	if km.callback != nil {
		km.callback(KeyboardEvent{
			KeyCode:   keyCode,
			Modifiers: flags,
		})
	}
}

// Start 启动键盘监控
//
// 创建 CGEventTap 并开始捕获键盘事件。
// 如果创建失败（通常是因为缺少辅助功能权限），返回错误。
// Parameters: callback - 键盘事件回调函数
// Returns: error - 启动失败时返回错误
func (km *DarwinKeyboardMonitor) Start(callback KeyboardCallback) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if km.isRunning {
		return fmt.Errorf("keyboard monitor already running")
	}

	// 保存回调函数
	km.callback = callback

	// 保存到全局实例（用于 C 回调）
	monitorMutex.Lock()
	defaultKeyboardMonitor = km
	monitorMutex.Unlock()

	// 创建事件 tap（调用 C 函数）
	km.eventTap = C.createEventTap()

	if km.eventTap == nil {
		// 创建失败，清理全局实例
		monitorMutex.Lock()
		defaultKeyboardMonitor = nil
		monitorMutex.Unlock()
		return fmt.Errorf("failed to create event tap: please grant accessibility permission")
	}

	// 重新创建停止通道（支持多次启停）
	km.stopChan = make(chan struct{})

	km.isRunning = true

	return nil
}

// Stop 停止键盘监控
//
// 销毁 CGEventTap 并释放相关资源。
// Returns: error - 停止失败时返回错误
func (km *DarwinKeyboardMonitor) Stop() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if !km.isRunning {
		return fmt.Errorf("keyboard monitor not running")
	}

	// 销毁事件 tap（调用 C 函数）
	if km.eventTap != nil {
		C.destroyEventTap(km.eventTap)
		km.eventTap = nil
	}

	// 清理全局实例
	monitorMutex.Lock()
	defaultKeyboardMonitor = nil
	monitorMutex.Unlock()

	km.isRunning = false

	// 安全地关闭停止通道（避免重复关闭）
	select {
	case <-km.stopChan:
		// 通道已关闭
	default:
		close(km.stopChan)
	}

	return nil
}

// IsRunning 检查运行状态
//
// 返回监控器当前的运行状态。
// Returns: bool - 监控器是否正在运行
func (km *DarwinKeyboardMonitor) IsRunning() bool {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.isRunning
}
