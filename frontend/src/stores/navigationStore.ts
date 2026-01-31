/**
 * 导航状态管理
 *
 * 管理应用当前显示的页面（主页面、设置页面等）
 */

import { create } from 'zustand';

/**
 * 页面类型
 */
export type Page = 'main' | 'settings' | 'dashboard' | 'automations' | 'knowledge';

/**
 * 导航状态接口
 */
interface NavigationState {
  /**
   * 当前页面
   */
  currentPage: Page;

  /**
   * 设置当前页面
   */
  setPage: (page: Page) => void;

  /**
   * 返回主页面
   */
  goToMain: () => void;

  /**
   * 打开设置页面
   */
  goToSettings: () => void;

  /**
   * 打开仪表板
   */
  goToDashboard: () => void;

  /**
   * 打开自动化管理
   */
  goToAutomations: () => void;

  /**
   * 打开知识图谱
   */
  goToKnowledge: () => void;
}

/**
 * 导航 store
 */
export const useNavigationStore = create<NavigationState>((set) => ({
  currentPage: 'main',

  setPage: (page) => set({ currentPage: page }),

  goToMain: () => set({ currentPage: 'main' }),

  goToSettings: () => set({ currentPage: 'settings' }),

  goToDashboard: () => set({ currentPage: 'dashboard' }),

  goToAutomations: () => set({ currentPage: 'automations' }),

  goToKnowledge: () => set({ currentPage: 'knowledge' }),
}));
