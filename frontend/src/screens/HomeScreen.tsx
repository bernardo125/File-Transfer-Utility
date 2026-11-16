import { useState } from 'react';
import { ArrowRight, Download, LogOut, Send, ShieldCheck } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { QuitApplication } from '../wailsjs/go/ui/App';
import { hasBackendMethod } from '../lib/wails';
import { AppHeader } from '../components/layout/AppHeader';
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
      className={`group flex flex-1 items-center gap-4 rounded-2xl border p-5 text-left transition-[border-color,background-color,transform,box-shadow] duration-200 cursor-pointer hover:-translate-y-0.5 ${
        primary
          ? 'border-primary bg-primary text-on-primary shadow-[0_14px_30px_-14px_var(--c-primary)] hover:bg-primary-hover'
          : 'border-line bg-surface text-fg shadow-card hover:border-accent/60'
      }`}
    >
      <span
        className={`flex size-12 shrink-0 items-center justify-center rounded-xl ${
          primary ? 'bg-white/15' : 'bg-accent-soft text-accent'
        }`}
      >
        <Icon className="size-6" aria-hidden />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block text-lg font-bold">{title}</span>
        <span className={`block text-sm ${primary ? 'text-on-primary' : 'text-muted'}`}>{description}</span>
      </span>
      <ArrowRight className="size-5 shrink-0 opacity-60 transition-transform group-hover:translate-x-1" aria-hidden />
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
      <AppHeader />
      <main className="flex flex-1 flex-col items-center justify-center overflow-y-auto px-6 py-8">
        <div className="anim-fade-up flex w-full max-w-2xl flex-col items-center gap-10">
          <BrandMark size="lg" />

          <div className="flex w-full flex-col gap-3 sm:flex-row">
            <ActionCard primary icon={Send} title="Send" description="Throw a file to a peer" onClick={() => onStart('send')} />
            <ActionCard icon={Download} title="Receive" description="Catch a file from a peer" onClick={() => onStart('receive')} />
          </div>

          <div className="flex w-full flex-wrap items-center justify-between gap-3 text-xs text-dim">
            <span className="inline-flex items-center gap-1.5">
              <ShieldCheck className="size-4 text-accent" aria-hidden />
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
