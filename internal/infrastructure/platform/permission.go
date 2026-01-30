package platform

// PermissionType 权限类型枚举
//
// 定义系统需要的各种权限类型
type PermissionType int

const (
	// PermissionAccessibility 辅助功能权限
	// 用于键盘监控、窗口标题获取等功能
	PermissionAccessibility PermissionType = iota

	// PermissionScreenCapture 屏幕录制权限
	// 用于屏幕截图、屏幕录制等功能（预留）
	PermissionScreenCapture

	// PermissionFiles 文件访问权限
	// 用于访问用户文件系统（预留）
	PermissionFiles
)

// String 返回权限类型的字符串表示
func (p PermissionType) String() string {
	switch p {
	case PermissionAccessibility:
		return "accessibility"
	case PermissionScreenCapture:
		return "screen_capture"
	case PermissionFiles:
		return "files"
	default:
		return "unknown"
	}
}

// PermissionStatus 权限状态枚举
//
// 表示权限的当前状态
type PermissionStatus int

const (
	// PermissionStatusGranted 权限已授予
	PermissionStatusGranted PermissionStatus = iota

	// PermissionStatusDenied 权限被拒绝
	PermissionStatusDenied

	// PermissionStatusUnknown 权限状态未知
	PermissionStatusUnknown
)

// String 返回权限状态的字符串表示
func (s PermissionStatus) String() string {
	switch s {
	case PermissionStatusGranted:
		return "granted"
	case PermissionStatusDenied:
		return "denied"
	case PermissionStatusUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// PermissionChecker 权限检查器接口
//
// 定义了检查系统权限的方法
type PermissionChecker interface {
	// CheckPermission 检查权限状态
	// Parameters: permType - 权限类型
	// Returns: PermissionStatus - 权限状态
	CheckPermission(permType PermissionType) PermissionStatus

	// RequestPermission 请求权限
	// 显示系统权限请求对话框，或引导用户手动授权
	// Parameters: permType - 权限类型
	// Returns: error - 请求失败时返回错误
	RequestPermission(permType PermissionType) error

	// OpenSystemSettings 打开系统设置
	// 直接打开系统偏好设置中的对应权限页面
	// Parameters: permType - 权限类型
	// Returns: error - 打开失败时返回错误
	OpenSystemSettings(permType PermissionType) error
}

// PermissionResult 权限检查结果
//
// 封装权限检查的完整结果信息
type PermissionResult struct {
	// Type 权限类型
	Type PermissionType

	// Status 权限状态
	Status PermissionStatus

	// Message 权限状态描述信息
	Message string
}

// IsGranted 检查权限是否已授予
// Returns: bool - true 表示权限已授予
func (r *PermissionResult) IsGranted() bool {
	return r.Status == PermissionStatusGranted
}

// IsDenied 检查权限是否被拒绝
// Returns: bool - true 表示权限被拒绝
func (r *PermissionResult) IsDenied() bool {
	return r.Status == PermissionStatusDenied
}
