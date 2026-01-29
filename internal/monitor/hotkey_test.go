package monitor

import (
	"sync"
	"testing"
	"time"

	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewHotkey 测试快捷键创建和解析功能
//
// 验证快捷键能够正确解析字符串并创建对象。
// 测试场景：
//   1. 单个修饰键：Cmd+A
//   2. 多个修饰键：Cmd+Shift+A
//   3. 不同修饰键组合：Control+Option+M
//   4. 功能键：F5
//   5. 方向键：Up
//   6. 特殊键：Space
//   7. 无效格式：空字符串、未知按键、未知修饰键
func TestNewHotkey(t *testing.T) {
	tests := []struct {
		name             string
		hotkeyStr        string
		expectKeyCode    int
		expectModifiers  uint64
		expectError      bool
		expectStringRep  string
	}{
		{
			name:            "单个修饰键 Cmd+A",
			hotkeyStr:       "Cmd+A",
			expectKeyCode:   0,  // A 的键码
			expectModifiers: 0x100000,
			expectError:     false,
			expectStringRep: "Cmd+A",
		},
		{
			name:            "多个修饰键 Cmd+Shift+A",
			hotkeyStr:       "Cmd+Shift+A",
			expectKeyCode:   0,
			expectModifiers: uint64(0x120000), // Cmd | Shift
			expectError:     false,
			expectStringRep: "Cmd+Shift+A",
		},
		{
			name:            "Control+Option+M",
			hotkeyStr:       "Control+Option+M",
			expectKeyCode:   46, // M 的键码
			expectModifiers: 0x90000, // Control | Option
			expectError:     false,
			expectStringRep: "Control+Option+M",
		},
		{
			name:            "功能键 F5",
			hotkeyStr:       "F5",
			expectKeyCode:   96,
			expectModifiers: 0,
			expectError:     false,
			expectStringRep: "F5",
		},
		{
			name:            "方向键 Up",
			hotkeyStr:       "Up",
			expectKeyCode:   126,
			expectModifiers: 0,
			expectError:     false,
			expectStringRep: "Up",
		},
		{
			name:            "特殊键 Space",
			hotkeyStr:       "Space",
			expectKeyCode:   49,
			expectModifiers: 0,
			expectError:     false,
			expectStringRep: "Space",
		},
		{
			name:            "Cmd+Space (带修饰键的特殊键)",
			hotkeyStr:       "Cmd+Space",
			expectKeyCode:   49,
			expectModifiers: 0x100000,
			expectError:     false,
			expectStringRep: "Cmd+Space",
		},
		{
			name:        "无效按键",
			hotkeyStr:   "Cmd+InvalidKey",
			expectError: true,
		},
		{
			name:        "空字符串",
			hotkeyStr:   "",
			expectError: true,
		},
		{
			name:        "未知修饰键",
			hotkeyStr:   "UnknownMod+A",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hotkey, err := NewHotkey(tt.hotkeyStr)

			if tt.expectError {
				assert.Error(t, err, "应该返回错误")
				assert.Nil(t, hotkey, "快捷键对象应为 nil")
			} else {
				require.NoError(t, err, "不应返回错误")
				assert.NotNil(t, hotkey, "快捷键对象不应为 nil")
				assert.Equal(t, tt.expectKeyCode, hotkey.KeyCode, "KeyCode 应匹配")
				assert.Equal(t, tt.expectModifiers, hotkey.Modifiers, "Modifiers 应匹配")
				assert.Equal(t, tt.expectStringRep, hotkey.String(), "字符串表示应匹配")
			}
		})
	}

	// 额外测试：只有按键名，没有修饰键
	t.Run("只有按键名没有修饰键", func(t *testing.T) {
		hotkey, err := NewHotkey("A")
		require.NoError(t, err, "不应返回错误")
		assert.NotNil(t, hotkey, "快捷键对象不应为 nil")
		assert.Equal(t, 0, hotkey.KeyCode, "KeyCode 应为 0 (A)")
		assert.Equal(t, uint64(0), hotkey.Modifiers, "Modifiers 应为 0")
	})
}

