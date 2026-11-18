// Command uishell hosts the real ui.App (the Go backend of the desktop app) in a plain HTTP server
// so Playwright can drive the real frontend against it without a native Wails window.
//
// It replaces only what needs a native window: the IPC channel (HTTP RPC + SSE events), file and
// folder dialogs, the clipboard and "open folder". Everything else (token creation, hole punching,
// QUIC, transfer, hashing, receiver loop) is the production code path.
//
//	go run ./internal/e2e/uishell -addr 127.0.0.1:4301 -dist frontend/dist
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"sync"

	"ftu/pkg/ui"
)

var errorType = reflect.TypeOf((*error)(nil)).Elem()

type event struct {
	Name string `json:"name"`
	Data any    `json:"data"`
}

type shell struct {
	mu        sync.Mutex
	app       *ui.App
	subs      map[chan event]struct{}
	picks     []string // paths the next "file dialog" will return
	opened    []string // folders the UI asked to open
	clipboard string
	dlDir     string
}

func (s *shell) newApp(dir string) {
	s.mu.Lock()
	old := s.app
	s.mu.Unlock()
	if old != nil {
		old.Shutdown(nil) // closes the QUIC session and the UDP socket, like quitting the app
	}
	app := ui.NewApp()
	app.SetEventEmitter(s.broadcast)
	if dir != "" {
		app.SetDownloadDirectory(dir)
	}
	s.mu.Lock()
	s.app, s.picks, s.opened, s.clipboard, s.dlDir = app, nil, nil, "", dir
	s.mu.Unlock()
}

func (s *shell) broadcast(name string, data ...any) {
	var payload any
	if len(data) > 0 {
		payload = data[0]
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- event{name, payload}:
		default: // a stalled browser must never block the backend
		}
	}
}

func (s *shell) events(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "no streaming", http.StatusInternalServerError)
		return
	}
	ch := make(chan event, 256)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.subs, ch)
		s.mu.Unlock()
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	fmt.Fprint(w, ": connected\n\n")
	fl.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-ch:
			b, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", b)
			fl.Flush()
		}
	}
}

type rpcResult struct {
	OK    bool   `json:"ok"`
	Value any    `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
}

func (s *shell) rpc(w http.ResponseWriter, r *http.Request) {
	method := r.PathValue("method")
	body, _ := io.ReadAll(r.Body)
	var raw []json.RawMessage
	if len(body) > 0 {
		if err := json.Unmarshal(body, &raw); err != nil {
			writeJSON(w, rpcResult{Error: "bad args: " + err.Error()})
			return
		}
	}
	s.mu.Lock()
	app := s.app
	s.mu.Unlock()

	if v, handled, err := s.native(app, method, raw); handled {
		writeJSON(w, toResult(v, err))
		return
	}
	m := reflect.ValueOf(app).MethodByName(method)
	if !m.IsValid() || method == "Startup" || method == "Shutdown" {
		writeJSON(w, rpcResult{Error: "unknown method " + method})
		return
	}
	t := m.Type()
	if len(raw) != t.NumIn() {
		writeJSON(w, rpcResult{Error: fmt.Sprintf("%s expects %d args, got %d", method, t.NumIn(), len(raw))})
		return
	}
	in := make([]reflect.Value, t.NumIn())
	for i := range in {
		p := reflect.New(t.In(i))
		if err := json.Unmarshal(raw[i], p.Interface()); err != nil {
			writeJSON(w, rpcResult{Error: "bad arg: " + err.Error()})
			return
		}
		in[i] = p.Elem()
	}
	var val any
	var err error
	for _, o := range m.Call(in) {
		if o.Type() == errorType {
			if !o.IsNil() {
				err = o.Interface().(error)
			}
			continue
		}
		val = o.Interface()
	}
	writeJSON(w, toResult(val, err))
}

// native answers the calls that need a real window with deterministic stand-ins.
func (s *shell) native(app *ui.App, method string, raw []json.RawMessage) (any, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch method {
	case "SelectFileNative": // the "choose a file" dialog: returns what the test queued, or "dismissed"
		if len(s.picks) == 0 {
			return nil, true, nil
		}
		path := s.picks[0]
		s.picks = s.picks[1:]
		s.mu.Unlock()
		info, err := app.InspectFile(path) // real hashing and stat
		s.mu.Lock()
		return info, true, err
	case "SelectDownloadDirectory":
		return s.dlDir, true, nil
	case "OpenFolder":
		var p string
		if len(raw) == 1 {
			_ = json.Unmarshal(raw[0], &p)
		}
		s.opened = append(s.opened, p)
		return nil, true, nil
	case "CopyToClipboard":
		if len(raw) == 1 {
			_ = json.Unmarshal(raw[0], &s.clipboard)
		}
		return nil, true, nil
	case "GetClipboardText":
		return s.clipboard, true, nil
	case "QuitApplication", "OpenGitHub":
		return nil, true, nil
	}
	return nil, false, nil
}

func toResult(v any, err error) rpcResult {
	if err != nil {
		return rpcResult{Error: err.Error()}
	}
	return rpcResult{OK: true, Value: v}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	addr := flag.String("addr", "127.0.0.1:4301", "listen address")
	dist := flag.String("dist", "frontend/dist", "built frontend")
	flag.Parse()

	s := &shell{subs: map[chan event]struct{}{}}
	s.newApp("")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /__shell/events", s.events)
	mux.HandleFunc("POST /__shell/rpc/{method}", s.rpc)
	// Test controls (never reachable from the app itself).
	mux.HandleFunc("POST /__shell/reset", func(w http.ResponseWriter, r *http.Request) {
		var b struct{ DownloadDir string }
		_ = json.NewDecoder(r.Body).Decode(&b)
		s.newApp(b.DownloadDir)
		writeJSON(w, map[string]bool{"ok": true})
	})
	mux.HandleFunc("POST /__shell/pick", func(w http.ResponseWriter, r *http.Request) {
		var b struct{ Path string }
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Path == "" {
			http.Error(w, errors.New("path required").Error(), http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		s.picks = append(s.picks, b.Path)
		s.mu.Unlock()
		writeJSON(w, map[string]bool{"ok": true})
	})
	mux.HandleFunc("GET /__shell/state", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		writeJSON(w, map[string]any{"opened": s.opened, "clipboard": s.clipboard, "downloadDir": s.dlDir})
	})
	mux.Handle("/", http.FileServer(http.Dir(*dist)))

	log.Printf("uishell listening on http://%s (dist=%s)", *addr, *dist)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
