// Domain types mirroring the DTOs in pkg/ui/events.go.

/** Event names emitted by the Go backend (pkg/ui/events.go). */
export const WailsEvent = {
  ConnectionState: 'p2p:connection-state',
  ConnectionSuccess: 'connection:success',
  ConnectionFailed: 'connection:failed',
  TransferIncoming: 'transfer:incoming',
  TransferProgress: 'transfer:progress',
  TransferStatus: 'transfer:status',
  TransferCompleted: 'transfer:completed',
  TransferError: 'transfer:error',
} as const;

export type ConnectionState =
  | 'idle'
  | 'discovery'
  | 'ready'
  | 'hole_punching'
  | 'upgrading'
  | 'connected'
  | 'error';

export interface ConnectionStateEvent {
  state: ConnectionState;
  detail: string;
  rttMs?: number;
}

export interface ConnectionSuccessEvent {
  addr: string;
  rttMs: number;
}

export interface ConnectionFailedEvent {
  reason: string;
}


export interface FileMetadata {
  path: string;
  name: string;
  size: number;
  hash: string;
}

export interface IncomingTransfer {
  /** Identifier the backend expects back with the user's decision. */
  offerId: string;
  fileName: string;
  fileSize: number;
  fileHash: string;
}

export type TransferStatus =
  | 'idle'
  | 'requesting'
  | 'waiting_approval'
  | 'transferring'
  | 'verifying'
  | 'completed'
  | 'rejected'
  | 'error';

export interface TransferProgress {
  bytesTransferred: number;
  totalBytes: number;
  percentage: number;
  speedMBs: number;
  etaSeconds: number;
}

export interface TransferCompleted {
  fileName: string;
  destPath: string;
  fileSize: number;
  durationSec: number;
  speedMBs: number;
  hashVerified: boolean;
}

export interface TransferError {
  code: string;
  message: string;
}

/**
 * What the transfer screen is showing, derived from backend events plus local intent.
 * See deriveTransferPhase in lib/transferPhase.ts.
 */
export type TransferPhase =
  | 'idle'          // sender: pick a file · receiver: waiting for an offer
  | 'preparing'     // sender: file picked, computing SHA-256 before the offer is sent
  | 'awaiting'      // offer sent, waiting for the receiver's decision
  | 'deciding'      // receiver: offer arrived, user must accept or decline
  | 'transferring'
  | 'verifying'     // sender: payload sent, waiting for receiver's integrity confirmation
  | 'completed'
  | 'declined'
  | 'failed';
