// Strict TypeScript types for File Transfer Utility UI
// Adheres strictly to: NO 'any' types allowed.

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
