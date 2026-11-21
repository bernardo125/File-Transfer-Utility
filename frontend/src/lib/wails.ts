/** True when the native Wails Go binding for the given App method is injected. */
export function hasBackendMethod(method: string): boolean {
  const app = (window as unknown as { go?: { ui?: { App?: Record<string, unknown> } } }).go?.ui?.App;
  return typeof app?.[method] === 'function';
}

/** Normalises errors thrown by Wails bindings (often plain strings) into a message. */
export function errorMessage(err: unknown): string {
  if (err instanceof Error) return err.message;
  if (typeof err === 'string') return err;
  return 'Unknown error';
}
