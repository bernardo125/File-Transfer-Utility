import React from 'react';
import { ArrowLeft, Download } from 'lucide-react';

interface ReceiveScreenProps {
  onBack: () => void;
}

export const ReceiveScreen: React.FC<ReceiveScreenProps> = ({ onBack }) => {
  return (
    <div className="flex-1 flex flex-col p-6 max-w-4xl mx-auto w-full">
      <div className="flex items-center gap-3 mb-6">
        <button
          type="button"
          onClick={onBack}
          className="h-8 px-3 rounded-[3px] bg-[var(--bg-panel)] hover:bg-[var(--bg-panel-hover)] border border-[var(--border-app)] transition-colors cursor-pointer flex items-center gap-1.5 text-xs font-medium"
          title="Voltar ao início"
        >
          <ArrowLeft className="w-3.5 h-3.5" />
          <span>Voltar</span>
        </button>
        <div className="flex items-center gap-2">
          <Download className="w-4 h-4 text-[#2fa340]" />
          <h2 className="text-sm font-bold uppercase tracking-wider font-mono">
            Receber Ficheiro (Receive)
          </h2>
        </div>
      </div>

      <div className="flex-1 flex flex-col items-center justify-center border border-dashed border-[var(--border-app)] rounded-[4px] bg-[var(--bg-panel)] p-8 text-center">
        <p className="text-xs font-semibold uppercase tracking-wider font-mono">
          Ecrã de Recepção Preparado
        </p>
        <p className="text-xs text-[var(--text-dim)] mt-1 font-mono">
          A aguardar especificações do utilizador para este ecrã.
        </p>
      </div>
    </div>
  );
};
