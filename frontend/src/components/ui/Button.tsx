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
  'inline-flex items-center justify-center gap-1.5 rounded-sm border font-semibold whitespace-nowrap cursor-pointer ' +
  'transition-colors duration-150 active:translate-y-px disabled:cursor-not-allowed disabled:opacity-45 disabled:active:translate-y-0';

const variants: Record<Variant, string> = {
  primary: 'border-primary-hover bg-primary text-on-primary uppercase tracking-wider hover:bg-primary-hover',
  secondary: 'border-line-strong bg-surface text-fg uppercase tracking-wider hover:bg-surface-2 hover:border-accent',
  ghost: 'border-transparent text-muted hover:text-fg hover:bg-surface-2',
  danger: 'border-line-strong bg-surface text-fg uppercase tracking-wider hover:text-danger hover:border-danger hover:bg-danger-soft',
};

const sizes: Record<Size, string> = {
  sm: 'h-7 px-2.5 text-xs',
  md: 'h-8 px-4 text-xs',
  lg: 'h-10 px-6 text-xs',
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
      {loading ? <Loader2 className="size-3.5 animate-spin" aria-hidden /> : Icon && <Icon className="size-3.5" aria-hidden />}
      {children}
    </button>
  );
}
