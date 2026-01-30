package services

import (
	"sync"
	"testing"
	"time"

	"github.com/chenyang-zz/flowmind/internal/infrastructure/platform"
	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/stretchr/testify/assert"
)

// MockPermissionChecker 模拟的权限检查器
// 用于测试 PermissionManager 的业务逻辑
type MockPermissionChecker struct {
	// permissions 存储权限状态
	permissions map[platform.PermissionType]platform.PermissionStatus

	// requestCalled 记录是否调用了请求权限
	requestCalled bool

	// requestPermission 记录请求的权限类型
	requestPermission platform.PermissionType

	// openSettingsCalled 记录是否调用了打开系统设置
	openSettingsCalled bool

	// openSettingsPermission 记录打开设置的权限类型
	openSettingsPermission platform.PermissionType

	// mu 互斥锁
	mu sync.Mutex
}

// NewMockPermissionChecker 创建模拟权限检查器
// Returns: *MockPermissionChecker - 新创建的模拟权限检查器
func NewMockPermissionChecker() *MockPermissionChecker {
	return &MockPermissionChecker{
		permissions: make(map[platform.PermissionType]platform.PermissionStatus),
	}
}

// SetPermission 设置权限状态（用于测试）
// Parameters: permType - 权限类型, status - 权限状态
func (m *MockPermissionChecker) SetPermission(permType platform.PermissionType, status platform.PermissionStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.permissions[permType] = status
}

// CheckPermission 检查权限状态
// Parameters: permType - 权限类型
// Returns: PermissionStatus - 权限状态
func (m *MockPermissionChecker) CheckPermission(permType platform.PermissionType) platform.PermissionStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	if status, ok := m.permissions[permType]; ok {
		return status
	}
	return platform.PermissionStatusDenied
}

// RequestPermission 请求权限
// Parameters: permType - 权限类型
// Returns: error - 请求失败时返回错误
func (m *MockPermissionChecker) RequestPermission(permType platform.PermissionType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestCalled = true
	m.requestPermission = permType

	// 模拟权限授予
	m.permissions[permType] = platform.PermissionStatusGranted

	return nil
}

// OpenSystemSettings 打开系统设置
// Parameters: permType - 权限类型
// Returns: error - 打开失败时返回错误
func (m *MockPermissionChecker) OpenSystemSettings(permType platform.PermissionType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.openSettingsCalled = true
	m.openSettingsPermission = permType

	return nil
}

// WasRequestCalled 检查是否调用了请求权限
// Returns: bool - true 表示已调用
func (m *MockPermissionChecker) WasRequestCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requestCalled
}

// GetRequestedPermission 获取请求的权限类型
// Returns: PermissionType - 请求的权限类型
func (m *MockPermissionChecker) GetRequestedPermission() platform.PermissionType {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requestPermission
}

// TestNewPermissionManager 测试PermissionManager的创建
//
// 验证PermissionManager能够正确创建，并且初始状态正确。
func TestNewPermissionManager(t *testing.T) {
	eventBus := events.NewEventBus()
	checker := NewMockPermissionChecker()

	pm := NewPermissionManager(checker, eventBus)

	assert.NotNil(t, pm)
	assert.NotNil(t, pm.checker)
	assert.NotNil(t, pm.eventBus)
}

// TestPermissionManager_CheckPermission 测试权限检查
//
// 验证权限检查功能正常工作，包括缓存机制。
func TestPermissionManager_CheckPermission(t *testing.T) {
	eventBus := events.NewEventBus()
	checker := NewMockPermissionChecker()
	pm := NewPermissionManager(checker, eventBus)

	// 设置权限状态为已授予
	checker.SetPermission(platform.PermissionAccessibility, platform.PermissionStatusGranted)

	// 首次检查（应该调用底层检查器）
	status := pm.CheckPermission(platform.PermissionAccessibility)
	assert.Equal(t, platform.PermissionStatusGranted, status)

	// 再次检查（应该从缓存获取，不调用底层检查器）
	status = pm.CheckPermission(platform.PermissionAccessibility)
	assert.Equal(t, platform.PermissionStatusGranted, status)
}

// TestPermissionManager_CheckPermissionDenied 测试权限被拒绝的情况
//
// 验证权限被拒绝时正确返回 Denied 状态。
func TestPermissionManager_CheckPermissionDenied(t *testing.T) {
	eventBus := events.NewEventBus()
	checker := NewMockPermissionChecker()
	pm := NewPermissionManager(checker, eventBus)

	// 设置权限状态为拒绝
	checker.SetPermission(platform.PermissionAccessibility, platform.PermissionStatusDenied)

	// 检查权限
	status := pm.CheckPermission(platform.PermissionAccessibility)
	assert.Equal(t, platform.PermissionStatusDenied, status)
}

