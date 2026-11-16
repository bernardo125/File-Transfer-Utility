interface ProgressBarProps {
  /** 0–100. `null` renders an empty track with `idleLabel`. */
  percent: number | null;
  idleLabel: string;
  failed?: boolean;
}

/**
 * µTorrent-style bar: a bordered track with the percentage centred on it. The filled part is a
 * full-width layer (own background + white label) clipped to the percentage, so the label always has
 * its own contrasting background and stays in sync with the fill.
 */
export function ProgressBar({ percent, idleLabel, failed }: ProgressBarProps) {
  const p = percent === null ? 0 : Math.min(100, Math.max(0, percent));
  const label = percent === null ? idleLabel : `${p.toFixed(1)} %`;
  const labelClass = 'absolute inset-0 flex items-center justify-center text-[0.6875rem] font-semibold tabular-nums';

  return (
    <div
      role="progressbar"
      aria-label="Transfer progress"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={percent === null ? undefined : Math.floor(p)}
      aria-valuetext={label}
      className="relative h-5 overflow-hidden rounded-sm border border-control bg-inset"
    >
      <span className={`${labelClass} ${percent === null ? 'text-dim' : 'text-fg'}`}>{label}</span>
      {p > 0 && (
        <span
          className={`${labelClass} forced-fill text-white transition-[clip-path] duration-200 ease-linear ${failed ? 'bg-danger-fill' : 'bg-primary'}`}
          style={{ clipPath: `inset(0 ${100 - p}% 0 0)` }}
          aria-hidden
        >
          {label}
        </span>
      )}
    </div>
  );
}
