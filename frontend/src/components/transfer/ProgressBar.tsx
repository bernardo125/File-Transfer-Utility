interface ProgressBarProps {
  /** 0–100. `null` renders an empty track with `idleLabel`. */
  percent: number | null;
  idleLabel: string;
  failed?: boolean;
}

/**
 * µTorrent-style bar: a bordered track with the percentage centred on it. The label is drawn twice
 * (once on the track, once clipped to the fill with the same transition) so it stays readable on both backgrounds.
 */
export function ProgressBar({ percent, idleLabel, failed }: ProgressBarProps) {
  const p = percent === null ? 0 : Math.min(100, Math.max(0, percent));
  const label = percent === null ? idleLabel : `${p.toFixed(1)} %`;
  const labelClass = 'absolute inset-y-0 flex items-center justify-center text-[11px] font-semibold tabular-nums';

  return (
    <div
      role="progressbar"
      aria-label="Transfer progress"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={percent === null ? undefined : Math.floor(p)}
      aria-valuetext={label}
      className="relative h-5 overflow-hidden rounded-sm border border-line-strong bg-inset"
    >
      {p > 0 && (
        <div
          className={`absolute inset-y-0 left-0 transition-[width] duration-200 ease-linear ${failed ? 'bg-danger' : 'bg-primary'}`}
          style={{ width: `${p}%` }}
          aria-hidden
        />
      )}
      <span className={`${labelClass} inset-x-0 ${percent === null ? 'text-dim' : 'text-fg'}`}>{label}</span>
      {p > 0 && (
        <span
          className={`${labelClass} inset-x-0 text-white transition-[clip-path] duration-200 ease-linear`}
          style={{ clipPath: `inset(0 ${100 - p}% 0 0)` }}
          aria-hidden
        >
          {label}
        </span>
      )}
    </div>
  );
}
