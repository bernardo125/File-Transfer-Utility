import type { Page } from '@playwright/test';

export class TransferPage {
  constructor(private readonly page: Page) {}

  /** The phase headline ("Incoming file", "Delivered", …). */
  get headline() {
    return this.page.getByRole('heading', { level: 1 });
  }
  get connection() {
    return this.page.getByRole('status').filter({ hasText: /connected|connection lost|not connected|reaching|securing/i });
  }
  get progress() {
    return this.page.getByRole('progressbar', { name: 'Transfer progress' });
  }
  get details() {
    return this.page.getByRole('table', { name: 'Current transfer' });
  }
  get error() {
    return this.page.getByRole('alert');
  }
  get chooseFile() {
    return this.page.getByRole('button', { name: /choose (a|another) file/i });
  }
  get sendAnother() {
    return this.page.getByRole('button', { name: 'Send another file' });
  }
  get accept() {
    return this.page.getByRole('button', { name: 'Accept file' });
  }
  get decline() {
    return this.page.getByRole('button', { name: 'Decline' });
  }
  get done() {
    return this.page.getByRole('button', { name: 'Done' });
  }
  get showInFolder() {
    return this.page.getByRole('button', { name: 'Show in folder' });
  }
  get waitForAnother() {
    return this.page.getByRole('button', { name: 'Wait for another file' });
  }
  get disconnect() {
    return this.page.getByRole('button', { name: /^disconnect$|stop transfer/i });
  }
  /** A cell of the transfer row (name, size, status…). */
  cell(text: string | RegExp) {
    return this.details.getByRole('cell').filter({ hasText: text });
  }
}
