// El visor de un documento: un archivo de Office se ve convertido a PDF solo si el servidor lo ofrece.
import { screen } from '@testing-library/react'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import fixtures from '../../../mocks/fixtures.json' with { type: 'json' }
import { App } from '../../app/App.tsx'
import { createQueryClient } from '../../shared/api/query.ts'
import type { Document } from '../../shared/api/types.ts'
import { stubApi } from '../../test-utils.tsx'

const recorded = fixtures['GET /api/documents/DOC-2026-0001'] as unknown as Document

function renderWord(previewable: boolean) {
  const word: Document = {
    ...recorded,
    versions: [{ ...recorded.versions[0]!, version: 1, fileName: 'evidencia.docx', mediaType: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', previewable }],
  }
  stubApi((method, url) =>
    method === 'GET' && url === '/api/documents/DOC-2026-0001' ? new Response(JSON.stringify(word), { headers: { 'Content-Type': 'application/json' } }) : undefined,
  )
  render(
    <MemoryRouter initialEntries={['/documentos/DOC-2026-0001']}>
      <App queryClient={createQueryClient()} />
    </MemoryRouter>,
  )
}

describe('visor de documentos de Office', () => {
  it('con conversor, muestra la vista previa en PDF y avisa que el original es la evidencia', async () => {
    renderWord(true)
    expect(await screen.findByText(/Vista previa convertida a PDF/)).toBeInTheDocument()
    expect(document.querySelector('object')).toHaveAttribute('data', '/api/documents/DOC-2026-0001/versions/1/preview')
    expect(screen.getByRole('link', { name: /Descargar/ })).toHaveAttribute('href', expect.stringContaining('/versions/1/content'))
  })

  it('sin conversor, solo ofrece descargarlo', async () => {
    renderWord(false)
    expect(await screen.findByText(/no se ve dentro del sistema/)).toBeInTheDocument()
    expect(document.querySelector('object')).toBeNull()
  })
})
