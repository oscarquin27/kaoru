// Métricas: cobertura por REQ, tiempo en estado, tamaño de regresión y defectos escapados. Leen la API
// con los filtros de la URL y muestran números, nunca metas.
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { COVERAGE_METRICS, DEFECT_STATE_METRICS, ESCAPED_METRICS, REGRESSION_METRICS } from '../../test-data.ts'
import { problemResponse, renderApp, routes } from '../../test-utils.tsx'

const api = routes({
  'GET /api/metrics/coverage': COVERAGE_METRICS,
  'GET /api/metrics/defect-states': DEFECT_STATE_METRICS,
  'GET /api/metrics/regression': REGRESSION_METRICS,
  'GET /api/metrics/escaped-defects': ESCAPED_METRICS,
})

const rowsOf = async (name: string) => within(await screen.findByRole('table', { name })).getAllByRole('row').slice(1)

describe('métricas', () => {
  it('la sección tiene sus cuatro pantallas', async () => {
    renderApp('/metricas', [], api)
    const nav = await screen.findByRole('navigation', { name: 'Métricas' })
    expect(within(nav).getAllByRole('link').map((link) => link.textContent)).toEqual(['Cobertura', 'Tiempo en estado', 'Regresión', 'Defectos escapados'])
  })

  it('cobertura: el resumen y cada REQ frente a su mínimo, lo que hay que atender primero', async () => {
    renderApp('/metricas', [], api)

    const kpis = await screen.findByRole('region', { name: 'Resumen de cobertura' })
    expect(kpis).toHaveTextContent('71 %')
    expect(kpis).toHaveTextContent('5 de 7 TC con su último RUN aprobado')
    expect(kpis).toHaveTextContent('1 de 4 (25 %) REQ')
    expect(kpis).toHaveTextContent('1 sin TC · 1 sin riesgo evaluado')

    const rows = await rowsOf('Cobertura por REQ')
    expect(rows.map((row) => row.textContent)).toEqual([
      expect.stringContaining('REQ-2026-0002'),
      expect.stringContaining('REQ-2026-0003'),
      expect.stringContaining('REQ-2026-0004'),
      expect.stringContaining('REQ-2026-0001'),
    ])
    expect(rows[0]).toHaveTextContent('1 de 3')
    expect(rows[0]).toHaveTextContent('33 %')
    expect(rows[0]).toHaveTextContent('80 %')
    expect(rows[0]).toHaveTextContent('Bajo el mínimo')
    expect(rows[1]).toHaveTextContent('Sin TC')
    expect(rows[2]).toHaveTextContent('Sin evaluar')
    expect(rows[2]).toHaveTextContent('—')
    expect(rows[2]).toHaveTextContent('Sin riesgo')
    expect(rows[3]).toHaveTextContent('Cumple')
  })

  it('cobertura: los filtros viajan en la consulta', async () => {
    const user = userEvent.setup()
    const calls = renderApp('/metricas?system=SYPAGO', [], api)

    await screen.findByRole('table', { name: 'Cobertura por REQ' })
    await user.selectOptions(screen.getByRole('combobox', { name: 'Resultado' }), 'Bajo el mínimo')
    await screen.findByRole('table', { name: 'Cobertura por REQ' })
    const last = calls.filter((call) => call.url.startsWith('/api/metrics/coverage')).at(-1)!
    expect(last.url).toContain('system=SYPAGO')
    expect(last.url).toContain('status=below')
  })

  it('tiempo en estado: mediana y percentil 90 por estado, sin medir la estadía en un estado final', async () => {
    renderApp('/metricas/defectos', [], api)

    expect(await screen.findByText(/Desde el 1 dic 2025: defectos creados en esas fechas/)).toBeInTheDocument()
    const kpis = screen.getByRole('region', { name: 'Resumen de defectos' })
    expect(kpis).toHaveTextContent('6,5 d')
    expect(kpis).toHaveTextContent('Percentil 90: 12,0 d')

    const rows = await rowsOf('Tiempo en estado')
    expect(rows[0]).toHaveTextContent(/Abierto\s*1\s*6\s*6\s*1,5 d\s*4,0 d/)
    expect(rows[1]).toHaveTextContent('3,2 d')
    expect(rows[1]).toHaveTextContent('9,8 d')
    expect(within(rows[2]!).getAllByText('Final')).toHaveLength(2)
    expect(within(rows[3]!).getAllByText('—')).toHaveLength(2)
  })

  it('tiempo en estado: un período al revés muestra el error de la API', async () => {
    renderApp(
      '/metricas/defectos?from=2026-11-14&to=2026-11-01',
      [],
      routes({
        'GET /api/metrics/defect-states': problemResponse(422, {
          code: 'INVALID_REQUEST',
          detail: 'Revisa los datos marcados.',
          errors: [{ field: 'to', code: 'invalid', message: 'La fecha final no puede ser anterior a la inicial.' }],
        }),
      }),
    )
    expect(await screen.findByRole('alert')).toHaveTextContent('La fecha final no puede ser anterior a la inicial.')
  })

  it('regresión: una fila por PKG con lo propuesto, lo ajustado y lo aprobado', async () => {
    renderApp('/metricas/regresion', [], api)

    expect(await screen.findByText(/Del 1 ene 2026 al 30 nov 2026/)).toBeInTheDocument()
    const kpis = screen.getByRole('region', { name: 'Resumen de regresión' })
    expect(kpis).toHaveTextContent('6,5')
    expect(kpis).toHaveTextContent('13 TC finales de 14 propuestos')
    expect(kpis).toHaveTextContent('2 agregados · 3 quitados')

    const rows = await rowsOf('Tamaño de regresión por PKG')
    expect(rows[0]).toHaveTextContent('PKG-2026-0002')
    expect(rows[0]).toHaveTextContent('5 por impacto · 2 críticos · 1 manuales')
    expect(rows[0]).toHaveTextContent('6 de 8')
    expect(within(rows[0]!).getByRole('link', { name: 'PKG-2026-0002' })).toHaveAttribute('href', '/paquetes/PKG-2026-0002')
  })

  it('escapados: la tasa, los meses con su número escrito y cada DEF con el PKG que lo llevó a producción', async () => {
    renderApp('/metricas/escapados', [], api)

    const kpis = await screen.findByRole('region', { name: 'Resumen de escapados' })
    expect(kpis).toHaveTextContent('2')
    expect(kpis).toHaveTextContent('De 8 creados en el período')
    expect(kpis).toHaveTextContent('25 %')

    const months = await rowsOf('Escapados por mes')
    expect(months).toHaveLength(3)
    expect(months[1]).toHaveTextContent(/oct/)
    expect(months[1]).toHaveTextContent('25 %')
    expect(within(months[1]!).getByRole('img', { name: '25 %' })).toBeInTheDocument()

    const [row] = await rowsOf('Defectos escapados')
    expect(row).toHaveTextContent('DEF-2026-0009')
    expect(row).toHaveTextContent('Alta')
    expect(row).toHaveTextContent('24,5 d')
    expect(within(row!).getByRole('link', { name: 'PKG-2026-0001' })).toBeInTheDocument()
  })

  it('escapados: sin escapados lo dice', async () => {
    renderApp('/metricas/escapados', [], routes({ 'GET /api/metrics/escaped-defects': { ...ESCAPED_METRICS, items: [], total: 0 } }))
    expect(await screen.findByText('Ningún defecto del período se abrió después de que su REQ llegó a producción.')).toBeInTheDocument()
  })
})