// TestPermissionManager_EnsurePermission 测试确保权限
//
// 验证权限已授予时 EnsurePermission 返回 nil。
func TestPermissionManager_EnsurePermission(t *testing.T) {
	eventBus := events.NewEventBus()
	checker := NewMockPermissionChecker()
	pm := NewPermissionManager(checker, eventBus)

	// 设置权限状态为已授予
	checker.SetPermission(platform.PermissionAccessibility, platform.PermissionStatusGranted)

	// 确保权限
	err := pm.EnsurePermission(platform.PermissionAccessibility)
	assert.NoError(t, err)
}

// TestPermissionManager_EnsurePermissionDenied 测试确保权限失败
//
// 验证权限被拒绝时 EnsurePermission 返回错误。
func TestPermissionManager_EnsurePermissionDenied(t *testing.T) {
	eventBus := events.NewEventBus()
	checker := NewMockPermissionChecker()
	pm := NewPermissionManager(checker, eventBus)

	// 设置权限状态为拒绝
	checker.SetPermission(platform.PermissionAccessibility, platform.PermissionStatusDenied)

	// 确保权限（应该失败）
	err := pm.EnsurePermission(platform.PermissionAccessibility)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "缺少")
}

// TestPermissionManager_CheckAndPrompt 测试检查并提示
//
// 验证权限缺失时 CheckAndPrompt 返回 false。
func TestPermissionManager_CheckAndPrompt(t *testing.T) {
	eventBus := events.NewEventBus()
	checker := NewMockPermissionChecker()
	pm := NewPermissionManager(checker, eventBus)

	// 设置权限状态为拒绝
	checker.SetPermission(platform.PermissionAccessibility, platform.PermissionStatusDenied)

	// 检查并提示（应该返回 false）
	granted := pm.CheckAndPrompt(platform.PermissionAccessibility)
	assert.False(t, granted)
}

// TestPermissionManager_CheckAndPromptGranted 测试权限已授予时的 CheckAndPrompt
//
// 验证权限已授予时 CheckAndPrompt 返回 true。
func TestPermissionManager_CheckAndPromptGranted(t *testing.T) {
	eventBus := events.NewEventBus()
	checker := NewMockPermissionChecker()
	pm := NewPermissionManager(checker, eventBus)

	// 设置权限状态为已授予
	checker.SetPermission(platform.PermissionAccessibility, platform.PermissionStatusGranted)

	// 检查并提示（应该返回 true）
	granted := pm.CheckAndPrompt(platform.PermissionAccessibility)
	assert.True(t, granted)
}

// TestPermissionManager_RequestPermission 测试请求权限
//
// 验证请求权限功能正常工作，并清除缓存。
func TestPermissionManager_RequestPermission(t *testing.T) {
	eventBus := events.NewEventBus()
	checker := NewMockPermissionChecker()
	pm := NewPermissionManager(checker, eventBus)

	// 设置权限状态为拒绝
	checker.SetPermission(platform.PermissionAccessibility, platform.PermissionStatusDenied)

	// 先检查一次（建立缓存）
	_ = pm.CheckPermission(platform.PermissionAccessibility)

	// 请求权限
	err := pm.RequestPermission(platform.PermissionAccessibility)
	assert.NoError(t, err)
	assert.True(t, checker.WasRequestCalled())
	assert.Equal(t, platform.PermissionAccessibility, checker.GetRequestedPermission())

	// 缓存应该被清除，再次检查应该获取新状态
	status := pm.CheckPermission(platform.PermissionAccessibility)
	assert.Equal(t, platform.PermissionStatusGranted, status)
}

// TestPermissionManager_OpenSystemSettings 测试打开系统设置
//
// 验证能够正确打开系统设置页面。
func TestPermissionManager_OpenSystemSettings(t *testing.T) {
	eventBus := events.NewEventBus()
	checker := NewMockPermissionChecker()
	pm := NewPermissionManager(checker, eventBus)

	// 打开系统设置
	err := pm.OpenSystemSettings(platform.PermissionAccessibility)
	assert.NoError(t, err)

	// 验证调用了底层检查器的 OpenSystemSettings
	assert.True(t, checker.openSettingsCalled)
	assert.Equal(t, platform.PermissionAccessibility, checker.openSettingsPermission)
}

