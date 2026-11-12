// Cliente HTTP. El navegador no maneja tokens: la sesión es la cookie del perímetro.
import { ApiError, SESSION_MESSAGE, networkError, readError } from './errors.ts'

export type QueryParams = Record<string, string | number | boolean | undefined>

interface RequestOptions {
  query?: QueryParams
  json?: unknown
  form?: FormData
}

/** Punto único por donde se sale de la SPA; las pruebas lo espían. */
export const navigation = {
  assign(url: string) {
    window.location.assign(url)
  },
}

export function apiUrl(path: string, query?: QueryParams): string {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query ?? {})) {
    if (value !== undefined && value !== '') params.set(key, String(value))
  }
  const search = params.toString()
  return search ? `${path}?${search}` : path
}

function redirectToLogin(): void {
  const current = window.location.pathname + window.location.search + window.location.hash
  navigation.assign(`/oauth2/start?rd=${encodeURIComponent(current)}`)
}

/** Devuelve la respuesta ya comprobada; cualquier fallo sale como ApiError. */
export async function requestRaw(method: string, path: string, options: RequestOptions = {}): Promise<Response> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  let body: BodyInit | undefined
  if (options.form) {
    body = options.form
  } else if (options.json !== undefined) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(options.json)
  }
  let response: Response
  try {
    response = await fetch(apiUrl(path, options.query), { method, headers, body, credentials: 'same-origin' })
  } catch {
    throw networkError()
  }
  if (response.status === 401) {
    redirectToLogin()
    throw new ApiError(401, 'UNAUTHENTICATED', SESSION_MESSAGE)
  }
  if (!response.ok) throw await readError(response)
  return response
}

async function request<T>(method: string, path: string, options?: RequestOptions): Promise<T> {
  const response = await requestRaw(method, path, options)
  if (response.status === 204) return undefined as T
  try {
    return (await response.json()) as T
  } catch {
    throw new ApiError(response.status, 'UNEXPECTED', 'El servidor respondió algo que no se pudo leer.')
  }
}

export const api = {
  get: <T>(path: string, query?: QueryParams) => request<T>('GET', path, { query }),
  post: <T>(path: string, json: unknown) => request<T>('POST', path, { json }),
  put: <T>(path: string, json: unknown) => request<T>('PUT', path, { json }),
  patch: <T>(path: string, json: unknown) => request<T>('PATCH', path, { json }),
  delete: (path: string) => request<void>('DELETE', path),
  upload: <T>(path: string, form: FormData) => request<T>('POST', path, { form }),
}
