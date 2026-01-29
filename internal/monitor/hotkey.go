package monitor

import (
	"fmt"
	"strings"
	"sync"

	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/chenyang-zz/flowmind/pkg/logger"
	"go.uber.org/zap"
)

// Modifiers 修饰键标志位常量（macOS CGEventFlags）
//
// 这些标志位用于标识键盘事件中的修饰键状态。
// macOS 的 CGEventFlags 使用位掩码表示多个修饰键的组合。
const (
	// ModifierCommand Command 键（⌘）标志位
	// 值为 0x100000，对应 CGEventFlagMaskCommand
	ModifierCommand uint64 = 0x100000

	// ModifierShift Shift 键（⇧）标志位
	// 值为 0x20000，对应 CGEventFlagMaskShift
	ModifierShift uint64 = 0x20000

	// ModifierControl Control 键（⌃）标志位
	// 值为 0x10000，对应 CGEventFlagMaskControl
	ModifierControl uint64 = 0x10000

	// ModifierOption Option 键（⌥）标志位
	// 值为 0x80000，对应 CGEventFlagMaskAlternate
	ModifierOption uint64 = 0x80000

	// ModifierCapsLock Caps Lock 键标志位
	// 值为 0x10000 的组合位
	ModifierCapsLock uint64 = 0x10000

	// ModifierFn Fn 键标志位
	// 值为 0x800000
	ModifierFn uint64 = 0x800000
)

// keyNameToKeyCode 按键名称到键码的映射表
//
// 将人类可读的按键名称映射到 macOS 虚拟键码。
// 支持：
//   - 字母 A-Z: keycodes 0-25
//   - 数字 0-9: keycodes 29-38, 45（0）
//   - 功能键: F1-F24
//   - 方向键: Up, Down, Left, Right
//   - 特殊键: Space, Enter, Tab, Escape, Delete, Backspace, Home, End, PageUp, PageDown
var keyNameToKeyCode = map[string]int{
	// 字母键
	"a": 0, "b": 11, "c": 8, "d": 2, "e": 14, "f": 3, "g": 5, "h": 4,
	"i": 34, "j": 38, "k": 40, "l": 37, "m": 46, "n": 45, "o": 31, "p": 35,
	"q": 12, "r": 15, "s": 1, "t": 17, "u": 32, "v": 9, "w": 13, "x": 7,
	"y": 16, "z": 6,

	// 数字键
	"0": 29, "1": 18, "2": 19, "3": 20, "4": 21, "5": 23, "6": 22,
	"7": 26, "8": 25, "9": 28,

	// 功能键 F1-F24
	"f1": 122, "f2": 120, "f3": 99, "f4": 118, "f5": 96, "f6": 97,
	"f7": 98, "f8": 100, "f9": 101, "f10": 109, "f11": 103, "f12": 111,
	"f13": 105, "f14": 107, "f15": 113, "f16": 106, "f17": 64, "f18": 79,
	"f19": 80, "f20": 90, "f21": 91, "f22": 92, "f23": 93, "f24": 94,

	// 方向键
	"up":     126, // 上箭头
	"down":   125, // 下箭头
	"left":   123, // 左箭头
	"right":  124, // 右箭头

	// 特殊键
	"space":      49, // 空格键
	"enter":      36, // 回车键
	"return":     36, // 回车键（别名）
	"tab":        48, // Tab 键
	"escape":     53, // Esc 键
	"esc":        53, // Esc 键（别名）
	"delete":     51, // Delete 键（向前删除）
	"backspace":  51, // Backspace 键（向后删除）
	"home":       115, // Home 键
	"end":        119, // End 键
	"pageup":     116, // Page Up 键
	"pagedown":   117, // Page Down 键
	"help":       114, // Help 键
	"forward":    124, // Forward Delete 键
}

// modifierNameToFlag 修饰键名称到标志位的映射表
//
// 支持的修饰键名称（不区分大小写）：
//   - Cmd, Command
//   - Shift
//   - Ctrl, Control
//   - Opt, Option, Alt
var modifierNameToFlag = map[string]uint64{
	"cmd":      ModifierCommand,
	"command":  ModifierCommand,
	"shift":    ModifierShift,
	"ctrl":     ModifierControl,
	"control":  ModifierControl,
	"opt":      ModifierOption,
	"option":   ModifierOption,
	"alt":      ModifierOption,
}

