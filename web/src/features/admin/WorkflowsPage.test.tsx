// La pantalla de flujos con las respuestas reales grabadas: qué muestra y qué envía.
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import fixtures from '../../../mocks/fixtures.json' with { type: 'json' }
import { renderApp } from '../../test-utils.tsx'

const PATH = '/administracion/flujos'

describe('administración de flujos', () => {
  it('lista los estados y las transiciones de cada flujo', async () => {
    renderApp(PATH)

    const states = await screen.findByRole('table', { name: 'Estados de Cambios' })
    expect(within(states).getByRole('cell', { name: 'Registrado' })).toBeInTheDocument()
    expect(within(states).getByRole('cell', { name: 'registered' })).toBeInTheDocument()
    // El inicial y los que tienen transiciones no se pueden quitar.
    expect(await within(states).findByRole('button', { name: 'Quitar Registrado' })).toBeDisabled()
    const transitions = screen.getByRole('table', { name: 'Transiciones de Cambios' })
    expect(within(transitions).getByRole('cell', { name: 'Registrado → Rechazado' })).toBeInTheDocument()
    expect(within(transitions).getByRole('checkbox', { name: 'Exige motivo: Registrado → Rechazado' })).toBeChecked()
  })

  it('sin admin.configure es de solo lectura', async () => {
    renderApp(PATH, [])

    expect(await screen.findByRole('table', { name: 'Estados de Cambios' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Agregar estado' })).toBeNull()
    expect(screen.queryByRole('button', { name: /Quitar/ })).toBeNull()
    expect(within(screen.getByRole('table', { name: 'Transiciones de Cambios' })).getByRole('checkbox', { name: 'Exige motivo: Registrado → Rechazado' })).toBeDisabled()
  })

  it('agregar un estado envía clave, nombre y posición', async () => {
    const user = userEvent.setup()
    const calls = renderApp(PATH)

    const section = await screen.findByRole('region', { name: 'Cambios' })
    await user.click(await within(section).findByRole('button', { name: 'Agregar estado' }))
    const dialog = screen.getByRole('dialog', { name: 'Agregar estado · Cambios' })
    await user.click(within(dialog).getByRole('button', { name: 'Agregar estado' }))
    expect(within(dialog).getByRole('textbox', { name: /Clave/ })).toHaveAccessibleDescription(expect.stringContaining('Este campo es obligatorio.'))

    await user.type(within(dialog).getByRole('textbox', { name: /Nombre/ }), 'En espera')
    await user.type(within(dialog).getByRole('textbox', { name: /Clave/ }), 'on_hold')
    await user.click(within(dialog).getByRole('button', { name: 'Agregar estado' }))
    expect(await within(dialog).findByRole('alert')).toHaveTextContent('no guarda cambios')
    expect(calls.find((call) => call.method === 'POST')).toMatchObject({
      url: '/api/admin/workflows/change/states',
      body: { key: 'on_hold', name: 'En espera', isFinal: false },
    })
  })

  it('cambiar «exige motivo» guarda la transición con su permiso actual', async () => {
    const user = userEvent.setup()
    const calls = renderApp(PATH)

    const transitions = await screen.findByRole('table', { name: 'Transiciones de Cambios' })
    await user.click(await within(transitions).findByRole('button', { name: 'Quitar Registrado → Aprobado' }).then(() => within(transitions).getByRole('checkbox', { name: 'Exige motivo: Registrado → Aprobado' })))
    expect(calls.find((call) => call.method === 'PUT')).toMatchObject({
      url: '/api/admin/workflows/change/transitions/registered/approved',
      body: { permission: 'chg.approve', requiresReason: true },
    })
  })
})

describe('responsables', () => {
  it('el formulario del REQ deja elegir a cualquier usuario que ya entró', async () => {
    const user = userEvent.setup()
    const calls = renderApp('/requerimientos', ['req.create'])

    // El id sale de la grabación: cambia cada vez que se regraba.
    const ana = (fixtures['GET /api/users'] as { items: { id: string; displayName: string }[] }).items.find((item) => item.displayName === 'Ana Rojas')
    await user.click(await screen.findByRole('button', { name: 'Nuevo REQ' }))
    const form = screen.getByRole('form', { name: 'Nuevo requerimiento' })
    const owner = within(form).getByRole('combobox', { name: 'Responsable' })
    expect(await within(owner).findByRole('option', { name: 'Ana Rojas' })).toBeInTheDocument()
    expect(within(owner).getByRole('option', { name: 'Yo (Marta León)' })).toBeInTheDocument()

    await user.type(within(form).getByRole('textbox', { name: /Título/ }), 'Tope por comercio')
    await user.selectOptions(within(form).getByRole('combobox', { name: /Sistema/ }), 'SyPago')
    await user.selectOptions(owner, 'Ana Rojas')
    await user.click(within(form).getByRole('button', { name: 'Registrar REQ' }))
    await within(form).findByRole('alert')
    expect(calls.find((call) => call.method === 'POST')?.body).toMatchObject({ title: 'Tope por comercio', owner: ana?.id })
  })
})
