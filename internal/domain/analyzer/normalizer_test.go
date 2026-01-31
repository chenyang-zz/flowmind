package analyzer

import (
	"testing"

	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/stretchr/testify/assert"
)

// TestEventNormalizer_NormalizeEvent_Keyboard 测试键盘事件标准化
func TestEventNormalizer_NormalizeEvent_Keyboard(t *testing.T) {
	config := DefaultEventNormalizerConfig()
	normalizer := NewEventNormalizer(config)

	tests := []struct {
		name         string
		keyCode      int
		expectedType events.EventType
		expectedAction string
	}{
		{
			name:           "功能键F1",
			keyCode:        122,
			expectedType:   events.EventTypeKeyboard,
			expectedAction: "function_key",
		},
		{
			name:           "修饰键Command",
			keyCode:        55,
			expectedType:   events.EventTypeKeyboard,
			expectedAction: "modifier_key",
		},
		{
			name:           "特殊键Return",
			keyCode:        36,
			expectedType:   events.EventTypeKeyboard,
			expectedAction: "special_key",
		},
		{
			name:           "字母键",
			keyCode:        0,  // A键
			expectedType:   events.EventTypeKeyboard,
			expectedAction: "alphanumeric_key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
				"keycode": float64(tt.keyCode),
			})
			event.Context = &events.EventContext{
				Application: "VSCode",
				BundleID:    "com.microsoft.vscode",
			}

			step := normalizer.NormalizeEvent(*event)

			assert.Equal(t, tt.expectedType, step.Type)
			assert.Equal(t, tt.expectedAction, step.Action)
			assert.NotNil(t, step.Context)
			assert.Equal(t, "VSCode", step.Context.Application)
		})
	}
}

// TestEventNormalizer_NormalizeEvent_Clipboard 测试剪贴板事件标准化
func TestEventNormalizer_NormalizeEvent_Clipboard(t *testing.T) {
	config := DefaultEventNormalizerConfig()
	normalizer := NewEventNormalizer(config)

	tests := []struct {
		name           string
		operation      string
		expectedAction string
	}{
		{
			name:           "复制操作",
			operation:      "copy",
			expectedAction: "clipboard_copy",
		},
		{
			name:           "粘贴操作",
			operation:      "paste",
			expectedAction: "clipboard_paste",
		},
		{
			name:           "剪切操作",
			operation:      "cut",
			expectedAction: "clipboard_cut",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := events.NewEvent(events.EventTypeClipboard, map[string]interface{}{
				"operation": tt.operation,
			})
			event.Context = &events.EventContext{
				Application: "Chrome",
				BundleID:    "com.google.chrome",
			}

			step := normalizer.NormalizeEvent(*event)

			assert.Equal(t, events.EventTypeClipboard, step.Type)
			assert.Equal(t, tt.expectedAction, step.Action)
			assert.NotNil(t, step.Context)
		})
	}
}

// TestEventNormalizer_NormalizeEvent_AppSwitch 测试应用切换事件标准化
func TestEventNormalizer_NormalizeEvent_AppSwitch(t *testing.T) {
	config := DefaultEventNormalizerConfig()
	normalizer := NewEventNormalizer(config)

	event := events.NewEvent(events.EventTypeAppSwitch, map[string]interface{}{
		"from":      "VSCode",
		"to":        "Chrome",
		"bundle_id": "com.google.chrome",
	})
	event.Context = &events.EventContext{
		Application: "Chrome",
		BundleID:    "com.google.chrome",
	}

	step := normalizer.NormalizeEvent(*event)

	assert.Equal(t, events.EventTypeAppSwitch, step.Type)
	assert.Equal(t, "app_switch", step.Action)
	assert.NotNil(t, step.Context)
	assert.Equal(t, "Chrome", step.Context.Application)
}

