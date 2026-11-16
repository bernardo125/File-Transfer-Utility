import type { ButtonHTMLAttributes, ComponentType } from 'react';

interface IconButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> {
  icon: ComponentType<{ className?: string; 'aria-hidden'?: boolean }>;
  label: string;
}

/** Round icon-only button (the floating GitHub / theme tools). `label` is the accessible name and tooltip. */
export function IconButton({ icon: Icon, label, className = '', ...rest }: IconButtonProps) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      className={`inline-flex size-10 items-center justify-center rounded-full border border-line bg-surface text-muted transition-colors cursor-pointer hover:bg-surface-2 hover:text-fg disabled:opacity-40 disabled:cursor-not-allowed ${className}`}
      {...rest}
    >
      <Icon className="size-5" aria-hidden />
    </button>
  );
}
