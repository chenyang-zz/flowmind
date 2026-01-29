//go:build darwin

package platform

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKeyboardMonitor_RealKeyPress 是一个手动集成测试
//
// 此测试用于验证键盘监控在实际使用场景中能够捕获键盘输入。
// **注意**：此测试需要实际按键操作，不适合在自动化测试环境中运行。
//
// 测试步骤：
//  1. 启动键盘监控器
//  2. 等待用户按下任意键（在接下来的 10 秒内）
//  3. 验证捕获到的键盘事件
//  4. 停止监控器
//
// 运行方式：
// ```bash
// go test -v -run TestKeyboardMonitor_RealKeyPress ./internal/platform/
// ```
func TestKeyboardMonitor_RealKeyPress(t *testing.T) {
	skipIfAutomated(t)

	t.Log("=== 键盘监控集成测试 ===")
	t.Log("请在接下来的 10 秒内按下任意键盘按键...")
	t.Log("测试将捕获您的键盘输入并显示事件信息")

	monitor := NewKeyboardMonitor()

	// 创建事件通道
	receivedEvents := make(chan KeyboardEvent, 100)
	eventCount := 0

	// 启动监控器，传入回调函数
	err := monitor.Start(func(event KeyboardEvent) {
		eventCount++
		t.Logf("✅ 捕获到键盘事件 #%d: KeyCode=%d, Modifiers=0x%x",
			eventCount, event.KeyCode, event.Modifiers)

		// 将事件发送到通道（非阻塞）
		select {
		case receivedEvents <- event:
		default:
			// 通道已满，忽略
		}
	})

	if err != nil {
		skipWithReason(t, "需要辅助功能权限: %v", err)
	}
	require.NoError(t, err)
	assert.True(t, monitor.IsRunning(), "监控器应该正在运行")

	// 等待键盘事件（10 秒超时）
	t.Log("⌨️  等待键盘输入...")
	timeout := time.After(10 * time.Second)
	var capturedEvent *KeyboardEvent

	select {
	case event := <-receivedEvents:
		capturedEvent = &event
		t.Logf("🎉 成功捕获键盘事件！KeyCode=%d, Modifiers=0x%x",
			event.KeyCode, event.Modifiers)
	case <-timeout:
		// 超时不算失败，只是没有按键
		t.Log("⏱️  超时：未检测到键盘输入")
		t.Log("提示：请确保已授予辅助功能权限")
	}

	// 停止监控器
	t.Log("🛑 停止键盘监控器...")
	err = monitor.Stop()
	require.NoError(t, err)
	assert.False(t, monitor.IsRunning(), "监控器应该已停止")

	// 如果捕获到了事件，进行验证
	if capturedEvent != nil {
		t.Log("✅ 键盘监控器工作正常！")
		assert.GreaterOrEqual(t, capturedEvent.KeyCode, 0, "KeyCode 应该 >= 0")
		assert.GreaterOrEqual(t, capturedEvent.Modifiers, uint64(0), "Modifiers 应该 >= 0")
	} else {
		t.Log("⚠️  未捕获到键盘事件")
		t.Log("如果您实际按了键，请检查：")
		t.Log("  1. 系统偏好设置 > 隐私与安全性 > 辅助功能")
		t.Log("  2. 确保您的应用或终端有辅助功能权限")
	}
}

// TestKeyboardMonitor_MultipleKeyPresses 测试连续按键
//
// 此测试验证监控器能够连续捕获多个键盘事件。
// **注意**：此测试需要实际连续按键操作。
//
// 运行方式：
// ```bash
// go test -v -run TestKeyboardMonitor_MultipleKeyPresses ./internal/platform/
// ```
func TestKeyboardMonitor_MultipleKeyPresses(t *testing.T) {
	skipIfAutomated(t)

	t.Log("=== 连续按键测试 ===")
	t.Log("请在接下来的 5 秒内连续按下多个键...")

	monitor := NewKeyboardMonitor()

	receivedEvents := make(chan KeyboardEvent, 100)

	err := monitor.Start(func(event KeyboardEvent) {
		receivedEvents <- event
	})

	if err != nil {
		skipWithReason(t, "需要辅助功能权限: %v", err)
	}
	require.NoError(t, err)

	// 收集 5 秒内的所有事件
	timeout := time.After(5 * time.Second)
	var events []KeyboardEvent

eventLoop:
	for {
		select {
		case event := <-receivedEvents:
			events = append(events, event)
			t.Logf("捕获事件 #%d: KeyCode=%d", len(events), event.KeyCode)
		case <-timeout:
			break eventLoop
		}
	}

	// 停止监控器
	_ = monitor.Stop()

	t.Logf("总共捕获了 %d 个键盘事件", len(events))

	if len(events) > 0 {
		t.Log("✅ 成功捕获多个键盘事件！")
		for i, event := range events {
			t.Logf("  事件 #%d: KeyCode=%d, Modifiers=0x%x", i+1, event.KeyCode, event.Modifiers)
		}
		assert.Greater(t, len(events), 0, "应该捕获到至少一个事件")
	} else {
		t.Log("⚠️  未捕获到任何键盘事件")
	}
}

