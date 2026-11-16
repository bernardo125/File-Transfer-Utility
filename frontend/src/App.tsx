import { useCallback, useEffect, useRef, useState } from 'react';
import { HomeScreen } from './screens/HomeScreen';
import { PairingScreen } from './screens/PairingScreen';
import { TransferScreen } from './screens/TransferScreen';
import { ThemeProvider } from './context/ThemeContext';
import { MotionProvider } from './context/MotionContext';
import { useDocumentTitle } from './hooks/useDocumentTitle';
import { SessionProvider } from './context/SessionContext';
import { FloatingTools } from './components/layout/FloatingTools';
import type { PeerRole, ViewMode } from './types/navigation';

function AppContent() {
  const [view, setView] = useState<ViewMode>('home');
  const [role, setRole] = useState<PeerRole>('send');

  const startPairing = useCallback((nextRole: PeerRole) => {
    setRole(nextRole);
    setView('pairing');
  }, []);
  const goHome = useCallback(() => setView('home'), []);
  const goTransfer = useCallback(() => setView('transfer'), []);

  const screenTitle = view === 'home' ? undefined : view === 'pairing' ? 'Pair with your peer' : role === 'send' ? 'Sending' : 'Receiving';
  useDocumentTitle(screenTitle);

  // Move focus to the new screen so keyboard and screen-reader users start at its top (WCAG 2.4.3).
  const viewRef = useRef<HTMLDivElement>(null);
  const firstRender = useRef(true);
  useEffect(() => {
    if (firstRender.current) {
      firstRender.current = false;
      return;
    }
    viewRef.current?.focus({ preventScroll: true });
  }, [view]);

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div key={view} ref={viewRef} tabIndex={-1} className="anim-view-in relative flex min-h-0 flex-1 flex-col outline-none">
        {view === 'home' && <HomeScreen onStart={startPairing} />}
        {view === 'pairing' && <PairingScreen role={role} onBack={goHome} onConnected={goTransfer} />}
        {view === 'transfer' && <TransferScreen role={role} onExit={goHome} />}
      </div>
      <FloatingTools />
    </div>
  );
}

export default function App() {
  return (
    <ThemeProvider>
      <MotionProvider>
        <SessionProvider>
          <AppContent />
        </SessionProvider>
      </MotionProvider>
    </ThemeProvider>
  );
}
