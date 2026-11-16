import { FileIcon, Fingerprint, ShieldCheck } from 'lucide-react';
import type { TransferCompleted, TransferPhase, TransferProgress } from '../../types/transfer';
import type { ActiveFile } from '../../hooks/useTransferController';
import { fileExtension, formatBytes, formatDuration, formatSpeed, shortHash } from '../../utils/format';

interface TransferDetailsProps {
  file: ActiveFile;
  phase: TransferPhase;
  progress: TransferProgress | null;
  completed: TransferCompleted | null;
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <dt className="text-[11px] font-medium uppercase tracking-wider text-dim">{label}</dt>
      <dd className="truncate text-sm font-semibold tabular-nums text-fg">{value}</dd>
    </div>
  );
}

function integrityLabel(phase: TransferPhase, completed: TransferCompleted | null): string {
  if (completed?.hashVerified) return 'Checksum verified';
  if (phase === 'verifying') return 'Verifying checksum…';
  if (phase === 'failed') return 'Not verified';
  return 'Verified on arrival';
}

/** File identity, honest progress figures and SHA-256 status for the active transfer. */
export function TransferDetails({ file, phase, progress, completed }: TransferDetailsProps) {
  const ext = fileExtension(file.name);
  const showBar = phase === 'transferring' || phase === 'verifying' || phase === 'completed' || (phase === 'failed' && progress);
  const percent = completed ? 100 : Math.min(100, Math.max(0, progress?.percentage ?? 0));
  const hasLiveRate = phase === 'transferring' && progress !== null && progress.speedMBs > 0;
  const verified = completed?.hashVerified === true;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <div className="relative flex size-12 shrink-0 items-center justify-center rounded-xl bg-accent-soft text-accent">
          <FileIcon className="size-6" aria-hidden />
          {ext && (
            <span className="absolute -bottom-1.5 rounded bg-surface px-1 text-[9px] font-bold tracking-wide text-accent ring-1 ring-line">
              {ext.slice(0, 4)}
            </span>
          )}
        </div>
        <div className="min-w-0 flex-1">
          <p className="truncate text-base font-bold text-fg" title={file.name}>{file.name}</p>
          <p className="text-sm text-muted tabular-nums">{formatBytes(file.size)}</p>
        </div>
        {showBar && (
          <span className={`text-2xl font-extrabold tabular-nums ${phase === 'failed' ? 'text-danger' : 'text-accent'}`}>
            {Math.floor(percent)}%
          </span>
        )}
      </div>

      {showBar && (
        <div
          role="progressbar"
          aria-label="Transfer progress"
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={Math.floor(percent)}
          className="h-2 overflow-hidden rounded-full bg-inset"
        >
          <div
            className={`h-full rounded-full transition-[width] duration-200 ease-linear ${phase === 'failed' ? 'bg-danger' : 'bg-accent'}`}
            style={{ width: `${percent}%` }}
          />
        </div>
      )}

      {(phase === 'transferring' || phase === 'verifying' || completed) && (
        <dl className="grid grid-cols-3 gap-4">
          {completed ? (
            <>
              <Metric label="Size" value={formatBytes(completed.fileSize)} />
              <Metric label="Time" value={formatDuration(completed.durationSec)} />
              <Metric label="Avg. speed" value={formatSpeed(completed.speedMBs)} />
            </>
          ) : (
            <>
              <Metric
                label="Transferred"
                value={progress ? `${formatBytes(progress.bytesTransferred)} of ${formatBytes(progress.totalBytes || file.size)}` : '—'}
              />
              <Metric label="Speed" value={hasLiveRate ? formatSpeed(progress.speedMBs) : '—'} />
              <Metric
                label="Time left"
                value={phase === 'verifying' ? 'Checking…' : hasLiveRate && progress.etaSeconds > 0 ? formatDuration(progress.etaSeconds) : '—'}
              />
            </>
          )}
        </dl>
      )}

      {file.hash && (
        <div className="flex items-center gap-2 rounded-lg border border-line bg-inset px-3 py-2 text-xs">
          {verified ? <ShieldCheck className="size-4 shrink-0 text-accent" aria-hidden /> : <Fingerprint className="size-4 shrink-0 text-dim" aria-hidden />}
          <span className="font-semibold text-muted">SHA-256</span>
          <span className="selectable truncate font-mono text-dim" title={file.hash}>{shortHash(file.hash, 10)}</span>
          <span className={`ml-auto shrink-0 font-semibold ${verified ? 'text-accent' : phase === 'failed' ? 'text-danger' : 'text-muted'}`}>
            {integrityLabel(phase, completed)}
          </span>
        </div>
      )}
    </div>
  );
}