// Hotkey 快捷键定义
//
// Hotkey 表示一个具体的快捷键组合，包含按键代码和修饰键状态。
// 支持字符串格式解析，如 "Cmd+Shift+A"。
type Hotkey struct {
	// KeyCode 按键代码，对应 macOS 虚拟键码
	// 常见键码：0=A, 1=S, 2=D, ..., 8=C, 46=M, 55=Cmd(左)
	KeyCode int

	// Modifiers 修饰键标志位组合
	// 使用位运算组合多个修饰键，如 ModifierCommand | ModifierShift
	Modifiers uint64

	// StringRepresentation 快捷键的字符串表示
	// 用于调试和日志记录，格式如 "Cmd+Shift+A"
	StringRepresentation string
}

// NewHotkey 从字符串创建快捷键
//
// 解析快捷键字符串并创建 Hotkey 对象。
//
// Parameters:
//   - s: 快捷键字符串，格式如 "Cmd+C", "Cmd+Shift+A", "Control+Option+M"
//     支持的修饰键名称（不区分大小写）：
//     - Cmd, Command
//     - Shift
//     - Ctrl, Control
//     - Opt, Option, Alt
//
// Returns:
//   - *Hotkey: 快捷键对象
//   - error: 解析失败时返回错误（如格式错误、未知按键）
//
// 示例：
//   hotkey, err := NewHotkey("Cmd+Shift+A")
//   hotkey, err := NewHotkey("Control+C")
func NewHotkey(s string) (*Hotkey, error) {
	if s == "" {
		return nil, fmt.Errorf("快捷键字符串不能为空")
	}

	keyCode, modifiers, err := parseHotkeyString(s)
	if err != nil {
		return nil, err
	}

	return &Hotkey{
		KeyCode:              keyCode,
		Modifiers:            modifiers,
		StringRepresentation: s,
	}, nil
}

// parseHotkeyString 解析快捷键字符串
//
// 支持的格式：
//   - "Cmd+A" -> KeyCode=0, Modifiers=0x100000
//   - "Cmd+Shift+A" -> KeyCode=0, Modifiers=0x120000
//   - "Control+Option+M" -> KeyCode=46, Modifiers=0x90000
//
// 解析步骤：
//   1. 将字符串按 "+" 分割
//   2. 最后一个部分是按键名称（如 "A", "M"）
//   3. 前面的部分是修饰键名称（不区分大小写）
//   4. 将修饰键名称映射到标志位常量
//   5. 将按键名称映射到键码（使用预定义的映射表）
//
// Parameters:
//   - s: 快捷键字符串
//
// Returns:
//   - keyCode: 按键代码
//   - modifiers: 修饰键标志位
//   - err: 解析失败时的错误信息
func parseHotkeyString(s string) (keyCode int, modifiers uint64, err error) {
	// 按 "+" 分割字符串
	parts := strings.Split(s, "+")
	if len(parts) == 0 {
		return 0, 0, fmt.Errorf("无效的快捷键格式：%s", s)
	}

	// 初始化修饰键
	modifiers = 0

	// 处理除最后一个部分外的所有部分（修饰键）
	for i := 0; i < len(parts)-1; i++ {
		modifierName := strings.TrimSpace(strings.ToLower(parts[i]))
		flag, ok := modifierNameToFlag[modifierName]
		if !ok {
			return 0, 0, fmt.Errorf("未知的修饰键：%s", parts[i])
		}
		modifiers |= flag
	}

	// 处理最后一个部分（按键名称）
	keyName := strings.TrimSpace(strings.ToLower(parts[len(parts)-1]))
	code, ok := keyNameToKeyCode[keyName]
	if !ok {
		return 0, 0, fmt.Errorf("未知的按键：%s", parts[len(parts)-1])
	}
	keyCode = code

	return keyCode, modifiers, nil
}

// Match 检查是否匹配给定的键盘事件
//
// 比较快捷键的 KeyCode 和 Modifiers 是否与给定的键盘事件完全匹配。
//
// Parameters:
//   - keyCode: 按键代码
//   - modifiers: 修饰键标志位
//
// Returns:
//   - bool: true 表示匹配，false 表示不匹配
func (h *Hotkey) Match(keyCode int, modifiers uint64) bool {
	return h.KeyCode == keyCode && h.Modifiers == modifiers
}

