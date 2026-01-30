/**
 * FlowMind 主应用组件
 *
 * 应用的顶层组件，负责：
 * 1. 初始化主题和快捷键
 * 2. 加载设置
 * 3. 渲染全局组件（菜单栏、AI 助手面板）
 */

import { useEffect } from "react";
import { Brain } from "lucide-react";
import { MenuBarIcon } from "./components/menubar/MenuBarIcon";
import { AIAssistantPanel } from "./components/ai-assistant/AIAssistantPanel";
import { useSettingsStore } from "./stores/settingsStore";
import { useNavigationStore } from "./stores/navigationStore";
import { initTheme } from "./lib/theme";
import { initShortcuts } from "./lib/shortcuts";
import { wails } from "./lib/wails";
import { Settings } from "./pages/Settings";

/**
 * 主页面组件
 */
function MainPage() {
  return (
    <div
      className="
        min-h-screen
        flex items-center justify-center
        bg-linear-to-br from-bg-primary to-bg-secondary
      "
    >
      <div className="text-center space-y-6 px-8">
        {/* Logo */}
        <div className="flex items-center justify-center gap-3 mb-8">
          <div
            className="
              flex items-center justify-center
              w-16 h-16
              rounded-2xl
              bg-indigo-500
              shadow-2xl
            "
          >
            <Brain size={36} className="text-white" />
          </div>
          <div className="text-left">
            <h1 className="text-3xl font-bold text-white/90">FlowMind</h1>
            <p className="text-sm text-white/45">AI 工作流智能体</p>
          </div>
        </div>

        {/* 功能介绍 */}
        <div className="max-w-md mx-auto space-y-4">
          <p className="text-white/60 leading-relaxed">
            FlowMind 是一个主动的 AI 工作流伴侣，通过监控学习你的工作模式，
            主动发现问题并提供智能自动化建议。
          </p>

          {/* 快捷键提示 */}
          <div
            className="
              glass-card
              px-6 py-4
              rounded-xl
              text-left
            "
          >
            <div className="text-xs text-white/35 mb-3">快捷键</div>
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-white/60">唤起 AI 助手</span>
                <kbd className="px-2 py-1 text-xs bg-white/5 rounded text-white/45">
                  ⌘⇧M
                </kbd>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-white/60">打开仪表板</span>
                <kbd className="px-2 py-1 text-xs bg-white/5 rounded text-white/45">
                  ⌘⇧D
                </kbd>
              </div>
            </div>
          </div>

          {/* 状态提示 */}
          <div className="flex items-center justify-center gap-2 text-sm text-white/35">
            <div className="w-2 h-2 rounded-full bg-green-400 animate-pulse" />
            <span>监控运行中</span>
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * 主应用组件
 */
function App() {
  const { loadSettings, theme } = useSettingsStore();
  const { currentPage } = useNavigationStore();

  // 初始化应用
  useEffect(() => {
    // 初始化主题
    initTheme();

    // 加载设置
    loadSettings();

    // 初始化快捷键系统
    initShortcuts();

    // 加载监控状态并设置应用上下文
    const initApp = async () => {
      try {
        const isRunning = await wails.isMonitoringRunning();
        console.log("监控运行状态:", isRunning);

        // TODO: 实时获取当前应用上下文
        // const context = await wails.getCurrentContext();
        // useAIAssistantStore.getState().setCurrentContext(context);
      } catch (error) {
        console.error("初始化应用失败:", error);
      }
    };

    initApp();
  }, [loadSettings]);

  return (
    <div className={`min-h-screen ${theme}`}>
      {/* 菜单栏图标 */}
      <div className="fixed top-4 right-4 z-30">
        <MenuBarIcon />
      </div>

      {/* AI 助手面板 */}
      <AIAssistantPanel />

      {/* 根据当前页面显示不同内容 */}
      {currentPage === "main" && <MainPage />}
      {currentPage === "settings" && <Settings />}
      {/* TODO: 其他页面 (dashboard, automations, knowledge) */}
    </div>
  );
}

export default App;
