// Alta y edición de un REQ. El sistema no se cambia después de registrar.
import { useNavigate } from 'react-router'
import { useMe, useSystems } from '../../shared/api/session.ts'
import { CheckField, Form, SelectField, TextAreaField, TextField, optional, text } from '../../shared/ui/Form.tsx'
import { useChangeOptions, useCreateRequirement, useUpdateRequirement } from './api.ts'

interface Editable {
  code: string
  title: string
  description: string
  acceptanceCriteria: string
  change?: { code: string }
  owner?: { id: string }
}

export function RequirementForm({ requirement, onClose }: { requirement?: Editable; onClose: () => void }) {
  const navigate = useNavigate()
  const me = useMe().data
  const systems = useSystems().data?.items ?? []
  const changes = useChangeOptions().data?.items ?? []
  const create = useCreateRequirement()
  const update = useUpdateRequirement(requirement?.code ?? '')

  async function submit(data: FormData) {
    const common = { title: text(data, 'title'), description: text(data, 'description'), acceptanceCriteria: text(data, 'acceptanceCriteria') }
    const mine = data.get('mine') === 'on'
    if (requirement) {
      await update.mutateAsync({ ...common, change: optional(data, 'change') ?? null, owner: mine ? me?.id : requirement.owner?.id === me?.id ? null : undefined })
      return
    }
    const created = await create.mutateAsync({ ...common, system: text(data, 'system'), change: optional(data, 'change'), owner: mine ? me?.id : undefined })
    void navigate(`/requerimientos/${created.code}`)
  }

  return (
    <Form
      label={requirement ? 'Editar requerimiento' : 'Nuevo requerimiento'}
      submitLabel={requirement ? 'Guardar' : 'Registrar REQ'}
      onSubmit={submit}
      onDone={onClose}
      onCancel={onClose}
    >
      <TextField name="title" label="Título" required defaultValue={requirement?.title} />
      <div className="form__row">
        {!requirement && (
          <SelectField name="system" label="Sistema" required defaultValue="">
            <option value="">Elige un sistema</option>
            {systems.map((system) => (
              <option key={system.key} value={system.key}>
                {system.name}
              </option>
            ))}
          </SelectField>
        )}
        <SelectField name="change" label="Cambio (CHG)" defaultValue={requirement?.change?.code ?? ''} key={changes.length}>
          <option value="">Sin cambio</option>
          {changes.map((change) => (
            <option key={change.code} value={change.code}>
              {change.code} · {change.title}
            </option>
          ))}
        </SelectField>
      </div>
      <TextAreaField name="description" label="Descripción" defaultValue={requirement?.description} />
      <TextAreaField
        name="acceptanceCriteria"
        label="Criterios de aceptación"
        hint="Uno por línea."
        defaultValue={requirement?.acceptanceCriteria}
      />
      <CheckField name="mine" label="Soy responsable de este requerimiento" defaultChecked={requirement ? requirement.owner?.id === me?.id : false} />
    </Form>
  )
}
