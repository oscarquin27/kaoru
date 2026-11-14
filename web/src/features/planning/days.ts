// Días como texto 2026-11-30, en UTC: así los guarda y los devuelve la API.

export function today(now = new Date()): string {
  return now.toISOString().slice(0, 10)
}

export function addDays(day: string, days: number): string {
  const date = new Date(`${day}T00:00:00Z`)
  date.setUTCDate(date.getUTCDate() + days)
  return date.toISOString().slice(0, 10)
}

/** Días de `from` a `to`; 0 si son el mismo. */
export function daysBetween(from: string, to: string): number {
  return Math.round((Date.parse(`${to}T00:00:00Z`) - Date.parse(`${from}T00:00:00Z`)) / 86_400_000)
}

/** Todos los días de `from` a `to`, ambos incluidos. */
export function daysOf(from: string, to: string): string[] {
  const out: string[] = []
  for (let day = from; day <= to; day = addDays(day, 1)) out.push(day)
  return out
}

/** El lunes de la semana del día. */
export function mondayOf(day: string): string {
  const weekday = new Date(`${day}T00:00:00Z`).getUTCDay()
  return addDays(day, -((weekday + 6) % 7))
}

const SHORT = new Intl.DateTimeFormat('es', { day: 'numeric', month: 'short', timeZone: 'UTC' })

/** «del 16 nov al 27 nov», o «el 16 nov» si es un solo día. */
export function spanText(from: string, to: string): string {
  const a = SHORT.format(new Date(`${from}T00:00:00Z`))
  return from === to ? `el ${a}` : `del ${a} al ${SHORT.format(new Date(`${to}T00:00:00Z`))}`
}

/** «3 días» o «1 día», contando los dos extremos. */
export function lengthText(from: string, to: string): string {
  const days = daysBetween(from, to) + 1
  return days === 1 ? '1 día' : `${days} días`
}
