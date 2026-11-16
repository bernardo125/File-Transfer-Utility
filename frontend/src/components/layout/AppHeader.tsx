import type { ReactNode } from 'react';
import { BrandMark } from '../common/BrandMark';
import { GitHubButton } from '../common/GitHubButton';
import { ThemeSwitcher } from '../common/ThemeSwitcher';

interface AppHeaderProps {
  /** Left-hand navigation (e.g. a Back button). The brand is shown when absent. */
  leading?: ReactNode;
  /** Centered status content. */
  center?: ReactNode;
}

export function AppHeader({ leading, center }: AppHeaderProps) {
  return (
    <header className="window-drag-region grid h-14 shrink-0 grid-cols-[1fr_auto_1fr] items-center gap-3 border-b border-line/70 px-4">
      <div className="window-no-drag flex min-w-0 items-center">{leading ?? <BrandMark />}</div>
      <div className="window-no-drag">{center}</div>
      <div className="window-no-drag flex items-center justify-end gap-1">
        <GitHubButton />
        <ThemeSwitcher />
      </div>
    </header>
  );
}
