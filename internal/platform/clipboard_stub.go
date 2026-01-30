//go:build !darwin

package platform

import (
	"fmt"
	"sync"
)

// StubClipboardMonitor 存根剪贴板监控器（非 macOS 平台）
//
// StubClipboardMonitor 是 ClipboardMonitor 接口的空实现，用于非 macOS 平台。
// 该实现保存回调函数和运行状态，但不会实际监控剪贴板内容变化。
// 这样设计允许代码在其他平台上编译通过，实现跨平台兼容性。
type StubClipboardMonitor struct {
	// callback 剪贴板事件回调函数（在此实现中不会被调用）
	callback ClipboardCallback
	// isRunning 监控器运行状态标志
	isRunning bool
	// mu 读写锁，保护并发访问
	mu sync.RWMutex
	// stopChan 停止信号通道（保留用于可能的异步操作）
	stopChan chan struct{}
}

// NewClipboardMonitor 创建剪贴板监控器
//
// 根据编译平台自动返回相应的 ClipboardMonitor 实现：
// - macOS 平台：返回 DarwinClipboardMonitor（完整实现）
// - 其他平台：返回 StubClipboardMonitor（空实现）
// Returns: ClipboardMonitor 接口实例
func NewClipboardMonitor() ClipboardMonitor {
	return &StubClipboardMonitor{
		stopChan: make(chan struct{}),
	}
}

// Start 启动剪贴板监控（非 macOS 实现）
//
// 在非 macOS 平台上，此方法只保存回调函数并设置运行状态，
// 不会实际开始监控剪贴板内容变化。
// Parameters: callback - 剪贴板事件回调函数
// Returns: error - 如果监控器已在运行则返回错误
func (sm *StubClipboardMonitor) Start(callback ClipboardCallback) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.isRunning {
		return fmt.Errorf("clipboard monitor already running")
	}

	sm.callback = callback
	sm.isRunning = true
	return nil
}

// Stop 停止剪贴板监控（非 macOS 实现）
//
// 在非 macOS 平台上，此方法只重置运行状态并关闭停止通道。
// Returns: error - 如果监控器未运行则返回错误
func (sm *StubClipboardMonitor) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isRunning {
		return fmt.Errorf("clipboard monitor not running")
	}

	sm.isRunning = false
	close(sm.stopChan)

	return nil
}

// IsRunning 检查运行状态（非 macOS 实现）
//
// 返回监控器当前的运行状态。
// Returns: bool - 监控器是否正在运行
func (sm *StubClipboardMonitor) IsRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.isRunning
}
