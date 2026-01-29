package monitor

import (
	"sync"

	"github.com/chenyang-zz/flowmind/internal/platform"
	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/chenyang-zz/flowmind/pkg/logger"
	"go.uber.org/zap"
)

// KeyboardMonitor 键盘监控器（业务层）
//
// 负责键盘输入的监控和事件处理。本监控器采用分层架构：
//   - 业务层（本结构体）：处理事件、添加上下文、发布到事件总线
//   - 平台层（platform字段）：与操作系统交互，捕获底层键盘事件
//
// 工作流程：
//   1. 平台层捕获原始键盘事件
//   2. 业务层接收平台事件
//   3. 获取当前应用上下文信息
//   4. 构造业务事件并附加上下文
//   5. 发布到事件总线供其他模块消费
type KeyboardMonitor struct {
	// platform 平台层键盘监控器，负责与操作系统交互
	platform platform.KeyboardMonitor

	// eventBus 事件总线，用于发布键盘事件
	eventBus *events.EventBus

	// contextMgr 上下文管理器，用于获取当前应用信息
	contextMgr platform.ContextProvider

	// hotkeyManager 快捷键管理器，用于快捷键注册和匹配
	hotkeyManager *HotkeyManager

	// isRunning 监控器运行状态标志
	isRunning bool

	// mu 读写锁，保护并发访问
	mu sync.RWMutex
}

// NewKeyboardMonitor 创建键盘监控器
//
// 创建一个新的键盘监控器实例，并初始化其依赖的平台层组件、上下文管理器和快捷键管理器。
//
// Parameters:
//   - eventBus: 事件总线实例，用于发布键盘事件
//
// Returns: Monitor - 新创建的键盘监控器实例（返回接口类型）
func NewKeyboardMonitor(eventBus *events.EventBus) Monitor {
	return &KeyboardMonitor{
		platform:      platform.NewKeyboardMonitor(),
		eventBus:      eventBus,
		contextMgr:    platform.NewContextProvider(),
		hotkeyManager: NewHotkeyManager(eventBus),
	}
}

// Start 启动键盘监控
//
// 启动平台层的键盘监控器，并注册事件回调函数。
// 如果监控器已经在运行，则幂等地返回成功。
//
// Returns: error - 启动失败时返回错误（如缺少系统权限）
func (km *KeyboardMonitor) Start() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if km.isRunning {
		logger.Debug("键盘监控器已在运行", zap.String("component", "keyboard"))
		return nil // 已经在运行
	}

	logger.Info("启动键盘监控器", zap.String("component", "keyboard"))

	// 启动平台层监控器，并传入回调函数
	if err := km.platform.Start(km.handlePlatformEvent); err != nil {
		logger.Error("启动平台层键盘监控器失败",
			zap.String("component", "keyboard"),
			zap.Error(err),
		)
		return err
	}

	// 启动快捷键管理器
	if km.hotkeyManager != nil {
		if err := km.hotkeyManager.Start(); err != nil {
			// 快捷键管理器启动失败，不影响主监控器
			logger.Warn("快捷键管理器启动失败，但不影响键盘监控",
				zap.String("component", "keyboard"),
				zap.Error(err),
			)
		}
	}

	km.isRunning = true
	logger.Info("键盘监控器启动成功", zap.String("component", "keyboard"))
	return nil
}

// Stop 停止键盘监控
//
// 停止平台层的键盘监控器并释放相关资源。
// 如果监控器未运行，则幂等地返回成功。
//
// Returns: error - 停止失败时返回错误
func (km *KeyboardMonitor) Stop() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if !km.isRunning {
		logger.Debug("键盘监控器未运行", zap.String("component", "keyboard"))
		return nil // 未运行
	}

	logger.Info("停止键盘监控器", zap.String("component", "keyboard"))

	// 停止快捷键管理器
	if km.hotkeyManager != nil {
		_ = km.hotkeyManager.Stop()
	}

	if err := km.platform.Stop(); err != nil {
		logger.Error("停止平台层键盘监控器失败",
			zap.String("component", "keyboard"),
			zap.Error(err),
		)
		return err
	}

	km.isRunning = false
	logger.Info("键盘监控器已停止", zap.String("component", "keyboard"))
	return nil
}

// IsRunning 检查运行状态
//
// 线程安全地检查监控器是否正在运行。
//
// Returns: bool - true 表示正在运行，false 表示已停止
func (km *KeyboardMonitor) IsRunning() bool {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.isRunning
}

// GetHotkeyManager 获取快捷键管理器
//
// 返回键盘监控器管理的快捷键管理器实例，可用于注册和取消注册快捷键。
//
// Returns: *HotkeyManager - 快捷键管理器实例，可能为 nil（在监控器未初始化时）
func (km *KeyboardMonitor) GetHotkeyManager() *HotkeyManager {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.hotkeyManager
}

// handlePlatformEvent 处理平台层传来的原始键盘事件
//
// 作为平台层监控器的回调函数，接收原始键盘事件并将其转换为业务事件。
// 处理流程：
//   1. 从上下文管理器获取当前应用信息
//   2. 提取键盘事件的关键信息（按键码、修饰键）
//   3. 构造业务键盘事件
//   4. 附加上下文信息（当前应用）
//   5. 发布到事件总线
//
// Parameters:
//   - event: 平台层的原始键盘事件
func (km *KeyboardMonitor) handlePlatformEvent(event platform.KeyboardEvent) {
	// Debug 级别记录键盘事件（开发时有用，生产环境可以关闭）
	logger.Debug("捕获键盘事件",
		zap.String("component", "keyboard"),
		zap.Int("keycode", event.KeyCode),
		zap.Uint64("modifiers", event.Modifiers),
	)

	// 1. 获取上下文
	context := km.contextMgr.GetContext()

	// 2. 构造业务事件数据
	data := map[string]interface{}{
		"keycode":   event.KeyCode,
		"modifiers": event.Modifiers,
	}

	// 3. 创建业务事件
	businessEvent := events.NewEvent(events.EventTypeKeyboard, data)
	businessEvent.WithContext(context)

	// 4. 发布到事件总线
	if err := km.eventBus.Publish(string(events.EventTypeKeyboard), *businessEvent); err != nil {
		logger.Error("发布键盘事件失败",
			zap.String("component", "keyboard"),
			zap.Error(err),
		)
	}
}
