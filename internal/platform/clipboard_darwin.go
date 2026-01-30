//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices

#include <CoreFoundation/CoreFoundation.h>
#include <Cocoa/Cocoa.h>
#include <ApplicationServices/ApplicationServices.h>

// getClipboardContent 获取剪贴板内容
// Returns: 剪贴板字符串内容，如果无法获取文本则返回 NULL
char* getClipboardContent() {
    NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
    if (pasteboard == nil) {
        return NULL;
    }

    // 检查是否包含字符串类型
    NSString *type = [pasteboard availableTypeFromArray:@[NSPasteboardTypeString]];
    if (type == nil) {
        return NULL;
    }

    // 获取字符串内容
    NSString *content = [pasteboard stringForType:NSPasteboardTypeString];
    if (content == nil) {
        return NULL;
    }

    // 转换为 C 字符串
    const char *cString = [content UTF8String];
    if (cString == NULL) {
        return NULL;
    }

    // 复制字符串（调用者需要释放）
    char *result = strdup(cString);
    return result;
}

// getClipboardChangeCount 获取剪贴板变更计数
// Returns: 当前剪贴板的变更计数，用于检测内容变化
long long getClipboardChangeCount() {
    NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
    if (pasteboard == nil) {
        return -1;
    }

    return [pasteboard changeCount];
}

// freeString 释放由 getClipboardContent 分配的字符串
// Parameters: str - 要释放的字符串指针
void freeString(char *str) {
    if (str != NULL) {
        free(str);
    }
}
*/
import "C"
import (
	"fmt"
	"sync"
	"time"

	"github.com/chenyang-zz/flowmind/pkg/logger"
	"go.uber.org/zap"
)

// DarwinClipboardMonitor macOS 平台的剪贴板监控器实现
//
// DarwinClipboardMonitor 使用 NSPasteboard API 来监控剪贴板内容变化。
// 通过定期检查剪贴板的 changeCount 来检测内容是否发生变化。
// 与键盘监控不同，剪贴板监控不需要辅助功能权限。
type DarwinClipboardMonitor struct {
	// callback 用户注册的剪贴板事件回调函数
	callback ClipboardCallback
	// isRunning 监控器运行状态标志
	isRunning bool
	// mu 读写锁，保护并发访问
	mu sync.RWMutex
	// stopChan 停止信号通道
	stopChan chan struct{}
	// lastChangeCount 上一次记录的剪贴板变更计数
	lastChangeCount int64
	// checkInterval 检查间隔（默认 500ms）
	checkInterval time.Duration
}

// NewClipboardMonitor 创建剪贴板监控器
//
// 在 macOS 平台上，此函数返回 DarwinClipboardMonitor 实例。
// Returns: ClipboardMonitor 接口的 macOS 实现
func NewClipboardMonitor() ClipboardMonitor {
	return &DarwinClipboardMonitor{
		stopChan:       make(chan struct{}),
		checkInterval:  500 * time.Millisecond,
		lastChangeCount: -1,
	}
}

// Start 启动剪贴板监控
//
// 启动后会定期检查剪贴板内容变化，当检测到变化时调用回调函数。
// 使用 NSPasteboard 的 changeCount 属性来检测变化，避免频繁读取内容。
// Parameters: callback - 剪贴板事件回调函数
// Returns: error - 启动失败时返回错误
func (cm *DarwinClipboardMonitor) Start(callback ClipboardCallback) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.isRunning {
		return fmt.Errorf("clipboard monitor already running")
	}

	// 保存回调函数
	cm.callback = callback

	// 获取当前剪贴板变更计数作为初始值
	cm.lastChangeCount = int64(C.getClipboardChangeCount())

	// 在独立的 goroutine 中定期检查剪贴板变化
	go cm.monitorLoop()

	cm.isRunning = true

	logger.Info("剪贴板监控已启动",
		zap.String("component", "clipboard"),
		zap.Duration("interval", cm.checkInterval))

	return nil
}

// monitorMonitor 监控循环
//
// 此方法在 goroutine 中执行，定期检查剪贴板变化。
// 通过比较 changeCount 来检测变化，避免频繁读取剪贴板内容。
func (cm *DarwinClipboardMonitor) monitorLoop() {
	ticker := time.NewTicker(cm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.stopChan:
			logger.Info("剪贴板监控循环收到停止信号", zap.String("component", "clipboard"))
			return
		case <-ticker.C:
			cm.checkClipboardChange()
		}
	}
}

// checkClipboardChange 检查剪贴板变化
//
// 通过比较 changeCount 来检测剪贴板内容是否变化。
// 如果检测到变化，读取剪贴板内容并触发回调。
func (cm *DarwinClipboardMonitor) checkClipboardChange() {
	cm.mu.RLock()
	if !cm.isRunning {
		cm.mu.RUnlock()
		return
	}
	cm.mu.RUnlock()

	// 获取当前变更计数
	currentChangeCount := C.getClipboardChangeCount()

	// 如果变更计数与上次相同，说明内容没有变化
	if int64(currentChangeCount) == cm.lastChangeCount {
		return
	}

	// 变更计数发生变化，读取剪贴板内容
	cContent := C.getClipboardContent()
	if cContent == nil {
		logger.Debug("剪贴板内容无法读取（非文本类型）", zap.String("component", "clipboard"))
		// 更新变更计数，避免重复尝试
		cm.lastChangeCount = int64(currentChangeCount)
		return
	}

	// 转换为 Go 字符串
	content := C.GoString(cContent)

	// 释放 C 字符串
	C.freeString(cContent)

	// 更新变更计数
	cm.lastChangeCount = int64(currentChangeCount)

	// 构造剪贴板事件
	event := ClipboardEvent{
		Content: content,
		Type:    "public.utf8-plain-text",
		Size:    int64(len(content)),
	}

	// 异步调用回调函数
	cm.mu.RLock()
	callback := cm.callback
	cm.mu.RUnlock()

	if callback != nil {
		go callback(event)
	}

	logger.Debug("检测到剪贴板内容变化",
		zap.String("component", "clipboard"),
		zap.Int("length", len(content)),
		zap.Int64("changeCount", cm.lastChangeCount))
}

// Stop 停止剪贴板监控
//
// 发送停止信号给监控循环，等待循环退出。
// Returns: error - 停止失败时返回错误
func (cm *DarwinClipboardMonitor) Stop() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.isRunning {
		return fmt.Errorf("clipboard monitor not running")
	}

	logger.Info("停止剪贴板监控", zap.String("component", "clipboard"))

	// 发送停止信号
	select {
	case <-cm.stopChan:
		// 通道已关闭（不应该发生）
	default:
		close(cm.stopChan)
	}

	// 等待监控循环退出（最多等待 1 秒）
	// 注意：这里简单等待，因为监控循环会在下一个 ticker 或立即收到停止信号
	time.Sleep(100 * time.Millisecond)

	// 重新创建停止通道（支持多次启停）
	cm.stopChan = make(chan struct{})
	cm.callback = nil
	cm.isRunning = false

	return nil
}

// IsRunning 检查运行状态
//
// 返回监控器当前的运行状态。
// Returns: bool - 监控器是否正在运行
func (cm *DarwinClipboardMonitor) IsRunning() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.isRunning
}
