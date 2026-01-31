/**
 * Package analyzer 模式识别引擎的分析组件
 *
 * AnalyzerEngine - 协调整个分析流程的主控制器
 */

package analyzer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chenyang-zz/flowmind/internal/domain/models"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/storage"
	"github.com/chenyang-zz/flowmind/pkg/events"
	"go.uber.org/zap"
)

/**
 * AnalyzerEngineConfig 分析引擎配置
 */
type AnalyzerEngineConfig struct {
	// PatternMiner 模式挖掘器配置
	PatternMiner PatternMinerConfig

	// SessionDivider 会话划分配置
	SessionDivider SessionDividerConfig

	// AIPatternFilter AI 过滤器配置
	AIPatternFilter AIPatternFilterConfig

	// AnalysisInterval 分析间隔（默认1小时）
	AnalysisInterval time.Duration

	// MinEventCount 最小事件数（少于此次数不分析）
	MinEventCount int

	// EnableAIAnalysis 是否启用 AI 分析
	EnableAIAnalysis bool
}

/**
 * DefaultAnalyzerEngineConfig 默认分析引擎配置
 */
func DefaultAnalyzerEngineConfig() AnalyzerEngineConfig {
	return AnalyzerEngineConfig{
		PatternMiner:     DefaultPatternMinerConfig(),
		SessionDivider:   DefaultSessionDividerConfig(),
		AIPatternFilter:  DefaultAIPatternFilterConfig(),
		AnalysisInterval: 1 * time.Hour,
		MinEventCount:    10,
		EnableAIAnalysis: true,
	}
}

/**
 * AnalyzerEngine 分析引擎主控制器
 *
 * 协调整个分析流程：事件 → 会话划分 → 模式挖掘 → AI 过滤
 */
type AnalyzerEngine struct {
	config       AnalyzerEngineConfig
	patternMiner *PatternMiner
	aiFilter     *AIPatternFilter
	eventRepo    storage.EventRepository
	patternRepo  models.PatternRepository
	eventBus     *events.EventBus

	// 调度相关
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 状态
	isRunning bool
	mu        sync.RWMutex

	// 增量分析状态
	lastAnalyzedAt time.Time
}

/**
 * AnalysisResult 分析结果
 */
type AnalysisResult struct {
	// EventCount 分析的事件数
	EventCount int

	// SessionCount 发现的会话数
	SessionCount int

	// PatternCount 挖掘的模式数
	PatternCount int

	// ValuablePatterns 值得自动化的模式数
	ValuablePatterns int

	// AnalyzedPatterns AI 分析的模式数
	AnalyzedPatterns int

	// Duration 分析耗时
	Duration time.Duration
}

/**
 * NewAnalyzerEngine 创建分析引擎
 *
 * Parameters:
 *   - config: 引擎配置
 *   - eventRepo: 事件仓储
 *   - patternRepo: 模式仓储
 *   - eventBus: 事件总线
 *
 * Returns: *AnalyzerEngine - 分析引擎实例
 */
func NewAnalyzerEngine(
	config AnalyzerEngineConfig,
	eventRepo storage.EventRepository,
	patternRepo models.PatternRepository,
	eventBus *events.EventBus,
) (*AnalyzerEngine, error) {
	if eventRepo == nil {
		return nil, fmt.Errorf("事件仓储不能为空")
	}
	if patternRepo == nil {
		return nil, fmt.Errorf("模式仓储不能为空")
	}
	if eventBus == nil {
		return nil, fmt.Errorf("事件总线不能为空")
	}

	// 创建模式挖掘器
	patternMiner := NewPatternMiner(config.PatternMiner)

	// 创建 AI 过滤器
	aiFilter, err := NewAIPatternFilter(config.AIPatternFilter)
	if err != nil {
		return nil, fmt.Errorf("创建 AI 过滤器失败: %w", err)
	}

	return &AnalyzerEngine{
		config:        config,
		patternMiner:  patternMiner,
		aiFilter:      aiFilter,
		eventRepo:     eventRepo,
		patternRepo:   patternRepo,
		eventBus:      eventBus,
		isRunning:     false,
		lastAnalyzedAt: time.Time{}, // 初始化为零值，表示分析所有历史事件
	}, nil
}

/**
 * Start 启动分析引擎
 *
 * 开始定时分析循环
 *
 * Returns: error - 错误信息
 */
