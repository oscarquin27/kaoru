// Alta y edición de una actividad: qué es, de qué CHG, de qué día a qué día y quiénes la hacen.
// Guardar una que se cruza con otra no se impide: la ficha muestra el solape después.
import { useNavigate } from 'react-router'
import type { FieldError } from '../../shared/api/errors.ts'
import { useUsers } from '../../shared/api/session.ts'
import type { Activity, ActivityKind } from '../../shared/api/types.ts'
import { ACTIVITY_KIND_LABELS, options } from '../../shared/lib/labels.ts'
import { ChangeField, resolveChange } from '../../shared/ui/ChangeField.tsx'
import { CheckGroupField, Form, SelectField, TextAreaField, TextField, text } from '../../shared/ui/Form.tsx'
import { useCreateActivity, useUpdateActivity } from './api.ts'

export interface ActivityDefaults {
  kind?: ActivityKind
  change?: string
  title?: string
}

function validate(data: FormData): FieldError[] {
  const errors: FieldError[] = []
  const start = text(data, 'startDate')
  const end = text(data, 'endDate')
  if (start && end && end < start) errors.push({ field: 'endDate', code: 'invalid', message: 'La fecha final no puede ser anterior a la inicial.' })
  if (text(data, 'kind') === 'certification' && !text(data, 'change')) errors.push({ field: 'change', code: 'required', message: 'La certificación es de un CHG: indica cuál.' })
  return errors
}

interface Props {
  activity?: Activity
  defaults?: ActivityDefaults
  /** Con él, al crear se queda donde está (la ficha del CHG) y recibe la actividad con sus solapes; sin él, va a la ficha nueva. */
  onCreated?: (activity: Activity) => void
  onClose: () => void
}

export function ActivityForm({ activity, defaults, onCreated, onClose }: Props) {
  const navigate = useNavigate()
  const users = useUsers().data?.items ?? []
  const create = useCreateActivity()
  const update = useUpdateActivity(activity?.code ?? '')

  async function submit(data: FormData) {
    const fields = {
      kind: text(data, 'kind') as ActivityKind,
      title: text(data, 'title'),
      startDate: text(data, 'startDate'),
      endDate: text(data, 'endDate'),
      operators: data.getAll('operators').map(String),
      notes: text(data, 'notes'),
    }
    // Si se pidió un CHG nuevo, se crea primero.
    const change = await resolveChange(data)
    if (activity) {
      await update.mutateAsync({ ...fields, change })
      return
    }
    const created = await create.mutateAsync({ ...fields, change: change || undefined })
    if (onCreated) onCreated(created)
    else void navigate(`/planificacion/actividades/${created.code}`)
  }

  return (
    <Form label={activity ? 'Editar actividad' : 'Nueva actividad'} submitLabel={activity ? 'Guardar' : 'Crear actividad'} onSubmit={submit} validate={validate} onDone={onClose} onCancel={onClose}>
      <div className="form__row">
        <SelectField name="kind" label="Clase" required defaultValue={activity?.kind ?? defaults?.kind ?? 'certification'}>
          {options(ACTIVITY_KIND_LABELS).map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </SelectField>
        <ChangeField
          label="CHG"
          emptyLabel="Sin CHG"
          askSystem
          defaultValue={activity?.change?.code ?? defaults?.change}
          hint="Obligatorio en la certificación. Si el CHG aún no existe, créalo aquí mismo."
        />
      </div>
      <TextField name="title" label="Actividad" required maxLength={200} defaultValue={activity?.title ?? defaults?.title} />
      <div className="form__row">
        <TextField name="startDate" label="Desde" type="date" required defaultValue={activity?.startDate} />
        <TextField name="endDate" label="Hasta" type="date" required defaultValue={activity?.endDate} hint="Incluido. Como máximo un año." />
      </div>
      <CheckGroupField
        name="operators"
        label="Operadores"
        choices={users.map((user) => ({ value: user.id, label: user.displayName }))}
        selected={activity?.operators.map((operator) => operator.id)}
      />
      <TextAreaField name="notes" label="Notas" defaultValue={activity?.notes} />
    </Form>
  )
}
