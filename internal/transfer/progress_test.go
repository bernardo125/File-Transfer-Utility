package transfer

import (
	"bytes"
	"io"
	"testing"
)

func TestWithProgressReportsCumulativeBytesAndPreservesData(t *testing.T) {
	payload := bytes.Repeat([]byte("ftu"), 50_000) // 150 KB, spans several buffer reads
	var reports []int64

	r := withProgress(bytes.NewReader(payload), func(done int64) { reports = append(reports, done) })

	var out bytes.Buffer
	n, err := io.CopyBuffer(&out, r, make([]byte, transferBufferSize))
	if err != nil {
		t.Fatalf("copy failed: %v", err)
	}
	if n != int64(len(payload)) || !bytes.Equal(out.Bytes(), payload) {
		t.Fatalf("payload altered: copied %d of %d bytes", n, len(payload))
	}
	if len(reports) < 2 {
		t.Fatalf("expected multiple progress reports, got %d", len(reports))
	}
	for i := 1; i < len(reports); i++ {
		if reports[i] <= reports[i-1] {
			t.Fatalf("progress not monotonic: %v", reports)
		}
	}
	if last := reports[len(reports)-1]; last != int64(len(payload)) {
		t.Fatalf("final report %d, want %d", last, len(payload))
	}
}

func TestWithProgressNilCallbackReturnsOriginalReader(t *testing.T) {
	src := bytes.NewReader([]byte("x"))
	if withProgress(src, nil) != io.Reader(src) {
		t.Fatal("nil callback should not wrap the reader")
	}
}
