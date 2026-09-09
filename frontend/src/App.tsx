import React, { useState } from 'react';
import { HomeScreen } from './screens/HomeScreen';
import { PairingScreen } from './screens/PairingScreen';
import { TransferScreen } from './screens/TransferScreen';
import { ThemeProvider, useTheme } from './context/ThemeContext';
import type { ViewMode, PairingMode } from './types/navigation';
import { ThemeSwitcher } from './components/common/ThemeSwitcher';
import { GitHubButton } from './components/common/GitHubButton';

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
      {/* Github and Theme Switcher Buttons */}
      <div className="absolute bottom-3 left-3 z-[9999] flex gap-4 p-4">
        <GitHubButton />
        <ThemeSwitcher theme={theme} onClick={toggleTheme}/>
      </div>
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