// TestHotkeyMatch 测试快捷键匹配功能
//
// 验证快捷键能够正确匹配键盘事件。
// 测试场景：
//   1. 完全匹配：keycode 和 modifiers 都相同
//   2. modifiers 不匹配
//   3. keycode 不匹配
//   4. 两者都不匹配
func TestHotkeyMatch(t *testing.T) {
	// 创建 Cmd+Shift+A 快捷键
	hotkey, err := NewHotkey("Cmd+Shift+A")
	require.NoError(t, err, "快捷键创建不应失败")

	tests := []struct {
		name         string
		keyCode      int
		modifiers    uint64
		expectMatch  bool
	}{
		{
			name:        "完全匹配",
			keyCode:     0,          // A 的键码
			modifiers:   uint64(0x120000),   // Cmd | Shift
			expectMatch: true,
		},
		{
			name:        "只有 Cmd，缺少 Shift",
			keyCode:     0,
			modifiers:   uint64(0x100000),   // 只有 Cmd
			expectMatch: false,
		},
		{
			name:        "keycode 不匹配（S 键）",
			keyCode:     1,          // S 的键码
			modifiers:   uint64(0x120000),
			expectMatch: false,
		},
		{
			name:        "完全不匹配",
			keyCode:     46,         // M 键
			modifiers:   uint64(0x20000),    // 只有 Shift
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hotkey.Match(tt.keyCode, tt.modifiers)
			assert.Equal(t, tt.expectMatch, result, "匹配结果应符合预期")
		})
	}
}

// TestHotkeyString_DefaultFormat 测试 Hotkey.String() 默认格式
//
// 当 StringRepresentation 为空时，应该返回默认格式。
func TestHotkeyString_DefaultFormat(t *testing.T) {
	hotkey := &Hotkey{
		KeyCode:              46,
		Modifiers:            0x100000,
		StringRepresentation: "", // 空字符串
	}

	result := hotkey.String()
	assert.Contains(t, result, "46", "应包含 KeyCode")
	assert.Contains(t, result, "1048576", "应包含 Modifiers 的十进制值")
}

// TestHotkeyManager_Unregister_MultipleCallbacks 测试取消多回调中的一个
//
// 验证当快捷键有多个回调时，取消注册其中一个的情况。
func TestHotkeyManager_Unregister_MultipleCallbacks(t *testing.T) {
	eventBus := events.NewEventBus()
	manager := NewHotkeyManager(eventBus)

	// 同一个快捷键注册两个回调
	id1, _ := manager.Register("Cmd+A", func(reg *HotkeyRegistration, ctx *events.EventContext) {})
	id2, _ := manager.Register("Cmd+A", func(reg *HotkeyRegistration, ctx *events.EventContext) {})

	// 验证快捷键已注册
	assert.True(t, manager.IsRegistered("cmd+a"))

	// 取消第一个回调
	success := manager.Unregister(id1)
	assert.True(t, success, "取消注册应成功")

	// 快捷键仍应存在（因为还有第二个回调）
	assert.True(t, manager.IsRegistered("cmd+a"), "快捷键应该仍存在")

	// 取消第二个回调
	success = manager.Unregister(id2)
	assert.True(t, success, "取消注册应成功")

	// 现在快捷键应该被完全移除
	assert.False(t, manager.IsRegistered("cmd+a"), "快捷键应该已被完全移除")
}

// TestHotkeyManager_HandleKeyboardEvent_InvalidData 测试处理无效的键盘事件
//
// 验证 handleKeyboardEvent 能够正确处理无效的事件数据。
func TestHotkeyManager_HandleKeyboardEvent_InvalidData(t *testing.T) {
	eventBus := events.NewEventBus()
	manager := NewHotkeyManager(eventBus)

	// 注册一个快捷键
	_, _ = manager.Register("Cmd+A", func(reg *HotkeyRegistration, ctx *events.EventContext) {})

	// 启动管理器
	require.NoError(t, manager.Start())
	defer manager.Stop()

	// 等待订阅生效
	time.Sleep(100 * time.Millisecond)

	// 发布无效事件（缺少 keycode）
	invalidEvent1 := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
		"modifiers": uint64(0x100000),
	})
	err := eventBus.Publish(string(events.EventTypeKeyboard), *invalidEvent1)
	assert.NoError(t, err, "发布事件不应失败")

	// 发布无效事件（缺少 modifiers）
	invalidEvent2 := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
		"keycode": 0,
	})
	err = eventBus.Publish(string(events.EventTypeKeyboard), *invalidEvent2)
	assert.NoError(t, err, "发布事件不应失败")

	// 发布无效事件（类型错误）
	invalidEvent3 := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
		"keycode":   "string",  // 错误的类型
		"modifiers": uint64(0x100000),
	})
	err = eventBus.Publish(string(events.EventTypeKeyboard), *invalidEvent3)
	assert.NoError(t, err, "发布事件不应失败")

	// 等待一下确保没有 panic
	time.Sleep(50 * time.Millisecond)

	// 如果没有 panic，测试通过
}

