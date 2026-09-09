import React from 'react';
import { useTheme } from '../../context/ThemeContext';
import { Sun, Moon } from 'lucide-react';
import type { ConnectionState } from '../../types/transfer';

interface TitleBarProps {
  connectionState?: ConnectionState;
}

export const TitleBar: React.FC<TitleBarProps> = ({ connectionState = 'idle' }) => {
  const { theme, toggleTheme } = useTheme();

  const isConnected = connectionState === 'connected';

  return (
    <header className="window-drag-region h-9 bg-[var(--bg-panel)] border-b border-[var(--border-app)] px-3 flex items-center justify-between select-none flex-shrink-0 text-xs text-[var(--text-main)] transition-colors duration-150">
      {/* Client Logo & Title */}
      <div className="flex items-center gap-2">
        {/* µTorrent-style micro emblem */}
        <div className="w-4 h-4 rounded-[2px] bg-[#2fa340] flex items-center justify-center text-white font-bold text-[10px] leading-none shadow-xs">
          µ
        </div>
        <span className="font-semibold text-[11px] tracking-tight text-[var(--text-main)]">
          File Transfer Utility <span className="font-normal text-[var(--text-dim)]">v1.0.0 (x64)</span>
        </span>
      </div>

      {/* Right Controls: Theme Toggle & Network Status */}
      <div className="window-no-drag flex items-center gap-3">
        {/* Network Status Dot */}
        <div className="flex items-center gap-1.5 px-2 py-0.5 rounded-[2px] bg-[var(--bg-inset)] border border-[var(--border-subtle)] text-[10px] text-[var(--text-muted)] font-mono">
          <span
            className={`w-2 h-2 rounded-full ${
              isConnected
                ? 'bg-[#2fa340]'
                : connectionState === 'hole_punching'
                ? 'bg-amber-500 animate-pulse'
                : 'bg-[#2fa340]/80'
            }`}
          />
          <span>{isConnected ? 'Connected' : 'Network OK'}</span>
        </div>

        {/* Theme Toggle Button */}
        <button
          type="button"
          onClick={toggleTheme}
          title={theme === 'dark' ? 'Mudar para Modo Claro' : 'Mudar para Modo Escuro'}
          aria-label="Alternar tema"
          className="p-1 rounded-[3px] hover:bg-[var(--border-subtle)] active:bg-[var(--border-app)] text-[var(--text-muted)] hover:text-[var(--text-main)] border border-transparent hover:border-[var(--border-subtle)] transition-colors cursor-pointer"
        >
          {theme === 'dark' ? (
            <Sun className="w-3.5 h-3.5 text-amber-400" />
          ) : (
            <Moon className="w-3.5 h-3.5 text-slate-700" />
          )}
        </button>
      </div>
    </header>
  );
};