// TestEventNormalizer_NormalizeEvent_AppSession 测试应用会话事件标准化
func TestEventNormalizer_NormalizeEvent_AppSession(t *testing.T) {
	config := DefaultEventNormalizerConfig()
	normalizer := NewEventNormalizer(config)

	tests := []struct {
		name           string
		action         string
		expectedAction string
	}{
		{
			name:           "会话开始",
			action:         "start",
			expectedAction: "app_session_start",
		},
		{
			name:           "会话结束",
			action:         "end",
			expectedAction: "app_session_end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := events.NewEvent(events.EventTypeAppSession, map[string]interface{}{
				"action": tt.action,
			})
			event.Context = &events.EventContext{
				Application: "VSCode",
				BundleID:    "com.microsoft.vscode",
			}

			step := normalizer.NormalizeEvent(*event)

			assert.Equal(t, events.EventTypeAppSession, step.Type)
			assert.Equal(t, tt.expectedAction, step.Action)
		})
	}
}

// TestEventNormalizer_NormalizeEvent_FileSystem 测试文件系统事件标准化
func TestEventNormalizer_NormalizeEvent_FileSystem(t *testing.T) {
	config := DefaultEventNormalizerConfig()
	normalizer := NewEventNormalizer(config)

	tests := []struct {
		name           string
		operation      string
		expectedAction string
	}{
		{
			name:           "创建文件",
			operation:      "create",
			expectedAction: "file_create",
		},
		{
			name:           "写入文件",
			operation:      "write",
			expectedAction: "file_write",
		},
		{
			name:           "删除文件",
			operation:      "remove",
			expectedAction: "file_remove",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := events.NewEvent(events.EventTypeFileSystem, map[string]interface{}{
				"operation": tt.operation,
				"path":      "/Users/test/file.txt",
			})
			event.Context = &events.EventContext{
				Application: "VSCode",
				BundleID:    "com.microsoft.vscode",
			}

			step := normalizer.NormalizeEvent(*event)

			assert.Equal(t, events.EventTypeFileSystem, step.Type)
			assert.Equal(t, tt.expectedAction, step.Action)
		})
	}
}

// TestEventNormalizer_NormalizeEvents 测试批量事件标准化
func TestEventNormalizer_NormalizeEvents(t *testing.T) {
	config := DefaultEventNormalizerConfig()
	normalizer := NewEventNormalizer(config)

	eventList := []events.Event{
		*events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"keycode": float64(122), // F1
		}),
		*events.NewEvent(events.EventTypeClipboard, map[string]interface{}{
			"operation": "copy",
		}),
		*events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"keycode": float64(55), // Command
		}),
	}

	// 设置上下文
	for i := range eventList {
		eventList[i].Context = &events.EventContext{
			Application: "VSCode",
			BundleID:    "com.microsoft.vscode",
		}
	}

	steps := normalizer.NormalizeEvents(eventList)

	assert.Len(t, steps, 3)
	assert.Equal(t, events.EventTypeKeyboard, steps[0].Type)
	assert.Equal(t, events.EventTypeClipboard, steps[1].Type)
	assert.Equal(t, events.EventTypeKeyboard, steps[2].Type)
}