func (e *AnalyzerEngine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.isRunning {
		logger.Warn("分析引擎已在运行")
		return fmt.Errorf("分析引擎已在运行")
	}

	logger.Info("启动分析引擎",
		zap.Duration("interval", e.config.AnalysisInterval),
		zap.Int("min_event_count", e.config.MinEventCount))

	e.ctx, e.cancel = context.WithCancel(context.Background())
	e.isRunning = true

	// 启动定时分析循环
	e.wg.Add(1)
	go e.analysisLoop()

	logger.Info("分析引擎已启动")
	return nil
}

/**
 * Stop 停止分析引擎
 *
 * 优雅关闭，等待当前分析完成
 *
 * Returns: error - 错误信息
 */
func (e *AnalyzerEngine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning {
		logger.Warn("分析引擎未运行")
		return fmt.Errorf("分析引擎未运行")
	}

	logger.Info("正在停止分析引擎...")

	// 取消上下文
	e.cancel()

	// 等待分析完成
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("分析引擎已停止")
		return nil
	case <-time.After(30 * time.Second):
		logger.Warn("分析引擎停止超时")
		return fmt.Errorf("停止超时")
	}
}

/**
 * IsRunning 检查运行状态
 *
 * Returns: bool - 是否正在运行
 */
func (e *AnalyzerEngine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isRunning
}

/**
 * analysisLoop 分析循环
 *
 * 定时触发分析任务
 */
func (e *AnalyzerEngine) analysisLoop() {
	defer e.wg.Done()

	// 立即执行一次分析
	e.runAnalysis(e.ctx)

	ticker := time.NewTicker(e.config.AnalysisInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.runAnalysis(e.ctx)
		case <-e.ctx.Done():
			logger.Info("分析循环停止")
			return
		}
	}
}

/**
 * runAnalysis 执行分析
 *
 * Parameters:
 *   - ctx: 上下文
 */
func (e *AnalyzerEngine) runAnalysis(ctx context.Context) {
	logger.Info("开始增量分析",
		zap.Time("last_analyzed", e.lastAnalyzedAt))

	startTime := time.Now()

	// 1. 读取新事件
	events, err := e.eventRepo.FindByTimeRange(
		e.lastAnalyzedAt,
		time.Now(),
	)
	if err != nil {
		logger.Error("读取新事件失败", zap.Error(err))
		return
	}

	if len(events) < e.config.MinEventCount {
		logger.Debug("事件数量不足，跳过分析",
			zap.Int("count", len(events)),
			zap.Int("min_required", e.config.MinEventCount))
		return
	}

	logger.Info("发现新事件",
		zap.Int("count", len(events)),
		zap.Time("from", e.lastAnalyzedAt),
		zap.Time("to", time.Now()))

	// 2. 执行分析
	result, err := e.AnalyzeNewEvents(ctx)
	if err != nil {
		logger.Error("分析失败", zap.Error(err))
		return
	}

	// 3. 更新最后分析时间
	e.lastAnalyzedAt = time.Now()

	duration := time.Since(startTime)

	logger.Info("分析完成",
		zap.Int("events", result.EventCount),
		zap.Int("sessions", result.SessionCount),
		zap.Int("patterns", result.PatternCount),
		zap.Int("valuable_patterns", result.ValuablePatterns),
		zap.Int("analyzed_patterns", result.AnalyzedPatterns),
		zap.Duration("duration", duration))

	// 4. 发布分析完成事件
	e.publishAnalysisResult(result)
}

/**
 * AnalyzeNewEvents 分析新事件
 *
 * 从上次分析时间到现在的事件
 *
 * Parameters:
 *   - ctx: 上下文
 *
 * Returns: *AnalysisResult - 分析结果, error - 错误信息
 */
func (e *AnalyzerEngine) AnalyzeNewEvents(ctx context.Context) (*AnalysisResult, error) {
	return e.AnalyzeRange(ctx, e.lastAnalyzedAt, time.Now())
}

/**
 * AnalyzeRange 分析指定时间范围的事件
 *
 * Parameters:
 *   - ctx: 上下文
 *   - start: 开始时间
 *   - end: 结束时间
 *
 * Returns: *AnalysisResult - 分析结果, error - 错误信息
 */
