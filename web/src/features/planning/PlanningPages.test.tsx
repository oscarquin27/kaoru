// Planificación (fase 6): la línea de tiempo por operador con sus solapes, la lista de actividades
// con su alta, la ficha con su reajuste y la certificación en la ficha del CHG.
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { CERTIFICATION, CERTIFICATION_SUMMARY, OP_ANA, OP_LUIS, OVERLAP, PRODUCTION_SUMMARY, REGRESSION_ACT, REGRESSION_SUMMARY, SCHEDULE } from '../../test-data.ts'
import { jsonResponse, pageOf, renderApp, routes } from '../../test-utils.tsx'
import { certificationState } from './ChangeCertification.tsx'
import { addDays, mondayOf, spanText, today } from './days.ts'
import { packLanes } from './Timeline.tsx'

describe('días y carriles', () => {
  it('cuenta en UTC y empieza la semana en lunes', () => {
    expect(mondayOf('2026-11-15')).toBe('2026-11-09') // domingo
    expect(mondayOf('2026-11-09')).toBe('2026-11-09')
    expect(addDays('2026-12-30', 3)).toBe('2027-01-02')
    expect(spanText('2026-11-18', '2026-11-20')).toBe('del 18 nov al 20 nov')
    expect(spanText('2026-11-21', '2026-11-21')).toBe('el 21 nov')
  })

  it('pone en otro carril solo lo que se pisa', () => {
    const lanes = packLanes([
      { key: 'a', start: '2026-11-10', end: '2026-11-20' },
      { key: 'b', start: '2026-11-18', end: '2026-11-24' },
      { key: 'c', start: '2026-11-21', end: '2026-11-22' },
    ])
    expect(lanes.map((lane) => lane.map((item) => item.key))).toEqual([['a', 'c'], ['b']])
  })

  it('la certificación sigue el criterio de la regla', () => {
    const now = '2026-11-12'
    expect(certificationState([], now).state).toBe('none')
    expect(certificationState([{ ...CERTIFICATION_SUMMARY, status: 'cancelled' }], now).state).toBe('none')
    expect(certificationState([{ ...CERTIFICATION_SUMMARY, operators: [] }], now).state).toBe('no_operators')
    expect(certificationState([CERTIFICATION_SUMMARY], now).state).toBe('ready')
    expect(certificationState([CERTIFICATION_SUMMARY], '2026-11-25').state).toBe('outside')
    // Una regresión del CHG no reserva el período.
    expect(certificationState([{ ...REGRESSION_SUMMARY, change: CERTIFICATION.change }], now).state).toBe('none')
  })
})

describe('planificación por operador', () => {
  it('muestra a cada operador con sus actividades, el solape dicho con palabras y los CHG del rango', async () => {
    const calls = renderApp('/planificacion?desde=2026-11-09&semanas=2', [], routes({ 'GET /api/planning/schedule': SCHEDULE }))

    const summary = await screen.findByRole('region', { name: 'Resumen del rango' })
    expect(summary).toHaveTextContent('Solapes1')
    expect(calls.find((call) => call.url.startsWith('/api/planning/schedule'))?.url).toBe('/api/planning/schedule?from=2026-11-09&to=2026-11-22')

    const overlaps = screen.getByRole('status', { name: 'Solapes' })
    expect(overlaps).toHaveTextContent('Luis Paz: ACT-2026-0001 y ACT-2026-0002 se cruzan del 18 nov al 20 nov (3 días).')

    const timeline = screen.getByRole('region', { name: 'Línea de tiempo por operador' })
    const luis = within(timeline).getByRole('group', { name: 'Luis Paz, 1 solapes' })
    expect(within(luis).getByText('1 solape')).toBeInTheDocument()
    expect(within(luis).getByRole('link', { name: /ACT-2026-0002 · Regresión SIMF · Regresión · del 18 nov al 24 nov/ })).toHaveAttribute('href', '/planificacion/actividades/ACT-2026-0002')
    expect(within(timeline).getByRole('group', { name: 'Ana Rojas' })).toHaveTextContent('1 actividad')

    // El CHG con ventana y sin certificación queda marcado.
    const projects = screen.getByRole('region', { name: 'Línea de tiempo por CHG' })
    expect(within(projects).getByRole('group', { name: 'CHG-2026-0001 Tope diario' })).toBeInTheDocument()
    expect(within(projects).getByRole('group', { name: 'CHG-2026-0003 Cobro QR, sin certificación planificada' })).toHaveTextContent('Ventana del 19 nov al 20 nov')

    expect(screen.getByRole('region', { name: 'Sin operador' })).toHaveTextContent('ACT-2026-0003')
    expect(screen.queryByRole('button', { name: 'Nueva actividad' })).toBeNull()
  })

  it('mueve el rango por semanas y filtra por operador', async () => {
    const user = userEvent.setup()
    const calls = renderApp('/planificacion?desde=2026-11-09', [], routes({ 'GET /api/planning/schedule': SCHEDULE }))
    await screen.findByRole('region', { name: 'Resumen del rango' })

    await user.click(screen.getByRole('button', { name: 'Semana siguiente' }))
    expect(calls.at(-1)?.url).toBe('/api/planning/schedule?from=2026-11-16&to=2026-12-27')
    await user.selectOptions(screen.getByRole('combobox', { name: 'Operador' }), 'Luis Paz')
    expect(calls.at(-1)?.url).toContain(`operator=${OVERLAP.operator.id}`)
    await user.click(screen.getByRole('button', { name: 'Esta semana' }))
    expect(calls.at(-1)?.url).toContain(`from=${mondayOf(today())}`)
  })

  it('sin actividades lo dice', async () => {
    const empty = { ...SCHEDULE, summary: { activities: 0, operators: 0, overlaps: 0 }, operators: [], unassigned: [], projects: [], windows: [] }
    renderApp('/planificacion', [], routes({ 'GET /api/planning/schedule': empty }))
    expect(await screen.findByText('Nadie tiene actividades en este rango.')).toBeInTheDocument()
    expect(screen.queryByRole('status', { name: 'Solapes' })).toBeNull()
  })
})

