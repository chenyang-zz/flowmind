//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#include <Cocoa/Cocoa.h>
#include <stdlib.h>

// 前向声明 Go 函数
extern void goAppSwitchHandleAppSwitch(char* to, char* bundleID, char* window);

// appswitchCallback NSWorkspace 通知回调函数（static 避免符号冲突）
// 这是 NSWorkspace 通知的回调函数，当应用切换时被调用
// Parameters: note - 通知对象，包含应用切换的信息
static void appswitchCallback(NSNotification* note) {
    @autoreleasepool {
        NSRunningApplication* newApp = [note.userInfo objectForKey:NSWorkspaceApplicationKey];
        if (newApp == nil) {
            return;
        }

        NSString* newAppName = [newApp localizedName];
        NSString* newBundleID = [newApp bundleIdentifier];
        NSString* windowTitle = @"";

        // 获取窗口标题
        AXUIElementRef appElement = AXUIElementCreateApplication([newApp processIdentifier]);
        if (appElement != NULL) {
            AXUIElementRef window = NULL;
            AXError err = AXUIElementCopyAttributeValue(appElement,
                                                        kAXFocusedWindowAttribute,
                                                        (CFTypeRef*)&window);
            if (err == kAXErrorSuccess && window != NULL) {
                CFStringRef title = NULL;
                if (AXUIElementCopyAttributeValue(window,
                                                   kAXTitleAttribute,
                                                   (CFTypeRef*)&title) == kAXErrorSuccess && title != NULL) {
                    windowTitle = (__bridge NSString*)title;
                }
                if (window != NULL) {
                    CFRelease(window);
                }
            }
            if (appElement != NULL) {
                CFRelease(appElement);
            }
        }

        // 调用 Go 回调（不需要传递 from，Go 层会维护状态）
        const char* toApp = [newAppName UTF8String];
        const char* bundleID = [newBundleID UTF8String];
        const char* window = [windowTitle UTF8String];

        goAppSwitchHandleAppSwitch(
            (char*)toApp,
            (char*)bundleID,
            (char*)window
        );
    }
}

// startAppSwitchMonitoring 启动应用切换监控（static 避免符号冲突）
// 注册 NSWorkspace 通知监听器，检测应用切换事件
// Returns: 0=成功, -1=失败
static int startAppSwitchMonitoring() {
    @autoreleasepool {
        // 注册通知监听器
        NSNotificationCenter* center = [NSWorkspace sharedWorkspace].notificationCenter;

        id observer = [center addObserverForName:NSWorkspaceDidActivateApplicationNotification
                                          object:nil
                                           queue:[NSOperationQueue mainQueue]
                                      usingBlock:^(NSNotification* note) {
            appswitchCallback(note);
        }];

        return (observer != nil) ? 0 : -1;
    }
}

// stopAppSwitchMonitoring 停止应用切换监控（static 避免符号冲突）
// Returns: 0=成功
static int stopAppSwitchMonitoring() {
    @autoreleasepool {
        // 移除所有通知监听器
        [[[NSWorkspace sharedWorkspace] notificationCenter] removeObserver:[[NSWorkspace sharedWorkspace] notificationCenter]];
        return 0;
    }
}
*/
import "C"
import (
	"fmt"
	"sync"
)

// DarwinAppSwitchMonitor macOS平台的应用切换监控器实现
type DarwinAppSwitchMonitor struct {
	// callback Go回调函数
	callback AppSwitchCallback

	// isRunning 监控器运行状态
	isRunning bool

	// mu 互斥锁，保护并发访问
	mu sync.RWMutex

	// currentAppName 当前应用名称（用于记录切换前的应用）
	currentAppName string
}

// 全局应用切换监控器实例（用于 C 回调）
//
// 由于 C 函数无法直接调用 Go 方法，需要维护一个全局实例引用。
// appSwitchMonitorMutex 用于保护此全局变量的并发访问。
var (
	defaultAppSwitchMonitor *DarwinAppSwitchMonitor
	appSwitchMonitorMutex   sync.Mutex
)

