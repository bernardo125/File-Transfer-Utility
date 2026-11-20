import { ShieldAlert, ShieldCheck } from 'lucide-react';
import { Button } from '../ui/Button';

interface FingerprintPanelProps {
  fingerprint: string;
  detail: string;
  busy: boolean;
  onMatch: () => void;
  onMismatch: () => void;
}

/**
 * Last pairing step. The tokens travelled over a channel nobody authenticated, so the two users
 * must read this code to each other (voice or another channel) and confirm it is identical. The
 * backend keeps the data channel closed until "They match" is pressed; this panel only reports the
 * user's decision.
 */
export function FingerprintPanel({ fingerprint, detail, busy, onMatch, onMismatch }: FingerprintPanelProps) {
  return (
    <div className="flex flex-col gap-4">
      <p className="text-xs text-muted" aria-live="polite">{detail}</p>
      <div className="rounded-sm border border-line bg-inset px-3 py-4 text-center">
        <p className="text-[0.6875rem] font-semibold uppercase tracking-wider text-dim">Pairing fingerprint</p>
        <p
          className="mt-1 select-all break-all font-mono text-lg font-semibold tracking-widest text-fg"
          aria-label={`Pairing fingerprint ${fingerprint.split('-').join(', ')}`}
        >
          {fingerprint}
        </p>
      </div>
      <p className="text-xs text-dim">
        Ask your peer to read theirs aloud. If a single character differs, someone may be tampering with the connection.
      </p>
      <div className="flex flex-wrap gap-2">
        <Button variant="primary" icon={ShieldCheck} disabled={busy} onClick={onMatch}>
          They match
        </Button>
        <Button variant="danger" icon={ShieldAlert} disabled={busy} onClick={onMismatch}>
          They don&apos;t match
        </Button>
      </div>
    </div>
  );
}