// TestEventNormalizer_GuessContentType 测试内容类型猜测
func TestEventNormalizer_GuessContentType(t *testing.T) {
	config := DefaultEventNormalizerConfig()
	normalizer := NewEventNormalizer(config)

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "URL内容",
			content:  "https://github.com",
			expected: "url",
		},
		{
			name:     "路径内容",
			content:  "/Users/test/file.txt",
			expected: "path",
		},
		{
			name:     "代码内容",
			content:  "func test() { return 1; }",
			expected: "code",
		},
		{
			name:     "普通文本",
			content:  "hello world",
			expected: "text",
		},
		{
			name:     "空内容",
			content:  "   ",
			expected: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.guessContentType(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestEventNormalizer_ExtractFileExtension 测试文件扩展名提取
func TestEventNormalizer_ExtractFileExtension(t *testing.T) {
	config := DefaultEventNormalizerConfig()
	normalizer := NewEventNormalizer(config)

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "文本文件",
			path:     "/Users/test/file.txt",
			expected: "txt",
		},
		{
			name:     "Go文件",
			path:     "/Users/test/main.go",
			expected: "go",
		},
		{
			name:     "多段扩展名",
			path:     "/Users/test/archive.tar.gz",
			expected: "gz",
		},
		{
			name:     "无扩展名",
			path:     "/Users/test/Makefile",
			expected: "unknown",
		},
		{
			name:     "隐藏文件",
			path:     "/Users/test/.gitignore",
			expected: "gitignore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.extractFileExtension(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestEventNormalizer_NoContext 测试无上下文事件
func TestEventNormalizer_NoContext(t *testing.T) {
	config := DefaultEventNormalizerConfig()
	normalizer := NewEventNormalizer(config)

	event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
		"keycode": float64(122),
	})
	// 不设置 Context

	step := normalizer.NormalizeEvent(*event)

	assert.Equal(t, events.EventTypeKeyboard, step.Type)
	assert.Equal(t, "function_key", step.Action)
	assert.Nil(t, step.Context)
}

// TestEventNormalizer_NoIncludeContext 测试禁用上下文包含
func TestEventNormalizer_NoIncludeContext(t *testing.T) {
	config := DefaultEventNormalizerConfig()
	config.IncludeContext = false
	normalizer := NewEventNormalizer(config)

	event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
		"keycode": float64(122),
	})
	event.Context = &events.EventContext{
		Application: "VSCode",
		BundleID:    "com.microsoft.vscode",
	}

	step := normalizer.NormalizeEvent(*event)

	assert.Equal(t, events.EventTypeKeyboard, step.Type)
	assert.Equal(t, "function_key", step.Action)
	assert.Nil(t, step.Context)
}

// TestEventNormalizer_UnknownEvent 测试未知事件类型
func TestEventNormalizer_UnknownEvent(t *testing.T) {
	config := DefaultEventNormalizerConfig()
	normalizer := NewEventNormalizer(config)

	event := events.NewEvent("unknown_type", map[string]interface{}{
		"data": "test",
	})

	step := normalizer.NormalizeEvent(*event)

	assert.Equal(t, events.EventType("unknown_type"), step.Type)
	assert.Equal(t, "unknown", step.Action)
}

// TestEventNormalizer_ClipboardNoOperation 测试无操作类型的剪贴板事件
func TestEventNormalizer_ClipboardNoOperation(t *testing.T) {
	config := DefaultEventNormalizerConfig()
	normalizer := NewEventNormalizer(config)

	event := events.NewEvent(events.EventTypeClipboard, map[string]interface{}{
		"content": "test content",
	})
	event.Context = &events.EventContext{
		Application: "Chrome",
		BundleID:    "com.google.chrome",
	}

	step := normalizer.NormalizeEvent(*event)

	assert.Equal(t, events.EventTypeClipboard, step.Type)
	assert.Equal(t, "clipboard_operation", step.Action)
}

// TestEventNormalizer_NoGeneralizeKeys 测试禁用按键泛化
func TestEventNormalizer_NoGeneralizeKeys(t *testing.T) {
	config := DefaultEventNormalizerConfig()
	config.GeneralizeKeys = false
	normalizer := NewEventNormalizer(config)

	event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
		"keycode": float64(122),
	})
	event.Context = &events.EventContext{
		Application: "VSCode",
		BundleID:    "com.microsoft.vscode",
	}

	step := normalizer.NormalizeEvent(*event)

	assert.Equal(t, events.EventTypeKeyboard, step.Type)
	// 应该返回具体的按键码，而不是泛化类型
	assert.NotNil(t, step.Context)
	assert.NotEmpty(t, step.Context.PatternValue)
}
