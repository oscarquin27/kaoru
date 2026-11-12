// Servidor de datos de ejemplo: responde /api/* con las respuestas grabadas en fixtures.json.
// Solo para `npm run dev` y las pruebas. No guarda nada.
import fixtures from './fixtures.json' with { type: 'json' }

export interface MockResponse {
  status: number
  contentType: string
  body: string
}

const recorded = fixtures as unknown as Record<string, unknown>

function problem(status: number, code: string, title: string, detail: string): MockResponse {
  return {
    status,
    contentType: 'application/problem+json',
    body: JSON.stringify({ type: 'about:blank', title, status, code, detail, traceId: 'ejemplo' }),
  }
}

// La paginación no cambia la grabación: se ignora al comparar.
function canonical(url: string): string {
  const parsed = new URL(url, 'http://ejemplo.invalid')
  parsed.searchParams.delete('page')
  parsed.searchParams.delete('size')
  parsed.searchParams.sort()
  return parsed.pathname + parsed.search
}

const index = new Map<string, { kind: string; value: unknown }>()
for (const [key, value] of Object.entries(recorded)) {
  const [kind, url] = key.split(' ')
  if (url?.startsWith('/api/')) index.set(canonical(url), { kind: kind ?? '', value })
}

export function mockResponse(method: string, url: string): MockResponse {
  if (method !== 'GET') {
    return problem(409, 'INVALID_ACTION', 'Acción no permitida', 'Servidor de datos de ejemplo: no guarda cambios.')
  }
  const hit = index.get(canonical(url))
  if (!hit) {
    return problem(404, 'NOT_FOUND', 'No encontrado', 'Servidor de datos de ejemplo: esta consulta no está grabada.')
  }
  if (hit.kind === 'TEXT') {
    return { status: 200, contentType: 'text/markdown; charset=utf-8', body: String(hit.value) }
  }
  return { status: 200, contentType: 'application/json', body: JSON.stringify(hit.value) }
}
