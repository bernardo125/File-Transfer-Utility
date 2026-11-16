import type { ButtonHTMLAttributes } from 'react';
import type { LucideIcon } from 'lucide-react';

interface IconButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> {
  icon: LucideIcon;
  label: string;
}

/** Square, icon-only button. `label` is used for both the accessible name and the tooltip. */
export function IconButton({ icon: Icon, label, className = '', ...rest }: IconButtonProps) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      className={`inline-flex size-9 items-center justify-center rounded-lg text-muted hover:text-fg hover:bg-surface-2 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed ${className}`}
      {...rest}
    >
      <Icon className="size-[18px]" aria-hidden />
    </button>
  );
}
