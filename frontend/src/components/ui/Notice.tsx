import type { ReactNode } from 'react';
import { AlertTriangle, CheckCircle2, Info, X } from 'lucide-react';

type Tone = 'error' | 'success' | 'info';

const tones: Record<Tone, { box: string; icon: typeof Info }> = {
  error: { box: 'border-l-danger [&_svg.tone]:text-danger', icon: AlertTriangle },
  success: { box: 'border-l-accent [&_svg.tone]:text-accent', icon: CheckCircle2 },
  info: { box: 'border-l-line-strong [&_svg.tone]:text-muted', icon: Info },
};

interface NoticeProps {
  tone?: Tone;
  children: ReactNode;
  onDismiss?: () => void;
}

/** Inline status message. Errors are announced assertively, others politely. */
export function Notice({ tone = 'info', children, onDismiss }: NoticeProps) {
  const { box, icon: Icon } = tones[tone];
  return (
    <div
      role={tone === 'error' ? 'alert' : 'status'}
      className={`anim-fade-up flex items-start gap-2.5 rounded-sm border border-l-[3px] border-line bg-surface-2 px-3 py-2 text-xs text-fg ${box}`}
    >
      <Icon className="tone mt-px size-4 shrink-0" aria-hidden />
      <div className="min-w-0 flex-1 break-words">{children}</div>
      {onDismiss && (
        <button type="button" onClick={onDismiss} aria-label="Dismiss message" className="-m-1 rounded-sm p-1 text-muted hover:text-fg cursor-pointer">
          <X className="size-3.5" aria-hidden />
        </button>
      )}
    </div>
  );
}
