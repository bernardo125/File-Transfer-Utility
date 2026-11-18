# End-to-end tests (UI + backend + peers)

These tests run the **real product**: two instances of the real Go backend (`ui.App`), each driven
through the real production frontend in its own browser window, talking to each other over real
UDP hole punching and QUIC on loopback. A "user" journey ends on the receiver's disk (bytes and
SHA-256), never on a banner alone.

```
Playwright (Alice's page) -> frontend/dist -> shell:4301 -> ui.App  \
                                                                         }-- UDP hole punch + QUIC --
Playwright (Bob's page)   -> frontend/dist -> shell:4302 -> ui.App  /
```

`tests/ui/shell` replaces only what needs a native window: the Wails IPC (HTTP RPC + SSE
events), the file/folder dialogs, the clipboard and "open folder". Token creation, hole punching,
QUIC, transfer, hashing and the receiver loop are production code. The token exchange between the
two users is done by the test (it plays the chat app), exactly as users do.

## Run
```
cd tests/ui
npm ci && npx playwright install chromium
npm test                # builds ../../frontend, starts both backends, runs the suite
npm run test:ui         # same, with the Playwright UI
# or from the repo root: make test-ui
```
Needs Go (the backends are started with `go run`). `PW_WEBKIT=1` adds a WebKit project (system
libraries required: `npx playwright install-deps webkit`).

## Locator policy (in priority order)
1. `getByRole(role, { name })` for anything interactive or structural (buttons, headings, alerts, progressbar, table, cells).
2. `getByLabel` for form fields and labelled outputs.
3. `getByText` only for content that is the thing being asserted (a file name, a message), scoped to a region when ambiguous.
4. `getByTestId` only when nothing semantic exists. None is needed today: the UI is accessible, and the suite also runs axe (WCAG A/AA).

Never CSS selectors, XPath or `nth()`. Never `waitForTimeout`: use web-first assertions (`expect(locator)…`,
`expect.poll`) so Playwright retries until the UI settles. Locators live in `pages/`, flows in `fixtures/peers.ts`, scenarios in `specs/`.

## What is not covered
- Real NAT traversal across networks: both peers share one host (needs separate network namespaces or machines).
- The native Wails window itself (title bar, native dialogs, drag and drop). A packaged-app smoke test on each OS is the way to cover it.
- Browser engines other than Chromium locally; WebKit runs in CI.
