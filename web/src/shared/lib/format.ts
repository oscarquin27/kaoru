// Formato de fechas, tiempos y tamaños. Todo en español y con la zona del navegador.

const DATE = new Intl.DateTimeFormat('es', { day: 'numeric', month: 'short', year: 'numeric' })
const DATE_TIME = new Intl.DateTimeFormat('es', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
const LONG_DATE = new Intl.DateTimeFormat('es', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })

export function formatDate(iso: string | undefined): string {
  return iso ? DATE.format(new Date(iso)) : 'Por definir'
}

// Un día de calendario («2026-11-12») no tiene hora ni zona: se muestra tal cual, sin correrlo a la
// zona del navegador.
const DAY = new Intl.DateTimeFormat('es', { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'UTC' })

/** Día de calendario de un hito: acepta «2026-11-12» o un instante a medianoche UTC. */
export function formatDay(day: string): string {
  return DAY.format(new Date(`${day.slice(0, 10)}T00:00:00Z`))
}

export function formatDateTime(iso: string): string {
  return DATE_TIME.format(new Date(iso))
}

export function formatLongDate(date: Date): string {
  return LONG_DATE.format(date)
}

/** Tiempo en un estado, corto: «3 d». */
export function formatDays(days: number): string {
  return days < 1 ? '< 1 d' : `${Math.round(days)} d`
}

/** Tiempo en un estado, con palabras: «3 días». */
export function formatDaysLong(days: number): string {
  if (days < 1) return 'menos de 1 día'
  const rounded = Math.round(days)
  return rounded === 1 ? '1 día' : `${rounded} días`
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  const kb = bytes / 1024
  const value = kb < 1024 ? kb : kb / 1024
  return `${value.toLocaleString('es', { maximumFractionDigits: 1 })} ${kb < 1024 ? 'KB' : 'MB'}`
}

/** SHA-256 abreviado para tablas: «9f2c41ab…e07d». */
export function shortChecksum(checksum: string): string {
  return `${checksum.slice(0, 8)}…${checksum.slice(-4)}`
}

/** Valor para un campo `datetime-local` con la hora local. */
export function toLocalInput(date: Date): string {
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}

/** De un campo `datetime-local` a fecha ISO; vacío si el campo está vacío. */
export function fromLocalInput(value: string): string | undefined {
  return value ? new Date(value).toISOString() : undefined
}

/** Mediana de una lista de números; 0 si está vacía. */
export function median(values: number[]): number {
  if (values.length === 0) return 0
  const sorted = [...values].sort((a, b) => a - b)
  const middle = Math.floor(sorted.length / 2)
  return sorted.length % 2 === 1 ? (sorted[middle] ?? 0) : ((sorted[middle - 1] ?? 0) + (sorted[middle] ?? 0)) / 2
}