// TestPermissionManager_InvalidateCache 测试清除所有缓存
//
// 验证能够正确清除所有权限缓存。
func TestPermissionManager_InvalidateCache(t *testing.T) {
	eventBus := events.NewEventBus()
	checker := NewMockPermissionChecker()
	pm := NewPermissionManager(checker, eventBus)

	// 设置权限状态并检查（建立缓存）
	checker.SetPermission(platform.PermissionAccessibility, platform.PermissionStatusGranted)
	_ = pm.CheckPermission(platform.PermissionAccessibility)

	// 清除缓存
	pm.InvalidateCache()

	// 修改底层权限状态
	checker.SetPermission(platform.PermissionAccessibility, platform.PermissionStatusDenied)

	// 再次检查应该获取新状态（缓存已清除）
	status := pm.CheckPermission(platform.PermissionAccessibility)
	assert.Equal(t, platform.PermissionStatusDenied, status)
}

// TestPermissionManager_InvalidatePermissionCache 测试清除指定权限缓存
//
// 验证能够正确清除指定权限的缓存。
func TestPermissionManager_InvalidatePermissionCache(t *testing.T) {
	eventBus := events.NewEventBus()
	checker := NewMockPermissionChecker()
	pm := NewPermissionManager(checker, eventBus)

	// 设置两个权限并检查（建立缓存）
	checker.SetPermission(platform.PermissionAccessibility, platform.PermissionStatusGranted)
	checker.SetPermission(platform.PermissionScreenCapture, platform.PermissionStatusDenied)
	_ = pm.CheckPermission(platform.PermissionAccessibility)
	_ = pm.CheckPermission(platform.PermissionScreenCapture)

	// 只清除辅助功能权限缓存
	pm.InvalidatePermissionCache(platform.PermissionAccessibility)

	// 修改辅助功能权限状态
	checker.SetPermission(platform.PermissionAccessibility, platform.PermissionStatusDenied)

	// 辅助功能权限应该获取新状态
	status := pm.CheckPermission(platform.PermissionAccessibility)
	assert.Equal(t, platform.PermissionStatusDenied, status)

	// 屏幕录制权限应该还是缓存值（未清除）
	status = pm.CheckPermission(platform.PermissionScreenCapture)
	assert.Equal(t, platform.PermissionStatusDenied, status)
}

// TestPermissionManager_SetCacheDuration 测试设置缓存有效期
//
// 验证能够正确设置缓存有效期。
func TestPermissionManager_SetCacheDuration(t *testing.T) {
	eventBus := events.NewEventBus()
	checker := NewMockPermissionChecker()
	pm := NewPermissionManager(checker, eventBus)

	// 设置较短的缓存时间
	pm.SetCacheDuration(10 * time.Millisecond)

	// 检查权限
	checker.SetPermission(platform.PermissionAccessibility, platform.PermissionStatusGranted)
	status := pm.CheckPermission(platform.PermissionAccessibility)
	assert.Equal(t, platform.PermissionStatusGranted, status)

	// 等待缓存过期
	time.Sleep(20 * time.Millisecond)

	// 修改底层权限状态
	checker.SetPermission(platform.PermissionAccessibility, platform.PermissionStatusDenied)

	// 再次检查应该获取新状态（缓存已过期）
	status = pm.CheckPermission(platform.PermissionAccessibility)
	assert.Equal(t, platform.PermissionStatusDenied, status)
}

// TestPermissionManager_PermissionEvent 测试权限事件发布
//
// 验证权限检查时能够正确发布事件。
func TestPermissionManager_PermissionEvent(t *testing.T) {
	eventBus := events.NewEventBus()
	checker := NewMockPermissionChecker()
	pm := NewPermissionManager(checker, eventBus)

	// 订阅权限事件
	eventReceived := false
	var eventData map[string]interface{}

	eventBus.Subscribe("permission", func(event events.Event) error {
		eventReceived = true
		eventData = event.Data
		return nil
	})

	// 设置权限为拒绝，然后检查
	checker.SetPermission(platform.PermissionAccessibility, platform.PermissionStatusDenied)
	pm.CheckPermission(platform.PermissionAccessibility)

	// 由于权限已授予，不会发布事件
	// 现在调用 CheckAndPrompt，会发布事件
	pm.CheckAndPrompt(platform.PermissionAccessibility)

	// 等待事件处理
	time.Sleep(50 * time.Millisecond)

	// 验证事件已发布
	assert.True(t, eventReceived, "权限事件应该被发布")
	assert.Equal(t, "accessibility", eventData["permission"])
	assert.Equal(t, "denied", eventData["status"])
	assert.NotEmpty(t, eventData["message"])
}
