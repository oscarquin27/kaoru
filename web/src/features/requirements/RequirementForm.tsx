// Alta y edición de un REQ. El sistema no se cambia después de registrar.
import { useNavigate } from 'react-router'
import { useSystems } from '../../shared/api/session.ts'
import type { UserRef } from '../../shared/api/types.ts'
import { Form, SelectField, TextAreaField, TextField, optional, text } from '../../shared/ui/Form.tsx'
import { ChangeField, resolveChange } from '../../shared/ui/ChangeField.tsx'
import { UserField } from '../../shared/ui/UserField.tsx'
import { useCreateRequirement, useUpdateRequirement } from './api.ts'

interface Editable {
  code: string
  title: string
  description: string
  acceptanceCriteria: string
  change?: { code: string }
  system?: { key: string }
  owner?: UserRef
}

export function RequirementForm({ requirement, onClose }: { requirement?: Editable; onClose: () => void }) {
  const navigate = useNavigate()
  const systems = useSystems().data?.items ?? []
  const create = useCreateRequirement()
  const update = useUpdateRequirement(requirement?.code ?? '')

  async function submit(data: FormData) {
    const common = { title: text(data, 'title'), description: text(data, 'description'), acceptanceCriteria: text(data, 'acceptanceCriteria') }
    // Un CHG nuevo toma el sistema del REQ; al editar, el que ya tiene el REQ.
    const change = await resolveChange(data, { system: requirement?.system?.key ?? text(data, 'system') })
    if (requirement) {
      await update.mutateAsync({ ...common, change: change || null, owner: optional(data, 'owner') ?? null })
      return
    }
    const created = await create.mutateAsync({ ...common, system: text(data, 'system'), change: change || undefined, owner: optional(data, 'owner') })
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
        <ChangeField defaultValue={requirement?.change?.code} hint="Si el CHG aún no existe, créalo aquí mismo." />
      </div>
      <TextAreaField name="description" label="Descripción" defaultValue={requirement?.description} />
      <TextAreaField
        name="acceptanceCriteria"
        label="Criterios de aceptación"
        hint="Uno por línea."
        defaultValue={requirement?.acceptanceCriteria}
      />
      <UserField name="owner" label="Responsable" current={requirement?.owner} />
    </Form>
  )
}
