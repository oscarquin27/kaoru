// Inicio (§7.7): las próximas ventanas de cambio y las alertas de atraso salen del calendario.
import { screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { CALENDAR, CERTIFICATION_SUMMARY, REGRESSION_SUMMARY, SCHEDULE } from '../test-data.ts'
import { fixtureMe, pageOf, renderApp, routes } from '../test-utils.tsx'

describe('inicio', () => {
  it('muestra solo las ventanas futuras y las alertas de atraso', async () => {
    renderApp('/', [], routes({ 'GET /api/calendar': CALENDAR }))

    const windows = await screen.findByRole('region', { name: 'Próximas ventanas de cambio' })
    expect(await within(windows).findByText('CHG-2026-0002')).toBeInTheDocument()
    expect(within(windows).queryByText('CHG-2026-0001')).toBeNull()

    const alerts = within(screen.getByRole('region', { name: 'Alertas de atraso' }))
    expect(await alerts.findByText('Congelar el alcance de 2.14')).toBeInTheDocument()
    expect(alerts.getByText('10 días de atraso')).toBeInTheDocument()
    // Los defectos por estado llevan a la métrica histórica, con los cerrados.
    expect(screen.getByRole('link', { name: 'Ver el histórico, con los cerrados' })).toHaveAttribute('href', '/metricas/defectos')
  })

  it('sin atrasos lo dice', async () => {
    renderApp('/', [], routes({ 'GET /api/calendar': { windows: [], alerts: [] } }))
    expect(await screen.findByText('Ningún hito ni ventana de cambio está atrasado.')).toBeInTheDocument()
    expect(screen.getByText('Ningún cambio tiene una ventana futura definida.')).toBeInTheDocument()
  })

  it('muestra tus actividades y, a quien planifica, los solapes', async () => {
    const calls = renderApp(
      '/',
      ['activity.manage'],
      routes({
        'GET /api/calendar': CALENDAR,
        'GET /api/activities': pageOf([CERTIFICATION_SUMMARY, { ...REGRESSION_SUMMARY, status: 'cancelled' }]),
        'GET /api/planning/schedule': SCHEDULE,
      }),
    )
    const mine = await screen.findByRole('region', { name: 'Tus actividades' })
    expect(await within(mine).findByText('ACT-2026-0001')).toBeInTheDocument()
    expect(within(mine).queryByText('ACT-2026-0002')).toBeNull()
    expect(await within(mine).findByText(/Hay 1 solape entre operadores/)).toBeInTheDocument()
    expect(calls.find((call) => call.url.startsWith('/api/activities'))?.url).toContain(`operator=${fixtureMe.id}`)
  })

  it('sin permiso de planificar no pide la planificación', async () => {
    const calls = renderApp('/', [], routes({ 'GET /api/calendar': CALENDAR, 'GET /api/activities': pageOf([]) }))
    expect(await screen.findByText('No tienes actividades en curso ni planificadas.')).toBeInTheDocument()
    expect(calls.some((call) => call.url.startsWith('/api/planning/schedule'))).toBe(false)
  })
})
