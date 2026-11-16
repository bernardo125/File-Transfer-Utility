import type { ButtonHTMLAttributes, ReactNode } from 'react';
import type { LucideIcon } from 'lucide-react';
import { Loader2 } from 'lucide-react';

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger';
type Size = 'sm' | 'md' | 'lg';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
  icon?: LucideIcon;
  loading?: boolean;
  children: ReactNode;
}

const base =
  'inline-flex items-center justify-center gap-2 rounded-lg font-semibold whitespace-nowrap cursor-pointer ' +
  'transition-[background-color,border-color,color,box-shadow,transform] duration-150 active:translate-y-px ' +
  'disabled:cursor-not-allowed disabled:opacity-45 disabled:active:translate-y-0';

const variants: Record<Variant, string> = {
  primary: 'bg-primary text-on-primary hover:bg-primary-hover shadow-[0_1px_0_rgba(255,255,255,0.12)_inset]',
  secondary: 'bg-surface-2 text-fg border border-line hover:border-line-strong hover:bg-inset',
  ghost: 'text-muted hover:text-fg hover:bg-surface-2',
  danger: 'bg-surface-2 text-fg border border-line hover:text-danger hover:border-danger/50 hover:bg-danger-soft',
};

const sizes: Record<Size, string> = {
  sm: 'h-8 px-3 text-xs',
  md: 'h-10 px-4 text-sm',
  lg: 'h-12 px-6 text-[15px]',
};

export function Button({ variant = 'secondary', size = 'md', icon: Icon, loading, disabled, className = '', children, ...rest }: ButtonProps) {
  return (
    <button
      type="button"
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={`${base} ${variants[variant]} ${sizes[size]} ${className}`}
      {...rest}
    >
      {loading ? <Loader2 className="size-4 animate-spin" aria-hidden /> : Icon && <Icon className="size-4" aria-hidden />}
      {children}
    </button>
  );
}
