/**
 * Wails 事件订阅 Hook
 *
 * 提供 React Hooks 用于订阅 Wails 后端事件
 * 以及一些常用的工具 Hooks
 */

import { useEffect, useState, useRef } from 'react';
import { EventsOn, EventsOff } from '../wailsjs/runtime';

/**
 * Wails 事件订阅 Hook
 *
 * 用于订阅来自 Wails 后端的事件
 *
 * @param eventName - 事件名称
 * @param handler - 事件处理函数
 *
 * @example
 * ```tsx
 * useWailsEvent('pattern:discovered', (pattern) => {
 *   console.log('New pattern:', pattern);
 *   toast.success(`发现新模式: ${pattern.name}`);
 * });
 * ```
 */
export function useWailsEvent<T = any>(
  eventName: string,
  handler: (data: T) => void
): void {
  // 使用 ref 存储 handler 引用，避免依赖变化导致重复订阅
  const handlerRef = useRef(handler);

  // 更新 handler 引用
  useEffect(() => {
    handlerRef.current = handler;
  }, [handler]);

  useEffect(() => {
    // 稳定的处理函数，使用 ref 中的最新 handler
    const stableHandler = (data: T) => {
      handlerRef.current(data);
    };

    // 订阅事件
    EventsOn(eventName, stableHandler);

    // 清理函数：取消订阅
    return () => {
      EventsOff(eventName);
    };
  }, [eventName]);
}

/**
 * 防抖 Hook
 *
 * 延迟更新值，只在用户停止输入指定时间后才更新
 *
 * @param value - 需要防抖的值
 * @param delay - 延迟时间（毫秒）
 * @returns 防抖后的值
 *
 * @example
 * ```tsx
 * const [searchTerm, setSearchTerm] = useState('');
 * const debouncedSearchTerm = useDebounce(searchTerm, 500);
 *
 * // 只有在用户停止输入 500ms 后，debouncedSearchTerm 才会更新
 * useEffect(() => {
 *   // 执行搜索
 *   search(debouncedSearchTerm);
 * }, [debouncedSearchTerm]);
 * ```
 */
export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    // 设置定时器
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    // 清理函数：清除定时器
    return () => {
      clearTimeout(handler);
    };
  }, [value, delay]);

  return debouncedValue;
}

/**
 * 间隔 Hook
 *
 * 定时执行某个函数
 *
 * @param callback - 要执行的回调函数
 * @param delay - 间隔时间（毫秒），null 表示停止
 *
 * @example
 * ```tsx
 * useInterval(() => {
 *   // 每秒执行一次
 *   console.log('Tick');
 * }, 1000);
 * ```
 */
export function useInterval(callback: () => void, delay: number | null): void {
  const savedCallback = useRef(callback);

  // 更新回调函数引用
  useEffect(() => {
    savedCallback.current = callback;
  }, [callback]);

  useEffect(() => {
    // 如果 delay 为 null，不执行
    if (delay === null) {
      return;
    }

    // 设置定时器
    const tick = () => {
      savedCallback.current();
    };

    const id = setInterval(tick, delay);

    // 清理函数：清除定时器
    return () => {
      clearInterval(id);
    };
  }, [delay]);
}

/**
 * LocalStorage Hook
 *
 * 用于同步状态到 localStorage
 *
 * @param key - localStorage 键名
 * @param initialValue - 初始值
 * @returns [value, setValue] 状态值和设置函数
 *
 * @example
 * ```tsx
 * const [name, setName] = useLocalStorage('name', 'Guest');
 *
 * // name 会自动保存到 localStorage
 * // 刷新页面后会恢复
 * ```
 */
export function useLocalStorage<T>(
  key: string,
  initialValue: T
): [T, (value: T | ((val: T) => T)) => void] {
  // 获取初始值
  const [storedValue, setStoredValue] = useState<T>(() => {
    try {
      // 从 localStorage 获取值
      const item = window.localStorage.getItem(key);
      return item ? JSON.parse(item) : initialValue;
    } catch (error) {
      console.error(`Error loading localStorage key "${key}":`, error);
      return initialValue;
    }
  });

  // 设置值
  const setValue = (value: T | ((val: T) => T)) => {
    try {
      // 允许传入函数来更新值
      const valueToStore = value instanceof Function ? value(storedValue) : value;

      setStoredValue(valueToStore);

      // 保存到 localStorage
      window.localStorage.setItem(key, JSON.stringify(valueToStore));
    } catch (error) {
      console.error(`Error setting localStorage key "${key}":`, error);
    }
  };

  return [storedValue, setValue];
}
