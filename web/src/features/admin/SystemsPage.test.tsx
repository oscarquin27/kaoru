// La pantalla de sistemas con las respuestas reales grabadas: qué muestra y qué envía.
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { renderApp } from '../../test-utils.tsx'

const PATH = '/administracion/sistemas'

describe('administración de sistemas', () => {
  it('lista los sistemas con su clave y su ruta por omisión', async () => {
    renderApp(PATH)

    const table = await screen.findByRole('table', { name: 'Sistemas' })
    const sypago = within(table).getByRole('cell', { name: 'SYPAGO' }).closest('tr') as HTMLElement
    expect(within(sypago).getByRole('cell', { name: 'SyPago' })).toBeInTheDocument()
    expect(within(sypago).getByRole('cell', { name: 'Ligera' })).toBeInTheDocument()
    expect(within(sypago).getByRole('button', { name: 'Editar SyPago' })).toBeInTheDocument()
  })

  it('sin admin.configure es de solo lectura', async () => {
    renderApp(PATH, [])

    expect(await screen.findByRole('table', { name: 'Sistemas' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Nuevo sistema' })).toBeNull()
    expect(screen.queryByRole('button', { name: /Editar/ })).toBeNull()
  })

  it('agregar un sistema envía clave, nombre y ruta', async () => {
    const user = userEvent.setup()
    const calls = renderApp(PATH)

    await user.click(await screen.findByRole('button', { name: 'Nuevo sistema' }))
    const dialog = screen.getByRole('dialog', { name: 'Nuevo sistema' })
    await user.click(within(dialog).getByRole('button', { name: 'Agregar sistema' }))
    expect(within(dialog).getByRole('textbox', { name: /Clave/ })).toHaveAccessibleDescription(expect.stringContaining('Este campo es obligatorio.'))

    await user.type(within(dialog).getByRole('textbox', { name: /Clave/ }), 'SYSAF')
    await user.type(within(dialog).getByRole('textbox', { name: /Nombre/ }), 'SysAF')
    await user.selectOptions(within(dialog).getByRole('combobox', { name: /Ruta por omisión/ }), 'high')
    await user.click(within(dialog).getByRole('button', { name: 'Agregar sistema' }))
    expect(calls.find((call) => call.method === 'POST')).toMatchObject({
      url: '/api/systems',
      body: { key: 'SYSAF', name: 'SysAF', defaultRoute: 'high' },
    })
  })

  it('editar no deja cambiar la clave', async () => {
    const user = userEvent.setup()
    const calls = renderApp(PATH)

    await user.click(await screen.findByRole('button', { name: 'Editar SyPago' }))
    const dialog = screen.getByRole('dialog', { name: 'Editar SyPago' })
    expect(within(dialog).queryByRole('textbox', { name: /Clave/ })).toBeNull()
    await user.selectOptions(within(dialog).getByRole('combobox', { name: /Ruta por omisión/ }), 'high')
    await user.click(within(dialog).getByRole('button', { name: 'Guardar' }))
    expect(calls.find((call) => call.method === 'PATCH')).toMatchObject({
      url: '/api/systems/SYPAGO',
      body: { name: 'SyPago', defaultRoute: 'high' },
    })
  })
})
