/**
 * 监控设置组件
 */

import { useState } from "react";
import { AlertCircle } from "lucide-react";
import { useSettingsStore } from "../../stores/settingsStore";
import { wails } from "../../lib/wails";
import { SettingItem } from "../common/SettingItem";
import { SettingGroup } from "../common/SettingGroup";

export function MonitoringSettings() {
  const { monitoring, setMonitoring } = useSettingsStore();
  const [checkingPermissions, setCheckingPermissions] = useState<
    Record<string, boolean>
  >({});
  const [permissionStatus, setPermissionStatus] = useState<
    Record<string, "granted" | "denied" | "unknown">
  >({});

  // 检查权限状态
  const checkPermissionStatus = async (permType: string) => {
    setCheckingPermissions((prev) => ({ ...prev, [permType]: true }));
    try {
      const status = await wails.checkPermission(permType);
      setPermissionStatus((prev) => ({
        ...prev,
        [permType]: status.status as any,
      }));
    } catch (error) {
      console.error("检查权限失败:", error);
    } finally {
      setCheckingPermissions((prev) => ({ ...prev, [permType]: false }));
    }
  };

  // 请求权限
  const requestPermission = async (permType: string) => {
    try {
      await wails.requestPermission(permType);
      // 重新检查状态
      await checkPermissionStatus(permType);
    } catch (error) {
      console.error("请求权限失败:", error);
    }
  };

  // 打开系统设置
  const openSystemSettings = (permType: string) => {
    // TODO: 调用后端 API 打开系统隐私设置
    console.log("打开系统设置:", permType);
  };

  return (
    <SettingGroup title="监控设置" description="配置 FlowMind 监控你的活动">
      {/* 监控项列表 */}
      <div className="space-y-3">
        {/* 键盘监控 */}
        {/* 键盘监控 */}
        <SettingItem
          type="switch"
          title="键盘监控"
          description="记录键盘输入，用于识别打字模式和快捷键"
          enabled={monitoring.keyboard}
          onToggle={() => setMonitoring("keyboard", !monitoring.keyboard)}
          statusIndicator={
            permissionStatus.accessibility === "granted"
              ? { status: "success" }
              : permissionStatus.accessibility === "denied"
                ? { status: "error" }
                : undefined
          }
          secondaryActions={[
            {
              label: checkingPermissions.keyboard ? "检查中..." : "检查权限",
              onClick: () => checkPermissionStatus("accessibility"),
              disabled: checkingPermissions.keyboard,
              loading: checkingPermissions.keyboard,
            },
            ...(permissionStatus.accessibility === "denied"
              ? [
                  {
                    label: "授予权限",
                    onClick: () => requestPermission("accessibility"),
                    variant: "primary" as const,
                  },
                  {
                    label: "打开设置",
                    onClick: () => openSystemSettings("accessibility"),
                    variant: "secondary" as const,
                  },
                ]
              : []),
          ]}
        />

        {/* 剪贴板监控 */}
        <SettingItem
          type="switch"
          title="剪贴板监控"
          description="监控剪贴板内容变化，用于智能剪藏功能"
          enabled={monitoring.clipboard}
          onToggle={() => setMonitoring("clipboard", !monitoring.clipboard)}
          statusIndicator={
            permissionStatus.accessibility === "granted"
              ? { status: "success" }
              : permissionStatus.accessibility === "denied"
                ? { status: "error" }
                : undefined
          }
          secondaryActions={[
            {
              label: checkingPermissions.clipboard ? "检查中..." : "检查权限",
              onClick: () => checkPermissionStatus("accessibility"),
              disabled: checkingPermissions.clipboard,
              loading: checkingPermissions.clipboard,
            },
            ...(permissionStatus.accessibility === "denied"
              ? [
                  {
                    label: "授予权限",
                    onClick: () => requestPermission("accessibility"),
                    variant: "primary" as const,
                  },
                  {
                    label: "打开设置",
                    onClick: () => openSystemSettings("accessibility"),
                    variant: "secondary" as const,
                  },
                ]
              : []),
          ]}
        />

        {/* 应用切换监控 */}
        <SettingItem
          type="switch"
          title="应用切换监控"
          description="跟踪应用切换，用于工作流分析"
          enabled={monitoring.appSwitch}
          onToggle={() => setMonitoring("appSwitch", !monitoring.appSwitch)}
          statusIndicator={
            permissionStatus.accessibility === "granted"
              ? { status: "success" }
              : permissionStatus.accessibility === "denied"
                ? { status: "error" }
                : undefined
          }
          secondaryActions={[
            {
              label: checkingPermissions.appSwitch ? "检查中..." : "检查权限",
              onClick: () => checkPermissionStatus("accessibility"),
              disabled: checkingPermissions.appSwitch,
              loading: checkingPermissions.appSwitch,
            },
            ...(permissionStatus.accessibility === "denied"
              ? [
                  {
                    label: "授予权限",
                    onClick: () => requestPermission("accessibility"),
                    variant: "primary" as const,
                  },
                  {
                    label: "打开设置",
                    onClick: () => openSystemSettings("accessibility"),
                    variant: "secondary" as const,
                  },
                ]
              : []),
          ]}
        />

        {/* 文件系统监控 */}
        <SettingItem
          type="switch"
          title="文件系统监控"
          description="监控文件变化，用于自动分类和管理"
          enabled={monitoring.fileSystem}
          onToggle={() => setMonitoring("fileSystem", !monitoring.fileSystem)}
          statusIndicator={
            permissionStatus.files === "granted"
              ? { status: "success" }
              : permissionStatus.files === "denied"
                ? { status: "error" }
                : undefined
          }
          secondaryActions={[
            {
              label: checkingPermissions.fileSystem ? "检查中..." : "检查权限",
              onClick: () => checkPermissionStatus("files"),
              disabled: checkingPermissions.fileSystem,
              loading: checkingPermissions.fileSystem,
            },
            ...(permissionStatus.files === "denied"
              ? [
                  {
                    label: "授予权限",
                    onClick: () => requestPermission("files"),
                    variant: "primary" as const,
                  },
                  {
                    label: "打开设置",
                    onClick: () => openSystemSettings("files"),
                    variant: "secondary" as const,
                  },
                ]
              : []),
          ]}
        />
      </div>

      {/* 提示信息 */}
      <div className="flex items-start gap-3 px-4 py-3 bg-indigo-500/5 border border-indigo-500/10 rounded-lg">
        <AlertCircle size={16} className="text-indigo-400 shrink-0 mt-0.5" />
        <div className="text-xs text-white/50 leading-relaxed">
          FlowMind
          需要相应的系统权限才能正常工作。所有数据都存储在本地，不会上传到云端。
        </div>
      </div>
    </SettingGroup>
  );
}
