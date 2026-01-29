/**
 * Package logger 提供结构化日志功能
 *
 * 基于 uber-go/zap 实现的高性能结构化日志系统。
 * 支持开发环境和生产环境的不同配置。
 */
package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// logger 全局日志实例
	logger *zap.Logger

	// once 确保日志只初始化一次
	once sync.Once

	// sugar 全局 sugared logger 实例（更方便使用）
	sugar *zap.SugaredLogger
)

// InitLogger 初始化日志系统
//
// 根据环境变量配置日志系统：
//   - 开发环境：控制台彩色输出，Debug 级别
//   - 生产环境：JSON 格式，Info 级别
//
// 环境变量：
//   - ENV: 环境类型（development/production），默认为 development
//   - LOG_LEVEL: 日志级别（debug/info/warn/error/fatal），默认根据环境自动设置
//
// Returns: error - 初始化失败时返回错误
func InitLogger() error {
	var initErr error
	once.Do(func() {
		env := getEnv("ENV", "development")

		if env == "production" {
			logger, initErr = initProductionLogger()
		} else {
			logger, initErr = initDevelopmentLogger()
		}

		if initErr != nil {
			return
		}

		sugar = logger.Sugar()
	})

	return initErr
}

// initDevelopmentLogger 初始化开发环境日志
//
// 开发环境配置：
//   - 控制台输出
//   - 彩色格式（易于阅读）
//   - Debug 级别（详细信息）
//   - 时间戳、调用者信息
//   - 友好的时间格式（2024-01-29 15:04:05.123）
//
// Returns:
//   - *zap.Logger: 配置好的 logger
//   - error: 初始化失败时返回错误
func initDevelopmentLogger() (*zap.Logger, error) {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    "",
		MessageKey:     "msg",
		StacktraceKey:   "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.999"),
		EncodeDuration:  zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 从环境变量读取日志级别
	level := getEnv("LOG_LEVEL", "debug")
	atomicLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		atomicLevel = zapcore.DebugLevel
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(atomicLevel),
		Development:      true,
		Encoding:         "console",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build(
		zap.AddCallerSkip(0), // 不跳过任何调用栈
	)
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// initProductionLogger 初始化生产环境日志
//
// 生产环境配置：
//   - JSON 格式（机器可解析）
//   - Info 级别（避免过多日志）
//   - 时间戳、调用者信息、堆栈跟踪
//   - 可选的文件输出
//
// Returns:
//   - *zap.Logger: 配置好的 logger
//   - error: 初始化失败时返回错误
func initProductionLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// 从环境变量读取日志级别
	level := getEnv("LOG_LEVEL", "info")
	atomicLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		atomicLevel = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(atomicLevel)

	// 检查是否需要输出到文件
	logFile := getEnv("LOG_FILE", "")
	if logFile != "" {
		// 如果指定了日志文件，同时输出到控制台和文件
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}

		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(config.EncoderConfig),
			zapcore.AddSync(file),
			config.Level,
		)

		logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
		return logger, nil
	}

	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(config.EncoderConfig),
		zapcore.AddSync(os.Stdout),
		config.Level,
	), zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, nil
}

// GetLogger 获取全局 logger 实例
//
// 如果日志系统未初始化，会自动初始化（开发模式）。
//
// Returns: *zap.Logger - 全局 logger 实例
func GetLogger() *zap.Logger {
	if logger == nil {
		_ = InitLogger()
	}
	return logger
}

// GetSugaredLogger 获取全局 sugared logger 实例
//
// Sugared logger 提供了更方便的 API，但性能略低。
// 适合非关键路径的日志记录。
//
// Returns: *zap.SugaredLogger - 全局 sugared logger 实例
func GetSugaredLogger() *zap.SugaredLogger {
	if sugar == nil {
		_ = InitLogger()
	}
	return sugar
}

// Sync 刷新日志缓冲区
//
// 应用退出前应该调用此方法确保所有日志都已写入。
// Returns: error - 刷新失败时返回错误
func Sync() error {
	if logger != nil {
		return logger.Sync()
	}
	return nil
}

// Debug 记录 Debug 级别日志
//
// 使用便利函数记录日志，无需手动获取 logger。
// Parameters:
//   - msg: 日志消息
//   - fields: 日志字段（可选）
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Info 记录 Info 级别日志
//
// 使用便利函数记录日志，无需手动获取 logger。
// Parameters:
//   - msg: 日志消息
//   - fields: 日志字段（可选）
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Warn 记录 Warn 级别日志
//
// 使用便利函数记录日志，无需手动获取 logger。
// Parameters:
//   - msg: 日志消息
//   - fields: 日志字段（可选）
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Error 记录 Error 级别日志
//
// 使用便利函数记录日志，无需手动获取 logger。
// Parameters:
//   - msg: 日志消息
//   - fields: 日志字段（可选）
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// Fatal 记录 Fatal 级别日志后退出程序
//
// 使用便利函数记录日志，无需手动获取 logger。
// 记录日志后会调用 os.Exit(1)。
// Parameters:
//   - msg: 日志消息
//   - fields: 日志字段（可选）
func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// With 创建带有预设字段的 logger
//
// 用于在日志中自动添加上下文信息（如请求 ID、用户 ID 等）。
// Parameters:
//   - fields: 预设的日志字段
//
// Returns: *zap.Logger - 带有预设字段的 logger
func With(fields ...zap.Field) *zap.Logger {
	return GetLogger().With(fields...)
}

// getEnv 获取环境变量
//
// 从系统环境变量中读取配置，如果不存在则返回默认值。
// Parameters:
//   - key: 环境变量名
//   - defaultValue: 默认值
//
// Returns: string - 环境变量值或默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
