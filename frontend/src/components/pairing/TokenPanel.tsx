import { useState } from 'react';
import { Check, ClipboardPaste, Copy, Link2 } from 'lucide-react';
import { CopyToClipboard, GetClipboardText } from '../../wailsjs/go/ui/App';
import { Button } from '../ui/Button';

interface TokenPanelProps {
  localToken: string;
  isLoadingToken: boolean;
  remoteToken: string;
  onRemoteTokenChange: (value: string) => void;
  onConnect: () => void;
}

function StepLabel({ n, children }: { n: number; children: string }) {
  return (
    <div className="mb-2 flex items-center gap-2">
      <span className="flex size-5 items-center justify-center rounded-full bg-accent-soft text-[11px] font-bold text-accent">{n}</span>
      <h2 className="text-sm font-semibold text-fg">{children}</h2>
    </div>
  );
}

/** The two pairing steps: share your token, then paste your peer's. */
export function TokenPanel({ localToken, isLoadingToken, remoteToken, onRemoteTokenChange, onConnect }: TokenPanelProps) {
  const [copied, setCopied] = useState(false);
  const canConnect = !isLoadingToken && remoteToken.trim().length > 0;

  const handleCopy = async () => {
    if (!localToken) return;
    try {
      await CopyToClipboard(localToken);
    } catch {
      await navigator.clipboard?.writeText(localToken);
    }
    setCopied(true);
    window.setTimeout(() => setCopied(false), 2000);
  };

  const handlePaste = async () => {
    try {
      const text = await GetClipboardText();
      if (text) onRemoteTokenChange(text.trim());
    } catch {
      const text = await navigator.clipboard?.readText().catch(() => '');
      if (text) onRemoteTokenChange(text.trim());
    }
  };

  return (
    <div className="flex flex-col gap-6">
      <section>
        <StepLabel n={1}>Share your token</StepLabel>
        <div className="flex items-stretch gap-2">
          <output
            aria-label="Your pairing token"
            aria-busy={isLoadingToken}
            className="selectable min-h-11 flex-1 break-all rounded-lg border border-line bg-inset px-3 py-2.5 font-mono text-xs leading-relaxed text-fg max-h-24 overflow-y-auto"
          >
            {isLoadingToken ? <span className="text-dim">Discovering your public address…</span> : localToken || <span className="text-dim">Unavailable</span>}
          </output>
          <Button
            variant={copied ? 'secondary' : 'primary'}
            icon={copied ? Check : Copy}
            disabled={isLoadingToken || !localToken}
            onClick={handleCopy}
            className="self-start"
          >
            {copied ? 'Copied' : 'Copy'}
          </Button>
        </div>
        <p className="mt-1.5 text-xs text-dim">Send it to your peer through any chat or email.</p>
      </section>

      <section>
        <StepLabel n={2}>Paste your peer's token</StepLabel>
        <div className="relative">
          <label htmlFor="remote-token" className="sr-only">Peer token</label>
          <textarea
            id="remote-token"
            value={remoteToken}
            onChange={(e) => onRemoteTokenChange(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                if (canConnect) onConnect();
              }
            }}
            placeholder="Paste the token your peer shared with you"
            rows={3}
            spellCheck={false}
            className="w-full resize-none rounded-lg border border-line bg-inset px-3 py-2.5 pr-24 font-mono text-xs leading-relaxed text-fg placeholder:font-sans placeholder:text-dim focus:border-accent focus:outline-none transition-colors"
          />
          <Button variant="ghost" size="sm" icon={ClipboardPaste} onClick={handlePaste} className="absolute right-1.5 top-1.5">
            Paste
          </Button>
        </div>
      </section>

      <Button variant="primary" size="lg" icon={Link2} disabled={!canConnect} onClick={onConnect} className="w-full">
        Connect to peer
      </Button>
    </div>
  );
}
