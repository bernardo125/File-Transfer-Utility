import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { initWailsMock } from './services/devMock'
import './index.css'
import App from './App'

// Initialize development mock if running outside of Wails native desktop webview
initWailsMock()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
