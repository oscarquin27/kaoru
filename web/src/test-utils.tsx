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

export function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

/** Una lista paginada con todo en la primera página. */
export function pageOf<T>(items: T[]) {
  return { items, page: 1, size: 25, total: items.length }
}

type Reply = unknown | ((url: string) => unknown)

/**
 * Respuestas en línea para `override`, por método y ruta sin filtros: `{ 'GET /api/builds': pageOf([...]) }`.
 * Un valor que no es una `Response` se responde como JSON; una función recibe la URL completa.
 */
export function routes(table: Record<string, Reply>): (method: string, url: string) => Response | undefined {
  return (method, url) => {
    const key = `${method} ${url.split('?')[0]}`
    if (!(key in table)) return undefined
    const value = table[key]
    const reply: unknown = typeof value === 'function' ? (value as (url: string) => unknown)(url) : value
    return reply instanceof Response ? reply : jsonResponse(reply)
  }
}

/**
 * Monta la aplicación completa en una ruta, con un usuario que tiene los permisos dados.
 * `override` responde lo que las grabaciones no tienen (las pantallas nuevas).
 */
export function renderApp(path: string, permissions: string[] = fixtureMe.permissions, override?: (method: string, url: string) => Response | undefined) {
  const calls = stubApi((method, url) => (method === 'GET' && url === '/api/me' ? jsonResponse({ ...fixtureMe, permissions }) : override?.(method, url)))
  render(
    <MemoryRouter initialEntries={[path]}>
      <App queryClient={createQueryClient()} />
    </MemoryRouter>,
  )
  return calls
}
