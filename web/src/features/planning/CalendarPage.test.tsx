// Calendario: alertas de atraso arriba, hitos y ventanas en una lista por fecha, y los hitos con
// su alta, su edición y su cumplimiento.
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { CALENDAR, MILESTONE, MILESTONE_FUTURE } from '../../test-data.ts'
import { pageOf, renderApp, routes } from '../../test-utils.tsx'

const api = (extra = {}) =>
  routes({
    'GET /api/calendar': CALENDAR,
    'GET /api/milestones': (url: string) => (url.includes('status=pending') ? pageOf([MILESTONE, MILESTONE_FUTURE]) : pageOf([MILESTONE, MILESTONE_FUTURE, { ...MILESTONE, id: 'x', title: 'Hito cumplido', completed: true, daysLate: 0 }])),
    ...extra,
  })

describe('calendario', () => {
  it('pone las alertas arriba, lo más atrasado primero, y debajo todo por fecha', async () => {
    renderApp('/calendario', [], api())

    const alerts = within(await screen.findByRole('list', { name: 'Alertas de atraso' })).getAllByRole('listitem')
    expect(alerts[0]).toHaveTextContent('CHG-2026-0001')
    expect(alerts[0]).toHaveTextContent('10 días de atraso')
    expect(alerts[1]).toHaveTextContent('Congelar el alcance de 2.14')
    expect(alerts[1]).toHaveTextContent('Hito para el 5 nov 2026')

    const rows = within(await screen.findByRole('table', { name: 'Calendario' })).getAllByRole('row').slice(1)
    expect(rows.map((row) => row.textContent)).toEqual([
      expect.stringContaining('CHG-2026-0001'),
      expect.stringContaining('Congelar el alcance de 2.14'),
      expect.stringContaining('Salida a producción'),
      expect.stringContaining('CHG-2026-0002'),
    ])
    expect(rows[1]).toHaveTextContent('7 días de atraso')
    expect(rows[2]).toHaveTextContent('Pendiente')
    expect(rows[2]).toHaveTextContent('Sin responsable')
    expect(rows[3]).toHaveTextContent('Aprobado')
    expect(screen.queryByText('Hito cumplido')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Nuevo hito' })).toBeNull()
  })

  it('con el filtro, también muestra los cumplidos', async () => {
    const user = userEvent.setup()
    renderApp('/calendario', [], api())

    await user.click(await screen.findByRole('checkbox', { name: 'Incluir hitos cumplidos' }))
    expect(await screen.findByText('Hito cumplido')).toBeInTheDocument()
    expect(screen.getByText('Cumplido')).toBeInTheDocument()
  })

  it('con milestone.manage crea un hito y marca otro como cumplido', async () => {
    const user = userEvent.setup()
    const calls = renderApp('/calendario', ['milestone.manage'], api({ 'POST /api/milestones': MILESTONE, [`PATCH /api/milestones/${MILESTONE.id}`]: { ...MILESTONE, completed: true } }))

    await user.click(await screen.findByRole('button', { name: 'Nuevo hito' }))
    const dialog = screen.getByRole('dialog', { name: 'Nuevo hito' })
    await user.type(within(dialog).getByRole('textbox', { name: 'Hito' }), 'Firma del acta')
    await user.type(within(dialog).getByLabelText(/^Fecha/), '2026-12-15')
    await user.click(within(dialog).getByRole('button', { name: 'Crear hito' }))
    expect(calls.find((call) => call.method === 'POST')?.body).toEqual({ title: 'Firma del acta', dueDate: '2026-12-15', owner: '', notes: '' })

    const row = (await screen.findAllByRole('row')).find((item) => item.textContent?.includes('Congelar el alcance'))!
    await user.click(within(row).getByRole('button', { name: 'Marcar cumplido' }))
    expect(calls.find((call) => call.method === 'PATCH')).toEqual({ method: 'PATCH', url: `/api/milestones/${MILESTONE.id}`, body: { completed: true } })
  })
})
