// Development fallback mock for when the frontend runs in a standalone browser
// (e.g. Vite dev server outside of the Wails desktop webview).

type EventCallback = (...args: unknown[]) => void;

interface WailsWindow extends Window {
  go?: {
    ui?: {
      App?: Record<string, (...args: unknown[]) => Promise<unknown> | unknown>;
    };
  };
  runtime?: {
    EventsOnMultiple?: (eventName: string, callback: EventCallback, maxCallbacks: number) => () => void;
    EventsOn?: (eventName: string, callback: EventCallback) => () => void;
    EventsOff?: (eventName: string, ...additionalEventNames: string[]) => void;
    EventsOffAll?: () => void;
    EventsOnce?: (eventName: string, callback: EventCallback) => () => void;
    EventsEmit?: (eventName: string, ...args: unknown[]) => void;
    ClipboardSetText?: (text: string) => Promise<boolean>;
    ClipboardGetText?: () => Promise<string>;
    BrowserOpenURL?: (url: string) => void;
    Quit?: () => void;
    LogPrint?: (message: string) => void;
    LogTrace?: (message: string) => void;
    LogDebug?: (message: string) => void;
    LogInfo?: (message: string) => void;
    LogWarning?: (message: string) => void;
    LogError?: (message: string) => void;
    LogFatal?: (message: string) => void;
  };
}

