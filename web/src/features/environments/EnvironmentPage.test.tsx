// Ambientes y despliegues: qué hay instalado, quién puede desplegar y qué se envía.
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { BUILD, DEPLOYMENT, ENVIRONMENT, PACKAGE, PRODUCTION } from '../../test-data.ts'
import { pageOf, renderApp, routes } from '../../test-utils.tsx'

const api = routes({
  'GET /api/environments': pageOf([ENVIRONMENT, PRODUCTION]),
  'GET /api/environments/ENV-2026-0001': ENVIRONMENT,
  'GET /api/environments/ENV-2026-0002': PRODUCTION,
  'GET /api/deployments': pageOf([DEPLOYMENT]),
  'GET /api/builds': pageOf([BUILD]),
  'GET /api/packages': pageOf([PACKAGE]),
})

describe('ambientes', () => {
  it('la lista dice el tipo, el SO y lo instalado por el último despliegue exitoso', async () => {
    renderApp('/ambientes', [], api)

    const rows = within(await screen.findByRole('table', { name: 'Ambientes' })).getAllByRole('row')
    expect(rows[1]).toHaveTextContent('QA-1')
    expect(rows[1]).toHaveTextContent('Línea base')
    expect(rows[1]).toHaveTextContent('Linux')
    expect(within(rows[1]!).getByRole('link', { name: 'BUILD-2026-0001' })).toHaveAttribute('href', '/paquetes/builds/BUILD-2026-0001')
    expect(rows[2]).toHaveTextContent('Producción')
    expect(rows[2]).toHaveTextContent('Sin despliegues exitosos')
    expect(screen.queryByRole('button', { name: 'Nuevo ENV' })).toBeNull()
  })

  it('la ficha muestra el historial de despliegues del ambiente', async () => {
    const calls = renderApp('/ambientes/ENV-2026-0001', [], api)

    const history = await screen.findByRole('table', { name: 'Despliegues' })
    expect(within(history).getByRole('link', { name: 'DEP-2026-0001' })).toHaveAttribute('href', '/ambientes/despliegues/DEP-2026-0001')
    expect(within(history).getByText('Exitoso')).toBeInTheDocument()
    expect(calls.some((call) => call.url === '/api/deployments?environment=ENV-2026-0001')).toBe(true)
    expect(screen.queryByRole('button', { name: 'Registrar despliegue' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Editar' })).toBeNull()
  })
})

describe('registrar un despliegue', () => {
  it('pide un BUILD o un PKG, uno de los dos, y envía el elegido', async () => {
    const user = userEvent.setup()
    const calls = renderApp('/ambientes/ENV-2026-0001', ['dep.create'], api)

    await user.click(await screen.findByRole('button', { name: 'Registrar despliegue' }))
    const dialog = screen.getByRole('dialog', { name: 'Registrar despliegue en QA-1' })
    const build = within(dialog).getByRole('combobox', { name: 'BUILD' })
    const bundle = within(dialog).getByRole('combobox', { name: 'PKG' })
    await user.selectOptions(within(dialog).getByRole('combobox', { name: 'Resultado' }), 'Exitoso')
    await user.click(within(dialog).getByRole('button', { name: 'Registrar DEP' }))
    expect(build).toHaveAccessibleDescription(expect.stringContaining('Elige un BUILD o un PKG.'))

    await user.selectOptions(build, await within(build).findByRole('option', { name: /BUILD-2026-0001/ }))
    await user.selectOptions(bundle, await within(bundle).findByRole('option', { name: /PKG-2026-0001/ }))
    await user.click(within(dialog).getByRole('button', { name: 'Registrar DEP' }))
    expect(bundle).toHaveAccessibleDescription(expect.stringContaining('no de los dos'))
    expect(calls.some((call) => call.method === 'POST')).toBe(false)

    await user.selectOptions(build, 'Ninguno')
    await user.click(within(dialog).getByRole('button', { name: 'Registrar DEP' }))
    await within(dialog).findByRole('alert')
    const sent = calls.find((call) => call.method === 'POST')
    expect(sent).toMatchObject({ url: '/api/deployments', body: { environment: 'ENV-2026-0001', package: 'PKG-2026-0001', result: 'success', notes: '' } })
    expect(sent?.body).not.toHaveProperty('build')
  })

  it('en producción el botón solo aparece con dep.create_production', async () => {
    renderApp('/ambientes/ENV-2026-0002', ['dep.create', 'env.manage'], api)
    expect(await screen.findByRole('heading', { level: 1, name: 'PROD' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Editar' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Registrar despliegue' })).toBeNull()
  })

  it('con dep.create_production se puede desplegar en producción, pero no en un ambiente de prueba', async () => {
    renderApp('/ambientes/ENV-2026-0002', ['dep.create_production'], api)
    expect(await screen.findByRole('button', { name: 'Registrar despliegue' })).toBeInTheDocument()
  })

  it('dep.create_production no alcanza para un ambiente de prueba', async () => {
    renderApp('/ambientes/ENV-2026-0001', ['dep.create_production'], api)
    expect(await screen.findByRole('heading', { level: 1, name: 'QA-1' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Registrar despliegue' })).toBeNull()
  })
})
