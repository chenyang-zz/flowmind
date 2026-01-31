/**
 * Package ai AI 服务基础设施层
 *
 * 智谱AI客户端单元测试
 */

package ai

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewZhipuClient 测试创建智谱AI客户端
func TestNewZhipuClient(t *testing.T) {
	// 保存原始环境变量
	origAPIKey := os.Getenv("ZHIPU_API_KEY")
	origModel := os.Getenv("ZHIPU_MODEL")
	t.Cleanup(func() {
		if origAPIKey != "" {
			os.Setenv("ZHIPU_API_KEY", origAPIKey)
		} else {
			os.Unsetenv("ZHIPU_API_KEY")
		}
		if origModel != "" {
			os.Setenv("ZHIPU_MODEL", origModel)
		} else {
			os.Unsetenv("ZHIPU_MODEL")
		}
	})

	tests := []struct {
		name        string
		config      *ZhipuConfig
		expectError bool
	}{
		{
			name: "有效配置",
			config: &ZhipuConfig{
				APIKey:      "test-api-key",
				Model:       "glm-4",
				MaxTokens:   4096,
				Temperature: nil,
				Timeout:     30 * time.Second,
			},
			expectError: false,
		},
		{
			name: "配置为空",
			config: &ZhipuConfig{
				APIKey:    "",
				Model:     "",
				MaxTokens: 0,
			},
			expectError: true,
		},
		{
			name: "从环境变量加载API Key",
			config: &ZhipuConfig{
				APIKey:    "", // 从环境变量读取
				Model:     "glm-4",
				MaxTokens: 4096,
			},
			expectError: true, // 因为没有设置环境变量
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 对于需要测试没有环境变量情况的测试，临时清除环境变量
			if tt.name == "配置为空" || tt.name == "从环境变量加载API Key" {
				os.Unsetenv("ZHIPU_API_KEY")
				os.Unsetenv("ZHIPU_MODEL")
				// 测试结束后恢复
				defer func() {
					if origAPIKey != "" {
						os.Setenv("ZHIPU_API_KEY", origAPIKey)
					}
					if origModel != "" {
						os.Setenv("ZHIPU_MODEL", origModel)
					}
				}()
			}

			client, err := NewZhipuClient(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, ModelTypeZhipu, client.GetType())
			}
		})
	}
}

// TestZhipuConfig_Validate 测试配置验证
func TestZhipuConfig_Validate(t *testing.T) {
	// 保存原始环境变量
	origAPIKey := os.Getenv("ZHIPU_API_KEY")
	origModel := os.Getenv("ZHIPU_MODEL")
	defer func() {
		if origAPIKey != "" {
			os.Setenv("ZHIPU_API_KEY", origAPIKey)
		} else {
			os.Unsetenv("ZHIPU_API_KEY")
		}
		if origModel != "" {
			os.Setenv("ZHIPU_MODEL", origModel)
		} else {
			os.Unsetenv("ZHIPU_MODEL")
		}
	}()

	temperature := float32(0.7)

	tests := []struct {
		name        string
		config      *ZhipuConfig
		expectError bool
	}{
		{
			name: "完整有效配置",
			config: &ZhipuConfig{
				APIKey:      "test-key",
				Model:       "glm-4",
				MaxTokens:   4096,
				Temperature: &temperature,
				Timeout:     30 * time.Second,
			},
			expectError: false,
		},
		{
			name: "从环境变量加载",
			config: &ZhipuConfig{
				APIKey: "", // 将从环境变量读取
				Model:  "", // 将从环境变量读取
			},
			expectError: true, // 因为没有设置环境变量
		},
		{
			name: "Temperature超出范围",
			config: &ZhipuConfig{
				APIKey:      "test-key",
				Model:       "glm-4",
				Temperature: func() *float32 { v := float32(1.5); return &v }(),
			},
			expectError: true,
		},
		{
			name: "Temperature为负数",
			config: &ZhipuConfig{
				APIKey:      "test-key",
				Model:       "glm-4",
				Temperature: func() *float32 { v := float32(-0.5); return &v }(),
			},
			expectError: true,
		},
		{
			name: "MaxTokens为0（将设置默认值）",
			config: &ZhipuConfig{
				APIKey:    "test-key",
				Model:     "glm-4",
				MaxTokens: 0,
			},
			expectError: false,
		},
		{
			name: "Timeout为0（将设置默认值）",
			config: &ZhipuConfig{
				APIKey:  "test-key",
				Model:   "glm-4",
				Timeout: 0,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 对于需要测试没有环境变量情况的测试，临时清除环境变量
			if tt.name == "从环境变量加载" {
				os.Unsetenv("ZHIPU_API_KEY")
				os.Unsetenv("ZHIPU_MODEL")
				// 测试结束后恢复
				defer func() {
					if origAPIKey != "" {
						os.Setenv("ZHIPU_API_KEY", origAPIKey)
					}
					if origModel != "" {
						os.Setenv("ZHIPU_MODEL", origModel)
					}
				}()
			}

			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// 验证默认值是否设置
				if tt.config.MaxTokens == 0 {
					assert.Equal(t, 4096, tt.config.MaxTokens)
				}
				if tt.config.Timeout == 0 {
					assert.Equal(t, 30*time.Second, tt.config.Timeout)
				}
			}
		})
	}
}