describe('actividades', () => {
  it('lista con filtros y, con activity.manage, da de alta y lleva a la ficha', async () => {
    const user = userEvent.setup()
    const calls = renderApp(
      '/planificacion/actividades?kind=certification',
      ['activity.manage'],
      routes({
        'GET /api/activities': pageOf([CERTIFICATION_SUMMARY, PRODUCTION_SUMMARY]),
        'POST /api/activities': REGRESSION_ACT,
        [`GET /api/activities/${REGRESSION_ACT.code}`]: REGRESSION_ACT,
      }),
    )

    const rows = within(await screen.findByRole('table', { name: 'Actividades' })).getAllByRole('row').slice(1)
    expect(rows[0]).toHaveTextContent('Ana Rojas, Luis Paz')
    expect(rows[0]).toHaveTextContent('11 días')
    expect(rows[1]).toHaveTextContent('Sin operador')
    expect(calls.find((call) => call.url.startsWith('/api/activities'))?.url).toBe('/api/activities?kind=certification')

    await user.click(screen.getByRole('button', { name: 'Nueva actividad' }))
    const dialog = screen.getByRole('dialog', { name: 'Nueva actividad' })
    // La certificación sin CHG no se envía.
    await user.type(within(dialog).getByRole('textbox', { name: /^Actividad/ }), 'Regresión SIMF')
    await user.type(within(dialog).getByLabelText(/^Desde/), '2026-11-18')
    await user.type(within(dialog).getByLabelText(/^Hasta/), '2026-11-24')
    await user.click(within(dialog).getByRole('button', { name: 'Crear actividad' }))
    expect(within(dialog).getByText('La certificación es de un CHG: indica cuál.')).toBeInTheDocument()

    await user.selectOptions(within(dialog).getByRole('combobox', { name: /^Clase/ }), 'Regresión')
    await user.click(within(dialog).getByRole('checkbox', { name: 'Luis Paz' }))
    await user.click(within(dialog).getByRole('button', { name: 'Crear actividad' }))
    expect(calls.find((call) => call.method === 'POST')?.body).toEqual({
      kind: 'regression',
      title: 'Regresión SIMF',
      startDate: '2026-11-18',
      endDate: '2026-11-24',
      operators: [OVERLAP.operator.id],
      notes: '',
    })
    expect(await screen.findByRole('heading', { name: 'Regresión SIMF' })).toBeInTheDocument()
  })
})

describe('CHG que todavía no existe', () => {
  it('se crea desde la actividad y la actividad queda con su código', async () => {
    const user = userEvent.setup()
    const calls = renderApp(
      '/planificacion/actividades',
      ['activity.manage'],
      routes({
        'GET /api/activities': pageOf([]),
        'POST /api/changes': { code: 'CHG-2026-0042', title: 'Corte de noviembre' },
        'POST /api/activities': CERTIFICATION,
        [`GET /api/activities/${CERTIFICATION.code}`]: CERTIFICATION,
      }),
    )

    await user.click(await screen.findByRole('button', { name: 'Nueva actividad' }))
    const dialog = screen.getByRole('dialog', { name: 'Nueva actividad' })
    await user.selectOptions(within(dialog).getByRole('combobox', { name: /^CHG/ }), '＋ Crear un CHG nuevo…')
    await user.type(within(dialog).getByRole('textbox', { name: /^Título del CHG nuevo/ }), 'Corte de noviembre')
    await user.selectOptions(within(dialog).getByRole('combobox', { name: /^Sistema del CHG/ }), await within(dialog).findByRole('option', { name: 'SyPago' }))
    await user.type(within(dialog).getByRole('textbox', { name: /^Actividad/ }), 'Certificación del corte')
    await user.type(within(dialog).getByLabelText(/^Desde/), '2026-11-18')
    await user.type(within(dialog).getByLabelText(/^Hasta/), '2026-11-24')
    await user.click(within(dialog).getByRole('button', { name: 'Crear actividad' }))

    expect(await screen.findByRole('heading', { name: CERTIFICATION.title })).toBeInTheDocument()
    const posts = calls.filter((call) => call.method === 'POST')
    expect(posts.map((call) => call.url)).toEqual(['/api/changes', '/api/activities'])
    expect(posts[0]?.body).toEqual({ title: 'Corte de noviembre', system: 'SYPAGO', route: 'light' })
    expect(posts[1]?.body).toMatchObject({ kind: 'certification', change: 'CHG-2026-0042', title: 'Certificación del corte' })
  })
})

