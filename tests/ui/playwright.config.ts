import { defineConfig, devices } from '@playwright/test';

const CI = !!process.env.CI;
export const SHELLS = { alice: 4301, bob: 4302 } as const;

/**
 * Real end-to-end tests: two instances of the real Go backend (ui.App), each driven through the
 * real production frontend in its own browser, talking to each other over real UDP/QUIC on
 * loopback. Only the native window is replaced (see ./shell). See README.md.
 */
export default defineConfig({
  testDir: './specs',
  // Both peers keep OS sockets and share two long-lived backends, so tests run one at a time.
  fullyParallel: false,
  workers: 1,
  forbidOnly: CI,
  retries: CI ? 1 : 0, // a retry is reported as "flaky" in the HTML report; locally a failure is a failure
  timeout: 120_000, // hole punching has a 60 s ceiling by design
  expect: { timeout: 15_000 },
  reporter: CI ? [['github'], ['html', { open: 'never' }]] : [['list']],
  use: {
    locale: 'en-US',
    reducedMotion: 'reduce',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } }, // WebView2 engine
    ...(CI || process.env.PW_WEBKIT ? [{ name: 'webkit', use: { ...devices['Desktop Safari'] } }] : []), // WebKitGTK/WKWebView engine
  ],
  // One backend per peer, serving the production build (built by `npm test`, see package.json).
  webServer: Object.values(SHELLS).map((port) => ({
    command: `go run ./shell -addr 127.0.0.1:${port} -dist ../../frontend/dist`,
    url: `http://127.0.0.1:${port}/`,
    reuseExistingServer: !CI,
    timeout: 180_000,
    stdout: 'pipe' as const,
  })),
});
