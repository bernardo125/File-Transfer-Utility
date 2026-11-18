import type { Page } from '@playwright/test';

export class PairingPage {
  constructor(private readonly page: Page) {}

  get heading() {
    return this.page.getByRole('heading', { level: 1 });
  }
  get backButton() {
    return this.page.getByRole('button', { name: 'Back' });
  }
  // Label: the <output> is named via aria-labelledby, the textarea via <label>.
  get localToken() {
    return this.page.getByLabel('Your token');
  }
  get peerTokenInput() {
    return this.page.getByLabel('Peer token');
  }
  get connectButton() {
    return this.page.getByRole('button', { name: 'Connect', exact: true });
  }
  get cancelButton() {
    return this.page.getByRole('button', { name: 'Cancel' });
  }
  get error() {
    return this.page.getByRole('alert');
  }
  /** The connection stage list; the active stage carries aria-current="step". */
  stage(label: string | RegExp) {
    return this.page.getByRole('listitem').filter({ hasText: label });
  }
}
