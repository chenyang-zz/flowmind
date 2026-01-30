//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices

#include <ApplicationServices/ApplicationServices.h>

// checkAccessibilityPermission 检查辅助功能权限（static 避免符号冲突）
// 使用 AXIsProcessTrusted 检查当前进程是否被信任
// Returns: 1=已授权, 0=未授权
static int checkAccessibilityPermission() {
    // 使用 AXIsProcessTrusted 检查辅助功能权限
    // 返回 1 表示已授权，0 表示未授权
    return AXIsProcessTrusted();
}

// requestAccessibilityPermission 请求辅助功能权限（static 避免符号冲突）
// 使用 AXIsProcessTrustedWithOptions 显示系统权限请求对话框
// Returns: 0=成功, -1=失败
static int requestAccessibilityPermission() {
    @autoreleasepool {
        // 创建选项字典，设置显示提示对话框
        NSDictionary *options = @{(__bridge id)kAXTrustedCheckOptionPrompt: @YES};

        // 请求辅助功能权限
        // 这会显示系统对话框提示用户授予权限
        BOOL trusted = AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options);

        return trusted ? 0 : -1;
    }
}
*/
import "C"
import (
	"fmt"
	"os/exec"
)

// DarwinPermissionChecker macOS 平台的权限检查器实现
type DarwinPermissionChecker struct{}

// NewPermissionChecker 创建 macOS 平台的权限检查器
// Returns: PermissionChecker - macOS 平台的权限检查器实例
func NewPermissionChecker() PermissionChecker {
	return &DarwinPermissionChecker{}
}

// CheckPermission 检查权限状态
// Parameters: permType - 权限类型
// Returns: PermissionStatus - 权限状态
func (c *DarwinPermissionChecker) CheckPermission(permType PermissionType) PermissionStatus {
	switch permType {
	case PermissionAccessibility:
		// 调用 C 函数检查辅助功能权限
		result := C.checkAccessibilityPermission()
		if result == 1 {
			return PermissionStatusGranted
		}
		return PermissionStatusDenied

	case PermissionScreenCapture:
		// 屏幕录制权限检查（预留实现）
		// TODO: 实现 CGScreenCaptureGetDisplayAccessStatus
		return PermissionStatusUnknown

	case PermissionFiles:
		// 文件访问权限检查（预留实现）
		// TODO: 实现文件访问权限检查
		return PermissionStatusUnknown

	default:
		return PermissionStatusUnknown
	}
}

// RequestPermission 请求权限
// 显示系统权限请求对话框，或引导用户手动授权
// Parameters: permType - 权限类型
// Returns: error - 请求失败时返回错误
func (c *DarwinPermissionChecker) RequestPermission(permType PermissionType) error {
	switch permType {
	case PermissionAccessibility:
		// 调用 C 函数请求辅助功能权限
		result := C.requestAccessibilityPermission()
		if result != 0 {
			return fmt.Errorf("请求辅助功能权限失败")
		}
		return nil

	case PermissionScreenCapture:
		// 屏幕录制权限请求（预留实现）
		return fmt.Errorf("屏幕录制权限请求功能尚未实现")

	case PermissionFiles:
		// 文件访问权限请求（预留实现）
		return fmt.Errorf("文件访问权限请求功能尚未实现")

	default:
		return fmt.Errorf("未知的权限类型: %v", permType)
	}
}

// OpenSystemSettings 打开系统设置
// 直接打开系统偏好设置中的对应权限页面
// Parameters: permType - 权限类型
// Returns: error - 打开失败时返回错误
func (c *DarwinPermissionChecker) OpenSystemSettings(permType PermissionType) error {
	var url string

	switch permType {
	case PermissionAccessibility:
		// 辅助功能权限的系统设置 URL
		url = "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility"

	case PermissionScreenCapture:
		// 屏幕录制权限的系统设置 URL
		url = "x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture"

	case PermissionFiles:
		// 文件访问权限的系统设置 URL
		url = "x-apple.systempreferences:com.apple.preference.security?Privacy_FilesAndFolders"

	default:
		return fmt.Errorf("未知的权限类型: %v", permType)
	}

	// 使用 open 命令打开系统设置
	cmd := exec.Command("open", url)
	err := cmd.Start()
	if err != nil {
		// 返回详细错误信息
		return fmt.Errorf("打开系统设置失败: %w", err)
	}

	// 不等待命令完成，立即返回
	return nil
}
