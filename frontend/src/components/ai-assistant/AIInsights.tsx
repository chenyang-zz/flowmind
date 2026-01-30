/**
 * AI 洞察卡片组件
 * 极简设计
 */

import { useAIAssistantStore } from '../../stores/aiAssistantStore';

/**
 * AI 洞察组件
 */
export function AIInsights() {
  const { insights } = useAIAssistantStore();

  if (insights.length === 0) {
    return null;
  }

  return (
    <div className="space-y-2.5">
      <div className="text-xs font-medium text-white/50 px-1">洞察</div>

      {insights.map((insight) => (
        <div
          key={insight.id}
          className="
            bg-white/[0.03]
            border border-white/[0.06]
            px-4 py-3
            rounded-lg
            hover:bg-white/[0.04]
            transition-colors duration-150
          "
        >
          <div className="text-sm font-medium text-white/90 mb-1">
            {insight.title}
          </div>
          <div className="text-xs text-white/40 leading-relaxed mb-2.5">
            {insight.description}
          </div>

          {insight.actionable && (
            <button
              className="
                px-3 py-1.5
                text-xs font-medium
                rounded-md
                bg-indigo-500
                text-white
                hover:bg-indigo-400
                transition-colors duration-150
              "
            >
              创建自动化
            </button>
          )}
        </div>
      ))}
    </div>
  );
}
