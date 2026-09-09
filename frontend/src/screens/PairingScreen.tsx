import React, { useState, useEffect, useRef, useCallback } from 'react';
import { GetLocalToken, ConnectPeer, CancelConnection, CopyToClipboard } from '../wailsjs/go/ui/App';
import { useWailsEvents } from '../hooks/useWailsEvents';
import type { PairingMode } from '../types/navigation';
import { ArrowLeft, Send, Download, Copy, Check, Clock, Link, X, AlertTriangle } from 'lucide-react';

interface PairingScreenProps {
  initialMode: PairingMode;
  onModeChange?: (m: PairingMode) => void;
  onBack: () => void;
  onConnected: (m: PairingMode) => void;
}

export const PairingScreen: React.FC<PairingScreenProps> = ({ initialMode, onModeChange, onBack, onConnected }) => {
  const [mode, setMode] = useState<PairingMode>(initialMode);
  const [localToken, setLocalToken] = useState('');
  const [remoteToken, setRemoteToken] = useState('');
  const [isLoadingToken, setIsLoadingToken] = useState(true);
  const [isCopied, setIsCopied] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false);
  const [countdown, setCountdown] = useState(30);
  const [error, setError] = useState<string | null>(null);
  const timerRef = useRef<number | null>(null);
  const { connectionSuccess, connectionFailed } = useWailsEvents();

  const stopTimer = useCallback(() => {
    if (timerRef.current !== null) { clearInterval(timerRef.current); timerRef.current = null; }
  }, []);

  useEffect(() => {
    setIsLoadingToken(true); setError(null);
    GetLocalToken().then((t) => { setLocalToken(t); setIsLoadingToken(false); })
      .catch((err) => { setIsLoadingToken(false); setError(`Erro STUN: ${err instanceof Error ? err.message : String(err)}`); });
    return () => stopTimer();
  }, [stopTimer]);

  useEffect(() => {
    if (!isConnecting) return;
    if (connectionSuccess) { stopTimer(); setIsConnecting(false); onConnected(mode); }
    else if (connectionFailed) { stopTimer(); setIsConnecting(false); setError(connectionFailed.reason || 'Falha de conexão'); }
  }, [connectionSuccess, connectionFailed, isConnecting, stopTimer, onConnected, mode]);

  const handleCopy = async () => {
    if (!localToken) return;
    try { await CopyToClipboard(localToken); } catch { await navigator.clipboard?.writeText(localToken); }
    setIsCopied(true); setTimeout(() => setIsCopied(false), 2000);
  };

  const handleConnect = async () => {
    const trimmed = remoteToken.trim();
    if (!trimmed) return;
    setError(null); setIsConnecting(true); setCountdown(30); stopTimer();
    timerRef.current = window.setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          stopTimer(); setIsConnecting(false); setError('Tempo limite excedido. Não foi possível conectar ao par.');
          CancelConnection().catch(() => {});
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    try { await ConnectPeer(trimmed); }
    catch (err) { stopTimer(); setIsConnecting(false); setError(`Erro ao conectar: ${err instanceof Error ? err.message : String(err)}`); }
  };

  const handleCancel = () => {
    stopTimer(); setIsConnecting(false); CancelConnection().catch(() => {});
  };

  const isSender = mode === 'send';

  return (
    <div className="flex-1 flex flex-col p-6 max-w-4xl mx-auto w-full select-none bg-[var(--bg-app)] text-[var(--text-main)] justify-center">
      {/* Navigation & Mode Toggle */}
      <div className="flex items-center justify-between border-b border-[var(--border-subtle)] pb-3 mb-6">
        <button type="button" disabled={isConnecting} onClick={onBack}
          className="h-7 px-2.5 rounded-[2px] bg-[var(--bg-panel)] hover:bg-[var(--bg-panel-hover)] disabled:opacity-40 border border-[var(--border-app)] text-xs font-mono flex items-center gap-1.5 cursor-pointer">
          <ArrowLeft className="w-3.5 h-3.5" /><span>Início</span>
        </button>

        <div className="flex items-center bg-[var(--bg-inset)] p-0.5 rounded-[3px] border border-[var(--border-app)]">
          <button type="button" disabled={isConnecting} onClick={() => { setMode('send'); onModeChange?.('send'); }}
            className={`px-3 py-1 rounded-[2px] text-xs font-mono font-semibold flex items-center gap-1.5 cursor-pointer ${isSender ? 'bg-[#2fa340] text-white' : 'text-[var(--text-muted)] hover:text-[var(--text-main)]'}`}>
            <Send className="w-3 h-3" /><span>Enviar (Sender)</span>
          </button>
          <button type="button" disabled={isConnecting} onClick={() => { setMode('receive'); onModeChange?.('receive'); }}
            className={`px-3 py-1 rounded-[2px] text-xs font-mono font-semibold flex items-center gap-1.5 cursor-pointer ${!isSender ? 'bg-[#2fa340] text-white' : 'text-[var(--text-muted)] hover:text-[var(--text-main)]'}`}>
            <Download className="w-3 h-3" /><span>Receber (Receiver)</span>
          </button>
        </div>
      </div>

      {/* Two-Column Layout */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-8 items-center">
        {/* Left: Graphic Placeholder (continuous 2s 360-deg rotation during connection) */}
        <div className="flex flex-col items-center justify-center p-4">
          <div className={`w-44 h-44 rounded-[4px] bg-[var(--bg-panel)] border border-[var(--border-app)] flex flex-col items-center justify-center ${isConnecting ? 'animate-rotate-2s' : ''}`}>
            <div className="w-16 h-16 rounded-[3px] bg-[var(--bg-inset)] border border-[var(--border-subtle)] flex flex-col items-center justify-center text-[#2fa340]">
              {isSender ? <Send className="w-7 h-7" /> : <Download className="w-7 h-7" />}
              <span className="text-[9px] font-mono font-bold uppercase mt-1">{isSender ? 'Sender' : 'Receiver'}</span>
            </div>
            <span className="text-[9px] font-mono text-[var(--text-dim)] uppercase mt-2">[Placeholder]</span>
          </div>
          <p className="text-[11px] font-mono text-[var(--text-muted)] text-center mt-3">
            {isConnecting ? 'A perfurar firewall via UDP Hole Punching...' : `Modo ${isSender ? 'Enviar' : 'Receber'} ativo`}
          </p>
        </div>

        {/* Right: State A (Tokens) or State B (Connecting) */}
        <div>
          {error && (
            <div className="mb-3 p-2.5 rounded-[3px] bg-rose-950/40 border border-rose-800/60 text-rose-200 text-xs flex items-center justify-between font-mono">
              <div className="flex items-center gap-2 truncate">
                <AlertTriangle className="w-3.5 h-3.5 text-rose-400 flex-shrink-0" /><span className="truncate">{error}</span>
              </div>
              <button type="button" onClick={() => setError(null)} className="text-rose-400 hover:text-rose-200 ml-2 cursor-pointer"><X className="w-3.5 h-3.5" /></button>
            </div>
          )}

          {!isConnecting ? (
            <div className="bg-[var(--bg-panel)] border border-[var(--border-app)] rounded-[4px] p-4 space-y-3 shadow-xs">
              <div>
                <div className="flex justify-between items-center mb-1">
                  <span className="text-[11px] font-mono text-[var(--text-muted)] uppercase font-semibold">O Teu Token ({isSender ? 'Sender' : 'Receiver'})</span>
                  <span className="text-[10px] font-mono text-[#2fa340]">STUN Pronto</span>
                </div>
                <div className="bg-[var(--bg-inset)] border border-[var(--border-app)] rounded-[3px] p-2 font-mono text-xs break-all select-all min-h-[44px] max-h-16 overflow-y-auto">
                  {isLoadingToken ? 'A obter token STUN...' : localToken || 'Nenhum token'}
                </div>
                <button type="button" disabled={isLoadingToken || !localToken} onClick={handleCopy}
                  className="mt-1.5 h-6 px-2.5 rounded-[2px] bg-[var(--bg-panel-hover)] hover:bg-[var(--border-app)] disabled:opacity-40 text-[11px] font-mono border border-[var(--border-subtle)] flex items-center gap-1 cursor-pointer">
                  {isCopied ? <Check className="w-3 h-3 text-[#2fa340]" /> : <Copy className="w-3 h-3 text-[var(--text-muted)]" />}
                  <span>{isCopied ? 'Copiado!' : 'Copiar Token'}</span>
                </button>
              </div>

              <div>
                <label className="block text-[11px] font-mono text-[var(--text-muted)] uppercase font-semibold mb-1">
                  {isSender ? 'Token do Destinatário' : 'Token do Remetente'}
                </label>
                <textarea value={remoteToken} onChange={(e) => setRemoteToken(e.target.value)}
                  placeholder="Cole aqui o token recebido do seu contacto..." rows={2}
                  className="w-full bg-[var(--bg-inset)] border border-[var(--border-app)] focus:border-[#2fa340] rounded-[3px] p-2 font-mono text-xs focus:outline-none resize-none transition-colors" />
              </div>

              <div className="flex justify-end pt-1">
                <button type="button" disabled={isLoadingToken || !remoteToken.trim()} onClick={handleConnect}
                  className="h-8 px-5 rounded-[3px] bg-[#2fa340] hover:bg-[#288d37] active:bg-[#1f752c] disabled:opacity-40 text-white font-semibold text-xs uppercase font-mono tracking-wider flex items-center gap-1.5 cursor-pointer">
                  <Link className="w-3 h-3" /><span>Conectar</span>
                </button>
              </div>
            </div>
          ) : (
            <div className="bg-[var(--bg-panel)] border border-[var(--border-app)] rounded-[4px] p-6 flex flex-col items-center justify-center space-y-4 shadow-xs text-center">
              <div>
                <p className="text-xs font-mono font-bold text-[#2fa340] uppercase animate-pulse">A perfurar firewall via UDP Hole Punching...</p>
                <p className="text-[11px] font-mono text-[var(--text-muted)] mt-0.5">Tentativa de sincronização e handshake QUIC</p>
              </div>
              <div className="p-3 px-6 rounded-[3px] bg-[var(--bg-inset)] border border-[var(--border-subtle)] flex flex-col items-center">
                <span className="text-[10px] font-mono text-[var(--text-dim)] flex items-center gap-1 mb-0.5"><Clock className="w-3 h-3 text-[#2fa340]" />TEMPO LIMITE</span>
                <span className="text-4xl font-mono font-bold text-[#2fa340] tracking-widest">00:{countdown < 10 ? `0${countdown}` : countdown}</span>
              </div>
              <button type="button" onClick={handleCancel}
                className="h-8 px-6 rounded-[3px] bg-[var(--bg-inset)] hover:text-rose-400 hover:border-rose-800/60 border border-[var(--border-app)] text-xs font-mono uppercase cursor-pointer flex items-center gap-1.5">
                <X className="w-3.5 h-3.5" /><span>Cancelar</span>
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
