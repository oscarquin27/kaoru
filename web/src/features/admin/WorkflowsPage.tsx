// Estados y transiciones de cada flujo. Los cambios valen de inmediato y quedan en la auditoría.
import { useState } from 'react'
import { useCan, useWorkflow } from '../../shared/api/session.ts'
import type { Workflow, WorkflowState, WorkflowTransition } from '../../shared/api/types.ts'
import { CheckField, Form, SelectField, TextField, text } from '../../shared/ui/Form.tsx'
import { Async, ErrorNotice } from '../../shared/ui/feedback.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { type FlowType, useAddState, useRemoveState, useRemoveTransition, useRolePermissions, useSetTransition, useUpdateState } from './api.ts'

type Editing = { kind: 'new-state' } | { kind: 'state'; state: WorkflowState } | { kind: 'new-transition' } | null

/** Posición de un formulario: vacía es un estado lateral. */
function positionOf(data: FormData): number | undefined {
  const raw = text(data, 'position')
  return raw === '' ? undefined : Number(raw)
}

function StateForm({ entityType, state, onClose }: { entityType: FlowType; state?: WorkflowState; onClose: () => void }) {
  const add = useAddState(entityType)
  const update = useUpdateState(entityType)

  async function submit(data: FormData) {
    const position = positionOf(data)
    if (state) await update.mutateAsync({ key: state.key, patch: { name: text(data, 'name'), position: position ?? null } })
    else await add.mutateAsync({ key: text(data, 'key'), name: text(data, 'name'), position, isFinal: data.get('isFinal') === 'on' })
  }

  return (
    <Form label={state ? 'Editar estado' : 'Agregar estado'} submitLabel={state ? 'Guardar' : 'Agregar estado'} onSubmit={submit} onDone={onClose} onCancel={onClose}>
      <TextField name="name" label="Nombre" required maxLength={60} defaultValue={state?.name} />
      <div className="form__row">
        {!state && <TextField name="key" label="Clave" required placeholder="en_espera" hint="Minúsculas, números y guion bajo. No se cambia después." />}
        <TextField name="position" label="Posición en la línea de vida" type="number" min={1} defaultValue={state?.position} hint="Vacía: estado lateral, fuera de la línea." />
      </div>
      {!state && <CheckField name="isFinal" label="Es un estado final" />}
    </Form>
  )
}

function TransitionForm({ entityType, workflow, permissions, onClose }: { entityType: FlowType; workflow: Workflow; permissions: Permission[]; onClose: () => void }) {
  const save = useSetTransition(entityType)
  const stateOptions = workflow.states.map((state) => (
    <option key={state.key} value={state.key}>
      {state.name}
    </option>
  ))
  return (
    <Form
      label="Agregar transición"
      submitLabel="Agregar transición"
      onSubmit={(data) => save.mutateAsync({ from: text(data, 'from'), to: text(data, 'to'), permission: text(data, 'permission'), requiresReason: data.get('requiresReason') === 'on' })}
      onDone={onClose}
      onCancel={onClose}
    >
      <div className="form__row">
        <SelectField name="from" label="Desde" required defaultValue="">
          <option value="">Elige un estado</option>
          {stateOptions}
        </SelectField>
        <SelectField name="to" label="Hacia" required defaultValue="">
          <option value="">Elige un estado</option>
          {stateOptions}
        </SelectField>
      </div>
      <SelectField name="permission" label="Permiso que exige" required defaultValue="">
        <option value="">Elige un permiso</option>
        {permissions.map((permission) => (
          <option key={permission.key} value={permission.key}>
            {permission.description} ({permission.key})
          </option>
        ))}
      </SelectField>
      <CheckField name="requiresReason" label="Exige escribir un motivo" />
    </Form>
  )
}

interface Permission {
  key: string
  description: string
}

