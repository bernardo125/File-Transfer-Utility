import React from 'react';

export const LogoPlaceholder: React.FC = () => {
  return (
    <div className="flex flex-col items-center justify-center gap-3 select-none">
      {/* µTorrent-style Emblem Box */}
      <div className="relative w-24 h-24 rounded-[4px] bg-[var(--bg-panel)] border border-[var(--border-app)] shadow-sm flex flex-col items-center justify-center group transition-all">
        {/* Subtle decorative corner accent */}
        <div className="absolute top-1 left-1 w-1.5 h-1.5 border-t border-l border-[#2fa340]" />
        <div className="absolute bottom-1 right-1 w-1.5 h-1.5 border-b border-r border-[#2fa340]" />

        {/* Central Stylized Glyph */}
        <div className="w-12 h-12 rounded-[3px] bg-[var(--bg-inset)] border border-[var(--border-subtle)] flex items-center justify-center shadow-inner">
          <span className="text-3xl font-bold font-serif text-[#2fa340] leading-none select-none">
            µ
          </span>
        </div>

        {/* Clear Placeholder Marker */}
        <span className="text-[9px] font-mono uppercase tracking-widest text-[var(--text-dim)] mt-1.5 font-semibold">
          [Logo Placeholder]
        </span>
      </div>

      {/* Software Title & Version */}
      <div className="text-center">
        <h1 className="text-base font-bold tracking-tight text-[var(--text-main)] uppercase font-mono">
          File Transfer Utility
        </h1>
        <p className="text-[11px] text-[var(--text-muted)] font-mono mt-0.5">
          QUIC & UDP Hole Punching P2P Client
        </p>
      </div>
    </div>
  );
};
