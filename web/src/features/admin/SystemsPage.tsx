// Catálogo de sistemas bajo prueba: alta y edición. La clave no cambia y un sistema no se borra,
// porque los CHG, REQ, TC y ambientes lo referencian.
import { useState } from 'react'
import { useCan, useSystems } from '../../shared/api/session.ts'
import type { System, SystemRoute } from '../../shared/api/types.ts'
import { ROUTE_LABELS } from '../../shared/lib/labels.ts'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { Form, SelectField, TextField, text } from '../../shared/ui/Form.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { useCreateSystem, useUpdateSystem } from './api.ts'

const SYSTEM_ROUTES: SystemRoute[] = ['high', 'light']

function SystemForm({ system, onClose }: { system?: System; onClose: () => void }) {
  const create = useCreateSystem()
  const update = useUpdateSystem(system?.key ?? '')

  async function submit(data: FormData) {
    const common = { name: text(data, 'name'), defaultRoute: text(data, 'defaultRoute') as SystemRoute }
    if (system) {
      await update.mutateAsync(common)
      return
    }
    await create.mutateAsync({ ...common, key: text(data, 'key') })
  }

  return (
    <Form label={system ? `Editar ${system.name}` : 'Nuevo sistema'} submitLabel={system ? 'Guardar' : 'Agregar sistema'} onSubmit={submit} onDone={onClose} onCancel={onClose}>
      <div className="form__row">
        {!system && (
          <TextField
            name="key"
            label="Clave"
            required
            maxLength={12}
            placeholder="SYSAF"
            hint="De 2 a 12 letras o dígitos, sin espacios. Se guarda en mayúsculas y no se cambia después."
          />
        )}
        <TextField name="name" label="Nombre" required maxLength={80} defaultValue={system?.name} placeholder="SysAF" />
      </div>
      <SelectField
        name="defaultRoute"
        label="Ruta por omisión"
        required
        defaultValue={system?.defaultRoute ?? ''}
        hint="La de los REQ del sistema que no tienen un CHG."
      >
        <option value="">Elige la ruta</option>
        {SYSTEM_ROUTES.map((route) => (
          <option key={route} value={route}>
            {ROUTE_LABELS[route]}
          </option>
        ))}
      </SelectField>
    </Form>
  )
}

export default function SystemsPage() {
  const can = useCan()
  const editable = can('admin.configure')
  const systems = useSystems()
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<System | null>(null)

  return (
    <>
      <Crumbs items={[{ label: 'Administración' }, { label: 'Sistemas' }]} />
      <PageHead
        title="Sistemas"
        subtitle="Los sistemas bajo prueba. Cada CHG, REQ, TC y ambiente pertenece a uno."
        actions={
          editable && (
            <button className="btn btn--primary" type="button" onClick={() => setCreating(true)}>
              <Icon name="plus" />
              Nuevo sistema
            </button>
          )
        }
      />
      <Async query={systems}>
        {(data) =>
          data.items.length === 0 ? (
            <Empty>No hay sistemas.</Empty>
          ) : (
            <div className="surface table-wrap">
              <table className="table" aria-label="Sistemas">
                <thead>
                  <tr>
                    <th className="col-fixed">Clave</th>
                    <th>Nombre</th>
                    <th>Ruta por omisión</th>
                    {editable && <th className="col-actions">Acciones</th>}
                  </tr>
                </thead>
                <tbody>
                  {data.items.map((system) => (
                    <tr key={system.key}>
                      <td className="col-fixed mono">{system.key}</td>
                      <td>{system.name}</td>
                      <td>{ROUTE_LABELS[system.defaultRoute]}</td>
                      {editable && (
                        <td className="col-actions">
                          <button className="btn btn--icon" type="button" title="Editar sistema" aria-label={`Editar ${system.name}`} onClick={() => setEditing(system)}>
                            <Icon name="pencil" />
                          </button>
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        }
      </Async>
      {creating && (
        <Dialog title="Nuevo sistema" onClose={() => setCreating(false)}>
          <SystemForm onClose={() => setCreating(false)} />
        </Dialog>
      )}
      {editing && (
        <Dialog title={`Editar ${editing.name}`} onClose={() => setEditing(null)}>
          <SystemForm system={editing} onClose={() => setEditing(null)} />
        </Dialog>
      )}
    </>
  )
}
