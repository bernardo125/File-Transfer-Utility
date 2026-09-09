import React from 'react';

export const LogoPlaceholder: React.FC = () => {
  return (
    <div className="flex flex-col items-center justify-center gap-3 select-none">
      <div className="relative w-24 h-24 rounded-[4px] bg-[var(--bg-panel)] border border-[var(--border-app)] shadow-sm flex flex-col items-center justify-center group transition-all">
    
      </div>

      {/* Software Title & Version */}
      <div className="text-center">
        <h1 className="text-base font-bold tracking-tight text-[var(--text-main)] uppercase">
          File Throw
        </h1>
        <p className="text-[11px] text-[var(--text-muted)] mt-0.5">
          QUIC TRANSFER UTILITY (v1.0)
        </p>
      </div>
    </div>
  );
};
