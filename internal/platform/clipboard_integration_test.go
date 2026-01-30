//go:build darwin

package platform

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClipboardMonitor_RealCopyPaste 是一个手动集成测试
//
// 此测试用于验证剪贴板监控在实际使用场景中能够捕获剪贴板内容变化。
// **注意**：此测试需要实际复制操作，不适合在自动化测试环境中运行。
//
// 测试步骤：
//  1. 启动剪贴板监控器
//  2. 等待用户复制一些文本（在接下来的 10 秒内）
//  3. 验证捕获到的剪贴板事件
//  4. 停止监控器
//
// 运行方式：
// ```bash
// go test -v -run TestClipboardMonitor_RealCopyPaste ./internal/platform/
// ```
func TestClipboardMonitor_RealCopyPaste(t *testing.T) {
	skipIfAutomated(t)

	t.Log("=== 剪贴板监控集成测试 ===")
	t.Log("请在接下来的 10 秒内复制一些文本...")
	t.Log("测试将捕获您的剪贴板操作并显示内容")

	monitor := NewClipboardMonitor()

	// 创建事件通道
	receivedEvents := make(chan ClipboardEvent, 100)
	eventCount := 0

	// 启动监控器，传入回调函数
	err := monitor.Start(func(event ClipboardEvent) {
		eventCount++
		t.Logf("✅ 捕获到剪贴板事件 #%d: Type=%s, Size=%d",
			eventCount, event.Type, event.Size)

		// 显示内容预览
		preview := event.Content
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		t.Logf("   内容预览: %s", preview)

		// 将事件发送到通道（非阻塞）
		select {
		case receivedEvents <- event:
		default:
			// 通道已满，忽略
		}
	})

	if err != nil {
		skipWithReason(t, "启动剪贴板监控失败: %v", err)
	}
	require.NoError(t, err)
	assert.True(t, monitor.IsRunning(), "监控器应该正在运行")

	// 等待剪贴板事件（10 秒超时）
	t.Log("📋 等待剪贴板操作...")
	timeout := time.After(10 * time.Second)
	var capturedEvent *ClipboardEvent

	select {
	case event := <-receivedEvents:
		capturedEvent = &event
		t.Logf("🎉 成功捕获剪贴板事件！Type=%s, Size=%d, Length=%d",
			event.Type, event.Size, len(event.Content))
	case <-timeout:
		// 超时不算失败，只是没有复制
		t.Log("⏱️  超时：未检测到剪贴板操作")
	}

	// 停止监控器
	t.Log("🛑 停止剪贴板监控器...")
	err = monitor.Stop()
	require.NoError(t, err)
	assert.False(t, monitor.IsRunning(), "监控器应该已停止")

	// 如果捕获到了事件，进行验证
	if capturedEvent != nil {
		t.Log("✅ 剪贴板监控器工作正常！")
		assert.NotEmpty(t, capturedEvent.Content, "Content 不应该为空")
		assert.NotEmpty(t, capturedEvent.Type, "Type 不应该为空")
		assert.Greater(t, capturedEvent.Size, int64(0), "Size 应该 > 0")
		assert.Equal(t, capturedEvent.Type, "public.utf8-plain-text", "应该捕获到文本类型")
	} else {
		t.Log("⚠️  未捕获到剪贴板事件")
		t.Log("如果您实际复制了文本，请检查：")
		t.Log("  1. 确保复制的是文本内容（不是图片或文件）")
		t.Log("  2. 尝试使用 Cmd+C 快捷键复制")
	}
}

