/**
 * Package logger 日志系统测试
 */
package logger

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// resetLogger 重置全局 logger 状态（仅用于测试）
func resetLogger() {
	// 通过创建新的包来重置 once
	// 这不是理想的解决方案，但是 Go 的 sync.Once 不支持重置
	// 在实际测试中，我们只运行一次初始化，或者使用子测试
	logger = nil
	sugar = nil
}

// resetOnce 重置 sync.Once（仅用于测试）
// 在测试中我们需要重置全局的 sync.Once 以便测试不同环境
// 虽然这不是最佳实践，但对于单元测试是必要的
func resetOnce() {
	// 创建新的 sync.Once，通过直接赋值重置
	// govet 会警告复制锁值，但对于测试这是可以接受的
	once = sync.Once{} //nolint:all
}

// TestInitLogger 测试日志系统初始化
//
// 验证日志系统能够正确初始化，并且可以在不同环境下运行。
// 测试场景：
//  1. 开发环境初始化
//  2. 生产环境初始化
//  3. 重复初始化（幂等性）
func TestInitLogger(t *testing.T) {
	// 重置全局变量
	resetOnce()
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
		resetOnce()
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
		resetOnce()
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
	resetOnce()
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
	resetOnce()
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
	resetOnce()
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
	resetOnce()
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
	resetOnce()
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

// TestProductionLoggerWithRotation 测试生产环境滚动日志
//
// 验证生产环境下配置滚动日志参数不会报错。
func TestProductionLoggerWithRotation(t *testing.T) {
	// 重置全局变量
	resetOnce()
	logger = nil
	sugar = nil

	// 设置生产环境配置
	os.Setenv("ENV", "production")
	os.Setenv("LOG_FILE", "/tmp/test.log")
	os.Setenv("LOG_MAX_SIZE", "10")
	os.Setenv("LOG_MAX_BACKUPS", "5")
	os.Setenv("LOG_MAX_AGE", "30")
	os.Setenv("LOG_COMPRESS", "true")
	defer func() {
		os.Unsetenv("ENV")
		os.Unsetenv("LOG_FILE")
		os.Unsetenv("LOG_MAX_SIZE")
		os.Unsetenv("LOG_MAX_BACKUPS")
		os.Unsetenv("LOG_MAX_AGE")
		os.Unsetenv("LOG_COMPRESS")
		_ = os.Remove("/tmp/test.log")
	}()

	// 初始化日志系统
	err := InitLogger()
	require.NoError(t, err, "初始化带滚动日志的生产环境不应失败")

	assert.NotNil(t, logger, "logger 不应为 nil")

	// 记录一些日志测试
	Info("测试滚动日志", zap.String("test", "rotation"))
}

// TestGetEnvInt 测试 getEnvInt 辅助函数
//
// 验证整数环境变量解析功能。
func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		expected     int
	}{
		{"有效整数", "100", 10, 100},
		{"无效值", "invalid", 10, 10},
		{"空值", "", 10, 10},
		{"负数", "-5", 10, -5},
		{"零", "0", 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("TEST_INT", tt.envValue)
				defer os.Unsetenv("TEST_INT")
			}
			result := getEnvInt("TEST_INT", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetEnvBool 测试 getEnvBool 辅助函数
//
// 验证布尔环境变量解析功能。
func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{"true", "true", false, true},
		{"True", "True", false, true},
		{"TRUE", "TRUE", false, true},
		{"1", "1", false, true},
		{"yes", "yes", false, true},
		{"false", "false", true, false},
		{"False", "False", true, false},
		{"0", "0", true, false},
		{"no", "no", true, false},
		{"空值", "", true, true},
		{"无效值", "invalid", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("TEST_BOOL", tt.envValue)
				defer os.Unsetenv("TEST_BOOL")
			}
			result := getEnvBool("TEST_BOOL", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestLogOutput 测试日志输出
//
// 这是一个手动的集成测试，用于验证日志输出是否符合预期。
// 运行此测试时应该能在控制台看到彩色输出（开发模式）。
func TestLogOutput(t *testing.T) {
	// 重置全局变量
	resetOnce()
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
