import { Check, Loader2, X } from 'lucide-react';
import type { ConnectionState } from '../../types/transfer';
import { Button } from '../ui/Button';

const STAGES: Array<{ state: ConnectionState; label: string; hint: string }> = [
  { state: 'hole_punching', label: 'Reaching your peer', hint: 'UDP hole punching through both networks' },
  { state: 'upgrading', label: 'Securing the channel', hint: 'QUIC + TLS handshake, verifying Ed25519 identity' },
  { state: 'connected', label: 'Connected', hint: 'Direct, encrypted peer-to-peer link' },
];

interface ConnectingPanelProps {
  state: ConnectionState;
  detail: string;
  onCancel: () => void;
}

/** Live view of the real connection stages reported by the backend. */
export function ConnectingPanel({ state, detail, onCancel }: ConnectingPanelProps) {
  const current = STAGES.findIndex((s) => s.state === state);

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h2 className="text-lg font-bold text-fg">Connecting…</h2>
        <p className="mt-1 text-sm text-muted" aria-live="polite">{detail || 'Waiting for the network.'}</p>
      </div>

      <ol className="flex flex-col gap-3">
        {STAGES.map((stage, i) => {
          const done = current > i || state === 'connected';
          const active = current === i && !done;
          return (
            <li key={stage.state} className="flex items-start gap-3" aria-current={active ? 'step' : undefined}>
              <span
                className={`mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full border ${
                  done ? 'border-accent bg-accent text-bg' : active ? 'border-accent text-accent' : 'border-line text-dim'
                }`}
              >
                {done ? <Check className="size-3.5" aria-hidden /> : active ? <Loader2 className="size-3.5 animate-spin" aria-hidden /> : <span className="text-[11px] font-semibold">{i + 1}</span>}
              </span>
              <span>
                <span className={`block text-sm font-semibold ${done || active ? 'text-fg' : 'text-dim'}`}>{stage.label}</span>
                <span className="block text-xs text-dim">{stage.hint}</span>
              </span>
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