// TestZhipuClient_GetType 测试获取模型类型
func TestZhipuClient_GetType(t *testing.T) {
	client := &ZhipuClient{
		apiKey: "test-key",
		config: &ZhipuConfig{Model: "glm-4"},
	}

	assert.Equal(t, ModelTypeZhipu, client.GetType())
}

// TestZhipuClient_Close 测试关闭客户端
func TestZhipuClient_Close(t *testing.T) {
	client := &ZhipuClient{
		apiKey: "test-key",
		config: &ZhipuConfig{Model: "glm-4"},
	}

	err := client.Close()
	assert.NoError(t, err)
}

// TestZhipuConfig_GetType 测试配置获取类型
func TestZhipuConfig_GetType(t *testing.T) {
	config := &ZhipuConfig{}
	assert.Equal(t, ModelTypeZhipu, config.GetType())
}

// TestNewZhipuClientFromConfig 测试从通用配置创建
func TestNewZhipuClientFromConfig(t *testing.T) {
	timeout := 30

	aiConfig := &AIConfig{
		Provider:  "zhipu",
		APIKey:    "test-api-key",
		Model:     "glm-4",
		MaxTokens: 4096,
		Timeout:   timeout,
	}

	model, err := NewZhipuClientFromConfig(aiConfig)

	assert.NoError(t, err)
	assert.NotNil(t, model)
	assert.Equal(t, ModelTypeZhipu, model.GetType())

	// 类型断言验证
	zhipuClient, ok := model.(*ZhipuClient)
	assert.True(t, ok)
	assert.Equal(t, "test-api-key", zhipuClient.apiKey)
	assert.Equal(t, "glm-4", zhipuClient.config.Model)
}

// TestZhipuClient_AnalyzePattern 测试模式分析（模拟）
func TestZhipuClient_AnalyzePattern(t *testing.T) {
	client := &ZhipuClient{
		apiKey: "test-key",
		config: &ZhipuConfig{
			Model:   "glm-4",
			Timeout: 30 * time.Second,
		},
	}

	patternData := map[string]interface{}{
		"pattern_id":    "test-123",
		"support_count": 10,
		"confidence":    0.8,
	}

	// 注意：由于没有真实的 chatModel，这个测试会失败
	// 实际测试需要真实的API Key和有效的 chatModel
	ctx := context.Background()
	_, err := client.AnalyzePattern(ctx, patternData)

	// 由于没有真实的 chatModel，会返回错误
	assert.Error(t, err)
}

