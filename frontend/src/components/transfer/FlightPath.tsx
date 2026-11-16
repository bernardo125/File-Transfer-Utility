import { Check, Laptop, ShieldCheck, X } from 'lucide-react';
import type { TransferPhase } from '../../types/transfer';
import type { PeerRole } from '../../types/navigation';

/*
 * The signature FTU visual: a paper plane flying from the sender's machine to the receiver's.
 * The plane's position on the arc IS the real transfer progress — it only moves when the
 * backend reports bytes moving, and parks/lands according to the actual phase.
 */

const VIEW_W = 600;
const VIEW_H = 190;
const P0 = { x: 118, y: 120 }; // sender edge
const P1 = { x: 300, y: 18 }; // arc control point
const P2 = { x: 482, y: 120 }; // receiver edge
const ARC = `M ${P0.x} ${P0.y} Q ${P1.x} ${P1.y} ${P2.x} ${P2.y}`;
const NODE_X = { sender: 70, receiver: 530 };
const NODE_Y = 120;

/** Point and heading on the quadratic Bézier at t∈[0,1]. The arc is shallow, so t ≈ arc length. */
function pointOnArc(t: number) {
  const u = 1 - t;
  const x = u * u * P0.x + 2 * u * t * P1.x + t * t * P2.x;
  const y = u * u * P0.y + 2 * u * t * P1.y + t * t * P2.y;
  const dx = 2 * u * (P1.x - P0.x) + 2 * t * (P2.x - P1.x);
  const dy = 2 * u * (P1.y - P0.y) + 2 * t * (P2.y - P1.y);
  return { x, y, angle: (Math.atan2(dy, dx) * 180) / Math.PI };
}

type Tone = 'idle' | 'active' | 'success' | 'muted' | 'danger';

interface Visual {
  t: number;
  tone: Tone;
  hovering: boolean; // plane bobs while a real wait or preparation is in progress
  dashFlow: boolean; // dashes flow along the route while waiting on the other side
  pulse: 'sender' | 'receiver' | null; // whose action/work is pending
  landed: boolean;
}

function visualFor(phase: TransferPhase, fraction: number | null): Visual {
  const base: Visual = { t: 0, tone: 'idle', hovering: false, dashFlow: false, pulse: null, landed: false };
  switch (phase) {
    case 'idle':
      return base;
    case 'preparing':
      return { ...base, tone: 'active', hovering: true, pulse: 'sender' };
    case 'awaiting':
    case 'deciding':
      return { ...base, tone: 'active', hovering: true, dashFlow: true, pulse: 'receiver' };
    case 'transferring':
      return { ...base, tone: 'active', t: fraction ?? 0, hovering: fraction === null };
    case 'verifying':
      return { ...base, tone: 'active', t: 1, pulse: 'receiver' };
    case 'completed':
      return { ...base, tone: 'success', t: 1, landed: true };
    case 'declined':
      return { ...base, tone: 'muted' };
    case 'failed':
      return { ...base, tone: 'danger', t: fraction ?? 0 };
  }
}

const TONE_COLOR: Record<Tone, string> = {
  idle: 'var(--c-dim)',
  active: 'var(--c-accent)',
  success: 'var(--c-accent)',
  muted: 'var(--c-dim)',
  danger: 'var(--c-danger)',
};

interface PeerNodeProps {
  side: 'sender' | 'receiver';
  isYou: boolean;
  highlight: boolean;
  pulse: boolean;
  badge: 'success' | 'verifying' | 'danger' | null;
}