// NewDarwinAppSwitchMonitor 创建macOS平台的应用切换监控器
// Returns: *DarwinAppSwitchMonitor - 新创建的监控器实例
func NewDarwinAppSwitchMonitor() *DarwinAppSwitchMonitor {
	return &DarwinAppSwitchMonitor{}
}

// Start 启动应用切换监控
// 启动后会持续监听应用切换事件并通过回调函数通知
// Parameters: callback - 应用切换事件回调函数
// Returns: error - 启动失败时返回错误（如已运行等）
func (m *DarwinAppSwitchMonitor) Start(callback AppSwitchCallback) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("应用切换监控器已在运行")
	}

	// 保存回调函数
	m.callback = callback

	// 保存到全局实例（用于 C 回调）
	appSwitchMonitorMutex.Lock()
	defaultAppSwitchMonitor = m
	appSwitchMonitorMutex.Unlock()

	// 启动监控
	ret := C.startAppSwitchMonitoring()
	if ret != 0 {
		appSwitchMonitorMutex.Lock()
		defaultAppSwitchMonitor = nil
		appSwitchMonitorMutex.Unlock()
		return fmt.Errorf("启动应用切换监控失败")
	}

	m.isRunning = true
	return nil
}

// Stop 停止应用切换监控
// 停止后会释放系统资源并取消事件监听
// Returns: error - 停止失败时返回错误（如未运行等）
func (m *DarwinAppSwitchMonitor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return fmt.Errorf("应用切换监控器未运行")
	}

	// 停止监控
	C.stopAppSwitchMonitoring()

	// 清理全局实例
	appSwitchMonitorMutex.Lock()
	defaultAppSwitchMonitor = nil
	appSwitchMonitorMutex.Unlock()

	m.callback = nil
	m.isRunning = false

	return nil
}

// IsRunning 检查运行状态
// Returns: bool - 监控器是否正在运行
func (m *DarwinAppSwitchMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// handleAppSwitch 处理应用切换事件
// Parameters:
//   - from: 切换前的应用名称
//   - to: 切换后的应用名称
//   - bundleID: 新应用的 Bundle ID
//   - window: 新应用的窗口标题
func (m *DarwinAppSwitchMonitor) handleAppSwitch(from, to, bundleID, window string) {
	m.mu.RLock()
	if !m.isRunning {
		m.mu.RUnlock()
		return
	}
	m.mu.RUnlock()

	// 调用上层注册的回调函数
	if m.callback != nil {
		event := AppSwitchEvent{
			From:     from,
			To:       to,
			BundleID: bundleID,
			Window:   window,
		}
		m.callback(event)
	}
}

// getCurrentAppName 获取当前应用名称
// Returns: string - 当前应用名称
func (m *DarwinAppSwitchMonitor) getCurrentAppName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentAppName
}

// setCurrentAppName 设置当前应用名称
// Parameters: appName - 新应用名称
func (m *DarwinAppSwitchMonitor) setCurrentAppName(appName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentAppName = appName
}

//export goAppSwitchHandleAppSwitch
// goAppSwitchHandleAppSwitch 应用切换的 Go 回调函数（由 C 调用）
// Parameters:
//   - to: 切换后的应用名称（C 字符串）
//   - bundleID: 新应用的 Bundle ID（C 字符串）
//   - window: 新应用的窗口标题（C 字符串）
func goAppSwitchHandleAppSwitch(to, bundleID, window *C.char) {
	appSwitchMonitorMutex.Lock()
	if defaultAppSwitchMonitor != nil {
		// 获取当前应用名称作为 from
		from := defaultAppSwitchMonitor.getCurrentAppName()

		// 更新当前应用名称
		defaultAppSwitchMonitor.setCurrentAppName(C.GoString(to))

		// 异步处理回调，避免阻塞 C 层
		go defaultAppSwitchMonitor.handleAppSwitch(
			from,
			C.GoString(to),
			C.GoString(bundleID),
			C.GoString(window),
		)
	}
	appSwitchMonitorMutex.Unlock()
}

// NewAppSwitchMonitor 创建macOS平台的应用切换监控器
// Returns: AppSwitchMonitor - macOS平台的应用切换监控器实例
func NewAppSwitchMonitor() AppSwitchMonitor {
	return NewDarwinAppSwitchMonitor()
}
