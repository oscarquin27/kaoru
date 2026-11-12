import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router'
import { App } from './app/App.tsx'
import { applyPreferences } from './app/preferences.ts'
import { createQueryClient } from './shared/api/query.ts'
import './styles/app.css'

applyPreferences()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <App queryClient={createQueryClient()} />
    </BrowserRouter>
  </StrictMode>,
)
