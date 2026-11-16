import type { HTMLAttributes, ReactNode } from 'react';

interface PanelProps extends Omit<HTMLAttributes<HTMLElement>, 'title'> {
  /** Optional title bar, styled like a desktop-utility group header. */
  title?: ReactNode;
}

/** Flat bordered surface with an optional title bar. The one container used across the app. */
export function Panel({ title, className = '', children, ...rest }: PanelProps) {
  return (
    <section className={`rounded-md border border-line bg-surface ${className}`} {...rest}>
      {title && (
        <h2 className="flex items-center gap-2 rounded-t-md border-b border-line bg-inset px-3 py-1.5 text-[0.6875rem] font-semibold uppercase tracking-wider text-muted">
          {title}
        </h2>
      )}
      {children}
    </section>
  );
}
