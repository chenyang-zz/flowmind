//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#include <Cocoa/Cocoa.h>
#include <stdlib.h>

// getFrontmostAppName 获取当前最前端应用的本地化名称
// 使用 NSWorkspace 获取 frontmostApplication，然后提取其 localizedName
// Returns: 新分配的 C 字符串，调用者需要使用 free() 释放
char* getFrontmostAppName() {
    NSRunningApplication *app = [NSWorkspace sharedWorkspace].frontmostApplication;
    if (app == nil) {
        return strdup("");
    }
    NSString *appName = [app localizedName];
    if (appName == nil) {
        return strdup("");
    }
    const char* cName = [appName UTF8String];
    return strdup(cName);
}

// getBundleID 获取最前端应用的 Bundle Identifier
// Bundle ID 是应用在 macOS 中的唯一标识符，格式如 "com.apple.Safari"
// Returns: 新分配的 C 字符串，调用者需要使用 free() 释放
char* getBundleID() {
    NSRunningApplication *app = [NSWorkspace sharedWorkspace].frontmostApplication;
    if (app == nil) {
        return strdup("");
    }
    NSString *bundleID = [app bundleIdentifier];
    if (bundleID == nil) {
        return strdup("");
    }
    const char* cBundleID = [bundleID UTF8String];
    return strdup(cBundleID);
}

// getFocusedWindowTitle 获取当前焦点窗口的标题
// 使用 Accessibility API (AXUIElement) 获取窗口的标题属性
// 这个过程包括：获取应用元素 -> 获取焦点窗口 -> 获取窗口标题
// Returns: 新分配的 C 字符串，调用者需要使用 free() 释放
char* getFocusedWindowTitle() {
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

    // 获取焦点窗口
    AXUIElementRef window = NULL;
    AXError err = AXUIElementCopyAttributeValue(appElement, kAXFocusedWindowAttribute, (CFTypeRef*)&window);
    if (err != kAXErrorSuccess || window == NULL) {
        if (appElement != NULL) {
            CFRelease(appElement);
        }
        return strdup("");
    }

    // 获取窗口标题
    CFStringRef title = NULL;
    err = AXUIElementCopyAttributeValue(window, kAXTitleAttribute, (CFTypeRef*)&title);

    if (err != kAXErrorSuccess || title == NULL) {
        if (window != NULL) {
            CFRelease(window);
        }
        if (appElement != NULL) {
            CFRelease(appElement);
        }
        return strdup("");
    }

    // 转换为 C 字符串
    NSString *nsTitle = (__bridge NSString*)title;
    const char* cTitle = [nsTitle UTF8String];
    char* result = strdup(cTitle);

    // 清理资源
    if (window != NULL) {
        CFRelease(window);
    }
    if (appElement != NULL) {
        CFRelease(appElement);
    }

    return result;
}
*/
import "C"
import (
	"unsafe"

	"github.com/chenyang-zz/flowmind/pkg/events"
)

// DarwinContextManager macOS 平台的上下文管理器实现
//
// DarwinContextManager 实现了 ContextProvider 接口，通过调用 macOS 系统 API
// (NSWorkspace 和 Accessibility API) 来获取当前活动应用和窗口的信息。
type DarwinContextManager struct{}

// NewContextProvider 创建上下文管理器
//
// 在 macOS 平台上，此函数返回 DarwinContextManager 实例。
// Returns: ContextProvider 接口的 macOS 实现
func NewContextProvider() ContextProvider {
	return &DarwinContextManager{}
}

// GetFrontmostApp 获取最前端应用名称
//
// 调用 C 函数 getFrontmostAppName() 使用 NSWorkspace API 获取应用名称。
// Returns: 应用的本地化名称（如 "Safari"、"Chrome"）
func (dm *DarwinContextManager) GetFrontmostApp() string {
	cStr := C.getFrontmostAppName()
	defer C.free(unsafe.Pointer(cStr))
	return C.GoString(cStr)
}

// GetBundleID 获取应用 Bundle ID
//
// 调用 C 函数 getBundleID() 使用 NSWorkspace API 获取应用的 Bundle Identifier。
// Bundle ID 是应用在 macOS 中的唯一标识符。
// Returns: 应用的 Bundle ID（如 "com.apple.Safari"）
func (dm *DarwinContextManager) GetBundleID() string {
	cStr := C.getBundleID()
	defer C.free(unsafe.Pointer(cStr))
	return C.GoString(cStr)
}

// GetFocusedWindowTitle 获取焦点窗口标题
//
// 调用 C 函数 getFocusedWindowTitle() 使用 Accessibility API 获取窗口标题。
// 注意：某些应用可能没有窗口标题，此方法会返回空字符串。
// Returns: 焦点窗口的标题文本
func (dm *DarwinContextManager) GetFocusedWindowTitle() string {
	cStr := C.getFocusedWindowTitle()
	defer C.free(unsafe.Pointer(cStr))
	return C.GoString(cStr)
}

// GetContext 获取完整的应用上下文
//
// 组合所有上下文信息，返回一个完整的 EventContext 对象。
// Returns: 包含应用名称、Bundle ID 和窗口标题的事件上下文
func (dm *DarwinContextManager) GetContext() *events.EventContext {
	return &events.EventContext{
		Application: dm.GetFrontmostApp(),
		BundleID:    dm.GetBundleID(),
		WindowTitle: dm.GetFocusedWindowTitle(),
	}
}
