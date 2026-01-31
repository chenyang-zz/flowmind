/**
 * AI 配置组件
 */

import { useState } from "react";
import { useSettingsStore } from "../../stores/settingsStore";
import { SettingItem } from "../common/SettingItem";
import { SettingGroup } from "../common/SettingGroup";

export function AISettings() {
  const {
    aiModel,
    setAIModel,
    apiKey,
    setApiKey,
    smartSuggestions,
    setSmartSuggestions,
    learningMode,
    setLearningMode,
  } = useSettingsStore();

  const [] = useState(false);

  return (
    <SettingGroup title="AI 配置" description="配置 AI 模型和智能功能">
      {/* AI 模型选择 */}
      <SettingItem type="custom" title="AI 模型">
        <div className="flex gap-3">
          <button
            onClick={() => setAIModel("claude")}
            className={`
              flex-1
              px-4 py-4
              rounded-lg
              border
              transition-all duration-150
              ${
                aiModel === "claude"
                  ? "bg-indigo-500/10 border-indigo-500/50"
                  : "bg-white/3 border-white/6 hover:border-white/12"
              }
            `}
          >
            <div className="text-left">
              <div
                className={`text-sm font-medium mb-1 ${
                  aiModel === "claude" ? "text-white/90" : "text-white/60"
                }`}
              >
                Claude
              </div>
              <div className="text-xs text-white/35">Anthropic AI 助手</div>
            </div>
          </button>

          <button
            onClick={() => setAIModel("ollama")}
            className={`
              flex-1
              px-4 py-4
              rounded-lg
              border
              transition-all duration-150
              ${
                aiModel === "ollama"
                  ? "bg-indigo-500/10 border-indigo-500/50"
                  : "bg-white/3 border-white/6 hover:border-white/12"
              }
            `}
          >
            <div className="text-left">
              <div
                className={`text-sm font-medium mb-1 ${
                  aiModel === "ollama" ? "text-white/90" : "text-white/60"
                }`}
              >
                Ollama
              </div>
              <div className="text-xs text-white/35">本地运行模型</div>
            </div>
          </button>
        </div>
      </SettingItem>

      {/* API Key */}
      {aiModel === "claude" && (
        <SettingItem
          type="input"
          value={apiKey}
          title="API Key"
          placeholder="sk-ant-..."
          onChange={(text) => setApiKey(text)}
          inputType="password"
          description="你的 API Key 只会存储在本地，不会发送到我们的服务器"
          action={
            <button
              onClick={() => {
                // TODO: 打开 Claude API 网站获取 key
                window.open("https://console.anthropic.com/", "_blank");
              }}
              className="
                  px-2.5 py-1.5
                  text-xs font-medium
                  rounded-md
                  bg-white/5
                  text-white/50
                  hover:bg-white/10
                  hover:text-white/70
                  transition-colors
                "
            >
              获取
            </button>
          }
        />
      )}

      {/* 智能建议 */}
      <SettingItem
        type="switch"
        title="智能建议"
        description="AI 主动提供工作流优化建议"
        enabled={smartSuggestions}
        onToggle={() => setSmartSuggestions(!smartSuggestions)}
      />

      {/* 学习模式 */}
      <SettingItem
        type="custom"
        title="学习模式"
        description={
          learningMode === "active"
            ? "AI 会主动分析你的工作模式并提供建议"
            : "AI 只在你主动询问时提供帮助"
        }
      >
        <div className="flex gap-3">
          <button
            onClick={() => setLearningMode("active")}
            className={`
              flex-1
              px-4 py-3
              rounded-lg
              border
              text-sm font-medium
              transition-all duration-150
              ${
                learningMode === "active"
                  ? "bg-indigo-500/10 border-indigo-500/50 text-white/90"
                  : "bg-white/3 border-white/6 text-white/40 hover:text-white/60"
              }
            `}
          >
            主动学习
          </button>
          <button
            onClick={() => setLearningMode("passive")}
            className={`
              flex-1
              px-4 py-3
              rounded-lg
              border
              text-sm font-medium
              transition-all duration-150
              ${
                learningMode === "passive"
                  ? "bg-indigo-500/10 border-indigo-500/50 text-white/90"
                  : "bg-white/3 border-white/6 text-white/40 hover:text-white/60"
              }
            `}
          >
            被动响应
          </button>
        </div>
      </SettingItem>
    </SettingGroup>
  );
}
