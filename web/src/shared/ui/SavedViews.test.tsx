// Vistas guardadas en la lista de requerimientos: aplicar, guardar y quitar.
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import { App } from '../../app/App.tsx'
import { createQueryClient } from '../api/query.ts'
import { stubApi } from '../../test-utils.tsx'

const VIEW = { id: '0195a0c0-0000-7000-8000-000000000001', screen: 'requirements', name: 'En pruebas', query: 'state=in_testing', createdAt: '2026-11-12T09:00:00Z' }

function renderList(path: string) {
  const calls = stubApi((method, url) =>
    method === 'GET' && url.startsWith('/api/saved-views')
      ? new Response(JSON.stringify({ items: [VIEW] }), { headers: { 'Content-Type': 'application/json' } })
      : method === 'GET' && url.startsWith('/api/requirements?')
        ? new Response(JSON.stringify({ items: [], page: 1, size: 50, total: 0 }), { headers: { 'Content-Type': 'application/json' } })
        : undefined,
  )
  render(
    <MemoryRouter initialEntries={[path]}>
      <App queryClient={createQueryClient()} />
    </MemoryRouter>,
  )
  return calls
}

describe('vistas guardadas', () => {
  it('elegir una vista aplica sus filtros a la lista', async () => {
    const user = userEvent.setup()
    const calls = renderList('/requerimientos')

    const filters = await screen.findByRole('search', { name: 'Filtros' })
    const select = within(filters).getByRole('combobox', { name: 'Vista guardada' })
    await user.selectOptions(select, await within(select).findByRole('option', { name: 'En pruebas' }))
    expect(within(filters).getByRole('combobox', { name: 'Estado' })).toHaveValue('in_testing')
    expect(calls.some((call) => call.url === '/api/requirements?state=in_testing')).toBe(true)
    // Con una vista aplicada se ofrece quitarla, no guardarla otra vez.
    await user.click(within(filters).getByRole('button', { name: 'Quitar vista' }))
    expect(calls.find((call) => call.method === 'DELETE')?.url).toBe(`/api/saved-views/${VIEW.id}`)
  })

  it('sin filtros no hay nada que guardar; con filtros se guarda con un nombre', async () => {
    const user = userEvent.setup()
    const calls = renderList('/requerimientos')
    const filters = await screen.findByRole('search', { name: 'Filtros' })
    expect(within(filters).getByRole('button', { name: 'Guardar vista' })).toBeDisabled()

    const system = within(filters).getByRole('combobox', { name: 'Sistema' })
    await user.selectOptions(system, await within(system).findByRole('option', { name: 'SyPago' }))
    await user.click(within(filters).getByRole('button', { name: 'Guardar vista' }))
    const dialog = screen.getByRole('dialog', { name: 'Guardar vista' })
    await user.type(within(dialog).getByRole('textbox', { name: /Nombre de la vista/ }), 'Solo SyPago')
    await user.click(within(dialog).getByRole('button', { name: 'Guardar' }))
    await within(dialog).findByRole('alert')
    expect(calls.find((call) => call.method === 'POST')).toMatchObject({
      url: '/api/saved-views',
      body: { screen: 'requirements', name: 'Solo SyPago', query: 'system=SYPAGO' },
    })
  })
})
