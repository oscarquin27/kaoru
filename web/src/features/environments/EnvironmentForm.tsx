// Alta y edición de un ambiente (ENV). El sistema no se cambia después de crear.
import { useNavigate } from 'react-router'
import { useSystems } from '../../shared/api/session.ts'
import type { Environment, EnvironmentKind, OperatingSystem } from '../../shared/api/types.ts'
import { ENVIRONMENT_KIND_LABELS, OS_LABELS, options } from '../../shared/lib/labels.ts'
import { Form, SelectField, TextAreaField, TextField, text } from '../../shared/ui/Form.tsx'
import { useCreateEnvironment, useUpdateEnvironment } from './api.ts'

export function EnvironmentForm({ environment, onClose }: { environment?: Environment; onClose: () => void }) {
  const navigate = useNavigate()
  const systems = useSystems().data?.items ?? []
  const create = useCreateEnvironment()
  const update = useUpdateEnvironment(environment?.code ?? '')

  async function submit(data: FormData) {
    const common = { name: text(data, 'name'), kind: text(data, 'kind') as EnvironmentKind, os: text(data, 'os') as OperatingSystem, notes: text(data, 'notes') }
    if (environment) {
      await update.mutateAsync(common)
      return
    }
    const created = await create.mutateAsync({ ...common, system: text(data, 'system') })
    void navigate(`/ambientes/${created.code}`)
  }

  return (
    <Form label={environment ? 'Editar ambiente' : 'Nuevo ambiente'} submitLabel={environment ? 'Guardar' : 'Registrar ENV'} onSubmit={submit} onDone={onClose} onCancel={onClose}>
      <div className="form__row">
        <TextField name="name" label="Nombre" required maxLength={80} defaultValue={environment?.name} placeholder="QA-1" hint="No se repite dentro de un sistema." />
        {!environment && (
          <SelectField name="system" label="Sistema" required defaultValue="">
            <option value="">Elige un sistema</option>
            {systems.map((system) => (
              <option key={system.key} value={system.key}>
                {system.name}
              </option>
            ))}
          </SelectField>
        )}
      </div>
      <div className="form__row">
        <SelectField name="kind" label="Tipo" required defaultValue={environment?.kind ?? ''} hint="En producción no se registran RUN, y sus despliegues los registra change management.">
          <option value="">Elige el tipo</option>
          {options(ENVIRONMENT_KIND_LABELS).map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </SelectField>
        <SelectField name="os" label="Sistema operativo" required defaultValue={environment?.os ?? ''}>
          <option value="">Elige el sistema operativo</option>
          {options(OS_LABELS).map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </SelectField>
      </div>
      <TextAreaField name="notes" label="Notas" defaultValue={environment?.notes} />
    </Form>
  )
}
