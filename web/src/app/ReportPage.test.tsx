// Ficha del informe de conformidad: el Markdown, el hash, la cadena de versiones y las descargas.
import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { CHECKSUM, REPORT, REPORT_V1 } from '../test-data.ts'
import { renderApp, routes } from '../test-utils.tsx'

const audit = { 'GET /api/audit': { items: [], page: 1, size: 25, total: 0 } }

describe('ficha del informe', () => {
  it('pinta el informe, su hash y las descargas en PDF y Markdown', async () => {
    renderApp('/informes/RPT-2026-0002', [], routes({ 'GET /api/reports/RPT-2026-0002': REPORT, ...audit }))

    expect(await screen.findByRole('heading', { level: 1, name: 'Informe de conformidad de PKG-2026-0001' })).toBeInTheDocument()
    expect(screen.getByText(CHECKSUM)).toBeInTheDocument()
    expect(screen.getByText('Vigente')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'RPT-2026-0001' })).toHaveAttribute('href', '/informes/RPT-2026-0001')
    expect(screen.getByRole('link', { name: 'Descargar PDF' })).toHaveAttribute('href', '/api/reports/RPT-2026-0002/export?format=pdf')
    expect(screen.getByRole('link', { name: 'Descargar Markdown' })).toHaveAttribute('download', 'informe-RPT-2026-0002-v2.md')
    expect(screen.getByRole('heading', { name: '1. Contenido' })).toBeInTheDocument()
    expect(screen.getByRole('cell', { name: '2.14.0' })).toBeInTheDocument()
  })

  it('una versión reemplazada lo dice y enlaza a la vigente', async () => {
    renderApp('/informes/RPT-2026-0001', [], routes({ 'GET /api/reports/RPT-2026-0001': REPORT_V1, ...audit }))

    expect(await screen.findByRole('note')).toHaveTextContent('Este informe fue reemplazado por RPT-2026-0002.')
    expect(screen.getByText('Es la primera versión')).toBeInTheDocument()
  })
})
