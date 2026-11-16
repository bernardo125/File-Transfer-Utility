import type { CSSProperties } from 'react';
import body from '../../assets/photos/bird/body.webp';
import wingUp from '../../assets/photos/bird/wing-up.webp';
import wingRight from '../../assets/photos/bird/wing-right.webp';
import tail from '../../assets/photos/bird/tail.webp';

/** `off` is the static magpie; `slow` hovers; `fast` is a cruising flap. */
export type FlapSpeed = 'off' | 'slow' | 'fast';

interface FlappingBirdProps {
  flap: FlapSpeed;
  className?: string;
  style?: CSSProperties;
}

const layer = 'bird-layer pointer-events-none';

/**
 * The FTU magpie split into body, two wings and tail (stacked, same canvas) so the wings can beat
 * around the shoulders. At rest it renders identically to the original artwork.
 */
export function FlappingBird({ flap, className = '', style }: FlappingBirdProps) {
  return (
    <div className={`bird ${flap === 'off' ? '' : `bird-flap bird-flap-${flap}`} ${className}`} style={style} aria-hidden>
      <img src={tail} alt="" className={`${layer} bird-tail`} width={359} height={360} />
      <img src={wingUp} alt="" className={`${layer} bird-wing-up`} width={359} height={360} />
      <img src={wingRight} alt="" className={`${layer} bird-wing-right`} width={359} height={360} />
      <img src={body} alt="" className={`${layer} bird-body`} width={359} height={360} />
    </div>
  );
}
