const ROWS = 4;
const COLS = 14;
const STEP_PX = 6;

/** Deterministic 0..1 pseudo-random so the trail looks organic but renders the same every time. */
const rand = (seed: number) => {
  const x = Math.sin(seed * 12.9898) * 43758.5453;
  return x - Math.floor(x);
};

const DIGITS = Array.from({ length: ROWS * COLS }, (_, n) => {
  const row = Math.floor(n / COLS);
  const col = n % COLS;
  return {
    key: n,
    top: `${6 + row * 26}%`,
    right: `${col * STEP_PX}px`,
    drift: `${-0.08 * (col + 1) * 10 - 6}px`,
    delay: `${-(col * 0.045 + rand(n + 1) * 0.5)}s`,
    flip: `${-rand(n + 7) * 0.3}s`,
    tone: (row + col) % 2 ? 'var(--c-bit-b)' : 'var(--c-bit-a)',
  };
});

interface BitStreamProps {
  /** True only while bytes are really moving; otherwise nothing is drawn. */
  flowing: boolean;
}

/**
 * The binary digits from the home artwork, alive: a dense trail of tiny 0s and 1s that keep
 * flipping and drift away behind the bird while data is genuinely flowing. It lives in the bird's
 * own frame so it follows the heading. Decorative, hidden from assistive technology.
 */
export function BitStream({ flowing }: BitStreamProps) {
  if (!flowing) return null;
  return (
    <div className="pointer-events-none absolute right-[80%] top-0 h-full w-20" aria-hidden>
      {DIGITS.map((d) => (
        <span
          key={d.key}
          className="anim-bit absolute text-[0.4375rem] font-extrabold leading-none"
          style={{ top: d.top, right: d.right, color: d.tone, animationDelay: d.delay, ['--bit-drift' as string]: d.drift }}
        >
          <i className="bit-a" style={{ animationDelay: d.flip }}>0</i>
          <i className="bit-b" style={{ animationDelay: d.flip }}>1</i>
        </span>
      ))}
    </div>
  );
}
