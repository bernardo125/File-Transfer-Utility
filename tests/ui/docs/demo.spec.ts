import { fileURLToPath } from 'node:url';
import { copyFileSync, mkdirSync } from 'node:fs';
import path from 'node:path';
import { test, expect, makeFile, Peer } from '../fixtures/peers';
import { SHELLS } from '../playwright.config';

const media = fileURLToPath(new URL('../../../docs/media/', import.meta.url));

// Not an assertion suite: it records the README demo from the receiver's point of view, in a single
// window. The sender runs in a second, unrecorded window. The pauses are deliberate: they let a
// viewer read each step. Run with `npm run docs:video`.
test('record the README demo video', async ({ browser, workDir }, testInfo) => {
  test.setTimeout(240_000);
  const file = await makeFile(workDir, 'holiday-photos.zip', 600 * 1024 * 1024);
  const origin = (p: number) => `http://127.0.0.1:${p}`;

  const senderCtx = await browser.newContext({ locale: 'en-US' });
  const recorded = await browser.newContext({
    locale: 'en-US',
    viewport: { width: 1100, height: 700 },
    recordVideo: { dir: testInfo.outputPath('video'), size: { width: 1100, height: 700 } },
  });
  const alice = new Peer('alice', await senderCtx.newPage(), origin(SHELLS.alice), path.join(workDir, 'alice-downloads'));
  const bob = new Peer('bob', await recorded.newPage(), origin(SHELLS.bob), path.join(workDir, 'bob-downloads'));
  await Promise.all([alice.launch(), bob.launch()]);
  const beat = (ms = 2000) => bob.page.waitForTimeout(ms);

  await expect(bob.home.receiveCard).toBeVisible();
  await beat(2500);
  await bob.home.receiveCard.click();
  await alice.startAs('send');
  await expect(bob.pairing.localToken).not.toHaveText(/Discovering|Unavailable/);
  await beat(2500);

  await bob.pairing.peerTokenInput.fill(await alice.token());
  await alice.pairing.peerTokenInput.fill(await bob.token());
  await beat(2000);
  await Promise.all([bob.pairing.connectButton.click(), alice.pairing.connectButton.click()]);
  await expect(bob.transfer.connection).toContainText('Connected');
  await beat(2500);

  await alice.willPick(file.path);
  await alice.transfer.chooseFile.click();
  await expect(bob.transfer.accept).toBeVisible();
  await beat(3000);

  await bob.transfer.accept.click();
  await expect(bob.transfer.headline).toHaveText('File received', { timeout: 120_000 });
  await beat(3500);

  await recorded.close(); // finalizes the video
  mkdirSync(media, { recursive: true });
  copyFileSync(await bob.page.video()!.path(), path.join(media, 'demo.webm'));
  await senderCtx.close();
});
