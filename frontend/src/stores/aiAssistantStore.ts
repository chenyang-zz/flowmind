/**
 * AI 助手状态管理
 *
 * 使用 Zustand 管理 AI 助手面板的状态
 */

import { create } from 'zustand';
import { AppContext } from '../lib/wails';

/**
 * AI 洞察数据结构
 */
export interface AIInsight {
  id: string;
  type: 'pattern' | 'optimization' | 'suggestion';
  title: string;
  description: string;
  actionable: boolean;
  timestamp: string;
}

/**
 * 快速操作按钮
 */
export interface QuickAction {
  id: string;
  label: string;
  icon: string;
  action: () => void;
}

/**
 * AI 助手 Store 状态
 */
interface AIAssistantState {
  /** 面板是否可见 */
  isOpen: boolean;

  /** 当前应用上下文 */
  currentContext: AppContext | null;

  /** AI 洞察列表 */
  insights: AIInsight[];

  /** 快速操作列表 */
  quickActions: QuickAction[];

  /** 用户输入的文本 */
  inputText: string;

  /** 是否正在加载 */
  isLoading: boolean;

  /** 打开面板 */
  openPanel: () => void;

  /** 关闭面板 */
  closePanel: () => void;

  /** 切换面板状态 */
  togglePanel: () => void;

  /** 设置当前应用上下文 */
  setCurrentContext: (context: AppContext | null) => void;

  /** 添加洞察 */
  addInsight: (insight: AIInsight) => void;

  /** 清空洞察 */
  clearInsights: () => void;

  /** 设置快速操作 */
  setQuickActions: (actions: QuickAction[]) => void;

  /** 设置输入文本 */
  setInputText: (text: string) => void;

  /** 发送消息 */
  sendMessage: () => Promise<void>;

  /** 设置加载状态 */
  setLoading: (loading: boolean) => void;
}

/**
 * AI 助手 Store
 */
export const useAIAssistantStore = create<AIAssistantState>((set, get) => ({
  // 初始状态
  isOpen: false,
  currentContext: null,
  insights: [],
  quickActions: [
    {
      id: 'search-code',
      label: '搜索相关代码',
      icon: '🔍',
      action: () => console.log('搜索相关代码'),
    },
    {
      id: 'generate-snippet',
      label: '生成代码片段',
      icon: '📋',
      action: () => console.log('生成代码片段'),
    },
    {
      id: 'explain-code',
      label: '解释代码',
      icon: '🤖',
      action: () => console.log('解释代码'),
    },
  ],
  inputText: '',
  isLoading: false,

  // 操作方法
  openPanel: () => set({ isOpen: true }),

  closePanel: () => set({ isOpen: false }),

  togglePanel: () => set((state) => ({ isOpen: !state.isOpen })),

  setCurrentContext: (context) => set({ currentContext: context }),

  addInsight: (insight) =>
    set((state) => ({
      insights: [...state.insights, insight],
    })),

  clearInsights: () => set({ insights: [] }),

  setQuickActions: (actions) => set({ quickActions: actions }),

  setInputText: (text) => set({ inputText: text }),

  sendMessage: async () => {
    const { inputText, currentContext } = get();
    if (!inputText.trim()) return;

    set({ isLoading: true });

    try {
      // TODO: 调用后端 API 发送消息到 AI
      console.log('发送消息:', inputText, '上下文:', currentContext);

      // 模拟 AI 响应
      await new Promise((resolve) => setTimeout(resolve, 1000));

      // 清空输入
      set({ inputText: '', isLoading: false });
    } catch (error) {
      console.error('发送消息失败:', error);
      set({ isLoading: false });
    }
  },

  setLoading: (loading) => set({ isLoading: loading }),
}));
