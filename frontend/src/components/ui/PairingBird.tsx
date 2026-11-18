import type { CSSProperties } from 'react';
import body from '../../assets/photos/bird/body.webp';
import wingUp from '../../assets/photos/bird/wing-up.webp';
import wingRight from '../../assets/photos/bird/wing-right.webp';
import tail from '../../assets/photos/bird/tail.webp';

/** Frames per sprite sheet (scripts/bird-sprites.py). Frame 0 is the original artwork. */
const FRAMES = 8;

interface PairingBirdProps {
  /** Play the gentle wing and tail cycle. When false the bird is the original, still artwork. */
  animated: boolean;
  className?: string;
}

function Sheet({ src }: { src: string }) {
  return (
    <div className="bird-layer">
      <img src={src} alt="" className="bird-sheet" draggable={false} />
    </div>
  );
}

/**
 * The pairing magpie as stacked layers. Wings and tail are pre-rendered sprite cycles (the wing
 * bends with its root fixed at the shoulder); the body is still. Amplitudes are deliberately small.
 */
export function PairingBird({ animated, className = '' }: PairingBirdProps) {
  return (
    <div className={`bird ${animated ? 'bird-animated' : ''} ${className}`} style={{ '--frames': FRAMES } as CSSProperties} aria-hidden>
      <Sheet src={tail} />
      <Sheet src={wingUp} />
      <Sheet src={wingRight} />
      <img src={body} alt="" className="bird-layer size-full" draggable={false} />
    </div>
  );
}
