import type { CSSProperties } from 'react';
import staticLayer from '../../assets/photos/bird/static.webp';
import tail from '../../assets/photos/bird/tail.webp';

/** Frames in the tail sprite sheet (assets/photos/bird/tail.webp). Frame 0 is the original artwork. */
const FRAMES = 8;

interface WaggingBirdProps {
  /** Sway the tail. When false the bird is the original, still artwork. */
  animated: boolean;
  className?: string;
}

/**
 * The pairing magpie. Body and wings are one still layer; only the tail sways, played from a
 * pre-rendered sprite sheet while a connection attempt is running.
 */
export function WaggingBird({ animated, className = '' }: WaggingBirdProps) {
  return (
    <div className={`bird ${animated ? 'bird-animated' : ''} ${className}`} style={{ '--frames': FRAMES } as CSSProperties} aria-hidden>
      <div className="bird-layer">
        <img src={tail} alt="" className="bird-sheet" draggable={false} />
      </div>
      <img src={staticLayer} alt="" className="bird-layer size-full" draggable={false} />
    </div>
  );
}