func (e *AnalyzerEngine) AnalyzeRange(
	ctx context.Context,
	start, end time.Time,
) (*AnalysisResult, error) {
	result := &AnalysisResult{}
	startTime := time.Now()

	// 1. 读取事件
	events, err := e.eventRepo.FindByTimeRange(start, end)
	if err != nil {
		return nil, fmt.Errorf("读取事件失败: %w", err)
	}

	result.EventCount = len(events)
	if result.EventCount == 0 {
		return result, nil
	}

	// 2. 划分会话
	sessionDivider := NewSessionDivider(e.config.SessionDivider)
	sessions := sessionDivider.Divide(events)
	result.SessionCount = len(sessions)

	if result.SessionCount == 0 {
		return result, nil
	}

	// 3. 挖掘模式
	patterns, err := e.patternMiner.MineFromSessions(sessions)
	if err != nil {
		return nil, fmt.Errorf("模式挖掘失败: %w", err)
	}

	result.PatternCount = len(patterns)
	if result.PatternCount == 0 {
		return result, nil
	}

	// 4. 保存模式到数据库
	err = e.patternRepo.SaveBatch(patterns)
	if err != nil {
		return nil, fmt.Errorf("保存模式失败: %w", err)
	}

	// 5. AI 分析（如果启用）
	if e.config.EnableAIAnalysis {
		analyzedPatterns, err := e.analyzePatternsWithAI(ctx, patterns)
		if err != nil {
			logger.Warn("AI 分析失败", zap.Error(err))
		} else {
			result.AnalyzedPatterns = analyzedPatterns

			// 统计值得自动化的模式
			for _, pattern := range patterns {
				if pattern.AIAnalysis != nil && pattern.AIAnalysis.ShouldAutomate {
					result.ValuablePatterns++
				}
			}
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

/**
 * analyzePatternsWithAI 使用 AI 分析模式
 *
 * Parameters:
 *   - ctx: 上下文
 *   - patterns: 模式列表
 *
 * Returns: int - 分析的模式数, error - 错误信息
 */
func (e *AnalyzerEngine) analyzePatternsWithAI(
	ctx context.Context,
	patterns []*models.Pattern,
) (int, error) {
	// 过滤出未分析的模式
	unanalyzed := make([]*models.Pattern, 0)
	for _, pattern := range patterns {
		if pattern.AIAnalysis == nil {
			unanalyzed = append(unanalyzed, pattern)
		}
	}

	if len(unanalyzed) == 0 {
		return 0, nil
	}

	logger.Info("开始 AI 分析模式", zap.Int("count", len(unanalyzed)))

	// 批量分析（自动使用缓存）
	results, err := e.aiFilter.ShouldAutomateBatch(ctx, unanalyzed)
	if err != nil {
		return 0, fmt.Errorf("AI 分析失败: %w", err)
	}

	// 更新模式并保存
	for _, pattern := range unanalyzed {
		if analysis, ok := results[pattern.ID]; ok {
			pattern.AIAnalysis = analysis
			if err := e.patternRepo.Update(pattern); err != nil {
				logger.Error("更新模式失败",
					zap.String("pattern_id", pattern.ID),
					zap.Error(err))
			}
		}
	}

	return len(unanalyzed), nil
}

/**
 * publishAnalysisResult 发布分析结果到事件总线
 *
 * Parameters:
 *   - result: 分析结果
 */
func (e *AnalyzerEngine) publishAnalysisResult(result *AnalysisResult) {
	statusEvent := events.NewEvent(events.EventTypeStatus, map[string]interface{}{
		"type":              "analysis_completed",
		"event_count":       result.EventCount,
		"session_count":     result.SessionCount,
		"pattern_count":     result.PatternCount,
		"valuable_patterns": result.ValuablePatterns,
		"analyzed_patterns": result.AnalyzedPatterns,
		"duration":          result.Duration.String(),
	})

	e.eventBus.Publish(string(events.EventTypeStatus), *statusEvent)
}

/**
 * GetLastAnalyzedTime 获取最后分析时间
 *
 * Returns: time.Time - 最后分析时间
 */
func (e *AnalyzerEngine) GetLastAnalyzedTime() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastAnalyzedAt
}

/**
 * Close 关闭引擎并释放资源
 */
func (e *AnalyzerEngine) Close() error {
	if e.aiFilter != nil {
		e.aiFilter.Close()
	}
	return nil
}
