# File Throw (FTU) — Security Audit (Phase 1)

Date: 2026-11-20 · Branch `security/hardening-2026-11-20` (from `dev` @ `b2ebd33`) · Method: manual source review, local tests, local tooling only.
Companion documents: `THREAT_MODEL.md`, `REMEDIATION_PLAN.md`.

## 1. Executive summary

FTU's core security design is sound where it matters most: **peer authentication is correctly enforced on both sides of the QUIC handshake** (Ed25519 key pinned from the token, `RequireAnyClientCert`, TLS 1.3 only), control frames are length-capped before allocation, payload streaming is bounded-memory, and a transfer is only reported successful after size and SHA-256 verification. No Critical findings and no remotely exploitable authentication bypass were found.

The weaknesses are concentrated in **what happens after authentication** and in the **pre-handshake UDP phase**:

- **F-01 (High)** The receiver opens the destination with `os.Create` directly at the final path: it silently truncates/overwrites an existing file (including dotfiles when the download directory falls back to `$HOME`), deletes it if the transfer then fails, follows symlinks, exposes partial data under the final name, and does not handle `..`, dotfiles, ADS, or device names.
- **F-02..F-06 (Medium)** Consent is not bound to a specific offer (shared decision channel, concurrent offers); the token is *obfuscated, not protected* (public static key, unauthenticated cipher, no expiry/version); `PUNCH` is unauthenticated and "first wins"; file names are unsanitised (terminal-escape injection in the CLI prompt, bidi/control spoofing in the GUI); Wails bindings are over-privileged and consent lives in the renderer.
- Eight Low and four Informational items cover STUN validation, timeouts, resource stalls, `log.Fatal` in library code, lifecycle races, and CI/supply-chain hygiene.

Overall: **a good foundation with a thin hardening layer around the filesystem, consent state, and UDP pre-auth phase.** All findings are fixable without changing the architecture; only F-03's full remedy changes the token format (compatibility impact documented in the remediation plan).

### Severity distribution

| Severity | Count | IDs |
|----------|-------|-----|
| Critical | 0 | — |
| High | 1 | F-01 |
| Medium | 5 | F-02, F-03, F-04, F-05, F-06 |
| Low | 8 | F-07 … F-14 |
| Informational | 4 | F-15 … F-18 |
| **Total** | **18** | |

Classification: Confirmed vulnerability: F-01, F-02, F-04, F-05, F-07, F-08, F-09, F-11, F-13 · Likely: F-14 · Architectural risk: F-03, F-06 · Hardening: F-10, F-12, F-17, F-18 · Informational: F-15, F-16.

## 2. Scope and method

In scope: all of `cmd/`, `internal/`, `pkg/`, `main.go`, `tests/ui/shell`, `frontend/src`, `.github/workflows/test.yml`, `go.mod/go.sum`, `frontend/package.json`, `wails.json`, `build/`. Out of scope: Wails/WebKitGTK internals, quic-go internals (read only to confirm defaults), OS-level behaviour on Windows/macOS (not testable on this Linux host; flagged where relevant). No external network tests, no stress tests, no tool installation. No code, config, dependency, or test file was modified; only `docs/security/*` was created.

Graphify was available (`graphify-out/`, built from an older commit) and used for orientation; all findings were verified against current source (~5.3 kLOC Go).

## 3. Architecture and attack surface

See `THREAT_MODEL.md` §1–3. Key facts verified in code:

- One UDP socket (`transport/udp-socket.go`) is shared by STUN, hole punching, and QUIC. 7 MiB buffers.
- Roles: `bytes.Compare(myPub, peerPub) > 0` → QUIC listener (`quic-setup.go:69`).
- Streams: bidi control stream per file; uni data stream per file; quic-go defaults 100/100 (`internal/protocol/params.go:40,43`).
- Wails binds the entire `*App` (`pkg/ui/run.go`): 16 exported methods reachable from JS.

## 4. Findings

