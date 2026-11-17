// Alta y edición de un componente (COMP). El sistema no se cambia después de crear.
// La clase se elige de las sugeridas o se escribe una nueva: desde ahí también se sugiere.
import { useId } from 'react'
import { useComponentOptions } from '../../shared/api/catalog.ts'
import { useSystems } from '../../shared/api/session.ts'
import type { Component } from '../../shared/api/types.ts'
import { componentKindFromText, componentKindLabel, componentKindOptions } from '../../shared/lib/labels.ts'
import { Form, SelectField, TextAreaField, TextField, text } from '../../shared/ui/Form.tsx'
import { useCreateComponent, useUpdateComponent } from './api.ts'

export function ComponentForm({ component, onClose }: { component?: Component; onClose: () => void }) {
  const systems = useSystems().data?.items ?? []
  const kinds = componentKindOptions((useComponentOptions().data?.items ?? []).map((item) => item.kind))
  const kindList = useId()
  const create = useCreateComponent()
  const update = useUpdateComponent()

  function submit(data: FormData) {
    const common = { kind: componentKindFromText(text(data, 'kind')), name: text(data, 'name'), description: text(data, 'description') }
    if (component) return update.mutateAsync({ code: component.code, patch: common })
    return create.mutateAsync({ ...common, system: text(data, 'system') })
  }

  return (
    <Form label={component ? 'Editar componente' : 'Nuevo componente'} submitLabel={component ? 'Guardar' : 'Agregar COMP'} onSubmit={submit} onDone={onClose} onCancel={onClose}>
      <TextField name="name" label="Nombre" required maxLength={120} defaultValue={component?.name} hint="No se repite dentro de un sistema." />
      <div className="form__row">
        {!component && (
          <SelectField name="system" label="Sistema" required defaultValue="">
            <option value="">Elige un sistema</option>
            {systems.map((system) => (
              <option key={system.key} value={system.key}>
                {system.name}
              </option>
            ))}
          </SelectField>
        )}
        <TextField
          name="kind"
          label="Clase"
          required
          maxLength={60}
          list={kindList}
          autoComplete="off"
          placeholder="Microservicio, DLL, Servicio de Windows…"
          defaultValue={component ? componentKindLabel(component.kind) : ''}
          hint="Elige una de la lista o escribe una nueva."
        />
        <datalist id={kindList}>
          {kinds.map((kind) => (
            <option key={kind.value} value={kind.label} />
          ))}
        </datalist>
      </div>
      <TextAreaField name="description" label="Descripción" defaultValue={component?.description} />
    </Form>
  )
}
