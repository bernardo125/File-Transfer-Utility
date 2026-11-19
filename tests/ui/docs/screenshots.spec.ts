import { fileURLToPath } from 'node:url';
import { test, expect, makeFile } from '../fixtures/peers';

const out = (name: string) => fileURLToPath(new URL(`../../../docs/media/${name}.png`, import.meta.url));

// Not an assertion suite: it drives the real app (two real backends, real QUIC) through the
// happy path and saves what the user sees. Re-run it whenever the UI changes.
test('capture the happy path for the README', async ({ alice, bob, workDir }) => {
  test.setTimeout(180_000);
  const file = await makeFile(workDir, 'holiday-photos.zip', 400 * 1024 * 1024);

  await expect(alice.home.sendCard).toBeVisible();
  await alice.page.screenshot({ path: out('01-home') });

  await Promise.all([alice.startAs('send'), bob.startAs('receive')]);
  await alice.pairing.peerTokenInput.fill(await bob.token());
  await bob.pairing.peerTokenInput.fill(await alice.token());
  await alice.page.screenshot({ path: out('02-pairing') });

  await Promise.all([alice.pairing.connectButton.click(), bob.pairing.connectButton.click()]);
  await Promise.all([expect(alice.transfer.connection).toContainText('Connected'), expect(bob.transfer.connection).toContainText('Connected')]);
  await alice.page.screenshot({ path: out('03-connected') });

  await alice.willPick(file.path);
  await alice.transfer.chooseFile.click();
  await expect(bob.transfer.accept).toBeVisible();
  await bob.page.screenshot({ path: out('04-incoming') });

  await bob.transfer.accept.click();
  await expect.poll(async () => Number(await bob.transfer.progress.getAttribute('aria-valuenow')), { timeout: 60_000 }).toBeGreaterThanOrEqual(40);
  await bob.page.screenshot({ path: out('05-transferring') });

  await expect(bob.transfer.headline).toHaveText('File received', { timeout: 90_000 });
  await expect(alice.transfer.headline).toHaveText('Delivered');
  await bob.page.screenshot({ path: out('06-received') });
});