function Flow({ title, entityType, workflow, permissions, editable }: { title: string; entityType: FlowType; workflow: Workflow; permissions: Permission[]; editable: boolean }) {
  const [editing, setEditing] = useState<Editing>(null)
  const removeState = useRemoveState(entityType)
  const setTransition = useSetTransition(entityType)
  const removeTransition = useRemoveTransition(entityType)
  const busy = removeState.isPending || setTransition.isPending || removeTransition.isPending
  const error = removeState.error ?? setTransition.error ?? removeTransition.error
  const stateName = (key: string) => workflow.states.find((state) => state.key === key)?.name ?? key
  const used = (key: string) => workflow.transitions.some((transition) => transition.from === key || transition.to === key)
  const close = () => setEditing(null)

  function change(transition: WorkflowTransition, patch: { permission?: string; requiresReason?: boolean }) {
    setTransition.mutate({ from: transition.from, to: transition.to, permission: transition.permission, requiresReason: transition.requiresReason, ...patch })
  }

  return (
    <section className="stack" aria-label={title}>
      <div className="row">
        <h2 className="h2 grow">{title}</h2>
        {editable && (
          <>
            <button className="btn" type="button" onClick={() => setEditing({ kind: 'new-state' })}>
              Agregar estado
            </button>
            <button className="btn" type="button" onClick={() => setEditing({ kind: 'new-transition' })}>
              Agregar transición
            </button>
          </>
        )}
      </div>
      {error && <ErrorNotice error={error} />}
      <div className="surface table-wrap">
        <table className="table" aria-label={`Estados de ${title}`}>
          <thead>
            <tr>
              <th>Estado</th>
              <th>Clave</th>
              <th>Posición</th>
              <th>Tipo</th>
              {editable && <th>Acciones</th>}
            </tr>
          </thead>
          <tbody>
            {workflow.states.map((state) => (
              <tr key={state.key}>
                <td>{state.name}</td>
                <td className="mono">{state.key}</td>
                <td className="num">{state.position ?? <span className="muted">Lateral</span>}</td>
                <td>{state.isInitial ? 'Inicial' : state.isFinal ? 'Final' : <span className="muted">—</span>}</td>
                {editable && (
                  <td>
                    <span className="row">
                      <button className="btn" type="button" aria-label={`Editar ${state.name}`} onClick={() => setEditing({ kind: 'state', state })}>
                        Editar
                      </button>
                      <button
                        className="btn"
                        type="button"
                        aria-label={`Quitar ${state.name}`}
                        disabled={busy || state.isInitial || used(state.key)}
                        title={state.isInitial ? 'El estado inicial no se quita' : used(state.key) ? 'Quita primero sus transiciones' : undefined}
                        onClick={() => removeState.mutate(state.key)}
                      >
                        Quitar
                      </button>
                    </span>
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="surface table-wrap">
        <table className="table" aria-label={`Transiciones de ${title}`}>
          <thead>
            <tr>
              <th>Transición</th>
              <th>Permiso que exige</th>
              <th>Exige motivo</th>
              <th>Reglas</th>
              {editable && <th>Acciones</th>}
            </tr>
          </thead>
          <tbody>
            {workflow.transitions.map((transition) => {
              const label = `${stateName(transition.from)} → ${stateName(transition.to)}`
              return (
                <tr key={`${transition.from}>${transition.to}`}>
                  <td>{label}</td>
                  <td>
                    {editable ? (
                      <select className="select" aria-label={`Permiso de ${label}`} value={transition.permission} disabled={busy} onChange={(event) => change(transition, { permission: event.target.value })}>
                        {permissions.map((permission) => (
                          <option key={permission.key} value={permission.key}>
                            {permission.description}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <span className="mono">{transition.permission}</span>
                    )}
                  </td>
                  <td>
                    <label className="check">
                      <input
                        type="checkbox"
                        aria-label={`Exige motivo: ${label}`}
                        checked={transition.requiresReason}
                        disabled={!editable || busy}
                        onChange={(event) => change(transition, { requiresReason: event.target.checked })}
                      />
                    </label>
                  </td>
                  <td className="num">{transition.rules.length || <span className="muted">—</span>}</td>
                  {editable && (
                    <td>
                      <button
                        className="btn"
                        type="button"
                        aria-label={`Quitar ${label}`}
                        disabled={busy}
                        title={transition.rules.length > 0 ? 'También se quitan sus reglas de bloqueo' : undefined}
                        onClick={() => removeTransition.mutate({ from: transition.from, to: transition.to })}
                      >
                        Quitar
                      </button>
                    </td>
                  )}
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      {editing?.kind === 'new-state' && (
        <Dialog title={`Agregar estado · ${title}`} onClose={close}>
          <StateForm entityType={entityType} onClose={close} />
        </Dialog>
      )}
      {editing?.kind === 'state' && (
        <Dialog title={`Editar estado · ${editing.state.name}`} onClose={close}>
          <StateForm entityType={entityType} state={editing.state} onClose={close} />
        </Dialog>
      )}
      {editing?.kind === 'new-transition' && (
        <Dialog title={`Agregar transición · ${title}`} onClose={close}>
          <TransitionForm entityType={entityType} workflow={workflow} permissions={permissions} onClose={close} />
        </Dialog>
      )}
    </section>
  )
}

const FLOWS: { entityType: FlowType; title: string }[] = [
  { entityType: 'requirement', title: 'Requerimientos' },
  { entityType: 'defect', title: 'Defectos' },
  { entityType: 'change', title: 'Cambios' },
]

function FlowSection({ entityType, title, permissions, editable }: { entityType: FlowType; title: string; permissions: Permission[]; editable: boolean }) {
  const workflow = useWorkflow(entityType)
  return <Async query={workflow}>{(data) => <Flow title={title} entityType={entityType} workflow={data} permissions={permissions} editable={editable} />}</Async>
}

export default function WorkflowsPage() {
  const editable = useCan()('admin.configure')
  // La lista de permisos solo la ve quien configura; sin ella, la pantalla es de solo lectura.
  const permissions = useRolePermissions(editable).data?.permissions ?? []
  return (
    <>
      <Crumbs items={[{ label: 'Administración', to: '/administracion' }, { label: 'Flujos' }]} />
      <PageHead title="Flujos" subtitle="Estados y transiciones de cada ciclo de vida. Un cambio vale de inmediato y queda en la auditoría." />
      {FLOWS.map((item) => (
        <FlowSection key={item.entityType} {...item} permissions={permissions} editable={editable && permissions.length > 0} />
      ))}
    </>
  )
}