// TestZhipuClient_parseAnalysisResponse 测试响应解析
func TestZhipuClient_parseAnalysisResponse(t *testing.T) {
	client := &ZhipuClient{}

	tests := []struct {
		name        string
		response    string
		expectError bool
	}{
		{
			name: "标准JSON响应",
			response: `{
				"should_automate": true,
				"reason": "这是一个高频操作",
				"estimated_time_saving": 30,
				"complexity": "low",
				"suggested_name": "快捷助手",
				"suggested_steps": ["步骤1", "步骤2"]
			}`,
			expectError: false,
		},
		{
			name:        "带Markdown代码块的响应",
			response:    "```json\n{\"should_automate\": true, \"reason\": \"测试\", \"estimated_time_saving\": 30}\n```",
			expectError: false,
		},
		{
			name:        "无效JSON",
			response:    `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis, err := client.parseAnalysisResponse(tt.response)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, analysis)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, analysis)
				assert.False(t, analysis.AnalyzedAt.IsZero())
			}
		})
	}
}

// TestZhipuClient_RealAPICall 真实API调用测试
//
// 该测试需要设置真实的环境变量才能运行：
// - ZHIPU_API_KEY: 智谱AI API密钥
// - ZHIPU_MODEL: 模型名称（可选，默认为 glm-4）
//
// 运行方式：go test -v -run TestZhipuClient_RealAPICall ./internal/infrastructure/ai
func TestZhipuClient_RealAPICall(t *testing.T) {
	// 检查环境变量
	apiKey := os.Getenv("ZHIPU_API_KEY")
	if apiKey == "" {
		t.Skip("跳过真实API调用测试：未设置 ZHIPU_API_KEY 环境变量")
	}

	model := os.Getenv("ZHIPU_MODEL")
	if model == "" {
		model = "glm-4" // 默认模型
	}

	// 创建客户端
	config := &ZhipuConfig{
		APIKey:    apiKey,
		Model:     model,
		MaxTokens: 4096,
		Timeout:   60 * time.Second,
	}

	client, err := NewZhipuClient(config)
	require.NoError(t, err, "创建智谱AI客户端失败")
	require.NotNil(t, client, "客户端不能为空")

	defer func() {
		err := client.Close()
		assert.NoError(t, err, "关闭客户端失败")
	}()

	// 准备测试数据
	patternData := map[string]interface{}{
		"pattern_id":    "test-real-api-123",
		"support_count": 50,
		"confidence":    0.92,
		"pattern_type":  "sequential",
		"events": []interface{}{
			map[string]interface{}{
				"event_type":  "click",
				"element":     "submit_button",
				"timestamp":   "2024-01-01T10:00:00Z",
				"wait_before": 1000,
			},
			map[string]interface{}{
				"event_type":  "type",
				"element":     "input_field",
				"text":        "test",
				"timestamp":   "2024-01-01T10:00:01Z",
				"wait_before": 500,
			},
		},
		"frequency": map[string]interface{}{
			"daily_count":   20,
			"weekly_count":  140,
			"total_count":   500,
		},
		"time_saving_estimate": map[string]interface{}{
			"per_occurrence": 30,
			"total_daily":    600,
		},
	}

	// 执行真实API调用
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	analysis, err := client.AnalyzePattern(ctx, patternData)

	// 验证结果
	require.NoError(t, err, "调用智谱AI API失败")
	require.NotNil(t, analysis, "分析结果不能为空")

	// 验证响应字段
	assert.NotEmpty(t, analysis.Reason, "分析原因不能为空")
	assert.Contains(t, []string{"low", "medium", "high"}, analysis.Complexity,
		"复杂度必须是 low、medium 或 high")
	assert.GreaterOrEqual(t, analysis.EstimatedTimeSaving, int64(0),
		"预计节省时间必须大于等于0")

	// 如果建议自动化，应该有名称和步骤
	if analysis.ShouldAutomate {
		assert.NotEmpty(t, analysis.SuggestedName, "自动化建议名称不能为空")
		assert.NotEmpty(t, analysis.SuggestedSteps, "自动化步骤不能为空")
	}

	// 验证分析时间
	assert.False(t, analysis.AnalyzedAt.IsZero(), "分析时间不能为零值")

	t.Logf("✅ 真实API调用成功")
	t.Logf("   - 是否建议自动化: %v", analysis.ShouldAutomate)
	t.Logf("   - 原因: %s", analysis.Reason)
	t.Logf("   - 复杂度: %s", analysis.Complexity)
	t.Logf("   - 预计节省时间: %d 秒", analysis.EstimatedTimeSaving)
	if analysis.ShouldAutomate {
		t.Logf("   - 建议名称: %s", analysis.SuggestedName)
		t.Logf("   - 建议步骤: %v", analysis.SuggestedSteps)
	}
}

// TestZhipuClient_RealBatchAPICall 批量真实API调用测试
//
// 该测试需要设置真实的环境变量才能运行
func TestZhipuClient_RealBatchAPICall(t *testing.T) {
	// 检查环境变量
	apiKey := os.Getenv("ZHIPU_API_KEY")
	if apiKey == "" {
		t.Skip("跳过批量API调用测试：未设置 ZHIPU_API_KEY 环境变量")
	}

	model := os.Getenv("ZHIPU_MODEL")
	if model == "" {
		model = "glm-4"
	}

	// 创建客户端
	config := &ZhipuConfig{
		APIKey:    apiKey,
		Model:     model,
		MaxTokens: 4096,
		Timeout:   60 * time.Second,
	}

	client, err := NewZhipuClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)

	defer func() {
		err := client.Close()
		assert.NoError(t, err)
	}()

	// 准备多个测试模式
	patterns := []map[string]interface{}{
		{
			"pattern_id":    "batch-test-1",
			"support_count": 10,
			"confidence":    0.85,
			"pattern_type":  "sequential",
		},
		{
			"pattern_id":    "batch-test-2",
			"support_count": 25,
			"confidence":    0.90,
			"pattern_type":  "parallel",
		},
	}

	// 执行批量API调用
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	results, err := client.AnalyzePatternBatch(ctx, patterns)

	// 验证结果
	require.NoError(t, err, "批量调用智谱AI API失败")
	require.Len(t, results, len(patterns), "结果数量必须与请求数量一致")

	// 验证每个结果
	for i, result := range results {
		assert.NotNil(t, result, fmt.Sprintf("结果[%d]不能为空", i))
		if result != nil {
			assert.NotEmpty(t, result.Reason, fmt.Sprintf("结果[%d]原因不能为空", i))
			assert.False(t, result.AnalyzedAt.IsZero(), fmt.Sprintf("结果[%d]分析时间不能为零值", i))
			t.Logf("   模式[%d]: should_automate=%v, complexity=%s",
				i, result.ShouldAutomate, result.Complexity)
		}
	}

	t.Logf("✅ 批量API调用成功，共处理 %d 个模式", len(results))
}
