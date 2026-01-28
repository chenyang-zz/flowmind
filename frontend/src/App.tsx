/**
 * FlowMind 主应用组件
 *
 * 这是应用的顶层组件，负责：
 * 1. 提供全局上下文
 * 2. 管理应用级状态
 * 3. 定义应用的基础布局
 */

import React from 'react';
import { useAppStore } from './stores/appStore';

/**
 * 主应用组件
 *
 * @returns {JSX.Element} 应用界面
 */
function App(): JSX.Element {
  // 从 Zustand store 获取应用状态
  const { isLoading } = useAppStore();

  /**
   * 渲染加载状态
   */
  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-gray-50">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto mb-4" />
          <p className="text-gray-600">加载中...</p>
        </div>
      </div>
    );
  }

  /**
   * 渲染主界面
   */
  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-7xl mx-auto px-4 py-8">
        {/* 页面头部 */}
        <header className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900">FlowMind</h1>
          <p className="text-gray-600 mt-2">AI 工作流智能体</p>
        </header>

        {/* 主内容区 */}
        <main>
          <div className="bg-white rounded-lg shadow-md p-6">
            <h2 className="text-xl font-semibold mb-4">欢迎使用 FlowMind</h2>

            {/* 应用介绍 */}
            <p className="text-gray-700 mb-6">
              FlowMind 是一个主动的 AI 工作流伴侣，通过监控学习你的工作模式，
              主动发现问题并提供智能自动化建议。
            </p>

            {/* 功能卡片网格 */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* 功能卡片：智能工作流发现 */}
              <FeatureCard
                icon="🧠"
                title="智能工作流发现"
                description="AI 自动识别重复性操作模式，主动建议自动化方案"
              />

              {/* 功能卡片：实时 AI 助手 */}
              <FeatureCard
                icon="💡"
                title="实时 AI 助手"
                description="全局快捷键唤起，AI 理解当前工作状态并提供帮助"
              />

              {/* 功能卡片：智能知识管理 */}
              <FeatureCard
                icon="📚"
                title="智能知识管理"
                description="AI 自动分类、打标签、建立知识图谱，智能推荐"
              />

              {/* 功能卡片：智能自动化 */}
              <FeatureCard
                icon="🤖"
                title="智能自动化"
                description="自然语言描述需求，AI 生成并执行自动化脚本"
              />
            </div>
          </div>
        </main>
      </div>
    </div>
  );
}

/**
 * 功能卡片组件
 *
 * @param icon - 功能图标
 * @param title - 功能标题
 * @param description - 功能描述
 * @returns {JSX.Element} 功能卡片元素
 */
interface FeatureCardProps {
  icon: string;
  title: string;
  description: string;
}

function FeatureCard({ icon, title, description }: FeatureCardProps): JSX.Element {
  return (
    <div className="p-4 border border-gray-200 rounded-lg hover:shadow-md transition-shadow duration-200">
      <h3 className="font-semibold mb-2 flex items-center">
        <span className="text-2xl mr-2">{icon}</span>
        {title}
      </h3>
      <p className="text-sm text-gray-600">{description}</p>
    </div>
  );
}

export default App;
