import React from 'react';
import type { ConnectionState } from '../types/transfer';
import { Wifi, WifiOff, RefreshCw, CheckCircle2, AlertTriangle, ShieldCheck } from 'lucide-react';

interface ConnectionBadgeProps {
  state: ConnectionState;
  rttMs?: number;
}

export const ConnectionBadge: React.FC<ConnectionBadgeProps> = ({ state, rttMs }) => {
  const getBadgeConfig = () => {
    switch (state) {
      case 'connected':
        return {
          icon: <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />,
          label: rttMs !== undefined && rttMs > 0 ? `Conectado (${rttMs}ms)` : 'Conectado (QUIC)',
          bg: 'bg-emerald-950/60 border-emerald-700/60 text-emerald-300',
          dot: 'bg-emerald-400',
        };
      case 'hole_punching':
        return {
          icon: <RefreshCw className="w-3.5 h-3.5 text-amber-400 animate-spin" />,
          label: 'Hole Punching...',
          bg: 'bg-amber-950/60 border-amber-700/60 text-amber-300',
          dot: 'bg-amber-400 animate-ping',
        };
      case 'upgrading':
        return {
          icon: <ShieldCheck className="w-3.5 h-3.5 text-cyan-400 animate-pulse" />,
          label: 'QUIC Handshake...',
          bg: 'bg-cyan-950/60 border-cyan-700/60 text-cyan-300',
          dot: 'bg-cyan-400',
        };
      case 'discovery':
        return {
          icon: <RefreshCw className="w-3.5 h-3.5 text-blue-400 animate-spin" />,
          label: 'STUN Discovery...',
          bg: 'bg-blue-950/60 border-blue-700/60 text-blue-300',
          dot: 'bg-blue-400',
        };
      case 'ready':
        return {
          icon: <Wifi className="w-3.5 h-3.5 text-indigo-400" />,
          label: 'Pronto (Token Gerado)',
          bg: 'bg-indigo-950/60 border-indigo-700/60 text-indigo-300',
          dot: 'bg-indigo-400',
        };
      case 'error':
        return {
          icon: <AlertTriangle className="w-3.5 h-3.5 text-rose-400" />,
          label: 'Falha na Ligação',
          bg: 'bg-rose-950/60 border-rose-700/60 text-rose-300',
          dot: 'bg-rose-400',
        };
      case 'idle':
      default:
        return {
          icon: <WifiOff className="w-3.5 h-3.5 text-slate-400" />,
          label: 'Desconectado',
          bg: 'bg-slate-900/80 border-slate-700/50 text-slate-400',
          dot: 'bg-slate-500',
        };
    }
  };

  const config = getBadgeConfig();

  return (
    <div
      className={`inline-flex items-center gap-2 px-2.5 py-1 rounded-full border text-xs font-medium backdrop-blur-sm transition-all duration-300 ${config.bg}`}
    >
      <span className={`w-1.5 h-1.5 rounded-full ${config.dot}`} />
      {config.icon}
      <span>{config.label}</span>
    </div>
  );
};