// String 返回快捷键的字符串表示
//
// Returns:
//   - string: 格式化的快捷键字符串
func (h *Hotkey) String() string {
	if h.StringRepresentation != "" {
		return h.StringRepresentation
	}
	return fmt.Sprintf("KeyCode:%d,Modifiers:%d", h.KeyCode, h.Modifiers)
}

// HotkeyCallback 快捷键回调函数类型
//
// 当快捷键被触发时调用此函数。
// 回调函数在独立的 goroutine 中执行，避免阻塞事件处理流程。
//
// Parameters:
//   - registration: 快捷键注册信息，包含 ID 和快捷键定义
//   - context: 事件上下文，包含当前应用等信息
type HotkeyCallback func(registration *HotkeyRegistration, context *events.EventContext)

// HotkeyRegistration 快捷键注册信息
//
// 表示一个已注册的快捷键及其处理逻辑。
type HotkeyRegistration struct {
	// ID 注册唯一标识符，用于取消注册
	// 格式："hotkey-{timestamp}-{nanos}"
	ID string

	// Hotkey 快捷键定义
	Hotkey *Hotkey

	// Callback 快捷键触发时的回调函数
	// 在独立的 goroutine 中执行，避免阻塞事件处理
	Callback HotkeyCallback

	// Enabled 快捷键是否启用
	// 可以通过 SetEnabled 动态切换
	Enabled bool
}

// HotkeyManager 快捷键管理器
//
// 负责快捷键的注册、匹配和生命周期管理。
//
// 工作流程：
//   1. 通过 Register 方法注册快捷键
//   2. 从事件总线订阅键盘事件
//   3. 收到键盘事件时，遍历所有已注册的快捷键进行匹配
//   4. 匹配成功时触发回调函数
type HotkeyManager struct {
	// registrations 已注册的快捷键映射
	// key: 快捷键的规范化字符串（如 "cmd+shift+a"）
	// value: 该快捷键的注册列表（支持同一快捷键多个回调）
	registrations map[string][]*HotkeyRegistration

	// keyCodeMap 快捷键索引（加速匹配）
	// key: keycode + modifiers 组合（使用位运算构造）
	// value: 注册信息切片的引用
	// 用于快速查找，避免遍历所有注册
	keyCodeMap map[uint64][]*HotkeyRegistration

	// eventBus 事件总线，用于订阅键盘事件
	eventBus *events.EventBus

	// subscription 事件总线的订阅 ID
	// 用于取消订阅
	subscription string

	// mu 读写锁，保护并发访问
	mu sync.RWMutex

	// isRunning 管理器运行状态标志
	isRunning bool
}

// NewHotkeyManager 创建快捷键管理器
//
// 创建一个新的快捷键管理器实例。
//
// Parameters:
//   - eventBus: 事件总线实例，用于订阅键盘事件
//
// Returns:
//   - *HotkeyManager: 新创建的快捷键管理器
func NewHotkeyManager(eventBus *events.EventBus) *HotkeyManager {
	return &HotkeyManager{
		registrations: make(map[string][]*HotkeyRegistration),
		keyCodeMap:    make(map[uint64][]*HotkeyRegistration),
		eventBus:      eventBus,
	}
}

