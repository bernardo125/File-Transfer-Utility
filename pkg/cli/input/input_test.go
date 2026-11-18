package input

import (
	"bufio"
	"io"
	"strings"
	"testing"
	"time"
)

func rd(s string) *bufio.Reader { return bufio.NewReader(strings.NewReader(s)) }

// within fails the test instead of hanging the whole suite when a prompt never returns.
func within[T any](t *testing.T, f func() (T, error)) (T, error) {
	t.Helper()
	type r struct {
		v   T
		err error
	}
	ch := make(chan r, 1)
	go func() { v, err := f(); ch <- r{v, err} }()
	select {
	case x := <-ch:
		return x.v, x.err
	case <-time.After(3 * time.Second):
		t.Fatal("prompt did not return (busy loop on closed input?)")
		var zero T
		return zero, nil
	}
}

// Detects: short/garbage tokens passing as peer tokens and whitespace breaking a pasted token.
func TestAskPeerToken(t *testing.T) {
	long := "abcdefghijklmnopqrstuvwxyz0123456789"
	tests := []struct {
		name, in, want string
	}{
		{"valid", long + "\n", long},
		{"trims whitespace and CRLF", "  " + long + " \r\n", long},
		{"rejects short tokens then accepts", "short\n123456789012\n" + long + "\n", long}, // 12 chars is rejected
		{"rejects blank lines", "\n\n" + long + "\n", long},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := within(t, func() (string, error) { return AskPeerToken(rd(tc.in)) })
			if err != nil || got != tc.want {
				t.Fatalf("got %q err=%v, want %q", got, err, tc.want)
			}
		})
	}
}

// Detects: an infinite busy loop when stdin is closed (Ctrl+D / piped input ended).
func TestAskPeerTokenReturnsErrorWhenInputEnds(t *testing.T) {
	if got, err := within(t, func() (string, error) { return AskPeerToken(rd("")) }); err == nil {
		t.Fatalf("returned %q without error on closed input", got)
	}
}

// Detects: Windows "Copy as path" quoting leaking into the path, or spaces being mangled.
func TestAskFilePath(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"plain", "/tmp/a.bin\n", "/tmp/a.bin"},
		{"spaces kept inside", "/tmp/my file.bin\n", "/tmp/my file.bin"},
		{"trims", "  /tmp/a.bin \r\n", "/tmp/a.bin"},
		{"windows quoted path", "\"C:\\Users\\me\\My File.txt\"\r\n", "C:\\Users\\me\\My File.txt"},
		{"exit keyword untouched", "exit\n", "exit"},
		{"blank line yields empty path", "\n", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := within(t, func() (string, error) { return AskFilePath(rd(tc.in)) })
			if err != nil || got != tc.want {
				t.Fatalf("got %q err=%v, want %q", got, err, tc.want)
			}
		})
	}
	if _, err := within(t, func() (string, error) { return AskFilePath(rd("")) }); err != io.EOF {
		t.Fatalf("closed input error = %v, want io.EOF", err)
	}
}

// Only an explicit Y (any case, surrounding space ok) means yes. Detects an accept prompt that
// treats junk/empty input as consent to receive a file.
func TestYesNoPrompts(t *testing.T) {
	prompts := map[string]func(*bufio.Reader) (bool, error){
		"AskSendAnother":       AskSendAnother,
		"AskContinueReceiving": AskContinueReceiving,
		"AskAcceptTransfer":    func(r *bufio.Reader) (bool, error) { return AskAcceptTransfer(r, "a.bin", 1<<20) },
	}
	answers := []struct {
		in   string
		want bool
	}{
		{"Y\n", true}, {"y\n", true}, {" y \r\n", true},
		{"N\n", false}, {"n\n", false}, {"\n", false}, {"yes\n", false}, {"1\n", false}, {"ys\n", false},
	}
	for name, ask := range prompts {
		for _, a := range answers {
			t.Run(name+"/"+strings.TrimSpace(a.in), func(t *testing.T) {
				got, err := within(t, func() (bool, error) { return ask(rd(a.in)) })
				if err != nil || got != a.want {
					t.Fatalf("input %q: got %v err=%v, want %v", a.in, got, err, a.want)
				}
			})
		}
		t.Run(name+"/closed input", func(t *testing.T) {
			if got, err := within(t, func() (bool, error) { return ask(rd("")) }); err == nil || got {
				t.Fatalf("closed input must be an error and never consent: got %v err=%v", got, err)
			}
		})
	}
}

// Detects: menu accepting garbage, or not tolerating surrounding whitespace / CRLF terminals.
func TestAskMenuOption(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"valid", "1\n", "1"},
		{"each valid option", "3\n", "3"},
		{"skips invalid until valid", "9\nabc\n\n0\n2\n", "2"},
		{"trims spaces and CRLF", "  2 \r\n", "2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := within(t, func() (string, error) { return AskMenuOption(rd(tc.in)) })
			if err != nil || got != tc.want {
				t.Fatalf("got %q err=%v, want %q", got, err, tc.want)
			}
		})
	}
}

func TestAskMenuOptionReturnsErrorWhenInputEnds(t *testing.T) {
	for _, in := range []string{"", "9\n", "1"} { // "1" has no newline: incomplete line
		if got, err := within(t, func() (string, error) { return AskMenuOption(rd(in)) }); err == nil {
			t.Errorf("input %q: returned %q without error", in, got)
		}
	}
}
