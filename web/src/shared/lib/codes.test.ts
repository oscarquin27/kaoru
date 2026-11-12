import { describe, expect, it } from 'vitest'
import { routeOf } from './codes.ts'

describe('routeOf', () => {
  it.each([
    ['REQ-2026-0042', '/requerimientos/REQ-2026-0042'],
    ['PKG-2026-0012', '/paquetes/PKG-2026-0012'],
    ['BUILD-2026-0214', '/paquetes/builds/BUILD-2026-0214'],
    ['ENV-2026-0001', '/ambientes/ENV-2026-0001'],
    ['DEP-2026-0003', '/ambientes/despliegues/DEP-2026-0003'],
    ['COMP-2026-0007', '/pruebas/componentes/COMP-2026-0007'],
  ])('%s lleva a su ficha', (code, route) => {
    expect(routeOf(code)).toBe(route)
  })

  it('un ID sin pantalla propia no tiene ruta', () => {
    expect(routeOf('TC-2026-0001')).toBeNull()
  })
})
