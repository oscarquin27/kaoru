// Abrir un DEF desde la ficha del REQ: si el RUN (o su BUILD) todavía no existe, se registra al guardar.
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { BUILD, ENVIRONMENT } from '../../test-data.ts'
import { pageOf, renderApp, routes } from '../../test-utils.tsx'

const PATH = '/requerimientos/REQ-2026-0001#defectos'
const CHECKSUM = 'a'.repeat(64)

async function openDefect(permissions: string[], table: Parameters<typeof routes>[0]) {
  const user = userEvent.setup()
  const calls = renderApp(PATH, permissions, routes({ 'GET /api/runs': pageOf([]), 'GET /api/environments': pageOf([ENVIRONMENT]), ...table }))
  await user.click(await screen.findByRole('button', { name: 'Abrir DEF' }))
  const form = screen.getByRole('form', { name: 'Abrir defecto' })
  await user.type(within(form).getByRole('textbox', { name: /^Título/ }), 'Acepta sobre el límite')
  await user.type(within(form).getByRole('textbox', { name: /^Descripción/ }), 'Pasos para reproducirlo')
  await user.selectOptions(within(form).getByRole('combobox', { name: /^Severidad/ }), 'Alta')
  return { user, calls, form }
}

describe('abrir un DEF sin RUN registrado', () => {
  it('registra el BUILD, luego el RUN y luego el DEF', async () => {
    const { user, calls, form } = await openDefect(['def.create', 'build.register'], {
      'GET /api/builds': pageOf([]),
      'POST /api/builds': { code: 'BUILD-2026-0042' },
      'POST /api/runs': { code: 'RUN-2026-0042' },
      'POST /api/defects': { code: 'DEF-2026-0042' },
    })
    // Sin RUN del REQ, ya viene elegido «Registrar un RUN nuevo».
    expect(within(form).getByRole('combobox', { name: /^RUN que lo reprodujo/ })).toHaveValue('__new__')
    await user.selectOptions(within(form).getByRole('combobox', { name: /^TC que lo reprodujo/ }), await within(form).findByRole('option', { name: /^TC-2026-0001/ }))
    // Un RUN que reproduce un defecto no puede estar aprobado.
    expect(within(within(form).getByRole('combobox', { name: /^Resultado del RUN/ })).queryByRole('option', { name: 'Aprobado' })).toBeNull()
    await user.selectOptions(within(form).getByRole('combobox', { name: /^BUILD/ }), '＋ Registrar un BUILD nuevo…')
    await user.type(within(form).getByRole('textbox', { name: /^Versión del BUILD nuevo/ }), '2.15.0')
    await user.type(within(form).getByRole('textbox', { name: /^SHA-256/ }), CHECKSUM)
    await user.selectOptions(within(form).getByRole('combobox', { name: /^Ambiente/ }), 'QA-1 · Línea base')
    await user.click(within(form).getByRole('button', { name: 'Abrir DEF' }))

    await waitFor(() => expect(calls.filter((call) => call.method === 'POST')).toHaveLength(3))
    const [build, run, defect] = calls.filter((call) => call.method === 'POST')
    expect(build).toMatchObject({ url: '/api/builds', body: { system: 'SYPAGO', version: '2.15.0', checksum: CHECKSUM, requirements: ['REQ-2026-0001'] } })
    expect(run).toMatchObject({
      url: '/api/runs',
      body: { testCase: 'TC-2026-0001', build: 'BUILD-2026-0042', environment: 'ENV-2026-0001', result: 'failed', notes: 'Pasos para reproducirlo' },
    })
    expect(defect).toMatchObject({ url: '/api/defects', body: { requirement: 'REQ-2026-0001', run: 'RUN-2026-0042', severity: 'high' } })
  })

  it('con un BUILD que ya existe solo registra el RUN; sin build.register no ofrece crear el BUILD', async () => {
    const { user, calls, form } = await openDefect(['def.create'], {
      'GET /api/builds': pageOf([BUILD]),
      'POST /api/runs': { code: 'RUN-2026-0042' },
      'POST /api/defects': { code: 'DEF-2026-0042' },
    })
    await user.selectOptions(within(form).getByRole('combobox', { name: /^TC que lo reprodujo/ }), await within(form).findByRole('option', { name: /^TC-2026-0001/ }))
    const build = within(form).getByRole('combobox', { name: /^BUILD/ })
    expect(within(build).queryByRole('option', { name: /Registrar un BUILD nuevo/ })).toBeNull()
    await user.selectOptions(build, await within(build).findByRole('option', { name: /BUILD-2026-0001/ }))
    await user.selectOptions(within(form).getByRole('combobox', { name: /^Ambiente/ }), 'QA-1 · Línea base')
    await user.click(within(form).getByRole('button', { name: 'Abrir DEF' }))

    await waitFor(() => expect(calls.filter((call) => call.method === 'POST')).toHaveLength(2))
    const [run, defect] = calls.filter((call) => call.method === 'POST')
    expect(run).toMatchObject({ url: '/api/runs', body: { build: 'BUILD-2026-0001' } })
    expect(defect).toMatchObject({ url: '/api/defects', body: { run: 'RUN-2026-0042' } })
  })
})

describe('DEF aceptado al aprobar su REQ con defectos', () => {
  it('la ficha dice quién lo aceptó y por qué, y que ya no bloquea', async () => {
    renderApp(
      '/defectos/DEF-2026-0042',
      [],
      routes({
        'GET /api/defects/DEF-2026-0042': {
          code: 'DEF-2026-0042',
          title: 'Acepta sobre el límite',
          description: 'Pasos',
          requirement: 'REQ-2026-0001',
          run: 'RUN-2026-0042',
          severity: 'high',
          state: { key: 'open', name: 'Abierto' },
          daysInState: 1,
          createdBy: { id: 'u1', displayName: 'Ana' },
          createdAt: '2026-11-19T12:00:00Z',
          updatedAt: '2026-11-20T12:00:00Z',
          acceptance: { by: { id: 'u2', displayName: 'Rosa' }, reason: 'El REQ vino incompleto', at: '2026-11-20T12:00:00Z' },
        },
      }),
    )
    const note = await screen.findByRole('note')
    expect(note).toHaveTextContent('Aceptado al aprobar REQ-2026-0001 con defectos, por Rosa')
    expect(note).toHaveTextContent('ya no bloquea al REQ ni al PKG')
    expect(note).toHaveTextContent('Motivo: El REQ vino incompleto')
  })
})
