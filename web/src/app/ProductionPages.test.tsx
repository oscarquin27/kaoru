// Fichas de producción: firmar los pasos del CHK, revisar un DIAG y ver un SMOKE.
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { CHECKLIST, DIAGNOSTIC, DIAGNOSTIC_REVIEWED, SMOKE } from '../test-data.ts'
import { renderApp, routes } from '../test-utils.tsx'

const common = { 'GET /api/documents': { items: [] }, 'GET /api/audit': { items: [], page: 1, size: 25, total: 0 } }

describe('ficha del checklist', () => {
  const api = (extra = {}) => routes({ 'GET /api/checklists/CHK-2026-0001': CHECKLIST, ...common, ...extra })

  it('muestra quién firmó cada paso y, con chk.sign, firma el que falta con su comentario', async () => {
    const user = userEvent.setup()
    const calls = renderApp('/produccion/CHK-2026-0001', ['chk.sign'], api({ 'POST /api/checklists/CHK-2026-0001/steps/2/signature': CHECKLIST }))

    expect(await screen.findByRole('heading', { level: 1, name: 'Checklist de producción de PKG-2026-0001' })).toBeInTheDocument()
    expect(screen.getByText('1 de 2 firmados')).toBeInTheDocument()
    const rows = within(screen.getByRole('table', { name: 'Pasos del checklist' })).getAllByRole('row')
    expect(rows[1]).toHaveTextContent('Juan Díaz')
    expect(rows[1]).toHaveTextContent('Respaldo en /backups/1112')
    expect(within(rows[1]!).queryByRole('button')).toBeNull()

    await user.click(within(rows[2]!).getByRole('button', { name: 'Firmar' }))
    const dialog = screen.getByRole('dialog', { name: 'Firmar el paso 2 de CHK-2026-0001' })
    await user.type(within(dialog).getByRole('textbox', { name: 'Comentario' }), 'Migración 00031 aplicada')
    await user.click(within(dialog).getByRole('button', { name: 'Firmar' }))
    expect(calls.find((call) => call.method === 'POST')).toEqual({
      method: 'POST',
      url: '/api/checklists/CHK-2026-0001/steps/2/signature',
      body: { comment: 'Migración 00031 aplicada' },
    })
  })

  it('sin chk.sign no hay botón de firmar', async () => {
    renderApp('/produccion/CHK-2026-0001', [], api())
    const table = await screen.findByRole('table', { name: 'Pasos del checklist' })
    expect(within(table).queryByRole('button')).toBeNull()
    expect(within(table).getByText('Sin firmar')).toBeInTheDocument()
  })
})

describe('ficha del diagnóstico', () => {
  it('QA lo revisa; si no está conforme, las notas son obligatorias', async () => {
    const user = userEvent.setup()
    const calls = renderApp(
      '/produccion/diagnosticos/DIAG-2026-0001',
      ['diag.review'],
      routes({ 'GET /api/diagnostics/DIAG-2026-0001': DIAGNOSTIC, 'POST /api/diagnostics/DIAG-2026-0001/review': DIAGNOSTIC_REVIEWED, ...common }),
    )

    expect(await screen.findByText('Todavía nadie de QA lo revisó.')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Revisar' }))
    const dialog = screen.getByRole('dialog', { name: 'Revisar DIAG-2026-0001' })
    await user.selectOptions(within(dialog).getByRole('combobox', { name: 'Resultado' }), 'No conforme')
    await user.click(within(dialog).getByRole('button', { name: 'Guardar revisión' }))
    expect(within(dialog).getByRole('textbox', { name: 'Notas' })).toHaveAccessibleDescription(expect.stringContaining('Di por qué no está conforme.'))
    expect(calls.some((call) => call.method === 'POST')).toBe(false)

    await user.type(within(dialog).getByRole('textbox', { name: 'Notas' }), 'Errores 500 en el cierre')
    await user.click(within(dialog).getByRole('button', { name: 'Guardar revisión' }))
    expect(calls.find((call) => call.method === 'POST')?.body).toEqual({ result: 'non_conforming', notes: 'Errores 500 en el cierre' })
  })

  it('ya revisado, muestra el resultado y no deja revisarlo otra vez', async () => {
    renderApp('/produccion/diagnosticos/DIAG-2026-0001', ['diag.review'], routes({ 'GET /api/diagnostics/DIAG-2026-0001': DIAGNOSTIC_REVIEWED, ...common }))

    const review = await screen.findByRole('region', { name: 'Revisión de QA' })
    expect(review).toHaveTextContent('Conforme')
    expect(review).toHaveTextContent('Ana Rojas')
    expect(within(review).queryByRole('button')).toBeNull()
  })
})

describe('ficha de la prueba de humo', () => {
  it('muestra el resultado, el despliegue y la pestaña de evidencia', async () => {
    renderApp('/produccion/humo/SMOKE-2026-0001', ['doc.upload'], routes({ 'GET /api/smoke-tests/SMOKE-2026-0001': SMOKE, ...common }))

    expect(await screen.findByRole('heading', { level: 1, name: 'Prueba de humo de PKG-2026-0001' })).toBeInTheDocument()
    expect(screen.getByText('El login devuelve 502')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'DEP-2026-0002' })).toHaveAttribute('href', '/ambientes/despliegues/DEP-2026-0002')
    expect(await screen.findByText('Este registro de prueba de humo no tiene documentos adjuntos.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Subir documento' })).toBeInTheDocument()
  })
})
