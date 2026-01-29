/**
 * Package logger 日志系统测试
 */
package logger

import (
	"os"
	"sync"
	"testing"

	"go.uber.org/zap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitLogger 测试日志系统初始化
//
// 验证日志系统能够正确初始化，并且可以在不同环境下运行。
// 测试场景：
//   1. 开发环境初始化
//   2. 生产环境初始化
//   3. 重复初始化（幂等性）
func TestInitLogger(t *testing.T) {
	// 重置全局变量
	once = *(new(sync.Once))
	logger = nil
	sugar = nil

	t.Run("开发环境初始化", func(t *testing.T) {
		os.Setenv("ENV", "development")
		defer os.Unsetenv("ENV")

		err := InitLogger()
		require.NoError(t, err, "初始化日志系统不应失败")

		assert.NotNil(t, logger, "logger 不应为 nil")
		assert.NotNil(t, sugar, "sugar logger 不应为 nil")
	})

	t.Run("生产环境初始化", func(t *testing.T) {
		// 重置全局变量
		once = *(new(sync.Once))
		logger = nil
		sugar = nil

		os.Setenv("ENV", "production")
		defer os.Unsetenv("ENV")

		err := InitLogger()
		require.NoError(t, err, "初始化日志系统不应失败")

		assert.NotNil(t, logger, "logger 不应为 nil")
		assert.NotNil(t, sugar, "sugar logger 不应为 nil")
	})

	t.Run("重复初始化（幂等性）", func(t *testing.T) {
		// 重置全局变量
		once = *(new(sync.Once))
		logger = nil
		sugar = nil

		os.Setenv("ENV", "development")
		defer os.Unsetenv("ENV")

		// 第一次初始化
		err := InitLogger()
		require.NoError(t, err, "第一次初始化不应失败")

		firstLogger := logger

		// 第二次初始化（应该被忽略）
		err = InitLogger()
		require.NoError(t, err, "第二次初始化不应失败")

		assert.Equal(t, firstLogger, logger, "重复初始化应该返回同一个实例")
	})
}

// TestGetLogger 测试获取 logger 实例
//
// 验证可以在未初始化的情况下自动初始化。
func TestGetLogger(t *testing.T) {
	// 重置全局变量
	once = *(new(sync.Once))
	logger = nil
	sugar = nil

	os.Setenv("ENV", "development")
	defer os.Unsetenv("ENV")

	// 未初始化时调用 GetLogger，应该自动初始化
	l := GetLogger()
	assert.NotNil(t, l, "logger 不应为 nil")
}

// TestGetSugaredLogger 测试获取 sugared logger 实例
//
// 验证可以在未初始化的情况下自动初始化。
func TestGetSugaredLogger(t *testing.T) {
	// 重置全局变量
	once = *(new(sync.Once))
	logger = nil
	sugar = nil

	os.Setenv("ENV", "development")
	defer os.Unsetenv("ENV")

	// 未初始化时调用 GetSugaredLogger，应该自动初始化
	s := GetSugaredLogger()
	assert.NotNil(t, s, "sugar logger 不应为 nil")
}

// TestConvenienceFunctions 测试便利函数
//
// 验证便利函数能够正常工作且不会 panic。
func TestConvenienceFunctions(t *testing.T) {
	// 重置全局变量
	once = *(new(sync.Once))
	logger = nil
	sugar = nil

	os.Setenv("ENV", "development")
	defer os.Unsetenv("ENV")

	// 初始化日志系统
	err := InitLogger()
	require.NoError(t, err)

	// 测试便利函数不会 panic
	t.Run("Debug", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Debug("test debug message", zap.String("key", "value"))
		})
	})

	t.Run("Info", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Info("test info message", zap.String("key", "value"))
		})
	})

	t.Run("Warn", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Warn("test warn message", zap.String("key", "value"))
		})
	})

	t.Run("Error", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Error("test error message", zap.String("key", "value"))
		})
	})
}

// TestWith 测试创建带有预设字段的 logger
//
// 验证 With 函数能够创建带有预设字段的 logger。
func TestWith(t *testing.T) {
	// 重置全局变量
	once = *(new(sync.Once))
	logger = nil
	sugar = nil

	os.Setenv("ENV", "development")
	defer os.Unsetenv("ENV")

	// 初始化日志系统
	err := InitLogger()
	require.NoError(t, err)

	// 创建带有预设字段的 logger
	fields := []zap.Field{
		zap.String("service", "test-service"),
		zap.String("version", "1.0.0"),
	}

	l := With(fields...)
	assert.NotNil(t, l, "带有预设字段的 logger 不应为 nil")

	// 测试日志输出
	assert.NotPanics(t, func() {
		l.Info("test message with fields")
	})
}

// TestSync 测试日志刷新
//
// 验证 Sync 函数能够正常工作。
func TestSync(t *testing.T) {
	// 重置全局变量
	once = *(new(sync.Once))
	logger = nil
	sugar = nil

	os.Setenv("ENV", "development")
	defer os.Unsetenv("ENV")

	// 初始化日志系统
	err := InitLogger()
	require.NoError(t, err)

	// 记录一些日志
	Info("message before sync")

	// 刷新缓冲区
	err = Sync()
	// 在测试环境中，stdout sync 可能会失败，这是正常的
	// 只要不 panic 就说明 Sync 函数本身工作正常
	if err != nil {
		t.Logf("Sync 返回错误（测试环境正常现象）: %v", err)
	}
}

// TestLogOutput 测试日志输出
//
// 这是一个手动的集成测试，用于验证日志输出是否符合预期。
// 运行此测试时应该能在控制台看到彩色输出（开发模式）。
func TestLogOutput(t *testing.T) {
	// 重置全局变量
	once = *(new(sync.Once))
	logger = nil
	sugar = nil

	os.Setenv("ENV", "development")
	defer os.Unsetenv("ENV")

	t.Log("===== 开发环境日志输出测试 =====")

	err := InitLogger()
	require.NoError(t, err)

	// 输出各级别日志
	Debug("这是 Debug 级别日志", zap.Int("level", 1))
	Info("这是 Info 级别日志", zap.Int("level", 2))
	Warn("这是 Warn 级别日志", zap.Int("level", 3))
	Error("这是 Error 级别日志", zap.Int("level", 4))

	t.Log("===== 如果看到彩色输出，说明日志系统工作正常 =====")
}