Severity rubric: Critical/High/Medium/Low/Informational by realistic impact × preconditions. CVSS 4.0 vectors are given only for F-01 and F-02; **scores were not computed** (no calculator available offline) and should be treated as unscored estimates.

---

### F-01 — Receiver clobbers, deletes, and exposes files at the final path
- **Severity:** High · **Class:** Confirmed vulnerability · **Confidence:** High
- **Component:** `internal/transfer` (ReceiveFile), `pkg/ui` receiver, CLI receiver
- **Files:** `internal/transfer/transfer.go:116,125-128,143-162`; `pkg/ui/receiver.go:87-93`; `internal/protocol/receiver.go:80-86`; `pkg/ui/app.go:46-54`
- **Description:** The destination is `filepath.Join(downloadDir, filepath.Base(name))` and is opened with `os.Create` (O_CREATE|O_TRUNC, mode 0666&~umask) **before** the data stream is even accepted. Consequences:
  1. An existing file with the same name is truncated immediately, overwritten, and — if the transfer fails for any reason (hash mismatch, short stream, peer disconnects) — **deleted** by `os.Remove(destPath)`. The consent prompt shows only name and size; it never says the name already exists.
  2. `downloadDir` falls back to `$HOME` when `~/Downloads` does not exist (`app.go:51`) and to `.` when the user lookup fails; a peer can then target `.bashrc`, `.profile`, `.zshrc`, … (`filepath.Base` keeps dotfiles). CLI mode writes into the current working directory.
  3. `os.Create` follows a pre-planted symlink (T9) and writes through it; `os.Remove` then removes only the link.
  4. Partial/unverified content is visible under the final name for the whole transfer; after a crash/kill/power loss a truncated file stays under its real name with no marker (cleanup only runs on handled error paths).
  5. `filepath.Base("..") == ".."` passes the guard (verified locally); on Windows reserved device names (`CON`, `NUL`, `COM1`), trailing dots/spaces and NTFS alternate data streams (`name:stream`) are not handled (**unverified on Windows**).
  6. File mode is world-readable (0644 typical) for received files.
- **Evidence:** Local check of `filepath.Base` outputs: `".."→".."`, `"a/.."→".."`, `"x/"→"x"`, `"..\\x"→"..\\x"` (Linux). Code trace above.
- **Preconditions:** Authenticated peer (T5) and one user click on "Accept" for a name that collides or targets a dotfile; symlink variant needs local write access to the download dir.
- **Impact:** Data loss of an existing file; overwrite of shell-init files → code execution at next login when the download dir is `$HOME`; partial/corrupt file left under a trusted name after a crash.
- **Exploitability:** Easy for a malicious peer; consent reduces but does not remove it because the prompt does not disclose overwrite and dotfile names look innocuous in a small dialog.
- **Existing mitigations:** `filepath.Base` strips separators; consent required; hash verified before success; partial file removed on handled errors.
- **Recommendation:** Write to an `O_EXCL|O_NOFOLLOW`-style temp file (0600) in the target dir, verify, then atomically `rename`/`link` to a **non-colliding** final name (auto-suffix; never overwrite); validate/normalise the name (reject `..`, empty, control chars, NUL, reserved device names/ADS/trailing dot-space, dotfile policy, length ≤ 255 bytes); do not create anything until the data stream is accepted; refuse hidden/dot names in `$HOME`.
- **Suggested tests:** Pre-existing file survives a failed transfer; never overwritten on success (suffix applied); symlink at destination not followed; `..`, `.bashrc`, `CON`, `a:b`, NUL, 300-byte names rejected/sanitised; kill-mid-transfer leaves no file under the final name; mode 0600/0644 per decision.
- **CVSS 4.0 (estimate, unscored):** `AV:N/AC:L/AT:N/PR:L/UI:A/VC:N/VI:H/VA:L/SC:N/SI:N/SA:N` (authenticated peer, user interaction).

