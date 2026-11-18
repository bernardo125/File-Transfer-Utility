import { useEffect, useReducer, useCallback } from 'react';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { WailsEvent } from '../types/transfer';
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

export interface SessionEvents {
  connection: ConnectionStateEvent;
  connectionSuccess: ConnectionSuccessEvent | null;
  connectionFailed: ConnectionFailedEvent | null;
  incomingTransfer: IncomingTransfer | null;
  transferStatus: TransferStatus;
  transferProgress: TransferProgress | null;
  transferCompleted: TransferCompleted | null;
  transferError: TransferError | null;
}

type Action =
  | { type: 'connection'; payload: ConnectionStateEvent }
  | { type: 'connectionSuccess'; payload: ConnectionSuccessEvent }
  | { type: 'connectionFailed'; payload: ConnectionFailedEvent }
  | { type: 'incoming'; payload: IncomingTransfer }
  | { type: 'status'; payload: TransferStatus }
  | { type: 'progress'; payload: TransferProgress }
  | { type: 'completed'; payload: TransferCompleted }
  | { type: 'error'; payload: TransferError }
  | { type: 'resetTransfer' }
  | { type: 'resetConnectionOutcome' };

const idleTransfer = {
  incomingTransfer: null,
  transferStatus: 'idle' as TransferStatus,
  transferProgress: null,
  transferCompleted: null,
  transferError: null,
};

const initialState: SessionEvents = {
  connection: { state: 'idle', detail: '' },
  connectionSuccess: null,
  connectionFailed: null,
  ...idleTransfer,
};

function reducer(state: SessionEvents, action: Action): SessionEvents {
  switch (action.type) {
    case 'connection':
      return { ...state, connection: action.payload };
    case 'connectionSuccess':
      return { ...state, connectionSuccess: action.payload, connectionFailed: null };
    case 'connectionFailed':
      return { ...state, connectionFailed: action.payload };
    case 'incoming':
      // A new offer starts a fresh transfer cycle on the receiving side.
      return { ...state, ...idleTransfer, incomingTransfer: action.payload, transferStatus: 'waiting_approval' };
    case 'status': {
      const status = action.payload;
      if (status === 'requesting') {
        return { ...state, ...idleTransfer, transferStatus: status };
      }
      if (status === 'transferring') {
        return { ...state, transferStatus: status, transferError: null, transferCompleted: null };
      }
      return { ...state, transferStatus: status };
    }
    case 'progress':
      return { ...state, transferProgress: action.payload };
    case 'completed':
      return {
        ...state,
        transferStatus: 'completed',
        transferCompleted: action.payload,
        transferProgress: {
          bytesTransferred: action.payload.fileSize,
          totalBytes: action.payload.fileSize,
          percentage: 100,
          speedMBs: action.payload.speedMBs,
          etaSeconds: 0,
        },
      };
    case 'error':
      return { ...state, transferStatus: 'error', transferError: action.payload };
    case 'resetTransfer':
      return { ...state, ...idleTransfer };
    case 'resetConnectionOutcome':
      return { ...state, connectionSuccess: null, connectionFailed: null };
  }
}

function isWailsRuntimeAvailable(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof (window as unknown as { runtime?: { EventsOnMultiple?: unknown } }).runtime?.EventsOnMultiple === 'function'
  );
}

/**
 * Subscribes to every backend event exactly once. Mount it a single time (see SessionProvider):
 * listeners are removed with the unsubscribe functions returned by EventsOn, so other
 * subscribers of the same event are never affected.
 */
export function useWailsEvents() {
  const [state, dispatch] = useReducer(reducer, initialState);

  useEffect(() => {
    let unsubscribers: Array<() => void> = [];
    let pollTimer: number | undefined;

    const bind = (): boolean => {
      if (!isWailsRuntimeAvailable()) return false;
      unsubscribers = [
        EventsOn(WailsEvent.ConnectionState, (d: ConnectionStateEvent) => dispatch({ type: 'connection', payload: d })),
        EventsOn(WailsEvent.ConnectionSuccess, (d: ConnectionSuccessEvent) => dispatch({ type: 'connectionSuccess', payload: d })),
        EventsOn(WailsEvent.ConnectionFailed, (d: ConnectionFailedEvent) => dispatch({ type: 'connectionFailed', payload: d })),
        EventsOn(WailsEvent.TransferIncoming, (d: IncomingTransfer) => dispatch({ type: 'incoming', payload: d })),
        EventsOn(WailsEvent.TransferStatus, (d: TransferStatus) => dispatch({ type: 'status', payload: d })),
        // The backend already throttles progress (~150 ms), so no extra frontend throttling is needed.
        EventsOn(WailsEvent.TransferProgress, (d: TransferProgress) => dispatch({ type: 'progress', payload: d })),
        EventsOn(WailsEvent.TransferCompleted, (d: TransferCompleted) => dispatch({ type: 'completed', payload: d })),
        EventsOn(WailsEvent.TransferError, (d: TransferError) => dispatch({ type: 'error', payload: d })),
      ];
      return true;
    };

    if (!bind()) {
      // The Wails bridge can be injected slightly after the first render.
      let attempts = 0;
      pollTimer = window.setInterval(() => {
        attempts++;
        if (bind() || attempts > 50) window.clearInterval(pollTimer);
      }, 150);
    }

    return () => {
      if (pollTimer) window.clearInterval(pollTimer);
      unsubscribers.forEach((off) => off());
    };
  }, []);

  const resetTransferState = useCallback(() => dispatch({ type: 'resetTransfer' }), []);
  const resetConnectionOutcome = useCallback(() => dispatch({ type: 'resetConnectionOutcome' }), []);

  return { ...state, resetTransferState, resetConnectionOutcome };
}

export type SessionValue = ReturnType<typeof useWailsEvents>;
