/**
 * FlowMind 主入口文件
 *
 * 负责 React 应用的初始化和渲染
 */

import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import './styles/index.css';

/**
 * 创建 React 根节点并渲染应用
 *
 * 功能说明：
 * 1. 获取 DOM 中的 #app 元素作为根容器
 * 2. 使用 React 18+ 的 createRoot API 创建并发渲染根节点
 * 3. 启用 StrictMode 进行额外的检查和警告
 */
const root = ReactDOM.createRoot(
  document.getElementById('app') as HTMLElement
);

/**
 * 渲染主应用组件
 *
 * @see App 主应用组件
 */
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
