// Único lector de errores (reglas §A.16.2). Todo fallo de la API, de la red o de una respuesta
// inesperada sale de aquí como ApiError con texto en español; ninguna pantalla muestra el
// mensaje técnico del navegador.

export interface FieldError {
  field: string
  code: string
  message: string
}

export class ApiError extends Error {
  /** Código HTTP; 0 si no hubo respuesta. */
  status: number
  code: string
  fieldErrors: FieldError[]
  traceId: string

  constructor(status: number, code: string, message: string, fieldErrors: FieldError[] = [], traceId = '') {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fieldErrors = fieldErrors
    this.traceId = traceId
  }
}

const NETWORK_MESSAGE = 'No hay conexión con el servidor. Revisa tu red e inténtalo de nuevo.'
const UNEXPECTED_MESSAGE = 'Algo salió mal. Inténtalo de nuevo; si sigue igual, avisa a soporte.'
export const SESSION_MESSAGE = 'Tu sesión terminó. Te llevamos al inicio de sesión.'

const STATUS_MESSAGES: Record<number, string> = {
  400: 'La solicitud no es válida.',
  401: SESSION_MESSAGE,
  403: 'No tienes permiso para hacer esto.',
  404: 'No existe o ya no está disponible.',
  409: 'No se puede hacer en el estado actual.',
  413: 'El archivo es demasiado grande.',
  415: 'Ese tipo de archivo no está permitido.',
  422: 'Revisa los campos marcados.',
  502: 'El servidor no responde. Inténtalo de nuevo en un momento.',
  503: 'El servicio no está disponible. Inténtalo de nuevo en un momento.',
  504: 'El servidor tardó demasiado. Inténtalo de nuevo en un momento.',
}

function statusMessage(status: number): string {
  return STATUS_MESSAGES[status] ?? UNEXPECTED_MESSAGE
}

function asText(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function asFieldErrors(value: unknown): FieldError[] {
  if (!Array.isArray(value)) return []
  return value.flatMap((item: unknown) => {
    if (typeof item !== 'object' || item === null) return []
    const { field, code, message } = item as Record<string, unknown>
    return asText(field) && asText(message) ? [{ field: asText(field), code: asText(code), message: asText(message) }] : []
  })
}

/** Convierte una respuesta con error en ApiError. Sirve para problem+json y para lo que no lo es. */
export async function readError(response: Response): Promise<ApiError> {
  const contentType = response.headers.get('Content-Type') ?? ''
  if (!contentType.includes('application/problem+json')) {
    return new ApiError(response.status, 'UNEXPECTED', statusMessage(response.status))
  }
  let problem: Record<string, unknown>
  try {
    problem = (await response.json()) as Record<string, unknown>
  } catch {
    return new ApiError(response.status, 'UNEXPECTED', statusMessage(response.status))
  }
  const fieldErrors = asFieldErrors(problem.errors)
  // Con errores de campo, el detalle ya va bajo cada campo: arriba basta el texto corto.
  const message = fieldErrors.length > 0 ? statusMessage(422) : asText(problem.detail) || asText(problem.title) || statusMessage(response.status)
  return new ApiError(response.status, asText(problem.code) || 'UNEXPECTED', message, fieldErrors, asText(problem.traceId))
}

export function networkError(): ApiError {
  return new ApiError(0, 'NETWORK', NETWORK_MESSAGE)
}

/** Cualquier cosa lanzada (también un fallo al pintar) termina en un ApiError con texto en español. */
export function toApiError(error: unknown): ApiError {
  return error instanceof ApiError ? error : new ApiError(0, 'UNEXPECTED', UNEXPECTED_MESSAGE)
}

/** Una lectura se repite solo si puede cambiar al repetirla: sin conexión, 502, 503 o 504. */
export function isRetryable(error: unknown): boolean {
  if (!(error instanceof ApiError)) return false
  return error.code === 'NETWORK' || [502, 503, 504].includes(error.status)
}
