import { useCallback, useState } from 'react';
import { HomeScreen } from './screens/HomeScreen';
import { PairingScreen } from './screens/PairingScreen';
import { TransferScreen } from './screens/TransferScreen';
import { ThemeProvider } from './context/ThemeContext';
import { SessionProvider } from './context/SessionContext';
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

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {view === 'home' && <HomeScreen onStart={startPairing} />}
      {view === 'pairing' && <PairingScreen role={role} onBack={goHome} onConnected={goTransfer} />}
      {view === 'transfer' && <TransferScreen role={role} onExit={goHome} />}
    </div>
  );
}

export default function App() {
  return (
    <ThemeProvider>
      <SessionProvider>
        <AppContent />
      </SessionProvider>
    </ThemeProvider>
  );
}
