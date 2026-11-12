// BUILD: alta con su manifiesto y ficha de solo lectura.
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import fixtures from '../../../mocks/fixtures.json' with { type: 'json' }
import { BUILD, CHECKSUM } from '../../test-data.ts'
import { pageOf, renderApp, routes } from '../../test-utils.tsx'

const api = routes({
  'GET /api/builds': pageOf([BUILD]),
  'GET /api/builds/BUILD-2026-0001': BUILD,
  // Los REQ del sistema elegido, para el manifiesto.
  'GET /api/requirements': (url: string) => pageOf(fixtures['GET /api/requirements'].items.filter((item) => url.includes(`system=${item.system.key}`))),
})

describe('alta de BUILD', () => {
  it('valida el checksum antes de enviar y manda el manifiesto', async () => {
    const user = userEvent.setup()
    const calls = renderApp('/paquetes/builds', ['build.register'], api)

    await user.click(await screen.findByRole('button', { name: 'Registrar BUILD' }))
    const dialog = screen.getByRole('dialog', { name: 'Registrar BUILD' })
    expect(dialog).toHaveTextContent('Un BUILD no se edita ni se borra después de registrarlo')
    await user.selectOptions(within(dialog).getByRole('combobox', { name: 'Sistema' }), await within(dialog).findByRole('option', { name: 'SyPago' }))
    await user.type(within(dialog).getByRole('textbox', { name: 'Versión' }), '2.15.0')
    const checksum = within(dialog).getByRole('textbox', { name: 'SHA-256 del artefacto' })
    await user.type(checksum, 'abc123')
    await user.click(within(dialog).getByRole('button', { name: 'Registrar BUILD' }))
    expect(checksum).toHaveAccessibleDescription(expect.stringContaining('64 caracteres hexadecimales'))
    expect(calls.some((call) => call.method === 'POST')).toBe(false)

    // Solo se ofrecen los REQ del sistema elegido.
    await user.click(await within(dialog).findByRole('checkbox', { name: /REQ-2026-0001/ }))
    expect(within(dialog).queryByRole('checkbox', { name: /REQ-2026-0005/ })).toBeNull()
    await user.clear(checksum)
    await user.click(checksum)
    await user.paste(CHECKSUM.toUpperCase())
    await user.type(within(dialog).getByRole('textbox', { name: 'Migraciones' }), '00031_limite.sql')
    await user.click(within(dialog).getByRole('button', { name: 'Registrar BUILD' }))
    await within(dialog).findByRole('alert')
    expect(calls.find((call) => call.method === 'POST')).toEqual({
      method: 'POST',
      url: '/api/builds',
      body: { system: 'SYPAGO', version: '2.15.0', checksum: CHECKSUM, requirements: ['REQ-2026-0001'], migrations: '00031_limite.sql', dependencies: '', notes: '' },
    })
  })

  it('sin build.register no hay alta', async () => {
    renderApp('/paquetes/builds', [], api)
    expect(await screen.findByRole('link', { name: 'BUILD-2026-0001' })).toHaveAttribute('href', '/paquetes/builds/BUILD-2026-0001')
    expect(screen.queryByRole('button', { name: 'Registrar BUILD' })).toBeNull()
  })
})

describe('ficha del BUILD', () => {
  it('muestra el manifiesto, el checksum completo y que no se edita', async () => {
    renderApp('/paquetes/builds/BUILD-2026-0001', ['build.register'], api)

    expect(await screen.findByRole('heading', { level: 1, name: 'Versión 2.14.0' })).toBeInTheDocument()
    expect(screen.getByText(CHECKSUM)).toHaveClass('mono')
    expect(within(screen.getByRole('list', { name: 'REQ incluidos' })).getByRole('link', { name: 'REQ-2026-0001' })).toHaveAttribute('href', '/requerimientos/REQ-2026-0001')
    expect(screen.getByText('kafka-client 3.7.0')).toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent('Un BUILD no se edita ni se borra.')
    expect(screen.queryByRole('button', { name: /Editar/ })).toBeNull()
  })
})
