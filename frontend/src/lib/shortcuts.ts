/**
 * 快捷键管理工具
 *
 * 提供全局快捷键注册和管理功能
 */

type ShortcutHandler = (event: KeyboardEvent) => void;

/**
 * 快捷键绑定存储
 */
const shortcuts = new Map<string, ShortcutHandler>();

/**
 * 解析快捷键字符串
 *
 * 示例: "Cmd+Shift+M" => { cmd: true, shift: true, key: "m" }
 */
function parseShortcut(shortcut: string) {
  const parts = shortcut.toLowerCase().split('+');
  const keys = {
    cmd: false,
    shift: false,
    ctrl: false,
    option: false,
    key: '',
  };

  parts.forEach((part) => {
    switch (part) {
      case 'cmd':
      case 'command':
      case '⌘':
        keys.cmd = true;
        break;
      case 'shift':
      case '⇧':
        keys.shift = true;
        break;
      case 'ctrl':
      case 'control':
      case '⌃':
        keys.ctrl = true;
        break;
      case 'option':
      case 'alt':
      case '⌥':
        keys.option = true;
        break;
      default:
        keys.key = part;
    }
  });

  return keys;
}

/**
 * 检查快捷键是否匹配
 */
function isShortcutMatch(event: KeyboardEvent, shortcut: string): boolean {
  const keys = parseShortcut(shortcut);

  // macOS 使用 metaKey，其他系统使用 ctrlKey
  const cmdPressed = event.metaKey || event.ctrlKey;

  return (
    keys.cmd === cmdPressed &&
    keys.shift === event.shiftKey &&
    keys.ctrl === (event.ctrlKey && !event.metaKey) &&
    keys.option === event.altKey &&
    event.key.toLowerCase() === keys.key
  );
}

/**
 * 全局快捷键处理器
 */
function handleKeyDown(event: KeyboardEvent) {
  // 检查是否在输入框中
  const target = event.target as HTMLElement;
  const isInput =
    target.tagName === 'INPUT' ||
    target.tagName === 'TEXTAREA' ||
    target.contentEditable === 'true';

  // 如果在输入框中，跳过快捷键处理（除非是 Escape）
  if (isInput && event.key !== 'Escape') {
    return;
  }

  // 遍历所有注册的快捷键
  shortcuts.forEach((handler, shortcut) => {
    if (isShortcutMatch(event, shortcut)) {
      event.preventDefault();
      handler(event);
    }
  });
}

/**
 * 注册快捷键
 *
 * @param shortcut - 快捷键字符串，例如 "Cmd+Shift+M"
 * @param handler - 快捷键处理函数
 * @returns 清理函数
 */
export function registerShortcut(shortcut: string, handler: ShortcutHandler): () => void {
  // 标准化快捷键字符串
  const normalized = shortcut
    .replace(/Command/g, 'Cmd')
    .replace(/⌘/g, 'Cmd')
    .replace(/⇧/g, 'Shift')
    .replace(/⌥/g, 'Option')
    .replace(/⌃/g, 'Ctrl');

  shortcuts.set(normalized, handler);

  // 返回清理函数
  return () => {
    shortcuts.delete(normalized);
  };
}

/**
 * 注销快捷键
 *
 * @param shortcut - 快捷键字符串
 */
export function unregisterShortcut(shortcut: string): void {
  const normalized = shortcut
    .replace(/Command/g, 'Cmd')
    .replace(/⌘/g, 'Cmd')
    .replace(/⇧/g, 'Shift')
    .replace(/⌥/g, 'Option')
    .replace(/⌃/g, 'Ctrl');

  shortcuts.delete(normalized);
}

/**
 * 初始化快捷键系统
 */
export function initShortcuts(): void {
  window.addEventListener('keydown', handleKeyDown);
}

/**
 * 销毁快捷键系统
 */
export function destroyShortcuts(): void {
  window.removeEventListener('keydown', handleKeyDown);
  shortcuts.clear();
}

/**
 * 预定义的快捷键常量
 */
export const SHORTCUTS = {
  TOGGLE_AI_ASSISTANT: 'Cmd+Shift+M',
  OPEN_DASHBOARD: 'Cmd+Shift+D',
  OPEN_CLIPBOARD: 'Cmd+Shift+C',
  QUICK_SAVE: 'Cmd+Shift+V',
  TOGGLE_MONITORING: 'Cmd+Shift+P',
} as const;
