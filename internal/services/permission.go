package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/platform"
	"github.com/chenyang-zz/flowmind/pkg/events"
	"go.uber.org/zap"
)

// PermissionManager 权限管理器
//
// 负责检查和管理系统权限，提供权限缓存和事件发布功能。
// 主要用于在监控器启动前验证权限，确保系统正常运行。
type PermissionManager struct {
	// checker 平台层权限检查器
	checker platform.PermissionChecker

	// eventBus 事件总线，用于发布权限事件
	eventBus *events.EventBus

	// cache 权限状态缓存，避免频繁检查系统权限
	cache map[platform.PermissionType]platform.PermissionStatus

	// cacheExpire 缓存过期时间
	cacheExpire map[platform.PermissionType]time.Time

	// mu 互斥锁，保护并发访问
	mu sync.RWMutex

	// cacheDuration 缓存有效期（默认 5 分钟）
	cacheDuration time.Duration
}

// NewPermissionManager 创建权限管理器
//
// 创建一个新的权限管理器实例，初始化权限检查器和事件总线。
//
// Parameters:
//   - checker: 平台层权限检查器实例
//   - eventBus: 事件总线实例，用于发布权限事件
//
// Returns: *PermissionManager - 新创建的权限管理器实例
func NewPermissionManager(checker platform.PermissionChecker, eventBus *events.EventBus) *PermissionManager {
	return &PermissionManager{
		checker:      checker,
		eventBus:     eventBus,
		cache:        make(map[platform.PermissionType]platform.PermissionStatus),
		cacheExpire:  make(map[platform.PermissionType]time.Time),
		cacheDuration: 5 * time.Minute, // 默认缓存 5 分钟
	}
}

// CheckPermission 检查权限状态
//
// 检查指定类型的权限状态，优先从缓存获取。
// 如果缓存不存在或已过期，则调用平台层检查器进行实际检查。
//
// Parameters:
//   - permType: 权限类型
//
// Returns: PermissionStatus - 权限状态
func (pm *PermissionManager) CheckPermission(permType platform.PermissionType) platform.PermissionStatus {
	pm.mu.RLock()
	// 检查缓存是否有效
	if expire, ok := pm.cacheExpire[permType]; ok && time.Now().Before(expire) {
		status := pm.cache[permType]
		pm.mu.RUnlock()
		logger.Debug("权限状态（缓存）",
			zap.String("permission", permType.String()),
			zap.String("status", status.String()),
		)
		return status
	}
	pm.mu.RUnlock()

	// 缓存不存在或已过期，调用平台层检查器
	pm.mu.Lock()
	defer pm.mu.Unlock()

	status := pm.checker.CheckPermission(permType)

	// 更新缓存
	pm.cache[permType] = status
	pm.cacheExpire[permType] = time.Now().Add(pm.cacheDuration)

	logger.Debug("权限状态（检查）",
		zap.String("permission", permType.String()),
		zap.String("status", status.String()),
	)

	return status
}

// EnsurePermission 确保权限已授予
//
// 检查权限状态，如果权限被拒绝，则引导用户授予权限。
// 这是监控器启动前的主要权限验证方法。
//
// Parameters:
//   - permType: 权限类型
//
// Returns: error - 权限未授予时返回错误
func (pm *PermissionManager) EnsurePermission(permType platform.PermissionType) error {
	status := pm.CheckPermission(permType)

	if status == platform.PermissionStatusGranted {
		return nil
	}

	// 权限未授予，返回详细错误
	errMsg := fmt.Sprintf("缺少 %s 权限，请在系统设置中授予权限", permType.String())
	logger.Warn("权限检查失败",
		zap.String("permission", permType.String()),
		zap.String("status", status.String()),
	)

	// 发布权限事件
	pm.publishPermissionEvent(permType, status, errMsg)

	return fmt.Errorf(errMsg)
}

// CheckAndPrompt 检查权限并在缺失时提示用户
//
// 检查权限状态，如果权限被拒绝，则显示用户友好的提示并引导授权。
// 返回布尔值表示权限是否已授予。
//
// Parameters:
//   - permType: 权限类型
//
// Returns: bool - true 表示权限已授予，false 表示需要用户授权
func (pm *PermissionManager) CheckAndPrompt(permType platform.PermissionType) bool {
	status := pm.CheckPermission(permType)

	if status == platform.PermissionStatusGranted {
		return true
	}

	// 权限未授予，显示提示
	logger.Warn("权限缺失提示",
		zap.String("permission", permType.String()),
		zap.String("message", pm.getPermissionHint(permType)),
	)

	// 发布权限事件
	pm.publishPermissionEvent(permType, status, pm.getPermissionHint(permType))

	return false
}

