// Alta y edición de un CHG. El sistema no se cambia después de crear.
import { useNavigate } from 'react-router'
import { useSystems } from '../../shared/api/session.ts'
import type { Change, Route } from '../../shared/api/types.ts'
import { fromLocalInput, toLocalInput } from '../../shared/lib/format.ts'
import { ROUTE_LABELS, options } from '../../shared/lib/labels.ts'
import { Form, SelectField, TextAreaField, TextField, text } from '../../shared/ui/Form.tsx'
import { useCreateChange, useUpdateChange } from './api.ts'

function localValue(iso: string | undefined): string {
  return iso ? toLocalInput(new Date(iso)) : ''
}

export function ChangeForm({ change, onClose }: { change?: Change; onClose: () => void }) {
  const navigate = useNavigate()
  const systems = useSystems().data?.items ?? []
  const create = useCreateChange()
  const update = useUpdateChange(change?.code ?? '')

  async function submit(data: FormData) {
    const common = { title: text(data, 'title'), description: text(data, 'description'), route: text(data, 'route') as Route }
    if (change) {
      await update.mutateAsync({ ...common, windowStart: fromLocalInput(text(data, 'windowStart')) ?? null, windowEnd: fromLocalInput(text(data, 'windowEnd')) ?? null })
      return
    }
    const created = await create.mutateAsync({
      ...common,
      system: text(data, 'system'),
      windowStart: fromLocalInput(text(data, 'windowStart')),
      windowEnd: fromLocalInput(text(data, 'windowEnd')),
    })
    void navigate(`/cambios/${created.code}`)
  }

  return (
    <Form label={change ? 'Editar cambio' : 'Nuevo cambio'} submitLabel={change ? 'Guardar' : 'Crear CHG'} onSubmit={submit} onDone={onClose} onCancel={onClose}>
      <TextField name="title" label="Título" required defaultValue={change?.title} />
      <div className="form__row">
        {!change && (
          <SelectField name="system" label="Sistema" required defaultValue="">
            <option value="">Elige un sistema</option>
            {systems.map((system) => (
              <option key={system.key} value={system.key}>
                {system.name}
              </option>
            ))}
          </SelectField>
        )}
        <SelectField name="route" label="Ruta" required defaultValue={change?.route ?? 'light'} hint="Alta para cambios mayores; ligera para el resto.">
          {options(ROUTE_LABELS).map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </SelectField>
      </div>
      <TextAreaField name="description" label="Descripción" defaultValue={change?.description} />
      <div className="form__row">
        <TextField name="windowStart" label="Inicio de la ventana" type="datetime-local" defaultValue={localValue(change?.windowStart)} />
        <TextField name="windowEnd" label="Fin de la ventana" type="datetime-local" defaultValue={localValue(change?.windowEnd)} />
      </div>
    </Form>
  )
}
