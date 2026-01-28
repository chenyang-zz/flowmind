//go:build !darwin

package platform

import (
	"github.com/chenyang-zz/flowmind/pkg/events"
)

// StubContextManager 存根上下文管理器（非 macOS 平台）
//
// StubContextManager 是 ContextProvider 接口的空实现，用于非 macOS 平台（如 Linux、Windows）。
// 该实现返回空值，因为在这些平台上尚未实现应用上下文获取功能。
// 这样设计允许代码在其他平台上编译通过，但不会提供实际的上下文信息。
type StubContextManager struct{}

// NewContextProvider 创建上下文管理器
//
// 根据编译平台自动返回相应的 ContextProvider 实现：
// - macOS 平台：返回 DarwinContextManager（完整实现）
// - 其他平台：返回 StubContextManager（空实现）
// Returns: ContextProvider 接口实例
func NewContextProvider() ContextProvider {
	return &StubContextManager{}
}

// GetFrontmostApp 获取最前端应用名称（非 macOS 实现）
//
// 此方法在非 macOS 平台上返回空字符串，表示不支持该功能。
// Returns: 始终返回空字符串
func (sm *StubContextManager) GetFrontmostApp() string {
	return ""
}

// GetBundleID 获取应用 Bundle ID（非 macOS 实现）
//
// 此方法在非 macOS 平台上返回空字符串，表示不支持该功能。
// Returns: 始终返回空字符串
func (sm *StubContextManager) GetBundleID() string {
	return ""
}

// GetFocusedWindowTitle 获取焦点窗口标题（非 macOS 实现）
//
// 此方法在非 macOS 平台上返回空字符串，表示不支持该功能。
// Returns: 始终返回空字符串
func (sm *StubContextManager) GetFocusedWindowTitle() string {
	return ""
}

// GetContext 获取完整的应用上下文（非 macOS 实现）
//
// 此方法在非 macOS 平台上返回空的事件上下文对象。
// 返回的结构体字段均为空字符串，表示不支持该功能。
// Returns: 空的 EventContext 对象
func (sm *StubContextManager) GetContext() *events.EventContext {
	return &events.EventContext{
		Application: "",
		BundleID:    "",
		WindowTitle: "",
	}
}
