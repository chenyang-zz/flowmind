/**
 * 快捷键设置组件
 */

import { Edit2 } from "lucide-react";
import { SHORTCUTS } from "../../lib/shortcuts";
import { SettingGroup } from "../common/SettingGroup";

export function ShortcutSettings() {
  return (
    <SettingGroup title="快捷键绑定" description="自定义全局快捷键">
      {/* 快捷键列表 */}
      <div className="space-y-2">
        <ShortcutItem action="打开仪表板" shortcut={SHORTCUTS.OPEN_DASHBOARD} />
        <ShortcutItem
          action="AI 助手"
          shortcut={SHORTCUTS.TOGGLE_AI_ASSISTANT}
        />
        <ShortcutItem action="剪贴板历史" shortcut={SHORTCUTS.OPEN_CLIPBOARD} />
        <ShortcutItem action="快速剪藏" shortcut={SHORTCUTS.QUICK_SAVE} />
        <ShortcutItem
          action="暂停/恢复监控"
          shortcut={SHORTCUTS.TOGGLE_MONITORING}
        />
      </div>

      {/* 提示信息 */}
      <div className="flex items-start gap-3 px-4 py-3 bg-white/3 border border-white/6 rounded-lg">
        <Edit2 size={16} className="text-white/30 shrink-0 mt-0.5" />
        <div className="text-xs text-white/40 leading-relaxed">
          快捷键自定义功能即将推出。目前可以在系统中修改快捷键绑定。
        </div>
      </div>
    </SettingGroup>
  );
}

/**
 * 快捷键项组件
 */
interface ShortcutItemProps {
  action: string;
  shortcut: string;
}

function ShortcutItem({ action, shortcut }: ShortcutItemProps) {
  // 格式化快捷键显示
  const formatShortcut = (key: string) => {
    return key
      .replace("Cmd", "⌘")
      .replace("Shift", "⇧")
      .replace("Ctrl", "⌃")
      .replace("Option", "⌥")
      .replace("+", "");
  };

  return (
    <div className="flex items-center justify-between px-5 py-3 bg-white/3 border border-white/6 rounded-lg hover:bg-white/4 transition-colors">
      <span className="text-sm text-white/70">{action}</span>
      <kbd className="px-3 py-1.5 text-xs bg-white/5 text-white/40 rounded font-mono">
        {formatShortcut(shortcut)}
      </kbd>
    </div>
  );
}
