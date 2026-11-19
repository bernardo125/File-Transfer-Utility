# Testing

All tests use the real transport (UDP sockets + `quic-go` + the production `QuicUpgrade`/hole punch).
Nothing needs the internet or privileges, except the opt-in `netem` layer.

| Layer | Command | Notes |
|---|---|---|
| Fast (unit / protocol) | `make test-fast` | `-short`: skips the heavy and impaired-network cases |
| Integration + E2E | `make test` | real peers on loopback, ~1 min |
| Race detector | `make race` | |
| Fuzzing | `make fuzz FUZZTIME=30s` | 4 targets: control frames, request decoding, token parsing, STUN parsing |
| Benchmarks | `make bench` | reports allocations, never asserts on speed |
| Static checks | `make lint`, `make vuln` | gofmt, `go vet`, govulncheck |
| Kernel impairment | `make netem` | needs Linux, root and `tc`; not run in CI by default |

Packages are listed explicitly (`./internal/... ./pkg/cli/...`) because `pkg/ui` needs Wails/GTK and a built frontend.

## Where things are
- `internal/protocol`: framing, request validation, fuzzers; `protocol_integration_test.go` runs the real sender and receiver (reject, hostile name, hash lie, bad metadata, disconnects, several files per session).
- `internal/transfer`: integrity at boundary sizes (0, 1, 64KiB-1, 64KiB, 64KiB+1, MiBs), truncation, over-send, mid-transfer disconnect, isolation, bounded memory.
- `internal/transport`: mutual authentication (impostor on either side), unavailable peer, timeouts.
- `internal/connectivity`: hole punch (success, stale candidate, wrong key, junk datagrams, own token, closed socket, probe shutdown).
- `internal/discovery`: token round trip, malformed tokens, STUN parsing, fuzzers.
- `internal/e2e`: token -> `GetPeer` -> hole punch -> QUIC -> transfer, both directions; seeded loss/duplication/reordering/latency; simultaneous sessions; connect/disconnect cycles with goroutine check.
- `internal/testutil`: peer/pair helpers and a seeded UDP impairment proxy (forwards datagrams only; it provides no discovery or relay).

## Fuzz corpus
Seed inputs live in the `f.Add` calls. When the fuzzer finds a failure, Go writes it to `testdata/fuzz/<Target>/` next to the test; commit that file and it runs as a regression test on every `go test`.

## Reproducing impaired-network failures
`testutil.Impairment.Seed` fixes the loss/duplication/reorder pattern per direction. Datagram *timing* is still real, so a seed reproduces the fault pattern, not the exact interleaving.

## netem layer
`go test -tags netem` applies `tc qdisc ... netem` to `lo` (loss, delay/jitter, duplication, reordering) and removes it on cleanup. It skips itself unless run as root with `tc` installed. It was written and compiles, but could only be verified in an environment with `CAP_NET_ADMIN`.

## Not covered (and why)
- **Real NAT traversal / different networks / firewalls.** Needs separate network namespaces or machines. Tests cover the same-host path through the real pipeline, stale candidates and wrong-key rejection, not NAT mapping behaviour.
- **STUN against a real server.** Parsing is tested; the network call is not (no internet in tests).
- **Reusing one UDP socket for a second session.** After QUIC has run on a socket, a second hole punch on it times out. The product treats every session as new (new socket/token); this is a limitation, not tested as working.
- **Sender that over-sends.** The receiver stops at the announced size; the over-sending sender stays blocked by flow control.
- **Token confidentiality/integrity.** The token is ChaCha20 without authentication under a fixed secret in the binary: it is obfuscation, a tampered token decodes to different addresses undetected (the peer key check at QUIC still protects the session). Not asserted by tests because it is a design property.
- **Multi-GB transfers.** Memory is checked on 96 MiB with streaming sampled during the copy.
- **UI (`pkg/ui`).**