function PeerNode({ side, isYou, highlight, pulse, badge }: PeerNodeProps) {
  const left = `${(NODE_X[side] / VIEW_W) * 100}%`;
  const top = `${(NODE_Y / VIEW_H) * 100}%`;
  return (
    <div className="absolute flex -translate-x-1/2 -translate-y-1/2 flex-col items-center" style={{ left, top }}>
      <div className={`relative ${badge === 'danger' ? 'anim-shake' : ''}`} key={badge === 'danger' ? 'failed' : 'ok'}>
        {pulse && <span className="anim-pulse-ring absolute inset-0 rounded-md border-2 border-accent" aria-hidden />}
        {badge === 'success' && <span className="anim-ripple absolute inset-0 rounded-md border-2 border-accent" aria-hidden />}
        <div
          className={`relative flex size-14 items-center justify-center rounded-md border-2 bg-surface transition-colors duration-300 ${
            highlight ? 'border-accent text-accent' : 'border-line-strong text-muted'
          }`}
        >
          <Laptop className="size-6" aria-hidden />
          {badge && (
            <span
              key={badge}
              className={`anim-pop absolute -right-2 -top-2 flex size-5 items-center justify-center rounded-full border-2 border-surface ${
                badge === 'danger' ? 'bg-danger text-white' : 'bg-primary text-white'
              }`}
              aria-hidden
            >
              {badge === 'success' ? <Check className="size-3" /> : badge === 'verifying' ? <ShieldCheck className="size-3" /> : <X className="size-3" />}
            </span>
          )}
        </div>
      </div>
      <span className="mt-2 text-xs font-bold text-fg">{isYou ? 'You' : 'Peer'}</span>
      <span className="text-[10px] font-semibold uppercase tracking-wider text-dim">{side === 'sender' ? 'Sender' : 'Receiver'}</span>
    </div>
  );
}

interface FlightPathProps {
  role: PeerRole;
  phase: TransferPhase;
  /** Real progress in [0,1], or null when the backend has not reported bytes yet. */
  fraction: number | null;
}

export function FlightPath({ role, phase, fraction }: FlightPathProps) {
  const v = visualFor(phase, fraction);
  const color = TONE_COLOR[v.tone];
  const plane = pointOnArc(v.t);
  const moving = phase === 'transferring' && fraction !== null;

  return (
    <div className="relative mx-auto w-full max-w-[min(600px,76vh)]" style={{ aspectRatio: `${VIEW_W} / ${VIEW_H}` }} aria-hidden>
      <svg viewBox={`0 0 ${VIEW_W} ${VIEW_H}`} className="absolute inset-0 size-full overflow-visible">
        {/* Route */}
        <path
          d={ARC}
          fill="none"
          stroke="var(--c-border-strong)"
          strokeWidth="2"
          strokeDasharray="2 10"
          strokeLinecap="round"
          className={v.dashFlow ? 'anim-dash' : undefined}
        />
        {/* Trail: the distance actually covered */}
        <path
          d={ARC}
          fill="none"
          stroke={color}
          strokeWidth="3"
          strokeLinecap="round"
          pathLength={1}
          strokeDasharray="1 1"
          strokeDashoffset={1 - v.t}
          style={{ transition: `stroke-dashoffset ${moving ? '160ms linear' : '600ms ease-out'}, stroke 300ms` }}
        />
        {/* Paper plane */}
        <g
          style={{
            transform: `translate(${plane.x}px, ${plane.y}px) rotate(${plane.angle}deg) scale(${v.landed ? 0.3 : 1})`,
            transition: `transform ${moving ? '160ms linear' : '600ms cubic-bezier(.3,.7,.4,1)'}, opacity 300ms`,
            opacity: v.landed ? 0 : 1,
          }}
        >
          <g className={v.hovering ? 'anim-hover' : undefined}>
            <path d="M16 0 L-12 -11 L-5 0 Z" fill={color} />
            <path d="M16 0 L-5 0 L-12 11 Z" fill={color} opacity="0.65" />
            <path d="M16 0 L-5 0 L-8 6 Z" fill="var(--c-bg)" opacity="0.35" />
          </g>
        </g>
      </svg>

      <PeerNode
        side="sender"
        isYou={role === 'send'}
        highlight={v.tone === 'active' || v.tone === 'success'}
        pulse={v.pulse === 'sender'}
        badge={null}
      />
      <PeerNode
        side="receiver"
        isYou={role === 'receive'}
        highlight={v.t >= 1 || v.pulse === 'receiver' || phase === 'deciding'}
        pulse={v.pulse === 'receiver'}
        badge={phase === 'completed' ? 'success' : phase === 'verifying' ? 'verifying' : phase === 'failed' || phase === 'declined' ? 'danger' : null}
      />
    </div>
  );
}