### F-02 — Consent not bound to an offer; shared decision channel
- **Severity:** Medium · **Class:** Confirmed · **Confidence:** High
- **Files:** `pkg/ui/receiver.go:39-69,82-100`; `pkg/ui/app.go:430-446`; `internal/transfer/transfer.go:125`
- **Description:** One `incomingDecisionCh` (cap 1) and one `offerPending` bool serve all offers. `startReceiverLoop` spawns a goroutine per control stream with no limit on concurrent undecided offers (≤100 by QUIC). Each new offer drains the channel and re-arms `offerPending`, but the earlier goroutines keep waiting on the same channel. `RespondIncomingTransfer(accept bool)` carries no offer identifier. A peer that opens offer A (innocuous) then offer B right before the user clicks can have the click consumed by *either* goroutine (non-deterministic). Further, `ReceiveFile` calls `AcceptUniStream` and takes **any** next uni stream, so the data stream is not bound to the control stream that was approved.
- **Impact:** A file the user never saw (or saw earlier and rejected) can be accepted; stale/duplicate clicks apply to the next offer.
- **Preconditions:** Authenticated peer, user interaction.
- **Existing mitigations:** 120 s timeout; hash/size checks; peer already authenticated.
- **Recommendation:** Per-offer ID generated by the backend, emitted in `transfer:incoming`, required by `RespondIncomingTransfer(id, accept)`; serialise offers (reject new offers with 429/409 while one is pending); bind the data stream to the offer (e.g. require a stream ID announced in the control exchange or serialise one transfer at a time per connection).
- **Tests:** Two concurrent offers → exactly one decision channel/ID; mismatched ID rejected; stale click after timeout rejected; second offer while pending is refused.
- **CVSS 4.0 (estimate, unscored):** `AV:N/AC:H/AT:N/PR:L/UI:A/VC:N/VI:L/VA:L/...`

### F-03 — Token is obfuscated, not protected (static public key, no authentication, expiry, or version)
- **Severity:** Medium · **Class:** Architectural risk · **Confidence:** High
- **Files:** `internal/discovery/signalingToken.go:21,63-84,87-110,136-153`
- **Description:**
  - **Confidentiality:** the ChaCha20 key is a constant in the source and binary; hex `6368616e…7365637265 74` decodes to the ASCII placeholder `change this password to a secret`. Anyone with a token (or the open-source repo) can decrypt/forge tokens. The token should be classified *encoded*, not *secret*. The embedded key is not a credential leak per se.
  - **Integrity/authenticity:** `NewUnauthenticatedCipher` is a raw stream cipher: no MAC, malleable bit-for-bit. Any 56- or 74-byte base64url string "decodes" into some pubkey and addresses; there is no checksum, magic, version, or length-of-address validation.
  - **Expiry/replay:** a token never expires; `a.myToken` persists until `Disconnect`. Replay by a third party does not yield impersonation (both sides pin the *other* token's key, so the attacker needs both the victim's expectation and the private key), but a swapped token gives the swapper a fully authenticated session.
  - **Trust root:** an attacker able to modify the token *in transit between users* (T4) replaces pubkey+address with their own and becomes the pinned peer on both sides (they can even relay between the two). The design offers no user-visible verification (no fingerprint/SAS).
  - **Privacy:** token contains the private LAN IPv4, public IPv4 (via STUN) and public IPv6 (`encodeTo62bytes`).
- **Impact:** MITM of the pairing if the token channel is not integrity-protected; tampering undetected; format cannot evolve safely.
- **Existing mitigations:** Mutual pinning (both tokens needed), per-session ephemeral key, same-key check (`ErrSameAdress`).
- **Recommendation (document only in Phase 1):** keep tokens as non-secret *public identifiers*; add magic+version+CRC or an AEAD with a **public, versioned** key purely for corruption detection; display a short fingerprint (SAS) of the pubkey for out-of-band comparison; optional expiry timestamp. Do **not** claim confidentiality. Do not mechanically swap the cipher; evaluate purpose first. Format change requires a compatibility strategy (see remediation plan).
- **Tests:** tampered token rejected; version/magic mismatch rejected; expired token rejected (if added); round trip.

