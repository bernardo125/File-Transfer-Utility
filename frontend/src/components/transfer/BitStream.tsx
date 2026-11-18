import type { CSSProperties } from 'react';
const COUNT = 44;
const TRAIL_LENGTH_PX = 84;

/** Deterministic 0..1 pseudo-random: an organic scatter that renders identically every time. */
const rand = (seed: number) => {
  const x = Math.sin(seed * 12.9898 + 78.233) * 43758.5453;
  return x - Math.floor(x);
};

/**
 * Digits are scattered through a cone that widens behind the bird. Each one has its own lifetime,
 * drift and flip rate, so the cloud never settles into rows or a repeating pattern.
 */
const DIGITS = Array.from({ length: COUNT }, (_, n) => {
  const along = Math.pow(rand(n * 3 + 1), 0.8) * TRAIL_LENGTH_PX; // distance behind the bird
  const spread = 2 + along * 0.2; // a narrow cone that widens only slightly with distance
  return {
    key: n,
    right: `${along.toFixed(1)}px`,
    top: `${((rand(n * 3 + 2) - 0.5) * 2 * spread).toFixed(1)}px`,
    style: {
      '--bit-dx': `${(-6 - rand(n * 7 + 3) * 14).toFixed(1)}px`,
      '--bit-dy': `${((rand(n * 7 + 4) - 0.5) * 6).toFixed(1)}px`,
      '--bit-dur': `${(0.55 + rand(n * 7 + 5) * 0.7).toFixed(2)}s`,
      '--bit-flip': `${(0.09 + rand(n * 7 + 6) * 0.14).toFixed(3)}s`,
      animationDelay: `${(-rand(n * 7 + 7) * 1.2).toFixed(2)}s`,
      color: rand(n * 5 + 9) > 0.5 ? 'var(--c-bit-a)' : 'var(--c-bit-b)',
    } as CSSProperties,
    flipDelay: `${(-rand(n * 5 + 8) * 0.2).toFixed(3)}s`,
  };
});

interface BitStreamProps {
  /** True only while bytes are really moving; otherwise nothing is drawn. */
  flowing: boolean;
}

/**
 * The binary digits from the home artwork, alive: a cloud of very small 0s and 1s that flip rapidly
 * and drift off behind the bird while data is genuinely flowing. It lives in the bird's own frame, so
 * it follows the heading. Decorative, hidden from assistive technology.
 */
export function BitStream({ flowing }: BitStreamProps) {
  if (!flowing) return null;
  return (
    <div className="pointer-events-none absolute right-[72%] top-1/2 h-0 w-0" aria-hidden>
      {DIGITS.map((d) => (
        <span key={d.key} className="anim-bit absolute text-[0.3125rem] font-extrabold leading-none" style={{ right: d.right, top: d.top, ...d.style }}>
          <i className="bit-a" style={{ animationDelay: d.flipDelay }}>0</i>
          <i className="bit-b" style={{ animationDelay: d.flipDelay }}>1</i>
        </span>
      ))}
    </div>
  );
}
