// Plantillas del checklist: la versión vigente de cada sistema y la publicación de otra.
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { TEMPLATE, TEMPLATE_V1 } from '../../test-data.ts'
import { renderApp, routes } from '../../test-utils.tsx'
import { parseSteps } from './forms.tsx'

describe('parseSteps', () => {
  it('toma un paso por renglón y la descripción después de la barra', () => {
    expect(parseSteps('Respaldar la base | Con pg_dump\n\n  Aplicar migraciones  \nAvisar | a | soporte')).toEqual([
      { title: 'Respaldar la base', description: 'Con pg_dump' },
      { title: 'Aplicar migraciones', description: '' },
      { title: 'Avisar', description: 'a | soporte' },
    ])
  })
})

describe('plantillas del checklist', () => {
  it('muestra la versión vigente y publica la siguiente con los pasos de la actual', async () => {
    const user = userEvent.setup()
    const calls = renderApp('/produccion/plantillas', ['chk.manage_template'], routes({ 'GET /api/checklist-templates': { items: [TEMPLATE, TEMPLATE_V1] }, 'POST /api/checklist-templates': TEMPLATE }))

    const card = await screen.findByRole('region', { name: 'SyPago · v2' })
    expect(card).toHaveTextContent('Respaldar la base · Con pg_dump')
    expect(card).toHaveTextContent('Versiones anteriores: v1 (1 paso)')

    await user.click(within(card).getByRole('button', { name: 'Nueva versión' }))
    const dialog = screen.getByRole('dialog', { name: 'Nueva versión de SyPago' })
    const steps = within(dialog).getByRole('textbox', { name: 'Pasos' })
    expect(steps).toHaveValue('Respaldar la base | Con pg_dump\nAplicar migraciones')
    await user.type(steps, '\nAvisar a soporte')
    await user.selectOptions(within(dialog).getByRole('combobox', { name: 'Sistema' }), await within(dialog).findByRole('option', { name: 'SyPago' }))
    await user.click(within(dialog).getByRole('button', { name: 'Publicar versión' }))
    expect(calls.find((call) => call.method === 'POST')?.body).toEqual({
      system: 'SYPAGO',
      steps: [
        { title: 'Respaldar la base', description: 'Con pg_dump' },
        { title: 'Aplicar migraciones', description: '' },
        { title: 'Avisar a soporte', description: '' },
      ],
    })
  })

  it('sin chk.manage_template no se publica', async () => {
    renderApp('/produccion/plantillas', [], routes({ 'GET /api/checklist-templates': { items: [] } }))
    expect(await screen.findByText(/Ningún sistema tiene plantilla publicada/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Publicar plantilla' })).toBeNull()
  })
})
