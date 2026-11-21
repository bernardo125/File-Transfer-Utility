# Release readiness audit: path to a general release (v1.0)

| | |
|---|---|
| **Product** | File Throw (FTU), direct peer-to-peer file transfer (Go, Wails, React, QUIC/TLS 1.3) |
| **Audit date** | 2026-11-21 |
| **Baseline** | branch `security/hardening-2026-11-20`, commit `036c4a7` plus the commit that adds this document |
| **Scope** | Everything needed to move from the current pre-release (v0.1.0) to a general release: security, quality, distribution, platforms, UX, legal, operations |
| **Author's caveat** | Prepared by the same AI assistant that wrote the code fixes. It is a self-assessment, not an independent audit (see GA-01) |
| **Companion documents** | [`docs/security/`](security) (audit, threat model, cryptography, remediation), [`SECURITY.md`](../SECURITY.md) |
| **Backlog** | 47 items (GA-01 to GA-47). 42 are GitHub issues (#12 to #53, see the index in section 7); the 5 that describe known unfixed weaknesses (GA-03, GA-05, GA-06, GA-13, GA-15) are deliberately kept out of the public tracker until the owner decides |

## 1. Executive summary

**Verdict.** The product is ready for a **v0.1.0 pre-release aimed at technical users** once three owner-side steps are done (dry-run of the release workflow, private vulnerability reporting enabled, branch protection; issues GA-22, GA-43, GA-46). It is **not ready for a general release**: distribution is unsigned, two of three operating systems have never run the software, nothing independent has reviewed the design, and there is no way to deliver a security fix to installed copies.

The engineering core is sound for its stage: authenticated, encrypted channel with pinned keys; safe file writing; bounded memory; fuzzed parsers; a scanned and clean dependency set. What separates it from a market-grade product is mostly *trust infrastructure* (signing, provenance, updates, review, process), not application code.

| Dimension | Rating | Why (evidence in section 2) |
|---|:--:|---|
| Cryptographic design | **Amber** | Standard building blocks, no custom crypto, hybrid post-quantum key exchange. Pairing authentication relies on users comparing a fingerprint; no independent review |
| Implementation security | **Amber/Green** | 18 audit findings addressed (8 fixed, 9 partial, 1 deferred); regression tests for each; binary scans clean. Residuals listed in GA-03/05/06/13/15 |
| Test engineering | **Amber** | Strong on Linux (unit, integration, race, fuzz smoke, Playwright). Windows/macOS unexecuted, no real-network matrix, no long fuzzing, no multi-GB tests |
| Supply chain and distribution | **Red** | Unsigned binaries, no provenance or SBOM, actions pinned by tag, no updater, release pipeline never executed |
| Platform coverage | **Red** | Linux executed; Windows built only; macOS only in CI configuration |
| Connectivity | **Amber** | By design no relay: some NATs cannot pair. IPv6 path untested. Failure messages not actionable |
| UX and accessibility | **Amber** | Axe checks pass; no assistive-technology testing; the critical fingerprint step has no onboarding; no cancel for transfers |
| Privacy and legal | **Amber** | No privacy notice, STUN disclosure or third-party licence inventory yet |
| Operations and governance | **Red** | No vulnerability process beyond `SECURITY.md`, no release process, no branch protection, single maintainer |

Rating legend: Green = meets the bar for general release; Amber = acceptable for a pre-release, needs work for GA; Red = blocks GA.

## 2. Evidence baseline

### 2.1 Verified (commands run on 2026-11-21, Linux, Go 1.26.11, Node 22)

| Check | Result |
|---|---|
| `gofmt -l`, `go vet ./...`, `go mod verify` | clean |
| `go test ./...` and `go test -race ./...` | pass, no data races |
| Fuzz targets (`make fuzz`, 2-8 s each, 7 targets) | no crashers (smoke test only) |
| `cd frontend && npx tsc -b`, `npm test` (Vitest) | pass, 50 tests |
| Playwright, chromium project, real backends and built bundle | pass, 10 tests |
| `govulncheck -mode=binary` on the Linux and Windows release binaries | "No vulnerabilities found" after moving to Go 1.26.11, `x/net` 0.63.0, `x/crypto` 0.59.0. The same scan on the earlier Go 1.26.4 build reported **29-30 vulnerabilities**, which a source-only scan did not show |
| `npm audit` (frontend, tests/ui) | 0 vulnerabilities after `npm audit fix` (2 high Vite dev-server advisories before) |
| `actionlint` on both workflows | clean after fixing invalid YAML in `test.yml` |
| End-to-end smoke test of the built Linux binary, two CLI processes | tokens v2 exchanged, identical fingerprint, 3 MB file received with equal SHA-256, no temp files left |
| Negotiated TLS parameters between two builds | TLS 1.3, `TLS_AES_128_GCM_SHA256`, key exchange X25519MLKEM768 |

### 2.2 Not verified (each one maps to backlog items)

- Execution of the Windows `.exe` and of any macOS build (GA-24, GA-29, GA-31).
- The packaged GUI in a real webview on any OS, including the Content-Security-Policy and native dialogs (GA-29).
- Connectivity across real NATs, CGNAT, IPv6-only networks, VPNs (GA-25, GA-35).
- Transfers larger than about 100 MB, full disks, long sessions (GA-26).
- Fuzzing beyond a smoke run; mutation and coverage gates (GA-27, GA-28).
- The release workflow itself, signing, notarisation, packaging, update flow (GA-16 to GA-22).
- Screen-reader and keyboard-only use (GA-30).
- Anything an independent reviewer would look at differently (GA-01).

## 3. What "market grade" means here

A user who downloads a networked file-transfer tool from a stranger needs to be able to answer "is this really the project's file, is it current, and what happens if a flaw is found?". The criteria this audit uses for general release:

1. **Authentic distribution:** signed installers on Windows and macOS, signed checksums and provenance everywhere, an SBOM.
2. **A path for fixes:** an authenticated updater, a documented vulnerability process, a rehearsed emergency release.
3. **Independent assurance:** a third-party review of the cryptographic design and the file/transport layers, with no open Critical or High.
4. **Parity across platforms:** the same automated suite green on Windows, macOS and Linux, plus manual smoke runs of the packaged app.
5. **Honest, usable security:** the user-facing security step (fingerprint) is explained, hard to skip, or replaced by automatic authentication.
6. **Compliance basics:** privacy notice, licence inventory, no undeclared telemetry.
7. **Operability:** a documented release and rollback process, protected main branch, contribution and support channels.

## 4. Findings by domain

Each paragraph states the gap and points to the backlog items. IDs are `GA-nn`.

### 4.1 Security architecture and cryptography
- **Pairing authentication (GA-02, GA-07, GA-36).** Authentication of the pairing depends on both users comparing a fingerprint. The session identity key is new each time, so nothing is remembered between sessions. Industry practice is a PAKE short code (Magic Wormhole, croc) or a persistent, verified identity (Syncthing, Signal safety numbers). This is the single largest design-level residual risk.
- **Hole-punch probes (GA-03).** Unauthenticated; mitigated by a source-IP filter and dial fallback, not eliminated.
- **Protocol identity and evolution (GA-04, GA-06, GA-09).** The ALPN is still the sample value `quic-echo-example`; no protocol version field; the data stream is not bound to the approved offer; legacy v1 tokens linger. Fix all wire changes in one transition before 1.0.
- **Consent trust boundary (GA-05).** The accept decision is made in the renderer.
- **File provenance on disk (GA-08).** Received executables carry no Mark-of-the-Web or quarantine attribute and therefore bypass SmartScreen and Gatekeeper.
- **Assurance (GA-01, GA-10).** No independent review; no standards-based self-assessment.

### 4.2 Application security
CLI/GUI parity (GA-11), Windows free-space check (GA-12), clipboard and drag-and-drop surface (GA-13), logging policy (GA-14), inbound connection limits (GA-15). None is individually severe; together they close the audit's Low findings.

### 4.3 Supply chain and release engineering
- **Signing (GA-16, GA-17)** is the first thing users see. Windows: certificates must live on hardware or a cloud HSM; EV no longer gives instant SmartScreen reputation, so reputation also needs time and a constant publisher identity. macOS: Developer ID plus notarisation is mandatory for a smooth install.
- **Provenance and SBOM (GA-20), pinned actions and scanning in CI (GA-21), reproducible builds (GA-23).** The 2026-11-21 incident is the argument: a source-only scan reported nothing, while the binary scan reported 29-30 findings from the Go standard library and two modules. Scanning the *built artefact* must be a release gate.
- **Updater (GA-19)** and **installers (GA-18)** are required for a product that handles other people's files.
- **The release pipeline has never run (GA-22).**

### 4.4 Quality engineering
Linux is well covered; the rest is not. The highest-value items are the Windows/macOS CI matrix (GA-24) and per-OS packaged-app smoke tests (GA-29), because platform-specific behaviour (file names, hard links, quarantine, WebView) cannot be inferred from Linux results. Then the real-network matrix (GA-25), multi-GB and resource tests (GA-26), and a long fuzzing campaign (GA-27).

### 4.5 Platforms and connectivity
Windows filename and filesystem behaviour (GA-31), Linux distribution matrix (GA-32), IPv6 coverage (GA-35). For reachability, decide the fallback policy (GA-34) and make failures understandable (GA-33): "direct only" is a legitimate product position if it is stated plainly and the app tells the user why a pairing failed.

### 4.6 UX and accessibility
The security-critical step needs onboarding and a measurable usability bar (GA-36). Add cancel (GA-37), consistent language (GA-38), and real assistive-technology testing (GA-30). Resumption (GA-39) is a product decision, not a defect.

### 4.7 Privacy, legal, operations
Privacy notice and a switch for STUN (GA-40), licence inventory (GA-41), name check (GA-42). Operations: vulnerability process and enabling private reporting (GA-43), community files (GA-44), release process (GA-45), branch protection and CODEOWNERS (GA-46), telemetry stance (GA-47).

## 5. Decisions required from the owner

| ID | Decision | Options | Recommendation |
|---|---|---|---|
| D1 | Pairing authentication (GA-02) | PAKE short code (new dependency) / strengthen fingerprint and remember peers (GA-07) / both, in that order | Fingerprint + remembered peers for v0.3; PAKE for v1.x after the external review |
| D2 | Identity model (GA-07) | Ephemeral per session / optional persistent identity in the OS keychain | Optional persistent identity; keep ephemeral as default for privacy |
| D3 | Signing providers and publisher identity (GA-16, GA-17, GA-42) | SignPath Foundation or Certum for Windows; Apple Developer Program for macOS | Start identity verification now: it has external lead time |
| D4 | Connectivity fallback (GA-34) | Direct only / optional user-run relay forwarding ciphertext | Direct only for 1.0, documented clearly; revisit after user feedback |
| D5 | Resume transfers (GA-39) | Out of scope / design it | Out of scope for 1.0 |
| D6 | Telemetry (GA-47) | None / opt-in local crash logs | None |

## 6. Roadmap and exit criteria

| Milestone | Theme | Contains | Exit criteria |
|---|---|---|---|
| **v0.1** | First pre-release | GA-22, GA-43, GA-46 | Release workflow produced all three artefacts; private reporting enabled; `main` protected; binaries manually smoke-tested on a Windows and a macOS machine; notes state the limitations |
| **v0.2** | Trusted distribution | GA-04, GA-08, GA-11, GA-12, GA-13, GA-16, GA-17, GA-18, GA-20, GA-21, GA-24, GA-29, GA-31, GA-32, GA-36, GA-40, GA-41, GA-44, GA-45 | Signed and notarised installers; CI green on three OSes; ALPN/versioning in place; vulnerability and provenance gates in CI; privacy notice published |
| **v0.3** | Hardening | GA-02, GA-03, GA-05, GA-06, GA-07, GA-14, GA-15, GA-19, GA-25, GA-26, GA-27, GA-33, GA-34, GA-35, GA-37, GA-38 | Stronger pairing shipped; authenticated probes; native consent; updater working from the previous release; network matrix published |
| **v1.0** | General availability | GA-01, GA-09, GA-10, GA-23, GA-28, GA-30, GA-42, GA-47 | External review complete with no open Critical/High; go/no-go checklist (section 8) fully ticked |
| **post-1.0** | Ideas | GA-39 | n/a |

Order inside a milestone: P0 first, then items that change the wire format (GA-04, GA-06, GA-02, GA-03) together so users face one compatibility break, then everything else.

## 7. Backlog index

Each item is tracked as a GitHub issue titled `[GA-nn] ...` (labels `priority:*`, `area:*`, `effort:*`, milestone = target release). Effort is a rough estimate for one maintainer: **S** up to 2 days, **M** up to 2 weeks, **L** longer or dependent on an external party. Totals: 8 P0, 22 P1, 15 P2, 2 P3.

| ID | Title | GitHub | Area | Prio | Effort | Milestone | Owner |
|---|---|---|---|:--:|:--:|---|---|
| GA-01 | Independent security review and penetration test | #12 | security | P0 | L | v1.0 | owner + external |
| GA-08 | Mark received files as downloaded from the internet | #16 | security | P0 | M | v0.2 | maintainer |
| GA-16 | Windows code signing | #22 | release | P0 | M | v0.2 | owner + external |
| GA-17 | macOS signing and notarisation | #23 | release | P0 | M | v0.2 | owner + external |
| GA-22 | Dry-run the release workflow on all three platforms | #28 | release | P0 | S | v0.1 | owner |
| GA-24 | CI that executes the test suite on Windows and macOS | #30 | testing | P0 | M | v0.2 | maintainer |
| GA-29 | Webview smoke tests per OS (CSP, native dialogs, packaging) | #35 | testing | P0 | M | v0.2 | maintainer |
| GA-43 | Vulnerability handling process | #49 | ops | P0 | S | v0.1 | owner |
| GA-02 | Pairing authenticated by a PAKE short code (decision) *(decision)* | #13 | security | P1 | L | v0.3 | owner decision |
| GA-03 | Authenticate hole-punch probes (challenge-response) | not created | security | P1 | M | v0.3 | maintainer |
| GA-04 | Product ALPN and explicit protocol versioning | #14 | security | P1 | S | v0.2 | maintainer |
| GA-05 | Consent decided outside the renderer (native dialog) | not created | security | P1 | M | v0.3 | maintainer |
| GA-10 | Security self-assessment against OWASP ASVS Level 2 and re-baselined threat model | #18 | security | P1 | M | v1.0 | maintainer |
| GA-18 | Installers and Linux packaging | #24 | release | P1 | M | v0.2 | maintainer |
| GA-19 | Authenticated automatic update | #25 | release | P1 | L | v0.3 | maintainer |
| GA-20 | Provenance, SBOM and signed checksums | #26 | release | P1 | M | v0.2 | maintainer |
| GA-21 | Pin CI actions, automate dependency and vulnerability checks | #27 | release | P1 | S | v0.2 | maintainer |
| GA-25 | Real-network test matrix | #31 | testing | P1 | L | v0.3 | maintainer + volunteers |
| GA-26 | Large-file and resource-exhaustion tests | #32 | testing | P1 | M | v0.3 | maintainer |
| GA-27 | Fuzzing campaign | #33 | testing | P1 | M | v0.3 | maintainer |
| GA-31 | Verify Windows file-name and filesystem behaviour | #37 | platform | P1 | S | v0.2 | maintainer |
| GA-33 | Connection diagnostics and actionable failure messages | #39 | network | P1 | M | v0.3 | maintainer |
| GA-34 | Decision: connectivity fallback policy (no relay vs optional relay) *(decision)* | #40 | network | P1 | S | v0.3 | owner decision |
| GA-35 | Exercise the IPv6 path in tests | #41 | network | P1 | M | v0.3 | maintainer |
| GA-36 | First-run guidance for the fingerprint step | #42 | ux | P1 | S | v0.2 | maintainer |
| GA-37 | Cancel a transfer in progress | #43 | ux | P1 | M | v0.3 | maintainer |
| GA-40 | Privacy notice and STUN disclosure | #46 | legal | P1 | S | v0.2 | owner |
| GA-41 | Third-party licence inventory | #47 | legal | P1 | S | v0.2 | maintainer |
| GA-45 | Release process, versioning and rollback | #51 | ops | P1 | S | v0.2 | maintainer |
| GA-46 | Branch protection, required checks and repository hygiene | #52 | ops | P1 | S | v0.1 | owner |
| GA-06 | Bind the data stream to the approved offer | not created | security | P2 | M | v0.3 | maintainer |
| GA-07 | Persistent identity and trusted-peer store (decision) *(decision)* | #15 | security | P2 | L | v0.3 | owner decision |
| GA-09 | Retire legacy v1 tokens | #17 | security | P2 | S | v1.0 | maintainer |
| GA-11 | CLI parity with the desktop app's limits | #19 | security | P2 | S | v0.2 | maintainer |
| GA-12 | Free-space precheck on Windows | #20 | app | P2 | S | v0.2 | maintainer |
| GA-13 | Review clipboard, drag-and-drop and external-open exposure | not created | app | P2 | S | v0.2 | maintainer |
| GA-14 | Logging policy: structured, redacted, user-controllable | #21 | app | P2 | M | v0.3 | maintainer |
| GA-15 | Bound inbound connection attempts and resource use | not created | app | P2 | M | v0.3 | maintainer |
| GA-23 | Reproducible-build check | #29 | release | P2 | M | v1.0 | maintainer |
| GA-28 | Coverage and mutation gates | #34 | testing | P2 | S | v1.0 | maintainer |
| GA-30 | Accessibility verification with assistive technology | #36 | testing | P2 | M | v1.0 | maintainer + testers |
| GA-32 | Linux runtime compatibility matrix | #38 | platform | P2 | S | v0.2 | maintainer |
| GA-38 | Consistent language and messages | #44 | ux | P2 | M | v0.3 | maintainer |
| GA-42 | Name and branding check | #48 | legal | P2 | S | v1.0 | owner |
| GA-44 | Community and contribution files | #50 | ops | P2 | S | v0.2 | maintainer |
| GA-39 | Resume interrupted transfers (decision) *(decision)* | #45 | ux | P3 | L | post-1.0 | owner decision |
| GA-47 | Telemetry and crash-reporting stance *(decision)* | #53 | ops | P3 | S | v1.0 | owner decision |

## 8. General-release go / no-go checklist

All must be true for v1.0.

- [ ] Independent security review delivered; no open Critical or High finding (GA-01).
- [ ] Windows installer signed and verified; macOS app signed, notarised and stapled (GA-16, GA-17).
- [ ] Every release artefact has signed checksums, provenance and an SBOM; verification commands are in the README (GA-20).
- [ ] CI runs the full suite on Windows, macOS and Linux and is green; the packaged app was smoke-tested on each OS for this release (GA-24, GA-29).
- [ ] `govulncheck -mode=binary` and `npm audit --omit=dev` are clean for the exact artefacts being shipped (GA-21).
- [ ] An authenticated update path works from the previous release on each OS, including rollback (GA-19).
- [ ] Received files are marked as downloaded from the internet on Windows and macOS (GA-08).
- [ ] Pairing authentication is automatic or the fingerprint step has passed a usability test (GA-02/07/36).
- [ ] Wire format finalised: product ALPN, protocol version, legacy tokens removed (GA-04, GA-06, GA-09).
- [ ] Vulnerability reporting enabled and a private drill completed; emergency release procedure rehearsed (GA-43, GA-45).
- [ ] Privacy notice, third-party notices and contribution files published (GA-40, GA-41, GA-44).
- [ ] Compatibility table (NAT types, IPv6, OSes, distributions) published with test evidence (GA-25, GA-32).
- [ ] `main` protected with required checks and CODEOWNERS for security-sensitive paths (GA-46).

## 9. Risk register

| # | Risk | Likelihood | Impact | Mitigation (backlog) |
|---|---|:--:|:--:|---|
| R1 | Users confirm the fingerprint without comparing; a man in the middle on the token channel pairs with them | Medium | High | GA-02, GA-07, GA-36 |
| R2 | Trojanised copies of the app distributed because the real binaries are unsigned and unverifiable | High | High | GA-16, GA-17, GA-20 |
| R3 | A platform-specific bug (file names, links, quarantine, webview) ships because only Linux was executed | Medium | Medium/High | GA-24, GA-29, GA-31 |
| R4 | A fixed vulnerability stays exploitable on installed copies (no updater) | Medium | High | GA-19, GA-21, GA-43 |
| R5 | A compromised dependency or CI action reaches a release | Low/Medium | High | GA-20, GA-21 |
| R6 | Poor first impressions from unreachable networks and unexplained errors | High | Medium | GA-33, GA-34, GA-25 |
| R7 | Design flaws shared by author and reviewer go unnoticed | Medium | High | GA-01, GA-10 |
| R8 | Privacy complaint: IPs sent to STUN operators and embedded in tokens, no notice | Medium | Medium | GA-40 |
| R9 | Single-maintainer bottleneck delays a security fix | High | Medium | GA-44, GA-45, GA-46 |
| R10 | Toolchain or dependency drift reintroduces known vulnerabilities between releases | Medium | Medium | GA-21 (binary scan on every release) |

## 10. Residual risk and how to describe the product

Even after everything above, a peer-to-peer tool that accepts files from people you pair with carries inherent residual risk: a trusted peer can still send an unwanted file, endpoint compromise defeats any transport security, and authentication is classical (Ed25519). The hybrid X25519 + ML-KEM key exchange protects recorded traffic against a future quantum computer but not a forged future handshake.

Until the go/no-go checklist is ticked, describe the product as **"pre-release, reviewed, not independently audited"** and keep the limitations list in the README and release notes. After it is ticked, claims should still be specific ("end-to-end encrypted with pinned keys, independently reviewed on <date>") and never "secure" without qualification.

## Appendix A. Standards and references

OWASP ASVS 4.0.3 (target: Level 2 for the chapters in Appendix B); OWASP Application Security Verification and Cheat Sheets (file upload, logging); CWE (22, 59, 73, 367, 295, 345); RFC 9000/9001 (QUIC), RFC 8446 (TLS 1.3), RFC 8489 (STUN), RFC 8445 (ICE, as the model for authenticated connectivity checks); NIST SP 800-218 (SSDF); SLSA v1.0 build levels; OpenSSF Scorecard; CycloneDX/SPDX for SBOMs; TUF for update design; GDPR for privacy basics; WCAG 2.1 AA / EN 301 549 for accessibility.

## Appendix B. ASVS Level 2, indicative self-assessment

Indicative only: not a certification and not independent. To be replaced by GA-10.

| ASVS chapter | Status | Evidence / gap |
|---|---|---|
| V1 Architecture, design and threat modelling | Partial | `docs/security/THREAT_MODEL.md`, `SECURITY_ARCHITECTURE.puml`; needs refresh after GA-02/03/05/07 |
| V6 Stored cryptography | Largely met | Keys memory-only, no persisted secrets; stronger story needed if GA-07 adds persistence |
| V8 Data protection | Partial | No persistent logs by default; logging policy and privacy notice open (GA-14, GA-40) |
| V9 Communications security | Largely met | TLS 1.3 enforced, mutual authentication, pin checked on every handshake; ALPN and probe authentication open (GA-03, GA-04) |
| V10 Malicious code | Partial | Dependency and binary scans clean; provenance, signing and pinned actions open (GA-20, GA-21) |
| V12 Files and resources | Largely met | Temp file, size+hash check, no overwrite, `os.Root`, name validation; Windows/macOS behaviour and MOTW open (GA-08, GA-31) |
| V13 API and web service | Partial | Renderer-callable surface pinned by a test; consent still in renderer (GA-05) |
| V14 Configuration | Partial | CSP present, mock excluded from production; per-OS webview verification open (GA-29) |
| V2/V3/V4 Authentication, session, access control | Not applicable in the web sense | Peer authentication covered under V9 and GA-02 |

## Appendix C. Reproducing the checks

```bash
gofmt -l internal pkg tests cmd main.go && go vet ./... && go test -race -count=1 ./...
make fuzz FUZZTIME=8s
(cd frontend && npx tsc -b && npm test && npm audit --omit=dev)
(cd tests/ui && npx playwright test --project=chromium)
go install golang.org/x/vuln/cmd/govulncheck@latest
wails build -platform linux/amd64 -tags webkit2_41 -o ftu-linux-amd64   # then:
govulncheck -mode=binary build/bin/ftu-linux-amd64                       # scan the artefact, not only the source
go install github.com/rhysd/actionlint/cmd/actionlint@latest && actionlint .github/workflows/*.yml
```

## Appendix D. Open audit findings carried into the backlog

From [`docs/security/SECURITY_AUDIT.md`](security/SECURITY_AUDIT.md): F-02 (data stream not bound to offer, GA-06), F-03 (no PAKE, GA-02/07), F-04 (probe challenge-response, GA-03), F-06 (native consent, clipboard, drag-and-drop, GA-05/13), F-09 (CLI deadline, GA-11), F-10 (ALPN, GA-04), F-14 (Windows free space, GA-12), F-16 (logging, GA-14), F-17 (CI pinning and scanners, GA-21), F-18 (webview CSP verification, GA-29), M-01 (quarantine marking, GA-08), M-02 to M-06 (GA-16 to GA-20, GA-40, GA-43).
