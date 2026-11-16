import { useEffect, useState } from 'react';
import { LogOut } from 'lucide-react';
import { useTransferController } from '../hooks/useTransferController';
import { describePhase } from '../lib/transferPhase';
import { AppHeader } from '../components/layout/AppHeader';
import { ConnectionIndicator } from '../components/layout/ConnectionIndicator';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Notice } from '../components/ui/Notice';
import { FlightPath } from '../components/transfer/FlightPath';
import { TransferDetails } from '../components/transfer/TransferDetails';
import { TransferActions } from '../components/transfer/TransferActions';
import type { PeerRole } from '../types/navigation';

interface TransferScreenProps {
  role: PeerRole;
  onExit: () => void;
}

const CONFIRM_WINDOW_MS = 4000;

export function TransferScreen({ role, onExit }: TransferScreenProps) {
  const controller = useTransferController(role);
  const { phase, file, progress, completed, errorText, connection, disconnect } = controller;
  const [confirmingDisconnect, setConfirmingDisconnect] = useState(false);

  const busy = phase === 'transferring' || phase === 'verifying';
  const peerLost = connection.state === 'error' || connection.state === 'idle';
  const copy = describePhase(phase, role);
  const fraction = progress && progress.totalBytes > 0 ? Math.min(1, progress.bytesTransferred / progress.totalBytes) : null;

  useEffect(() => {
    if (!confirmingDisconnect) return;
    const id = window.setTimeout(() => setConfirmingDisconnect(false), CONFIRM_WINDOW_MS);
    return () => window.clearTimeout(id);
  }, [confirmingDisconnect]);

  const handleDisconnect = async () => {
    // Disconnecting mid-transfer aborts it, so ask for a second click first.
    if (busy && !peerLost && !confirmingDisconnect) {
      setConfirmingDisconnect(true);
      return;
    }
    await disconnect();
    onExit();
  };

  return (
    <>
      <AppHeader
        leading={
          <Button variant={confirmingDisconnect ? 'danger' : 'ghost'} size="sm" icon={LogOut} onClick={handleDisconnect}>
            {confirmingDisconnect ? 'Stop transfer & disconnect?' : 'Disconnect'}
          </Button>
        }
        center={<ConnectionIndicator state={connection.state} rttMs={connection.rttMs} />}
      />
      <main className="flex flex-1 flex-col items-center justify-center overflow-y-auto px-6 py-4">
        <div className="flex w-full max-w-2xl flex-col items-center gap-5">
          <FlightPath role={role} phase={phase} fraction={fraction} />

          <div className="text-center" aria-live="polite">
            <h1 className={`text-2xl font-extrabold tracking-tight ${phase === 'failed' ? 'text-danger' : 'text-fg'}`}>{copy.title}</h1>
            <p className="mt-1 text-sm text-muted">{copy.detail}</p>
          </div>

          {peerLost && (
            <div className="w-full">
              <Notice tone="error">
                {connection.detail || 'The connection to your peer was lost.'} Disconnect and pair again to continue.
              </Notice>
            </div>
          )}

          {file && (
            <Card key={file.name + file.size} className="anim-fade-up w-full p-5">
              <TransferDetails file={file} phase={phase} progress={progress} completed={completed} />
            </Card>
          )}

          {phase === 'failed' && errorText && (
            <div className="w-full">
              <Notice tone="error">{errorText}</Notice>
            </div>
          )}

          {!peerLost && <TransferActions role={role} controller={controller} />}
        </div>
      </main>
    </>
  );
}
