// Iconos: un solo trazo y tamaño; el nombre dice la forma, no el uso (reglas §A.16.2).
// Los trazos son los del boceto (ICONS en tools/boceto/js/boceto.js); Icon.test.tsx comprueba
// que no se desvíen. `user`, `pencil` y `trash` son nuevos, con el mismo trazo.
import type { ReactNode } from 'react'

export const ICONS = {
  house: <><path d="M4 11l8-7 8 7"/><path d="M6 10v10h12V10"/></>,
  swap: <><path d="M4 8h14l-3-3"/><path d="M20 16H6l3 3"/></>,
  file: <><path d="M7 3h7l4 4v14H7z"/><path d="M14 3v4h4"/></>,
  flask: <><path d="M9 3h6"/><path d="M10 3v6L5 19a1.5 1.5 0 0 0 1.4 2h11.2A1.5 1.5 0 0 0 19 19l-5-10V3"/></>,
  bug: <><path d="M8 10a4 4 0 0 1 8 0v5a4 4 0 0 1-8 0z"/><path d="M4 13h4M16 13h4M5 7l3 2M19 7l-3 2M5 19l3-2M19 19l-3-2"/></>,
  box: <><path d="M4 8l8-4 8 4v8l-8 4-8-4z"/><path d="M4 8l8 4 8-4M12 12v8"/></>,
  server: <><path d="M4 5h16v6H4zM4 13h16v6H4z"/><path d="M8 8h.01M8 16h.01"/></>,
  flag: <><path d="M6 21V4"/><path d="M6 5h11l-2 4 2 4H6"/></>,
  chart: <path d="M4 20h16M7 16v-5M12 16V6M17 16V9"/>,
  calendar: <path d="M5 6h14v14H5zM5 10h14M9 3v4M15 3v4"/>,
  sliders: <><path d="M4 7h10M18 7h2M4 17h2M10 17h10"/><circle cx="16" cy="7" r="2"/><circle cx="8" cy="17" r="2"/></>,
  search: <><circle cx="11" cy="11" r="6"/><path d="M16 16l4 4"/></>,
  sun: <><circle cx="12" cy="12" r="4"/><path d="M12 3v2M12 19v2M3 12h2M19 12h2M5.6 5.6l1.4 1.4M17 17l1.4 1.4M5.6 18.4L7 17M17 7l1.4-1.4"/></>,
  moon: <path d="M20 14A8 8 0 1 1 10 4a6 6 0 0 0 10 10z"/>,
  check: <path d="M5 12l5 5 9-10"/>,
  clock: <><circle cx="12" cy="12" r="8"/><path d="M12 8v4l3 2"/></>,
  lock: <><path d="M7 11h10v9H7z"/><path d="M9 11V8a3 3 0 0 1 6 0v3"/></>,
  triangle: <><path d="M12 4l9 16H3z"/><path d="M12 10v4M12 17h.01"/></>,
  circle: <circle cx="12" cy="12" r="8"/>,
  target: <><circle cx="12" cy="12" r="8"/><circle cx="12" cy="12" r="3"/></>,
  link: <><path d="M10 14a4 4 0 0 0 5.7 0l3-3A4 4 0 0 0 13 5.3l-1 1"/><path d="M14 10a4 4 0 0 0-5.7 0l-3 3A4 4 0 0 0 11 18.7l1-1"/></>,
  grid: <path d="M4 5h16v14H4zM4 10h16M4 15h16M10 5v14"/>,
  'arrow-out': <><path d="M14 5h5v5M19 5l-8 8"/><path d="M11 7H5v12h12v-6"/></>,
  'arrow-right': <path d="M5 12h14M13 6l6 6-6 6"/>,
  x: <path d="M6 6l12 12M18 6L6 18"/>,
  download: <path d="M12 4v11M7 11l5 5 5-5M5 20h14"/>,
  rows: <path d="M4 6h16M4 12h16M4 18h16"/>,
  layers: <><path d="M12 4l9 5-9 5-9-5z"/><path d="M3 14l9 5 9-5"/></>,
  info: <><circle cx="12" cy="12" r="8"/><path d="M12 11v5M12 8h.01"/></>,
  minus: <path d="M6 12h12"/>,
  plus: <path d="M12 5v14M5 12h14"/>,
  /* Símbolo de marca: sello índigo con la K de una línea continua que une cuatro puntos
     (REQ → TC → RUN → PKG). Se pinta con el color del texto (el acento) y la K en on-accent. */
  kaoru: <><rect x="2" y="2" width="20" height="20" rx="5" fill="currentColor" stroke="none"/><g transform="translate(3.92 3.92) scale(0.66)" stroke="var(--on-accent)" strokeWidth="2.6"><path d="M7 4v16V12l10-8L7 12l10 8"/><circle cx="7" cy="4" r="2.2" fill="var(--on-accent)" stroke="none"/><circle cx="7" cy="20" r="2.2" fill="var(--on-accent)" stroke="none"/><circle cx="17" cy="4" r="2.2" fill="var(--on-accent)" stroke="none"/><circle cx="17" cy="20" r="2.2" fill="var(--on-accent)" stroke="none"/></g></>,
  user: <><circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0 1 16 0"/></>,
  pencil: <><path d="M4 20l1-5L16 4l4 4L9 19z"/><path d="M14 6l4 4"/></>,
  trash: <path d="M5 7h14M10 7V4h4v3M7 7l1 13h8l1-13"/>,
  gauge: <><path d="M4 17a8 8 0 1 1 16 0"/><path d="M12 17l4-5"/></>,
} satisfies Record<string, ReactNode>

export type IconName = keyof typeof ICONS

export function Icon({ name, size, className = 'icon' }: { name: IconName; size?: number; className?: string }) {
  const style = size ? { width: size, height: size } : undefined
  return (
    <svg className={className} viewBox="0 0 24 24" aria-hidden="true" style={style}>
      {ICONS[name]}
    </svg>
  )
}
