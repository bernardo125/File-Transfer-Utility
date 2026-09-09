import React from 'react';
import type { IncomingTransfer } from '../types/transfer';
import { DownloadCloud, Check, X, FileText } from 'lucide-react';
import { RespondIncomingTransfer } from '../wailsjs/go/ui/App';

interface IncomingTransferModalProps {
  incoming: IncomingTransfer;
  onDecision: () => void;
}

export const IncomingTransferModal: React.FC<IncomingTransferModalProps> = ({ incoming, onDecision }) => {
  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  const handleAccept = async () => {
    try {
      await RespondIncomingTransfer(true);
      onDecision();
    } catch (err) {
      console.error('Failed to accept transfer:', err);
    }
  };

  const handleReject = async () => {
    try {
      await RespondIncomingTransfer(false);
      onDecision();
    } catch (err) {
      console.error('Failed to reject transfer:', err);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
      <div className="bg-slate-900 border border-slate-700/80 rounded-2xl shadow-2xl max-w-md w-full p-6 text-slate-100 animate-in fade-in zoom-in-95 duration-200">
        <div className="flex items-center gap-3 mb-4">
          <div className="w-10 h-10 rounded-xl bg-cyan-500/10 border border-cyan-500/20 flex items-center justify-center text-cyan-400">
            <DownloadCloud className="w-5 h-5" />
          </div>
          <div>
            <h3 className="text-base font-semibold text-slate-100">Pedido de Transferência</h3>
            <p className="text-xs text-slate-400">O teu peer pretende enviar um ficheiro</p>
          </div>
        </div>

        <div className="bg-slate-950/60 border border-slate-800 rounded-xl p-3.5 mb-5 flex items-center gap-3">
          <div className="w-9 h-9 rounded-lg bg-slate-800 flex items-center justify-center text-slate-300 flex-shrink-0">
            <FileText className="w-5 h-5 text-cyan-400" />
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium text-slate-200 truncate">{incoming.fileName}</p>
            <p className="text-xs text-slate-400">{formatBytes(incoming.fileSize)}</p>
          </div>
        </div>

        {incoming.fileHash && (
          <div className="mb-5 bg-slate-950/40 px-3 py-2 rounded-lg border border-slate-800/60">
            <span className="text-[11px] text-slate-500 block mb-0.5">SHA-256 Esperado:</span>
            <span className="text-xs font-mono text-slate-400 break-all select-all">{incoming.fileHash}</span>
          </div>
        )}

        <div className="flex gap-3">
          <button
            type="button"
            onClick={handleReject}
            className="flex-1 px-4 py-2.5 rounded-xl border border-slate-700 hover:bg-slate-800 text-slate-300 text-sm font-medium flex items-center justify-center gap-2 transition-colors cursor-pointer"
          >
            <X className="w-4 h-4 text-rose-400" />
            Recusar
          </button>
          <button
            type="button"
            onClick={handleAccept}
            className="flex-1 px-4 py-2.5 rounded-xl bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium flex items-center justify-center gap-2 transition-colors shadow-lg shadow-cyan-900/30 cursor-pointer"
          >
            <Check className="w-4 h-4" />
            Aceitar
          </button>
        </div>
      </div>
    </div>
  );
};
