import React from 'react';
import logo from '../../assets/photos/magpie.png';

export const LogoPlaceholder: React.FC = () => {
  return (
    <div className="flex flex-col items-center justify-center select-none">
      <div className="relative w-70 h-70 flex  items-center">
        <img src={logo} alt="Logo" />
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
