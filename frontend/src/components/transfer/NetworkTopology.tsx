import React from 'react';
import { Monitor, ArrowLeftRight } from 'lucide-react';

interface NetworkTopologyProps {
  role: 'send' | 'receive';
}

export const NetworkTopology: React.FC<NetworkTopologyProps> = ({ role }) => {
  return (
    <div className="w-full bg-[var(--bg-panel)] border border-[var(--border-app)] rounded-[4px] p-3 shadow-xs mb-5">
      <div className="flex items-center justify-between max-w-lg mx-auto">
        {/* PC Local (Cisco Terminal Style) */}
        <div className="flex flex-col items-center gap-1 min-w-[90px]">
          <div className="w-12 h-12 rounded-[3px] bg-[var(--bg-inset)] border border-[var(--border-subtle)] flex items-center justify-center text-[#2fa340] shadow-inner">
            <Monitor className="w-6 h-6 stroke-[1.8]" />
          </div>
          <span className="text-[11px] font-mono font-bold text-[var(--text-main)]">PC Local</span>
          <span className="text-[9px] font-mono text-[#2fa340] uppercase font-semibold">
            {role === 'send' ? 'Sender' : 'Receiver'}
          </span>
        </div>

        {/* Cisco-style QUIC Active Data Link */}
        <div className="flex-1 px-4 flex flex-col items-center">
          <div className="w-full flex items-center justify-center relative">
            <div className="w-full border-t-2 border-dashed border-[#2fa340]/60" />
            <div className="absolute px-2.5 py-0.5 rounded-[2px] bg-[var(--bg-inset)] border border-[#2fa340]/50 text-[#2fa340] text-[9px] font-mono font-bold flex items-center gap-1.5 shadow-xs">
              <span className="w-2 h-2 rounded-full bg-[#2fa340] animate-ping" />
              <ArrowLeftRight className="w-3 h-3" />
              <span>QUIC TUNNEL</span>
            </div>
          </div>
          <span className="text-[9px] font-mono text-[var(--text-dim)] mt-2">128-bit ChaCha20-Poly1305</span>
        </div>

        {/* PC Remoto */}
        <div className="flex flex-col items-center gap-1 min-w-[90px]">
          <div className="w-12 h-12 rounded-[3px] bg-[var(--bg-inset)] border border-[var(--border-subtle)] flex items-center justify-center text-[var(--text-muted)] shadow-inner">
            <Monitor className="w-6 h-6 stroke-[1.8]" />
          </div>
          <span className="text-[11px] font-mono font-bold text-[var(--text-main)]">PC Remoto</span>
          <span className="text-[9px] font-mono text-[var(--text-dim)] uppercase font-semibold">
            {role === 'send' ? 'Receiver' : 'Sender'}
          </span>
        </div>
      </div>
    </div>
  );
};
