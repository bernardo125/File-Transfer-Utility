import type { CSSProperties } from 'react';
import body from '../../assets/photos/bird/body.webp';
import wingUp from '../../assets/photos/bird/wing-up.webp';
import wingRight from '../../assets/photos/bird/wing-right.webp';
import tail from '../../assets/photos/bird/tail.webp';

/** `off` is the static magpie; `slow` hovers; `fast` is a cruising flap. */
export type FlapSpeed = 'off' | 'slow' | 'fast';

/** Frames per sprite sheet (see assets/photos/bird). Frame 0 is the original, unwarped artwork. */
const FRAMES = 8;

interface FlappingBirdProps {
  flap: FlapSpeed;
  className?: string;
  style?: CSSProperties;
}

function Sheet({ src, className = '' }: { src: string; className?: string }) {
  return (
    <div className={`bird-layer ${className}`}>
      <img src={src} alt="" className="bird-sheet" draggable={false} />
    </div>
  );
}

/**
 * The FTU magpie as stacked layers. The wings and tail are pre-rendered flap cycles (the wing bends:
 * fixed at the shoulder, tip sweeping) played as a sprite sheet; at rest it is the original artwork.
 */
export function FlappingBird({ flap, className = '', style }: FlappingBirdProps) {
  const cssVars = { '--frames': FRAMES, ...style } as CSSProperties;
  return (
    <div className={`bird ${flap === 'off' ? '' : `bird-flap bird-flap-${flap}`} ${className}`} style={cssVars} aria-hidden>
      <Sheet src={tail} />
      <Sheet src={wingUp} />
      <Sheet src={wingRight} />
      <div className="bird-layer bird-body">
        <img src={body} alt="" draggable={false} className="size-full" />
      </div>
    </div>
  );
}
