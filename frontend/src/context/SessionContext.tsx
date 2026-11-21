import { createContext, useContext, type ReactNode } from 'react';
import { useWailsEvents, type SessionValue } from '../hooks/useWailsEvents';

const SessionContext = createContext<SessionValue | null>(null);

/** Owns the single backend event subscription for the whole app. */
export function SessionProvider({ children }: { children: ReactNode }) {
  const session = useWailsEvents();
  return <SessionContext.Provider value={session}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionValue {
  const ctx = useContext(SessionContext);
  if (!ctx) throw new Error('useSession must be used within a SessionProvider');
  return ctx;
}
