//go:build !darwin

package platform

import (
	"fmt"
	"sync"
)

// StubKeyboardMonitor 存根键盘监控器（非 macOS 平台）
//
// StubKeyboardMonitor 是 KeyboardMonitor 接口的空实现，用于非 macOS 平台。
// 该实现保存回调函数和运行状态，但不会实际捕获键盘事件。
// 这样设计允许代码在其他平台上编译通过，实现跨平台兼容性。
type StubKeyboardMonitor struct {
	// callback 键盘事件回调函数（在此实现中不会被调用）
	callback KeyboardCallback
	// isRunning 监控器运行状态标志
	isRunning bool
	// mu 读写锁，保护并发访问
	mu sync.RWMutex
	// stopChan 停止信号通道（保留用于可能的异步操作）
	stopChan chan struct{}
}

// NewKeyboardMonitor 创建键盘监控器
//
// 根据编译平台自动返回相应的 KeyboardMonitor 实现：
// - macOS 平台：返回 DarwinKeyboardMonitor（完整实现）
// - 其他平台：返回 StubKeyboardMonitor（空实现）
// Returns: KeyboardMonitor 接口实例
func NewKeyboardMonitor() KeyboardMonitor {
	return &StubKeyboardMonitor{
		stopChan: make(chan struct{}),
	}
}

// Start 启动键盘监控（非 macOS 实现）
//
// 在非 macOS 平台上，此方法只保存回调函数并设置运行状态，
// 不会实际开始捕获键盘事件。
// Parameters: callback - 键盘事件回调函数
// Returns: error - 如果监控器已在运行则返回错误
func (sm *StubKeyboardMonitor) Start(callback KeyboardCallback) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.isRunning {
		return fmt.Errorf("keyboard monitor already running")
	}

	sm.callback = callback
	sm.isRunning = true
	return nil
}

// Stop 停止键盘监控（非 macOS 实现）
//
// 在非 macOS 平台上，此方法只重置运行状态并关闭停止通道。
// Returns: error - 如果监控器未运行则返回错误
func (sm *StubKeyboardMonitor) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isRunning {
		return fmt.Errorf("keyboard monitor not running")
	}

	sm.isRunning = false
	close(sm.stopChan)

	return nil
}

// IsRunning 检查运行状态（非 macOS 实现）
//
// 返回监控器当前的运行状态。
// Returns: bool - 监控器是否正在运行
func (sm *StubKeyboardMonitor) IsRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.isRunning
}
