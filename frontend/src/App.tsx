import React, { useState } from 'react';
import { HomeScreen } from './screens/HomeScreen';
import { PairingScreen } from './screens/PairingScreen';
import { TransferScreen } from './screens/TransferScreen';
import { ThemeProvider, useTheme } from './context/ThemeContext';
import { Sun, Moon } from 'lucide-react';
import type { ViewMode, PairingMode } from './types/navigation';

const AppContent: React.FC = () => {
  const [currentView, setCurrentView] = useState<ViewMode>('home');
  const [pairingMode, setPairingMode] = useState<PairingMode>('send');
  const { theme, toggleTheme } = useTheme();

  const handleNavigate = (view: ViewMode, mode?: PairingMode) => {
    if (mode) setPairingMode(mode);
    setCurrentView(view);
  };

  return (
    <div className="flex flex-col h-screen w-screen bg-[var(--bg-app)] text-[var(--text-main)] select-none overflow-hidden font-sans transition-colors duration-150 relative">
      {/* Discreet Theme Switcher */}
      <button
        type="button"
        onClick={toggleTheme}
        title={theme === 'dark' ? 'Modo Claro' : 'Modo Escuro'}
        className="absolute top-3 right-3 z-50 p-1.5 rounded-[3px] bg-[var(--bg-panel)] hover:bg-[var(--bg-panel-hover)] border border-[var(--border-app)] text-[var(--text-muted)] hover:text-[var(--text-main)] transition-colors cursor-pointer"
      >
        {theme === 'dark' ? <Sun className="w-3.5 h-3.5 text-amber-400" /> : <Moon className="w-3.5 h-3.5 text-blue-400" />}
      </button>

      {/* Main View Router */}
      <main className="flex-1 flex flex-col overflow-hidden relative">
        {currentView === 'home' && (
          <HomeScreen onNavigate={handleNavigate} />
        )}

        {currentView === 'pairing' && (
          <PairingScreen
            initialMode={pairingMode}
            onModeChange={(newMode) => setPairingMode(newMode)}
            onBack={() => setCurrentView('home')}
            onConnected={(connectedMode) => setCurrentView(connectedMode)}
          />
        )}

        {(currentView === 'send' || currentView === 'receive') && (
          <TransferScreen
            role={currentView}
            onBack={() => setCurrentView('home')}
          />
        )}
      </main>
    </div>
  );
};

export const App: React.FC = () => {
  return (
    <ThemeProvider>
      <AppContent />
    </ThemeProvider>
  );
};

export default App;
