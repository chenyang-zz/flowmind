/**
 * 设置状态管理
 *
 * 使用 Zustand 管理应用设置
 */

import { create } from 'zustand';
import { Theme } from '../lib/theme';

/**
 * AI 模型类型
 */
export type AIModel = 'claude' | 'ollama';

/**
 * 设置 Store 状态
 */
interface SettingsState {
  /** 主题 */
  theme: Theme;

  /** 语言 */
  language: string;

  /** 是否自动启动 */
  autoStart: boolean;

  /** 是否启用通知 */
  notificationsEnabled: boolean;

  /** AI 模型 */
  aiModel: AIModel;

  /** API Key */
  apiKey: string;

  /** 智能建议开关 */
  smartSuggestions: boolean;

  /** 学习模式 */
  learningMode: 'active' | 'passive';

  /** 监控设置 */
  monitoring: {
    keyboard: boolean;
    clipboard: boolean;
    appSwitch: boolean;
    fileSystem: boolean;
  };

  /** 隐私设置 */
  privacy: {
    dataStorage: 'local' | 'cloud';
    filterSensitive: boolean;
    retentionDays: number;
  };

  /** 设置主题 */
  setTheme: (theme: Theme) => void;

  /** 设置语言 */
  setLanguage: (language: string) => void;

  /** 设置自动启动 */
  setAutoStart: (enabled: boolean) => void;

  /** 设置通知开关 */
  setNotificationsEnabled: (enabled: boolean) => void;

  /** 设置 AI 模型 */
  setAIModel: (model: AIModel) => void;

  /** 设置 API Key */
  setApiKey: (key: string) => void;

  /** 设置智能建议 */
  setSmartSuggestions: (enabled: boolean) => void;

  /** 设置学习模式 */
  setLearningMode: (mode: 'active' | 'passive') => void;

  /** 设置监控项 */
  setMonitoring: (key: keyof SettingsState['monitoring'], value: boolean) => void;

  /** 设置隐私项 */
  setPrivacy: <K extends keyof SettingsState['privacy']>(
    key: K,
    value: SettingsState['privacy'][K]
  ) => void;

  /** 重置所有设置 */
  resetSettings: () => void;

  /** 从 localStorage 加载设置 */
  loadSettings: () => void;

  /** 保存设置到 localStorage */
  saveSettings: () => void;
}

/**
 * 默认设置
 */
const defaultSettings: Omit<
  SettingsState,
  | 'setTheme'
  | 'setLanguage'
  | 'setAutoStart'
  | 'setNotificationsEnabled'
  | 'setAIModel'
  | 'setApiKey'
  | 'setSmartSuggestions'
  | 'setLearningMode'
  | 'setMonitoring'
  | 'setPrivacy'
  | 'resetSettings'
  | 'loadSettings'
  | 'saveSettings'
> = {
  theme: 'dark',
  language: 'zh-CN',
  autoStart: true,
  notificationsEnabled: true,
  aiModel: 'claude',
  apiKey: '',
  smartSuggestions: true,
  learningMode: 'active',
  monitoring: {
    keyboard: true,
    clipboard: true,
    appSwitch: true,
    fileSystem: false,
  },
  privacy: {
    dataStorage: 'local',
    filterSensitive: true,
    retentionDays: 30,
  },
};

/**
 * 设置 Store
 */
export const useSettingsStore = create<SettingsState>((set, get) => ({
  ...defaultSettings,

  // 设置主题
  setTheme: (theme) => {
    set({ theme });
    get().saveSettings();
  },

  // 设置语言
  setLanguage: (language) => {
    set({ language });
    get().saveSettings();
  },

  // 设置自动启动
  setAutoStart: (autoStart) => {
    set({ autoStart });
    get().saveSettings();
  },

  // 设置通知开关
  setNotificationsEnabled: (notificationsEnabled) => {
    set({ notificationsEnabled });
    get().saveSettings();
  },

  // 设置 AI 模型
  setAIModel: (aiModel) => {
    set({ aiModel });
    get().saveSettings();
  },

  // 设置 API Key
  setApiKey: (apiKey) => {
    set({ apiKey });
    get().saveSettings();
  },

  // 设置智能建议
  setSmartSuggestions: (smartSuggestions) => {
    set({ smartSuggestions });
    get().saveSettings();
  },

  // 设置学习模式
  setLearningMode: (learningMode) => {
    set({ learningMode });
    get().saveSettings();
  },

  // 设置监控项
  setMonitoring: (key, value) => {
    set((state) => ({
      monitoring: {
        ...state.monitoring,
        [key]: value,
      },
    }));
    get().saveSettings();
  },

  // 设置隐私项
  setPrivacy: (key, value) => {
    set((state) => ({
      privacy: {
        ...state.privacy,
        [key]: value,
      },
    }));
    get().saveSettings();
  },

  // 重置所有设置
  resetSettings: () => {
    set(defaultSettings);
    get().saveSettings();
  },

  // 从 localStorage 加载设置
  loadSettings: () => {
    try {
      const stored = localStorage.getItem('flowmind-settings');
      if (stored) {
        const parsed = JSON.parse(stored);
        set(parsed);
      }
    } catch (error) {
      console.error('加载设置失败:', error);
    }
  },

  // 保存设置到 localStorage
  saveSettings: () => {
    try {
      const state = get();
      const settings = {
        theme: state.theme,
        language: state.language,
        autoStart: state.autoStart,
        notificationsEnabled: state.notificationsEnabled,
        aiModel: state.aiModel,
        apiKey: state.apiKey,
        smartSuggestions: state.smartSuggestions,
        learningMode: state.learningMode,
        monitoring: state.monitoring,
        privacy: state.privacy,
      };
      localStorage.setItem('flowmind-settings', JSON.stringify(settings));
    } catch (error) {
      console.error('保存设置失败:', error);
    }
  },
}));
