//go:build darwin

package platform

// HandleCallbackExporter 导出 handleCallback 方法用于测试
func (km *DarwinKeyboardMonitor) HandleCallbackExporter(keyCode int, flags uint64) {
	km.handleCallback(keyCode, flags)
}
