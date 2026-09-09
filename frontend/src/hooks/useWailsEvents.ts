import { useEffect, useState, useCallback, useRef } from 'react';
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime';
import type {
  ConnectionStateEvent,
  ConnectionSuccessEvent,
  ConnectionFailedEvent,
  IncomingTransfer,
  TransferStatus,
  TransferProgress,
  TransferCompleted,
  TransferError,
} from '../types/transfer';

// Check if Wails runtime bridge is injected and available on window
function isWailsRuntimeAvailable(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof (window as unknown as { runtime?: { EventsOnMultiple?: unknown } }).runtime?.EventsOnMultiple === 'function'
  );
}

export function useWailsEvents() {
  const [connection, setConnection] = useState<ConnectionStateEvent>({
    state: 'idle',
    detail: 'Aguardando inicialização...',
  });

  const [connectionSuccess, setConnectionSuccess] = useState<ConnectionSuccessEvent | null>(null);
  const [connectionFailed, setConnectionFailed] = useState<ConnectionFailedEvent | null>(null);
  const [incomingTransfer, setIncomingTransfer] = useState<IncomingTransfer | null>(null);
  const [transferStatus, setTransferStatus] = useState<TransferStatus>('idle');
  const [transferProgress, setTransferProgress] = useState<TransferProgress | null>(null);
  const [transferCompleted, setTransferCompleted] = useState<TransferCompleted | null>(null);
  const [transferError, setTransferError] = useState<TransferError | null>(null);


  // Throttling ref to prevent excessive re-renders during high-frequency telemetry
  const lastProgressUpdateRef = useRef<number>(0);

  useEffect(() => {
    let cleanupFn: (() => void) | undefined;
    let pollTimer: number | undefined;

    const setupListeners = () => {
      if (!isWailsRuntimeAvailable()) return false;

      try {
        // 1. Connection state
        EventsOn('p2p:connection-state', (data: ConnectionStateEvent) => {
          setConnection(data);
        });

        // 2. Incoming transfer request from peer
        EventsOn('transfer:incoming', (data: IncomingTransfer) => {
          setIncomingTransfer(data);
          setTransferStatus('waiting_approval');
        });

        // 3. High-level transfer status
        EventsOn('transfer:status', (status: TransferStatus) => {
          setTransferStatus(status);
          if (status === 'transferring') {
            setTransferError(null);
            setTransferCompleted(null);
          }
        });

        // 4. Fine-grained telemetry (throttled to max 1 update per 100ms)
        EventsOn('transfer:progress', (progress: TransferProgress) => {
          const now = performance.now();
          if (now - lastProgressUpdateRef.current > 100 || progress.percentage === 100) {
            lastProgressUpdateRef.current = now;
            setTransferProgress(progress);
          }
        });

        // 5. Transfer completed
        EventsOn('transfer:completed', (completed: TransferCompleted) => {
          setTransferCompleted(completed);
          setTransferStatus('completed');
          setIncomingTransfer(null);
          setTransferProgress({
            bytesTransferred: completed.fileSize,
            totalBytes: completed.fileSize,
            percentage: 100,
            speedMBs: completed.speedMBs,
            etaSeconds: 0,
          });
        });

        // 6. Transfer error
        EventsOn('transfer:error', (err: TransferError) => {
          setTransferError(err);
          setTransferStatus('error');
        });

        // 7. Connection success (Pairing handshake complete)
        EventsOn('connection:success', (data: ConnectionSuccessEvent) => {
          setConnectionSuccess(data);
          setConnectionFailed(null);
        });

        // 8. Connection failed (Pairing failed or aborted)
        EventsOn('connection:failed', (data: ConnectionFailedEvent) => {
          setConnectionFailed(data);
        });

        cleanupFn = () => {
          if (isWailsRuntimeAvailable()) {
            try {
              EventsOff('p2p:connection-state');
              EventsOff('connection:success');
              EventsOff('connection:failed');
              EventsOff('transfer:incoming');
              EventsOff('transfer:status');
              EventsOff('transfer:progress');
              EventsOff('transfer:completed');
              EventsOff('transfer:error');
            } catch {
              // ignore unbind errors on exit
            }
          }
        };

        return true;
      } catch (err) {
        console.warn('Failed to attach Wails event listeners:', err);
        return false;
      }
    };

    // If Wails runtime is already present, bind immediately
    const bound = setupListeners();

    // If not yet available (e.g. initial DOM mount before Wails script evaluation),
    // poll gently until Wails runtime appears
    if (!bound) {
      let attempts = 0;
      pollTimer = window.setInterval(() => {
        attempts++;
        if (setupListeners() || attempts > 50) {
          clearInterval(pollTimer);
        }
      }, 150);
    }

    return () => {
      if (pollTimer) clearInterval(pollTimer);
      if (cleanupFn) cleanupFn();
    };
  }, []);

  const clearIncomingTransfer = useCallback(() => {
    setIncomingTransfer(null);
  }, []);

  const resetTransferState = useCallback(() => {
    setTransferStatus('idle');
    setTransferProgress(null);
    setTransferCompleted(null);
    setTransferError(null);
    setIncomingTransfer(null);
    setConnectionSuccess(null);
    setConnectionFailed(null);
  }, []);

  return {
    connection,
    connectionSuccess,
    connectionFailed,
    incomingTransfer,
    transferStatus,
    transferProgress,
    transferCompleted,
    transferError,
    clearIncomingTransfer,
    resetTransferState,
  };
}

