/** Two tidy rows of digits that drift back from the bird's tail. Purely decorative. */
const ROWS: string[][] = [
  ['1', '0', '1', '1', '0'],
  ['0', '1', '0', '0', '1'],
];
const ROW_TOP = ['18%', '58%'];
const STEP_PX = 11;

interface BitStreamProps {
  /** True only while bytes are really moving; otherwise nothing is drawn. */
  flowing: boolean;
}

/**
 * The binary digits from the home artwork, in motion: they leave the bird's tail and fade out
 * behind it while data is genuinely flowing. Positioned in the bird's own frame so they follow its
 * heading, and kept off the route line. Hidden from assistive technology.
 */
export function BitStream({ flowing }: BitStreamProps) {
  if (!flowing) return null;
  return (
    <div className="pointer-events-none absolute right-[82%] top-0 h-full w-12" aria-hidden>
      {ROWS.map((row, r) =>
        row.map((bit, i) => (
          <span
            key={`${r}-${i}`}
            className="anim-bit absolute text-[0.6875rem] font-extrabold leading-none"
            style={{
              top: ROW_TOP[r],
              right: `${i * STEP_PX}px`,
              color: (r + i) % 2 ? 'var(--c-bit-b)' : 'var(--c-bit-a)',
              animationDelay: `${-(i * 0.2 + r * 0.45)}s`,
            }}
          >
            {bit}
          </span>
        )),
      )}
    </div>
  );
}
