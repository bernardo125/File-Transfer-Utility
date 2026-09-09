import React, { useState } from 'react';
import { LogoPlaceholder } from '../components/common/LogoPlaceholder';
import { QuitApplication } from '../wailsjs/go/ui/App';
import type { ViewMode, PairingMode } from '../types/navigation';
import { Send, Download, LogOut } from 'lucide-react';

interface HomeScreenProps {
  onNavigate: (view: ViewMode, mode?: PairingMode) => void;
}


export const HomeScreen: React.FC<HomeScreenProps> = ({ onNavigate }) => {
  const [isExiting, setIsExiting] = useState<boolean>(false);

  const handleExit = async () => {
    if (isExiting) return;
    setIsExiting(true);
    try {
      const hasWailsQuit =
        typeof window !== 'undefined' &&
        typeof (window as unknown as { go?: { ui?: { App?: { QuitApplication?: unknown } } } }).go?.ui?.App?.QuitApplication === 'function';

      if (hasWailsQuit) {
        await QuitApplication();
      } else {
        console.warn('Wails backend not connected (running in browser mode).');
        setTimeout(() => setIsExiting(false), 1500);
      }
    } catch (err: unknown) {
      console.error('Failed to quit application:', err);
      setIsExiting(false);
    }
  };


  return (
    <div className="flex-1 flex items-center justify-center p-6">
      <div className="flex flex-col items-center gap-7 max-w-md w-full -mt-6">
        <LogoPlaceholder />
        <div className="flex items-center gap-2.5 w-full">
          <button
            type="button"
            disabled={isExiting}
            onClick={() => onNavigate('pairing', 'send')}
            className="flex-1 min-w-[120px] h-10 px-5 rounded-[3px] bg-[#2fa340] hover:bg-[#288d37] active:bg-[#1f752c] disabled:opacity-50 text-white font-semibold text-xs tracking-wider uppercase border border-[#278535] shadow-xs flex items-center justify-center gap-2 transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400"
          >
            <Send className="w-3.5 h-3.5 stroke-[2.2]" />
            <span>Send</span>
          </button>

          {/* Receive Button: Desktop Utility Panel Button */}
          <button
            type="button"
            disabled={isExiting}
            onClick={() => onNavigate('pairing', 'receive')}
            className="flex-1 min-w-[120px] h-10 px-5 rounded-[3px] bg-[var(--bg-panel)] hover:bg-[var(--bg-panel-hover)] active:bg-[var(--bg-inset)] disabled:opacity-50 hover:border-[#2fa340] border border-[var(--border-app)] font-semibold text-xs tracking-wider uppercase shadow-xs flex items-center justify-center gap-2 transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400"
          >
            <Download className="w-3.5 h-3.5 text-[#2fa340] stroke-[2.2]" />
            <span>Receive</span>
          </button>

          {/* Exit Button: Neutral / Subtle Desktop Button */}
          <button
            type="button"
            disabled={isExiting}
            onClick={handleExit}
            className="min-w-[95px] h-10 px-4 rounded-[3px] bg-[var(--bg-panel)] hover:bg-rose-950/20 active:bg-[var(--bg-inset)] hover:text-rose-400 hover:border-rose-800/60 disabled:opacity-50 text-[var(--text-muted)] border border-[var(--border-app)] font-semibold text-xs tracking-wider uppercase shadow-xs flex items-center justify-center gap-1.5 transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-rose-500/50"
          >
            <LogOut className="w-3.5 h-3.5 stroke-[2.2]" />
            <span>{isExiting ? '...' : 'Exit'}</span>
          </button>
        </div>
      </div>
    </div>
  );
};
