import type { ConnectionState } from '../../types/transfer';

const LABELS: Record<ConnectionState, string> = {
  idle: 'Not connected',
  discovery: 'Discovering address',
  ready: 'Ready to pair',
  hole_punching: 'Reaching peer',
  upgrading: 'Securing channel',
  connected: 'Connected',
  error: 'Connection lost',
};

const DOT: Record<ConnectionState, string> = {
  idle: 'bg-dim',
  discovery: 'bg-warn',
  ready: 'bg-dim',
  hole_punching: 'bg-warn',
  upgrading: 'bg-warn',
  connected: 'bg-accent',
  error: 'bg-danger',
};

interface ConnectionIndicatorProps {
  state: ConnectionState;
  rttMs?: number;
}

/** Compact live connection status shown in the header. */
export function ConnectionIndicator({ state, rttMs }: ConnectionIndicatorProps) {
  return (
    <div role="status" className="inline-flex items-center gap-2 rounded-sm border border-line bg-inset px-2.5 py-1 text-[11px] text-muted">
      <span className={`size-2 rounded-full ${DOT[state]}`} aria-hidden />
      <span className="font-medium text-fg">{LABELS[state]}</span>
      {state === 'connected' && rttMs ? <span className="tabular-nums">· {rttMs} ms RTT</span> : null}
    </div>
  );
}