export function initWailsMock(): void {
  if (typeof window === 'undefined') return;

  const w = window as unknown as WailsWindow;

  // 1. Mock Wails Runtime if not injected
  if (!w.runtime || typeof w.runtime.EventsOnMultiple !== 'function') {
    const eventListeners: Record<string, EventCallback[]> = {};

    w.runtime = {
      EventsOnMultiple: (eventName: string, callback: EventCallback) => {
        if (!eventListeners[eventName]) {
          eventListeners[eventName] = [];
        }
        eventListeners[eventName].push(callback);
        return () => {
          eventListeners[eventName] = (eventListeners[eventName] || []).filter((cb) => cb !== callback);
        };
      },
      EventsOn: (eventName: string, callback: EventCallback) => {
        if (!eventListeners[eventName]) {
          eventListeners[eventName] = [];
        }
        eventListeners[eventName].push(callback);
        return () => {
          eventListeners[eventName] = (eventListeners[eventName] || []).filter((cb) => cb !== callback);
        };
      },
      EventsOff: (eventName: string, ...additionalEventNames: string[]) => {
        const events = [eventName, ...additionalEventNames];
        events.forEach((ev) => {
          delete eventListeners[ev];
        });
      },
      EventsOffAll: () => {
        Object.keys(eventListeners).forEach((k) => delete eventListeners[k]);
      },
      EventsOnce: (eventName: string, callback: EventCallback) => {
        const onceWrapper: EventCallback = (...args) => {
          callback(...args);
          w.runtime?.EventsOff?.(eventName);
        };
        return w.runtime?.EventsOn?.(eventName, onceWrapper) || (() => {});
      },
      EventsEmit: (eventName: string, ...args: unknown[]) => {
        const listeners = eventListeners[eventName] || [];
        listeners.forEach((cb) => {
          try {
            cb(...args);
          } catch (e) {
            console.error(`Error in mock event listener for ${eventName}:`, e);
          }
        });
      },
      ClipboardSetText: async (text: string) => {
        try {
          await navigator.clipboard.writeText(text);
          return true;
        } catch {
          return false;
        }
      },
      ClipboardGetText: async () => {
        try {
          return await navigator.clipboard.readText();
        } catch {
          return '';
        }
      },
      BrowserOpenURL: (url: string) => {
        window.open(url, '_blank');
      },
      Quit: () => {
        console.info('[Wails Mock] Quit application called.');
      },
      LogPrint: (msg) => console.log('[Wails]', msg),
      LogTrace: (msg) => console.trace('[Wails]', msg),
      LogDebug: (msg) => console.debug('[Wails]', msg),
      LogInfo: (msg) => console.info('[Wails]', msg),
      LogWarning: (msg) => console.warn('[Wails]', msg),
      LogError: (msg) => console.error('[Wails]', msg),
      LogFatal: (msg) => console.error('[Wails FATAL]', msg),
    };
  }

  // 2. Mock Go Bindings if not injected
  if (!w.go || !w.go.ui || !w.go.ui.App) {
    const mockToken =
      'FTU-STUN-k9v2x4Q8mJ1wP5zL3yR7tN0bC6hF2dV4sA8qE1gK3uT5wX7zY9-DEV-MOCK';

    if (!w.go) w.go = {};
    if (!w.go.ui) w.go.ui = {};

    w.go.ui.App = {
      GetLocalToken: async () => {
        // Simulate STUN network latency
        await new Promise((r) => setTimeout(r, 400));
        return mockToken;
      },
      GetMyToken: async () => {
        await new Promise((r) => setTimeout(r, 400));
        return mockToken;
      },
      ConnectPeer: async (token: unknown) => {
        console.info('[Wails Mock] ConnectPeer called with token:', token);
        // Simulate async UDP hole punch and QUIC upgrade (completes in 2.5s)
        setTimeout(() => {
          w.runtime?.EventsEmit?.('connection:success', {
            addr: '192.168.1.100:48234',
            rttMs: 18,
          });
          w.runtime?.EventsEmit?.('p2p:connection-state', {
            state: 'connected',
            detail: 'Conectado a 192.168.1.100:48234 (Mock)',
            rttMs: 18,
          });
        }, 2500);
        return undefined;
      },
      CancelConnection: async () => {
        console.info('[Wails Mock] CancelConnection called.');
        w.runtime?.EventsEmit?.('connection:failed', {
          reason: 'Ligação cancelada pelo utilizador',
        });
        w.runtime?.EventsEmit?.('p2p:connection-state', {
          state: 'ready',
          detail: 'Tentativa cancelada',
        });
        return undefined;
      },
      CopyToClipboard: async (text: unknown) => {
        try {
          await navigator.clipboard.writeText(String(text));
        } catch {
          // ignore
        }
        return undefined;
      },
      GetClipboardText: async () => {
        try {
          return await navigator.clipboard.readText();
        } catch {
          return '';
        }
      },
      QuitApplication: async () => {
        console.info('[Wails Mock] QuitApplication called.');
        return undefined;
      },
      OpenGitHub: async () => {
        window.open('https://github.com/bernardo125/File-Transfer-Utility', '_blank');
        return undefined;
      },
      Disconnect: async () => {
        console.info('[Wails Mock] Disconnect called.');
        w.runtime?.EventsEmit?.('p2p:connection-state', {
          state: 'idle',
          detail: 'Desconectado',
        });
        return undefined;
      },
      SelectFileNative: async () => {
        return {
          path: '/mock/path/document_sample.zip',
          name: 'document_sample.zip',
          size: 14857600,
          hash: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
        };
      },
      InspectFile: async (path: unknown) => {
        return {
          path: String(path),
          name: 'document_sample.zip',
          size: 14857600,
          hash: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
        };
      },
      SelectDownloadDirectory: async () => '/home/user/Downloads',
      GetDownloadDirectory: async () => '/home/user/Downloads',
      OpenFolder: async (path: unknown) => {
        console.info('[Wails Mock] OpenFolder:', path);
        return undefined;
      },
      SendSelectedFile: async () => {
        console.info('[Wails Mock] SendSelectedFile started.');
        w.runtime?.EventsEmit?.('transfer:status', 'waiting_approval');
        setTimeout(() => {
          w.runtime?.EventsEmit?.('transfer:status', 'transferring');
          let currentPct = 0;
          const totalBytes = 14857600;
          const interval = setInterval(() => {
            currentPct += 20;
            if (currentPct >= 100) {
              clearInterval(interval);
              w.runtime?.EventsEmit?.('transfer:progress', {
                bytesTransferred: totalBytes,
                totalBytes,
                percentage: 100,
                speedMBs: 42.5,
                etaSeconds: 0,
              });
              w.runtime?.EventsEmit?.('transfer:completed', {
                fileName: 'document_sample.zip',
                destPath: '/home/user/Downloads/document_sample.zip',
                fileSize: totalBytes,
                durationSec: 1.25,
                speedMBs: 42.5,
                hashVerified: true,
              });
            } else {
              w.runtime?.EventsEmit?.('transfer:progress', {
                bytesTransferred: Math.round((totalBytes * currentPct) / 100),
                totalBytes,
                percentage: currentPct,
                speedMBs: 38.2,
                etaSeconds: Math.round((5 - (currentPct / 20))),
              });
            }
          }, 300);
        }, 1500);
        return undefined;
      },
      RespondIncomingTransfer: async (accept: unknown) => {
        console.info('[Wails Mock] RespondIncomingTransfer:', accept);
        if (accept) {
          w.runtime?.EventsEmit?.('transfer:status', 'transferring');
          let currentPct = 0;
          const totalBytes = 14857600;
          const interval = setInterval(() => {
            currentPct += 25;
            if (currentPct >= 100) {
              clearInterval(interval);
              w.runtime?.EventsEmit?.('transfer:progress', {
                bytesTransferred: totalBytes,
                totalBytes,
                percentage: 100,
                speedMBs: 35.8,
                etaSeconds: 0,
              });
              w.runtime?.EventsEmit?.('transfer:completed', {
                fileName: 'document_sample.zip',
                destPath: '/home/user/Downloads/document_sample.zip',
                fileSize: totalBytes,
                durationSec: 1.4,
                speedMBs: 35.8,
                hashVerified: true,
              });
            } else {
              w.runtime?.EventsEmit?.('transfer:progress', {
                bytesTransferred: Math.round((totalBytes * currentPct) / 100),
                totalBytes,
                percentage: currentPct,
                speedMBs: 34.0,
                etaSeconds: Math.round(4 - (currentPct / 25)),
              });
            }
          }, 300);
        } else {
          w.runtime?.EventsEmit?.('transfer:status', 'rejected');
        }
        return undefined;
      },
    };

    // Auto-trigger an incoming proposal for receiver in browser mock after connection
    setTimeout(() => {
      w.runtime?.EventsEmit?.('transfer:incoming', {
        fileName: 'sample_archive_package.zip',
        fileSize: 48576000,
        fileHash: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
      });
    }, 4000);
  }
}