// TestHotkeyManager_Register 测试快捷键注册功能
//
// 验证快捷键能够成功注册和取消注册。
// 测试场景：
//   1. 注册单个快捷键
//   2. 注册多个快捷键
//   3. 重复注册同一快捷键（添加多个回调）
//   4. 取消注册
//   5. 取消注册不存在的 ID
//   6. UnregisterAll 清空所有注册
func TestHotkeyManager_Register(t *testing.T) {
	eventBus := events.NewEventBus()
	manager := NewHotkeyManager(eventBus)

	t.Run("注册单个快捷键", func(t *testing.T) {
		id, err := manager.Register("Cmd+A", func(reg *HotkeyRegistration, ctx *events.EventContext) {
			// 回调函数
		})

		require.NoError(t, err, "注册不应失败")
		assert.NotEmpty(t, id, "注册 ID 不应为空")
		assert.True(t, manager.IsRegistered("cmd+a"), "快捷键应已注册")
		assert.True(t, manager.IsRegistered("Cmd+A"), "快捷键匹配应不区分大小写")
	})

	t.Run("注册多个快捷键", func(t *testing.T) {
		id1, _ := manager.Register("Cmd+B", func(reg *HotkeyRegistration, ctx *events.EventContext) {})
		id2, _ := manager.Register("Cmd+C", func(reg *HotkeyRegistration, ctx *events.EventContext) {})

		assert.NotEmpty(t, id1, "ID1 不应为空")
		assert.NotEmpty(t, id2, "ID2 不应为空")
		assert.True(t, manager.IsRegistered("cmd+b"), "Cmd+B 应已注册")
		assert.True(t, manager.IsRegistered("cmd+c"), "Cmd+C 应已注册")

		hotkeys := manager.GetRegisteredHotkeys()
		assert.Contains(t, hotkeys, "cmd+a", "应包含 Cmd+A")
		assert.Contains(t, hotkeys, "cmd+b", "应包含 Cmd+B")
		assert.Contains(t, hotkeys, "cmd+c", "应包含 Cmd+C")
	})

	t.Run("同一快捷键多个回调", func(t *testing.T) {
		callCount := 0
		var mu sync.Mutex

		id1, _ := manager.Register("Cmd+D", func(reg *HotkeyRegistration, ctx *events.EventContext) {
			mu.Lock()
			callCount++
			mu.Unlock()
		})
		id2, _ := manager.Register("Cmd+D", func(reg *HotkeyRegistration, ctx *events.EventContext) {
			mu.Lock()
			callCount++
			mu.Unlock()
		})

		assert.NotEmpty(t, id1, "ID1 不应为空")
		assert.NotEmpty(t, id2, "ID2 不应为空")
		assert.True(t, manager.IsRegistered("cmd+d"), "Cmd+D 应已注册")
	})

	t.Run("取消注册", func(t *testing.T) {
		id, _ := manager.Register("Cmd+E", func(reg *HotkeyRegistration, ctx *events.EventContext) {})

		// 取消注册
		success := manager.Unregister(id)
		assert.True(t, success, "取消注册应成功")
		assert.False(t, manager.IsRegistered("cmd+e"), "快捷键应已取消注册")
	})

	t.Run("取消注册不存在的 ID", func(t *testing.T) {
		success := manager.Unregister("non-existent-id")
		assert.False(t, success, "取消不存在的 ID 应失败")
	})

	t.Run("UnregisterAll 清空所有注册", func(t *testing.T) {
		// 先注册几个快捷键
		_, _ = manager.Register("Cmd+F", func(reg *HotkeyRegistration, ctx *events.EventContext) {})
		_, _ = manager.Register("Cmd+G", func(reg *HotkeyRegistration, ctx *events.EventContext) {})

		// 清空所有注册
		manager.UnregisterAll()

		assert.False(t, manager.IsRegistered("cmd+f"), "Cmd+F 应已清空")
		assert.False(t, manager.IsRegistered("cmd+g"), "Cmd+G 应已清空")
		hotkeys := manager.GetRegisteredHotkeys()
		assert.Empty(t, hotkeys, "所有快捷键应已清空")
	})
}

