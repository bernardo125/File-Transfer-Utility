import pairingBird from '../../assets/photos/magpie-pairing.webp';

const RADIUS = 92;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;

interface PairingVisualProps {
  /** When set, a ring shows the real connection timeout draining. */
  countdown?: { remaining: number; total: number };
}

/** The magpie waiting to pair; while connecting, a timeout ring counts down around it. */
export function PairingVisual({ countdown }: PairingVisualProps) {
  const fraction = countdown ? countdown.remaining / countdown.total : 1;

  return (
    <div className="relative flex aspect-square w-full max-w-[240px] items-center justify-center">
      <svg viewBox="0 0 200 200" className="absolute inset-0 size-full -rotate-90" aria-hidden>
        <circle cx="100" cy="100" r={RADIUS} fill="var(--c-surface)" stroke="var(--c-border)" strokeWidth="3" />
        {countdown && (
          <circle
            cx="100"
            cy="100"
            r={RADIUS}
            fill="none"
            stroke="var(--c-accent)"
            strokeWidth="4"
            strokeLinecap="round"
            strokeDasharray={CIRCUMFERENCE}
            strokeDashoffset={CIRCUMFERENCE * (1 - fraction)}
            style={{ transition: 'stroke-dashoffset 1s linear' }}
          />
        )}
      </svg>
      <img src={pairingBird} alt="" width={359} height={360} className={`relative w-[58%] ${countdown ? 'anim-hover' : ''}`} />
      {countdown && (
        <span
          className="absolute bottom-4 rounded-full border border-line bg-surface px-2.5 py-0.5 font-mono text-xs tabular-nums text-muted"
          role="timer"
          aria-label={`${countdown.remaining} seconds before the attempt times out`}
        >
          0:{String(countdown.remaining).padStart(2, '0')}
        </span>
      )}
    </div>
  );
}
