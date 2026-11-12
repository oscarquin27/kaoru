// Preferencias del usuario, como en el boceto: tema, transparencia y densidad.
// Se guardan en localStorage y se aplican a <html> antes de pintar.
import { useState } from 'react'

const PREFERENCES = {
  theme: { initial: 'dark', other: 'light' },
  glass: { initial: 'on', other: 'off' },
  density: { initial: 'comfortable', other: 'compact' },
} as const

export type Preference = keyof typeof PREFERENCES

function read(key: Preference): string {
  try {
    return localStorage.getItem(`kaoru.${key}`) ?? PREFERENCES[key].initial
  } catch {
    return PREFERENCES[key].initial
  }
}

function write(key: Preference, value: string) {
  document.documentElement.dataset[key] = value
  try {
    localStorage.setItem(`kaoru.${key}`, value)
  } catch {
    // Sin almacenamiento: vale solo en esta visita.
  }
}

export function applyPreferences() {
  for (const key of Object.keys(PREFERENCES) as Preference[]) {
    document.documentElement.dataset[key] = read(key)
  }
}

/** `on` es verdadero cuando la preferencia está en su valor alterno (claro, sin vidrio, compacto). */
export function usePreference(key: Preference): { on: boolean; toggle: () => void } {
  const { initial, other } = PREFERENCES[key]
  const [value, setValue] = useState(() => read(key))
  const on = value === other
  return {
    on,
    toggle() {
      const next = on ? initial : other
      write(key, next)
      setValue(next)
    },
  }
}
