import path from 'node:path';
import { test, expect, pair, makeFile, sha256Of } from './fixtures/peers';

// Real UI <-> real Go backend <-> real UDP hole punch + QUIC <-> real Go backend <-> real UI.
// Each assertion about a transfer ends on the receiver's disk: bytes and SHA-256, not just a green banner.

test('Alice sends a file to Bob: both UIs follow the transfer and Bob gets identical bytes', async ({ alice, bob, workDir }) => {
  const file = await makeFile(workDir, 'photo.bin', 5 * 1024 * 1024 + 7);
  await pair(alice, bob);

  await alice.willPick(file.path);
  await alice.transfer.chooseFile.click();
  await expect(bob.transfer.headline).toHaveText('Incoming file');
  await expect(bob.transfer.cell('photo.bin')).toBeVisible();
  await expect(bob.transfer.cell('5.00 MB')).toBeVisible();
  await expect(alice.transfer.headline).toHaveText('Waiting for your peer');

  await bob.transfer.accept.click();
  await expect(alice.transfer.headline).toHaveText('Delivered');
  await expect(bob.transfer.headline).toHaveText('File received');
  await expect(bob.transfer.progress).toHaveAttribute('aria-valuenow', '100');
  await expect(bob.page.getByText('verified')).toBeVisible();

  const saved = path.join(bob.downloadDir, 'photo.bin');
  expect(bob.receivedFiles()).toEqual(['photo.bin']);
  expect(await sha256Of(saved)).toBe(file.sha256);
});

test('Bob declines: Alice sees it declined and nothing is written to Bob’s disk', async ({ alice, bob, workDir }) => {
  const file = await makeFile(workDir, 'secret.bin', 200_000);
  await pair(alice, bob);

  await alice.willPick(file.path);
  await alice.transfer.chooseFile.click();
  await bob.transfer.decline.click();

  await expect(alice.transfer.headline).toHaveText('Transfer declined');
  await expect(bob.transfer.headline).toHaveText('Transfer declined');
  await expect(alice.transfer.error).toHaveCount(0);
  expect(bob.receivedFiles()).toEqual([]);
});

test('several files in one session: each arrives intact and the second offer is not tangled with the first', async ({ alice, bob, workDir }) => {
  const first = await makeFile(workDir, 'one.bin', 300_000);
  const second = await makeFile(workDir, 'two.bin', 70_001);
  await pair(alice, bob);

  await alice.willPick(first.path);
  await alice.transfer.chooseFile.click();
  await bob.transfer.accept.click();
  await expect(alice.transfer.headline).toHaveText('Delivered');
  await expect(bob.transfer.headline).toHaveText('File received');
  await bob.transfer.done.click();
  await expect(bob.transfer.headline).toHaveText('Ready to catch');

  await alice.willPick(second.path);
  await alice.transfer.sendAnother.click();
  await expect(bob.transfer.headline).toHaveText('Incoming file');
  await expect(bob.transfer.cell('two.bin')).toBeVisible();
  await bob.transfer.accept.click();
  await expect(alice.transfer.headline).toHaveText('Delivered');
  await expect(bob.transfer.headline).toHaveText('File received');

  expect(bob.receivedFiles().sort()).toEqual(['one.bin', 'two.bin']);
  expect(await sha256Of(path.join(bob.downloadDir, 'one.bin'))).toBe(first.sha256);
  expect(await sha256Of(path.join(bob.downloadDir, 'two.bin'))).toBe(second.sha256);
});

test('an empty file is a valid transfer', async ({ alice, bob, workDir }) => {
  const file = await makeFile(workDir, 'empty.txt', 0);
  await pair(alice, bob);
  await alice.willPick(file.path);
  await alice.transfer.chooseFile.click();
  await bob.transfer.accept.click();

  await expect(alice.transfer.headline).toHaveText('Delivered');
  await expect(bob.transfer.headline).toHaveText('File received');
  expect(await sha256Of(path.join(bob.downloadDir, 'empty.txt'))).toBe(file.sha256);
});

test('a 128 MiB file completes and is byte-identical', async ({ alice, bob, workDir }) => {
  const file = await makeFile(workDir, 'big.bin', 128 * 1024 * 1024);
  await pair(alice, bob);
  await alice.willPick(file.path);
  await alice.transfer.chooseFile.click();
  await bob.transfer.accept.click();

  await expect(alice.transfer.headline).toHaveText('Delivered', { timeout: 90_000 });
  await expect(bob.transfer.headline).toHaveText('File received');
  expect(await sha256Of(path.join(bob.downloadDir, 'big.bin'))).toBe(file.sha256);
});

test('"Show in folder" opens the folder that really contains the received file', async ({ alice, bob, workDir }) => {
  const file = await makeFile(workDir, 'doc.bin', 10_000);
  await pair(alice, bob);
  await alice.willPick(file.path);
  await alice.transfer.chooseFile.click();
  await bob.transfer.accept.click();
  await expect(bob.transfer.headline).toHaveText('File received');

  await bob.transfer.showInFolder.click();
  await expect.poll(() => bob.openedFolders()).toEqual([path.join(bob.downloadDir, 'doc.bin')]);
});

test('the sender dies mid-transfer: Bob’s UI reports a failure and no partial file is left', async ({ alice, bob, workDir }) => {
  const file = await makeFile(workDir, 'huge.bin', 512 * 1024 * 1024);
  await pair(alice, bob);
  await alice.willPick(file.path);
  await alice.transfer.chooseFile.click();
  await bob.transfer.accept.click();
  await expect(bob.transfer.headline).toHaveText('Receiving');
  await expect.poll(async () => Number(await bob.transfer.progress.getAttribute('aria-valuenow'))).toBeGreaterThan(0);

  await alice.crash(); // Alice's app closes: socket and QUIC session vanish

  await expect(bob.transfer.headline).toHaveText('Transfer failed', { timeout: 45_000 });
  await expect(bob.page.getByText('verified')).toHaveCount(0);
  expect(bob.receivedFiles()).toEqual([]);
});
