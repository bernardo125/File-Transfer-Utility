/** Fixed bit pattern: rows of digits that drift backwards from the bird. Purely decorative. */
const ROWS: string[][] = [
  ['0', '1', '1', '0', '1', '0'],
  ['1', '1', '0', '1', '0', '0'],
  ['0', '1', '0', '0', '1', '1'],
];
const ROW_TOP = ['12%', '46%', '78%'];

interface BitStreamProps {
  /** True only while bytes are really moving; otherwise the digits are hidden. */
  flowing: boolean;
}

/**
 * The binary digits from the home artwork, now alive: they spawn at the bird's tail and drift back
 * along the route while data is flowing. Drawn behind the bird; hidden from assistive tech.
 */
export function BitStream({ flowing }: BitStreamProps) {
  if (!flowing) return null;
  return (
    <div className="pointer-events-none absolute right-[78%] top-0 h-full w-[130%]" aria-hidden>
      {ROWS.map((row, r) =>
        row.map((bit, i) => (
          <span
            key={`${r}-${i}`}
            className={`anim-bit absolute text-[0.95rem] font-extrabold leading-none ${(r + i) % 2 ? 'text-[#7d6f66]' : 'text-[#2f6aa8]'}`}
            style={{ top: ROW_TOP[r], right: `${i * 15}%`, animationDelay: `${-(i * 0.22 + r * 0.31)}s` }}
          >
            {bit}
          </span>
        )),
      )}
    </div>
  );
}
