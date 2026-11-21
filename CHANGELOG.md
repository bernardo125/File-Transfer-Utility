# Changelog

All notable changes to File Throw. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and the project uses [Semantic Versioning](https://semver.org/). Versions below 1.0.0 may change behaviour
between releases.

## [Unreleased]

## [0.1.0] - 2026-11-21

First public release, **beta**: code-reviewed, not independently audited, binaries not code-signed. The release page on GitHub carries the checksums and the limitations below.

### Added
- Direct peer-to-peer file transfer (QUIC, TLS 1.3, mutual Ed25519 authentication), desktop app and `--cli` mode.
- Token format v2 (`ftu2.`: version, 24 h expiry, checksum); v1 tokens are still read.
- Mandatory pairing-fingerprint confirmation in the desktop app and the CLI before any data flows.
- Per-offer consent identifiers; one offer at a time.
- Production Content-Security-Policy; `SECURITY.md`; security audit, threat model and cryptography reference in `docs/security`.
- Release workflow (Windows, macOS, Linux) with stable asset names and `SHA256SUMS.txt`. Versions `0.x` are published as normal releases titled "(beta)" so the `releases/latest` download links work.
- Vulnerability handling procedure and steps for releasing a fix (`docs/security/VULNERABILITY_HANDLING.md`).

### Changed
- Received files go to a private temporary file and are published only after size and SHA-256 match; an existing file is never overwritten.
- STUN replies must answer our own request; the hole punch only trusts packets from the token's addresses and falls back across candidates.
- The key pin is enforced on every TLS handshake, including resumed ones.
- The interface can only send files picked in the native dialog and only open folders inside the download directory.
- Go 1.26.11, `golang.org/x/net` 0.63.0 and `golang.org/x/crypto` 0.59.0 (fixes all `govulncheck` findings for the release binaries).

### Fixed
- Failed or cancelled transfers could truncate or delete a file with the same name.
- Cancelling a connection attempt could leave a reader on the shared UDP socket.
- A machine without a network route could terminate the app.
- Invalid YAML in the `e2e-ui` CI job.

### Known limitations
See the release notes: unsigned binaries, no PAKE pairing, no independent audit.
