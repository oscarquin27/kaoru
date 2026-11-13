// Ficha del PKG: qué falta, transiciones según permisos, contenido y regresión.
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import type { Package, Regression } from '../shared/api/types.ts'
import {
  BUILD,
  BUILD_2,
  CHECKLIST,
  DIAGNOSTIC,
  PACKAGE,
  PACKAGE_DEPLOYED,
  PACKAGE_IN_TESTING,
  PACKAGE_WORKFLOW,
  PRODUCTION,
  PRODUCTION_DEPLOYMENT,
  REGRESSION,
  REGRESSION_CONFIRMED,
  REPORT,
  REPORT_V1,
} from '../test-data.ts'
import { pageOf, problemResponse, renderApp, routes } from '../test-utils.tsx'

const PATH = '/paquetes/PKG-2026-0001'
const notProposed = () => problemResponse(404, { code: 'NOT_FOUND_RESOURCE', detail: 'El paquete todavía no tiene selección de regresión.' })

function api(bundle: Package, regression?: Regression, extra: Parameters<typeof routes>[0] = {}) {
  return routes({
    'GET /api/packages/PKG-2026-0001': bundle,
    'GET /api/workflows/package': PACKAGE_WORKFLOW,
    'GET /api/packages/PKG-2026-0001/regression': regression ?? notProposed,
    'GET /api/documents': { items: [] },
    'GET /api/builds': pageOf([BUILD, BUILD_2]),
    'GET /api/reports': pageOf([]),
    ...extra,
  })
}

