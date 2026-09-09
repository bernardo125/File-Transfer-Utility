import React from 'react';
import { Monitor } from 'lucide-react';

interface NetworkTopologyProps {
  role: 'send' | 'receive';
}

export const NetworkTopology: React.FC<NetworkTopologyProps> = () => {
  return (
    <div className="w-full items-center justify-center rounded-[4px] p-3  mb-5">
      <div className="flex items-center justify-between max-w-lg mx-auto">
        {/* PC Local (Cisco Terminal Style) */}
        <div className="flex flex-col items-center gap-1 min-w-[90px]">
          <div className="w-12 h-12 rounded-[3px] bg-[var(--bg-inset)] border border-[var(--border-subtle)] flex items-center justify-center text-[#2fa340] shadow-inner">
            <Monitor className="w-6 h-6 stroke-[1.8]" />
          </div>
          <span className="text-[11px]   -bold text-[var(--text-main)]">Local PC</span>

        </div>
        {/*Inter PCs trace */}
        <div className="flex-1 px-4 flex flex-col items-center">
          <div className="w-full flex items-center justify-center relative">
            <div className="w-full border-t-2 border-dashed border-[#2fa340]/60" />
            <div className=" px-2.5 py-0.5 rounded-[2px] bg-[var(--bg-inset)] border border-[#2fa340]/50 text-[#2fa340] text-[9px]    -bold flex items-center gap-1.5 shadow-xs">
            </div>
          </div>
        </div>

        {/* PC Remoto */}
        <div className="flex flex-col items-center gap-1 min-w-[90px]">
          <div className="w-12 h-12 rounded-[3px] bg-[var(--bg-inset)] border border-[var(--border-subtle)] flex items-center justify-center text-[var(--text-muted)] shadow-inner">
            <Monitor className="w-6 h-6 stroke-[1.8]" />
          </div>
          <span className="text-[11px]  -    -bold text-[var(--text-main)]">Remote PC</span>
        </div>
      </div>
    </div>
  );
};
