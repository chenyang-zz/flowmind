/**
 * 当前应用上下文显示组件
 * 极简设计
 */

import { AppContext } from '../../lib/wails';
import { Monitor, Globe, Compass, Keyboard, Folder, Palette, MessageSquare, FileText } from 'lucide-react';

interface ContextDisplayProps {
  context?: AppContext;
}

/**
 * 应用图标映射
 */
const APP_ICONS: Record<string, React.ReactNode> = {
  'VS Code': <Monitor size={18} className="text-white/60" />,
  'Chrome': <Globe size={18} className="text-white/60" />,
  'Safari': <Compass size={18} className="text-white/60" />,
  'Terminal': <Keyboard size={18} className="text-white/60" />,
  'Finder': <Folder size={18} className="text-white/60" />,
  'Figma': <Palette size={18} className="text-white/60" />,
  'Slack': <MessageSquare size={18} className="text-white/60" />,
  'Notion': <FileText size={18} className="text-white/60" />,
};

/**
 * 获取应用图标
 */
function getAppIcon(appName: string): React.ReactNode {
  return APP_ICONS[appName] || null;
}

/**
 * 上下文显示组件
 */
export function ContextDisplay({ context }: ContextDisplayProps) {
  if (!context) return null;

  const appIcon = getAppIcon(context.application);

  return (
    <div
      className="
        bg-white/[0.03]
        border border-white/[0.06]
        px-4 py-3
        rounded-lg
        space-y-1.5
      "
    >
      <div className="flex items-center gap-3">
        {appIcon && <div className="flex-shrink-0">{appIcon}</div>}
        <div className="flex-1 min-w-0">
          <div className="text-sm font-medium text-white/90 truncate">
            {context.application}
          </div>
          {context.windowTitle && (
            <div className="text-xs text-white/40 truncate mt-0.5">
              {context.windowTitle}
            </div>
          )}
        </div>
      </div>

      {context.filePath && (
        <div
          className="
            text-xs text-white/30
            font-mono
            bg-black/20
            px-2 py-1
            rounded
            truncate
          "
        >
          {context.filePath}
        </div>
      )}
    </div>
  );
}
