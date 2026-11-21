package ui

import (
	"context"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// emit delivers a backend event to the frontend. In the desktop app that is the Wails runtime;
// a headless host (the E2E shell in tests/ui/shell) installs its own emitter instead.
func (a *App) emit(event string, data ...any) {
	if a.emitter != nil {
		a.emitter(event, data...)
		return
	}
	wailsruntime.EventsEmit(a.ctx, event, data...)
}

// The helpers below let a headless host (tests/ui/shell, no native window) drive the real App.
// They are package-level functions, NOT methods: Wails binds every exported method of the App to
// the renderer, and none of these may ever be callable from JavaScript.

// HeadlessSetEventEmitter replaces how events reach the UI. Call it before the App is used.
func HeadlessSetEventEmitter(a *App, fn func(event string, data ...any)) { a.setEventEmitter(fn) }

// HeadlessSetDownloadDirectory sets where received files are saved (the native folder dialog does
// this in the desktop app).
func HeadlessSetDownloadDirectory(a *App, dir string) { a.setDownloadDirectory(dir) }

// HeadlessAllowSendPath marks a path as chosen in the (simulated) native file dialog.
func HeadlessAllowSendPath(a *App, path string) { a.allowSendPath(path) }

// HeadlessShutdown closes the session and the UDP socket, like quitting the app.
func HeadlessShutdown(a *App) { a.shutdown(context.Background()) }

func (a *App) setEventEmitter(fn func(event string, data ...any)) { a.emitter = fn }

func (a *App) setDownloadDirectory(dir string) {
	a.mu.Lock()
	a.downloadDir = dir
	a.mu.Unlock()
}
