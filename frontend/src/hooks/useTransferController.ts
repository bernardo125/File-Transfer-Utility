import { useCallback, useMemo, useState } from 'react';
import { Disconnect, RespondIncomingTransfer, SelectFileNative, SendSelectedFile } from '../wailsjs/go/ui/App';
import { useSession } from '../context/SessionContext';
import { deriveTransferPhase } from '../lib/transferPhase';
import { errorMessage } from '../lib/wails';
import type { FileMetadata, TransferPhase } from '../types/transfer';
import type { PeerRole } from '../types/navigation';

/** The file currently on the screen, whichever side we are on. */
export interface ActiveFile {
  name: string;
  size: number;
  hash: string;
}

/**
 * Transfer-screen state machine: combines backend events (SessionContext) with local
 * intent (picked file, pending decision) and exposes the actions each phase allows.
 */
export function useTransferController(role: PeerRole) {
  const session = useSession();
  const { transferStatus, incomingTransfer, transferCompleted, transferError, transferProgress, resetTransferState, resetConnectionOutcome } =
    session;

  const [selectedFile, setSelectedFile] = useState<FileMetadata | null>(null);
  const [isPicking, setIsPicking] = useState(false);
  const [isResponding, setIsResponding] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);
  // The incoming offer is kept after the decision so its details stay visible during the transfer.
  const [lastOffer, setLastOffer] = useState(incomingTransfer);
  if (incomingTransfer && incomingTransfer !== lastOffer) setLastOffer(incomingTransfer);

  const phase: TransferPhase = deriveTransferPhase({
    role,
    status: transferStatus,
    hasSelectedFile: selectedFile !== null,
    hasIncoming: incomingTransfer !== null,
    hasCompleted: transferCompleted !== null,
    hasError: transferError !== null || localError !== null,
  });

  const file: ActiveFile | null = useMemo(() => {
    if (role === 'send') {
      return selectedFile ? { name: selectedFile.name, size: selectedFile.size, hash: selectedFile.hash } : null;
    }
    const offer = incomingTransfer ?? lastOffer;
    if (offer) return { name: transferCompleted?.fileName ?? offer.fileName, size: offer.fileSize, hash: offer.fileHash };
    return transferCompleted ? { name: transferCompleted.fileName, size: transferCompleted.fileSize, hash: '' } : null;
  }, [role, selectedFile, incomingTransfer, lastOffer, transferCompleted]);

  /** Clears the finished cycle so a new file can be offered or received. */
  const startOver = useCallback(() => {
    resetTransferState();
    setSelectedFile(null);
    setLocalError(null);
    setLastOffer(null);
  }, [resetTransferState]);

  const chooseFile = useCallback(async () => {
    setIsPicking(true);
    try {
      const picked = await SelectFileNative();
      if (!picked?.path) return; // dialog dismissed
      startOver();
      setSelectedFile(picked);
      SendSelectedFile(picked.path).catch((err: unknown) => {
        const message = errorMessage(err);
        // A decline is reported through the 'rejected' status, not as a failure.
        if (!/rejected by peer/i.test(message)) setLocalError(message);
      });
    } catch (err) {
      setLocalError(`Could not open the file: ${errorMessage(err)}`);
    } finally {
      setIsPicking(false);
    }
  }, [startOver]);

  const respond = useCallback(async (accept: boolean) => {
    const offerId = (incomingTransfer ?? lastOffer)?.offerId;
    if (!offerId) {
      setLocalError('This offer is no longer pending.');
      return;
    }
    setIsResponding(true);
    try {
      await RespondIncomingTransfer(offerId, accept);
    } catch (err) {
      setLocalError(errorMessage(err));
    } finally {
      setIsResponding(false);
    }
  }, [incomingTransfer, lastOffer]);

  const disconnect = useCallback(async () => {
    try {
      await Disconnect();
    } catch {
      // Leaving the session regardless; the backend tears the connection down on its side.
    }
    startOver();
    resetConnectionOutcome();
  }, [startOver, resetConnectionOutcome]);

  return {
    phase,
    file,
    progress: transferProgress,
    completed: transferCompleted,
    errorText: transferError?.message ?? localError,
    connection: session.connection,
    isPicking,
    isResponding,
    chooseFile,
    respond,
    startOver,
    disconnect,
  };
}

export type TransferController = ReturnType<typeof useTransferController>;
