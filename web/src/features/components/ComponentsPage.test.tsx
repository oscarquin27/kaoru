// Catálogo de componentes y declaración de impacto en la ficha del REQ.
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { COMPONENT, COMPONENT_2 } from '../../test-data.ts'
import { pageOf, renderApp, routes } from '../../test-utils.tsx'

const catalog = routes({ 'GET /api/components': pageOf([COMPONENT, { ...COMPONENT_2, retired: true }]) })

describe('catálogo de componentes', () => {
  it('el alta envía sistema, clase, nombre y descripción', async () => {
    const user = userEvent.setup()
    const calls = renderApp('/pruebas/componentes', ['comp.manage'], catalog)

    await user.click(await screen.findByRole('button', { name: 'Nuevo COMP' }))
    const dialog = screen.getByRole('dialog', { name: 'Nuevo componente' })
    await user.type(within(dialog).getByRole('textbox', { name: 'Nombre' }), 'conciliador')
    await user.selectOptions(within(dialog).getByRole('combobox', { name: 'Sistema' }), await within(dialog).findByRole('option', { name: 'SyPago' }))
    await user.type(within(dialog).getByRole('combobox', { name: 'Clase' }), 'librería')
    await user.type(within(dialog).getByRole('textbox', { name: 'Descripción' }), 'Cuadra los pagos del día')
    await user.click(within(dialog).getByRole('button', { name: 'Agregar COMP' }))
    await within(dialog).findByRole('alert')
    expect(calls.find((call) => call.method === 'POST')).toEqual({
      method: 'POST',
      url: '/api/components',
      body: { system: 'SYPAGO', kind: 'library', name: 'conciliador', description: 'Cuadra los pagos del día' },
    })
  })

  it('una clase que no está en la lista se envía tal como se escribió', async () => {
    const user = userEvent.setup()
    const calls = renderApp('/pruebas/componentes', ['comp.manage'], catalog)

    await user.click(await screen.findByRole('button', { name: 'Nuevo COMP' }))
    const dialog = screen.getByRole('dialog', { name: 'Nuevo componente' })
    await user.type(within(dialog).getByRole('textbox', { name: 'Nombre' }), 'cobros-win')
    await user.selectOptions(within(dialog).getByRole('combobox', { name: 'Sistema' }), await within(dialog).findByRole('option', { name: 'SyPago' }))
    await user.type(within(dialog).getByRole('combobox', { name: 'Clase' }), ' Servicio de Windows ')
    await user.click(within(dialog).getByRole('button', { name: 'Agregar COMP' }))
    await within(dialog).findByRole('alert')
    expect(calls.find((call) => call.method === 'POST')?.body).toMatchObject({ kind: 'Servicio de Windows' })
  })

  it('retirar y reactivar cambian solo `retired`', async () => {
    const user = userEvent.setup()
    const calls = renderApp('/pruebas/componentes?includeRetired=true', ['comp.manage'], catalog)

    const table = await screen.findByRole('table', { name: 'Componentes' })
    expect(calls.some((call) => call.url === '/api/components?includeRetired=true')).toBe(true)
    expect(within(table).getByRole('link', { name: 'COMP-2026-0001' })).toHaveAttribute('href', '/pruebas/componentes/COMP-2026-0001')
    await user.click(within(table).getByRole('button', { name: 'Retirar motor-pagos' }))
    expect(calls.find((call) => call.method === 'PATCH')).toMatchObject({ url: '/api/components/COMP-2026-0001', body: { retired: true } })
    expect(within(table).getByRole('button', { name: 'Reactivar pagos.eventos' })).toBeInTheDocument()
  })

  it('sin comp.manage es de solo lectura', async () => {
    renderApp('/pruebas/componentes', [], catalog)

    const table = await screen.findByRole('table', { name: 'Componentes' })
    expect(within(table).getByRole('cell', { name: 'Microservicio' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Nuevo COMP' })).toBeNull()
    expect(within(table).queryByRole('button')).toBeNull()
  })
})

describe('impacto en la ficha del REQ', () => {
  const impact = routes({
    'GET /api/requirements/REQ-2026-0001/components': { items: [{ code: COMPONENT.code, name: COMPONENT.name, kind: COMPONENT.kind }] },
    'GET /api/components': pageOf([COMPONENT, COMPONENT_2]),
  })

  it('muestra los componentes como chips y guarda la lista completa con PUT', async () => {
    const user = userEvent.setup()
    const calls = renderApp('/requerimientos/REQ-2026-0001', ['req.edit'], impact)

    const card = await screen.findByRole('region', { name: 'Componentes que toca' })
    expect(await within(card).findByText('motor-pagos')).toBeInTheDocument()
    await user.click(within(card).getByRole('button', { name: 'Declarar componentes' }))
    const dialog = screen.getByRole('dialog', { name: 'Componentes que toca' })
    expect(await within(dialog).findByRole('checkbox', { name: 'motor-pagos · Microservicio' })).toBeChecked()
    await user.click(within(dialog).getByRole('checkbox', { name: 'pagos.eventos · Tópico de Kafka' }))
    await user.click(within(dialog).getByRole('button', { name: 'Guardar' }))
    await within(dialog).findByRole('alert')
    // Solo se ofrecen los componentes del sistema del REQ.
    expect(calls.some((call) => call.url === '/api/components?system=SYPAGO&size=200')).toBe(true)
    expect(calls.find((call) => call.method === 'PUT')).toEqual({
      method: 'PUT',
      url: '/api/requirements/REQ-2026-0001/components',
      body: { components: ['COMP-2026-0001', 'COMP-2026-0002'] },
    })
  })

  it('sin req.edit solo se ven', async () => {
    renderApp('/requerimientos/REQ-2026-0001', [], impact)
    const card = await screen.findByRole('region', { name: 'Componentes que toca' })
    expect(await within(card).findByText('motor-pagos')).toBeInTheDocument()
    expect(within(card).queryByRole('button')).toBeNull()
  })

  it('con testdesign.manage, cada TC declara los componentes que ejercita', async () => {
    const user = userEvent.setup()
    const calls = renderApp(
      '/requerimientos/REQ-2026-0001#pruebas',
      ['testdesign.manage'],
      routes({ 'GET /api/test-cases/TC-2026-0001/components': { items: [] }, 'GET /api/components': pageOf([COMPONENT]) }),
    )

    await user.click(await screen.findByRole('button', { name: 'Componentes de TC-2026-0001' }))
    const dialog = screen.getByRole('dialog', { name: 'Componentes que ejercita TC-2026-0001' })
    await user.click(await within(dialog).findByRole('checkbox', { name: 'motor-pagos · Microservicio' }))
    await user.click(within(dialog).getByRole('button', { name: 'Guardar' }))
    await within(dialog).findByRole('alert')
    expect(calls.find((call) => call.method === 'PUT')).toMatchObject({ url: '/api/test-cases/TC-2026-0001/components', body: { components: ['COMP-2026-0001'] } })
  })
})
