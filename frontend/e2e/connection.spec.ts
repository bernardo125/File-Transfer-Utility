import { test, expect, pair } from './fixtures/peers';

test('pasting garbage as the peer token is rejected by the real backend and nothing hangs', async ({ alice }) => {
  await alice.startAs('send');
  await alice.pairing.peerTokenInput.fill('definitely-not-a-valid-token-but-long-enough-to-pass-the-length-check');
  await alice.pairing.connectButton.click();

  await expect(alice.pairing.error).toContainText(/token/i);
  await expect(alice.pairing.connectButton).toBeVisible(); // back on the form, not stuck "connecting"
});

test('using your own token is refused', async ({ alice }) => {
  await alice.startAs('send');
  await alice.pairing.peerTokenInput.fill(await alice.token());
  await alice.pairing.connectButton.click();
  await expect(alice.pairing.error).toBeVisible();
  await expect(alice.pairing.connectButton).toBeVisible();
});

test('cancelling a connection attempt leaves the app usable: the same peers can then pair', async ({ alice, bob }) => {
  await Promise.all([alice.startAs('send'), bob.startAs('receive')]);
  const [tokenA, tokenB] = [await alice.token(), await bob.token()];

  // Alice dials but Bob has not pressed Connect yet.
  await alice.pairing.peerTokenInput.fill(tokenB);
  await alice.pairing.connectButton.click();
  await expect(alice.pairing.cancelButton).toBeVisible();
  await alice.pairing.cancelButton.click();
  await expect(alice.pairing.connectButton).toBeVisible();
  await expect(alice.pairing.error).toHaveCount(0);

  // Now both really connect.
  await bob.pairing.peerTokenInput.fill(tokenA);
  await Promise.all([alice.pairing.connectButton.click(), bob.pairing.connectButton.click()]);
  await Promise.all([expect(alice.transfer.connection).toContainText('Connected'), expect(bob.transfer.connection).toContainText('Connected')]);
});

test('the peer closes the app while connected and idle: the other UI says the connection was lost', async ({ alice, bob }) => {
  await pair(alice, bob);
  await bob.crash();

  await expect(alice.transfer.connection).toContainText('Connection lost', { timeout: 45_000 });
  await expect(alice.transfer.error).toContainText(/pair again/i);
  await expect(alice.transfer.chooseFile).toHaveCount(0); // can't pick a file for a dead link
});

test('Disconnect on one side is noticed by the other', async ({ alice, bob }) => {
  await pair(alice, bob);
  await alice.transfer.disconnect.click();
  await expect(alice.home.sendCard).toBeVisible();
  await expect(bob.transfer.connection).toContainText('Connection lost', { timeout: 45_000 });
});

test('a token pasted from a chat, with stray spaces and a trailing newline, still connects', async ({ alice, bob }) => {
  await Promise.all([alice.startAs('send'), bob.startAs('receive')]);
  const [tokenA, tokenB] = [await alice.token(), await bob.token()];
  await alice.pairing.peerTokenInput.fill(`  ${tokenB}  \n`);
  await bob.pairing.peerTokenInput.fill(`\n${tokenA} `);
  await Promise.all([alice.pairing.connectButton.click(), bob.pairing.connectButton.click()]);
  await Promise.all([expect(alice.transfer.connection).toContainText('Connected'), expect(bob.transfer.connection).toContainText('Connected')]);
});
