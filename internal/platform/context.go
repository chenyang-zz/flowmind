package platform

import (
	"github.com/chenyang-zz/flowmind/pkg/events"
)

// ContextProvider 提供应用上下文信息的接口
//
// ContextProvider 定义了获取当前活动应用上下文的方法，包括应用名称、
// Bundle ID 和窗口标题等信息。这些信息用于事件系统提供更丰富的上下文。
type ContextProvider interface {
	// GetFrontmostApp 获取最前端应用名称
	// Returns: 应用本地化名称，如 "Chrome"、"Safari" 等
	GetFrontmostApp() string

	// GetBundleID 获取应用 Bundle ID（仅 macOS）
	// Bundle ID 是应用的唯一标识符，格式如 "com.google.Chrome"
	// Returns: 应用的 Bundle ID 标识符
	GetBundleID() string

	// GetFocusedWindowTitle 获取焦点窗口标题
	// Returns: 当前焦点窗口的标题文本，可能为空
	GetFocusedWindowTitle() string

	// GetContext 获取完整的应用上下文
	// Returns: 包含应用名称、Bundle ID 和窗口标题的事件上下文对象
	GetContext() *events.EventContext
}

// KeyboardEvent 键盘原始事件数据
//
// KeyboardEvent 封装了键盘事件的基本信息，包括按键代码和修饰键状态。
type KeyboardEvent struct {
	// KeyCode 按键代码，对应 macOS 虚拟键码
	// 常见键码: 0=Q, 1=W, 2=E, ..., 40=K, 41=;, 55=Command(左), 54=Command(右)
	KeyCode int
	// Modifiers 修饰键标志位
	// 包含 Command、Shift、Control、Option 等修饰键的状态
	Modifiers uint64
}

// KeyboardCallback 键盘事件回调函数类型
//
// 当检测到键盘事件时，监控器会调用此回调函数，将事件数据传递给调用者。
// Parameters: event - 键盘事件数据，包含按键代码和修饰键状态
type KeyboardCallback func(KeyboardEvent)

// KeyboardMonitor 键盘监控器接口
//
// KeyboardMonitor 定义了键盘事件监控的生命周期管理方法。
// 监控器可以捕获系统级的键盘按键事件，用于快捷键、自动化等场景。
// 注意：在 macOS 上需要授予辅助功能权限才能正常工作。
type KeyboardMonitor interface {
	// Start 启动键盘监控
	// 启动后会持续捕获键盘事件并通过回调函数通知
	// Parameters: callback - 键盘事件回调函数
	// Returns: error - 启动失败时返回错误（如缺少权限、已运行等）
	Start(callback KeyboardCallback) error

	// Stop 停止键盘监控
	// 停止后会释放系统资源并取消事件监听
	// Returns: error - 停止失败时返回错误（如未运行等）
	Stop() error

	// IsRunning 检查运行状态
	// Returns: bool - 监控器是否正在运行
	IsRunning() bool
}
