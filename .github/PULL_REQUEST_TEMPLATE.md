## What and why

<!-- One or two sentences. Link the issue: "Closes #123". -->

## How it was tested

- [ ] `gofmt -l internal pkg tests cmd main.go` prints nothing, `go vet ./...` passes
- [ ] `go test -race -count=1 ./...` passes
- [ ] Frontend changes: `cd frontend && npx tsc -b && npm test`; UI behaviour: `cd tests/ui && npx playwright test --project=chromium`
- [ ] New behaviour has a test that fails without the change

## Security checklist (tick or write "n/a")

- [ ] Input from the peer is validated in the Go backend, never trusted from the frontend
- [ ] No new dependency (or it was discussed in an issue first)
- [ ] No change to the token or wire format, or the compatibility impact is described below
- [ ] No weakening of TLS 1.3, key pinning, SHA-256 verification or the consent step
- [ ] Files written to disk still use the temporary-file, no-overwrite path
- [ ] Nothing sensitive (keys, tokens, file names, IP addresses) added to logs or error messages

## Compatibility and migration

<!-- Anything users or older versions will notice. "None" is a valid answer. -->

## Docs

- [ ] README, `docs/` or `CHANGELOG.md` updated where behaviour changed
