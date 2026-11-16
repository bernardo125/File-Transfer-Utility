import { Zap, ZapOff } from 'lucide-react';
import { useMotion } from '../../context/MotionContext';
import { IconButton } from '../ui/IconButton';

/** Toggle for animations. A pressed button means animations are reduced. */
export function MotionToggle() {
  const { reduced, toggleReduced } = useMotion();
  return <IconButton icon={reduced ? ZapOff : Zap} label="Reduce animations" aria-pressed={reduced} onClick={toggleReduced} />;
}
