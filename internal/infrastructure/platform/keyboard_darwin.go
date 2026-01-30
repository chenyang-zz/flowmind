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

// runRunLoopInMode 在当前线程运行 CFRunLoop（带超时）
// 此函数会运行 CFRunLoop 一小段时间（0.1 秒），然后返回
// 通过反复调用此函数，可以在每次循环之间检查停止信号
// Returns: true 表示 RunLoop 处理了事件，false 表示超时
static int runRunLoopInMode() {
    // 运行 RunLoop 0.1 秒
    CFRunLoopRunResult result = CFRunLoopRunInMode(kCFRunLoopDefaultMode, 0.1, false);
    return (result == kCFRunLoopRunHandledSource);
}

// checkRunLoopSource 检查 RunLoop 是否有待处理的事件源
// Returns: true 表示有待处理的事件，false 表示空闲
static int checkRunLoopSource() {
    CFRunLoopRef currentLoop = CFRunLoopGetCurrent();
    CFRunLoopSourceRef src = NULL;
    return (currentLoop != NULL && CFRunLoopContainsSource(currentLoop, src, kCFRunLoopDefaultMode));
}
*/
import "C"
import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"go.uber.org/zap"
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
	// runLoopDone CFRunLoop 退出信号通道，用于等待 RunLoop 线程结束
	runLoopDone chan struct{}
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
		stopChan:    make(chan struct{}),
		runLoopDone: make(chan struct{}),
	}
}

// goKeyboardCallback C 到 Go 的桥接函数
//
// 这是一个导出函数，由 C 层的 CGEventTap 回调调用。
// 它将键盘事件异步转发给当前活跃的监控器实例处理。
// Parameters: keyCode - 按键代码, flags - 修饰键标志
//
//export goKeyboardCallback
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
// 创建 CGEventTap 并在独立的 goroutine 中运行 CFRunLoop 以捕获键盘事件。
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

	// 重新创建停止通道和 RunLoop 完成通道（支持多次启停）
	km.stopChan = make(chan struct{})
	km.runLoopDone = make(chan struct{})

	// 在独立的 goroutine 中运行 CFRunLoop
	// CFRunLoop 是 macOS 的事件循环机制，负责处理系统事件
	// 使用 CFRunLoopRunInMode 而不是 CFRunLoopRun，以便定期检查停止信号
	go func() {
		logger.Info("CFRunLoop 监控线程启动", zap.String("component", "keyboard"))

		// 使用 runtime.LockOSThread 确保 goroutine 绑定到 OS 线程
		// 这对于 CFRunLoop 的正确运行很重要

		for {
			// 运行 f 一小段时间（0.1 秒）
			// 这允许 RunLoop 处理事件，同时我们可以定期检查停止信号
			C.runRunLoopInMode()

			// 检查是否收到停止信号
			select {
			case <-km.stopChan:
				logger.Info("收到停止信号，退出 CFRunLoop", zap.String("component", "keyboard"))
				close(km.runLoopDone)
				return
			default:
				// 继续运行
			}
		}
	}()

	km.isRunning = true

	return nil
}

// Stop 停止键盘监控
//
// 发送停止信号给 CFRunLoop，销毁 CGEventTap 并释放相关资源。
// Returns: error - 停止失败时返回错误
func (km *DarwinKeyboardMonitor) Stop() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if !km.isRunning {
		return fmt.Errorf("keyboard monitor not running")
	}

	logger.Info("停止键盘监控", zap.String("component", "keyboard"))

	// 发送停止信号（关闭 stopChan 会触发 select case）
	// CFRunLoop 线程会在下一个循环检查到此信号并退出
	select {
	case <-km.stopChan:
		// 通道已关闭（不应该发生，因为我们在 Start 时创建新通道）
	default:
		close(km.stopChan)
	}

	// 等待 CFRunLoop 线程退出（最多等待 2 秒）
	select {
	case <-km.runLoopDone:
		logger.Info("CFRunLoop 线程已正常退出", zap.String("component", "keyboard"))
	case <-time.After(2 * time.Second):
		logger.Warn("等待 CFRunLoop 线程退出超时", zap.String("component", "keyboard"))
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
