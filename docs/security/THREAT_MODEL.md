# File Throw (FTU) — Threat Model

Audit date: 2026-11-20 · Branch: `security/hardening-2026-11-20` · Base commit: `b2ebd33`
Status: Phase 1 (audit only). Source code, `go.mod`, and tests are the source of truth; README claims were treated as context, not proof.

## 1. System overview

FTU is a direct peer-to-peer desktop file-transfer app (Go backend, Wails shell, React 19 UI, plus a `--cli` mode).

Flow as implemented:

1. `GetMyToken` / `CreateKey` (`pkg/ui/app.go:~75`, `internal/discovery/signalingToken.go:31`) opens **one shared UDP socket**, runs a single STUN binding request against `stun.l.google.com:19302`, collects the local IPv4 and an IPv6 source address, and generates a **fresh in-memory Ed25519 keypair per session** (never persisted).
2. The token is `base64url( nonce(12) || ChaCha20(static key, nonce, [pubkey32 | localIPv4:port | stunIPv4:port | optional IPv6:port]) )`. The ChaCha20 key is a constant embedded in the source (`signalingToken.go:21`, hex of the ASCII text `change this password to a secret`).
3. Each user pastes the **other** user's token (out of band). `GetPeer` decodes it into candidate addresses and the peer's Ed25519 public key.
4. `UDPHolePunch` sends 33-byte `PUNCH(0x01) || ownPubKey` datagrams to all candidates and reads the shared socket; the first datagram whose bytes 1..33 equal the expected peer public key supplies the remote address.
5. `QuicUpgrade` picks roles deterministically (`bytes.Compare(myPub, peerPub) > 0` → listener, else dialer) and runs a QUIC/TLS 1.3 handshake on the same socket. Both sides present a self-signed Ed25519 certificate and pin the peer's raw public key from the token (`VerifyPeerCertificate`, `InsecureSkipVerify: true`, `ClientAuth: RequireAnyClientCert`).
6. Per file: sender opens a bidirectional **control stream**, sends length-prefixed (u32, max 64 KiB) JSON `{FileName, FileSize, FileHash}`; the receiver asks the user, answers `{Status}`; the sender then writes the payload on a **unidirectional data stream** (64 KiB buffer); the receiver hashes while writing and answers `200` or `409`.

No central server, relay, or upload service exists. External contacts: public STUN servers (only the first is used).

## 2. Assets

| ID | Asset | Why it matters |
|----|-------|----------------|
| A1 | Files on the receiver's disk (esp. pre-existing files in the download dir, dotfiles when the dir is `$HOME`) | Integrity/availability of user data |
| A2 | Files on the sender's disk | Confidentiality; only the *selected* file should leave |
| A3 | Ed25519 private key (memory only, per session) | Peer identity |
| A4 | Session token (pubkey + private LAN IP + public IP + IPv6) | Privacy; routing; the trust root of the pairing |
| A5 | User consent decision | Gate for writing to disk |
| A6 | Host availability (CPU, sockets, disk, FDs) | |
| A7 | User's trust in what the UI/terminal displays (file name, size) | Consent is only meaningful if the prompt is truthful |

## 3. Trust boundaries and attack surface

| Boundary | Crossing data | Controls found |
|----------|---------------|----------------|
| TB1 Internet/LAN → shared UDP socket | Arbitrary datagrams to the ephemeral port: PUNCH probes, STUN replies, QUIC Initial packets | 33-byte PUNCH key match; STUN parser bounds checks; QUIC stack |
| TB2 User ↔ peer (out-of-band token exchange) | Token string | Mutual pinning of the public key in the token; **no integrity/authenticity on the token itself** |
| TB3 Authenticated peer → receiver | Control JSON, data stream | 64 KiB frame cap before alloc; hash regex; size ≥ 0; `LimitReader`; user consent |
| TB4 STUN server → client | Binding response | Length checks only (no txid/cookie/source validation) |
| TB5 Webview (React) → Go (Wails bindings) | `App` methods | None beyond Go types; renderer is trusted for consent and file paths |
| TB6 Local processes → files/sockets | Download dir, token on clipboard, stdout | OS permissions only |
| TB7 Supply chain | Go modules, npm packages, CI actions | `go.sum`, `package-lock.json`, `contents: read` permission |

Entry points: UDP socket (single port, all protocols multiplexed); QUIC streams (≤100 bidi + ≤100 uni, quic-go defaults); Wails methods `ConnectPeer`, `SendSelectedFile`, `InspectFile`, `RespondIncomingTransfer`, `OpenFolder`, `CopyToClipboard`, `GetClipboardText`, `SelectDownloadDirectory`, `Disconnect`, `CancelConnection`; CLI stdin.

## 4. Attacker profiles

| ID | Profile | Capabilities assumed |
|----|---------|----------------------|
| T1 | Remote with IP:port only | Send arbitrary UDP to the socket; no token |
| T2 | Holds a victim's token (not the key) | Knows pubkey + addresses; can forge PUNCH from any source; cannot complete mTLS |
| T3 | Packet modifier / replayer (on-path) | Alters/drops/replays UDP; can read PUNCH probes (they carry the pubkey in clear) |
| T4 | Network MITM of the **token channel** (chat/e-mail) | Can swap the token the user pastes |
| T5 | Malicious/compromised *authenticated* peer | Completes mTLS; full control of control-frame content and stream timing |
| T6 | Malicious sender (bad metadata) | Subset of T5 |
| T7 | Malicious receiver | Authenticated; can lie about status codes |
| T8 | LAN attacker | Sees local candidate, can spoof LAN packets, run STUN-like replies |
| T9 | Malicious local process (same user) | Can pre-create files/symlinks in download dir, read clipboard |
| T10 | Compromised dependency | Code execution in-process |
| T11 | Compromised renderer (XSS / hostile frontend dependency) | Calls any bound `App` method with arbitrary arguments |

