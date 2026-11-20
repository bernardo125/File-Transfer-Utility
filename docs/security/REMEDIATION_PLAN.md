# File Throw (FTU) — Remediation Plan (Phase 1 proposal)

Status of every item: **Not fixed** (Phase 1 is audit-only). To be updated in Phase 2 with `Fixed / Partially fixed / Accepted risk / Deferred / Not fixed`, commit refs, and validating tests.
Constraints honoured by every item: direct P2P only, no relay/server, bounded-memory streaming, explicit consent, no resumption, no custom crypto, no new dependencies.

| ID | Priority | Complexity | Wire/token compat impact |
|----|----------|------------|--------------------------|
| F-01 | P0 | Medium | None |
| F-02 | P1 | Medium | Frontend↔backend API change (`RespondIncomingTransfer` gains an ID); wire unchanged unless data-stream binding is added (see below) |
| F-05 | P1 | Low | None (stricter validation of an existing field; peers sending control chars are rejected) |
| F-04 | P1 | Medium | Additive punch format (legacy 33-byte probe accepted for one release) |
| F-06 | P1 | Medium | Wails API change (internal) |
| F-03 | P2 | High | **Token format change** + pairing confirmation (A: PAKE needs new dependency, or B: mandatory fingerprint) — needs versioning/migration and approval |
| F-07 | P2 | Low | None |
| F-08 | P2 | Low | None |
| F-09 | P2 | Medium | None |
| F-13 | P2 | Medium | None |
| F-14 | P2 | Low | None (status codes reuse existing `406`) |
| F-10 | P3 | Medium | ALPN/cert profile change is a **wire break** if done; pin-in-`VerifyConnection` alone is not |
| F-11 | P3 | Low | None |
| F-12 | P3 | Low | None |
| F-15 | P4 | Low | Docs only |
| F-16 | P4 | Low | None |
| F-17 | P3 | Low | None |
| F-18 | P3 | Low | None |

---

### F-01 — Safe destination handling (P0, Medium)
- **Proposed change:** Introduce a single helper in `internal/transfer` used by GUI and CLI: `SanitizeFileName(name) (string, error)` and a receive path that (1) accepts the data stream first, (2) opens the download directory once as an `os.Root` (Go ≥ 1.24, stdlib; confines every later operation to that directory and rejects `..`/symlink escapes), (3) creates a temp file through that root with `O_CREATE|O_EXCL|O_WRONLY` and mode 0600 (named `.ftu-<random>.part`), (4) streams+hashes, (5) `Sync`, verifies size and SHA-256, (6) publishes with a **no-clobber** operation: `Root.Link(tmp, final)` then remove the temp; on `EEXIST` pick `name (1).ext` and retry (bounded attempts). **Do not use `os.Rename` for publishing: it silently replaces an existing destination on both Unix and Windows.** Where hard links are unsupported (e.g. FAT/exFAT, some network shares), fall back to creating the final name with `O_EXCL` and copying from the temp file (still no-clobber), (7) `os.Remove` the temp on every error/cancel path, (8) startup sweep of stale `.ftu-*.part`, (9) apply Mark-of-the-Web / quarantine marking to the published file (see M-01). Reject `..`, empty, NUL, control chars, reserved Windows names, trailing dot/space, `:`; length ≤255 bytes; refuse dot-prefixed names (policy: rename `.bashrc` → `bashrc` or reject). Final mode 0644 or 0600 (decide; document).
- **Files:** `internal/transfer/transfer.go`, `pkg/ui/receiver.go`, `internal/protocol/receiver.go`, new helper + tests.
- **Security benefit:** No overwrite/deletion of existing files, no partial data under real names, no symlink following, crash-safe.
- **Potential regressions:** Users expecting overwrite get a suffixed name; temp files on same filesystem required (rename atomicity); the `completed.destPath` must report the *final* chosen name; existing tests that assert the file path must follow the returned path.
- **Compatibility:** None on wire. Behavioural change documented in README.
- **Tests:** pre-existing file preserved on failure and on success; symlink target untouched; kill/crash leaves only `.part`; name table (`..`, `.bashrc`, `CON`, `a:b`, NUL, 300 bytes); disk-full; cancel; race test for two sequential transfers with the same name.

