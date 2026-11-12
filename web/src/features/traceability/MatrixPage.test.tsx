// La matriz con la columna PKG y los filtros de la fase 3.
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import fixtures from '../../../mocks/fixtures.json' with { type: 'json' }
import type { Matrix } from '../../shared/api/types.ts'
import { COMPONENT, PACKAGE } from '../../test-data.ts'
import { pageOf, renderApp, routes } from '../../test-utils.tsx'

const recorded = fixtures['GET /api/traceability/matrix'] as unknown as Matrix
// La grabación es anterior a la fase 3: aquí cada fila trae sus paquetes.
const matrix: Matrix = { ...recorded, items: recorded.items.map((row, index) => ({ ...row, packages: index === 0 ? [PACKAGE.code] : [] })) }

const api = routes({ 'GET /api/traceability/matrix': matrix, 'GET /api/packages': pageOf([PACKAGE]), 'GET /api/components': pageOf([COMPONENT]) })

describe('matriz de trazabilidad', () => {
  it('tiene la columna PKG con enlaces a la ficha del paquete', async () => {
    renderApp('/pruebas', [], api)

    const table = within(await screen.findByRole('region', { name: 'Matriz' }))
    expect(table.getByRole('columnheader', { name: 'PKG' })).toBeInTheDocument()
    expect(table.getByRole('link', { name: 'PKG-2026-0001' })).toHaveAttribute('href', '/paquetes/PKG-2026-0001')
    expect(table.getAllByText('Sin paquete').length).toBeGreaterThan(0)
  })

  it('filtra por PKG y por componente', async () => {
    const user = userEvent.setup()
    const calls = renderApp('/pruebas', [], api)

    const filters = await screen.findByRole('search', { name: 'Filtros' })
    const bundle = within(filters).getByRole('combobox', { name: 'PKG' })
    await user.selectOptions(bundle, await within(bundle).findByRole('option', { name: 'PKG-2026-0001 · SyPago 2.14' }))
    const component = within(filters).getByRole('combobox', { name: 'Componente' })
    await user.selectOptions(component, await within(component).findByRole('option', { name: 'motor-pagos' }))
    expect(calls.some((call) => call.url === '/api/traceability/matrix?package=PKG-2026-0001&component=COMP-2026-0001')).toBe(true)
    // Los filtros nuevos entran en lo que se guarda como vista.
    expect(within(filters).getByRole('button', { name: 'Guardar vista' })).toBeEnabled()
  })

  it('con la respuesta grabada antes de la fase 3 (sin `packages`) la matriz sigue pintando', async () => {
    renderApp('/pruebas')
    const table = within(await screen.findByRole('region', { name: 'Matriz' }))
    expect(table.getAllByText('Sin paquete').length).toBeGreaterThan(0)
  })
})