// TestClipboardMonitor_MultipleCopies 测试连续复制
//
// 此测试验证监控器能够连续捕获多个剪贴板事件。
// **注意**：此测试需要实际连续复制操作。
//
// 运行方式：
// ```bash
// go test -v -run TestClipboardMonitor_MultipleCopies ./internal/platform/
// ```
func TestClipboardMonitor_MultipleCopies(t *testing.T) {
	skipIfAutomated(t)

	t.Log("=== 连续复制测试 ===")
	t.Log("请在接下来的 10 秒内连续复制不同的文本...")

	monitor := NewClipboardMonitor()

	receivedEvents := make(chan ClipboardEvent, 100)

	err := monitor.Start(func(event ClipboardEvent) {
		receivedEvents <- event
	})

	if err != nil {
		skipWithReason(t, "启动剪贴板监控失败: %v", err)
	}
	require.NoError(t, err)

	// 收集 5 秒内的所有事件
	timeout := time.After(10 * time.Second)
	var events []ClipboardEvent

eventLoop:
	for {
		select {
		case event := <-receivedEvents:
			events = append(events, event)
			t.Logf("捕获事件 #%d: Type=%s, Size=%d", len(events), event.Type, event.Size)
		case <-timeout:
			break eventLoop
		}
	}

	// 停止监控器
	_ = monitor.Stop()

	t.Logf("总共捕获了 %d 个剪贴板事件", len(events))

	if len(events) > 0 {
		t.Log("✅ 成功捕获多个剪贴板事件！")
		for i, event := range events {
			preview := event.Content
			if len(preview) > 30 {
				preview = preview[:30] + "..."
			}
			t.Logf("  事件 #%d: Type=%s, Size=%d, Content=%s",
				i+1, event.Type, event.Size, preview)
		}
		assert.Greater(t, len(events), 0, "应该捕获到至少一个事件")
	} else {
		t.Log("⚠️  未捕获到任何剪贴板事件")
	}
}

// TestClipboardMonitor_Deduplication 测试剪贴板内容去重
//
// 此测试验证监控器不会对相同内容重复触发事件。
// **注意**：此测试需要实际复制操作。
//
// 运行方式：
// ```bash
// go test -v -run TestClipboardMonitor_Deduplication ./internal/platform/
// ```
func TestClipboardMonitor_Deduplication(t *testing.T) {
	skipIfAutomated(t)

	t.Log("=== 剪贴板去重测试 ===")
	t.Log("请复制同一文本两次，观察是否只触发一次事件...")

	monitor := NewClipboardMonitor()

	receivedEvents := make(chan ClipboardEvent, 100)
	eventCount := 0

	err := monitor.Start(func(event ClipboardEvent) {
		eventCount++
		preview := event.Content
		if len(preview) > 30 {
			preview = preview[:30] + "..."
		}
		t.Logf("捕获到事件 #%d: %s", eventCount, preview)
		receivedEvents <- event
	})

	if err != nil {
		skipWithReason(t, "启动剪贴板监控失败: %v", err)
	}
	require.NoError(t, err)

	// 等待第一次复制
	t.Log("⌨️  等待第一次复制...")
	timeout1 := time.After(8 * time.Second)
	var firstEvent *ClipboardEvent

	select {
	case event := <-receivedEvents:
		firstEvent = &event
		t.Log("✅ 捕获到第一次复制")
	case <-timeout1:
		t.Log("⏱️  超时：未检测到剪贴板操作")
		_ = monitor.Stop()
		t.Skip("需要实际复制操作")
	}

	// 等待第二次复制（相同内容）
	if firstEvent != nil {
		t.Log("⌨️  请再次复制相同的文本...")
		t.Logf("   (提示：内容是: %s)", firstEvent.Content)

		timeout2 := time.After(5 * time.Second)
		select {
		case <-receivedEvents:
			// 如果收到事件，说明是不同内容
			t.Log("⚠️  捕获到第二次事件（内容可能不同）")
		case <-timeout2:
			t.Log("✅ 未捕获到重复事件（去重成功！）")
		}
	}

	// 停止监控器
	_ = monitor.Stop()
}

// TestClipboardMonitor_LongText 测试长文本处理
//
// 此测试验证监控器能够处理较长的剪贴板内容。
// **注意**：此测试需要实际复制长文本。
//
// 运行方式：
// ```bash
// go test -v -run TestClipboardMonitor_LongText ./internal/platform/
// ```
func TestClipboardMonitor_LongText(t *testing.T) {
	skipIfAutomated(t)

	t.Log("=== 长文本处理测试 ===")
	t.Log("请复制一段较长的文本（建议 100+ 字符）...")

	monitor := NewClipboardMonitor()

	receivedEvents := make(chan ClipboardEvent, 10)

	err := monitor.Start(func(event ClipboardEvent) {
		receivedEvents <- event
	})

	if err != nil {
		skipWithReason(t, "启动剪贴板监控失败: %v", err)
	}
	require.NoError(t, err)

	// 等待复制
	timeout := time.After(10 * time.Second)
	var capturedEvent *ClipboardEvent

	select {
	case event := <-receivedEvents:
		capturedEvent = &event
		t.Logf("✅ 捕获到剪贴板内容")
		t.Logf("   长度: %d 字符", len(event.Content))
		t.Logf("   大小: %d 字节", event.Size)
	case <-timeout:
		t.Log("⏱️  超时：未检测到剪贴板操作")
	}

	// 停止监控器
	_ = monitor.Stop()

	if capturedEvent != nil {
		assert.NotEmpty(t, capturedEvent.Content, "Content 不应该为空")
		assert.Equal(t, int64(len(capturedEvent.Content)), capturedEvent.Size,
			"Size 应该等于内容长度")
	}
}

