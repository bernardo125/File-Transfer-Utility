import type { ReactNode } from 'react';

interface AppHeaderProps {
  /** Left-hand navigation (e.g. a Back or Disconnect button). */
  leading?: ReactNode;
  /** Centered status content. */
  center?: ReactNode;
}

/** Floating screen controls (a banner landmark). No bar: it overlays the content and takes no layout space. */
export function AppHeader({ leading, center }: AppHeaderProps) {
  return (
    <header className="pointer-events-none absolute inset-x-0 top-0 z-10 grid grid-cols-[1fr_auto_1fr] items-center gap-3 p-3">
      <div className="pointer-events-auto flex min-w-0 items-center">{leading}</div>
      <div className="pointer-events-auto">{center}</div>
      <div />
    </header>
  );
}
