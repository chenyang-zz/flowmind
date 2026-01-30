/**
 * 通用设置组件
 */

import { Moon, Sun } from "lucide-react";
import { useSettingsStore } from "../../stores/settingsStore";
import { SettingItem } from "../common/SettingItem";
import { SettingGroup } from "../common/SettingGroup";

export function GeneralSettings() {
  const {
    theme,
    setTheme,
    language,
    setLanguage,
    autoStart,
    setAutoStart,
    notificationsEnabled,
    setNotificationsEnabled,
  } = useSettingsStore();

  return (
    <SettingGroup title="通用设置" description="配置应用的基本行为">
      {/* 主题设置 */}
      <SettingItem title="主题" type="custom">
        <div className="flex gap-3">
          <button
            onClick={() => setTheme("dark")}
            className={`
              flex-1
              px-4 py-5
              rounded-lg
              border
              text-sm font-medium
              transition-all duration-150
              ${
                theme === "dark"
                  ? "bg-indigo-500/10 border-indigo-500/50 text-white/90"
                  : "bg-white/3 border-white/6 text-white/40 hover:text-white/60"
              }
            `}
          >
            <div className="flex flex-col items-center gap-2">
              <div
                className={`
                  w-12 h-12 rounded-lg border-2 flex items-center justify-center
                  ${
                    theme === "dark"
                      ? "border-indigo-500 bg-bg-primary"
                      : "border-white/10 bg-white/5"
                  }
                `}
              >
                <Moon
                  size={24}
                  className={
                    theme === "dark" ? "text-indigo-400" : "text-white/40"
                  }
                />
              </div>
              <span>深色</span>
            </div>
          </button>
          <button
            onClick={() => setTheme("light")}
            className={`
              flex-1
              px-4 py-5
              rounded-lg
              border
              text-sm font-medium
              transition-all duration-150
              ${
                theme === "light"
                  ? "bg-indigo-500/10 border-indigo-500/50 text-white/90"
                  : "bg-white/3 border-white/6 text-white/40 hover:text-white/60"
              }
            `}
          >
            <div className="flex flex-col items-center gap-2">
              <div
                className={`
                  w-12 h-12 rounded-lg border-2 flex items-center justify-center
                  ${
                    theme === "light"
                      ? "border-indigo-500 bg-white"
                      : "border-white/10 bg-white/5"
                  }
                `}
              >
                <Sun
                  size={24}
                  className={
                    theme === "light" ? "text-indigo-500" : "text-white/40"
                  }
                />
              </div>
              <span>浅色</span>
            </div>
          </button>
        </div>
      </SettingItem>

      {/* 语言设置 */}
      <SettingItem
        type="select"
        title="语言"
        description="选择应用显示语言"
        value={language}
        onChange={setLanguage}
        options={[
          { label: "简体中文", value: "zh-CN" },
          { label: "English", value: "en-US" },
        ]}
      />

      {/* 启动和通知选项 */}
      <div className="space-y-3">
        <SettingItem
          type="switch"
          title="登录时自动启动"
          description="系统启动时自动运行 FlowMind"
          enabled={autoStart}
          onToggle={() => setAutoStart(!autoStart)}
        />

        <SettingItem
          type="switch"
          title="启用通知"
          description="显示 AI 洞察和建议通知"
          enabled={notificationsEnabled}
          onToggle={() => setNotificationsEnabled(!notificationsEnabled)}
        />
      </div>
    </SettingGroup>
  );
}
