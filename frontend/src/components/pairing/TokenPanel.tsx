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

const fieldLabel = 'mb-1 block text-[0.6875rem] font-semibold uppercase tracking-wider text-muted';

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
    <div className="flex flex-col gap-4">
      <p role="status" className="sr-only">{copied ? 'Your token was copied to the clipboard.' : ''}</p>
      <div>
        <span id="your-token-label" className={fieldLabel}>Your token</span>
        <output
          aria-labelledby="your-token-label"
          aria-busy={isLoadingToken}
          className="selectable block max-h-20 min-h-11 overflow-y-auto break-all rounded-sm border border-control bg-inset px-2.5 py-2 text-xs leading-relaxed text-fg"
        >
          {isLoadingToken ? <span className="animate-pulse text-dim">Discovering your public address…</span> : localToken || <span className="text-dim">Unavailable</span>}
        </output>
        <div className="mt-1.5 flex items-center justify-between gap-3">
          <p className="text-[0.6875rem] text-dim">Send it to your peer through any chat or email.</p>
          <Button variant="secondary" size="sm" icon={copied ? Check : Copy} disabled={isLoadingToken || !localToken} onClick={handleCopy}>
            {copied ? 'Copied' : 'Copy token'}
          </Button>
        </div>
      </div>

      <div>
        <label htmlFor="remote-token" className={fieldLabel}>Peer token</label>
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
          autoComplete="off"
          className="w-full resize-none rounded-sm border border-control bg-inset px-2.5 py-2 text-xs leading-relaxed text-fg placeholder:text-dim focus:border-accent transition-colors"
        />
        <div className="mt-1.5 flex justify-end">
          <Button variant="secondary" size="sm" icon={ClipboardPaste} onClick={handlePaste}>
            Paste
          </Button>
        </div>
      </div>

      <Button variant="primary" size="lg" icon={Link2} disabled={!canConnect} onClick={onConnect} className="w-full">
        Connect
      </Button>
    </div>
  );
}
