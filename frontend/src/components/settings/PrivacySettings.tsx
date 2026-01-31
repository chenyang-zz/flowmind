/**
 * 隐私设置组件
 */

import { useSettingsStore } from "../../stores/settingsStore";
import { SettingItem } from "../common/SettingItem";
import { SettingGroup } from "../common/SettingGroup";

export function PrivacySettings() {
  const { privacy, setPrivacy } = useSettingsStore();

  return (
    <SettingGroup title="隐私与安全" description="控制你的数据存储方式">
      {/* 数据存储位置 */}
      <SettingItem title="数据存储位置" type="custom">
        <div className="flex gap-3">
          <button
            onClick={() => setPrivacy("dataStorage", "local")}
            className={`
              flex-1
              px-4 py-4
              rounded-lg
              border
              transition-all duration-150
              ${
                privacy.dataStorage === "local"
                  ? "bg-indigo-500/10 border-indigo-500/50"
                  : "bg-white/3 border-white/6 hover:border-white/12"
              }
            `}
          >
            <div className="text-left">
              <div
                className={`text-sm font-medium mb-1 ${
                  privacy.dataStorage === "local"
                    ? "text-white/90"
                    : "text-white/60"
                }`}
              >
                仅本地
              </div>
              <div className="text-xs text-white/35">
                数据仅存储在你的设备上
              </div>
            </div>
          </button>

          <button
            onClick={() => setPrivacy("dataStorage", "cloud")}
            className={`
              flex-1
              px-4 py-4
              rounded-lg
              border
              transition-all duration-150
              ${
                privacy.dataStorage === "cloud"
                  ? "bg-indigo-500/10 border-indigo-500/50"
                  : "bg-white/3 border-white/6 hover:border-white/12"
              }
            `}
          >
            <div className="text-left">
              <div
                className={`text-sm font-medium mb-1 ${
                  privacy.dataStorage === "cloud"
                    ? "text-white/90"
                    : "text-white/60"
                }`}
              >
                云端同步
              </div>
              <div className="text-xs text-white/35">
                加密同步到云端（即将推出）
              </div>
            </div>
          </button>
        </div>
      </SettingItem>

      {/* 敏感内容过滤 */}
      <SettingItem
        type="switch"
        title="过滤敏感内容"
        description="自动过滤剪贴板中的密码、信用卡等敏感信息"
        enabled={privacy.filterSensitive}
        onToggle={() => setPrivacy("filterSensitive", !privacy.filterSensitive)}
      />

      {/* 数据保留期 */}
      <SettingItem
        type="select"
        title="数据保留时间"
        description="超过保留时间的数据将被自动删除"
        value={privacy.retentionDays.toString()}
        onChange={(value) => setPrivacy("retentionDays", parseInt(value))}
        options={[
          { label: "最近 7 天", value: "7" },
          { label: "最近 30 天", value: "30" },
          { label: "最近 90 天", value: "90" },
          { label: "最近 1 年", value: "365" },
          { label: "永久保留", value: "-1" },
        ]}
      />

      {/* 数据操作 */}
      <SettingItem type="custom" title="数据操作">
        <div className="flex gap-3">
          <button
            className="
              flex-1
              px-4 py-3
              rounded-lg
              border border-white/8
              bg-white/3
              text-white/60
              text-sm font-medium
              hover:bg-white/5
              hover:text-white/80
              transition-colors
            "
          >
            清除所有数据
          </button>
          <button
            className="
              flex-1
              px-4 py-3
              rounded-lg
              border border-white/8
              bg-white/3
              text-white/60
              text-sm font-medium
              hover:bg-white/5
              hover:text-white/80
              transition-colors
            "
          >
            导出数据
          </button>
        </div>
      </SettingItem>
    </SettingGroup>
  );
}
