// Alta y edición de un hito del calendario. Cumplirlo se marca aparte, desde la lista.
import type { Milestone } from '../../shared/api/types.ts'
import { Form, TextAreaField, TextField, text } from '../../shared/ui/Form.tsx'
import { UserField } from '../../shared/ui/UserField.tsx'
import { useCreateMilestone, useUpdateMilestone } from './api.ts'

export function MilestoneForm({ milestone, onClose }: { milestone?: Milestone; onClose: () => void }) {
  const create = useCreateMilestone()
  const update = useUpdateMilestone(milestone?.id ?? '')
  return (
    <Form
      label={milestone ? 'Editar hito' : 'Nuevo hito'}
      submitLabel={milestone ? 'Guardar' : 'Crear hito'}
      onSubmit={(data) => {
        const fields = { title: text(data, 'title'), dueDate: text(data, 'dueDate'), owner: text(data, 'owner'), notes: text(data, 'notes') }
        return milestone ? update.mutateAsync(fields) : create.mutateAsync(fields)
      }}
      onDone={onClose}
      onCancel={onClose}
    >
      <TextField name="title" label="Hito" required maxLength={200} defaultValue={milestone?.title} />
      <TextField name="dueDate" label="Fecha" type="date" required defaultValue={milestone?.dueDate} hint="Pasada esta fecha sin cumplirse, el hito sale en las alertas de atraso." />
      <UserField name="owner" label="Responsable" current={milestone?.owner} />
      <TextAreaField name="notes" label="Notas" defaultValue={milestone?.notes} />
    </Form>
  )
}
