// El RUN se registra sobre un BUILD y un ambiente del catálogo, del sistema del TC.
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { BUILD, ENVIRONMENT, PRODUCTION } from '../../test-data.ts'
import { pageOf, renderApp, routes } from '../../test-utils.tsx'

const PATH = '/requerimientos/REQ-2026-0001#pruebas'

describe('formulario de RUN', () => {
  it('ofrece los BUILD y los ambientes de prueba del sistema, y envía sus códigos', async () => {
    const user = userEvent.setup()
    const calls = renderApp(PATH, ['run.create'], routes({ 'GET /api/builds': pageOf([BUILD]), 'GET /api/environments': pageOf([ENVIRONMENT, PRODUCTION]) }))

    await user.click(await screen.findByRole('button', { name: 'Registrar RUN de TC-2026-0001' }))
    const dialog = screen.getByRole('dialog', { name: 'Registrar RUN de TC-2026-0001' })
    const build = within(dialog).getByRole('combobox', { name: 'BUILD' })
    const environment = within(dialog).getByRole('combobox', { name: 'Ambiente' })
    await user.selectOptions(build, await within(build).findByRole('option', { name: /2\.14\.0 · BUILD-2026-0001/ }))
    // QA no ejecuta en producción: ese ambiente no se ofrece.
    expect(within(environment).getAllByRole('option').map((option) => option.textContent)).toEqual(['Elige el ambiente', 'QA-1 · Línea base'])
    await user.selectOptions(environment, 'QA-1 · Línea base')
    await user.selectOptions(within(dialog).getByRole('combobox', { name: 'Resultado' }), 'Aprobado')
    await user.click(within(dialog).getByRole('button', { name: 'Registrar RUN' }))
    await within(dialog).findByRole('alert')

    // Los catálogos se piden para el sistema del TC.
    expect(calls.some((call) => call.url === '/api/builds?system=SYPAGO&size=200')).toBe(true)
    expect(calls.some((call) => call.url === '/api/environments?system=SYPAGO&size=200')).toBe(true)
    const sent = calls.find((call) => call.method === 'POST')
    expect(sent?.url).toBe('/api/runs')
    expect(sent?.body).toMatchObject({ testCase: 'TC-2026-0001', build: 'BUILD-2026-0001', environment: 'ENV-2026-0001', result: 'passed' })
    expect(sent?.body).not.toHaveProperty('buildVersion')
    expect(sent?.body).not.toHaveProperty('buildChecksum')
    expect(sent?.body).not.toHaveProperty('environmentName')
  })

  it('sin BUILD ni ambientes de prueba dice dónde registrarlos', async () => {
    const user = userEvent.setup()
    renderApp(PATH, ['run.create'], routes({ 'GET /api/builds': pageOf([]), 'GET /api/environments': pageOf([PRODUCTION]) }))

    await user.click(await screen.findByRole('button', { name: 'Registrar RUN de TC-2026-0001' }))
    const notice = await within(screen.getByRole('dialog')).findByRole('status')
    expect(notice).toHaveTextContent('no tiene ningún BUILD registrado')
    expect(notice).toHaveTextContent('no tiene ambientes de prueba registrados')
    expect(within(notice).getByRole('link', { name: 'Paquetes · BUILD' })).toHaveAttribute('href', '/paquetes/builds')
    expect(within(notice).getByRole('link', { name: 'Ambientes' })).toHaveAttribute('href', '/ambientes')
  })

  it('la exploración también elige BUILD y ambiente del catálogo', async () => {
    const user = userEvent.setup()
    const calls = renderApp(PATH, ['exp.create'], routes({ 'GET /api/builds': pageOf([BUILD]), 'GET /api/environments': pageOf([ENVIRONMENT]) }))

    await user.click(await screen.findByRole('button', { name: 'Registrar EXP' }))
    const dialog = screen.getByRole('dialog', { name: 'Registrar exploración' })
    const build = within(dialog).getByRole('combobox', { name: 'BUILD' })
    await user.selectOptions(build, await within(build).findByRole('option', { name: /BUILD-2026-0001/ }))
    await user.selectOptions(within(dialog).getByRole('combobox', { name: 'Ambiente' }), 'QA-1 · Línea base')
    await user.click(within(dialog).getByRole('button', { name: 'Registrar EXP' }))
    await within(dialog).findByRole('alert')
    expect(calls.find((call) => call.method === 'POST')).toMatchObject({
      url: '/api/explorations',
      body: { requirement: 'REQ-2026-0001', build: 'BUILD-2026-0001', environment: 'ENV-2026-0001' },
    })
  })
})
