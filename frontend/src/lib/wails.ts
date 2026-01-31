/**
 * Wails 通信封装
 *
 * 提供类型安全的前后端通信接口
 */

import { EventsOn } from '../../wailsjs/runtime/runtime';

/**
 * 后端 API 方法定义
 */
export interface WailsApp {
  // 仪表板数据
  GetDashboardData(): Promise<Record<string, unknown>>;

  // 事件查询
  GetEvents(limit: number): Promise<Record<string, unknown>[]>;

  // 模式识别
  GetPatterns(): Promise<Record<string, unknown>[]>;

  // 自动化管理
  CreateAutomation(req: Record<string, unknown>): Promise<Record<string, unknown>>;

  // 监控控制
  StartMonitoring(): Promise<void>;
  StopMonitoring(): Promise<void>;
  IsMonitoringRunning(): Promise<boolean>;

  // 权限管理
  CheckPermission(permType: string): Promise<Record<string, unknown>>;
  RequestPermission(permType: string): Promise<void>;
}

/**
 * 获取当前应用上下文
 */
export interface AppContext {
  application: string;
  bundleId: string;
  windowTitle: string;
  filePath?: string;
  selection?: string;
}

/**
 * 事件数据结构
 */
export interface Event {
  id: string;
  type: string;
  timestamp: string;
  data: Record<string, unknown>;
  metadata?: Record<string, string>;
  context?: AppContext;
}

/**
 * Wails 运行时封装
 */
export class WailsRuntime {
  /**
   * 调用后端方法
   */
  static async call<T>(method: string, ...args: unknown[]): Promise<T> {
    try {
      // @ts-ignore - Wails 会注入这些方法
      const result = await window.runtime.go.main.App[method](...args);
      return result as T;
    } catch (error) {
      console.error(`Wails 调用失败: ${method}`, error);
      throw error;
    }
  }

  /**
   * 监听后端事件
   */
  static on(eventName: string, callback: (...args: unknown[]) => void): void {
    EventsOn(eventName, callback);
  }

  /**
   * 获取仪表板数据
   */
  static async getDashboardData(): Promise<Record<string, unknown>> {
    return this.call<Record<string, unknown>>('GetDashboardData');
  }

  /**
   * 获取事件列表
   */
  static async getEvents(limit: number = 100): Promise<Event[]> {
    return this.call<Record<string, unknown>[]>('GetEvents', limit) as Promise<Event[]>;
  }

  /**
   * 获取模式列表
   */
  static async getPatterns(): Promise<Record<string, unknown>[]> {
    return this.call<Record<string, unknown>[]>('GetPatterns');
  }

  /**
   * 创建自动化
   */
  static async createAutomation(
    req: Record<string, unknown>
  ): Promise<Record<string, unknown>> {
    return this.call<Record<string, unknown>>('CreateAutomation', req);
  }

  /**
   * 启动监控
   */
  static async startMonitoring(): Promise<void> {
    return this.call<void>('StartMonitoring');
  }

  /**
   * 停止监控
   */
  static async stopMonitoring(): Promise<void> {
    return this.call<void>('StopMonitoring');
  }

  /**
   * 检查监控运行状态
   */
  static async isMonitoringRunning(): Promise<boolean> {
    return this.call<boolean>('IsMonitoringRunning');
  }

  /**
   * 检查权限状态
   */
  static async checkPermission(permType: string): Promise<Record<string, unknown>> {
    return this.call<Record<string, unknown>>('CheckPermission', permType);
  }

  /**
   * 请求权限
   */
  static async requestPermission(permType: string): Promise<void> {
    return this.call<void>('RequestPermission', permType);
  }
}

/**
 * 导出单例
 */
export const wails = WailsRuntime;