// Register 注册快捷键
//
// 注册一个新的快捷键及其回调函数。如果快捷键已被注册，会添加到该快捷键的回调列表中。
// 同一快捷键可以有多个回调，触发时会按注册顺序依次调用。
//
// Parameters:
//   - hotkeyStr: 快捷键字符串，格式如 "Cmd+Shift+A"
//   - callback: 快捷键触发时的回调函数
//
// Returns:
//   - string: 注册 ID，用于 Unregister 取消注册
//   - error: 快捷键格式错误或注册失败时返回错误
//
// 示例：
//   id, err := manager.Register("Cmd+Shift+A", func(reg *HotkeyRegistration, ctx *events.EventContext) {
//       fmt.Println("快捷键触发！当前应用：", ctx.Application)
//   })
func (hm *HotkeyManager) Register(hotkeyStr string, callback HotkeyCallback) (string, error) {
	// 解析快捷键
	hotkey, err := NewHotkey(hotkeyStr)
	if err != nil {
		logger.Warn("解析快捷键失败",
			zap.String("hotkey", hotkeyStr),
			zap.Error(err),
		)
		return "", err
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	// 规范化快捷键字符串（用于索引和存储）
	normalizedKey := strings.ToLower(strings.ReplaceAll(hotkeyStr, " ", ""))
	hotkey.StringRepresentation = normalizedKey

	// 创建注册信息
	reg := &HotkeyRegistration{
		ID:       fmt.Sprintf("hotkey-%d-%d", hotkey.KeyCode, hotkey.Modifiers),
		Hotkey:   hotkey,
		Callback: callback,
		Enabled:  true,
	}

	// 添加到注册映射
	hm.registrations[normalizedKey] = append(hm.registrations[normalizedKey], reg)

	// 添加到快速查找索引
	lookupKey := hm.buildLookupKey(hotkey.KeyCode, hotkey.Modifiers)
	hm.keyCodeMap[lookupKey] = append(hm.keyCodeMap[lookupKey], reg)

	logger.Info("注册快捷键",
		zap.String("hotkey", hotkeyStr),
		zap.String("id", reg.ID),
	)

	return reg.ID, nil
}

// Unregister 取消注册快捷键
//
// 根据 ID 取消注册快捷键。如果该快捷键有多个回调，只删除指定的回调。
// 当快捷键的所有回调都被删除后，该快捷键的注册会被完全移除。
//
// Parameters:
//   - registrationID: 注册 ID（由 Register 返回）
//
// Returns:
//   - bool: true 表示成功取消注册，false 表示 ID 不存在
//
// 示例：
//   manager.Unregister(id)
func (hm *HotkeyManager) Unregister(registrationID string) bool {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// 在所有注册中查找匹配的 ID
	for normalizedKey, regs := range hm.registrations {
		for i, reg := range regs {
			if reg.ID == registrationID {
				// 从注册映射中删除
				if len(regs) == 1 {
					// 只有一个注册，删除整个条目
					delete(hm.registrations, normalizedKey)
				} else {
					// 多个注册，删除指定的
					hm.registrations[normalizedKey] = append(regs[:i], regs[i+1:]...)
				}

				// 从快速查找索引中删除
				lookupKey := hm.buildLookupKey(reg.Hotkey.KeyCode, reg.Hotkey.Modifiers)
				indexRegs := hm.keyCodeMap[lookupKey]
				for j, indexReg := range indexRegs {
					if indexReg.ID == registrationID {
						if len(indexRegs) == 1 {
							delete(hm.keyCodeMap, lookupKey)
						} else {
							hm.keyCodeMap[lookupKey] = append(indexRegs[:j], indexRegs[j+1:]...)
						}
						break
					}
				}

				logger.Info("取消注册快捷键",
					zap.String("id", registrationID),
					zap.String("hotkey", normalizedKey),
				)

				return true
			}
		}
	}

	logger.Debug("快捷键注册不存在",
		zap.String("id", registrationID),
	)

	return false
}

// UnregisterAll 取消所有快捷键注册
//
// 清空所有已注册的快捷键。用于清理或重置状态。
func (hm *HotkeyManager) UnregisterAll() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.registrations = make(map[string][]*HotkeyRegistration)
	hm.keyCodeMap = make(map[uint64][]*HotkeyRegistration)
}

// IsRegistered 检查快捷键是否已注册
//
// Parameters:
//   - hotkeyStr: 快捷键字符串
//
// Returns:
//   - bool: true 表示已注册，false 表示未注册
func (hm *HotkeyManager) IsRegistered(hotkeyStr string) bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	normalizedKey := strings.ToLower(strings.ReplaceAll(hotkeyStr, " ", ""))
	_, exists := hm.registrations[normalizedKey]
	return exists
}

// SetEnabled 启用/禁用快捷键
//
// 动态切换快捷键的启用状态。禁用后的快捷键不会被触发，但仍保留注册。
//
// Parameters:
//   - registrationID: 注册 ID
//   - enabled: true 启用，false 禁用
//
// Returns:
//   - bool: true 表示成功，false 表示 ID 不存在
func (hm *HotkeyManager) SetEnabled(registrationID string, enabled bool) bool {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	for _, regs := range hm.registrations {
		for _, reg := range regs {
			if reg.ID == registrationID {
				reg.Enabled = enabled
				return true
			}
		}
	}

	return false
}

