import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App'

// The browser-mode mock exists for `npm run dev` only. Gating on DEV keeps it (and its fake
// token/transfer flows) out of the production bundle entirely.
if (import.meta.env.DEV) {
  void import('./services/devMock').then((m) => m.initWailsMock())
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
