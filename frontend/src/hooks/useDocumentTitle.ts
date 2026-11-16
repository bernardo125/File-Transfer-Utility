import { useEffect } from 'react';

const APP_NAME = 'File Throw';

/** Keeps the window/document title in sync with the current screen (WCAG 2.4.2). */
export function useDocumentTitle(screen?: string) {
  useEffect(() => {
    document.title = screen ? `${screen} · ${APP_NAME}` : APP_NAME;
  }, [screen]);
}
