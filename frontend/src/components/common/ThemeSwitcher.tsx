import { Sun, Moon } from 'lucide-react';

interface ThemeSwitcherProps {
  theme: 'light' | 'dark';
  onClick: () => void;
}
export const ThemeSwitcher : React.FC<ThemeSwitcherProps> = ({ theme, onClick }) => {

  return (
    <div className=" inline-flex items-center">
      {/* Discreet Theme Switcher */}
      <button
        type="button"
        onClick={onClick}
        title={theme === 'dark' ? 'Modo Claro' : 'Modo Escuro'}
        className="z-50 p-1.5 rounded-[20px] bg-[var(--bg-panel)] hover:bg-[var(--bg-panel-hover)] border border-[var(--border-app)] text-[var(--text-muted)] hover:text-[var(--text-main)] transition-colors cursor-pointer"
      >
        {theme === 'dark' ? <Sun className="w-8 h-8 text-amber-400" /> : <Moon className="w-8 h-8 text-blue-400" />}
      </button>

    </div>
  );
};
