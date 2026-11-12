// Botones de transición de un REQ, CHG o DEF: salen del flujo configurado y de los permisos.
import { useState } from 'react'
import { ApiError, toApiError } from '../api/errors.ts'
import type { TransitionInput, Workflow, WorkflowTransition } from '../api/types.ts'
import { ErrorNotice } from './feedback.tsx'
import { Form, TextAreaField, text } from './Form.tsx'
import { Icon } from './Icon.tsx'
import { Dialog } from './Modal.tsx'

interface TransitionsProps {
  workflow: Workflow | undefined
  stateKey: string
  can: (permission: string) => boolean
  /** Destinos con condiciones que bloquean: el botón lo avisa; decide el servidor. */
  blocked?: string[]
  onTransition: (input: TransitionInput) => Promise<unknown>
}

export function Transitions({ workflow, stateKey, can, blocked = [], onTransition }: TransitionsProps) {
  const [asking, setAsking] = useState<WorkflowTransition | null>(null)
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<ApiError | null>(null)
  if (!workflow) return null

  const stateOf = (key: string) => workflow.states.find((state) => state.key === key)
  const available = workflow.transitions.filter((transition) => transition.from === stateKey && can(transition.permission))
  if (available.length === 0) return null
  // El paso principal es el que sigue en la línea de vida.
  const main = available.find((transition) => stateOf(transition.to)?.position !== undefined)

  async function run(transition: WorkflowTransition) {
    setPending(true)
    try {
      await onTransition({ to: transition.to })
      setError(null)
    } catch (caught) {
      // Un 409 INVALID_ACTION trae en su detalle qué falta para avanzar.
      setError(toApiError(caught))
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="stack-s">
      <div className="row" role="group" aria-label="Cambiar de estado">
        {available.map((transition) => {
          const isBlocked = blocked.includes(transition.to)
          return (
            <button
              key={transition.to}
              className={transition === main ? 'btn btn--primary' : 'btn'}
              type="button"
              disabled={pending}
              title={isBlocked ? 'Tiene condiciones pendientes: mira qué falta para avanzar' : undefined}
              onClick={() => (transition.requiresReason ? setAsking(transition) : void run(transition))}
            >
              {isBlocked && <Icon name="lock" />}
              Pasar a «{stateOf(transition.to)?.name ?? transition.to}»
            </button>
          )
        })}
      </div>
      {error && <ErrorNotice error={error} />}
      {asking && (
        <Dialog title={`Pasar a «${stateOf(asking.to)?.name ?? asking.to}»`} onClose={() => setAsking(null)}>
          <Form
            label="Motivo del cambio de estado"
            submitLabel="Cambiar estado"
            onSubmit={(data) => onTransition({ to: asking.to, reason: text(data, 'reason') })}
            onDone={() => setAsking(null)}
            onCancel={() => setAsking(null)}
          >
            <TextAreaField name="reason" label="Motivo" required hint="Queda en la auditoría." />
          </Form>
        </Dialog>
      )}
    </div>
  )
}
