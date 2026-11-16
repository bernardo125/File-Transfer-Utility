import { GitHubButton } from '../common/GitHubButton';
import { MotionToggle } from '../common/MotionToggle';
import { ThemeSwitcher } from '../common/ThemeSwitcher';

/** GitHub link, theme and motion toggles, pinned to the bottom-left corner on every screen. */
export function FloatingTools() {
  return (
    <aside aria-label="Application tools" className="fixed bottom-3 left-3 z-50 flex gap-2">
      <GitHubButton />
      <ThemeSwitcher />
      <MotionToggle />
    </aside>
  );
}
