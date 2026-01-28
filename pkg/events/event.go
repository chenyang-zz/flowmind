/**
 * Package events 提供事件系统的核心类型定义
 *
 * 事件系统是 FlowMind 的核心通信机制，用于：
 * - 监控器发布事件
 * - 服务订阅和处理事件
 * - 前端接收实时更新
 */

package events

import (
	"time"

	"github.com/google/uuid"
)

/**
 * EventType 事件类型枚举
 */
type EventType string

/**
 * 所有事件类型常量
 */
const (
	// 监控事件
	EventTypeKeyboard   EventType = "keyboard"    // 键盘事件
	EventTypeClipboard  EventType = "clipboard"   // 剪贴板事件
	EventTypeAppSwitch  EventType = "app_switch"  // 应用切换事件
	EventTypeAppSession EventType = "app_session" // 应用会话事件
	EventTypeFileSystem EventType = "file_system" // 文件系统事件

	// 系统事件
	EventTypeError      EventType = "error"       // 错误事件
	EventTypePermission EventType = "permission"  // 权限事件
	EventTypeStatus     EventType = "status"      // 状态事件
)

/**
 * Event 统一事件结构
 *
 * 所有监控器和系统事件都使用此结构
 */
type Event struct {
	// ID 事件唯一标识符
	ID string `json:"id"`

	// Type 事件类型
	Type EventType `json:"type"`

	// Timestamp 事件发生时间
	Timestamp time.Time `json:"timestamp"`

	// Data 事件数据（类型特定的数据）
	Data map[string]interface{} `json:"data"`

	// Metadata 事件元数据（可选的额外信息）
	Metadata map[string]string `json:"metadata,omitempty"`

	// Context 事件上下文信息（捕获事件时的环境）
	Context *EventContext `json:"context,omitempty"`
}

/**
 * EventContext 事件上下文
 *
 * 描述事件发生时的应用环境
 */
type EventContext struct {
	// Application 当前活动应用名称
	Application string `json:"application,omitempty"`

	// BundleID 应用 Bundle ID（macOS）
	BundleID string `json:"bundle_id,omitempty"`

	// WindowTitle 当前窗口标题
	WindowTitle string `json:"window_title,omitempty"`

	// FilePath 相关文件路径（如适用）
	FilePath string `json:"file_path,omitempty"`

	// Selection 选中文本（如适用）
	Selection string `json:"selection,omitempty"`
}

/**
 * NewEvent 创建新事件
 *
 * Parameters:
 *   - eventType: 事件类型
 *   - data: 事件数据
 *
 * Returns:
 *   - *Event: 新创建的事件
 */
func NewEvent(eventType EventType, data map[string]interface{}) *Event {
	return &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
		Metadata:  make(map[string]string),
	}
}

/**
 * WithContext 设置事件上下文
 *
 * Parameters:
 *   - context: 事件上下文
 *
 * Returns:
 *   - *Event: 返回自身，支持链式调用
 */
func (e *Event) WithContext(context *EventContext) *Event {
	e.Context = context
	return e
}

/**
 * WithMetadata 添加元数据
 *
 * Parameters:
 *   - key: 元数据键
 *   - value: 元数据值
 *
 * Returns:
 *   - *Event: 返回自身，支持链式调用
 */
func (e *Event) WithMetadata(key, value string) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

/**
 * generateEventID 生成事件唯一 ID
 *
 * 使用 UUID v4 确保全局唯一性
 *
 * Returns:
 *   - string: UUID 字符串
 */
func generateEventID() string {
	return uuid.New().String()
}

/**
 * KeyboardEventData 键盘事件数据
 */
type KeyboardEventData struct {
	KeyCode   int    `json:"keycode"`   // 按键代码
	Modifiers uint64 `json:"modifiers"` // 修饰键标志
}

/**
 * ClipboardEventData 剪贴板事件数据
 */
type ClipboardEventData struct {
	Content string `json:"content"` // 剪贴板内容
	Length  int    `json:"length"`  // 内容长度
}

/**
 * AppSwitchEventData 应用切换事件数据
 */
type AppSwitchEventData struct {
	From      string `json:"from"`      // 切换前的应用
	To        string `json:"to"`        // 切换后的应用
	BundleID  string `json:"bundle_id"` // 目标应用 Bundle ID
	Window    string `json:"window"`    // 窗口标题
}

/**
 * AppSessionEventData 应用会话数据
 */
type AppSessionEventData struct {
	AppName  string    `json:"app_name"`  // 应用名称
	BundleID string    `json:"bundle_id"` // Bundle ID
	Start    time.Time `json:"start"`     // 会话开始时间
	End      time.Time `json:"end"`       // 会话结束时间
	Duration float64   `json:"duration"`  // 时长（秒）
}

/**
 * FileSystemEventData 文件系统事件数据
 */
type FileSystemEventData struct {
	Path      string `json:"path"`       // 文件路径
	Operation string `json:"operation"`  // 操作类型
	IsCreate  bool   `json:"is_create"`  // 是否创建
	IsWrite   bool   `json:"is_write"`   // 是否写入
	IsRemove  bool   `json:"is_remove"`  // 是否删除
	IsRename  bool   `json:"is_rename"`  // 是否重命名
}