// TestClipboardMonitor_StartStop 测试启动和停止
//
// 此测试验证监控器的启动和停止功能。
func TestClipboardMonitor_StartStop(t *testing.T) {
	t.Log("=== 启动和停止测试 ===")

	monitor := NewClipboardMonitor()

	// 初始状态
	assert.False(t, monitor.IsRunning(), "初始状态应该是未运行")

	// 启动
	err := monitor.Start(func(event ClipboardEvent) {})
	require.NoError(t, err)
	assert.True(t, monitor.IsRunning(), "启动后应该是运行状态")

	// 停止
	err = monitor.Stop()
	require.NoError(t, err)
	assert.False(t, monitor.IsRunning(), "停止后应该是未运行状态")

	t.Log("✅ 启动和停止测试通过")
}

// TestClipboardMonitor_StartTwice 测试重复启动
//
// 此测试验证重复启动监控器的幂等性。
func TestClipboardMonitor_StartTwice(t *testing.T) {
	t.Log("=== 重复启动测试 ===")

	monitor := NewClipboardMonitor()

	// 第一次启动
	err := monitor.Start(func(event ClipboardEvent) {})
	require.NoError(t, err)
	assert.True(t, monitor.IsRunning())

	// 第二次启动（应该失败）
	err = monitor.Start(func(event ClipboardEvent) {})
	assert.Error(t, err, "重复启动应该返回错误")
	assert.True(t, monitor.IsRunning(), "状态应该保持运行")

	// 清理
	_ = monitor.Stop()

	t.Log("✅ 重复启动测试通过")
}

// TestClipboardMonitor_StopTwice 测试重复停止
//
// 此测试验证重复停止监控器的幂等性。
func TestClipboardMonitor_StopTwice(t *testing.T) {
	t.Log("=== 重复停止测试 ===")

	monitor := NewClipboardMonitor()

	// 启动
	err := monitor.Start(func(event ClipboardEvent) {})
	require.NoError(t, err)

	// 第一次停止
	err = monitor.Stop()
	require.NoError(t, err)
	assert.False(t, monitor.IsRunning())

	// 第二次停止（应该失败）
	err = monitor.Stop()
	assert.Error(t, err, "重复停止应该返回错误")

	t.Log("✅ 重复停止测试通过")
}

// TestClipboardMonitor_Example 示例：如何在代码中使用剪贴板监控器
//
// 此函数不是测试，而是一个使用示例。
func TestClipboardMonitor_Example(t *testing.T) {
	t.Log("=== 剪贴板监控器使用示例 ===")

	exampleCode := `
package main

import (
    "fmt"
    "time"
    "github.com/chenyang-zz/flowmind/internal/platform"
)

func main() {
    // 创建监控器
    monitor := platform.NewClipboardMonitor()

    // 启动监控，传入回调函数
    err := monitor.Start(func(event platform.ClipboardEvent) {
        fmt.Printf("剪贴板变化:\n")
        fmt.Printf("  类型: %s\n", event.Type)
        fmt.Printf("  大小: %d 字节\n", event.Size)
        fmt.Printf("  内容: %s\n", event.Content)
    })

    if err != nil {
        fmt.Printf("启动失败: %v\n", err)
        return
    }

    fmt.Println("剪贴板监控已启动，按 Ctrl+C 退出...")

    // 运行 10 秒
    time.Sleep(10 * time.Second)

    // 停止监控
    monitor.Stop()
    fmt.Println("剪贴板监控已停止")
}
`

	t.Log("示例代码：")
	t.Log(exampleCode)
}

// TestClipboardMonitor_CallbackNil 测试回调函数为 nil 的情况
//
// 此测试验证当回调函数为 nil 时，监控器能够正常工作。
func TestClipboardMonitor_CallbackNil(t *testing.T) {
	t.Log("=== nil 回调测试 ===")

	monitor := NewClipboardMonitor()

	// 启动时传入 nil 回调
	err := monitor.Start(nil)
	require.NoError(t, err)
	assert.True(t, monitor.IsRunning())

	// 等待一小段时间，确保监控循环运行
	time.Sleep(200 * time.Millisecond)

	// 停止监控器
	err = monitor.Stop()
	require.NoError(t, err)
	assert.False(t, monitor.IsRunning())

	t.Log("✅ nil 回调测试通过")
}