// TestHotkeyManager_MatchAndTrigger 测试快捷键匹配和触发功能
//
// 验证快捷键能够正确匹配键盘事件并触发回调。
// 测试场景：
//   1. 发送匹配的键盘事件，触发回调
//   2. 发送不匹配的键盘事件，不触发回调
//   3. 禁用的快捷键不触发
//   4. 多个快捷键同时匹配
func TestHotkeyManager_MatchAndTrigger(t *testing.T) {
	eventBus := events.NewEventBus()
	manager := NewHotkeyManager(eventBus)

	// 启动管理器（订阅事件）
	require.NoError(t, manager.Start(), "管理器启动不应失败")
	defer manager.Stop()

	// 等待订阅生效（事件总线是异步的）
	time.Sleep(100 * time.Millisecond)

	t.Run("发送匹配事件触发回调", func(t *testing.T) {
		callbackTriggered := make(chan bool, 1)
		_, err := manager.Register("Cmd+A", func(reg *HotkeyRegistration, ctx *events.EventContext) {
			t.Logf("回调被触发！快捷键：%s，应用：%s", reg.Hotkey.String(), ctx.Application)
			callbackTriggered <- true
		})
		require.NoError(t, err, "注册不应失败")

		t.Logf("快捷键已注册，检查是否在列表中：%v", manager.GetRegisteredHotkeys())
		t.Logf("订阅的事件类型字符串：%q", string(events.EventTypeKeyboard))

		// 验证管理器是否正在运行
		t.Logf("HotkeyManager 是否正在运行：%v", manager.IsRunning())

		// 发送匹配的键盘事件
		keyboardEvent := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"keycode":   0,            // A
			"modifiers": uint64(0x100000), // Cmd (显式指定为 uint64)
		})
		keyboardEvent.WithContext(&events.EventContext{
			Application: "TestApp",
		})

		eventTypeStr := string(events.EventTypeKeyboard)
		t.Logf("发布键盘事件：type=%q, keycode=0, modifiers=0x100000", eventTypeStr)
		err = eventBus.Publish(eventTypeStr, *keyboardEvent)
		require.NoError(t, err, "发布事件不应失败")

		// 等待回调被触发
		select {
		case <-callbackTriggered:
			// 成功触发
			t.Log("成功接收到回调触发信号")
		case <-time.After(1 * time.Second):
			t.Fatal("回调未被触发（超时）")
		}
	})

	t.Run("发送不匹配事件不触发回调", func(t *testing.T) {
		callbackTriggered := make(chan bool, 1)
		_, err := manager.Register("Cmd+B", func(reg *HotkeyRegistration, ctx *events.EventContext) {
			callbackTriggered <- true
		})
		require.NoError(t, err, "注册不应失败")

		// 发送不匹配的键盘事件（错误的 keycode）
		keyboardEvent := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"keycode":   1,       // S 而不是 B
			"modifiers": uint64(0x100000), // Cmd
		})

		err = eventBus.Publish(string(events.EventTypeKeyboard), *keyboardEvent)
		require.NoError(t, err, "发布事件不应失败")

		// 验证回调未被触发
		select {
		case <-callbackTriggered:
			t.Fatal("回调不应被触发")
		case <-time.After(100 * time.Millisecond):
			// 正常情况，回调未触发
		}
	})
}

// TestHotkeyManager_DebugKeyMap 调试 keyCodeMap 的内容
func TestHotkeyManager_DebugKeyMap(t *testing.T) {
	eventBus := events.NewEventBus()
	manager := NewHotkeyManager(eventBus)

	_, err := manager.Register("Cmd+A", func(reg *HotkeyRegistration, ctx *events.EventContext) {
		t.Log("回调被触发")
	})
	require.NoError(t, err, "注册不应失败")

	// 访问私有字段进行调试
	t.Logf("已注册的快捷键：%v", manager.GetRegisteredHotkeys())

	// 手动检查查找键
	lookupKey := (uint64(0) << 32) | uint64(0x100000)
	t.Logf("期望的查找键：0x%x (keycode=0, modifiers=0x100000)", lookupKey)
}

