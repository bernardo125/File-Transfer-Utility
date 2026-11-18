package ui

import wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

// emit delivers a backend event to the frontend. In the desktop app that is the Wails runtime;
// a headless host (the E2E shell in internal/e2e/uishell) installs its own emitter instead.
func (a *App) emit(event string, data ...any) {
	if a.emitter != nil {
		a.emitter(event, data...)
		return
	}
	wailsruntime.EventsEmit(a.ctx, event, data...)
}

// SetEventEmitter replaces how events reach the UI. Call it before the App is used.
// It exists so the real App can run without a native window (end-to-end tests).
func (a *App) SetEventEmitter(fn func(event string, data ...any)) { a.emitter = fn }

// SetDownloadDirectory sets where received files are saved (the native folder dialog does this
// in the desktop app).
func (a *App) SetDownloadDirectory(dir string) {
	a.mu.Lock()
	a.downloadDir = dir
	a.mu.Unlock()
}
