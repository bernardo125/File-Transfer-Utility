import React, { useState, useEffect } from 'react';
import { SelectFileNative, SendSelectedFile, RespondIncomingTransfer, OpenFolder, Disconnect } from '../wailsjs/go/ui/App';
import { useWailsEvents } from '../hooks/useWailsEvents';
import { NetworkTopology } from '../components/transfer/NetworkTopology';
import { ProgressPanel } from '../components/transfer/ProgressPanel';
import { formatBytes } from '../utils/format';
import type { FileMetadata } from '../types/transfer';
import { ArrowLeft, Upload, FileText, Check, X, Loader2 } from 'lucide-react';

interface TransferScreenProps {
  role: 'send' | 'receive';
  onBack: () => void;
}

export const TransferScreen: React.FC<TransferScreenProps> = ({ role, onBack }) => {
  const isSender = role === 'send';
  const { incomingTransfer, transferProgress, transferCompleted, transferStatus, transferError } = useWailsEvents();
  const [selectedFile, setSelectedFile] = useState<FileMetadata | null>(null);
  const [isWaitingApproval, setIsWaitingApproval] = useState(false);
  const [isRejected, setIsRejected] = useState(false);

  useEffect(() => {
    if (transferStatus === 'waiting_approval') setIsWaitingApproval(true);
    if (transferStatus === 'rejected') {
      setIsWaitingApproval(false); setIsRejected(true);
      const t = setTimeout(() => { setIsRejected(false); setSelectedFile(null); }, 3500);
      return () => clearTimeout(t);
    }
  }, [transferStatus]);

  const handleSelectFile = async () => {
    try {
      const file = await SelectFileNative();
      if (!file?.path) return;
      setSelectedFile(file); setIsWaitingApproval(true); setIsRejected(false);
      SendSelectedFile(file.path).catch(() => setIsWaitingApproval(false));
    } catch (err) { console.error('Pick error:', err); }
  };

  const handleDecision = (accept: boolean) => {
    RespondIncomingTransfer(accept).catch(() => {});
  };

  const handleAbort = () => {
    Disconnect().catch(() => {});
    onBack();
  };

  const isTransferActive = transferProgress !== null || transferCompleted !== null || transferStatus === 'transferring';
  const activeFileName = selectedFile?.name || incomingTransfer?.fileName || transferCompleted?.fileName || 'Ficheiro';
  const activeFileSize = selectedFile?.size || incomingTransfer?.fileSize || transferCompleted?.fileSize || 0;

  return (
    <div className="flex-1 flex flex-col justify-center py-6 mx-auto w-full">
      {/* Top Bar with Session Status */}
      <div className="absolute top-1 items-center justify-between border-b border-[var(--border-subtle)] pb-3 mb-4">
        <button type="button" onClick={handleAbort}
          className="h-7 px-2.5 rounded-[2px] bg-[var(--bg-panel)] hover:bg-[var(--bg-panel-hover)] border border-[var(--border-app)] text-xs flex items-center gap-1.5 cursor-pointer text-[var(--text-muted)] hover:text-[var(--text-main)]">
          <ArrowLeft className="w-3.5 h-3.5" /><span>Close Connection</span>
        </button>
      </div>

      {/* Cisco-Style Network Topology */}
      <NetworkTopology role={role} />

      {/* Dynamic Central Area */}
      <div className="min-h-[200px] w-full flex flex-col items-center justify-center">
        {isTransferActive ? (
          <ProgressPanel fileName={activeFileName} fileSize={activeFileSize} progress={transferProgress}
            completed={transferCompleted} isReceiver={!isSender} onAbort={handleAbort}
            onOpenFolder={(path) => OpenFolder(path).catch(() => {})}
          />
        ) : isSender ? (
          isRejected ? (
            <div className="w-full max-w-md p-4 rounded-[4px] bg-[var(--bg-panel)] border border-[var(--border-subtle)] flex items-center justify-between text-zinc-500 animate-pulse">
              <div className="flex items-center gap-2 truncate">
                <FileText className="w-4 h-4 text-zinc-500" /><span className="text-xs truncate">{selectedFile?.name}</span>
              </div>
              <span className="text-xs  uppercase text-zinc-400">Cancelled</span>
            </div>
          ) : isWaitingApproval && selectedFile ? (
            <div className="w-full max-w-md p-5 rounded-[4px] bg-[var(--bg-panel)] border border-[var(--border-app)] flex flex-col items-center gap-3 text-center shadow-xs">
              <div className="flex items-center gap-2 text-xs font-semibold">
                <FileText className="w-4 h-4 text-[#2fa340]" /><span className="truncate">{selectedFile.name} ({formatBytes(selectedFile.size)})</span>
              </div>
              <div className="flex items-center gap-2 text-xs text-[#2fa340] animate-pulse">
                <Loader2 className="w-3.5 h-3.5 animate-spin" /><span>Waiting for receiver confimation...</span>
              </div>
            </div>
          ) : (
            <div className="flex flex-col items-center gap-3">
              <button type="button" onClick={handleSelectFile}
                className="h-11 px-8 rounded-[3px] bg-[#2fa340] hover:bg-[#288d37] active:bg-[#1f752c] text-white  text-xs uppercase tracking-wider border border-[#278535] shadow-xs flex items-center gap-2 cursor-pointer transition-colors">
                <Upload className="w-4 h-4" /><span>Upload File</span>
              </button>
              <p className="text-[11px] text-[var(--text-dim)]">Select a file  to share with your friend</p>
            </div>
          )
        ) : incomingTransfer ? (
          <div className="w-full max-w-md p-5 rounded-[4px] bg-[var(--bg-panel)] border border-[var(--border-app)] space-y-4 shadow-sm">
            <div className="flex items-center justify-between border-b border-[var(--border-subtle)] pb-2">
              <span className="text-xs text-[var(--text-muted)] uppercase">You want to receive:</span>
              <span className="text-[10px] text-[#2fa340] ">Waiting for your response</span>
            </div>
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-[3px] bg-[var(--bg-inset)] border border-[var(--border-subtle)] flex items-center justify-center text-[#2fa340]">
                <FileText className="w-4 h-4" />
              </div>
              <div className="truncate flex-1">
                <p className="text-xs  truncate">{incomingTransfer.fileName}</p>
                <p className="text-[10px] text-[var(--text-dim)]">{formatBytes(incomingTransfer.fileSize)}</p>
              </div>
            </div>
            <div className="flex items-center justify-end gap-2.5 pt-1">
              <button type="button" onClick={() => handleDecision(false)}
                className="h-8 px-4 rounded-[2px] bg-[var(--bg-inset)] hover:text-rose-400 hover:border-rose-800/60 border border-[var(--border-app)] text-xs uppercase cursor-pointer flex items-center gap-1 transition-colors">
                <X className="w-3.5 h-3.5" /><span>No (Reject)</span>
              </button>
              <button type="button" onClick={() => handleDecision(true)}
                className="h-8 px-5 rounded-[2px] bg-[#2fa340] hover:bg-[#288d37] text-white text-xs  uppercase cursor-pointer flex items-center gap-1 shadow-xs transition-colors">
                <Check className="w-3.5 h-3.5" /><span>Yes (Accept)</span>
              </button>
            </div>
          </div>
        ) : (
          <div className="flex flex-col items-center gap-2 text-center p-6 border border-dashed border-[var(--border-app)] rounded-[4px] bg-[var(--bg-panel)] w-full max-w-md">
            <Loader2 className="w-5 h-5 text-[#2fa340] animate-spin mb-1" />
            <p className="text-xs font-semibold">Waiting for sender choose a file to send...</p>
            <p className="text-[10px] text-[var(--text-dim)]">The sender will select a file to share</p>
          </div>
        )}
      </div>

      {transferError && (
        <div className="mt-4 p-2 rounded-[2px] bg-rose-950/40 border border-rose-800/60 text-rose-300 text-[11px] text-center">
          {transferError.message || 'Tranfer error'}
        </div>
      )}
    </div>
  );
};