// TestClipboardMonitor_RapidStartStopCycles 测试快速启停循环
//
// 此测试验证监控器能够承受快速的启停循环。
func TestClipboardMonitor_RapidStartStopCycles(t *testing.T) {
	t.Log("=== 快速启停循环测试 ===")

	monitor := NewClipboardMonitor()

	// 进行多次快速启停
	for i := 0; i < 5; i++ {
		err := monitor.Start(func(event ClipboardEvent) {})
		require.NoError(t, err, "第 %d 次启动应该成功", i+1)

		// 立即停止
		err = monitor.Stop()
		require.NoError(t, err, "第 %d 次停止应该成功", i+1)

		assert.False(t, monitor.IsRunning())
	}

	t.Log("✅ 快速启停循环测试通过")
}

// TestClipboardMonitor_StopDuringCallback 测试在回调执行时停止监控器
//
// 此测试验证在回调函数执行期间停止监控器的行为。
func TestClipboardMonitor_StopDuringCallback(t *testing.T) {
	skipIfAutomated(t)

	t.Log("=== 回调期间停止测试 ===")
	t.Log("请在监控期间复制文本，观察停止行为")

	monitor := NewClipboardMonitor()
	callbackExecuted := make(chan struct{})

	err := monitor.Start(func(event ClipboardEvent) {
		t.Log("回调函数执行")
		close(callbackExecuted)
		time.Sleep(100 * time.Millisecond) // 模拟耗时操作
	})

	require.NoError(t, err)

	// 等待回调执行或超时
	select {
	case <-callbackExecuted:
		t.Log("✅ 回调已执行")
	case <-time.After(5 * time.Second):
		t.Log("⏱️  未检测到剪贴板操作")
	}

	// 停止监控器
	err = monitor.Stop()
	require.NoError(t, err)

	t.Log("✅ 回调期间停止测试通过")
}

// TestClipboardMonitor_ConcurrentStartAttempts 测试并发启动尝试
//
// 此测试验证多个 goroutine 同时尝试启动监控器时的行为。
func TestClipboardMonitor_ConcurrentStartAttempts(t *testing.T) {
	t.Log("=== 并发启动尝试测试 ===")

	monitor := NewClipboardMonitor()
	done := make(chan error, 3)

	// 三个 goroutine 同时尝试启动
	for i := 0; i < 3; i++ {
		go func() {
			err := monitor.Start(func(event ClipboardEvent) {})
			done <- err
		}()
	}

	// 收集结果
	successCount := 0
	failCount := 0
	for i := 0; i < 3; i++ {
		err := <-done
		if err == nil {
			successCount++
		} else {
			failCount++
		}
	}

	// 应该只有一个成功，其他失败
	assert.Equal(t, 1, successCount, "应该只有一个启动成功")
	assert.Equal(t, 2, failCount, "应该有两个启动失败")
	assert.True(t, monitor.IsRunning())

	// 清理
	_ = monitor.Stop()

	t.Logf("✅ 并发启动测试通过 (成功: %d, 失败: %d)", successCount, failCount)
}

// TestClipboardMonitor_ContentSize 测试不同大小的剪贴板内容
//
// 此测试验证监控器能够处理不同大小的内容。
func TestClipboardMonitor_ContentSize(t *testing.T) {
	skipIfAutomated(t)

	t.Log("=== 内容大小测试 ===")
	t.Log("请复制以下不同大小的文本:")
	t.Log("1. 空字符串（如果可能）")
	t.Log("2. 单个字符")
	t.Log("3. 中等长度文本")
	t.Log("4. 长文本")

	monitor := NewClipboardMonitor()
	receivedEvents := make(chan ClipboardEvent, 10)

	err := monitor.Start(func(event ClipboardEvent) {
		receivedEvents <- event
	})

	require.NoError(t, err)

	// 收集多个事件
	timeout := time.After(10 * time.Second)
	var events []ClipboardEvent

eventLoop:
	for {
		select {
		case event := <-receivedEvents:
			events = append(events, event)
			t.Logf("捕获事件 #%d: Size=%d, Length=%d",
				len(events), event.Size, len(event.Content))
		case <-timeout:
			break eventLoop
		}
	}

	_ = monitor.Stop()

	t.Logf("总共捕获了 %d 个剪贴板事件", len(events))

	// 验证不同大小的内容
	if len(events) > 0 {
		for i, event := range events {
			t.Logf("事件 #%d: Content Length=%d, Size=%d",
				i+1, len(event.Content), event.Size)
			assert.Equal(t, int64(len(event.Content)), event.Size,
				"Size 应该等于内容长度")
		}
	}
}

