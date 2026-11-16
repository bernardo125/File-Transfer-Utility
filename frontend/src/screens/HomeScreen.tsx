import { useState } from 'react';
import { ArrowRight, Download, LogOut, Send, ShieldCheck } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { QuitApplication } from '../wailsjs/go/ui/App';
import { hasBackendMethod } from '../lib/wails';
import { BrandMark } from '../components/common/BrandMark';
import { Button } from '../components/ui/Button';
import type { PeerRole } from '../types/navigation';

interface HomeScreenProps {
  onStart: (role: PeerRole) => void;
}

interface ActionCardProps {
  icon: LucideIcon;
  title: string;
  description: string;
  primary?: boolean;
  onClick: () => void;
}

function ActionCard({ icon: Icon, title, description, primary, onClick }: ActionCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`group flex flex-1 items-center gap-3 rounded-md border px-4 py-3.5 text-left transition-colors cursor-pointer ${
        primary
          ? 'border-primary-hover bg-primary text-on-primary hover:bg-primary-hover'
          : 'border-line-strong bg-surface text-fg hover:border-accent hover:bg-surface-2'
      }`}
    >
      <Icon className={`size-5 shrink-0 ${primary ? '' : 'text-accent'}`} aria-hidden />
      <span className="min-w-0 flex-1">
        <span className="block text-sm font-semibold uppercase tracking-wider">{title}</span>
        <span className={`block text-xs ${primary ? 'text-on-primary' : 'text-muted'}`}>{description}</span>
      </span>
      <ArrowRight className="size-4 shrink-0 opacity-60 transition-transform duration-200 group-hover:translate-x-1 group-focus-visible:translate-x-1" aria-hidden />
    </button>
  );
}

export function HomeScreen({ onStart }: HomeScreenProps) {
  const [isExiting, setIsExiting] = useState(false);

  const handleExit = async () => {
    if (!hasBackendMethod('QuitApplication')) {
      console.warn('Wails backend not connected (browser mode).');
      return;
    }
    setIsExiting(true);
    try {
      await QuitApplication();
    } catch (err) {
      console.error('Failed to quit application:', err);
      setIsExiting(false);
    }
  };

  return (
    <>
      <main className="flex flex-1 flex-col items-center justify-center overflow-y-auto px-6 py-8">
        <div className="flex w-full max-w-xl flex-col items-center gap-7">
          <BrandMark />

          <div className="flex w-full flex-col gap-3 sm:flex-row">
            <ActionCard primary icon={Send} title="Send" description="Throw a file to a peer" onClick={() => onStart('send')} />
            <ActionCard icon={Download} title="Receive" description="Catch a file from a peer" onClick={() => onStart('receive')} />
          </div>

          <div className="flex w-full flex-wrap items-center justify-between gap-3 border-t border-line pt-3 text-[11px] text-dim">
            <span className="inline-flex items-center gap-1.5">
              <ShieldCheck className="size-3.5 text-accent" aria-hidden />
              Encrypted QUIC (TLS 1.3) · Ed25519 peer identity · SHA-256 verified
            </span>
            <Button variant="ghost" size="sm" icon={LogOut} loading={isExiting} onClick={handleExit}>
              Exit
            </Button>
          </div>
        </div>
      </main>
    </>
  );
}
