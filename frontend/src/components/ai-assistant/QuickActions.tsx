/**
 * 快速操作按钮组件
 * 极简设计
 */

import { useAIAssistantStore } from '../../stores/aiAssistantStore';

/**
 * 快速操作组件
 */
export function QuickActions() {
  const { quickActions } = useAIAssistantStore();

  return (
    <div className="space-y-2.5">
      <div className="text-xs font-medium text-white/50 px-1">快速操作</div>

      <div className="grid grid-cols-2 gap-2">
        {quickActions.map((action) => (
          <button
            key={action.id}
            onClick={action.action}
            className="
              bg-white/[0.03]
              border border-white/[0.06]
              px-3 py-2.5
              rounded-lg
              text-left
              hover:bg-white/[0.05]
              hover:border-white/[0.09]
              transition-all duration-150
            "
          >
            <div className="flex items-center gap-2">
              <span className="text-sm">{action.icon}</span>
              <span className="text-xs font-medium text-white/80">
                {action.label}
              </span>
            </div>
          </button>
        ))}
      </div>

      {/* 智能建议 - 更简洁 */}
      <div
        className="
          bg-white/[0.03]
          border border-white/[0.06]
          px-4 py-3
          rounded-lg
          mt-3
        "
      >
        <div className="text-xs text-white/40 leading-relaxed mb-2.5">
          检测到重复模式：每次写代码后手动搜索文档
        </div>
        <div className="flex items-center gap-2">
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
          <button
            className="
              px-3 py-1.5
              text-xs font-medium
              rounded-md
              text-white/40
              hover:text-white/60
              transition-colors duration-150
            "
          >
            忽略
          </button>
        </div>
      </div>
    </div>
  );
}
