package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"ftu/internal/transfer"
	"io"
	"strings"
	"testing"
)

const validHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func frame(payload []byte) []byte {
	b := binary.BigEndian.AppendUint32(nil, uint32(len(payload)))
	return append(b, payload...)
}

// Detects: framing regressions (length prefix off by one, short reads treated as complete).
func TestControlFrameRoundTrip(t *testing.T) {
	for _, size := range []int{0, 1, 255, 4096, MaxControlMessageSize} {
		payload := bytes.Repeat([]byte{0xAB}, size)
		var wire bytes.Buffer
		if err := sendControlMessage(payload, &wire); err != nil {
			t.Fatalf("size %d: send: %v", size, err)
		}
		if wire.Len() != 4+size {
			t.Fatalf("size %d: wire length %d, want %d", size, wire.Len(), 4+size)
		}
		got, err := receiveControlMessage(&wire)
		if err != nil || !bytes.Equal(got, payload) {
			t.Fatalf("size %d: round trip mismatch (err=%v, got %d bytes)", size, err, len(got))
		}
	}
}

// Detects: unbounded allocation from a peer-controlled length prefix (OOM / DoS).
func TestReceiveControlMessageRejectsOversizedFrameWithoutReadingBody(t *testing.T) {
	for _, announced := range []uint32{MaxControlMessageSize + 1, 1 << 30, 0xFFFFFFFF} {
		r := bytes.NewReader(binary.BigEndian.AppendUint32(nil, announced))
		_, err := receiveControlMessage(r)
		if !errors.Is(err, ErrFrameTooLarge) {
			t.Errorf("announced %d: err=%v, want ErrFrameTooLarge", announced, err)
		}
	}
	if err := sendControlMessage(make([]byte, MaxControlMessageSize+1), io.Discard); !errors.Is(err, ErrFrameTooLarge) {
		t.Errorf("sender must refuse oversized payload, got %v", err)
	}
}

// Detects: truncated frames being returned as complete/partial messages.
func TestReceiveControlMessageTruncatedInput(t *testing.T) {
	tests := []struct {
		name string
		wire []byte
	}{
		{"empty stream", nil},
		{"partial length prefix", []byte{0, 0}},
		{"prefix but no body", []byte{0, 0, 0, 5}},
		{"body shorter than announced", append([]byte{0, 0, 0, 5}, "abc"...)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := receiveControlMessage(bytes.NewReader(tc.wire))
			if err == nil {
				t.Fatalf("expected error, got payload %q", got)
			}
		})
	}
}

// Detects: metadata the receiver would act on blindly (negative size disables size checks,
// empty/short hash disables integrity verification).
func TestDecodeRequestValidation(t *testing.T) {
	enc := func(r Request) []byte { b, _ := json.Marshal(r); return b }
	tests := []struct {
		name    string
		payload []byte
		wantErr bool
	}{
		{"valid", enc(Request{"a.txt", 10, validHash}), false},
		{"valid empty file", enc(Request{"empty", 0, validHash}), false},
		{"negative size", enc(Request{"a", -1, validHash}), true},
		{"missing hash", enc(Request{"a", 1, ""}), true},
		{"short hash", enc(Request{"a", 1, validHash[:63]}), true},
		{"long hash", enc(Request{"a", 1, validHash + "0"}), true},
		{"non-hex hash", enc(Request{"a", 1, strings.Repeat("g", 64)}), true},
		{"uppercase hash", enc(Request{"a", 1, strings.ToUpper(validHash)}), true},
		{"not json", []byte("{nope"), true},
		{"empty payload", nil, true},
		{"json null", []byte("null"), true},
		{"size overflow", []byte(`{"FileName":"a","FileSize":99999999999999999999,"FileHash":"` + validHash + `"}`), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := DecodeRequest(tc.payload)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v (req=%+v)", err, tc.wantErr, req)
			}
			if err != nil && !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("error %v does not wrap ErrInvalidRequest", err)
			}
		})
	}
}

// Property: any valid request survives encode -> frame -> decode unchanged.
func FuzzRequestRoundTrip(f *testing.F) {
	f.Add("report.pdf", int64(1234), validHash)
	f.Add("", int64(0), validHash)
	f.Add("../../etc/passwd", int64(1<<40), validHash)
	f.Fuzz(func(t *testing.T, name string, size int64, hash string) {
		if len(name) > 1024 {
			t.Skip()
		}
		in := Request{FileName: name, FileSize: size, FileHash: hash}
		payload, err := json.Marshal(in)
		if err != nil {
			t.Skip() // invalid UTF-8 names etc. never reach the wire
		}
		var wire bytes.Buffer
		if err := sendControlMessage(payload, &wire); err != nil {
			t.Skip()
		}
		got, err := receiveControlMessage(&wire)
		if err != nil {
			t.Fatalf("framing lost a valid payload: %v", err)
		}
		out, err := DecodeRequest(got)
		if err != nil {
			// invalid requests must be rejected, valid ones must be preserved exactly
			if size >= 0 && len(hash) == 64 && strings.Trim(hash, "0123456789abcdef") == "" && transfer.ValidatePeerFileName(name) == nil {
				t.Fatalf("valid request rejected: %v", err)
			}
			return
		}
		if out.FileSize != size || out.FileHash != hash {
			t.Fatalf("round trip changed request: %+v -> %+v", in, *out)
		}
	})
}

// Invariant: arbitrary bytes on the wire never panic, never allocate beyond the cap, and
// never yield a "successful" request that fails validation.
func FuzzControlFrameDecoder(f *testing.F) {
	f.Add(frame([]byte(`{"FileName":"a","FileSize":1,"FileHash":"` + validHash + `"}`)))
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	f.Add([]byte{0, 0, 0, 3, '{', '}'})
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, wire []byte) {
		payload, err := receiveControlMessage(bytes.NewReader(wire))
		if err != nil {
			return
		}
		if len(payload) > MaxControlMessageSize {
			t.Fatalf("payload %d exceeds cap", len(payload))
		}
		if req, err := DecodeRequest(payload); err == nil {
			if req.FileSize < 0 || len(req.FileHash) != 64 {
				t.Fatalf("invalid request accepted: %+v", req)
			}
		}
	})
}

// Detects (F-05): control/bidi characters or oversized names reaching the consent prompt.
func TestDecodeRequestRejectsUnsafeFileNames(t *testing.T) {
	for name, fn := range map[string]string{
		"empty":    "",
		"escape":   "a\u001b[2Jb.pdf",
		"newline":  "a\nb",
		"bidi":     "invoice\u202egpj.exe",
		"too long": strings.Repeat("a", 256),
	} {
		t.Run(name, func(t *testing.T) {
			payload, _ := json.Marshal(Request{FileName: fn, FileSize: 1, FileHash: validHash})
			if _, err := DecodeRequest(payload); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("name %q accepted (err=%v)", fn, err)
			}
		})
	}
}
