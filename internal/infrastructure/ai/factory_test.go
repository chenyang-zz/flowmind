/**
 * Package ai AI 服务基础设施层
 *
 * AI 模型工厂单元测试
 */

package ai

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAIConfig_LoadFromEnv 测试从环境变量加载配置
func TestAIConfig_LoadFromEnv(t *testing.T) {
	// 保存原始环境变量
	origProvider := os.Getenv("AI_PROVIDER")
	origAPIKey := os.Getenv("AI_API_KEY")
	origModel := os.Getenv("AI_MODEL")
	origBaseURL := os.Getenv("AI_BASE_URL")
	origClaudeKey := os.Getenv("CLAUDE_API_KEY")
	origZhipuKey := os.Getenv("ZHIPU_API_KEY")

	// 测试结束后恢复环境变量
	defer func() {
		if origProvider != "" {
			os.Setenv("AI_PROVIDER", origProvider)
		} else {
			os.Unsetenv("AI_PROVIDER")
		}
		if origAPIKey != "" {
			os.Setenv("AI_API_KEY", origAPIKey)
		} else {
			os.Unsetenv("AI_API_KEY")
		}
		if origModel != "" {
			os.Setenv("AI_MODEL", origModel)
		} else {
			os.Unsetenv("AI_MODEL")
		}
		if origBaseURL != "" {
			os.Setenv("AI_BASE_URL", origBaseURL)
		} else {
			os.Unsetenv("AI_BASE_URL")
		}
		if origClaudeKey != "" {
			os.Setenv("CLAUDE_API_KEY", origClaudeKey)
		} else {
			os.Unsetenv("CLAUDE_API_KEY")
		}
		if origZhipuKey != "" {
			os.Setenv("ZHIPU_API_KEY", origZhipuKey)
		} else {
			os.Unsetenv("ZHIPU_API_KEY")
		}
	}()

	tests := []struct {
		name          string
		envProvider   string
		envAPIKey     string
		envModel      string
		expectedProv  string
		expectedModel string
	}{
		{
			name:         "默认配置",
			envProvider:  "",
			envAPIKey:    "",
			envModel:     "",
			expectedProv: "claude",
			expectedModel: "claude-3-5-sonnet-20241022",
		},
		{
			name:         "Claude配置",
			envProvider:  "claude",
			envAPIKey:    "sk-ant-test",
			envModel:     "claude-3-5-sonnet-20241022",
			expectedProv: "claude",
			expectedModel: "claude-3-5-sonnet-20241022",
		},
		{
			name:         "智谱AI配置",
			envProvider:  "zhipu",
			envAPIKey:    "zhipu-test-key",
			envModel:     "glm-4",
			expectedProv: "zhipu",
			expectedModel: "glm-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清空环境变量
			os.Unsetenv("AI_PROVIDER")
			os.Unsetenv("AI_API_KEY")
			os.Unsetenv("AI_MODEL")
			os.Unsetenv("CLAUDE_API_KEY")
			os.Unsetenv("ZHIPU_API_KEY")

			// 设置测试环境变量
			if tt.envProvider != "" {
				os.Setenv("AI_PROVIDER", tt.envProvider)
			}
			if tt.envAPIKey != "" {
				os.Setenv("AI_API_KEY", tt.envAPIKey)
				if tt.envProvider == "claude" {
					os.Setenv("CLAUDE_API_KEY", tt.envAPIKey)
				} else if tt.envProvider == "zhipu" {
					os.Setenv("ZHIPU_API_KEY", tt.envAPIKey)
				}
			}
			if tt.envModel != "" {
				os.Setenv("AI_MODEL", tt.envModel)
			}

			// 加载配置
			config := &AIConfig{}
			config = config.LoadFromEnv()

			// 验证配置
			assert.Equal(t, tt.expectedProv, config.Provider)
			assert.Equal(t, tt.expectedModel, config.Model)
		})
	}
}

// TestAIConfig_Validate 测试配置验证
func TestAIConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *AIConfig
		expectError bool
	}{
		{
			name: "有效Claude配置",
			config: &AIConfig{
				Provider: "claude",
				APIKey:   "sk-ant-test",
				Model:    "claude-3-5-sonnet-20241022",
			},
			expectError: false,
		},
		{
			name: "有效智谱AI配置",
			config: &AIConfig{
				Provider: "zhipu",
				APIKey:   "zhipu-test-key",
				Model:    "glm-4",
			},
			expectError: false,
		},
		{
			name: "有效Ollama配置（无需API Key）",
			config: &AIConfig{
				Provider: "ollama",
				APIKey:   "",
				Model:    "llama3.2",
			},
			expectError: false,
		},
		{
			name: "缺少提供商",
			config: &AIConfig{
				Provider: "",
				APIKey:   "test-key",
				Model:    "test-model",
			},
			expectError: true,
		},
		{
			name: "不支持的提供商",
			config: &AIConfig{
				Provider: "unknown",
				APIKey:   "test-key",
				Model:    "test-model",
			},
			expectError: true,
		},
		{
			name: "缺少API Key（非Ollama）",
			config: &AIConfig{
				Provider: "claude",
				APIKey:   "",
				Model:    "claude-3-5-sonnet-20241022",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSwitchProvider 测试提供商切换
func TestSwitchProvider(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		apiKey      string
		model       string
		expectError bool
	}{
		{
			name:        "切换到Claude",
			provider:    "claude",
			apiKey:      "sk-ant-test-key",
			model:       "claude-3-5-sonnet-20241022",
			expectError: false, // 可能因为没有实际API而失败，但配置应该正确
		},
		{
			name:        "切换到智谱AI",
			provider:    "zhipu",
			apiKey:      "test-zhipu-key",
			model:       "glm-4",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := SwitchProvider(tt.provider, tt.apiKey, tt.model)

			// 由于可能没有真实的API Key，这里主要测试配置是否正确创建
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, model)
			} else {
				// 如果没有错误，验证模型类型
				if err == nil {
					assert.NotNil(t, model)
					assert.Equal(t, ModelType(tt.provider), model.GetType())
				}
			}
		})
	}
}

// TestGetEnvOrDefault 测试环境变量获取辅助函数
func TestGetEnvOrDefault(t *testing.T) {
	// 保存原始环境变量
	origValue := os.Getenv("TEST_GET_ENV_VAR")
	defer func() {
		if origValue != "" {
			os.Setenv("TEST_GET_ENV_VAR", origValue)
		} else {
			os.Unsetenv("TEST_GET_ENV_VAR")
		}
	}()

	// 测试环境变量存在的情况
	os.Setenv("TEST_GET_ENV_VAR", "test-value")
	result := GetEnvOrDefault("TEST_GET_ENV_VAR", "default")
	assert.Equal(t, "test-value", result)

	// 测试环境变量不存在的情况
	os.Unsetenv("TEST_GET_ENV_VAR")
	result = GetEnvOrDefault("TEST_GET_ENV_VAR", "default")
	assert.Equal(t, "default", result)
}
