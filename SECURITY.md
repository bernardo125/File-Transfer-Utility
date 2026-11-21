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

**There are no fixed response or fix times.** File Throw is a free utility maintained by one person in
their spare time, so no deadline can be promised. What you can expect: reports are read privately, serious
problems (critical and high) are looked at first, and you are told what was decided and credited if you wish.

Please give a reasonable time to fix the problem before you publish details. If you have not heard back
after a few weeks, a polite reminder on the advisory is welcome, and you are free to disclose responsibly
if the problem is serious and there is no sign of progress.

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
- how reports are handled and how fixes are released: [`docs/security/VULNERABILITY_HANDLING.md`](docs/security/VULNERABILITY_HANDLING.md)
- threat model: [`docs/security/THREAT_MODEL.md`](docs/security/THREAT_MODEL.md)
- audit findings and remediation status: [`docs/security/SECURITY_AUDIT.md`](docs/security/SECURITY_AUDIT.md)

Known limitations at the time of writing, so you do not need to report them:

- the pairing is authenticated by the two users comparing a fingerprint; there is no short-code (PAKE) pairing yet;
- the release binaries are **not code-signed** yet, and received files are not marked by the OS as downloaded from the internet;
- the project has had a code review and automated tests, but **no independent security audit or penetration test**.

## Verifying a download

Each release includes `SHA256SUMS.txt`. Compare it with the checksum of the file you downloaded
(`sha256sum`, `shasum -a 256` or `certutil -hashfile <file> SHA256`).
