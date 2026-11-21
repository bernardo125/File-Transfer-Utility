import { fileURLToPath } from 'node:url';
import type { Locator, Page } from '@playwright/test';
import { test, expect, makeFile, confirmFingerprints } from '../fixtures/peers';

const out = (name: string) => fileURLToPath(new URL(`../../../docs/media/${name}.png`, import.meta.url));

// A headless screenshot has no mouse pointer, so draw one where the user would be pointing. The
// target is also hovered for real, so the control shows its hover state like it would for a person.
const ARROW =
  '<svg xmlns="http://www.w3.org/2000/svg" width="26" height="30" viewBox="0 0 13 20"><path d="M0.8 0.8 0.8 15.6 4.6 12.3 7.2 18.6 9.6 17.6 7.1 11.5 12 11.5Z" fill="#fff" stroke="#111" stroke-width="1.1" stroke-linejoin="round"/></svg>';

async function shotWithCursor(page: Page, target: Locator, file: string, at: [number, number] = [0.6, 0.65]) {
  await target.hover();
  const box = (await target.boundingBox())!;
  const x = box.x + box.width * at[0];
  const y = box.y + box.height * at[1];
  await page.evaluate(
    ({ x, y, svg }) => {
      const el = document.createElement('div');
      el.id = '__demo-cursor';
      el.innerHTML = svg;
      Object.assign(el.style, { position: 'fixed', left: `${x}px`, top: `${y}px`, zIndex: '2147483647', pointerEvents: 'none', filter: 'drop-shadow(0 2px 3px rgba(0,0,0,.4))' });
      document.body.appendChild(el);
    },
    { x, y, svg: ARROW },
  );
  await page.screenshot({ path: file });
  await page.evaluate(() => document.getElementById('__demo-cursor')?.remove());
}

// Not an assertion suite: it drives the real app (two real backends, real QUIC) through the
// happy path and saves what the user sees. Re-run it whenever the UI changes.
test('capture the happy path for the README', async ({ alice, bob, workDir }) => {
  test.setTimeout(180_000);
  const file = await makeFile(workDir, 'holiday-photos.zip', 400 * 1024 * 1024);

  await expect(alice.home.sendCard).toBeVisible();
  await shotWithCursor(alice.page, alice.home.sendCard, out('01-home'));

  await Promise.all([alice.startAs('send'), bob.startAs('receive')]);
  await alice.pairing.peerTokenInput.fill(await bob.token());
  await bob.pairing.peerTokenInput.fill(await alice.token());
  await shotWithCursor(alice.page, alice.pairing.connectButton, out('02-pairing'));

  await Promise.all([alice.pairing.connectButton.click(), bob.pairing.connectButton.click()]);
  await Promise.all([expect(alice.pairing.fingerprint).toBeVisible(), expect(bob.pairing.fingerprint).toBeVisible()]);
  await shotWithCursor(alice.page, alice.pairing.matchButton, out('03-fingerprint'));
  await confirmFingerprints(alice, bob);
  await Promise.all([expect(alice.transfer.connection).toContainText('Connected'), expect(bob.transfer.connection).toContainText('Connected')]);
  await shotWithCursor(alice.page, alice.transfer.chooseFile, out('04-connected'));

  await alice.willPick(file.path);
  await alice.transfer.chooseFile.click();
  await expect(bob.transfer.accept).toBeVisible();
  await shotWithCursor(bob.page, bob.transfer.accept, out('05-incoming'));

  await bob.transfer.accept.click();
  await expect.poll(async () => Number(await bob.transfer.progress.getAttribute('aria-valuenow')), { timeout: 60_000 }).toBeGreaterThanOrEqual(40);
  await shotWithCursor(bob.page, bob.transfer.cell('holiday-photos.zip'), out('06-transferring'), [0.1, 0.7]);

  await expect(bob.transfer.headline).toHaveText('File received', { timeout: 90_000 });
  await expect(alice.transfer.headline).toHaveText('Delivered');
  await shotWithCursor(bob.page, bob.transfer.showInFolder, out('07-received'));
});

