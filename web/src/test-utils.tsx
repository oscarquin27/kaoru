// Ayudas de las pruebas: la API se simula con las respuestas reales grabadas en mocks/fixtures.json.
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { vi } from 'vitest'
import { mockResponse, type MockResponse } from '../mocks/handler.ts'
import fixtures from '../mocks/fixtures.json' with { type: 'json' }
import { App } from './app/App.tsx'
import { createQueryClient } from './shared/api/query.ts'
import type { Me } from './shared/api/types.ts'

export const fixtureMe = fixtures['GET /api/me'] as Me

export function problemResponse(status: number, body: Record<string, unknown>): Response {
  return new Response(JSON.stringify({ type: 'about:blank', title: 'Error', status, traceId: 'trace-1', ...body }), {
    status,
    headers: { 'Content-Type': 'application/problem+json' },
  })
}

/** Sustituye `fetch` por el servidor de ejemplo. `override` responde antes que las grabaciones. */
export function stubApi(override?: (method: string, url: string) => Response | undefined) {
  const calls: { method: string; url: string; body: unknown }[] = []
  vi.stubGlobal('fetch', async (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    calls.push({ method, url, body: typeof init?.body === 'string' ? JSON.parse(init.body) : init?.body })
    const custom = override?.(method, url)
    if (custom) return custom
    const recorded: MockResponse = mockResponse(method, url)
    return new Response(recorded.body, { status: recorded.status, headers: { 'Content-Type': recorded.contentType } })
  })
  return calls
}

/** Monta la aplicación completa en una ruta, con un usuario que tiene los permisos dados. */
export function renderApp(path: string, permissions: string[] = fixtureMe.permissions) {
  const calls = stubApi((method, url) =>
    method === 'GET' && url === '/api/me' ? new Response(JSON.stringify({ ...fixtureMe, permissions }), { headers: { 'Content-Type': 'application/json' } }) : undefined,
  )
  render(
    <MemoryRouter initialEntries={[path]}>
      <App queryClient={createQueryClient()} />
    </MemoryRouter>,
  )
  return calls
}
