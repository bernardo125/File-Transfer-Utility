import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';

interface MotionContextType {
  /** True when animations are reduced, by OS setting or by the user's choice. */
  reduced: boolean;
  toggleReduced: () => void;
}

const STORAGE_KEY = 'ftu_reduce_motion';
const MotionContext = createContext<MotionContextType | undefined>(undefined);

function systemPrefersReduced(): boolean {
  return window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false;
}

function initialPreference(): boolean {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === 'true' || saved === 'false') return saved === 'true';
  } catch {
    // storage unavailable — use the OS preference
  }
  return systemPrefersReduced();
}

/** Lets users turn animations off (or back on) regardless of the OS setting. */
export function MotionProvider({ children }: { children: ReactNode }) {
  const [reduced, setReduced] = useState(initialPreference);

  useEffect(() => {
    const root = document.documentElement;
    root.classList.toggle('reduce-motion', reduced);
    root.classList.toggle('motion-on', !reduced);
  }, [reduced]);

  const toggleReduced = useCallback(() => {
    setReduced((prev) => {
      try {
        localStorage.setItem(STORAGE_KEY, String(!prev));
      } catch {
        // ignore
      }
      return !prev;
    });
  }, []);

  const value = useMemo(() => ({ reduced, toggleReduced }), [reduced, toggleReduced]);
  return <MotionContext.Provider value={value}>{children}</MotionContext.Provider>;
}

export function useMotion(): MotionContextType {
  const ctx = useContext(MotionContext);
  if (!ctx) throw new Error('useMotion must be used within a MotionProvider');
  return ctx;
}
