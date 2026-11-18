package protocol

import (
	"bytes"
	"encoding/json"
	"testing"
)

func BenchmarkControlFrameRoundTrip(b *testing.B) {
	payload, _ := json.Marshal(Request{FileName: "archive-2026-11.tar.zst", FileSize: 1 << 33, FileHash: validHash})
	var buf bytes.Buffer
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := sendControlMessage(payload, &buf); err != nil {
			b.Fatal(err)
		}
		if _, err := receiveControlMessage(&buf); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeRequest(b *testing.B) {
	payload, _ := json.Marshal(Request{FileName: "archive-2026-11.tar.zst", FileSize: 1 << 33, FileHash: validHash})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := DecodeRequest(payload); err != nil {
			b.Fatal(err)
		}
	}
}
