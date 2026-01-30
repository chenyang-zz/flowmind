package monitor

import (
	"sync"

	"github.com/chenyang-zz/flowmind/internal/infrastructure/platform"
	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// ClipboardMonitor 剪贴板监控器（业务层）
//
// 负责剪贴板内容变化的监控和事件处理。本监控器采用分层架构：
//   - 业务层（本结构体）：处理事件、添加上下文、发布到事件总线
//   - 平台层（platform字段）：与操作系统交互，监控剪贴板变化
//
// 工作流程：
//   1. 平台层检测到剪贴板内容变化
//   2. 业务层接收平台事件
//   3. 获取当前应用上下文信息
//   4. 构造业务事件并附加上下文
//   5. 发布到事件总线供其他模块消费
type ClipboardMonitor struct {
	// platform 平台层剪贴板监控器，负责与操作系统交互
	platform platform.ClipboardMonitor

	// eventBus 事件总线，用于发布剪贴板事件
	eventBus *events.EventBus

	// contextMgr 上下文管理器，用于获取当前应用信息
	contextMgr platform.ContextProvider

	// isRunning 监控器运行状态标志
	isRunning bool

	// mu 读写锁，保护并发访问
	mu sync.RWMutex

	// lastContent 上一次记录的剪贴板内容，用于去重
	lastContent string
}

// NewClipboardMonitor 创建剪贴板监控器
//
// 创建一个新的剪贴板监控器实例，并初始化其依赖的平台层组件和上下文管理器。
//
// Parameters:
//   - eventBus: 事件总线实例，用于发布剪贴板事件
//
// Returns: Monitor - 新创建的剪贴板监控器实例（返回接口类型）
func NewClipboardMonitor(eventBus *events.EventBus) Monitor {
	return &ClipboardMonitor{
		platform:   platform.NewClipboardMonitor(),
		eventBus:   eventBus,
		contextMgr: platform.NewContextProvider(),
	}
}

// Start 启动剪贴板监控
//
// 启动平台层的剪贴板监控器，并注册事件回调函数。
// 如果监控器已经在运行，则幂等地返回成功。
//
// Returns: error - 启动失败时返回错误
func (cm *ClipboardMonitor) Start() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.isRunning {
		logger.Debug("剪贴板监控器已在运行", zap.String("component", "clipboard"))
		return nil // 已经在运行
	}

	logger.Info("启动剪贴板监控器", zap.String("component", "clipboard"))

	// 启动平台层监控器，并传入回调函数
	if err := cm.platform.Start(cm.handlePlatformEvent); err != nil {
		logger.Error("启动平台层剪贴板监控器失败",
			zap.String("component", "clipboard"),
			zap.Error(err),
		)
		return err
	}

	cm.isRunning = true
	logger.Info("剪贴板监控器启动成功", zap.String("component", "clipboard"))
	return nil
}

// Stop 停止剪贴板监控
//
// 停止平台层的剪贴板监控器并释放相关资源。
// 如果监控器未运行，则幂等地返回成功。
//
// Returns: error - 停止失败时返回错误
func (cm *ClipboardMonitor) Stop() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.isRunning {
		logger.Debug("剪贴板监控器未运行", zap.String("component", "clipboard"))
		return nil // 未运行
	}

	logger.Info("停止剪贴板监控器", zap.String("component", "clipboard"))

	if err := cm.platform.Stop(); err != nil {
		logger.Error("停止平台层剪贴板监控器失败",
			zap.String("component", "clipboard"),
			zap.Error(err),
		)
		return err
	}

	cm.isRunning = false
	cm.lastContent = ""
	logger.Info("剪贴板监控器已停止", zap.String("component", "clipboard"))
	return nil
}

// IsRunning 检查运行状态
//
// 线程安全地检查监控器是否正在运行。
//
// Returns: bool - true 表示正在运行，false 表示已停止
func (cm *ClipboardMonitor) IsRunning() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.isRunning
}

// handlePlatformEvent 处理平台层传来的剪贴板变化事件
//
// 作为平台层监控器的回调函数，接收剪贴板事件并将其转换为业务事件。
// 处理流程：
//   1. 检查内容是否与上次相同（去重）
//   2. 从上下文管理器获取当前应用信息
//   3. 提取剪贴板事件的关键信息（内容、类型、大小）
//   4. 构造业务剪贴板事件
//   5. 附加上下文信息（当前应用）
//   6. 发布到事件总线
//
// Parameters:
//   - event: 平台层的剪贴板事件
func (cm *ClipboardMonitor) handlePlatformEvent(event platform.ClipboardEvent) {
	// 1. 检查内容是否与上次相同（去重）
	// 平台层已经通过 changeCount 进行了去重，这里是二次保险
	if event.Content == cm.lastContent {
		logger.Debug("剪贴板内容未变化，忽略", zap.String("component", "clipboard"))
		return
	}

	cm.mu.Lock()
	cm.lastContent = event.Content
	cm.mu.Unlock()

	// 记录日志（截取内容以避免日志过长）
	contentPreview := event.Content
	if len(contentPreview) > 100 {
		contentPreview = contentPreview[:100] + "..."
	}

	logger.Info("检测到剪贴板内容变化",
		zap.String("component", "clipboard"),
		zap.String("type", event.Type),
		zap.Int64("size", event.Size),
		zap.String("preview", contentPreview),
	)

	// 2. 获取上下文
	context := cm.contextMgr.GetContext()

	// 3. 构造业务事件数据
	data := map[string]interface{}{
		"content": event.Content,
		"type":    event.Type,
		"size":    event.Size,
		"length":  len(event.Content),
	}

	// 4. 创建业务事件
	businessEvent := events.NewEvent(events.EventTypeClipboard, data)
	businessEvent.WithContext(context)

	// 5. 发布到事件总线
	if err := cm.eventBus.Publish(string(events.EventTypeClipboard), *businessEvent); err != nil {
		logger.Error("发布剪贴板事件失败",
			zap.String("component", "clipboard"),
			zap.Error(err),
		)
	}
}