## 5. Threat scenarios (prevented vs. remaining)

| # | Scenario | Attacker | Status | Evidence / Finding |
|---|----------|----------|--------|--------------------|
| S1 | Open a QUIC session without the private key | T1,T2 | **Prevented** (both roles verify the pinned key; TLS 1.3 CertificateVerify proves possession; tests `TestListenerRejectsDialerWithUnexpectedKey`, `TestDialerRejectsListenerWithUnexpectedKey`) | `quic-setup.go:20-55` |
| S2 | TLS/QUIC downgrade | T3 | **Prevented** (quic-go forces `MinVersion = TLS13`) | quic-go `internal/handshake/tls_config_go126.go` |
| S3 | Oversized control frame allocation | T5 | **Prevented** (length checked before `make`) | `message.go:117-124` |
| S4 | Path traversal via `../` in file name | T5 | **Prevented for separators** (`filepath.Base`); `..`, dotfiles, ADS, device names not handled | F-01, F-05 |
| S5 | Unbounded write by sender | T5 | **Prevented** (`LimitReader(expectedSize)`) | `transfer.go:138` |
| S6 | Corrupted/truncated file reported OK | T3,T5 | **Prevented** for transport corruption (size + SHA-256 verified before `200`) | `transfer.go:147-165` |
| S7 | Silent overwrite / destruction of an existing file | T5 (+consent) | **Remaining** | F-01 |
| S8 | Consent applied to the wrong offer | T5 | **Remaining** | F-02 |
| S9 | Token swapped in transit → attacker becomes the pinned peer | T4 | **Remaining** (architectural: token has no authenticity) | F-03 |
| S10 | Spoofed PUNCH redirects the dial / stalls pairing | T2,T3 | **Remaining** | F-04, F-07 |
| S11 | Forged STUN reply poisons advertised address | T1,T8 | **Remaining** (needs source-port guess within a 5 s window) | F-08 |
| S12 | Prompt spoofing with control/bidi characters in file name | T5 | **Remaining** | F-05 |
| S13 | Renderer compromise → arbitrary file read/send, URI launch | T11 | **Remaining** (no current sink found) | F-06 |
| S14 | Peer stalls receiver (no data stream, slow stream) | T5 | **Remaining** | F-09 |
| S15 | Resume-time bypass of cert pinning | T1 | Not exploitable today (no client session cache; fresh server config per handshake) but fragile | F-10 |
| S16 | Replay of an old token | T2 | **Prevented from impersonation** (needs both tokens + key); token has no expiry (lives until `Disconnect`) | F-03 |
| S17 | Memory growth with file size | T5 | **Prevented** (fixed 64 KiB buffers; quic-go flow-control windows) | `transfer.go:12,136` |
| S18 | Compromised dependency | T10 | **Remaining**, unverifiable offline | F-17 |

## 6. Assumptions

1. Users exchange tokens over a channel that is at least integrity-protected (not enforced anywhere in the code).
2. The local OS account is trusted; no sandboxing (T9 is mostly out of scope except for download-dir races).
3. The renderer is only the bundled React app (no remote content). Wails serves `frontend/dist` from the embedded asset FS.
4. Public STUN servers behave honestly (single server, no authentication).
5. Peers are *not* mutually trusted after pairing: an authenticated peer can still be hostile (T5/T6), so all peer input must be validated.

## 7. Residual risks (after audit, before Phase 2)

- Token integrity/authenticity and absence of expiry/versioning (F-03). The pairing model is trust-on-token; this is inherent to a no-server design but can be strengthened without a server (e.g. checksum/MAC with versioned format, SAS verification).
- Consent UI lives in the (untrusted) renderer (F-06).
- Receiver file handling is not crash-safe or collision-safe (F-01).
- Off-path UDP injection can deny pairing (F-04, F-07, F-08).
- Dependency/vulnerability status is **unverified** (no scanners installed, network scans not authorized).

## 8. Status after Phase 2

Scenario outcomes (see `SECURITY_AUDIT.md` for commits and tests):

| # | Scenario | Now |
|---|----------|-----|
| S4 | Path traversal / dotfiles / device names | **Prevented** (sanitiser + `os.Root`) |
| S7 | Silent overwrite/destruction of an existing file | **Prevented** (no-clobber publish, temp file) |
| S8 | Consent applied to the wrong offer | **Prevented for consent**; data stream still not bound to the offer (peer-authenticated, hash/size enforced) |
| S9 | Token swapped in transit | **Mitigated, not prevented**: mandatory fingerprint comparison; a user who confirms without comparing is still exposed (option A/PAKE not implemented) |
| S10 | Forged PUNCH redirects the dial | **Mitigated**: source IP must be in the token, dial falls back across candidates; a spoofer using a token IP can still cost one attempt |
| S11 | Forged STUN reply | **Prevented** (txid/source/cookie validated) |
| S12 | Prompt spoofing with control/bidi characters | **Prevented** (validation + `%q`) |
| S13 | Compromised renderer | **Reduced**: no arbitrary send/open, hooks unbound, CSP; consent still decided in the renderer |
| S14 | Peer stalls receiver | **Mitigated** (timeouts, one offer at a time); CLI control read still unbounded |
| S15 | Resume-time bypass of pinning | **Prevented** (`VerifyConnection`) |

Residual risks after Phase 2: pairing authenticity depends on the users' fingerprint comparison; no relay means some NATs cannot pair; consent UI in the renderer; no MOTW/quarantine on received files; ALPN is still the sample value; dependency status re-checked before the first release (see `SECURITY_AUDIT.md`) but must be re-scanned for every release; no Windows/macOS verification; no independent penetration test.
