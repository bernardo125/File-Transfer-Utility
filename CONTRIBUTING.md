# Contributing to File Throw

Thanks for your interest. This is a small project, so the process is light, but because it moves other
people's files over the network, a few rules are strict.

## Before you start

- **Security problems are not issues.** Report them privately, see [`SECURITY.md`](SECURITY.md).
- Look at the [roadmap](ROADMAP.md) and the open [issues](https://github.com/bernardo125/File-Transfer-Utility/issues).
  For anything larger than a small fix, open or comment on an issue first so we agree on the approach.
- Changes to the project principles (direct peer to peer, no relay, bounded memory, explicit consent,
  end-to-end verification) need a discussion before any code.

## Setting up

Prerequisites and build commands are in the README ([Getting started](README.md#getting-started)).
On Linux with WebKitGTK 4.1 add `-tags webkit2_41` to `wails build` and `wails dev`.

| Task | Command |
|---|---|
| Fast tests (seconds) | `make test-fast` |
| All backend tests, real sockets (about a minute) | `make test` |
| Backend tests with the race detector | `make race` |
| Frontend unit tests | `make test-frontend` |
| UI end-to-end (real backends and browser) | `make test-ui` |
| Fuzz targets | `make fuzz FUZZTIME=30s` |
| Format and vet | `gofmt -w .` and `make lint` |
| Known vulnerabilities in dependencies | `make vuln`, and `govulncheck -mode=binary <built binary>` for a release build |

`pkg/ui` needs the Wails and GTK libraries; the Makefile lists the packages that do not.

## Making a change

1. Work on a branch (`feature/...`, `fix/...`, `security/...`, `docs/...`). Never commit directly to `main`.
2. Keep the change focused: no unrelated refactoring in the same pull request.
3. **Tests are part of the change.** New behaviour needs a test that fails without it. Security fixes need a
   regression test that demonstrates the weakness is gone.
4. Run `gofmt`, `go vet ./...` and `go test -race ./...` before pushing. Frontend changes also run `npx tsc -b` and `npm test`.
5. Commit messages follow the existing style: `type(scope): summary`, for example `fix(security): ...`, `feat(ui): ...`,
   `docs: ...`, `ci: ...`. Explain the *why* in the body.
6. Open a pull request using the template and fill in the security checklist.

## Rules for code that touches security

These areas get extra scrutiny in review (see `.github/CODEOWNERS`): `internal/transport`, `internal/transfer`,
`internal/discovery`, `internal/connectivity`, `internal/protocol`, `pkg/ui`, and the CI workflows.

- **The frontend is untrusted.** Validate and authorise everything in the Go backend. Exported methods of the Wails `App`
  are callable from JavaScript; a test pins the list, so adding one is a deliberate decision.
- **Bound every input** (sizes, counts, durations) before allocating or waiting.
- **Files:** write through the temporary-file path, never overwrite, never follow user-controlled paths.
- **Cryptography:** use the standard library and well-known protocols only. Do not invent primitives or weaken TLS 1.3,
  key pinning or SHA-256 verification.
- **Compatibility:** do not change the token or wire format silently. Describe the migration impact in the pull request.
- **Dependencies:** adding or upgrading one is discussed in an issue first. Run `make vuln` and, for release builds,
  `govulncheck -mode=binary` on the artefact.
- **Logs and errors** must not contain keys, tokens, file contents, file names or IP addresses at the default level.

## Reviewing and merging

A pull request needs a passing CI run and an approving review from a code owner. Security-sensitive changes
should be reviewed by someone other than the author whenever more than one maintainer is available.

## Licence

By contributing you agree that your contribution is licensed under the project's [MIT licence](LICENSE).
