# Roadmap

File Throw is a pre-release (v0.1.0). This page says where the project is going and what has to be true
before each step. The work itself is tracked as [GitHub issues](https://github.com/bernardo125/File-Transfer-Utility/issues)
grouped in milestones; the reasoning, risks and the full gap analysis are in
[`docs/RELEASE_READINESS_AUDIT.md`](docs/RELEASE_READINESS_AUDIT.md).

## Principles that do not change

- **Direct peer to peer.** No signalling server, no relay, no upload server.
- **Bounded memory.** Files are streamed in chunks, so memory does not depend on file size.
- **Explicit consent.** Nothing is written until the receiver accepts a specific offer.
- **Verified end to end.** A file is reported as received only if size and SHA-256 match.
- **No home-made cryptography.** Standard libraries and protocols only.

Changing one of these is a project-level decision, not a normal backlog item.

## Milestones

| Milestone | Theme | Done when |
|---|---|---|
| [**v0.1**](https://github.com/bernardo125/File-Transfer-Utility/milestone/2) | First pre-release | The release workflow has produced all three platform builds, private vulnerability reporting is on, `main` is protected, and the builds were smoke-tested on real Windows and macOS machines |
| [**v0.2**](https://github.com/bernardo125/File-Transfer-Utility/milestone/3) | Trusted distribution | Signed and notarised installers, CI green on Windows, macOS and Linux, quarantine marking on received files, supply-chain scans and provenance in CI, privacy notice |
| [**v0.3**](https://github.com/bernardo125/File-Transfer-Utility/milestone/4) | Hardening | Stronger pairing, authenticated hole-punch probes, native consent dialog, authenticated updater, published network-compatibility table |
| [**v1.0**](https://github.com/bernardo125/File-Transfer-Utility/milestone/5) | General availability | Independent security review finished with no open Critical or High finding, and the go/no-go checklist in the audit is fully ticked |
| [**post-1.0**](https://github.com/bernardo125/File-Transfer-Utility/milestone/1) | Later | Ideas and open decisions |

Changes that alter the wire format (protocol identifier, versioning, pairing) are grouped into as few
releases as possible so users face one compatibility break, not several.

## Open decisions

These need an explicit decision before work starts; each has an issue labelled `decision-needed`.

- How the pairing is authenticated: a short-code PAKE, a fingerprint users compare (today), remembered peers, or a combination.
- Whether to offer an optional, user-run relay for networks that block direct connections (never run by the project).
- Whether to support resuming interrupted transfers (not planned for 1.0).
- Whether the app keeps a long-lived identity or stays ephemeral per session.

## How priorities are labelled

| Label | Meaning |
|---|---|
| `priority:P0` | Blocks a general release |
| `priority:P1` | Needed before 1.0 |
| `priority:P2` | Should be done |
| `priority:P3` | Could be done |
| `effort:S` / `M` / `L` | Rough size for one maintainer: up to 2 days, up to 2 weeks, longer or dependent on someone else |

## Proposing a change

Open an issue with the *Feature request* form, or comment on an existing one. Security problems do **not**
go in public issues: see [`SECURITY.md`](SECURITY.md). Contribution guidelines: [`CONTRIBUTING.md`](CONTRIBUTING.md).
