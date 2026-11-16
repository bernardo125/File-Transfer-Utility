import { Check, Loader2, X } from 'lucide-react';
import type { ConnectionState } from '../../types/transfer';
import { Button } from '../ui/Button';

const STAGES: Array<{ state: ConnectionState; label: string; hint: string }> = [
  { state: 'hole_punching', label: 'Reaching your peer', hint: 'UDP hole punching' },
  { state: 'upgrading', label: 'Securing the channel', hint: 'QUIC + TLS, Ed25519 identity' },
  { state: 'connected', label: 'Connected', hint: 'Direct peer-to-peer link' },
];

interface ConnectingPanelProps {
  state: ConnectionState;
  detail: string;
  onCancel: () => void;
}

/** Live view of the real connection stages reported by the backend, as a compact status list. */
export function ConnectingPanel({ state, detail, onCancel }: ConnectingPanelProps) {
  const current = STAGES.findIndex((s) => s.state === state);

  return (
    <div className="flex flex-col gap-4">
      <p className="text-xs text-muted" aria-live="polite">{detail || 'Waiting for the network…'}</p>

      <ol className="overflow-hidden rounded-sm border border-line">
        {STAGES.map((stage, i) => {
          const done = current > i || state === 'connected';
          const active = current === i && !done;
          return (
            <li
              key={stage.state}
              aria-current={active ? 'step' : undefined}
              className={`flex items-center gap-3 border-b border-line px-3 py-2 last:border-b-0 transition-colors duration-300 ${active ? 'bg-accent-soft' : i % 2 ? 'bg-inset' : 'bg-surface'}`}
            >
              <span className="flex size-4 shrink-0 items-center justify-center">
                {done ? (
                  <Check key="done" className="anim-pop size-4 text-accent" aria-hidden />
                ) : active ? (
                  <Loader2 className="size-4 animate-spin text-accent" aria-hidden />
                ) : (
                  <span className="size-1.5 rounded-full bg-dim" aria-hidden />
                )}
              </span>
              <span className={`flex-1 text-xs font-semibold ${done || active ? 'text-fg' : 'text-dim'}`}>{stage.label}</span>
              <span className="text-[11px] text-dim">{stage.hint}</span>
            </li>
          );
        })}
      </ol>

      <Button variant="danger" icon={X} onClick={onCancel} className="self-start">
        Cancel
      </Button>
    </div>
  );
}
