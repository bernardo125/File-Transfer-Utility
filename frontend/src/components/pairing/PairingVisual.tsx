import { WaggingBird } from '../ui/WaggingBird';

const RADIUS = 92;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;

interface PairingVisualProps {
  /** When set, a ring shows the real connection timeout draining. */
  countdown?: { remaining: number; total: number };
}

/** The magpie; while connecting its tail sways and a ring around it drains with the real timeout. */
export function PairingVisual({ countdown }: PairingVisualProps) {
  const fraction = countdown ? Math.max(0, countdown.remaining) / countdown.total : 1;

  return (
    <div className="relative flex aspect-square w-full max-w-[220px] items-center justify-center">
      <svg viewBox="0 0 200 200" className={`absolute inset-0 size-full -rotate-90 transition-opacity duration-300 ${countdown ? 'opacity-100' : 'opacity-0'}`} aria-hidden>
        <circle cx="100" cy="100" r={RADIUS} fill="none" stroke="var(--c-border)" strokeWidth="2" />
        <circle
          cx="100"
          cy="100"
          r={RADIUS}
          fill="none"
          stroke="var(--c-accent)"
          strokeWidth="3"
          strokeLinecap="round"
          strokeDasharray={CIRCUMFERENCE}
          strokeDashoffset={CIRCUMFERENCE * (1 - fraction)}
          style={{ transition: 'stroke-dashoffset 1s linear' }}
        />
      </svg>
      <WaggingBird
        animated={!!countdown}
        className={`transition-[width] duration-500 ease-out ${countdown ? 'w-[60%]' : 'w-[72%]'}`}
      />
      {countdown && (
        <span
          className="absolute bottom-3 rounded-sm border border-line bg-surface px-2 py-0.5 text-[0.6875rem] font-semibold tabular-nums text-muted"
          role="timer"
          aria-label={`${countdown.remaining} seconds before the attempt times out`}
        >
          0:{String(countdown.remaining).padStart(2, '0')}
        </span>
      )}
    </div>
  );
}
