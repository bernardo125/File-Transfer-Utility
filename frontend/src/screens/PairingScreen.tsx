import { useEffect, useState } from 'react';
import { ArrowLeft, Download, Send } from 'lucide-react';
import { CancelConnection, ConnectPeer, GetLocalToken } from '../wailsjs/go/ui/App';
import { useSession } from '../context/SessionContext';
import { useCountdown } from '../hooks/useCountdown';
import { errorMessage } from '../lib/wails';
import { AppHeader } from '../components/layout/AppHeader';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Notice } from '../components/ui/Notice';
import { PairingVisual } from '../components/pairing/PairingVisual';
import { TokenPanel } from '../components/pairing/TokenPanel';
import { ConnectingPanel } from '../components/pairing/ConnectingPanel';
import type { PeerRole } from '../types/navigation';

const CONNECT_TIMEOUT_S = 30;

interface PairingScreenProps {
  role: PeerRole;
  onBack: () => void;
  onConnected: () => void;
}

export function PairingScreen({ role, onBack, onConnected }: PairingScreenProps) {
  const { connection, connectionSuccess, connectionFailed, resetConnectionOutcome } = useSession();
  const [localToken, setLocalToken] = useState('');
  const [isLoadingToken, setIsLoadingToken] = useState(true);
  const [remoteToken, setRemoteToken] = useState('');
  const [isConnecting, setIsConnecting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const { remaining, start: startTimeout, stop: stopTimeout } = useCountdown(CONNECT_TIMEOUT_S, () => {
    setIsConnecting(false);
    setError(`Could not reach your peer within ${CONNECT_TIMEOUT_S} seconds. Check that both tokens are current and try again.`);
    CancelConnection().catch(() => {});
  });

  useEffect(() => {
    let cancelled = false;
    GetLocalToken()
      .then((token) => !cancelled && setLocalToken(token))
      .catch((err) => !cancelled && setError(`Could not discover your public address: ${errorMessage(err)}`))
      .finally(() => !cancelled && setIsLoadingToken(false));
    return () => {
      cancelled = true;
    };
  }, []);

  // The backend's verdict for the current attempt. Success navigates away; failure is shown inline.
  const failedReason = isConnecting && connectionFailed ? connectionFailed.reason || 'The connection failed.' : null;
  const connecting = isConnecting && !failedReason;
  const shownError = error ?? failedReason;

  useEffect(() => {
    if (!isConnecting) return;
    if (connectionSuccess) {
      stopTimeout();
      onConnected();
    } else if (connectionFailed) {
      stopTimeout();
    }
  }, [connectionSuccess, connectionFailed, isConnecting, onConnected, stopTimeout]);

  const dismissError = () => {
    setError(null);
    setIsConnecting(false);
  };

  const handleConnect = async () => {
    const token = remoteToken.trim();
    if (!token) return;
    stopTimeout();
    setError(null);
    resetConnectionOutcome();
    setIsConnecting(true);
    startTimeout();
    try {
      await ConnectPeer(token);
    } catch (err) {
      stopTimeout();
      setIsConnecting(false);
      setError(errorMessage(err));
    }
  };

  const handleCancel = () => {
    stopTimeout();
    setIsConnecting(false);
    CancelConnection().catch(() => {});
  };

  const RoleIcon = role === 'send' ? Send : Download;

  return (
    <>
      <AppHeader
        leading={
          <Button variant="ghost" size="sm" icon={ArrowLeft} disabled={connecting} onClick={onBack}>
            Back
          </Button>
        }
        center={
          <span className="inline-flex items-center gap-2 rounded-full border border-line bg-surface px-3 py-1 text-xs font-semibold text-fg">
            <RoleIcon className="size-3.5 text-accent" aria-hidden />
            {role === 'send' ? 'Sending' : 'Receiving'} · Pair with your peer
          </span>
        }
      />
      <main className="flex flex-1 items-center justify-center overflow-y-auto px-6 py-8">
        <Card className="anim-fade-up grid w-full max-w-3xl items-center gap-8 p-8 md:grid-cols-[240px_1fr]">
          <div className="flex flex-col items-center gap-4 text-center">
            <PairingVisual countdown={connecting ? { remaining, total: CONNECT_TIMEOUT_S } : undefined} />
            <p className="max-w-[220px] text-xs text-dim">
              Tokens are exchanged by you, not by a server. Both peers connect directly.
            </p>
          </div>

          <div className="flex min-w-0 flex-col gap-4">
            {shownError && (
              <Notice tone="error" onDismiss={dismissError}>
                {shownError}
              </Notice>
            )}
            {connecting ? (
              <ConnectingPanel state={connection.state} detail={connection.detail} onCancel={handleCancel} />
            ) : (
              <TokenPanel
                localToken={localToken}
                isLoadingToken={isLoadingToken}
                remoteToken={remoteToken}
                onRemoteTokenChange={setRemoteToken}
                onConnect={handleConnect}
              />
            )}
          </div>
        </Card>
      </main>
    </>
  );
}