// RequestPermission 请求权限
//
// 显示系统权限请求对话框，或引导用户手动授权。
// 调用后建议使用 OpenSystemSettings 打开系统设置页面。
//
// Parameters:
//   - permType: 权限类型
//
// Returns: error - 请求失败时返回错误
func (pm *PermissionManager) RequestPermission(permType platform.PermissionType) error {
	logger.Info("请求权限", zap.String("permission", permType.String()))

	err := pm.checker.RequestPermission(permType)
	if err != nil {
		logger.Error("请求权限失败",
			zap.String("permission", permType.String()),
			zap.Error(err),
		)
		return err
	}

	// 清除缓存，强制下次检查时重新获取状态
	pm.InvalidatePermissionCache(permType)

	logger.Info("权限请求已发送",
		zap.String("permission", permType.String()),
		zap.String("message", "请在系统设置中完成授权"),
	)

	return nil
}

// OpenSystemSettings 打开系统设置
//
// 直接打开系统偏好设置中的对应权限页面，方便用户手动授权。
//
// Parameters:
//   - permType: 权限类型
//
// Returns: error - 打开失败时返回错误
func (pm *PermissionManager) OpenSystemSettings(permType platform.PermissionType) error {
	logger.Info("打开系统设置", zap.String("permission", permType.String()))

	err := pm.checker.OpenSystemSettings(permType)
	if err != nil {
		logger.Error("打开系统设置失败",
			zap.String("permission", permType.String()),
			zap.Error(err),
		)
		return err
	}

	logger.Info("系统设置已打开",
		zap.String("permission", permType.String()),
	)

	return nil
}

// InvalidateCache 清除权限缓存
//
// 清除所有权限类型的缓存，强制下次检查时重新获取状态。
func (pm *PermissionManager) InvalidateCache() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 清除所有缓存
	pm.cache = make(map[platform.PermissionType]platform.PermissionStatus)
	pm.cacheExpire = make(map[platform.PermissionType]time.Time)
	logger.Debug("已清除所有权限缓存")
}

// InvalidatePermissionCache 清除指定权限的缓存
//
// 清除指定权限类型的缓存，强制下次检查时重新获取状态。
//
// Parameters: permType - 权限类型
func (pm *PermissionManager) InvalidatePermissionCache(permType platform.PermissionType) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.cache, permType)
	delete(pm.cacheExpire, permType)
	logger.Debug("已清除权限缓存", zap.String("permission", permType.String()))
}

// SetCacheDuration 设置缓存有效期
//
// 设置权限状态的缓存时间，避免频繁检查系统权限。
//
// Parameters: duration - 缓存有效期
func (pm *PermissionManager) SetCacheDuration(duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.cacheDuration = duration
	logger.Debug("设置权限缓存有效期", zap.Duration("duration", duration))
}

// publishPermissionEvent 发布权限事件
//
// 发布权限状态变化事件到事件总线。
//
// Parameters:
//   - permType: 权限类型
//   - status: 权限状态
//   - message: 权限状态描述信息
func (pm *PermissionManager) publishPermissionEvent(permType platform.PermissionType, status platform.PermissionStatus, message string) {
	event := events.NewEvent(events.EventTypePermission, map[string]interface{}{
		"permission": permType.String(),
		"status":     status.String(),
		"message":    message,
	})

	pm.eventBus.Publish(string(events.EventTypePermission), *event)

	logger.Debug("权限事件已发布",
		zap.String("permission", permType.String()),
		zap.String("status", status.String()),
	)
}

// getPermissionHint 获取权限提示信息
//
// 返回用户友好的权限提示，指导用户如何授予权限。
//
// Parameters: permType - 权限类型
//
// Returns: string - 权限提示信息
func (pm *PermissionManager) getPermissionHint(permType platform.PermissionType) string {
	switch permType {
	case platform.PermissionAccessibility:
		return "需要辅助功能权限来监控键盘输入和获取窗口标题。" +
			"请在【系统偏好设置 > 安全性与隐私 > 辅助功能】中启用此应用。"

	case platform.PermissionScreenCapture:
		return "需要屏幕录制权限来捕获屏幕内容。" +
			"请在【系统偏好设置 > 安全性与隐私 > 屏幕录制】中启用此应用。"

	case platform.PermissionFiles:
		return "需要文件访问权限来读取和写入文件。" +
			"请在【系统偏好设置 > 安全性与隐私 > 完全磁盘访问权限】中启用此应用。"

	default:
		return "需要相关权限才能正常工作。"
	}
}
