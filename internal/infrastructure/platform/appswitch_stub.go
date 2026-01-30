//go:build !darwin

package platform

import (
	"fmt"
	"sync"
)

// StubAppSwitchMonitor 应用切换监控器的stub实现（非macOS平台）
// 在非macOS平台上，应用切换监控功能不可用
type StubAppSwitchMonitor struct {
	// isRunning 监控器运行状态
	isRunning bool

	// mu 互斥锁，保护并发访问
	mu sync.RWMutex
}

// NewStubAppSwitchMonitor 创建应用切换监控器的stub实现
// Returns: *StubAppSwitchMonitor - 新创建的监控器实例
func NewStubAppSwitchMonitor() *StubAppSwitchMonitor {
	return &StubAppSwitchMonitor{}
}

// Start 启动应用切换监控（stub实现，直接返回错误）
// Parameters: callback - 应用切换事件回调函数
// Returns: error - 在非macOS平台上始终返回错误
func (m *StubAppSwitchMonitor) Start(callback AppSwitchCallback) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return fmt.Errorf("应用切换监控仅在macOS平台上可用")
}

// Stop 停止应用切换监控（stub实现）
// Returns: error - 始终返回nil
func (m *StubAppSwitchMonitor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return nil
}

// IsRunning 检查运行状态（stub实现）
// Returns: bool - 始终返回false
func (m *StubAppSwitchMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return false
}

// NewAppSwitchMonitor 创建应用切换监控器的stub实现
// Returns: AppSwitchMonitor - stub实现的应用切换监控器实例
func NewAppSwitchMonitor() AppSwitchMonitor {
	return NewStubAppSwitchMonitor()
}
