/**
 * 数据管理组件
 */

import {
  Trash2,
  Download,
  BarChart3,
  Database,
  Brain,
  Zap,
} from "lucide-react";
import { SettingItem } from "../common/SettingItem";

export function DataManager() {
  // TODO: 从后端获取实际数据统计
  const stats = {
    dbSize: "124.5 MB",
    eventCount: 15234,
    patternCount: 128,
    clipCount: 89,
  };

  const clearAllData = async () => {
    if (!confirm("确定要清除所有数据吗？此操作不可恢复。")) {
      return;
    }

    try {
      // TODO: 调用后端 API 清除数据
      console.log("清除所有数据");
    } catch (error) {
      console.error("清除数据失败:", error);
    }
  };

  const exportData = async () => {
    try {
      // TODO: 调用后端 API 导出数据
      console.log("导出数据");
    } catch (error) {
      console.error("导出数据失败:", error);
    }
  };

  const backupData = async () => {
    try {
      // TODO: 调用后端 API 备份数据
      console.log("备份数据");
    } catch (error) {
      console.error("备份数据失败:", error);
    }
  };

  return (
    <div className="space-y-8">
      {/* 标题 */}
      <div>
        <h2 className="text-xl font-semibold text-white/90 mb-1">数据管理</h2>
        <p className="text-sm text-white/40">查看和管理你的数据</p>
      </div>

      {/* 数据统计 */}
      <div className="grid grid-cols-2 gap-4">
        <StatCard
          icon={<Database size={20} className="text-white/40" />}
          label="数据库大小"
          value={stats.dbSize}
        />
        <StatCard
          icon={<Zap size={20} className="text-white/40" />}
          label="事件数"
          value={stats.eventCount.toLocaleString()}
        />
        <StatCard
          icon={<Brain size={20} className="text-white/40" />}
          label="模式数"
          value={stats.patternCount}
        />
        <StatCard
          icon={<BarChart3 size={20} className="text-white/40" />}
          label="剪藏数"
          value={stats.clipCount}
        />
      </div>

      {/* 数据操作 */}
      <SettingItem title="数据操作" type="custom">
        <div className="space-y-2">
          <button
            onClick={backupData}
            className="
              w-full
              px-4 py-3
              rounded-lg
              border border-white/8
              bg-white/3
              text-white/60
              text-sm font-medium
              hover:bg-white/5
              hover:text-white/80
              transition-colors
              flex items-center justify-center gap-2
            "
          >
            <Download size={16} />
            备份数据
          </button>

          <button
            onClick={exportData}
            className="
              w-full
              px-4 py-3
              rounded-lg
              border border-white/8
              bg-white/3
              text-white/60
              text-sm font-medium
              hover:bg-white/5
              hover:text-white/80
              transition-colors
              flex items-center justify-center gap-2
            "
          >
            <BarChart3 size={16} />
            查看统计详情
          </button>

          <button
            onClick={clearAllData}
            className="
              w-full
              px-4 py-3
              rounded-lg
              border border-red-500/30
              bg-red-500/5
              text-red-400
              text-sm font-medium
              hover:bg-red-500/10
              hover:text-red-300
              transition-colors
              flex items-center justify-center gap-2
            "
          >
            <Trash2 size={16} />
            清除所有数据
          </button>
        </div>
      </SettingItem>
    </div>
  );
}

/**
 * 统计卡片组件
 */
interface StatCardProps {
  icon: React.ReactNode;
  label: string;
  value: string | number;
}

function StatCard({ icon, label, value }: StatCardProps) {
  return (
    <div className="bg-white/3 border border-white/6 rounded-lg px-4 py-4">
      <div className="flex items-center gap-2 mb-2">
        {icon}
        <span className="text-xs text-white/40">{label}</span>
      </div>
      <div className="text-2xl font-semibold text-white/90">{value}</div>
    </div>
  );
}
