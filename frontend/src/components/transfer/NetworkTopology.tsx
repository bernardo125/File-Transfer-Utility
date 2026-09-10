import React from 'react';
import { Monitor } from 'lucide-react';

interface NetworkTopologyProps {
  role: 'send' | 'receive';
}

export const NetworkTopology: React.FC<NetworkTopologyProps> = () => {
  return (
    <div className="w-full flex items-center justify-center mb-6">
      <div className="flex items-start justify-between w-full max-w-[470px]">
        {/* PC Local (Cisco Terminal Style) */}
        <div className="flex flex-col items-center gap-1 min-w-[90px]">
          <div className="w-12 h-12 rounded-[3px] bg-[var(--bg-inset)] border border-[var(--border-subtle)] flex items-center justify-center text-[#2fa340] shadow-inner">
            <Monitor className="w-7 h-7" />
          </div>
          <span className="text-[11px] text-[var(--text-main)]">Local PC</span>
        </div>

        {/* Inter PCs trace */}
        <div className="flex-1 px-4 h-12 flex items-center justify-center">
          <div className="w-full border-t-2 border-dashed border-[#2fa340]/60" />
        </div>

        {/* PC Remoto */}
        <div className="flex flex-col items-center gap-1 min-w-[90px]">
          <div className="w-12 h-12 rounded-[3px] bg-[var(--bg-inset)] border border-[var(--border-subtle)] flex items-center justify-center text-[var(--text-muted)] shadow-inner">
            <Monitor className="w-7 h-7" />
          </div>
          <span className="text-[11px] text-[var(--text-main)]">Remote PC</span>
        </div>
      </div>
    </div>
  );
};