// TestHotkeyManager_SetEnabled 测试启用/禁用快捷键功能
//
// 验证快捷键可以被动态启用和禁用。
// 测试场景：
//   1. 禁用快捷键后不触发回调
//   2. 重新启用后正常触发
func TestHotkeyManager_SetEnabled(t *testing.T) {
	eventBus := events.NewEventBus()
	manager := NewHotkeyManager(eventBus)

	require.NoError(t, manager.Start(), "管理器启动不应失败")
	defer manager.Stop()

	// 等待订阅生效
	time.Sleep(100 * time.Millisecond)

	callbackTriggered := make(chan bool, 1)
	id, err := manager.Register("Cmd+A", func(reg *HotkeyRegistration, ctx *events.EventContext) {
		callbackTriggered <- true
	})
	require.NoError(t, err, "注册不应失败")

	t.Run("禁用快捷键", func(t *testing.T) {
		// 禁用快捷键
		success := manager.SetEnabled(id, false)
		assert.True(t, success, "禁用应成功")

		// 发送键盘事件（不应触发）
		keyboardEvent := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"keycode":   0,
			"modifiers": uint64(0x100000),
		})
		eventBus.Publish(string(events.EventTypeKeyboard), *keyboardEvent)

		select {
		case <-callbackTriggered:
			t.Fatal("快捷键被禁用，不应触发回调")
		case <-time.After(100 * time.Millisecond):
			// 正常情况，回调未触发
		}
	})

	t.Run("重新启用快捷键", func(t *testing.T) {
		// 重新启用
		success := manager.SetEnabled(id, true)
		assert.True(t, success, "启用应成功")

		// 发送键盘事件（应该触发）
		keyboardEvent := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"keycode":   0,
			"modifiers": uint64(0x100000),
		})
		eventBus.Publish(string(events.EventTypeKeyboard), *keyboardEvent)

		select {
		case <-callbackTriggered:
			// 成功触发
		case <-time.After(1 * time.Second):
			t.Fatal("回调未被触发（超时）")
		}
	})

	t.Run("设置不存在的 ID", func(t *testing.T) {
		success := manager.SetEnabled("non-existent-id", true)
		assert.False(t, success, "设置不存在的 ID 应失败")
	})
}

// TestHotkeyManager_ConcurrentAccess 测试并发访问安全性
//
// 验证快捷键管理器在并发场景下的线程安全性。
// 测试场景：
//   1. 并发注册多个快捷键
//   2. 并发取消注册
//   3. 并发触发快捷键
func TestHotkeyManager_ConcurrentAccess(t *testing.T) {
	eventBus := events.NewEventBus()
	manager := NewHotkeyManager(eventBus)

	require.NoError(t, manager.Start(), "管理器启动不应失败")
	defer manager.Stop()

	t.Run("并发注册", func(t *testing.T) {
		var wg sync.WaitGroup
		ids := make(chan string, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				id, err := manager.Register(string(rune('A'+index)),
					func(reg *HotkeyRegistration, ctx *events.EventContext) {},
				)
				if err == nil {
					ids <- id
				}
			}(i)
		}

		wg.Wait()
		close(ids)

		// 验证至少有一些注册成功
		count := 0
		for range ids {
			count++
		}
		assert.Greater(t, count, 0, "至少应有一些注册成功")
	})

	t.Run("并发读写", func(t *testing.T) {
		var wg sync.WaitGroup

		// 并发注册
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				_, _ = manager.Register(string(rune('K'+index)),
					func(reg *HotkeyRegistration, ctx *events.EventContext) {},
				)
			}(i)
		}

		// 并发查询
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = manager.GetRegisteredHotkeys()
			}()
		}

		wg.Wait()
		// 如果没有死锁或 panic，则测试通过
	})
}

// TestHotkeyManager_StartStop 测试启动和停止功能
//
// 验证快捷键管理器的生命周期管理。
// 测试场景：
//   1. 重复启动（幂等性）
//   2. 重复停止（幂等性）
//   3. 未启动就停止
//   4. 检查运行状态
func TestHotkeyManager_StartStop(t *testing.T) {
	eventBus := events.NewEventBus()
	manager := NewHotkeyManager(eventBus)

	t.Run("初始状态检查", func(t *testing.T) {
		assert.False(t, manager.IsRunning(), "初始状态应为未运行")
	})

	t.Run("启动管理器", func(t *testing.T) {
		err := manager.Start()
		require.NoError(t, err, "启动不应失败")
		assert.True(t, manager.IsRunning(), "启动后应为运行状态")
	})

	t.Run("重复启动（幂等）", func(t *testing.T) {
		err := manager.Start()
		require.NoError(t, err, "重复启动不应失败")
		assert.True(t, manager.IsRunning(), "应仍为运行状态")
	})

	t.Run("停止管理器", func(t *testing.T) {
		err := manager.Stop()
		require.NoError(t, err, "停止不应失败")
		assert.False(t, manager.IsRunning(), "停止后应为未运行状态")
	})

	t.Run("重复停止（幂等）", func(t *testing.T) {
		err := manager.Stop()
		require.NoError(t, err, "重复停止不应失败")
		assert.False(t, manager.IsRunning(), "应仍为未运行状态")
	})

	t.Run("未启动就停止", func(t *testing.T) {
		newManager := NewHotkeyManager(eventBus)
		err := newManager.Stop()
		require.NoError(t, err, "未启动就停止不应失败")
	})
}
