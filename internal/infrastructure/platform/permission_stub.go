//go:build !darwin

package platform

import "fmt"

// StubPermissionChecker 权限检查器的 stub 实现（非 macOS 平台）
// 在非 macOS 平台上，权限管理功能不可用
type StubPermissionChecker struct{}

// NewPermissionChecker 创建权限检查器的 stub 实现
// Returns: PermissionChecker - stub 实现的权限检查器实例
func NewPermissionChecker() PermissionChecker {
	return &StubPermissionChecker{}
}

// CheckPermission 检查权限状态（stub 实现，始终返回 Unknown）
// Parameters: permType - 权限类型
// Returns: PermissionStatus - 始终返回 Unknown
func (c *StubPermissionChecker) CheckPermission(permType PermissionType) PermissionStatus {
	return PermissionStatusUnknown
}

// RequestPermission 请求权限（stub 实现，直接返回错误）
// Parameters: permType - 权限类型
// Returns: error - 在非 macOS 平台上始终返回错误
func (c *StubPermissionChecker) RequestPermission(permType PermissionType) error {
	return fmt.Errorf("权限管理仅在 macOS 平台上可用")
}

// OpenSystemSettings 打开系统设置（stub 实现，直接返回错误）
// Parameters: permType - 权限类型
// Returns: error - 在非 macOS 平台上始终返回错误
func (c *StubPermissionChecker) OpenSystemSettings(permType PermissionType) error {
	return fmt.Errorf("系统设置仅在 macOS 平台上可用")
}