### F-02 — Offer-bound consent (P1, Medium)
- **Proposed change:** Backend creates `offerID` (random 128-bit) per control stream; `transfer:incoming` carries it; `RespondIncomingTransfer(offerID, accept)` validates it against the single active offer; refuse (status `409`/`429`) a second offer while one is pending or transferring; remove the shared channel in favour of a per-offer struct; bind the receive to the approved connection stream by serialising transfers per connection (data stream accepted only inside the approved offer's lifetime; reject unexpected uni streams).
- **Files:** `pkg/ui/app.go`, `pkg/ui/receiver.go`, `pkg/ui/events.go`, `frontend/src/hooks/useTransferController.ts`, `frontend/src/services/*`, `wailsjs` bindings, UI tests.
- **Benefit:** Closes consent confusion/stale-click races.
- **Regressions:** Frontend must echo the ID; concurrent-offer UX (queue vs reject) decision; Wails binding regeneration.
- **Compatibility:** Internal API only.
- **Tests:** two concurrent offers; wrong/missing ID; click after timeout; double accept; `-race`.

### F-05 — File-name validation and safe display (P1, Low)
- **Change:** In `protocol.DecodeRequest` validate UTF-8, ≤255 bytes, no `unicode.Is(Cc|Cf|Zl|Zp)`, non-empty after sanitation; CLI prompt/logs use `%q`; GUI shows the final sanitised on-disk name and a visible warning for executable-type extensions (optional).
- **Files:** `internal/protocol/message.go`, `pkg/cli/input/input.go`, `internal/protocol/receiver.go`, `pkg/ui/receiver.go`.
- **Benefit:** Prompt integrity. **Regressions:** Legitimate names with exotic format characters (e.g. ZWJ in emoji) — decide allow/strip; extend fuzz corpus. **Compat:** peers emitting such names are rejected with `400`.
- **Tests:** table + fuzz on names; CLI prompt output contains no raw ESC.

### F-04 — PUNCH candidate verification and dial fallback (P1, Medium)
- **Change (revised to match ICE-style practice, RFC 8445):** IP filtering alone is only a stop-gap, so the plan is two layers.
  1. *Filter:* accept packets only from a source IP that matches a token candidate (NAT-mapped ports differ, so match IP, not port; document the trade-off); collect all matching sources; on dial failure retry the next.
  2. *Challenge-response (authenticates the path, not just the key):* `PUNCH` becomes a 3-message exchange. Each side sends `PUNCH_REQ(pubkey, nonceA)`; the receiver answers `PUNCH_RSP(pubkey, nonceA, nonceB, sig)` where `sig = Ed25519(priv, "ftu-punch-v2" ‖ nonceA ‖ nonceB ‖ srcAddr ‖ dstAddr)`; a candidate address is *verified* only after a valid signed response to a nonce this side generated recently (replay and forgery fail; the address is bound into the signature). Rate-limit responses (e.g. ≤ 20/s per source) so the socket cannot be used as an amplifier (responses are never larger than 2× the request).
  The format is additive: a 33-byte legacy `PUNCH` is still accepted (filter-only) for one release behind a warning, then removed.
- **Files:** `internal/connectivity/holePunch.go`, `pkg/ui/app.go`, `pkg/cli/cli.go`, `internal/transport/quic-setup.go`.
- **Benefit:** Removes trivial pairing DoS/misdirection. **Regressions:** Symmetric NAT/ CGNAT cases where the source IP differs from STUN IP; keep an escape hatch.
- **Compat:** none required for source filtering; signed PUNCH is additive.
- **Tests:** forged-then-genuine; fallback on first failure; existing punch tests.

### F-06 — Restrict Wails surface (P1, Medium)
- **Change:** Backend stores `allowedSendPaths` populated only by `SelectFileNative`/file-drop events; `SendSelectedFile(handle)` rather than raw path; `OpenFolder` restricted to `downloadDir` and its children (use `filepath.Rel` + `EvalSymlinks`) and invokes `xdg-open -- path`; consider native consent dialog; add CSP `default-src 'self'; img-src 'self' data:`; exclude `devMock.ts` from production builds.
- **Files:** `pkg/ui/app.go`, `pkg/ui/run.go`, `frontend/index.html`, `frontend/vite.config.ts`, frontend callers.
- **Benefit:** Contains renderer compromise. **Regressions:** drag-and-drop flow and tests (`pkg/ui/app_test.go`, Playwright shell) rely on arbitrary paths; update; CSP may block inline styles from Tailwind (test).
- **Compat:** none external. **Tests:** reject unselected path; reject URL in `OpenFolder`; UI e2e.

### F-03 — Token format v2 (P2, High) — requires design approval before implementation
- **Proposed change (not to be started without sign-off):** Keep tokens as **public, non-secret identifiers**. v2 = `"ftu2." + base64url( magic("FT") | version | flags | pubkey | candidates | expiry | checksum )`; the checksum (truncated SHA-256) detects corruption and typos only — **it is not authentication, and the plan must not present it as such**. Reject unknown versions/expired tokens. Do not claim confidentiality.
- **Authentication of the pairing (industry-standard layer, required to close the token-swap MITM):** a checksum cannot stop an attacker who edits the token and recomputes it, so the pairing needs one of:
  - **Option A — PAKE with a short code** (Magic Wormhole/croc model, SPAKE2 or CPace): users type a short code that is combined with the exchange; an active MITM gets one guess. Needs a vetted library → **new dependency, requires explicit approval** (never hand-written). Largest security gain, changes the UX (a code to read out) and the pairing protocol.
  - **Option B — mandatory fingerprint confirmation (SAS, Signal/ZRTP model), recommended as the minimum:** on first contact both sides display the same 4-group fingerprint of the *two* public keys and the transfer layer **refuses to open the data channel until the user confirms the match** (`ConfirmPeerFingerprint(offerID?, ok)` backend-enforced; not skippable by the renderer). Optional "trusted peer" memory of confirmed fingerprints (stored locally, user-clearable) to avoid repeating the check.
  - *Not acceptable as the only measure:* an optional/ignorable fingerprint display.
  Decision required: A, B, or B now + A later.
- **Files:** `internal/discovery/signalingToken.go`, new pairing-confirmation state in `pkg/ui/app.go` and `pkg/cli`, UI/CLI display, docs; (A) new PAKE dependency.
- **Benefit:** Detects tampering/typos, enables evolution; B/A actually authenticate the pairing against a token-channel MITM.
- **Regressions:** tokens from older builds; extra confirmation step in the flow. **Compat:** accept v1 (44/62-byte payload) tokens for N releases with a warning (v1 has no fingerprint-confirmation interplay: treat v1 peers as "unverified" and require confirmation); the `ftu2.` prefix disambiguates v2 from v1 (`.` is not in the base64url alphabet; raw lengths coincide at 56/74 bytes); emit v2 only after both ends upgrade (or emit v1+v2 behind a flag); document migration; no change to QUIC/TLS.
- **Tests:** round trip; tampered byte rejected (v2); v1 still parses; fuzz `GetPeer`; data channel refused until fingerprint confirmed; renderer cannot skip confirmation; expired token rejected.

### F-07 — Absolute punch deadline (P2, Low)
- **Change:** `deadline := time.Now().Add(Timeout)` set once; error back-off on repeated read errors.
- **Files:** `internal/connectivity/holePunch.go`. **Regressions:** none expected. **Tests:** junk-flood timeout test.

### F-08 — STUN response validation (P2, Low)
- **Change:** verify message type `0x0101`, cookie, txid, length, source address; loop until deadline; attribute iteration; try next server on failure.
- **Files:** `internal/discovery/stun.go` (+ call site). **Regressions:** servers that reply from a different address (rare; document). **Tests:** forged/short/wrong-txid replies; fuzz.

### F-09 — Deadlines and caps (P2, Medium)
- **Change:** deadline context for `AcceptUniStream`; stall detection (no bytes for N s) on data stream; read deadline for control frames; cap pending offers (ties to F-02); never create the file before the stream (F-01).
- **Files:** `internal/transfer/transfer.go`, `internal/protocol/message.go`, `pkg/ui/receiver.go`. **Regressions:** slow links near the stall limit — choose generous (e.g. 60 s). **Tests:** accept-without-data; stall; control stream idle.

### F-13 — Connect lifecycle (P2, Medium)
- **Change:** cancellable hole punch (context/stop channel instead of the deadline hack); single in-flight connect; close old `quicConn` on replace; do not hold `a.mu` across sleeps; `QuicUpgrade` takes a context.
- **Files:** `pkg/ui/app.go`, `internal/connectivity/holePunch.go`, `internal/transport/quic-setup.go`, `pkg/cli/cli.go`. **Regressions:** public signature changes (update tests). **Tests:** cancel/reconnect under `-race`.

### F-14 — Fail-closed integrity (P2, Low)
- **Change:** `ReceiveFile` rejects empty/invalid hash; after reading `expectedSize` check the stream is at EOF (reject trailing data); free-space check → `406`; distinct status for I/O vs hash failures.
- **Files:** `internal/transfer/transfer.go`, `pkg/ui/receiver.go`, `internal/protocol/receiver.go`. **Regressions:** tests that pass `""`. **Tests:** trailing bytes, empty hash, low-space simulation.

### F-10 — Pinning in `VerifyConnection`, product ALPN (P3, Medium)
- **Change:** perform the pin in `VerifyConnection` (runs on resumption too); assert TLS 1.3; set cert validity. ALPN rename (`ftu/1`) is a **wire break**: roll out with both ALPNs accepted for one release, or defer.
- **Files:** `internal/transport/quic-setup.go`. **Tests:** existing pin tests + ALPN mismatch.

### F-11 — Return errors instead of exiting (P3, Low)
- **Change:** `getLocalIPV4`, `InitializeUDPSocket`, `ed55519TokenGen` return errors; callers surface UI/CLI errors.
- **Files:** `internal/discovery/*`, `internal/transport/udp-socket.go`, `pkg/ui/app.go`, `pkg/cli/cli.go`. **Regressions:** signature changes across tests/helpers. **Tests:** offline simulation.

### F-12 — Validate token candidates (P3, Low)
- **Change:** reject port 0, unspecified/multicast/broadcast; allow loopback only in tests; omit IPv6 when nil; trim/sanitise pasted token.
- **Files:** `signalingToken.go`, `local.go`, `pkg/ui/app.go`. **Compat:** tokens that previously carried `::` still parse (field ignored). **Tests:** table of special addresses.

### F-15 / F-16 — Documentation and privacy (P4, Low)
- Document that SHA-256 is transport integrity only; document token contents/privacy; consider redacting peer addresses/hashes in persistent logs and normalising log language; set final file mode deliberately (see F-01).

### F-17 — CI and supply chain (P3, Low)
- **Change:** pin actions by commit SHA; pin `govulncheck` to a version; add Dependabot for gomod/npm/actions; add `npm audit --omit=dev` and a secret-scan step; keep `contents: read`.
- **Files:** `.github/workflows/test.yml`, `.github/dependabot.yml` (new). **Constraint:** adding tools/dependencies needs explicit approval per `CLAUDE.md`. **Regressions:** CI noise. **Tests:** workflow dry run.

### F-18 — Frontend config (P3, Low)
- See F-06 (CSP, dev-mock exclusion). Run `npm audit` locally with approval for network access.

## Market-readiness additions (not audit findings; identified in the standards review)

These are gaps relative to common industry practice for a desktop file-transfer product. They are outside the 18 audit findings and are tracked separately as `M-xx`. None is implemented; M-02..M-06 need explicit approval (tools, services, or certificates).

| ID | Priority | Complexity | Compat impact |
|----|----------|------------|---------------|
| M-01 | P1 | Medium | None |
| M-02 | P2 | High | Release process |
| M-03 | P2 | Medium | Adds update channel (client→release host only; no relay/upload server) |
| M-04 | P2 | Low | None |
| M-05 | P3 | Medium | None |
| M-06 | P3 | Low–Medium | Docs/UX |

### M-01 — Mark-of-the-Web / quarantine for received files (P1, Medium)
- **Change:** after the verified publish (F-01), mark the file as downloaded from an untrusted source: Windows NTFS `Zone.Identifier` stream with `ZoneId=3` (and `HostUrl`/`ReferrerUrl` omitted or set to the peer fingerprint, never the IP); macOS `com.apple.quarantine` xattr; Linux: no OS standard (optionally `user.xdg.origin.url`-style xattr omitted for privacy; document). Failure to mark is logged but must not fail the transfer; behaviour documented.
- **Files:** new `internal/transfer/markofweb_{windows,darwin,other}.go`, tests per platform (build tags).
- **Benefit:** Received executables are not exempt from SmartScreen/Gatekeeper, the mitigation browsers and chat apps already apply.
- **Regressions:** none on integrity; Windows ADS handling must not collide with name validation (F-01 rejects `:` in *peer-supplied* names only). **Tests:** Windows/macOS CI jobs (needs runners), unit tests with injected writer.

### M-02 — Signed releases and supply-chain provenance (P2, High)
- **Change:** Windows Authenticode signing, macOS Developer ID signing + notarisation, Linux signed packages/checksums; signed tags; SBOM (CycloneDX/SPDX) and provenance (SLSA / GitHub artifact attestations) per release; pinned toolchain; reproducible-build checks where feasible.
- **Needs approval:** certificates/accounts, new CI tooling. **Benefit:** users can verify what they run; required by OS gatekeepers. **Tests:** release dry run; verify signature/attestation in CI.

### M-03 — Authenticated update path (P2, Medium)
- **Change:** an update check/installer that verifies a signature (e.g. minisign/TUF-style or the platform updater) before applying; no telemetry, no central transfer server — only fetch of release metadata/binaries from the release host. Without it, fixed vulnerabilities never reach users. Wire it to the pinned public key shipped in the binary.
- **Needs approval:** new dependency/service. **Tests:** reject tampered/unsigned/downgraded update; rollback behaviour.

### M-04 — Vulnerability management (P2, Low)
- **Change:** add `SECURITY.md` (private reporting channel, supported versions, response targets, disclosure policy); enable GitHub private vulnerability reporting; Dependabot + `govulncheck`/`npm audit --omit=dev` in CI (see F-17); optional OpenSSF Scorecard.
- **Needs approval:** CI tools and repo settings. **Benefit:** a defined way to receive and ship fixes.

### M-05 — Independent verification and cross-platform testing (P3, Medium)
- **Change:** run the Phase 1 "unverified" items on Windows and macOS (reserved names, ADS, `explorer`/`open` URI handling, WebView2 CSP behaviour, hard-link fallback); add Windows/macOS CI runners; schedule fuzzing in CI (`make fuzz` extended with `DecodeRequest`/name sanitiser/punch parser targets); commission an external penetration test and a cryptographic review of the pairing protocol (F-03/F-04) before a general release.
- **Needs approval:** runners/services/vendor. **Benefit:** closes the audit's unverified areas.

### M-06 — Privacy notice and connectivity expectations (P3, Low–Medium)
- **Change:** a privacy notice (GDPR-relevant for EU users): the STUN query reveals the client IP to the configured STUN operator; tokens contain private/public IPs; define log retention (no persistent logs with file names/hashes by default). Document the **known limitation of the "no relay" invariant**: pairing can fail behind symmetric NAT/CGNAT/restrictive corporate networks, and show a clear error message (not a bypass). Any relay would violate `CLAUDE.md` invariants and is **not proposed**.
- **Files:** README/docs, `pkg/ui` error messages, logging config. **Compat:** none.

## Suggested order of work (Phase 2)
1. F-01 (+ helper tests, `os.Root`, `Link`-based publish, M-01 hook) → 2. F-05 → 3. F-02 → 4. F-06 → 5. F-04 (filter + challenge-response), F-07, F-08, F-13 → 6. F-09, F-14 → 7. F-10 (pin in `VerifyConnection` first), F-11, F-12 → 8. F-17/F-18, M-04 → 9. M-01 (Windows/macOS) → 10. **Pause for decision** on F-03 (token v2 + pairing option A/B), F-10 ALPN (wire change), M-02/M-03/M-05 (certificates, services, vendor) before touching them.
