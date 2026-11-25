// Sesiones exploratorias (EXP) de un REQ. Un hallazgo no es un defecto hasta reproducirlo en un RUN.
import { useRef, useState } from 'react'
import { fromLocalInput, formatDateTime, toLocalInput } from '../../shared/lib/format.ts'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { Form, TextAreaField, TextField, text } from '../../shared/ui/Form.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { RunTarget } from '../../shared/ui/RunTarget.tsx'
import { useCreateExploration, useRequirementExplorations } from './api.ts'
import { TargetFields, resolveBuild, validateNewBuild, type Registered } from './Target.tsx'

function ExplorationForm({ requirementCode, systemKey, onClose }: { requirementCode: string; systemKey: string; onClose: () => void }) {
  const create = useCreateExploration()
  const registered = useRef<Registered>(new Map())
  return (
    <Form
      label="Registrar exploración"
      submitLabel="Registrar EXP"
      validate={validateNewBuild}
      onSubmit={async (data) =>
        create.mutateAsync({
          requirement: requirementCode,
          build: await resolveBuild(data, { system: systemKey, requirement: requirementCode, registered: registered.current }),
          environment: text(data, 'environment'),
          manualChanges: text(data, 'manualChanges'),
          findings: text(data, 'findings'),
          executedAt: fromLocalInput(text(data, 'executedAt')) ?? '',
        })
      }
      onDone={onClose}
      onCancel={onClose}
    >
      <TargetFields systemKey={systemKey} />
      <TextField name="executedAt" label="Fecha de la sesión" type="datetime-local" required defaultValue={toLocalInput(new Date())} />
      <TextAreaField name="manualChanges" label="Cambios manuales en el ambiente" hint="Lo que se tocó a mano y lo aparta de la línea base." />
      <TextAreaField name="findings" label="Hallazgos" />
    </Form>
  )
}

/** `systemKey` es el sistema del REQ: de ahí salen los BUILD y los ambientes que se ofrecen. */
export function Explorations({ requirementCode, systemKey, canCreate }: { requirementCode: string; systemKey: string; canCreate: boolean }) {
  const explorations = useRequirementExplorations(requirementCode)
  const [creating, setCreating] = useState(false)
  return (
    <section className="stack" aria-labelledby="h-exp">
      <div className="row row--between">
        <div>
          <h3 className="h2" id="h-exp">
            Exploraciones
          </h3>
          <p className="subtitle">No cuentan para la cobertura. Un hallazgo se abre como DEF solo después de reproducirlo en un RUN.</p>
        </div>
        {canCreate && (
          <button className="btn" type="button" onClick={() => setCreating(true)}>
            <Icon name="plus" />
            Registrar EXP
          </button>
        )}
      </div>
      <Async query={explorations}>
        {(data) =>
          data.items.length === 0 ? (
            <Empty>Sin sesiones exploratorias.</Empty>
          ) : (
            <div className="surface table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>EXP</th>
                    <th>BUILD · ENV</th>
                    <th>Cambios manuales</th>
                    <th>Hallazgos</th>
                    <th>Ejecutó</th>
                  </tr>
                </thead>
                <tbody>
                  {data.items.map((exploration) => (
                    <tr key={exploration.code}>
                      <td>
                        <Code code={exploration.code} />
                      </td>
                      <td>
                        <RunTarget target={exploration} />
                      </td>
                      <td className="wrap">{exploration.manualChanges || <span className="muted">Ninguno</span>}</td>
                      <td className="wrap">{exploration.findings || <span className="muted">Ninguno</span>}</td>
                      <td>
                        {exploration.executedBy.displayName} · {formatDateTime(exploration.executedAt)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        }
      </Async>
      {creating && (
        <Dialog title="Registrar exploración" onClose={() => setCreating(false)}>
          <ExplorationForm requirementCode={requirementCode} systemKey={systemKey} onClose={() => setCreating(false)} />
        </Dialog>
      )}
    </section>
  )
}
