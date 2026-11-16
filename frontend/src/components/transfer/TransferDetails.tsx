import { FileIcon, ShieldCheck } from 'lucide-react';
import type { ReactNode } from 'react';
import type { TransferCompleted, TransferPhase, TransferProgress } from '../../types/transfer';
import type { PeerRole } from '../../types/navigation';
import type { ActiveFile } from '../../hooks/useTransferController';
import { shortStatus } from '../../lib/transferPhase';
import { fileExtension, formatBytes, formatDuration, formatSpeed, shortHash } from '../../utils/format';
import { Panel } from '../ui/Panel';
import { ProgressBar } from './ProgressBar';

interface TransferDetailsProps {
  role: PeerRole;
  file: ActiveFile;
  phase: TransferPhase;
  progress: TransferProgress | null;
  completed: TransferCompleted | null;
}

const COLUMNS = 'grid grid-cols-[minmax(0,1fr)_72px_128px_84px_72px] items-center gap-3 px-3';

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex min-w-0 gap-2">
      <dt className="w-24 shrink-0 text-dim">{label}</dt>
      <dd className="min-w-0 truncate text-fg">{children}</dd>
    </div>
  );
}

/** The active transfer as a download-manager row: list columns, progress bar and a General strip. */
export function TransferDetails({ role, file, phase, progress, completed }: TransferDetailsProps) {
  const ext = fileExtension(file.name);
  const hasBytes = phase === 'transferring' || phase === 'verifying' || phase === 'completed' || (phase === 'failed' && progress !== null);
  const percent = completed ? 100 : hasBytes ? progress?.percentage ?? 0 : null;
  const live = phase === 'transferring' && progress !== null && progress.speedMBs > 0;
  const verified = completed?.hashVerified === true;
  const statusTone = phase === 'failed' || phase === 'declined' ? 'text-danger' : phase === 'completed' ? 'text-accent' : 'text-fg';

  return (
    <Panel>
      <div className={`${COLUMNS} rounded-t-md border-b border-line bg-inset py-1.5 text-[11px] font-semibold uppercase tracking-wider text-muted`}>
        <span>Name</span>
        <span>Size</span>
        <span>Status</span>
        <span>Speed</span>
        <span>Time left</span>
      </div>

      <div className={`${COLUMNS} py-2.5 text-xs tabular-nums`}>
        <span className="flex min-w-0 items-center gap-2">
          <FileIcon className="size-4 shrink-0 text-accent" aria-hidden />
          <span className="truncate font-semibold text-fg" title={file.name}>{file.name}</span>
          {ext && <span className="shrink-0 rounded-sm border border-line px-1 text-[10px] font-semibold text-muted">{ext.slice(0, 5)}</span>}
        </span>
        <span className="text-fg">{formatBytes(file.size)}</span>
        <span className={`truncate font-semibold ${statusTone}`}>{shortStatus(phase, role)}</span>
        <span className="text-fg">{live ? formatSpeed(progress.speedMBs) : '—'}</span>
        <span className="text-fg">{live && progress.etaSeconds > 0 ? formatDuration(progress.etaSeconds) : '—'}</span>
      </div>

      <div className="px-3 pb-3">
        <ProgressBar percent={percent} idleLabel="Not started" failed={phase === 'failed'} />
      </div>

      <dl className="grid gap-x-6 gap-y-1 rounded-b-md border-t border-line bg-inset px-3 py-2 text-[11px] sm:grid-cols-2">
        <Field label="Transferred">
          {progress && hasBytes ? `${formatBytes(progress.bytesTransferred)} of ${formatBytes(progress.totalBytes || file.size)}` : '—'}
        </Field>
        <Field label={completed ? 'Avg. speed' : 'Direction'}>
          {completed ? `${formatSpeed(completed.speedMBs)} in ${formatDuration(completed.durationSec)}` : role === 'send' ? 'You → Peer' : 'Peer → You'}
        </Field>
        <Field label="SHA-256">
          {file.hash ? (
            <span className="inline-flex items-center gap-1.5">
              <span className="selectable" title={file.hash}>{shortHash(file.hash, 8)}</span>
              {verified ? (
                <span className="anim-pop inline-flex items-center gap-1 font-semibold text-accent"><ShieldCheck className="size-3" aria-hidden />verified</span>
              ) : phase === 'verifying' ? (
                <span className="text-muted">checking…</span>
              ) : null}
            </span>
          ) : (
            '—'
          )}
        </Field>
        {role === 'receive' && completed ? <Field label="Saved to"><span className="selectable" title={completed.destPath}>{completed.destPath}</span></Field> : <span />}
      </dl>
    </Panel>
  );
}
