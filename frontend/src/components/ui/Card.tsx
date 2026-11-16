import type { HTMLAttributes } from 'react';

/** Base elevated surface used by every panel in the app. */
export function Card({ className = '', ...rest }: HTMLAttributes<HTMLDivElement>) {
  return <div className={`rounded-2xl border border-line bg-surface shadow-card ${className}`} {...rest} />;
}
