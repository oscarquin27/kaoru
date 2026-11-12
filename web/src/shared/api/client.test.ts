import { afterEach, describe, expect, it, vi } from 'vitest'
import { problemResponse } from '../../test-utils.tsx'
import { api, navigation } from './client.ts'
import { ApiError, isRetryable } from './errors.ts'

function stubFetch(response: Response | Error) {
  const fetchMock = vi.fn<(url: string, init?: RequestInit) => Promise<Response>>(async () => {
    if (response instanceof Error) throw response
    return response
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

async function failure(promise: Promise<unknown>): Promise<ApiError> {
  const error = await promise.then(
    () => undefined,
    (caught: unknown) => caught,
  )
  expect(error).toBeInstanceOf(ApiError)
  return error as ApiError
}

describe('lector único de errores', () => {
  afterEach(() => window.history.replaceState(null, '', '/'))

  it('envía la cookie de sesión del mismo origen y nunca un token', async () => {
    const fetchMock = stubFetch(new Response('{"items":[]}', { headers: { 'Content-Type': 'application/json' } }))
    await api.get('/api/systems', { q: 'límite', empty: '', missing: undefined })
    expect(fetchMock).toHaveBeenCalledWith('/api/systems?q=l%C3%ADmite', expect.objectContaining({ method: 'GET', credentials: 'same-origin' }))
    expect(fetchMock.mock.calls[0]?.[1]?.headers).not.toHaveProperty('Authorization')
  })

  it('lee un problem+json: code, detail y traceId', async () => {
    stubFetch(problemResponse(409, { code: 'INVALID_ACTION', detail: 'Falta 1 TC sin RUN aprobado: TC-2026-0003.' }))
    const error = await failure(api.post('/api/requirements/REQ-2026-0001/transitions', { to: 'approved_qa' }))
    expect(error).toMatchObject({ status: 409, code: 'INVALID_ACTION', message: 'Falta 1 TC sin RUN aprobado: TC-2026-0003.', traceId: 'trace-1' })
  })

  it('lee los errors[] de un 422 y deja arriba un texto corto', async () => {
    const errors = [{ field: 'title', code: 'required', message: 'Escribe un título.' }]
    stubFetch(problemResponse(422, { code: 'VALIDATION', detail: 'validation failed', errors }))
    const error = await failure(api.post('/api/requirements', {}))
    expect(error.fieldErrors).toEqual(errors)
    expect(error.message).toBe('Revisa los campos marcados.')
  })

  it('con un 401 redirige al inicio de sesión con la ruta actual', async () => {
    const assign = vi.spyOn(navigation, 'assign').mockImplementation(() => undefined)
    window.history.replaceState(null, '', '/requerimientos/REQ-2026-0001?x=1#pruebas')
    stubFetch(new Response('', { status: 401 }))
    const error = await failure(api.get('/api/me'))
    expect(assign).toHaveBeenCalledWith(`/oauth2/start?rd=${encodeURIComponent('/requerimientos/REQ-2026-0001?x=1#pruebas')}`)
    expect(error.message).toBe('Tu sesión terminó. Te llevamos al inicio de sesión.')
    expect(isRetryable(error)).toBe(false)
  })

  it('un fallo de red no muestra el mensaje técnico del navegador', async () => {
    stubFetch(new TypeError('Failed to fetch'))
    const error = await failure(api.get('/api/me'))
    expect(error.message).toBe('No hay conexión con el servidor. Revisa tu red e inténtalo de nuevo.')
    expect(error.message).not.toContain('Failed to fetch')
    expect(isRetryable(error)).toBe(true)
  })

  it('una respuesta de error que no es problem+json sale con texto en español', async () => {
    stubFetch(new Response('<html>502 Bad Gateway</html>', { status: 502, headers: { 'Content-Type': 'text/html' } }))
    const error = await failure(api.get('/api/me'))
    expect(error.message).toBe('El servidor no responde. Inténtalo de nuevo en un momento.')
    expect(isRetryable(error)).toBe(true)
  })

  it('una respuesta 200 que no es JSON tampoco deja pasar el error técnico', async () => {
    stubFetch(new Response('<html>', { status: 200, headers: { 'Content-Type': 'text/html' } }))
    const error = await failure(api.get('/api/me'))
    expect(error.message).toBe('El servidor respondió algo que no se pudo leer.')
  })

  it.each([
    [400, false],
    [403, false],
    [404, false],
    [409, false],
    [422, false],
    [500, false],
    [502, true],
    [503, true],
    [504, true],
  ])('un %i se reintenta: %s', (status, retryable) => {
    expect(isRetryable(new ApiError(status, 'X', 'x'))).toBe(retryable)
  })
})