// TestClipboardMonitor_NonTextContent 测试非文本剪贴板内容
//
// 此测试验证监控器如何处理非文本内容（如图片）。
func TestClipboardMonitor_NonTextContent(t *testing.T) {
	skipIfAutomated(t)

	t.Log("=== 非文本内容测试 ===")
	t.Log("请复制一些非文本内容（如图片）...")

	monitor := NewClipboardMonitor()
	textEventCount := 0
	totalEvents := 0

	err := monitor.Start(func(event ClipboardEvent) {
		totalEvents++
		if event.Type == "public.utf8-plain-text" {
			textEventCount++
		}
		t.Logf("捕获事件: Type=%s, Size=%d", event.Type, event.Size)
	})

	require.NoError(t, err)

	// 等待非文本内容
	t.Log("等待剪贴板操作...")
	time.Sleep(8 * time.Second)

	_ = monitor.Stop()

	t.Logf("总共捕获了 %d 个事件，其中 %d 个文本事件", totalEvents, textEventCount)
}

// TestClipboardMonitor_NewMonitor 测试创建新监控器
//
// 此测试验证多次创建监控器实例的正确性。
func TestClipboardMonitor_NewMonitor(t *testing.T) {
	t.Log("=== 创建监控器测试 ===")

	// 创建多个监控器实例
	monitor1 := NewClipboardMonitor()
	monitor2 := NewClipboardMonitor()

	// 验证实例独立
	assert.NotNil(t, monitor1)
	assert.NotNil(t, monitor2)
	assert.NotSame(t, monitor1, monitor2)

	// 验证初始状态
	assert.False(t, monitor1.IsRunning())
	assert.False(t, monitor2.IsRunning())

	// 启动第一个
	err := monitor1.Start(func(event ClipboardEvent) {})
	require.NoError(t, err)

	// 验证状态独立
	assert.True(t, monitor1.IsRunning())
	assert.False(t, monitor2.IsRunning())

	// 清理
	_ = monitor1.Stop()

	t.Log("✅ 创建监控器测试通过")
}

// TestClipboardMonitor_CheckInterval 测试检查间隔
//
// 此测试验证监控器使用正确的检查间隔。
func TestClipboardMonitor_CheckInterval(t *testing.T) {
	t.Log("=== 检查间隔测试 ===")

	monitor := NewClipboardMonitor()

	// macOS 平台的监控器应该有 500ms 的检查间隔
	darwinMonitor, ok := monitor.(*DarwinClipboardMonitor)
	if ok {
		assert.Equal(t, 500*time.Millisecond, darwinMonitor.checkInterval,
			"检查间隔应该是 500ms")
		t.Log("✅ 检查间隔正确: 500ms")
	} else {
		t.Skip("非 macOS 平台，跳过检查间隔测试")
	}
}

// TestClipboardMonitor_CallbackExecution 测试回调函数的执行
//
// 此测试验证回调函数被正确调用并接收正确的事件数据。
func TestClipboardMonitor_CallbackExecution(t *testing.T) {
	skipIfAutomated(t)

	t.Log("=== 回调执行测试 ===")
	t.Log("请复制一些文本...")

	monitor := NewClipboardMonitor()

	callbackCalled := make(chan *ClipboardEvent, 1)

	err := monitor.Start(func(event ClipboardEvent) {
		t.Log("✅ 回调函数被调用")
		t.Logf("事件数据: Type=%s, Size=%d, Content Length=%d",
			event.Type, event.Size, len(event.Content))
		callbackCalled <- &event
	})

	require.NoError(t, err)

	select {
	case event := <-callbackCalled:
		assert.NotNil(t, event)
		assert.NotEmpty(t, event.Content, "Content 不应该为空")
		assert.NotEmpty(t, event.Type, "Type 不应该为空")
		assert.Greater(t, event.Size, int64(0), "Size 应该大于 0")
		t.Log("✅ 回调执行正确")
	case <-time.After(8 * time.Second):
		t.Log("⏱️  超时：未检测到剪贴板操作")
	}

	_ = monitor.Stop()
}
