package monitor

import (
	"sync"

	"github.com/chenyang-zz/flowmind/internal/infrastructure/platform"
	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// ApplicationMonitor 应用监控器（业务层）
//
// 负责应用切换的监控和事件处理。本监控器采用分层架构：
//   - 业务层（本结构体）：处理事件、添加上下文、发布到事件总线
//   - 平台层（platform字段）：与操作系统交互，捕获底层应用切换事件
//
// 工作流程：
//   1. 平台层捕获应用切换事件
//   2. 业务层接收平台事件
//   3. 更新应用会话追踪器
//   4. 构造业务事件并附加上下文
//   5. 发布到事件总线供其他模块消费
type ApplicationMonitor struct {
	// platform 平台层应用切换监控器，负责与操作系统交互
	platform platform.AppSwitchMonitor

	// eventBus 事件总线，用于发布应用切换事件
	eventBus *events.EventBus

	// contextMgr 上下文管理器，用于获取当前应用信息
	contextMgr platform.ContextProvider

	// appTracker 应用会话追踪器，记录应用使用时长
	appTracker *AppTracker

	// isRunning 监控器运行状态标志
	isRunning bool

	// mu 读写锁，保护并发访问
	mu sync.RWMutex
}

// NewApplicationMonitor 创建应用监控器
//
// 创建一个新的应用监控器实例，并初始化其依赖的平台层组件、上下文管理器和应用会话追踪器。
//
// Parameters:
//   - eventBus: 事件总线实例，用于发布应用切换事件
//
// Returns: Monitor - 新创建的应用监控器实例（返回接口类型）
func NewApplicationMonitor(eventBus *events.EventBus) Monitor {
	return &ApplicationMonitor{
		platform:   platform.NewAppSwitchMonitor(),
		eventBus:   eventBus,
		contextMgr: platform.NewContextProvider(),
		appTracker: NewAppTracker(eventBus),
	}
}

// Start 启动应用监控
//
// 启动平台层应用监控器，并注册事件回调函数。
// 如果监控器已经在运行，则幂等地返回成功。
//
// Returns: error - 启动失败时返回错误（如缺少系统权限）
func (am *ApplicationMonitor) Start() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.isRunning {
		logger.Debug("应用监控器已在运行", zap.String("component", "application"))
		return nil // 已经在运行
	}

	logger.Info("启动应用监控器", zap.String("component", "application"))

	// 启动平台层监控器，并传入回调函数
	if err := am.platform.Start(am.handlePlatformEvent); err != nil {
		logger.Error("启动平台层应用监控器失败",
			zap.String("component", "application"),
			zap.Error(err),
		)
		return err
	}

	am.isRunning = true
	logger.Info("应用监控器启动成功", zap.String("component", "application"))
	return nil
}

// Stop 停止应用监控
//
// 停止平台层应用监控器，结束所有应用会话，并释放相关资源。
// 如果监控器未运行，则幂等地返回成功。
//
// Returns: error - 停止失败时返回错误
func (am *ApplicationMonitor) Stop() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if !am.isRunning {
		logger.Debug("应用监控器未运行", zap.String("component", "application"))
		return nil // 未运行
	}

	logger.Info("停止应用监控器", zap.String("component", "application"))

	// 结束所有应用会话
	am.appTracker.EndAllSessions()

	if err := am.platform.Stop(); err != nil {
		logger.Error("停止平台层应用监控器失败",
			zap.String("component", "application"),
			zap.Error(err),
		)
		return err
	}

	am.isRunning = false
	logger.Info("应用监控器已停止", zap.String("component", "application"))
	return nil
}

// IsRunning 检查运行状态
//
// 线程安全地检查监控器是否正在运行。
//
// Returns: bool - true 表示正在运行，false 表示已停止
func (am *ApplicationMonitor) IsRunning() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.isRunning
}

// handlePlatformEvent 处理平台层应用切换事件
//
// 当平台层检测到应用切换时调用此方法。该方法负责：
//   1. 更新应用会话追踪器
//   2. 构造业务事件
//   3. 添加应用上下文
//   4. 发布到事件总线
//
// Parameters: event - 平台层应用切换事件
func (am *ApplicationMonitor) handlePlatformEvent(event platform.AppSwitchEvent) {
	// 更新应用会话追踪器
	am.appTracker.SwitchApp(event.From, event.To, event.BundleID)

	// 构造应用切换事件
	appSwitchEvent := events.NewEvent(events.EventTypeAppSwitch, map[string]interface{}{
		"from":      event.From,
		"to":        event.To,
		"bundle_id": event.BundleID,
		"window":    event.Window,
	})

	// 添加上下文信息
	context := am.contextMgr.GetContext()
	appSwitchEvent.WithContext(context)

	// 发布事件到事件总线
	am.eventBus.Publish(string(events.EventTypeAppSwitch), *appSwitchEvent)

	logger.Debug("应用切换事件已发布",
		zap.String("component", "application"),
		zap.String("from", event.From),
		zap.String("to", event.To),
		zap.String("bundle_id", event.BundleID),
	)
}
