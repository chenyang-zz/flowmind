package monitor

import (
	"fmt"
	"sync"

	"github.com/chenyang-zz/flowmind/pkg/events"
)

// Engine 监控引擎，管理所有监控器
//
// 负责统一管理和协调各个监控器的生命周期，是监控系统的核心组件。
// 引擎维护所有监控器的运行状态，并提供统一的启动/停止接口。
// 同时负责发布监控引擎的状态变更事件到事件总线。
type Engine struct {
	// keyboard 键盘监控器实例
	keyboard *KeyboardMonitor

	// eventBus 事件总线，用于发布和订阅事件
	eventBus *events.EventBus

	// isRunning 引擎运行状态标志
	isRunning bool

	// mu 读写锁，保护并发访问
	mu sync.RWMutex
}

// NewEngine 创建监控引擎
//
// Parameters:
//   - eventBus: 事件总线实例，用于发布监控事件
//
// Returns: Monitor - 新创建的监控引擎实例（返回接口类型）
func NewEngine(eventBus *events.EventBus) Monitor {
	return &Engine{
		eventBus: eventBus,
	}
}

// Start 启动监控引擎
//
// 初始化并启动所有监控器，包括键盘监控器等。
// 启动成功后会发布状态事件到事件总线。
// 如果引擎已经运行，则返回错误。
//
// Returns: error - 启动失败时返回错误，如引擎已运行或监控器启动失败
func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.isRunning {
		return fmt.Errorf("monitor engine already running")
	}

	// 初始化键盘监控器
	e.keyboard = NewKeyboardMonitor(e.eventBus)

	// 启动键盘监控器
	if err := e.keyboard.Start(); err != nil {
		return fmt.Errorf("failed to start keyboard monitor: %w", err)
	}

	e.isRunning = true

	// 发布状态事件
	statusEvent := events.NewEvent(events.EventTypeStatus, map[string]interface{}{
		"status":   "started",
		"monitors": []string{"keyboard"},
	})
	e.eventBus.Publish(string(events.EventTypeStatus), *statusEvent)

	return nil
}

// Stop 停止监控引擎
//
// 停止所有正在运行的监控器并释放相关资源。
// 停止成功后会发布状态事件到事件总线。
// 如果引擎未运行，则返回错误。
//
// Returns: error - 停止失败时返回错误，如引擎未运行或监控器停止失败
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning {
		return fmt.Errorf("monitor engine not running")
	}

	// 停止键盘监控器
	if e.keyboard != nil {
		if err := e.keyboard.Stop(); err != nil {
			return fmt.Errorf("failed to stop keyboard monitor: %w", err)
		}
	}

	e.isRunning = false

	// 发布状态事件
	statusEvent := events.NewEvent(events.EventTypeStatus, map[string]interface{}{
		"status": "stopped",
	})
	e.eventBus.Publish(string(events.EventTypeStatus), *statusEvent)

	return nil
}

// IsRunning 检查运行状态
//
// 线程安全地检查引擎是否正在运行。
//
// Returns: bool - true 表示正在运行，false 表示已停止
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isRunning
}

// GetKeyboardMonitor 获取键盘监控器实例
//
// 返回引擎管理的键盘监控器实例，可用于直接访问键盘监控器。
// 注意：返回的实例可能为 nil（在引擎未启动时）。
//
// Returns: *KeyboardMonitor - 键盘监控器实例，可能为 nil
func (e *Engine) GetKeyboardMonitor() *KeyboardMonitor {
	return e.keyboard
}
