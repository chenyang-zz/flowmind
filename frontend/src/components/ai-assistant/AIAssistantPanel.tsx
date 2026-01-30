/**
 * AI 助手面板组件
 *
 * 快捷键 ⌘⇧M 唤起的浮层面板
 * 极简设计，无标题栏
 */

import { useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { X } from "lucide-react";
import { useAIAssistantStore } from "../../stores/aiAssistantStore";
import { registerShortcut } from "../../lib/shortcuts";
import { SHORTCUTS } from "../../lib/shortcuts";
import { ContextDisplay } from "./ContextDisplay";
import { AIInsights } from "./AIInsights";
import { QuickActions } from "./QuickActions";
import { ChatInput } from "./ChatInput";

/**
 * AI 助手面板组件
 */
export function AIAssistantPanel() {
  const { isOpen, closePanel } = useAIAssistantStore();

  // 注册快捷键
  useEffect(() => {
    const cleanup = registerShortcut(SHORTCUTS.TOGGLE_AI_ASSISTANT, () => {
      if (isOpen) {
        closePanel();
      } else {
        useAIAssistantStore.getState().openPanel();
      }
    });

    return cleanup;
  }, [isOpen, closePanel]);

  // ESC 键关闭面板
  useEffect(() => {
    if (!isOpen) return;

    const handleEsc = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        closePanel();
      }
    };

    window.addEventListener("keydown", handleEsc);
    return () => window.removeEventListener("keydown", handleEsc);
  }, [isOpen, closePanel]);

  return (
    <AnimatePresence>
      {isOpen && (
        <>
          {/* 背景遮罩 */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.15 }}
            className="
              fixed inset-0
              bg-black/40
              backdrop-blur-sm
              z-40
            "
            onClick={closePanel}
          />

          {/* 浮层面板 - 无标题栏设计 */}
          <motion.div
            initial={{ opacity: 0, scale: 0.96, y: 10 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.96, y: 10 }}
            transition={{
              duration: 0.18,
              ease: [0.16, 1, 0.3, 1],
            }}
            className="
              fixed top-1/2 left-1/2
              -translate-x-1/2 -translate-y-1/2
              w-135 max-w-[90vw]
              max-h-[75vh]
              glass-panel
              z-50
              rounded-lg
              shadow-2xl
              overflow-hidden
              flex flex-col
            "
            onClick={(e: React.MouseEvent) => e.stopPropagation()}
          >
            {/* 顶部工具栏 - 只有关闭按钮 */}
            <div className="flex justify-end px-4 py-3 border-b border-white/5">
              <button
                onClick={closePanel}
                className="
                  flex items-center justify-center
                  w-7 h-7
                  rounded-md
                  text-white/35
                  hover:text-white/60
                  hover:bg-white/5
                  transition-all duration-120
                "
              >
                <X size={16} />
              </button>
            </div>

            {/* 内容区域 */}
            <div className="flex-1 overflow-y-auto px-5 py-4 space-y-4">
              {/* 当前应用上下文 */}
              <ContextDisplay />

              {/* AI 洞察 */}
              <AIInsights />

              {/* 快速操作 */}
              <QuickActions />
            </div>

            {/* 底部输入框 */}
            <div className="border-t border-white/5 px-4 py-3">
              <ChatInput />
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}
