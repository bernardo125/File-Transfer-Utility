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
    <div className="w-full h-screen flex flex-col relative overflow-hidden">
      {/* Github and Theme Switcher Buttons */}
      <div className="absolute bottom-3 left-1 flex gap-3 ">
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