describe('ficha del PKG', () => {
  it('pinta el estado, la línea de estados y qué falta para avanzar', async () => {
    renderApp(PATH, [], api(PACKAGE))

    expect(await screen.findByRole('heading', { level: 1, name: 'SyPago 2.14' })).toBeInTheDocument()
    const stages = within(await screen.findByRole('list', { name: 'Línea de vida' })).getAllByRole('listitem')
    expect(stages.map((stage) => stage.dataset.state)).toEqual(['current', 'blocked', 'pending', 'pending', 'pending', 'pending'])
    expect(stages[0]).toHaveTextContent('En armado')

    const missing = screen.getByRole('region', { name: 'Qué falta para avanzar a «En pruebas»' })
    expect(missing).toHaveTextContent('1 condición pendiente')
    expect(missing).toHaveTextContent('La selección de regresión no está confirmada.')
    expect(missing).toHaveTextContent('Bloquea el avance.')
    expect(within(missing).getByRole('link', { name: 'Ver regresión' })).toHaveAttribute('href', expect.stringContaining('#regresion'))
  })

  it('separa lo que bloquea de lo que solo avisa, con los IDs como enlaces', async () => {
    renderApp(PATH, [], api(PACKAGE_IN_TESTING))

    const missing = await screen.findByRole('region', { name: 'Qué falta para avanzar a «Listo para liberar»' })
    expect(missing).toHaveTextContent('1 condición pendiente · 1 aviso')
    const conditions = within(missing).getAllByRole('listitem')
    expect(conditions.map((item) => item.dataset.blocks)).toEqual(['true', 'false'])
    expect(within(conditions[1]!).getByRole('link', { name: 'DEF-2026-0001' })).toHaveAttribute('href', '/defectos/DEF-2026-0001')
  })

  it('solo salen las transiciones del flujo para las que el rol tiene permiso', async () => {
    renderApp(PATH, ['pkg.reopen', 'pkg.mark_ready', 'pkg.deploy'], api(PACKAGE_IN_TESTING))

    const group = await screen.findByRole('group', { name: 'Cambiar de estado' })
    expect(within(group).getAllByRole('button').map((button) => button.textContent)).toEqual(['Pasar a «En armado»', 'Pasar a «Listo para liberar»'])
  })

  it('sin permisos no hay botones de transición ni de edición', async () => {
    renderApp(PATH, [], api(PACKAGE))

    expect(await screen.findByRole('heading', { level: 1, name: 'SyPago 2.14' })).toBeInTheDocument()
    await screen.findByRole('list', { name: 'Línea de vida' })
    expect(screen.queryByRole('group', { name: 'Cambiar de estado' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Editar' })).toBeNull()
  })

  it('devolver a armado pide el motivo, y un 409 muestra su detalle', async () => {
    const user = userEvent.setup()
    const detail = 'Falta 1 TC de la regresión sin RUN aprobado: TC-2026-0003.'
    const calls = renderApp(
      PATH,
      ['pkg.reopen', 'pkg.mark_ready'],
      api(PACKAGE_IN_TESTING, undefined, { 'POST /api/packages/PKG-2026-0001/transitions': () => problemResponse(409, { code: 'INVALID_ACTION', detail }) }),
    )

    await user.click(await screen.findByRole('button', { name: 'Pasar a «Listo para liberar»' }))
    expect(await screen.findByText(detail)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Pasar a «En armado»' }))
    const dialog = screen.getByRole('dialog', { name: 'Pasar a «En armado»' })
    await user.type(within(dialog).getByRole('textbox', { name: 'Motivo' }), 'Falta un BUILD')
    await user.click(within(dialog).getByRole('button', { name: 'Cambiar estado' }))
    await within(dialog).findByRole('alert')
    expect(calls.filter((call) => call.method === 'POST').map((call) => call.body)).toEqual([{ to: 'ready_to_release' }, { to: 'in_assembly', reason: 'Falta un BUILD' }])
  })
})

describe('contenido del PKG', () => {
  it('en armado y con pkg.build se agregan y quitan BUILD y REQ', async () => {
    const user = userEvent.setup()
    const calls = renderApp(PATH, ['pkg.build'], api(PACKAGE))

    const builds = await screen.findByRole('table', { name: 'BUILD del paquete' })
    expect(within(builds).getByRole('link', { name: 'BUILD-2026-0001' })).toHaveAttribute('href', '/paquetes/builds/BUILD-2026-0001')
    expect(within(screen.getByRole('table', { name: 'REQ del paquete' })).getByRole('button', { name: 'Quitar REQ-2026-0002' })).toBeInTheDocument()
    expect(await screen.findByRole('button', { name: 'Agregar REQ' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Agregar BUILD' }))
    const dialog = screen.getByRole('dialog', { name: 'Agregar BUILD a PKG-2026-0001' })
    const select = within(dialog).getByRole('combobox', { name: 'BUILD' })
    // El que ya está en el paquete no se ofrece otra vez.
    await user.selectOptions(select, await within(select).findByRole('option', { name: /BUILD-2026-0002/ }))
    expect(within(select).queryByRole('option', { name: /BUILD-2026-0001/ })).toBeNull()
    await user.click(within(dialog).getByRole('button', { name: 'Agregar BUILD' }))
    await within(dialog).findByRole('alert')
    expect(calls.find((call) => call.method === 'PUT')?.url).toBe('/api/packages/PKG-2026-0001/builds/BUILD-2026-0002')

    await user.click(within(builds).getByRole('button', { name: 'Quitar BUILD-2026-0001' }))
    expect(calls.find((call) => call.method === 'DELETE')?.url).toBe('/api/packages/PKG-2026-0001/builds/BUILD-2026-0001')
  })

  it('fuera de armado el contenido es de solo lectura, aunque el rol tenga pkg.build', async () => {
    renderApp(PATH, ['pkg.build'], api(PACKAGE_IN_TESTING))

    const builds = await screen.findByRole('table', { name: 'BUILD del paquete' })
    await screen.findByRole('list', { name: 'Línea de vida' })
    expect(within(builds).queryByRole('button')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Agregar BUILD' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Agregar REQ' })).toBeNull()
    expect(screen.getByText('El contenido de un paquete solo cambia mientras está «En armado».')).toBeInTheDocument()
  })

  it('en armado pero sin pkg.build tampoco se edita', async () => {
    renderApp(PATH, ['pkg.start_testing'], api(PACKAGE))

    expect(await screen.findByRole('button', { name: 'Pasar a «En pruebas»' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Agregar BUILD' })).toBeNull()
    expect(screen.queryByRole('button', { name: /^Quitar/ })).toBeNull()
  })
})

describe('regresión del PKG', () => {
  const TAB = `${PATH}#regresion`

  it('sin selección, se propone con regression.manage', async () => {
    const user = userEvent.setup()
    const calls = renderApp(TAB, ['regression.manage'], api(PACKAGE))

    await user.click(await screen.findByRole('button', { name: 'Proponer regresión' }))
    expect(calls.find((call) => call.method === 'POST')?.url).toBe('/api/packages/PKG-2026-0001/regression/proposal')
  })

  it('sin regression.manage no se puede proponer', async () => {
    renderApp(TAB, [], api(PACKAGE))
    expect(await screen.findByText('Este paquete todavía no tiene una selección de regresión.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Proponer regresión' })).toBeNull()
  })

  it('muestra el resumen y, por TC, origen, componentes, criterio y último RUN', async () => {
    renderApp(TAB, ['regression.manage'], api(PACKAGE, REGRESSION))

    const summary = await screen.findByRole('region', { name: 'Resumen de la regresión' })
    expect(summary).toHaveTextContent('1 agregado · 1 quitado por QA')
    expect(summary).toHaveTextContent('1 aprobado')
    expect(summary).toHaveTextContent('1 pendiente')
    const rows = within(screen.getByRole('table', { name: 'TC de la regresión' })).getAllByRole('row')
    expect(rows[1]).toHaveTextContent('Impacto')
    expect(rows[1]).toHaveTextContent('motor-pagos')
    expect(within(rows[1]!).getByRole('link', { name: '2.14.0' })).toHaveAttribute('href', '/paquetes/builds/BUILD-2026-0001')
    expect(rows[2]).toHaveTextContent('Lo cubre TC-2026-0001')
    expect(rows[2]).toHaveTextContent('Ana Rojas')
    expect(rows[3]).toHaveTextContent('Manual')
    expect(rows.slice(1).map((row) => row.dataset.included)).toEqual(['true', 'false', 'true'])
  })

  it('quitar un TC propuesto pide el criterio, lo envía y pinta el error del servidor bajo el campo', async () => {
    const user = userEvent.setup()
    const rejected = () => problemResponse(422, { code: 'VALIDATION', errors: [{ field: 'reason', code: 'too_short', message: 'El criterio debe explicar la decisión.' }] })
    const calls = renderApp(TAB, ['regression.manage'], api(PACKAGE, REGRESSION, { 'PUT /api/packages/PKG-2026-0001/regression/items/TC-2026-0001': rejected }))

    await user.click(await screen.findByRole('button', { name: 'Quitar TC-2026-0001' }))
    const dialog = screen.getByRole('dialog', { name: 'Quitar TC-2026-0001' })
    const reason = within(dialog).getByRole('textbox', { name: 'Criterio' })
    await user.click(within(dialog).getByRole('button', { name: 'Quitar de la regresión' }))
    expect(reason).toHaveAccessibleDescription(expect.stringContaining('Este campo es obligatorio.'))
    expect(calls.some((call) => call.method === 'PUT')).toBe(false)

    await user.type(reason, 'No')
    await user.click(within(dialog).getByRole('button', { name: 'Quitar de la regresión' }))
    await screen.findByText('El criterio debe explicar la decisión.')
    expect(reason).toHaveAccessibleDescription(expect.stringContaining('El criterio debe explicar la decisión.'))
    expect(calls.find((call) => call.method === 'PUT')).toEqual({
      method: 'PUT',
      url: '/api/packages/PKG-2026-0001/regression/items/TC-2026-0001',
      body: { included: false, reason: 'No' },
    })
  })

  it('volver a lo propuesto no pide criterio; agregar un TC por código, sí', async () => {
    const user = userEvent.setup()
    const calls = renderApp(TAB, ['regression.manage'], api(PACKAGE, REGRESSION))

    // TC-2026-0002 fue propuesto y QA lo quitó: incluirlo es volver a lo propuesto.
    await user.click(await screen.findByRole('button', { name: 'Incluir TC-2026-0002' }))
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(calls.find((call) => call.method === 'PUT')).toMatchObject({ url: '/api/packages/PKG-2026-0001/regression/items/TC-2026-0002', body: { included: true } })

    await user.click(screen.getByRole('button', { name: 'Agregar TC' }))
    const dialog = screen.getByRole('dialog', { name: 'Agregar TC a la regresión' })
    await user.type(within(dialog).getByRole('textbox', { name: 'Código del TC' }), 'TC-2026-0007')
    await user.type(within(dialog).getByRole('textbox', { name: 'Criterio' }), 'Toca el cierre del día')
    await user.click(within(dialog).getByRole('button', { name: 'Agregar TC' }))
    await within(dialog).findByRole('alert')
    expect(calls.filter((call) => call.method === 'PUT').at(-1)).toMatchObject({
      url: '/api/packages/PKG-2026-0001/regression/items/TC-2026-0007',
      body: { included: true, reason: 'Toca el cierre del día' },
    })
  })

  it('confirmar avisa que ya no cambia y envía la confirmación', async () => {
    const user = userEvent.setup()
    const calls = renderApp(TAB, ['regression.manage'], api(PACKAGE, REGRESSION))

    await user.click(await screen.findByRole('button', { name: 'Confirmar' }))
    const dialog = screen.getByRole('dialog', { name: 'Confirmar la regresión' })
    expect(dialog).toHaveTextContent('la selección no cambia')
    await user.click(within(dialog).getByRole('button', { name: 'Confirmar regresión' }))
    await within(dialog).findByRole('alert')
    expect(calls.find((call) => call.method === 'POST')?.url).toBe('/api/packages/PKG-2026-0001/regression/confirmation')
  })

  it('confirmada es de solo lectura y dice quién la confirmó', async () => {
    renderApp(TAB, ['regression.manage'], api(PACKAGE, REGRESSION_CONFIRMED))

    const table = await screen.findByRole('table', { name: 'TC de la regresión' })
    expect(screen.getByText(/Confirmada por Ana Rojas/)).toBeInTheDocument()
    expect(within(table).queryByRole('button', { name: /Quitar|Incluir/ })).toBeNull()
    for (const name of ['Agregar TC', 'Volver a proponer', 'Confirmar', 'Proponer regresión']) {
      expect(screen.queryByRole('button', { name })).toBeNull()
    }
  })

  it('sin regression.manage la selección solo se lee', async () => {
    renderApp(TAB, [], api(PACKAGE, REGRESSION))

    const table = await screen.findByRole('table', { name: 'TC de la regresión' })
    expect(within(table).queryByRole('button', { name: /Quitar|Incluir/ })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Confirmar' })).toBeNull()
  })
})

describe('producción del PKG', () => {
  const TAB = `${PATH}#produccion`
  const production = {
    'GET /api/checklists': pageOf([CHECKLIST]),
    'GET /api/diagnostics': pageOf([DIAGNOSTIC]),
    'GET /api/smoke-tests': pageOf([]),
    'GET /api/environments': pageOf([PRODUCTION]),
    'GET /api/deployments': pageOf([PRODUCTION_DEPLOYMENT]),
  }

  it('la regla que falta lleva a la pestaña de producción, que explica qué exige la ruta', async () => {
    const user = userEvent.setup()
    renderApp(PATH, [], api(PACKAGE_DEPLOYED, undefined, production))

    const missing = await screen.findByRole('region', { name: 'Qué falta para avanzar a «Verificado»' })
    expect(screen.getByText('Alta')).toBeInTheDocument()
    await user.click(within(missing).getByRole('link', { name: 'Ver diagnósticos' }))
    expect(await screen.findByText(/Ruta alta: el PKG pasa a desplegado/)).toBeInTheDocument()
    expect(within(screen.getByRole('table', { name: 'Checklists de producción' })).getByText('1 de 2 firmados')).toBeInTheDocument()
    expect(within(screen.getByRole('table', { name: 'Diagnósticos' })).getByText('Sin revisar')).toBeInTheDocument()
    expect(screen.getByText('El PKG no tiene pruebas de humo.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Abrir checklist' })).toBeNull()
  })

  it('cada acción sale con su permiso: abrir el CHK en producción y entregar un DIAG de un despliegue', async () => {
    const user = userEvent.setup()
    const calls = renderApp(TAB, ['chk.start', 'diag.submit'], api(PACKAGE_DEPLOYED, undefined, { ...production, 'POST /api/checklists': CHECKLIST, 'POST /api/diagnostics': DIAGNOSTIC }))

    expect(await screen.findByRole('button', { name: 'Entregar DIAG' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Registrar SMOKE' })).toBeNull()

    await user.click(screen.getByRole('button', { name: 'Abrir checklist' }))
    let dialog = screen.getByRole('dialog', { name: 'Abrir checklist de PKG-2026-0001' })
    await user.selectOptions(within(dialog).getByRole('combobox', { name: 'Ambiente de producción' }), await within(dialog).findByRole('option', { name: /PROD/ }))
    await user.click(within(dialog).getByRole('button', { name: 'Abrir CHK' }))
    expect(calls.find((call) => call.url === '/api/checklists' && call.method === 'POST')?.body).toEqual({ package: 'PKG-2026-0001', environment: 'ENV-2026-0002' })

    await user.click(await screen.findByRole('button', { name: 'Entregar DIAG' }))
    dialog = screen.getByRole('dialog', { name: 'Entregar diagnóstico de PKG-2026-0001' })
    await user.selectOptions(within(dialog).getByRole('combobox', { name: 'Despliegue' }), await within(dialog).findByRole('option', { name: /DEP-2026-0002/ }))
    await user.type(within(dialog).getByRole('textbox', { name: 'Notas' }), 'Sin errores')
    await user.click(within(dialog).getByRole('button', { name: 'Entregar DIAG' }))
    expect(calls.find((call) => call.url === '/api/diagnostics' && call.method === 'POST')?.body).toEqual({ deployment: 'DEP-2026-0002', notes: 'Sin errores' })
  })
})

describe('informes del PKG', () => {
  const TAB = `${PATH}#informes`

  it('lista las versiones y, con rpt.issue, emite otra y abre su ficha', async () => {
    const user = userEvent.setup()
    const calls = renderApp(
      TAB,
      ['rpt.issue'],
      api(PACKAGE_DEPLOYED, undefined, { 'GET /api/reports': pageOf([REPORT, REPORT_V1]), 'POST /api/reports': REPORT, 'GET /api/reports/RPT-2026-0002': REPORT }),
    )

    const rows = within(await screen.findByRole('table', { name: 'Informes de conformidad' })).getAllByRole('row')
    expect(rows[1]).toHaveTextContent('Vigente')
    expect(rows[2]).toHaveTextContent('Reemplazado')
    expect(within(rows[2]!).getByRole('link', { name: 'RPT-2026-0002' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Emitir nueva versión' }))
    expect(calls.find((call) => call.method === 'POST')?.body).toEqual({ package: 'PKG-2026-0001' })
    expect(await screen.findByRole('heading', { level: 1, name: 'Informe de conformidad de PKG-2026-0001' })).toBeInTheDocument()
  })

  it('sin rpt.issue no se emite', async () => {
    renderApp(TAB, [], api(PACKAGE_DEPLOYED))
    expect(await screen.findByText('El PKG no tiene informes de conformidad.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Emitir informe' })).toBeNull()
  })
})
