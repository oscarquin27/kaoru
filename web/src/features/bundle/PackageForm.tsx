// Alta y edición de un paquete (PKG). Nace «En armado»; el sistema no se cambia después.
import { useNavigate } from 'react-router'
import { useSystems } from '../../shared/api/session.ts'
import type { Package } from '../../shared/api/types.ts'
import { Form, SelectField, TextAreaField, TextField, text } from '../../shared/ui/Form.tsx'
import { useCreatePackage, useUpdatePackage } from './api.ts'

export function PackageForm({ bundle, onClose }: { bundle?: Package; onClose: () => void }) {
  const navigate = useNavigate()
  const systems = useSystems().data?.items ?? []
  const create = useCreatePackage()
  const update = useUpdatePackage(bundle?.code ?? '')

  async function submit(data: FormData) {
    const common = { title: text(data, 'title'), description: text(data, 'description') }
    if (bundle) {
      await update.mutateAsync(common)
      return
    }
    const created = await create.mutateAsync({ ...common, system: text(data, 'system') })
    void navigate(`/paquetes/${created.code}`)
  }

  return (
    <Form label={bundle ? 'Editar paquete' : 'Nuevo paquete'} submitLabel={bundle ? 'Guardar' : 'Crear PKG'} onSubmit={submit} onDone={onClose} onCancel={onClose}>
      <TextField name="title" label="Título" required maxLength={200} defaultValue={bundle?.title} />
      {!bundle && (
        <SelectField name="system" label="Sistema" required defaultValue="" hint="Un paquete solo lleva BUILD y REQ de su sistema.">
          <option value="">Elige un sistema</option>
          {systems.map((system) => (
            <option key={system.key} value={system.key}>
              {system.name}
            </option>
          ))}
        </SelectField>
      )}
      <TextAreaField name="description" label="Descripción" defaultValue={bundle?.description} />
    </Form>
  )
}