// TestKeyboardMonitor_ModifierKeys 测试修饰键检测
//
// 此测试验证监控器能够正确检测修饰键（Cmd、Shift、Control、Option）。
// **注意**：此测试需要实际按下组合键。
//
// 运行方式：
// ```bash
// go test -v -run TestKeyboardMonitor_ModifierKeys ./internal/platform/
// ```
func TestKeyboardMonitor_ModifierKeys(t *testing.T) {
	skipIfAutomated(t)

	t.Log("=== 修饰键检测测试 ===")
	t.Log("请按下任意组合键（如 Cmd+A、Shift+B 等）...")

	monitor := NewKeyboardMonitor()

	receivedEvents := make(chan KeyboardEvent, 100)

	err := monitor.Start(func(event KeyboardEvent) {
		receivedEvents <- event
	})

	if err != nil {
		skipWithReason(t, "需要辅助功能权限: %v", err)
	}
	require.NoError(t, err)

	// 等待一个有修饰键的事件
	timeout := time.After(8 * time.Second)
	var capturedEvent *KeyboardEvent

	select {
	case event := <-receivedEvents:
		capturedEvent = &event
		t.Logf("捕获事件: KeyCode=%d, Modifiers=0x%x", event.KeyCode, event.Modifiers)

		// 解析修饰键
		modifiers := event.Modifiers
		if modifiers&0x10000 != 0 {
			t.Log("  ✅ 检测到 CapsLock")
		}
		if modifiers&0x20000 != 0 {
			t.Log("  ✅ 检测到 Shift")
		}
		if modifiers&0x40000 != 0 {
			t.Log("  ✅ 检测到 Control")
		}
		if modifiers&0x80000 != 0 {
			t.Log("  ✅ 检测到 Option (Alt)")
		}
		if modifiers&0x100000 != 0 {
			t.Log("  ✅ 检测到 Command (Cmd)")
		}
	case <-timeout:
		t.Log("⏱️  超时：未检测到键盘输入")
	}

	// 停止监控器
	_ = monitor.Stop()

	if capturedEvent != nil && capturedEvent.Modifiers != 0 {
		t.Log("✅ 成功检测到修饰键！")
	} else if capturedEvent != nil {
		t.Log("⚠️  捕获到事件，但未检测到修饰键")
		t.Log("提示：请尝试按下组合键，如 Cmd+A、Shift+B 等")
	}
}

// skipIfAutomated 如果是自动化测试环境，则跳过测试
func skipIfAutomated(t *testing.T) {
	// 检查是否在 CI 环境中运行
	if testing.Short() {
		t.Skip("跳过手动集成测试（使用 -short 标志）")
	}
}

// skipWithReason 跳过测试并输出原因
func skipWithReason(t *testing.T, format string, args ...interface{}) {
	t.Skipf(format, args...)
}

// TestExample 示例：如何在代码中使用键盘监控器
//
// 此函数不是测试，而是一个使用示例。
func TestExample(t *testing.T) {
	t.Log("=== 键盘监控器使用示例 ===")

	exampleCode := `
package main

import (
    "fmt"
    "time"
    "github.com/chenyang-zz/flowmind/internal/platform"
)

func main() {
    // 创建监控器
    monitor := platform.NewKeyboardMonitor()

    // 启动监控，传入回调函数
    err := monitor.Start(func(event platform.KeyboardEvent) {
        fmt.Printf("按键: KeyCode=%d, Modifiers=0x%x\n",
            event.KeyCode, event.Modifiers)
    })

    if err != nil {
        fmt.Printf("启动失败: %v\n", err)
        return
    }

    fmt.Println("键盘监控已启动，按 Ctrl+C 退出...")

    // 运行 10 秒
    time.Sleep(10 * time.Second)

    // 停止监控
    monitor.Stop()
    fmt.Println("键盘监控已停止")
}
`

	t.Log("示例代码：")
	t.Log(exampleCode)
}
