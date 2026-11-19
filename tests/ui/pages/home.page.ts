import type { Page } from '@playwright/test';

export class HomePage {
  constructor(private readonly page: Page) {}

  // Locator priority: role + accessible name > label > text > test id. Anchored regexes because
  // the button's accessible name also contains its description.
  get sendCard() {
    return this.page.getByRole('button', { name: /^send\b/i });
  }
  get receiveCard() {
    return this.page.getByRole('button', { name: /^receive\b/i });
  }
}
