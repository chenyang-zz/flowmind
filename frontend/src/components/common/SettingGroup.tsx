/**
 * 设置组标题组件
 */

import { ReactNode } from "react";

interface SettingGroupProps {
  title: string;
  description?: string;
  children: ReactNode;
}

export function SettingGroup({
  title,
  description,
  children,
}: SettingGroupProps) {
  return (
    <div className="space-y-6">
      {/* 标题 */}
      <div>
        <h2 className="text-xl font-semibold text-white/90 mb-1">{title}</h2>
        {description && <p className="text-sm text-white/50">{description}</p>}
      </div>

      {/* 内容区域 */}
      {children}
    </div>
  );
}
