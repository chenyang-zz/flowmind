/**
 * 设置页面
 *
 * 应用配置和权限管理
 */

import { useMemo, useState } from "react";
import {
  ArrowLeft,
  Settings as SettingsIcon,
  Brain,
  BarChart3,
  Lock,
  Keyboard,
  Database,
} from "lucide-react";
import { useSettingsStore } from "../stores/settingsStore";
import { useNavigationStore } from "../stores/navigationStore";
import { GeneralSettings } from "../components/settings/GeneralSettings";
import { AISettings } from "../components/settings/AISettings";
import { MonitoringSettings } from "../components/settings/MonitoringSettings";
import { PrivacySettings } from "../components/settings/PrivacySettings";
import { ShortcutSettings } from "../components/settings/ShortcutSettings";
import { DataManager } from "../components/settings/DataManager";

/**
 * 设置选项卡类型
 */
type SettingsTab =
  | "general"
  | "ai"
  | "monitoring"
  | "privacy"
  | "shortcuts"
  | "data";

/**
 * 设置选项卡配置
 */
const TABS: { id: SettingsTab; label: string; icon: React.ReactNode }[] = [
  { id: "general", label: "通用", icon: <SettingsIcon size={16} /> },
  { id: "ai", label: "AI", icon: <Brain size={16} /> },
  { id: "monitoring", label: "监控", icon: <BarChart3 size={16} /> },
  { id: "privacy", label: "隐私", icon: <Lock size={16} /> },
  { id: "shortcuts", label: "快捷键", icon: <Keyboard size={16} /> },
  { id: "data", label: "数据", icon: <Database size={16} /> },
];

/**
 * 设置页面组件
 */
export function Settings() {
  const [activeTab, setActiveTab] = useState<SettingsTab>("general");
  const { theme } = useSettingsStore();
  const { goToMain } = useNavigationStore();

  const mainContent = useMemo(() => {
    return (
      <div className="px-6 pb-8 flex-1 overflow-auto">
        <div className="flex flex-col lg:flex-row gap-6 lg:gap-8 pt-6">
          {/* 内容区 */}
          <div className="flex-1 min-w-0">
            <div className="bg-white/2 border border-white/6 rounded-xl p-6 lg:p-8">
              {activeTab === "general" && <GeneralSettings />}
              {activeTab === "ai" && <AISettings />}
              {activeTab === "monitoring" && <MonitoringSettings />}
              {activeTab === "privacy" && <PrivacySettings />}
              {activeTab === "shortcuts" && <ShortcutSettings />}
              {activeTab === "data" && <DataManager />}
            </div>
          </div>
        </div>
      </div>
    );
  }, [activeTab]);

  return (
    <div
      className={`h-screen bg-linear-to-br from-bg-primary to-bg-secondary flex flex-col ${theme} `}
    >
      {/* 顶部栏 - 固定在顶部，带背景和阴影 */}
      <div className="z-20 border-b border-white/5 bg-bg-primary/95 backdrop-blur-sm px-4 pb-4 pt-8">
        <div className="flex items-center gap-3  mx-auto">
          <button
            onClick={goToMain}
            className="
              p-2
              rounded-lg
              text-white/40
              hover:text-white/70
              hover:bg-white/5
              transition-colors
            "
          >
            <ArrowLeft size={18} />
          </button>
          <div className="flex items-center gap-3 flex-1">
            <div
              className="
                flex items-center justify-center
                w-10 h-10
                rounded-xl
                bg-indigo-500
                shadow-lg
              "
            >
              <SettingsIcon size={20} className="text-white" />
            </div>
            <div>
              <h1 className="text-lg font-semibold text-white/90">设置</h1>
              <p className="text-xs text-white/40">配置 FlowMind</p>
            </div>
          </div>
        </div>
      </div>

      {/* 移动端：水平滚动的标签栏 */}
      <div className="w-full lg:w-56 lg:hidden shrink-0 flex-1 flex flex-col overflow-hidden">
        <div className="space-y-1">
          <div className=" flex gap-2 overflow-x-auto pb-2 scrollbar-hide px-6 py-2 border-b border-white/5">
            {TABS.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`
                      shrink-0
                      px-4 py-2
                      rounded-lg
                      text-sm font-medium
                      flex items-center gap-2
                      transition-all duration-150
                      whitespace-nowrap
                      ${
                        activeTab === tab.id
                          ? "bg-white/6 text-white/90"
                          : "text-white/40 hover:text-white/60 hover:bg-white/3"
                      }
                    `}
              >
                {tab.icon}
                <span>{tab.label}</span>
              </button>
            ))}
          </div>
        </div>
        <div className="flex-1 overflow-auto">
          {/* 主体内容 */}
          {mainContent}
        </div>
      </div>

      <div className="flex-1 lg:flex max-lg:hidden overflow-hidden">
        {/* 桌面端：垂直标签栏 */}
        <div className="space-y-1 pl-4 py-4 w-56 overflow-auto">
          {TABS.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`
                      w-full
                      px-4 py-2.5
                      rounded-lg
                      text-left
                      text-sm font-medium
                      flex items-center gap-3
                      transition-all duration-150
                      ${
                        activeTab === tab.id
                          ? "bg-white/6 text-white/90"
                          : "text-white/40 hover:text-white/60 hover:bg-white/3"
                      }
                    `}
            >
              {tab.icon}
              <span>{tab.label}</span>
            </button>
          ))}
        </div>

        <div className="flex-1 overflow-auto">
          {/* 主体内容 */}
          {mainContent}
        </div>
      </div>
    </div>
  );
}
