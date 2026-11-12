// La ficha del REQ con la respuesta real grabada (REQ-2026-0001: en pruebas y bloqueado).
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { fixtureMe, problemResponse, renderApp } from '../test-utils.tsx'

const PATH = '/requerimientos/REQ-2026-0001'

describe('ficha del REQ', () => {
  it('pinta el encabezado y la línea de vida con el estado de cada etapa', async () => {
    renderApp(PATH)

    expect(await screen.findByRole('heading', { level: 1, name: 'Límite diario por ordenante' })).toBeInTheDocument()
    const stages = within(screen.getByRole('list', { name: 'Línea de vida' })).getAllByRole('listitem')
    expect(stages.map((stage) => stage.dataset.state)).toEqual(['done', 'done', 'done', 'done', 'done', 'current', 'blocked', 'pending', 'pending', 'pending', 'pending'])

    const current = stages[5]!
    expect(current).toHaveAttribute('aria-current', 'step')
    expect(current).toHaveTextContent('En pruebas')
    expect(current).toHaveTextContent('< 1 d · actual')
    expect(stages[0]).toHaveTextContent('Registrado')
    expect(stages[0]).toHaveTextContent('completada')
    expect(stages[6]).toHaveTextContent('Aprobado QA')
    expect(stages[6]).toHaveTextContent('bloqueada')
    expect(stages[10]).toHaveTextContent('pendiente')
  })

  it('«qué falta para avanzar» separa lo que bloquea de lo que solo avisa', async () => {
    renderApp(PATH)

    const section = await screen.findByRole('region', { name: 'Qué falta para avanzar a «Aprobado QA»' })
    expect(section).toHaveTextContent('2 condiciones pendientes · 1 aviso')
    const conditions = within(section).getAllByRole('listitem')
    expect(conditions.map((item) => item.dataset.blocks)).toEqual(['false', 'true', 'true'])
    expect(conditions[0]).toHaveTextContent('1 enlace de Google Sheets no tiene instantánea')
    expect(conditions[0]).toHaveTextContent('No bloquea: solo avisa.')
    expect(conditions[1]).toHaveTextContent('Falta 1 TC sin RUN aprobado: TC-2026-0003.')
    expect(conditions[1]).toHaveTextContent('Bloquea el avance.')
    // Los IDs del mensaje son enlaces a su ficha.
    expect(within(conditions[2]!).getByRole('link', { name: 'DEF-2026-0001' })).toHaveAttribute('href', '/defectos/DEF-2026-0001')
    expect(within(conditions[1]!).getByRole('link', { name: 'Ver pruebas' })).toHaveAttribute('href', expect.stringContaining('#pruebas'))
  })

  it('las pestañas muestran los datos de cada parte', async () => {
    const user = userEvent.setup()
    renderApp(PATH)

    await user.click(await screen.findByRole('tab', { name: /Pruebas/ }))
    expect(await screen.findByText('2 de 3')).toBeInTheDocument()
    expect(await screen.findByRole('cell', { name: 'Un pago reversado libera el acumulado' })).toBeInTheDocument()
    expect(screen.getByText('Fallido')).toBeInTheDocument()

    await user.click(screen.getByRole('tab', { name: /Enlaces/ }))
    expect(await screen.findByText('Especificación SyPago · pestaña Pagos · A1:F40')).toBeInTheDocument()
    const open = screen.getByRole('link', { name: 'Abrir en Google Sheets' })
    expect(open).toHaveAttribute('target', '_blank')
    expect(open).toHaveAttribute('rel', 'noopener noreferrer')
    expect(screen.getByText('Sin instantánea')).toBeInTheDocument()

    await user.click(screen.getByRole('tab', { name: /Actividad/ }))
    expect(await screen.findByText('Queda en espera de la corrección de DEF-2026-0001.')).toBeInTheDocument()
    expect(screen.getByText(/Ana Rojas cambió el estado de requerimiento/, { selector: 'li:first-child' })).toHaveTextContent('Con defectos → En pruebas')
  })
})

