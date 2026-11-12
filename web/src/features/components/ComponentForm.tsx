// Alta y edición de un componente (COMP). El sistema no se cambia después de crear.
import { useSystems } from '../../shared/api/session.ts'
import type { Component, ComponentKind } from '../../shared/api/types.ts'
import { COMPONENT_KIND_LABELS, options } from '../../shared/lib/labels.ts'
import { Form, SelectField, TextAreaField, TextField, text } from '../../shared/ui/Form.tsx'
import { useCreateComponent, useUpdateComponent } from './api.ts'

export function ComponentForm({ component, onClose }: { component?: Component; onClose: () => void }) {
  const systems = useSystems().data?.items ?? []
  const create = useCreateComponent()
  const update = useUpdateComponent()

  function submit(data: FormData) {
    const common = { kind: text(data, 'kind') as ComponentKind, name: text(data, 'name'), description: text(data, 'description') }
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
        <SelectField name="kind" label="Clase" required defaultValue={component?.kind ?? ''}>
          <option value="">Elige la clase</option>
          {options(COMPONENT_KIND_LABELS).map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </SelectField>
      </div>
      <TextAreaField name="description" label="Descripción" defaultValue={component?.description} />
    </Form>
  )
}
