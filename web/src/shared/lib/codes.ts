// IDs legibles: son las rutas (`/requerimientos/REQ-2026-0042`).

export const CODE_PATTERN = /[A-Z]+-\d{4}-\d{4,}/g

const ROUTES: Record<string, string> = {
  REQ: '/requerimientos',
  CHG: '/cambios',
  DEF: '/defectos',
  DOC: '/documentos',
  PKG: '/paquetes',
  BUILD: '/paquetes/builds',
  ENV: '/ambientes',
  DEP: '/ambientes/despliegues',
  COMP: '/pruebas/componentes',
}

export function prefixOf(code: string): string {
  return code.split('-')[0] ?? ''
}

/** Ruta de la ficha completa de un ID; `null` si ese tipo no tiene pantalla propia. */
export function routeOf(code: string): string | null {
  const base = ROUTES[prefixOf(code)]
  return base ? `${base}/${code}` : null
}