// GetRegisteredHotkeys 获取所有已注册的快捷键列表
//
// Returns:
//   - []string: 快捷键字符串列表，格式如 ["Cmd+C", "Cmd+Shift+A"]
func (hm *HotkeyManager) GetRegisteredHotkeys() []string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	hotkeys := make([]string, 0, len(hm.registrations))
	for normalizedKey := range hm.registrations {
		hotkeys = append(hotkeys, normalizedKey)
	}
	return hotkeys
}

// buildLookupKey 构造快速查找键
//
// 使用位运算将 keycode 和 modifiers 组合成一个 uint64 值，用于快速查找。
// 格式：高32位存储 keycode，低32位存储 modifiers。
//
// Parameters:
//   - keyCode: 按键代码
//   - modifiers: 修饰键标志位
//
// Returns:
//   - uint64: 组合后的查找键
func (hm *HotkeyManager) buildLookupKey(keyCode int, modifiers uint64) uint64 {
	return (uint64(keyCode) << 32) | modifiers
}

// Start 启动快捷键管理器
//
// 订阅键盘事件，开始匹配和触发快捷键。
// 如果管理器已经在运行，会幂等地返回成功。
//
// Returns:
//   - error: 启动失败时返回错误
func (hm *HotkeyManager) Start() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.isRunning {
		return nil // 已经在运行
	}

	// 订阅键盘事件
	hm.subscription = hm.eventBus.Subscribe(string(events.EventTypeKeyboard), hm.handleKeyboardEvent)
	hm.isRunning = true

	return nil
}

// Stop 停止快捷键管理器
//
// 取消订阅键盘事件，停止快捷键匹配。
// 不会清空已注册的快捷键，调用 Start 可以重新启动。
//
// Returns:
//   - error: 停止失败时返回错误
func (hm *HotkeyManager) Stop() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if !hm.isRunning {
		return nil // 未运行
	}

	// 取消订阅
	if hm.subscription != "" {
		hm.eventBus.Unsubscribe(hm.subscription)
		hm.subscription = ""
	}

	hm.isRunning = false
	return nil
}

// IsRunning 检查运行状态
//
// Returns:
//   - bool: 监控器是否正在运行
func (hm *HotkeyManager) IsRunning() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.isRunning
}

// handleKeyboardEvent 处理键盘事件
//
// 订阅键盘事件的回调函数，负责：
//   1. 从事件中提取 keycode 和 modifiers
//   2. 构造快速查找键
//   3. 查找匹配的快捷键注册
//   4. 触发所有匹配的快捷键回调
//
// Parameters:
//   - event: 键盘事件
//
// Returns:
//   - error: 处理失败时返回错误
func (hm *HotkeyManager) handleKeyboardEvent(event events.Event) error {
	// 1. 提取 keycode 和 modifiers
	keycode, ok := event.Data["keycode"].(int)
	if !ok {
		return nil // 无效事件，忽略
	}

	modifiers, ok := event.Data["modifiers"].(uint64)
	if !ok {
		return nil // 无效事件，忽略
	}

	// 2. 构造快速查找键（keycode + modifiers 组合）
	lookupKey := hm.buildLookupKey(keycode, modifiers)

	// 3. 查找匹配的注册
	hm.mu.RLock()
	registrations := hm.keyCodeMap[lookupKey]
	hm.mu.RUnlock()

	if len(registrations) == 0 {
		return nil // 没有匹配的快捷键
	}

	// 4. 触发所有匹配的快捷键回调
	for _, reg := range registrations {
		if !reg.Enabled {
			continue // 跳过禁用的快捷键
		}

		logger.Info("快捷键被触发",
			zap.String("hotkey", reg.Hotkey.String()),
			zap.String("id", reg.ID),
		)

		// 在独立的 goroutine 中执行回调，避免阻塞
		go func(r *HotkeyRegistration) {
			defer func() {
				if rec := recover(); rec != nil {
					// 捕获回调中的 panic，防止崩溃
					logger.Error("快捷键回调 panic",
						zap.String("id", r.ID),
						zap.Any("panic", rec),
					)
				}
			}()
			r.Callback(r, event.Context)
		}(reg)
	}

	return nil
}
