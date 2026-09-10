import React from 'react';
import { FileText, X, CheckCircle, FolderOpen } from 'lucide-react';
import { formatBytes, formatEta } from '../../utils/format';
import type { TransferProgress, TransferCompleted } from '../../types/transfer';

interface ProgressPanelProps {
  fileName: string;
  fileSize: number;
  progress: TransferProgress | null;
  completed: TransferCompleted | null;
  isReceiver: boolean;
  onAbort: () => void;
  onOpenFolder?: (path: string) => void;
}

export const ProgressPanel: React.FC<ProgressPanelProps> = ({
  fileName,
  fileSize,
  progress,
  completed,
  isReceiver,
  onAbort,
  onOpenFolder,
}) => {
  const percent = completed ? 100 : Math.min(100, Math.max(0, Math.round(progress?.percentage || 0)));
  const transferred = completed ? fileSize : (progress?.bytesTransferred || 0);
  const speed = completed ? completed.speedMBs : (progress?.speedMBs || 0);
  const eta = completed ? 0 : (progress?.etaSeconds || 0);

  return (
    <div className="w-full bg-[var(--bg-panel)] border border-[var(--border-app)] rounded-[4px] p-4 shadow-sm space-y-3">
      {/* File Info Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2.5 truncate">
          <div className="w-8 h-8 rounded-[3px] bg-[var(--bg-inset)] border border-[var(--border-subtle)] flex items-center justify-center text-[#2fa340] flex-shrink-0">
            <FileText className="w-4 h-4" />
          </div>
          <div className="truncate">
            <p className="text-[14px] font-bold text-[var(--text-main)] truncate">{fileName}</p>
            <p className="text-[11px] text-[var(--text-dim)]">
              {formatBytes(transferred)} / {formatBytes(fileSize)}
            </p>
          </div>
        </div>

        {/* State Badge */}
        {completed ? (
          <span className="text-emerald-400 text-[14px] flex items-center gap-1 font-semibold">
            <CheckCircle className="w-4 h-4" /> Done
          </span>
        ) : (
          <span className="text-[11px] font-bold text-[#2fa340]">{percent}%</span>
        )}
      </div>

      {/* µTorrent-style Fluid Progress Bar */}
      <div className="w-full h-3 bg-[var(--bg-inset)] border border-[var(--border-app)] rounded-[2px] overflow-hidden p-[1px]">
        <div
          className="h-full bg-[#2fa340] transition-all duration-200 rounded-[1px]"
          style={{ width: `${percent}%` }}
        />
      </div>

      {/* Speed, ETA & Actions Footer */}
      <div className="flex items-center justify-between text-[11px] pt-1 border-t border-[var(--border-subtle)] text-[var(--text-muted)]">
        <div className="flex items-center gap-4">
          <span>Vel: <strong className="text-[var(--text-main)]">{speed.toFixed(1)} MB/s</strong></span>
          <span>ETA: <strong className="text-[var(--text-main)]">{completed ? '00:00:00' : formatEta(eta)}</strong></span>
        </div>

        {completed ? (
          isReceiver && completed.destPath && onOpenFolder && (
            <button
              type="button"
              onClick={() => onOpenFolder(completed.destPath)}
              className="h-7 px-3 rounded-[2px] bg-[#2fa340] hover:bg-[#288d37] text-white text-[11px] font-semibold flex items-center gap-1 cursor-pointer"
            >
              <FolderOpen className="w-3.5 h-3.5" />
              <span>Open Directory</span>
            </button>
          )
        ) : (
          <button
            type="button"
            onClick={onAbort}
            className="h-6 px-2.5 rounded-[2px] bg-[var(--bg-inset)] hover:text-rose-400 hover:border-rose-800/60 border border-[var(--border-app)] text-[10px] uppercase flex items-center gap-1 cursor-pointer transition-colors"
          >
            <X className="w-3 h-3" />
            <span>Cancel</span>
          </button>
        )}
      </div>
    </div>
  );
};
