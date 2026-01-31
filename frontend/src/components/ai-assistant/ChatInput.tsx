/**
 * 聊天输入框组件
 * 极简设计
 */

import { useState } from 'react';
import { ArrowUp } from 'lucide-react';
import { useAIAssistantStore } from '../../stores/aiAssistantStore';

/**
 * 聊天输入框组件
 */
export function ChatInput() {
  const { inputText, setInputText, sendMessage, isLoading } = useAIAssistantStore();
  const [isFocused, setIsFocused] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await sendMessage();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    // Enter 发送，Shift+Enter 换行
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  return (
    <form onSubmit={handleSubmit} className="relative">
      <div
        className={`
          relative
          bg-white/[0.04]
          border
          rounded-lg
          transition-all duration-150
          ${
            isFocused
              ? 'border-indigo-500/50 bg-white/[0.05]'
              : 'border-white/[0.08] hover:border-white/[0.12]'
          }
        `}
      >
        <textarea
          value={inputText}
          onChange={(e) => setInputText(e.target.value)}
          onFocus={() => setIsFocused(true)}
          onBlur={() => setIsFocused(false)}
          onKeyDown={handleKeyDown}
          placeholder="输入消息..."
          disabled={isLoading}
          rows={1}
          className="
            flex-1
            bg-transparent
            px-3.5 py-2.5
            pr-10
            text-sm
            text-white/90
            placeholder:text-white/25
            outline-none
            resize-none
            disabled:opacity-50
            w-full
          "
          style={{
            minHeight: '38px',
            maxHeight: '120px',
          }}
        />

        <button
          type="submit"
          disabled={!inputText.trim() || isLoading}
          className="
            absolute
            right-2
            bottom-2
            flex items-center justify-center
            w-6 h-6
            rounded-md
            bg-indigo-500
            text-white
            disabled:opacity-30
            disabled:bg-white/10
            hover:bg-indigo-400
            active:scale-95
            transition-all duration-150
          "
        >
          {isLoading ? (
            <div className="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin" />
          ) : (
            <ArrowUp size={12} />
          )}
        </button>
      </div>
    </form>
  );
}
