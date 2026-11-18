import AxeBuilder from '@axe-core/playwright';
import type { Page } from '@playwright/test';
import { test, expect, pair, makeFile } from './fixtures/peers';

// WCAG 2.x A/AA on the screens a real session reaches (real token, real connection, real offer).
const violations = async (page: Page) =>
  (await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze()).violations.map((v) => `${v.id}: ${v.help} (${v.nodes.length})`);

test('home and pairing screens have no WCAG A/AA violations', async ({ alice }) => {
  await expect(alice.home.sendCard).toBeVisible();
  expect(await violations(alice.page)).toEqual([]);
  await alice.startAs('send');
  expect(await violations(alice.page)).toEqual([]);
});

test('transfer screens (incoming offer, receiving, received) have no WCAG A/AA violations', async ({ alice, bob, workDir }) => {
  const file = await makeFile(workDir, 'a11y.bin', 1_000_000);
  await pair(alice, bob);
  await alice.willPick(file.path);
  await alice.transfer.chooseFile.click();
  await expect(bob.transfer.accept).toBeVisible();
  expect(await violations(bob.page)).toEqual([]);
  await bob.transfer.accept.click();
  await expect(bob.transfer.headline).toHaveText('File received');
  expect(await violations(bob.page)).toEqual([]);
  expect(await violations(alice.page)).toEqual([]);
});
