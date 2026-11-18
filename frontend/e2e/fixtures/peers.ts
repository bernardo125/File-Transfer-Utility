import { test as base, expect, type Browser, type Page } from '@playwright/test';
import { createHash, randomBytes } from 'node:crypto';
import { createWriteStream, existsSync, mkdirSync, readdirSync, createReadStream } from 'node:fs';
import { once } from 'node:events';
import path from 'node:path';
import { SHELLS } from '../../playwright.config';
import { HomePage } from '../pages/home.page';
import { PairingPage } from '../pages/pairing.page';
import { TransferPage } from '../pages/transfer.page';

/** Replaces only Wails' native IPC: calls go to the real Go App over HTTP, events come back over SSE. */
function installBridge(methods: string[]) {
  const w = window as unknown as Record<string, any>;
  const listeners = new Map<string, Set<(d: unknown) => void>>();
  const es = new EventSource('/__shell/events');
  const open = new Promise<void>((resolve) => { es.onopen = () => resolve(); });
  es.onmessage = (m) => {
    const { name, data } = JSON.parse(m.data);
    listeners.get(name)?.forEach((cb) => cb(data));
  };
  const app: Record<string, (...a: unknown[]) => Promise<unknown>> = {};
  for (const method of methods) {
    app[method] = async (...args) => {
      await open; // never call the backend before we can hear its events
      const res = await (await fetch(`/__shell/rpc/${method}`, { method: 'POST', body: JSON.stringify(args) })).json();
      if (!res.ok) throw res.error; // Wails rejects with a plain string
      return res.value ?? null;
    };
  }
  w.go = { ui: { App: app } };
  w.runtime = {
    EventsOnMultiple: (name: string, cb: (d: unknown) => void) => {
      if (!listeners.has(name)) listeners.set(name, new Set());
      listeners.get(name)!.add(cb);
      return () => listeners.get(name)?.delete(cb);
    },
    EventsOff: (name: string) => listeners.delete(name),
    BrowserOpenURL: () => {},
    Quit: () => {},
    LogPrint: () => {},
  };
}

// Methods of the generated binding (src/wailsjs/go/ui/App.js) the UI may call.
const APP_METHODS = [
  'CancelConnection', 'ConnectPeer', 'CopyToClipboard', 'Disconnect', 'GetClipboardText', 'GetDownloadDirectory',
  'GetLocalToken', 'GetMyToken', 'InspectFile', 'OpenFolder', 'OpenGitHub', 'QuitApplication',
  'RespondIncomingTransfer', 'SelectDownloadDirectory', 'SelectFileNative', 'SendSelectedFile',
];

/** One user: a real browser window on the real UI, backed by its own real Go App. */
export class Peer {
  readonly home: HomePage;
  readonly pairing: PairingPage;
  readonly transfer: TransferPage;

  constructor(
    readonly name: string,
    readonly page: Page,
    private readonly origin: string,
    /** Where this peer's backend saves received files. */
    readonly downloadDir: string,
  ) {
    this.home = new HomePage(page);
    this.pairing = new PairingPage(page);
    this.transfer = new TransferPage(page);
  }

  private async control(route: string, body?: unknown) {
    const res = await fetch(`${this.origin}/__shell/${route}`, { method: body === undefined ? 'GET' : 'POST', body: JSON.stringify(body) });
    return res.json();
  }

  /** Fresh backend (new socket and identity) and a freshly loaded UI, like launching the app. */
  async launch() {
    mkdirSync(this.downloadDir, { recursive: true });
    await this.control('reset', { DownloadDir: this.downloadDir });
    await this.page.addInitScript(installBridge, APP_METHODS);
    await this.page.goto(`${this.origin}/`);
  }

  /** What the native "choose a file" dialog will return next. */
  async willPick(filePath: string) {
    await this.control('pick', { Path: filePath });
  }

  async openedFolders(): Promise<string[]> {
    return (await this.control('state')).opened;
  }

  /** Simulates the app being closed / the machine vanishing: socket and QUIC session die. */
  async crash() {
    await this.control('reset', { DownloadDir: this.downloadDir });
  }

  receivedFiles(): string[] {
    return existsSync(this.downloadDir) ? readdirSync(this.downloadDir) : [];
  }

  async startAs(role: 'send' | 'receive') {
    await (role === 'send' ? this.home.sendCard : this.home.receiveCard).click();
    await expect(this.pairing.localToken).not.toHaveText(/Discovering|Unavailable/);
  }

  async token(): Promise<string> {
    return ((await this.pairing.localToken.textContent()) ?? '').trim();
  }
}

/**
 * Both peers go through the exact user journey: each shows its token, the users exchange them out
 * of band (the test plays the chat app), both press Connect, and hole punching + QUIC do the rest.
 */
export async function pair(a: Peer, b: Peer, roles: { a: 'send' | 'receive'; b: 'send' | 'receive' } = { a: 'send', b: 'receive' }) {
  await Promise.all([a.startAs(roles.a), b.startAs(roles.b)]);
  const [tokenA, tokenB] = [await a.token(), await b.token()];
  await a.pairing.peerTokenInput.fill(tokenB);
  await b.pairing.peerTokenInput.fill(tokenA);
  await Promise.all([a.pairing.connectButton.click(), b.pairing.connectButton.click()]);
  await Promise.all([expect(a.transfer.connection).toContainText('Connected'), expect(b.transfer.connection).toContainText('Connected')]);
}

/** A real file of the given size with random content; returns its path and SHA-256. */
export async function makeFile(dir: string, name: string, size: number): Promise<{ path: string; sha256: string }> {
  mkdirSync(dir, { recursive: true });
  const file = path.join(dir, name);
  const out = createWriteStream(file);
  const hash = createHash('sha256');
  for (let left = size; left > 0; ) {
    const chunk = randomBytes(Math.min(left, 1 << 20));
    hash.update(chunk);
    if (!out.write(chunk)) await once(out, 'drain');
    left -= chunk.length;
  }
  out.end();
  await once(out, 'finish');
  return { path: file, sha256: hash.digest('hex') };
}

export async function sha256Of(file: string): Promise<string> {
  const hash = createHash('sha256');
  for await (const chunk of createReadStream(file)) hash.update(chunk as Buffer);
  return hash.digest('hex');
}

async function newPeer(browser: Browser, name: 'alice' | 'bob', dir: string) {
  const context = await browser.newContext({ locale: 'en-US', reducedMotion: 'reduce' });
  const peer = new Peer(name, await context.newPage(), `http://127.0.0.1:${SHELLS[name]}`, path.join(dir, `${name}-downloads`));
  await peer.launch();
  return { peer, context };
}

export const test = base.extend<{ alice: Peer; bob: Peer; workDir: string }>({
  workDir: async ({}, use, testInfo) => use(testInfo.outputPath('files')),
  alice: async ({ browser, workDir }, use) => {
    const { peer, context } = await newPeer(browser, 'alice', workDir);
    await use(peer);
    await context.close();
  },
  bob: async ({ browser, workDir }, use) => {
    const { peer, context } = await newPeer(browser, 'bob', workDir);
    await use(peer);
    await context.close();
  },
});

export { expect };
