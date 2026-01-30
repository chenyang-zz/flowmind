package monitor

import (
	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// 预定义快捷键常量
//
// 这些是 FlowMind 的默认快捷键，用户可以在设置中自定义。
const (
	// HotkeyAIAssistant AI 助手面板快捷键
	// 功能：打开/关闭 AI 助手面板，提供智能建议和自动化选项
	HotkeyAIAssistant = "Cmd+Shift+M"

	// HotkeyAutomationSuggestions 自动化建议快捷键
	// 功能：显示当前操作的自动化建议列表
	HotkeyAutomationSuggestions = "Cmd+Shift+A"

	// HotkeyKeybindings 快捷键列表快捷键
	// 功能：显示所有可用快捷键及其功能
	HotkeyKeybindings = "Cmd+Shift+K"

	// HotkeyToggleMonitoring 暂停/恢复监控快捷键
	// 功能：暂停或恢复工作流监控（隐私保护）
	HotkeyToggleMonitoring = "Cmd+Shift+P"

	// HotkeyShowStatus 显示状态快捷键
	// 功能：显示 FlowMind 当前状态和统计信息
	HotkeyShowStatus = "Cmd+Shift+H"
)

// 预定义快捷键事件类型
//
// 这些是快捷键触发后发布的自定义事件类型，
// 前端可以订阅这些事件来响应快捷键操作。
const (
	// EventTypeHotkeyToggleAI toggles AI 助手面板事件
	EventTypeHotkeyToggleAI events.EventType = "hotkey.toggle_ai"

	// EventTypeHotkeyShowSuggestions 显示自动化建议事件
	EventTypeHotkeyShowSuggestions events.EventType = "hotkey.show_suggestions"

	// EventTypeHotkeyShowKeybindings 显示快捷键列表事件
	EventTypeHotkeyShowKeybindings events.EventType = "hotkey.show_keybindings"

	// EventTypeHotkeyToggleMonitoring 切换监控状态事件
	EventTypeHotkeyToggleMonitoring events.EventType = "hotkey.toggle_monitoring"

	// EventTypeHotkeyShowStatus 显示状态事件
	EventTypeHotkeyShowStatus events.EventType = "hotkey.show_status"
)

// registerPresetHotkeys 注册预定义的快捷键
//
// 在监控引擎启动时自动注册这些快捷键。
// 每个快捷键触发时会发布相应的事件到事件总线，
// 前端或其他模块可以订阅这些事件来响应快捷键操作。
//
// Parameters:
//   - manager: 快捷键管理器实例
//   - eventBus: 事件总线实例，用于发布快捷键事件
func registerPresetHotkeys(manager *HotkeyManager, eventBus *events.EventBus) {
	// 注册 AI 助手面板快捷键
	if _, err := manager.Register(HotkeyAIAssistant, createToggleAIHandler(eventBus)); err != nil {
		logger.Warn("注册快捷键失败",
			zap.String("hotkey", HotkeyAIAssistant),
			zap.Error(err),
		)
	}

	// 注册自动化建议快捷键
	if _, err := manager.Register(HotkeyAutomationSuggestions, createShowSuggestionsHandler(eventBus)); err != nil {
		logger.Warn("注册快捷键失败",
			zap.String("hotkey", HotkeyAutomationSuggestions),
			zap.Error(err),
		)
	}

	// 注册快捷键列表快捷键
	if _, err := manager.Register(HotkeyKeybindings, createShowKeybindingsHandler(eventBus)); err != nil {
		logger.Warn("注册快捷键失败",
			zap.String("hotkey", HotkeyKeybindings),
			zap.Error(err),
		)
	}

	// 注册切换监控快捷键
	if _, err := manager.Register(HotkeyToggleMonitoring, createToggleMonitoringHandler(eventBus)); err != nil {
		logger.Warn("注册快捷键失败",
			zap.String("hotkey", HotkeyToggleMonitoring),
			zap.Error(err),
		)
	}

	// 注册显示状态快捷键
	if _, err := manager.Register(HotkeyShowStatus, createShowStatusHandler(eventBus)); err != nil {
		logger.Warn("注册快捷键失败",
			zap.String("hotkey", HotkeyShowStatus),
			zap.Error(err),
		)
	}

	logger.Info("预定义快捷键注册完成",
		zap.Int("count", 5),
		zap.String("shortcuts", "Cmd+Shift+M/A/K/P/H"),
	)
}

// createToggleAIHandler 创建 AI 助手面板切换处理函数
//
// 功能：打开/关闭 AI 助手面板
// 发布事件：EventTypeHotkeyToggleAI
func createToggleAIHandler(eventBus *events.EventBus) HotkeyCallback {
	return func(reg *HotkeyRegistration, ctx *events.EventContext) {
		logger.Info("🤖 快捷键触发: AI 助手面板",
			zap.String("hotkey", reg.Hotkey.String()),
			zap.String("application", ctx.Application),
		)

		// 发布快捷键事件
		event := events.NewEvent(EventTypeHotkeyToggleAI, map[string]interface{}{
			"action": "toggle",
			"source": "hotkey",
		})
		event.WithContext(ctx)

		// 发布到事件总线
		if err := eventBus.Publish(string(EventTypeHotkeyToggleAI), *event); err != nil {
			logger.Error("发布快捷键事件失败",
				zap.String("event_type", string(EventTypeHotkeyToggleAI)),
				zap.Error(err),
			)
		}

		logger.Info("✅ AI 助手面板切换事件已发布",
			zap.String("event_type", string(EventTypeHotkeyToggleAI)),
		)
	}
}

// createShowSuggestionsHandler 创建显示自动化建议处理函数
//
// 功能：显示当前操作的自动化建议
// 发布事件：EventTypeHotkeyShowSuggestions
func createShowSuggestionsHandler(eventBus *events.EventBus) HotkeyCallback {
	return func(reg *HotkeyRegistration, ctx *events.EventContext) {
		logger.Info("💡 快捷键触发: 显示自动化建议",
			zap.String("hotkey", reg.Hotkey.String()),
			zap.String("application", ctx.Application),
		)

		// 发布快捷键事件
		event := events.NewEvent(EventTypeHotkeyShowSuggestions, map[string]interface{}{
			"action": "show",
			"source": "hotkey",
		})
		event.WithContext(ctx)

		// 发布到事件总线
		if err := eventBus.Publish(string(EventTypeHotkeyShowSuggestions), *event); err != nil {
			logger.Error("发布快捷键事件失败",
				zap.String("event_type", string(EventTypeHotkeyShowSuggestions)),
				zap.Error(err),
			)
		}

		logger.Info("✅ 自动化建议事件已发布",
			zap.String("event_type", string(EventTypeHotkeyShowSuggestions)),
			zap.String("current_app", ctx.Application),
		)
	}
}

// createShowKeybindingsHandler 创建显示快捷键列表处理函数
//
// 功能：显示所有可用快捷键
// 发布事件：EventTypeHotkeyShowKeybindings
func createShowKeybindingsHandler(eventBus *events.EventBus) HotkeyCallback {
	return func(reg *HotkeyRegistration, ctx *events.EventContext) {
		logger.Info("⌨️  快捷键触发: 显示快捷键列表",
			zap.String("hotkey", reg.Hotkey.String()),
		)

		// 发布快捷键事件
		event := events.NewEvent(EventTypeHotkeyShowKeybindings, map[string]interface{}{
			"action": "show",
			"source": "hotkey",
		})
		event.WithContext(ctx)

		// 发布到事件总线
		if err := eventBus.Publish(string(EventTypeHotkeyShowKeybindings), *event); err != nil {
			logger.Error("发布快捷键事件失败",
				zap.String("event_type", string(EventTypeHotkeyShowKeybindings)),
				zap.Error(err),
			)
		}

		logger.Info("✅ 快捷键列表事件已发布",
			zap.String("event_type", string(EventTypeHotkeyShowKeybindings)),
		)
	}
}

// createToggleMonitoringHandler 创建切换监控状态处理函数
//
// 功能：暂停或恢复工作流监控
// 发布事件：EventTypeHotkeyToggleMonitoring
func createToggleMonitoringHandler(eventBus *events.EventBus) HotkeyCallback {
	return func(reg *HotkeyRegistration, ctx *events.EventContext) {
		logger.Info("⏯️  快捷键触发: 切换监控状态",
			zap.String("hotkey", reg.Hotkey.String()),
		)

		// 发布快捷键事件
		event := events.NewEvent(EventTypeHotkeyToggleMonitoring, map[string]interface{}{
			"action": "toggle",
			"source": "hotkey",
		})
		event.WithContext(ctx)

		// 发布到事件总线
		if err := eventBus.Publish(string(EventTypeHotkeyToggleMonitoring), *event); err != nil {
			logger.Error("发布快捷键事件失败",
				zap.String("event_type", string(EventTypeHotkeyToggleMonitoring)),
				zap.Error(err),
			)
		}

		logger.Info("✅ 切换监控状态事件已发布",
			zap.String("event_type", string(EventTypeHotkeyToggleMonitoring)),
		)
	}
}

// createShowStatusHandler 创建显示状态处理函数
//
// 功能：显示 FlowMind 当前状态和统计信息
// 发布事件：EventTypeHotkeyShowStatus
func createShowStatusHandler(eventBus *events.EventBus) HotkeyCallback {
	return func(reg *HotkeyRegistration, ctx *events.EventContext) {
		logger.Info("📊 快捷键触发: 显示状态信息",
			zap.String("hotkey", reg.Hotkey.String()),
		)

		// 发布快捷键事件
		event := events.NewEvent(EventTypeHotkeyShowStatus, map[string]interface{}{
			"action": "show",
			"source": "hotkey",
		})
		event.WithContext(ctx)

		// 发布到事件总线
		if err := eventBus.Publish(string(EventTypeHotkeyShowStatus), *event); err != nil {
			logger.Error("发布快捷键事件失败",
				zap.String("event_type", string(EventTypeHotkeyShowStatus)),
				zap.Error(err),
			)
		}

		logger.Info("✅ 显示状态信息事件已发布",
			zap.String("event_type", string(EventTypeHotkeyShowStatus)),
		)
	}
}
