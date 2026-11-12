import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, it, vi } from 'vitest'
import { ApiError } from '../api/errors.ts'
import { Form, SelectField, TextAreaField, TextField } from './Form.tsx'

function renderForm(onSubmit: (data: FormData) => Promise<unknown>) {
  render(
    <Form label="Nuevo requerimiento" submitLabel="Registrar" onSubmit={onSubmit}>
      <TextField name="title" label="Título" required />
      <SelectField name="system" label="Sistema" defaultValue="SIMF">
        <option value="SIMF">SIMF</option>
      </SelectField>
      <TextAreaField name="description" label="Descripción" />
    </Form>,
  )
}

it('pinta los errors[] del servidor bajo cada campo y enfoca el primero', async () => {
  const user = userEvent.setup()
  const serverError = new ApiError(422, 'VALIDATION', 'Revisa los campos marcados.', [
    { field: 'system', code: 'invalid', message: 'El sistema no existe.' },
    { field: 'description', code: 'invalid', message: 'La descripción es demasiado larga.' },
    { field: 'change', code: 'invalid', message: 'El CHG no existe.' },
  ])
  renderForm(() => Promise.reject(serverError))

  await user.type(screen.getByLabelText(/Título/), 'Límite diario')
  await user.click(screen.getByRole('button', { name: 'Registrar' }))

  const system = await screen.findByRole('combobox', { name: 'Sistema' })
  expect(system).toHaveAttribute('aria-invalid', 'true')
  expect(system).toHaveAccessibleDescription('El sistema no existe.')
  expect(system).toHaveFocus()
  expect(screen.getByLabelText('Descripción')).toHaveAccessibleDescription('La descripción es demasiado larga.')
  expect(screen.getByLabelText(/Título/)).not.toHaveAttribute('aria-invalid')
  // Un error de un campo que el formulario no tiene se dice arriba, no se pierde.
  expect(screen.getByRole('alert')).toHaveTextContent('El CHG no existe.')
  expect(screen.getByRole('alert')).not.toHaveTextContent('El sistema no existe.')
})

it('un campo obligatorio vacío se marca sin llamar al servidor', async () => {
  const user = userEvent.setup()
  const onSubmit = vi.fn(() => Promise.resolve())
  renderForm(onSubmit)

  await user.click(screen.getByRole('button', { name: 'Registrar' }))

  expect(onSubmit).not.toHaveBeenCalled()
  const title = screen.getByLabelText(/Título/)
  expect(title).toHaveAccessibleDescription('Este campo es obligatorio.')
  expect(title).toHaveFocus()
})

it('un error sin campos (409) muestra su detalle', async () => {
  const user = userEvent.setup()
  renderForm(() => Promise.reject(new ApiError(409, 'INVALID_ACTION', 'Servidor de datos de ejemplo: no guarda cambios.')))

  await user.type(screen.getByLabelText(/Título/), 'x')
  await user.click(screen.getByRole('button', { name: 'Registrar' }))

  expect(await screen.findByRole('alert')).toHaveTextContent('Servidor de datos de ejemplo: no guarda cambios.')
})

it('un fallo inesperado no muestra el mensaje técnico', async () => {
  const user = userEvent.setup()
  renderForm(() => Promise.reject(new TypeError("Cannot read properties of undefined (reading 'x')")))

  await user.type(screen.getByLabelText(/Título/), 'x')
  await user.click(screen.getByRole('button', { name: 'Registrar' }))

  const alert = await screen.findByRole('alert')
  expect(alert).toHaveTextContent('Algo salió mal.')
  expect(alert).not.toHaveTextContent('Cannot read')
})