describe('permisos: el front solo oculta lo que el rol no puede usar', () => {
  it('solo salen las transiciones del flujo para las que el rol tiene permiso', async () => {
    renderApp(PATH)

    const group = await screen.findByRole('group', { name: 'Cambiar de estado' })
    const labels = within(group).getAllByRole('button').map((button) => button.textContent)
    // QA admin puede aprobar y marcar con defectos; diferir y rechazar son de change manager.
    expect(fixtureMe.permissions).not.toContain('req.defer')
    expect(labels).toEqual(['Pasar a «Aprobado QA»', 'Pasar a «Con defectos»'])
  })

  it('sin ningún permiso no hay botones de transición ni de edición', async () => {
    renderApp(PATH, [])

    expect(await screen.findByRole('heading', { level: 1, name: 'Límite diario por ordenante' })).toBeInTheDocument()
    expect(screen.queryByRole('group', { name: 'Cambiar de estado' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Editar' })).toBeNull()
    expect(screen.queryByRole('button', { name: /riesgo/ })).toBeNull()
  })

  it('una transición con motivo lo pide antes de enviarla', async () => {
    const user = userEvent.setup()
    const calls = renderApp(PATH, ['req.defer'])

    await user.click(await screen.findByRole('button', { name: 'Pasar a «Diferido a otro paquete»' }))
    const dialog = screen.getByRole('dialog', { name: 'Pasar a «Diferido a otro paquete»' })
    await user.click(within(dialog).getByRole('button', { name: 'Cambiar estado' }))
    expect(within(dialog).getByRole('textbox', { name: /Motivo/ })).toHaveAccessibleDescription(expect.stringContaining('Este campo es obligatorio.'))
    expect(calls.some((call) => call.method === 'POST')).toBe(false)

    await user.type(within(dialog).getByRole('textbox', { name: /Motivo/ }), 'Sale en el paquete siguiente')
    await user.click(within(dialog).getByRole('button', { name: 'Cambiar estado' }))
    // El servidor de ejemplo no guarda: responde 409 y su detalle se muestra tal cual.
    expect(await within(dialog).findByRole('alert')).toHaveTextContent('Servidor de datos de ejemplo: no guarda cambios.')
    expect(calls.find((call) => call.method === 'POST')).toMatchObject({
      url: '/api/requirements/REQ-2026-0001/transitions',
      body: { to: 'deferred', reason: 'Sale en el paquete siguiente' },
    })
  })

  it('un 409 INVALID_ACTION muestra qué falta para avanzar', async () => {
    const user = userEvent.setup()
    renderApp(PATH)
    const detail = 'Falta 1 TC sin RUN aprobado: TC-2026-0003. Hay 1 DEF abierto de severidad alta: DEF-2026-0001.'
    const original = globalThis.fetch
    globalThis.fetch = (async (url: string, init?: RequestInit) =>
      init?.method === 'POST' ? problemResponse(409, { code: 'INVALID_ACTION', detail }) : original(url, init)) as typeof fetch

    await user.click(await screen.findByRole('button', { name: 'Pasar a «Aprobado QA»' }))
    expect(await screen.findByText(detail)).toBeInTheDocument()
    globalThis.fetch = original
  })
})

describe('listas', () => {
  it('«Nuevo REQ» solo aparece con el permiso req.create', async () => {
    renderApp('/requerimientos')
    expect(await screen.findByRole('link', { name: 'REQ-2026-0001' })).toBeInTheDocument()
    expect(fixtureMe.permissions).not.toContain('req.create')
    expect(screen.queryByRole('button', { name: 'Nuevo REQ' })).toBeNull()
  })

  it('con req.create, «Nuevo REQ» abre el formulario', async () => {
    const user = userEvent.setup()
    renderApp('/requerimientos', ['req.create'])
    await user.click(await screen.findByRole('button', { name: 'Nuevo REQ' }))
    expect(screen.getByRole('form', { name: 'Nuevo requerimiento' })).toBeInTheDocument()
  })
})
