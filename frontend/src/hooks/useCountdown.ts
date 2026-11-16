import { useCallback, useEffect, useRef, useState } from 'react';

/**
 * A one-second countdown. `onExpire` fires once when it reaches zero.
 * The latest `onExpire` is always used, so callers don't need to memoise it.
 */
export function useCountdown(seconds: number, onExpire: () => void) {
  const [remaining, setRemaining] = useState(seconds);
  const intervalRef = useRef<number | null>(null);
  const onExpireRef = useRef(onExpire);
  useEffect(() => {
    onExpireRef.current = onExpire;
  });

  const stop = useCallback(() => {
    if (intervalRef.current !== null) {
      window.clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  const start = useCallback(() => {
    stop();
    let left = seconds;
    setRemaining(left);
    intervalRef.current = window.setInterval(() => {
      left -= 1;
      setRemaining(left);
      if (left <= 0) {
        stop();
        onExpireRef.current();
      }
    }, 1000);
  }, [seconds, stop]);

  useEffect(() => stop, [stop]);

  return { remaining, start, stop };
}
