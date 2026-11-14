// Inicio (§7.7): las próximas ventanas de cambio y las alertas de atraso salen del calendario.
import { screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { CALENDAR } from '../test-data.ts'
import { renderApp, routes } from '../test-utils.tsx'

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
})
