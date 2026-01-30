/**
 * Package analyzer 模式识别引擎的分析组件
 *
 * 负责会话划分、事件标准化、模式挖掘等核心功能
 */

package analyzer

import (
	"strings"

	"github.com/chenyang-zz/flowmind/internal/domain/models"
	"github.com/chenyang-zz/flowmind/pkg/events"
)

/**
 * EventNormalizerConfig 标准化器配置
 */
type EventNormalizerConfig struct {
	// GeneralizeKeys 是否泛化按键（将具体按键转为类型）
	GeneralizeKeys bool

	// GeneralizeClipboard 是否泛化剪贴板内容（不记录具体内容）
	GeneralizeClipboard bool

	// IncludeContext 是否包含上下文信息
	IncludeContext bool
}

/**
 * DefaultEventNormalizerConfig 默认配置
 */
func DefaultEventNormalizerConfig() EventNormalizerConfig {
	return EventNormalizerConfig{
		GeneralizeKeys:      true,
		GeneralizeClipboard: true,
		IncludeContext:      true,
	}
}

/**
 * EventNormalizer 事件标准化器
 *
 * 将原始事件转换为标准化的 EventStep，用于模式挖掘
 */
type EventNormalizer struct {
	config EventNormalizerConfig
}

/**
 * NewEventNormalizer 创建事件标准化器
 *
 * Parameters:
 *   - config: 配置（使用 DefaultEventNormalizerConfig() 获取默认配置）
 *
 * Returns: *EventNormalizer - 事件标准化器实例
 */
func NewEventNormalizer(config EventNormalizerConfig) *EventNormalizer {
	return &EventNormalizer{
		config: config,
	}
}

/**
 * NormalizeEvent 标准化单个事件
 *
 * 将原始事件转换为 EventStep
 *
 * Parameters:
 *   - event: 原始事件
 *
 * Returns: *models.EventStep - 标准化后的事件步骤
 */
func (n *EventNormalizer) NormalizeEvent(event events.Event) *models.EventStep {
	step := &models.EventStep{
		Type:   event.Type,
		Action: n.determineAction(event),
	}

	// 添加上下文信息
	if n.config.IncludeContext && event.Context != nil {
		step.Context = &models.StepContext{
			Application: event.Context.Application,
			BundleID:    event.Context.BundleID,
		}

		// 根据事件类型添加模式值
		step.Context.PatternValue = n.extractPatternValue(event)
	}

	return step
}

/**
 * NormalizeEvents 标准化事件列表
 *
 * Parameters:
 *   - eventList: 原始事件列表
 *
 * Returns: []models.EventStep - 标准化后的事件步骤列表
 */
func (n *EventNormalizer) NormalizeEvents(eventList []events.Event) []models.EventStep {
	steps := make([]models.EventStep, len(eventList))
	for i, event := range eventList {
		step := n.NormalizeEvent(event)
		if step != nil {
			steps[i] = *step
		}
	}
	return steps
}

/**
 * determineAction 确定事件动作
 *
 * 根据事件类型和数据提取具体的动作类型
 *
 * Parameters:
 *   - event: 原始事件
 *
 * Returns: string - 动作类型
 */
func (n *EventNormalizer) determineAction(event events.Event) string {
	switch event.Type {
	case events.EventTypeKeyboard:
		return n.normalizeKeyboardAction(event)

	case events.EventTypeClipboard:
		return n.normalizeClipboardAction(event)

	case events.EventTypeAppSwitch:
		return "app_switch"

	case events.EventTypeAppSession:
		return n.normalizeAppSessionAction(event)

	case events.EventTypeFileSystem:
		return n.normalizeFileSystemAction(event)

	default:
		return "unknown"
	}
}

/**
 * normalizeKeyboardAction 标准化键盘事件动作
 *
 * Parameters:
 *   - event: 键盘事件
 *
 * Returns: string - 标准化的动作类型
 */
func (n *EventNormalizer) normalizeKeyboardAction(event events.Event) string {
	if keyCode, ok := event.Data["keycode"].(float64); ok {
		keyCodeInt := int(keyCode)

		// 功能键区域
		if isFunctionKey(keyCodeInt) {
			return "function_key"
		}

		// 修饰键区域
		if isModifierKey(keyCodeInt) {
			return "modifier_key"
		}

		// 特殊控制键
		if isSpecialKey(keyCodeInt) {
			return "special_key"
		}

		// 字母数字键
		return "alphanumeric_key"
	}

	return "keypress"
}

/**
 * normalizeClipboardAction 标准化剪贴板事件动作
 *
 * Parameters:
 *   - event: 剪贴板事件
 *
 * Returns: string - 标准化的动作类型
 */
func (n *EventNormalizer) normalizeClipboardAction(event events.Event) string {
	if operation, ok := event.Data["operation"].(string); ok {
		return "clipboard_" + operation
	}
	return "clipboard_operation"
}

