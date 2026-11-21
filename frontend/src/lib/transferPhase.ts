import type { TransferPhase, TransferStatus } from '../types/transfer';
import type { PeerRole } from '../types/navigation';

interface PhaseInput {
  role: PeerRole;
  status: TransferStatus;
  hasSelectedFile: boolean;
  hasIncoming: boolean;
  hasCompleted: boolean;
  hasError: boolean;
}

/** Maps backend transfer status (plus local sender intent) to a single UI phase. */
export function deriveTransferPhase({ role, status, hasSelectedFile, hasIncoming, hasCompleted, hasError }: PhaseInput): TransferPhase {
  // On the sending side every phase belongs to the file this peer picked; events without one
  // (e.g. an offer arriving from the other peer) must not move the sender's screen.
  if (role === 'send' && !hasSelectedFile) return 'idle';
  if (hasCompleted || status === 'completed') return 'completed';
  if (hasError || status === 'error') return 'failed';
  if (status === 'rejected') return 'declined';
  if (status === 'verifying') return 'verifying';
  if (status === 'transferring') return 'transferring';

  if (role === 'send') {
    if (status === 'waiting_approval' || status === 'requesting') return 'awaiting';
    return 'preparing';
  }
  return hasIncoming || status === 'waiting_approval' ? 'deciding' : 'idle';
}

export interface PhaseCopy {
  title: string;
  detail: string;
}

/** Human-readable headline for each phase, from the point of view of this peer. */
export function describePhase(phase: TransferPhase, role: PeerRole): PhaseCopy {
  const sending = role === 'send';
  switch (phase) {
    case 'idle':
      return sending
        ? { title: 'Ready to throw', detail: 'Choose a file to offer to your peer.' }
        : { title: 'Ready to catch', detail: 'Waiting for your peer to choose a file.' };
    case 'preparing':
      return { title: 'Preparing file', detail: 'Computing the SHA-256 checksum before sending the offer.' };
    case 'awaiting':
      return { title: 'Waiting for your peer', detail: 'The receiver needs to accept the file before it is sent.' };
    case 'deciding':
      return { title: 'Incoming file', detail: 'Your peer wants to send you a file. Accept to start the transfer.' };
    case 'transferring':
      return sending
        ? { title: 'Sending', detail: 'Streaming the file directly to your peer over QUIC.' }
        : { title: 'Receiving', detail: 'The file is streaming directly from your peer and being checked as it arrives.' };
    case 'verifying':
      return { title: 'Verifying integrity', detail: 'All data sent. Waiting for the receiver to confirm the SHA-256 checksum.' };
    case 'completed':
      return sending
        ? { title: 'Delivered', detail: 'Your peer received the file and confirmed its checksum.' }
        : { title: 'File received', detail: 'Saved to your download folder with a matching SHA-256 checksum.' };
    case 'declined':
      return sending
        ? { title: 'Transfer declined', detail: 'The receiver declined the file, or did not respond in time.' }
        : { title: 'Transfer declined', detail: 'The offer was declined. Nothing was saved.' };
    case 'failed':
      return { title: 'Transfer failed', detail: 'The transfer stopped before it finished.' };
  }
}

/** One- or two-word status for compact list rows. */
export function shortStatus(phase: TransferPhase, role: PeerRole): string {
  switch (phase) {
    case 'idle': return 'Ready';
    case 'preparing': return 'Preparing';
    case 'awaiting': return 'Awaiting approval';
    case 'deciding': return 'Awaiting you';
    case 'transferring': return role === 'send' ? 'Sending' : 'Receiving';
    case 'verifying': return 'Verifying';
    case 'completed': return role === 'send' ? 'Delivered' : 'Received';
    case 'declined': return 'Declined';
    case 'failed': return 'Failed';
  }
}