### F-04 — Unauthenticated `PUNCH`, first-valid-packet wins, source not bound to token candidates
- **Severity:** Medium · **Class:** Confirmed · **Confidence:** High
- **Files:** `internal/connectivity/holePunch.go:87-107,109-146`; consumer `pkg/ui/app.go:~204-232`, `quic-setup.go:77-91`
- **Description:** The probe payload is `0x01 || pubkey` and the pubkey is public (it is in the token and on the wire in clear). `listenPeer` accepts the first datagram that matches from **any source address** and returns that address as the peer's. When this peer is the **dialer**, `quic.Dial` goes to that address with a 15 s timeout and **no retry** with the legitimate candidates; the attacker (T2/T3, or any host that has seen one probe) cannot authenticate, so pairing fails (denial of pairing, repeatable). When the peer is the listener the address is unused.
- **Impact:** Targeted denial of connection; misdirected outbound QUIC Initials to an attacker-chosen address (small information leak: the victim's QUIC source address/port).
- **Recommendation:** Prefer addresses that appear in the token; on dial failure fall back to the next verified candidate; optionally authenticate PUNCH (e.g. include a nonce signed by the ephemeral key, which does not change the trust model); rate-limit.
- **Tests:** Forged PUNCH from a stray address followed by a genuine one still connects; dial failure triggers fallback.

### F-05 — File name not validated: terminal-escape injection and UI spoofing
- **Severity:** Medium · **Class:** Confirmed · **Confidence:** High
- **Files:** `internal/protocol/message.go:62-72` (`DecodeRequest`), `pkg/cli/input/input.go:86-87`, `internal/protocol/receiver.go:77,99`, `pkg/ui/receiver.go:52-56`
- **Description:** `DecodeRequest` validates size and hash but not `FileName` (no length bound beyond the 64 KiB frame, no control/bidi/format characters, no NUL). The CLI prompt prints it with `'%s'`, so ANSI/CSI sequences, `\r`, and newlines let a peer forge the prompt (e.g. rewrite the displayed name and size, or hide the question). Logs interpolate the same string (log injection). In the GUI, React escapes HTML (no XSS), but Unicode bidi overrides (`U+202E`) and zero-width/long names can disguise extensions (`invoice‮gpj.exe`).
- **Impact:** Consent made on false information.
- **Recommendation:** Validate in `DecodeRequest` (backend): reject/replace control (`Cc`), format (`Cf`), line/paragraph separators, NUL; enforce ≤255 bytes and valid UTF-8; print with `%q` in CLI/logs; show the sanitised on-disk name in the consent dialog.
- **Tests:** Fuzz/boundary names (`\x1b[2J`, `\r`, `‮`, 300 bytes, invalid UTF-8, empty).

### F-06 — Over-privileged Wails bindings; consent and paths decided in the renderer
- **Severity:** Medium · **Class:** Architectural risk · **Confidence:** Medium (no exploitable sink found today)
- **Files:** `pkg/ui/run.go` (Bind), `pkg/ui/app.go:292,331,430,448-486`
- **Description:** The renderer is treated as trusted for:
  - `InspectFile(path)` / `SendSelectedFile(path)`: any readable path; there is no binding to the path chosen in the native dialog or dropped file. A compromised renderer could connect to an attacker token (`ConnectPeer`) and send `~/.ssh/id_ed25519`.
  - `RespondIncomingTransfer(accept)`: the consent decision is made by JS; a compromised renderer can auto-accept.
  - `OpenFolder(path)`: for a non-existent path it passes the raw string to `xdg-open` / `open` / `explorer` — a URL or URI scheme string is opened by the OS handler (no `--` separator; Windows `explorer` URI handling **unverified**).
  - `GetClipboardText()` returns clipboard contents to JS.
  Grep over `frontend/src` found **no** `dangerouslySetInnerHTML`, `innerHTML`, `eval`, or untrusted `href`; `index.html` has no CSP; `devMock.ts` ships in the bundle.
- **Impact:** Any future XSS/dependency compromise in the renderer becomes arbitrary file exfiltration and consent bypass (T11).
- **Recommendation:** Backend remembers the paths it obtained from `SelectFileNative`/drop events and accepts only those handles; make `OpenFolder` accept only `downloadDir` or paths the backend produced, and insert `--`; consider a native dialog (`wailsruntime.MessageDialog`) for consent; add a CSP (`default-src 'self'`); tree-shake the dev mock.
- **Tests:** `SendSelectedFile` with an unselected path is rejected; `OpenFolder("https://…")` rejected.

### F-07 — Hole-punch timeout is per read, not per attempt
- **Severity:** Low · **Class:** Confirmed · **Confidence:** High
- **File:** `internal/connectivity/holePunch.go:114-117`
- **Description:** The read deadline is re-armed (`Now()+Timeout`) on every loop iteration, including after each junk datagram. Any off-path sender that delivers one packet per <60 s keeps `listenPeer` alive forever: `done` never closes and the UI sits in `hole_punching` indefinitely. Additionally, any non-timeout, non-closed read error `continue`s with no back-off (possible hot loop).
- **Recommendation:** Compute one absolute deadline at start; add back-off on repeated errors.
- **Tests:** Junk packets every 100 ms for > Timeout must still yield a timeout result.

### F-08 — STUN response not validated
- **Severity:** Low · **Class:** Confirmed · **Confidence:** High
- **File:** `internal/discovery/stun.go:44-60,62-95`
- **Description:** The 12-byte transaction ID is generated but never compared; message type (0x0101), magic cookie, length field, and source address are not checked; the first datagram arriving on the shared socket within 5 s is parsed. An off-path attacker who guesses the ephemeral port (or any LAN host) can inject a forged mapped address, which is then advertised in the token. Also only `DefaultSTUNServers[0]` is ever used and the parser requires XOR-MAPPED-ADDRESS to be the *first* attribute, so servers that lead with other attributes degrade silently to the 0.0.0.0 fallback. The parser itself is bounds-safe (fuzz target `FuzzParseXORMappedAddress`).
- **Impact:** Pairing denial / wrong public candidate; no confidentiality impact (pinned keys protect the session).
- **Recommendation:** Validate type, cookie, txid, length, and source = queried server; loop until valid or deadline; iterate attributes.
- **Tests:** Forged reply with wrong txid ignored; correct reply after a forged one is accepted.

### F-09 — No deadlines/limits on peer-driven waits; empty file left behind
- **Severity:** Low · **Class:** Confirmed · **Confidence:** High
- **Files:** `internal/transfer/transfer.go:116-127`; `pkg/ui/receiver.go:24,39-100`; `internal/protocol/message.go:117-129`
- **Description:** After consent `ReceiveFile` creates the file, then waits on `AcceptUniStream(context.Background())`. A peer that accepts-then-never-opens the data stream (or writes 1 byte/min) parks the goroutine and file handle indefinitely (keep-alive pings keep the connection alive), leaving an empty/partial file under the final name (compounds F-01). Control streams have no read deadline: a peer can open up to 100 idle control streams, each holding a goroutine.
- **Recommendation:** Context with deadline for `AcceptUniStream`; idle/stall deadline on the data stream; read deadline on control frames; create the temp file only after the stream is accepted; cap concurrent offers.
- **Tests:** Accept-without-data fails within the deadline and leaves no file.

### F-10 — mTLS pinning hardening (resumption, ALPN, certificate profile)
- **Severity:** Low · **Class:** Hardening · **Confidence:** Medium
- **File:** `internal/transport/quic-setup.go:20-56`
- **Description:** Pinning is implemented in `VerifyPeerCertificate`, which Go does **not** invoke on resumed connections (doc: *"This callback is not invoked on resumed connections"*, `crypto/tls/common.go:660`). Today there is no client session cache and each listener builds a fresh config, so no exploitable path exists, but any future cache/ticket sharing would silently skip pinning. `VerifyConnection` runs on every handshake. Additional: ALPN is the sample value `quic-echo-example`; the certificate has no validity period/subject (adequate for key-pinning but non-standard); the cert key is the identity key itself (no separation).
- **Recommendation:** Move the pin check into `VerifyConnection` (also assert `HandshakeComplete`, TLS 1.3, ALPN); use a product ALPN (`ftu/1`) — protocol/ALPN change is a compatibility break, document; set `NotBefore/NotAfter`.
- **Tests:** Existing key-mismatch tests plus a resumption-style test; ALPN mismatch rejected.

### F-11 — `log.Fatal`/`panic` in library code
- **Severity:** Low · **Class:** Confirmed · **Confidence:** High
- **Files:** `internal/discovery/local.go:15`, `internal/transport/udp-socket.go:14`, `signalingToken.go:57,102,107`
- **Description:** `getLocalIPV4` calls `log.Fatal` when `net.Dial("udp4","8.8.8.8:80")` fails (no default route, e.g. offline); `InitializeUDPSocket` likewise. In the GUI this terminates the whole process from a Wails-bound call. Crypto RNG failures `panic`.
- **Recommendation:** Return errors; surface them as `connection:failed`.
- **Tests:** Injected dial failure returns an error, process survives.

### F-12 — Token candidates are not validated; zero-IPv6 advertised
- **Severity:** Low · **Class:** Hardening · **Confidence:** High
- **Files:** `signalingToken.go:63-84,156-195`; `pkg/ui/app.go:146-170`, `internal/discovery/local.go:33-37`
- **Description:** `GetPeer` accepts port 0, `0.0.0.0`, loopback, multicast, broadcast, and link-local addresses, so a crafted token makes the app send up to 100 probes/candidate to arbitrary hosts (33 bytes each, ≈30 s). `GetMachineAddrs` always returns a non-nil IPv6 `UDPAddr` (nil IP when no IPv6), so every token is the 62-byte form and the nil IPv6 encodes as `::` (on Linux a probe to `[::]:port` loops back to the local socket). `ConnectPeer` only checks `len(peerToken) > 12`.
- **Recommendation:** Reject unusable/special-purpose addresses and port 0; omit IPv6 when absent; trim whitespace/zero-width characters from pasted tokens.

### F-13 — Connection lifecycle races
- **Severity:** Low · **Class:** Confirmed · **Confidence:** Medium
- **Files:** `pkg/ui/app.go:118-140,146-260`
- **Description:** `CancelConnection` "cancels" the blocking read by setting a read deadline of *now*, sleeping 20 ms, then clearing it; `listenPeer` re-arms its own deadline in a loop, so the cancel can be lost and a second `ConnectPeer` then has two goroutines reading the shared socket (stealing packets). The connect goroutine ignores `ctx` after punching (`QuicUpgrade` has no context), and a repeated `ConnectPeer` overwrites `a.quicConn` without closing the previous connection. `a.mu` is held across a `time.Sleep`.
- **Recommendation:** Give `UDPHolePunch` a context/stop channel; reject/replace `ConnectPeer` while connecting/connected; close the old conn.
- **Tests:** Race test: cancel mid-punch then reconnect; `go test -race`.

### F-14 — Integrity checks are fail-open at the edges
- **Severity:** Low · **Class:** Likely vulnerability / hardening · **Confidence:** Medium
- **File:** `internal/transfer/transfer.go:107-165`
- **Description:** `expectedHash != ""` skips verification; today the protocol layer always supplies a validated 64-hex hash, so this is latent. Bytes beyond `expectedSize` on the stream are silently ignored (stream not drained/checked) and the transfer still reports success; no free-space check before accepting (`StatusNotAcceptable` is defined but unused); every receive failure maps to `409 Conflict`. Disk-full during `io.CopyBuffer` is handled (error path), good.
- **Recommendation:** Fail closed on empty/invalid hash inside `ReceiveFile`; read-to-EOF after size and reject extra data; pre-check free space; use distinct status codes.

### F-15 — SHA-256 provides transfer integrity, not origin authenticity (Informational)
The hash is chosen by the sender and sent on the same authenticated channel. It detects corruption/truncation, not a malicious or compromised sender, and does not authenticate the file's provenance. This is by design; documentation should not suggest otherwise.

### F-16 — Privacy and logging (Informational)
Tokens embed private+public IPs; STUN contacts Google (first server only); the UI and logs show the peer's address; logs (`log.Printf`) include peer-controlled file names and hashes; messages mix Portuguese/English. No private keys, tokens, or file contents are logged (verified by grep of all `log.*` and `fmt.Print*` calls); `info.PrintMyToken` prints the token intentionally. Received files are created 0666&~umask.

### F-17 — Supply chain / CI (Informational → Hardening)
`.github/workflows/test.yml`: top-level `permissions: contents: read` (good), triggers `push`/`pull_request`/`workflow_dispatch` (no `pull_request_target`), `npm ci` with lockfiles. Gaps: actions pinned to mutable tags (`@v4`), `govulncheck@latest` unpinned (runs `go run …@latest` from the network), no Dependabot, no SAST/secret-scan job, `npx playwright install --with-deps` (root apt). `go mod verify`: **all modules verified**. Versions in `go.mod` (Go 1.26.4, quic-go 0.61.0, x/crypto 0.55.0, wails 2.15.0) could not be compared against advisory databases offline → **vulnerability status unverified**. No secrets were found by pattern scan of tracked files and history (`BEGIN … PRIVATE KEY`, `AKIA…`, `ghp_…`, `api_key=`); `.gitignore` excludes `.env*` and `.claude/`, `CLAUDE.md`.

### F-18 — Frontend runtime restrictions (Informational → Hardening)
No CSP; dev mock bundled; `window.open(REPO_URL,'_blank','noopener')` is a constant URL. React text rendering is the only sink observed. Frontend deps are caret-ranged but locked (`package-lock.json`); `npm audit` **not run** (requires registry access).

---

### Rejected / false positives (verified)
| Hypothesis | Verdict | Reason |
|------------|---------|--------|
| `InsecureSkipVerify: true` allows any peer | Rejected | The pin in `VerifyPeerCertificate` runs on both sides; TLS 1.3 CertificateVerify proves key possession; tests cover both directions |
| Path traversal via `../../x` | Rejected | `filepath.Base` strips separators (see F-01 for what remains) |
| Unbounded control-frame allocation | Rejected | `length > 64 KiB` checked before `make` |
| Memory grows with file size | Rejected | 64 KiB buffers, `LimitReader`, quic-go flow control |
| TLS < 1.3 / downgrade | Rejected | quic-go sets `MinVersion=TLS13` |
| STUN parser panic on hostile input | Rejected | All lengths checked; fuzz target exists |
| Token replay yields impersonation | Rejected | Impersonation needs the private key; both sides pin |
| XSS via file name in React | Rejected (no sink found) | JSX escaping; no raw HTML sinks |

## 5. Assessments

**Cryptography & key management.** Ed25519 keys come from `crypto/rand`, live in memory only, and are regenerated per `GetMyToken`; there is no persistent identity (positive for privacy, but means pinning is only as good as token delivery — F-03). TLS 1.3 + mTLS verified; no custom primitives. Token cipher is unauthenticated ChaCha20 with a public key (F-03); 96-bit random nonces under one key are collision-safe at this volume. SHA-256 usage is correct and streaming (F-15). Constant-time comparison is not required for public data.

**Authentication / authorisation / consent.** AuthN: strong. AuthZ and consent: the check exists but is not offer-bound (F-02) and lives in the renderer (F-06). Token *possession* is not authentication — correctly, only key possession is.

**Network (STUN/UDP/QUIC).** F-04, F-07, F-08, F-12, F-13. The shared-socket design works but relies on strict ordering (hole punch reader must stop before QUIC starts), which F-13 can violate.

**File transfer / filesystem.** Streaming and size bound are good; the destination handling is the weakest area (F-01, F-05, F-09, F-14).

**Go backend.** `go vet` clean, race tests clean. Issues: `log.Fatal`/`panic` (F-11), unbounded handler goroutines bounded only by QUIC defaults, `context.Background()` waits (F-09), lock held across sleep (F-13).

**Frontend / Wails.** No dangerous sinks, but trust model is too permissive (F-06, F-18).

**Dependencies / CI.** F-17; unverified vulnerability state.

## 6. Test execution (exact commands, host Linux, Go 1.26.4)

| Command | Result |
|---------|--------|
| `git status --porcelain` | clean before branching |
| `go vet ./internal/... ./pkg/cli/... ./tests/...` | pass, no output |
| `go vet ./pkg/ui/ .` | pass, no output |
| `gofmt -l .` (excluding node_modules) | no files listed |
| `go test -count=1 ./internal/... ./pkg/cli/... ./tests/...` | **pass** (connectivity, discovery, e2e 7.6 s, protocol, transfer 3.8 s, transport, cli/input) |
| `go test -race -count=1 ./internal/... ./pkg/cli/... ./tests/...` | **pass** (no races; e2e 10.2 s, transfer 14.4 s) |
| `go test -race -count=1 ./pkg/ui/` | **pass** (34.8 s) |
| `cd frontend && npm test` (Vitest) | **pass** 2 files / 50 tests |
| `go mod verify` | all modules verified |
| Secret pattern scan (`git grep`, history `-S"PRIVATE KEY"`) | no hits |

Not run: `make fuzz` (long-running, not required for audit), Playwright UI e2e (needs browser downloads), `netem` (root), `govulncheck`/`gosec`/`staticcheck` (not installed; installing was not authorised), `npm audit` (network). A passing suite does **not** show F-01…F-14 are absent: none of them is covered by an existing test (the suite tests pinning, framing, integrity, and bounded behaviour only).

## 7. Limitations and unverified areas

- Windows/macOS behaviour (reserved names, ADS, `explorer`/`open` URI handling, WebView2 CSP behaviour) is **unverified**.
- Dependency CVE status for Go modules and npm packages is **unverified**.
- No live network testing (STUN spoofing, off-path injection) was performed; F-04/F-07/F-08 are code-trace findings supported by unit-level reasoning.
- quic-go internals were read only for defaults (streams, TLS min version).
- `tests/ui/shell` (test-only HTTP shell, documented to bind `127.0.0.1`) was only skimmed.

## 8. Prioritised roadmap

1. F-01 (temp file + atomic no-clobber rename + name validation) — also resolves part of F-09 and F-14.
2. F-02 + F-05 (offer IDs/serialisation; backend name sanitisation; `%q` prompt).
3. F-04/F-07/F-08 (UDP pre-auth hardening) and F-13.
4. F-06 (handle-based paths, restricted `OpenFolder`, CSP).
5. F-03 (token format v2 with magic/version/checksum + fingerprint display; compatibility plan).
6. F-09–F-12, F-14 hygiene; F-17/F-18 CI and frontend config.

## 9. Overall assessment

Authentication and transport security are implemented correctly and tested; the product meets its stated "direct P2P, bounded memory, consent, integrity" invariants in the normal path. The most important gap is that **a one-click consent grants an authenticated peer more filesystem power than the dialog communicates** (F-01, F-02, F-05). Fixing those, plus the UDP pre-auth issues, brings the app to a solid baseline. This audit is not a guarantee of security: unverified areas above remain.
