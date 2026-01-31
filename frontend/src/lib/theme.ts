/**
 * 主题管理工具
 *
 * 提供主题切换和持久化功能
 */

export type Theme = 'light' | 'dark';

/**
 * 主题存储键名
 */
const THEME_STORAGE_KEY = 'flowmind-theme';

/**
 * 获取系统偏好主题
 */
export function getSystemTheme(): Theme {
  if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    return 'dark';
  }
  return 'light';
}

/**
 * 获取当前主题
 */
export function getTheme(): Theme {
  const stored = localStorage.getItem(THEME_STORAGE_KEY);
  if (stored === 'light' || stored === 'dark') {
    return stored;
  }
  return getSystemTheme();
}

/**
 * 设置主题
 */
export function setTheme(theme: Theme): void {
  localStorage.setItem(THEME_STORAGE_KEY, theme);
  applyTheme(theme);
}

/**
 * 切换主题
 */
export function toggleTheme(): void {
  const current = getTheme();
  const next = current === 'dark' ? 'light' : 'dark';
  setTheme(next);
}

/**
 * 应用主题到 DOM
 */
export function applyTheme(theme: Theme): void {
  const root = document.documentElement;
  root.classList.remove('light', 'dark');
  root.classList.add(theme);
}

/**
 * 初始化主题
 */
export function initTheme(): void {
  const theme = getTheme();
  applyTheme(theme);

  // 监听系统主题变化
  if (window.matchMedia) {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    mediaQuery.addEventListener('change', (e) => {
      // 只在没有手动设置主题时跟随系统
      if (!localStorage.getItem(THEME_STORAGE_KEY)) {
        applyTheme(e.matches ? 'dark' : 'light');
      }
    });
  }
}