describe('ficha de la actividad', () => {
  const withOverlap = { ...CERTIFICATION, overlaps: [OVERLAP] }

  it('muestra el solape y la auditoría con nombres, y se cancela', async () => {
    const user = userEvent.setup()
    const calls = renderApp(
      `/planificacion/actividades/${CERTIFICATION.code}`,
      ['activity.manage', 'audit.read'],
      routes({
        [`GET /api/activities/${CERTIFICATION.code}`]: withOverlap,
        [`PATCH /api/activities/${CERTIFICATION.code}`]: { ...withOverlap, status: 'cancelled', overlaps: [] },
        'GET /api/audit': pageOf([
          {
            seq: 2,
            occurredAt: '2026-11-12T11:00:00Z',
            actor: { id: 'x', displayName: 'Marta León' },
            entityType: 'activity',
            entityCode: CERTIFICATION.code,
            action: 'update',
            field: 'operators',
            oldValue: '',
            newValue: `${OP_LUIS.id},${OP_ANA.id}`,
          },
        ]),
      }),
    )

    expect(await screen.findByRole('heading', { name: CERTIFICATION.title })).toBeInTheDocument()
    expect(screen.getByRole('status', { name: 'Solapes de la actividad' })).toHaveTextContent('se cruzan del 18 nov al 20 nov (3 días)')
    const history = await screen.findByRole('region', { name: 'Actividad' })
    expect(await within(history).findByText(/cambió los operadores/)).toBeInTheDocument()
    expect(history).toHaveTextContent('Marta León cambió los operadores de actividad ACT-2026-0001: ninguno → Luis Paz, Ana Rojas.')

    await user.click(screen.getByRole('button', { name: 'Cancelar actividad' }))
    expect(calls.find((call) => call.method === 'PATCH')?.body).toEqual({ cancelled: true })
  })

  it('sin permiso no se edita', async () => {
    renderApp(`/planificacion/actividades/${CERTIFICATION.code}`, [], routes({ [`GET /api/activities/${CERTIFICATION.code}`]: CERTIFICATION }))
    expect(await screen.findByText('Ninguno de sus operadores tiene otra actividad esos días.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Editar' })).toBeNull()
  })
})

describe('certificación en la ficha del CHG', () => {
  it('sin período avisa y deja planificarlo con el CHG puesto', async () => {
    const user = userEvent.setup()
    const calls = renderApp(
      '/cambios/CHG-2026-0001',
      ['activity.manage'],
      routes({ 'GET /api/activities': pageOf([]), 'POST /api/activities': { ...CERTIFICATION, overlaps: [OVERLAP] } }),
    )

    const section = await screen.findByRole('region', { name: 'Certificación' })
    expect(await within(section).findByText('Sin período de certificación')).toBeInTheDocument()
    expect(calls.find((call) => call.url.startsWith('/api/activities'))?.url).toBe('/api/activities?change=CHG-2026-0001&size=100')

    await user.click(within(section).getByRole('button', { name: 'Planificar certificación' }))
    const dialog = screen.getByRole('dialog', { name: 'Planificar la certificación de CHG-2026-0001' })
    expect(within(dialog).getByRole('combobox', { name: /^CHG/ })).toHaveValue('CHG-2026-0001')
    await user.type(within(dialog).getByLabelText(/^Desde/), '2026-11-10')
    await user.type(within(dialog).getByLabelText(/^Hasta/), '2026-11-20')
    await user.click(within(dialog).getByRole('button', { name: 'Crear actividad' }))
    expect(calls.find((call) => call.method === 'POST')?.body).toMatchObject({ kind: 'certification', change: 'CHG-2026-0001', title: 'Certificación CHG-2026-0001' })
    // Se queda en la ficha del CHG y avisa el solape que produjo.
    expect(await screen.findByRole('status', { name: `Solapes de ${CERTIFICATION.code}` })).toHaveTextContent('Luis Paz: ACT-2026-0001 y ACT-2026-0002')
    expect(screen.getByRole('region', { name: 'Certificación' })).toBeInTheDocument()
  })

  it('con período vigente y operadores lo dice', async () => {
    const now = today()
    const current = { ...CERTIFICATION_SUMMARY, startDate: addDays(now, -1), endDate: addDays(now, 3) }
    renderApp('/cambios/CHG-2026-0001', [], routes({ 'GET /api/activities': () => jsonResponse(pageOf([current])) }))
    const section = await screen.findByRole('region', { name: 'Certificación' })
    expect(await within(section).findByText('En período de certificación')).toBeInTheDocument()
    expect(within(section).getByText(/Ana Rojas, Luis Paz/)).toBeInTheDocument()
  })
})
