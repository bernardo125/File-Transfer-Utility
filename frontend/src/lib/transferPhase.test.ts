import { describe, expect, it } from 'vitest';
import { deriveTransferPhase, describePhase, shortStatus } from './transferPhase';
import type { TransferPhase, TransferStatus } from '../types/transfer';

type Input = Parameters<typeof deriveTransferPhase>[0];
const base: Input = { role: 'receive', status: 'idle', hasSelectedFile: false, hasIncoming: false, hasCompleted: false, hasError: false };
const phase = (over: Partial<Input>) => deriveTransferPhase({ ...base, ...over });

// The transfer screen is a pure function of backend status + local intent. A wrong row here shows
// the user the wrong screen (e.g. an "Accept" button during a transfer), so every rule is pinned.
describe('deriveTransferPhase', () => {
  it.each<[string, Partial<Input>, TransferPhase]>([
    ['receiver idle waits for an offer', {}, 'idle'],
    ['receiver sees an incoming offer', { hasIncoming: true }, 'deciding'],
    ['receiver waiting_approval status also means deciding', { status: 'waiting_approval' }, 'deciding'],
    ['receiver transferring', { status: 'transferring', hasIncoming: true }, 'transferring'],
    ['receiver verifying', { status: 'verifying' }, 'verifying'],
    ['receiver declined', { status: 'rejected', hasIncoming: true }, 'declined'],
    ['receiver completed', { status: 'completed', hasCompleted: true }, 'completed'],
    ['sender with no file ignores everything else (foreign offers must not move its screen)', { role: 'send', hasIncoming: true, status: 'waiting_approval' }, 'idle'],
    ['sender with a file but no status is preparing (hashing)', { role: 'send', hasSelectedFile: true }, 'preparing'],
    ['sender requesting', { role: 'send', hasSelectedFile: true, status: 'requesting' }, 'awaiting'],
    ['sender waiting for the receiver', { role: 'send', hasSelectedFile: true, status: 'waiting_approval' }, 'awaiting'],
    ['sender transferring', { role: 'send', hasSelectedFile: true, status: 'transferring' }, 'transferring'],
    ['sender declined', { role: 'send', hasSelectedFile: true, status: 'rejected' }, 'declined'],
  ])('%s', (_name, over, want) => {
    expect(phase(over)).toBe(want);
  });

  it('an error beats progress states, and completion beats an error', () => {
    expect(phase({ status: 'transferring', hasError: true })).toBe('failed');
    expect(phase({ status: 'error' })).toBe('failed');
    expect(phase({ status: 'rejected', hasError: true })).toBe('failed');
    expect(phase({ status: 'error', hasCompleted: true })).toBe('completed');
  });

  it('every backend status maps to a phase for both roles (no status is unhandled)', () => {
    const statuses: TransferStatus[] = ['idle', 'requesting', 'waiting_approval', 'transferring', 'verifying', 'completed', 'rejected', 'error'];
    for (const role of ['send', 'receive'] as const) {
      for (const status of statuses) {
        expect(phase({ role, status, hasSelectedFile: true }), `${role}/${status}`).toBeTruthy();
      }
    }
  });
});

describe('phase copy', () => {
  const phases: TransferPhase[] = ['idle', 'preparing', 'awaiting', 'deciding', 'transferring', 'verifying', 'completed', 'declined', 'failed'];

  it('every phase has a headline, detail and short status for both roles', () => {
    for (const role of ['send', 'receive'] as const) {
      for (const p of phases) {
        const copy = describePhase(p, role);
        expect(copy.title, `${role}/${p} title`).not.toBe('');
        expect(copy.detail, `${role}/${p} detail`).not.toBe('');
        expect(shortStatus(p, role), `${role}/${p} status`).not.toBe('');
      }
    }
  });

  it('the wording follows the role: sender "Delivered", receiver "File received"', () => {
    expect(describePhase('completed', 'send').title).toBe('Delivered');
    expect(describePhase('completed', 'receive').title).toBe('File received');
    expect(shortStatus('transferring', 'send')).toBe('Sending');
    expect(shortStatus('transferring', 'receive')).toBe('Receiving');
  });
});
