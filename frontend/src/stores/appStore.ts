/**
 * 应用全局状态管理 Store
 *
 * 使用 Zustand 进行全局状态管理
 * 负责管理应用级别的状态，如加载状态、错误信息等
 */

import { create } from 'zustand';

/**
 * AppStore 接口定义
 *
 * 定义了应用全局状态的结构
 */
interface AppStore {
  // ========== 状态 ==========

  /** 是否正在加载 */
  isLoading: boolean;

  /** 错误信息 */
  error: string | null;

  // ========== Actions ==========

  /**
   * 设置加载状态
   *
   * @param loading - 是否正在加载
   */
  setLoading: (loading: boolean) => void;

  /**
   * 设置错误信息
   *
   * @param error - 错误信息，null 表示清除错误
   */
  setError: (error: string | null) => void;
}

/**
 * 创建 App Store
 *
 * 使用 Zustand 的 create API 创建状态管理 store
 *
 * @example
 * ```tsx
 * const { isLoading, error, setLoading } = useAppStore();
 * ```
 */
export const useAppStore = create<AppStore>((set) => ({
  // ========== 初始状态 ==========

  /** 初始加载状态为 false */
  isLoading: false,

  /** 初始无错误 */
  error: null,

  // ========== Actions 实现 ==========

  /**
   * 设置加载状态
   *
   * @param loading - 是否正在加载
   */
  setLoading: (loading: boolean) =>
    set({ isLoading: loading }),

  /**
   * 设置错误信息
   *
   * @param error - 错误信息，null 表示清除错误
   */
  setError: (error: string | null) =>
    set({ error }),
}));
