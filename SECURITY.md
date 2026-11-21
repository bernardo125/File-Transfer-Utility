# Security policy

File Throw moves files directly between two computers, so security reports are taken seriously.
Thank you for helping keep it safe.

## Reporting a vulnerability

**Please do not open a public issue for a security problem.**

Report it privately through GitHub:

1. Open the repository's **Security** tab.
2. Choose **Report a vulnerability** (GitHub *private vulnerability reporting*).
3. Describe the problem.

> **Maintainer note:** this only works once private vulnerability reporting is enabled in the repository
> settings (*Settings → Code security → Private vulnerability reporting*). Until it is enabled the button
> does not appear. If you prefer a different channel (for example an email address), put it here.

Useful things to include:

- what you found and what an attacker could do with it;
- the version or commit, and the operating system;
- the steps to reproduce it, or a proof of concept;
- whether you want to be credited.

Please give a reasonable time to fix the problem before you publish details. This is a small project
maintained by one person, so the targets below are goals, not guarantees:

| Step | Goal |
|---|---|
| Acknowledge your report | within 7 days |
| First assessment (accepted, needs more information, or not a vulnerability) | within 14 days |
| Fix or mitigation for a confirmed issue | depends on severity; critical and high first |

## Supported versions

Only the **latest release** (and the `main` branch) receives security fixes. There is no automatic
updater yet, so please update manually.

## What is in scope

- the transport and pairing (QUIC/TLS, key pinning, tokens, STUN and hole punching);
- the transfer protocol and how received files are written to disk;
- the desktop app's boundary between the interface and the backend;
- the release and CI configuration.

Out of scope: attacks that need you to confirm a pairing fingerprint you did not compare, social
engineering of a user into accepting a file, denial of service by flooding your own network, and
vulnerabilities in third-party software with no way to reach it through File Throw (report those
upstream).

## What the design does and does not protect

A plain-language summary is in the [README](README.md#security-model). Details:

- cryptography, keys and known limits: [`docs/security/CRYPTOGRAPHY.md`](docs/security/CRYPTOGRAPHY.md)
- threat model: [`docs/security/THREAT_MODEL.md`](docs/security/THREAT_MODEL.md)
- audit findings and remediation status: [`docs/security/SECURITY_AUDIT.md`](docs/security/SECURITY_AUDIT.md)

Known limitations at the time of writing, so you do not need to report them:

- the pairing is authenticated by the two users comparing a fingerprint; there is no short-code (PAKE) pairing yet;
- the release binaries are **not code-signed** yet, and received files are not marked by the OS as downloaded from the internet;
- the project has had a code review and automated tests, but **no independent security audit or penetration test**.

## Verifying a download

Each release includes `SHA256SUMS.txt`. Compare it with the checksum of the file you downloaded
(`sha256sum`, `shasum -a 256` or `certutil -hashfile <file> SHA256`).
