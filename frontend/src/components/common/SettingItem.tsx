/**
 * 通用设置项组件
 * 支持多种控件类型：switch、input、select、buttonGroup
 */

import { Check, X, ChevronDown, Eye, EyeOff } from "lucide-react";
import { ReactNode, useState } from "react";

interface SecondaryAction {
  label: string;
  onClick: () => void;
  variant?: "default" | "primary" | "secondary";
  disabled?: boolean;
  loading?: boolean;
}

interface BaseSettingItemProps {
  title: string;
  description?: string;
  icon?: ReactNode;
  statusIndicator?: {
    status: "success" | "error" | "warning";
    icon?: ReactNode;
  };
  secondaryActions?: SecondaryAction[];
}

interface SwitchSettingItemProps extends BaseSettingItemProps {
  type: "switch";
  enabled: boolean;
  onToggle: () => void;
}

interface InputSettingItemProps extends BaseSettingItemProps {
  type: "input";
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  inputType?: "text" | "number" | "password";
  action?: ReactNode;
}

interface SelectOption {
  label: string;
  value: string;
}

interface SelectSettingItemProps extends BaseSettingItemProps {
  type: "select";
  value: string;
  onChange: (value: string) => void;
  options: SelectOption[];
}

interface CustomSettingItemProps extends BaseSettingItemProps {
  type: "custom";
  children: ReactNode;
}

type SettingItemProps =
  | SwitchSettingItemProps
  | InputSettingItemProps
  | SelectSettingItemProps
  | CustomSettingItemProps;

export function SettingItem(props: SettingItemProps) {
  const { title, description, icon, statusIndicator, secondaryActions } = props;

  const [showPassword, setShowPassword] = useState(false);

  const renderContent = () => {
    switch (props.type) {
      case "switch":
        return (
          <div className="flex items-center justify-between gap-6">
            <div className="text-xs text-white/40 leading-relaxed flex-1">
              {description}
            </div>
            <div className="shrink-0">
              {" "}
              <button
                onClick={props.onToggle}
                className={`
              relative
              w-11 h-6
              rounded-full
              transition-all duration-200
              shrink-0
              ${props.enabled ? "bg-indigo-500 shadow-lg shadow-indigo-500/30" : "bg-white/10 hover:bg-white/15"}
            `}
              >
                <div
                  className={`
                absolute top-0.5
                w-5 h-5
                rounded-full
                bg-white
                shadow-md
                transition-all duration-200
                ${props.enabled ? "translate-x-5" : "translate-x-0.5"}
              `}
                />
              </button>
            </div>
          </div>
        );

      case "select":
        return (
          <div className="flex items-center justify-between gap-6">
            <div className="text-xs text-white/40 leading-relaxed flex-1">
              {description}
            </div>
            <div className="shrink-0">
              <div className="relative min-w-50">
                <select
                  value={props.value}
                  onChange={(e) => props.onChange(e.target.value)}
                  className="
                w-full
                px-3 py-2
                pr-9
                text-sm
                bg-white/5
                border border-white/10
                rounded-lg
                text-white/90
                appearance-none
                focus:outline-none
                focus:ring-2
                focus:ring-indigo-500/40
                focus:border-indigo-500/40
                focus:bg-white/8
                transition-all
                cursor-pointer
              "
                >
                  {props.options.map((option) => (
                    <option
                      key={option.value}
                      value={option.value}
                      className="bg-[#1a1b26] text-white/80"
                    >
                      {option.label}
                    </option>
                  ))}
                </select>
                <ChevronDown
                  size={16}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-white/40 pointer-events-none"
                />
              </div>
            </div>
          </div>
        );
    }
  };

  const renderStatusIcon = () => {
    if (!statusIndicator) return null;

    if (statusIndicator.icon) {
      return statusIndicator.icon;
    }

    switch (statusIndicator.status) {
      case "success":
        return <Check size={14} className="text-green-400" />;
      case "error":
        return <X size={14} className="text-red-400" />;
      case "warning":
        return <span className="text-yellow-400">!</span>;
    }
  };

  const getButtonClassName = (
    variant: SecondaryAction["variant"] = "default",
  ) => {
    const baseClasses = `
      px-3 py-1.5
      text-xs font-medium
      rounded-lg
      transition-all duration-200
      disabled:opacity-50
      disabled:cursor-not-allowed
    `;

    switch (variant) {
      case "primary":
        return `${baseClasses} bg-indigo-500 text-white hover:bg-indigo-600 active:scale-95`;
      case "secondary":
        return `${baseClasses} bg-white/8 text-white/50 hover:bg-white/12 hover:text-white/70 border border-white/10`;
      default:
        return `${baseClasses} bg-white/8 text-white/60 hover:bg-white/12 hover:text-white/80`;
    }
  };

  if (props.type === "input") {
    return (
      <div>
        <label className="text-sm font-medium text-white/70">{title}</label>
        <div className="relative w-full my-2">
          <input
            type={
              props.inputType === "password" && !showPassword
                ? "password"
                : "text"
            }
            value={props.value}
            onChange={(e) => props.onChange(e.target.value)}
            placeholder={props.placeholder}
            className="
                w-full
                px-4 py-2.5
                pr-24
                bg-white/4
                border border-white/8
                rounded-lg
                text-sm
                text-white/90
                placeholder:text-white/20
                focus:outline-none
                focus:border-indigo-500/50
                transition-colors
              "
          />
          <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1">
            <button
              onClick={() => setShowPassword(!showPassword)}
              className="p-1.5 text-white/30 hover:text-white/60 transition-colors"
            >
              {showPassword ? <Eye size={16} /> : <EyeOff size={16} />}
            </button>
            {props.action}
          </div>
        </div>
        {description && <p className="text-xs text-white/30">{description}</p>}
      </div>
    );
  }

  if (props.type === "custom") {
    return (
      <div>
        <label className="text-sm font-medium text-white/70">{title}</label>
        <div className="relative w-full my-2">{props.children}</div>
        {description && <p className="text-xs text-white/30">{description}</p>}
      </div>
    );
  }

  return (
    <div className="bg-white/2 hover:bg-white/4 border border-white/8 rounded-xl px-6 py-5 transition-all duration-200">
      {/* Header: 标题、状态指示器和次要按钮 */}
      <div className="flex items-center justify-between gap-4 mb-3">
        <div className="flex items-center gap-2">
          {icon && <div className="shrink-0 text-white/60">{icon}</div>}
          <span className="text-sm font-semibold text-white/90">{title}</span>
          {renderStatusIcon()}
        </div>

        {/* 次要操作按钮 */}
        {secondaryActions && secondaryActions.length > 0 && (
          <div className="flex items-center gap-2">
            {secondaryActions.map((action, index) => (
              <button
                key={index}
                onClick={action.onClick}
                disabled={action.disabled || action.loading}
                className={getButtonClassName(action.variant)}
              >
                {action.loading ? "处理中..." : action.label}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Content: 描述和主控件 */}
      {renderContent()}
    </div>
  );
}