/**
 * normalizeAppSessionAction 标准化应用会话动作
 *
 * Parameters:
 *   - event: 应用会话事件
 *
 * Returns: string - 标准化的动作类型
 */
func (n *EventNormalizer) normalizeAppSessionAction(event events.Event) string {
	if action, ok := event.Data["action"].(string); ok {
		return "app_session_" + action
	}
	return "app_session"
}

/**
 * normalizeFileSystemAction 标准化文件系统动作
 *
 * Parameters:
 *   - event: 文件系统事件
 *
 * Returns: string - 标准化的动作类型
 */
func (n *EventNormalizer) normalizeFileSystemAction(event events.Event) string {
	if operation, ok := event.Data["operation"].(string); ok {
		return "file_" + operation
	}
	return "file_operation"
}

/**
 * extractPatternValue 提取模式值
 *
 * 根据配置提取用于模式匹配的值
 *
 * Parameters:
 *   - event: 原始事件
 *
 * Returns: string - 模式值
 */
func (n *EventNormalizer) extractPatternValue(event events.Event) string {
	switch event.Type {
	case events.EventTypeKeyboard:
		if n.config.GeneralizeKeys {
			return n.determineAction(event)
		}
		// 返回具体的按键码
		if keyCode, ok := event.Data["keycode"].(float64); ok {
			return intToString(int(keyCode))
		}

	case events.EventTypeClipboard:
		if n.config.GeneralizeClipboard {
			return "clipboard_content"
		}
		// 返回内容类型
		if content, ok := event.Data["content"].(string); ok {
			return n.guessContentType(content)
		}

	case events.EventTypeFileSystem:
		if path, ok := event.Data["path"].(string); ok {
			return n.extractFileExtension(path)
		}
	}

	return ""
}

/**
 * guessContentType 猜测剪贴板内容类型
 *
 * Parameters:
 *   - content: 剪贴板内容
 *
 * Returns: string - 内容类型
 */
func (n *EventNormalizer) guessContentType(content string) string {
	// 空内容
	if strings.TrimSpace(content) == "" {
		return "empty"
	}

	// 检查是否为 URL
	if strings.HasPrefix(content, "http://") || strings.HasPrefix(content, "https://") {
		return "url"
	}

	// 检查是否为路径
	if strings.Contains(content, "/") || strings.Contains(content, "\\") {
		return "path"
	}

	// 检查是否为代码
	if strings.Contains(content, "{") || strings.Contains(content, ";") || strings.Contains(content, "=") {
		return "code"
	}

	// 默认为文本
	return "text"
}

/**
 * extractFileExtension 提取文件扩展名
 *
 * Parameters:
 *   - path: 文件路径
 *
 * Returns: string - 扩展名（不含点）
 */
func (n *EventNormalizer) extractFileExtension(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

/**
 * isFunctionKey 判断是否为功能键
 *
 * Parameters:
 *   - keyCode: 按键码
 *
 * Returns: bool - true表示是功能键
 */
func isFunctionKey(keyCode int) bool {
	// F1-F24: 122-145
	return keyCode >= 122 && keyCode <= 145
}

/**
 * isModifierKey 判断是否为修饰键
 *
 * Parameters:
 *   - keyCode: 按键码
 *
 * Returns: bool - true表示是修饰键
 */
func isModifierKey(keyCode int) bool {
	// 常见修饰键码
	modifierKeys := []int{
		54,  // Right Command
		55,  // Left Command
		56,  // Left Shift
		57,  // Left Control
		58,  // Left Option
		59,  // Right Shift
		60,  // Right Control
		61,  // Right Option
		62,  // Right Command
	}

	for _, key := range modifierKeys {
		if keyCode == key {
			return true
		}
	}
	return false
}

/**
 * isSpecialKey 判断是否为特殊键
 *
 * Parameters:
 *   - keyCode: 按键码
 *
 * Returns: bool - true表示是特殊键
 */
func isSpecialKey(keyCode int) bool {
	// 特殊键码
	specialKeys := []int{
		36,  // Return
		48,  // Tab
		49,  // Space
		51,  // Delete
		53,  // Escape
		63,  // Forward Delete
		76,  // Insert
		114, // Help
		115, // Home
		116, // End
		117, // Page Up
		118, // Page Down
		119, // Up Arrow
		120, // Down Arrow
		121, // Left Arrow
		122, // Right Arrow
	}

	for _, key := range specialKeys {
		if keyCode == key {
			return true
		}
	}
	return false
}

/**
 * intToString 整数转字符串辅助函数
 *
 * Parameters:
 *   - v: 整数值
 *
 * Returns: string - 字符串表示
 */
func intToString(v int) string {
	return string(rune(v))
}
