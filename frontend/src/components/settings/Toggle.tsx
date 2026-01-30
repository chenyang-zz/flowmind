/**
 * 开关切换组件
 *
 * 用于设置页面中的开关选项
 */

interface ToggleProps {
  /**
   * 是否开启
   */
  enabled: boolean;

  /**
   * 切换回调
   */
  onToggle: () => void;

  /**
   * 是否禁用（默认 false）
   */
  disabled?: boolean;

  /**
   * 自定义类名
   */
  className?: string;
}

/**
 * 开关切换组件
 *
 * 提供统一的开关样式
 */
export function Toggle({ enabled, onToggle, disabled = false, className = '' }: ToggleProps) {
  return (
    <button
      onClick={onToggle}
      disabled={disabled}
      className={`
        relative
        w-11 h-6
        rounded-full
        transition-colors duration-150
        focus:outline-none
        ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}
        ${enabled ? 'bg-indigo-500' : 'bg-white/10'}
        ${className}
      `}
      type="button"
    >
      <div
        className={`
          absolute top-1
          w-4 h-4
          rounded-full
          bg-white
          shadow-sm
          transition-transform duration-150
          ${enabled ? 'translate-x-6' : 'translate-x-1'}
        `}
      />
    </button>
  );
}
