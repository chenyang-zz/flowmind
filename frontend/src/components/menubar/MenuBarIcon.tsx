/**
 * 菜单栏图标组件
 *
 * 简洁模式 - 只显示图标，点击展开菜单
 */

import { useState, useRef, useEffect } from "react";
import { Brain, CheckCircle, PauseCircle } from "lucide-react";
import { useAIAssistantStore } from "../../stores/aiAssistantStore";
import { useSettingsStore } from "../../stores/settingsStore";
import { useNavigationStore } from "../../stores/navigationStore";
import { wails } from "../../lib/wails";

/**
 * 菜单项数据结构
 */
interface MenuItem {
  label: string;
  icon?: React.ReactNode;
  shortcut?: string;
  action: () => void;
  divider?: boolean;
}

/**
 * 菜单栏图标组件
 */
export function MenuBarIcon() {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isMonitoringRunning, setIsMonitoringRunning] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  const { openPanel } = useAIAssistantStore();
  const { theme } = useSettingsStore();
  const { goToSettings } = useNavigationStore();

  // 加载监控状态
  useEffect(() => {
    const loadMonitoringStatus = async () => {
      try {
        const isRunning = await wails.isMonitoringRunning();
        setIsMonitoringRunning(isRunning);
      } catch (error) {
        console.error("加载监控状态失败:", error);
      }
    };

    loadMonitoringStatus();
  }, []);

  // 点击外部关闭菜单
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsMenuOpen(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // 切换监控状态
  const toggleMonitoring = async () => {
    try {
      if (isMonitoringRunning) {
        await wails.stopMonitoring();
        setIsMonitoringRunning(false);
      } else {
        await wails.startMonitoring();
        setIsMonitoringRunning(true);
      }
    } catch (error) {
      console.error("切换监控状态失败:", error);
    }
    setIsMenuOpen(false);
  };

  // 菜单项配置
  const menuItems: MenuItem[] = [
    {
      label: "打开仪表板",
      shortcut: "⌘⇧D",
      action: () => {
        console.log("打开仪表板");
        setIsMenuOpen(false);
      },
    },
    {
      label: "AI 助手",
      shortcut: "⌘⇧M",
      action: () => {
        openPanel();
        setIsMenuOpen(false);
      },
    },
    {
      label: "设置",
      action: () => {
        goToSettings();
        setIsMenuOpen(false);
      },
    },
    {
      divider: true,
      label: "",
      action: () => {},
    },
    {
      label: isMonitoringRunning ? "监控运行中" : "监控已暂停",
      icon: isMonitoringRunning ? (
        <CheckCircle size={14} className="text-green-400" />
      ) : (
        <PauseCircle size={14} className="text-yellow-400" />
      ),
      action: () => {},
    },
    {
      label: isMonitoringRunning ? "暂停监控" : "恢复监控",
      action: toggleMonitoring,
    },
    {
      divider: true,
      label: "",
      action: () => {},
    },
    {
      label: "退出",
      action: () => {
        console.log("退出应用");
        setIsMenuOpen(false);
      },
    },
  ];

  return (
    <div className="relative" ref={menuRef}>
      {/* 菜单栏图标 */}
      <button
        onClick={() => setIsMenuOpen(!isMenuOpen)}
        className={`
          flex items-center justify-center
          w-8 h-8
          rounded-lg
          transition-all duration-150
          ${isMenuOpen ? "bg-white/10" : "hover:bg-white/5"}
        `}
        style={{
          color:
            theme === "dark"
              ? "rgba(255, 255, 255, 0.9)"
              : "rgba(0, 0, 0, 0.9)",
        }}
      >
        <Brain size={18} />
      </button>

      {/* 下拉菜单 */}
      {isMenuOpen && (
        <div
          className={`
            absolute top-full right-0 mt-2
            min-w-50
            glass-panel
            animate-slide-in-right
            z-50
          `}
        >
          <div className="p-2">
            <div
              className="
                px-3 py-2
                text-xs font-semibold
                text-white/45
                uppercase tracking-wider
              "
            >
              FlowMind
            </div>

            {menuItems.map((item, index) => {
              if (item.divider) {
                return (
                  <div
                    key={index}
                    className="
                      my-1
                      h-px
                      bg-white/10
                    "
                  />
                );
              }

              return (
                <button
                  key={index}
                  onClick={item.action}
                  className="
                    w-full
                    px-3 py-2
                    text-left
                    rounded-lg
                    text-sm
                    flex items-center justify-between
                    transition-colors duration-150
                    hover:bg-white/5
                    text-white/90
                  "
                >
                  <span className="flex items-center gap-2">
                    {item.icon && <span>{item.icon}</span>}
                    {item.label}
                  </span>
                  {item.shortcut && (
                    <span className="text-xs text-white/35">
                      {item.shortcut}
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
