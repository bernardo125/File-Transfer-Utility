import React, { useState, useEffect } from 'react';
import type { TransferProgress, TransferStatus } from '../types/transfer';
import { Gauge, Clock, HardDrive, CheckCircle2 } from 'lucide-react';

interface TelemetryProgressProps {
  status: TransferStatus;
  progress: TransferProgress | null;
  fileName?: string;
  totalSize?: number;
}

export const TelemetryProgress: React.FC<TelemetryProgressProps> = ({
  status,
  progress,
  fileName,
  totalSize,
}) => {
  const [elapsed, setElapsed] = useState<number>(0);

  useEffect(() => {
    let timer: number | undefined;
    if (status === 'transferring' || status === 'verifying') {
      const start = Date.now();
      timer = window.setInterval(() => {
        setElapsed(Math.floor((Date.now() - start) / 1000));
      }, 500);
    } else {
      setElapsed(0);
    }
    return () => {
      if (timer) clearInterval(timer);
    };
  }, [status]);

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  const hasFineGrained =
    progress !== null &&
    progress.percentage > 0 &&
    progress.percentage < 100 &&
    progress.totalBytes > 0;

  const percentage = progress ? Math.round(progress.percentage) : 0;
  const speed = progress?.speedMBs ?? 0;
  const eta = progress?.etaSeconds ?? 0;

  return (
    <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 shadow-lg space-y-4">
      <div className="flex items-center justify-between">
        <div className="min-w-0 flex-1">
          <h4 className="text-sm font-semibold text-slate-200 truncate">
            {fileName || 'Transferência P2P'}
          </h4>
          <span className="text-xs text-slate-400 capitalize">
            {status === 'transferring'
              ? 'Transferindo dados via QUIC...'
              : status === 'verifying'
              ? 'A verificar integridade SHA-256...'
              : status === 'completed'
              ? 'Concluído com sucesso'
              : status}
          </span>
        </div>

        <div className="text-right">
          {hasFineGrained ? (
            <span className="text-xl font-bold font-mono text-cyan-400">{percentage}%</span>
          ) : status === 'completed' ? (
            <span className="inline-flex items-center gap-1.5 text-sm font-medium text-emerald-400">
              <CheckCircle2 className="w-4 h-4" /> 100%
            </span>
          ) : (
            <span className="text-xs font-mono text-cyan-400 animate-pulse">Ativo</span>
          )}
        </div>
      </div>

      {/* Progress bar */}
      <div className="w-full h-2.5 bg-slate-800/90 rounded-full overflow-hidden relative">
        {hasFineGrained ? (
          <div
            className="h-full bg-gradient-to-r from-cyan-500 to-blue-500 transition-all duration-200 ease-out"
            style={{ width: `${Math.min(100, percentage)}%` }}
          />
        ) : status === 'completed' ? (
          <div className="h-full bg-emerald-500 w-full" />
        ) : status === 'transferring' || status === 'verifying' ? (
          <div className="h-full bg-gradient-to-r from-cyan-500 via-blue-500 to-cyan-500 w-full animate-pulse" />
        ) : null}
      </div>

      {/* Telemetry Metrics Grid */}
      <div className="grid grid-cols-3 gap-3 pt-1">
        {/* Speed */}
        <div className="bg-slate-950/60 border border-slate-800/80 rounded-xl p-3 flex items-center gap-2.5">
          <div className="p-2 rounded-lg bg-cyan-500/10 text-cyan-400">
            <Gauge className="w-4 h-4" />
          </div>
          <div>
            <span className="text-[11px] text-slate-500 block uppercase font-medium">Velocidade</span>
            <span className="text-xs font-mono font-semibold text-slate-200">
              {speed > 0 ? `${speed.toFixed(2)} MB/s` : '-- MB/s'}
            </span>
          </div>
        </div>

        {/* Elapsed / ETA */}
        <div className="bg-slate-950/60 border border-slate-800/80 rounded-xl p-3 flex items-center gap-2.5">
          <div className="p-2 rounded-lg bg-blue-500/10 text-blue-400">
            <Clock className="w-4 h-4" />
          </div>
          <div>
            <span className="text-[11px] text-slate-500 block uppercase font-medium">
              {eta > 0 ? 'Tempo Restante' : 'Decorrido'}
            </span>
            <span className="text-xs font-mono font-semibold text-slate-200">
              {eta > 0 ? `${eta}s` : `${elapsed}s`}
            </span>
          </div>
        </div>

        {/* Bytes / Total */}
        <div className="bg-slate-950/60 border border-slate-800/80 rounded-xl p-3 flex items-center gap-2.5">
          <div className="p-2 rounded-lg bg-indigo-500/10 text-indigo-400">
            <HardDrive className="w-4 h-4" />
          </div>
          <div>
            <span className="text-[11px] text-slate-500 block uppercase font-medium">Volume</span>
            <span className="text-xs font-mono font-semibold text-slate-200 truncate block">
              {totalSize ? formatBytes(totalSize) : '--'}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};
