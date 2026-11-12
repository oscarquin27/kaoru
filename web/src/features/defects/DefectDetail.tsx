// Ficha de un DEF: datos, tiempo en estado y transiciones (con motivo al diferir o reabrir).
import { useState, type ReactNode } from 'react'
import { useCan, useWorkflow } from '../../shared/api/session.ts'
import { formatDate, formatDaysLong } from '../../shared/lib/format.ts'
import { SeverityChip, StateChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, PageSkeleton } from '../../shared/ui/feedback.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, MetaItem } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { Transitions } from '../../shared/ui/Transitions.tsx'
import { useDefect, useTransitionDefect } from './api.ts'
import { DefectForm } from './DefectForm.tsx'

/** `activity` es la actividad (auditoría y comentarios), que arma `app/` con otros features. */
export function DefectDetail({ code, activity }: { code: string; activity: ReactNode }) {
  const can = useCan()
  const defect = useDefect(code)
  const workflow = useWorkflow('defect').data
  const transition = useTransitionDefect(code)
  const [editing, setEditing] = useState(false)

  return (
    <>
      <Crumbs
        items={[
          ...(defect.data ? [{ label: defect.data.requirement, to: `/requerimientos/${defect.data.requirement}#defectos`, code: true }] : [{ label: 'Defectos', to: '/defectos' }]),
          { label: code, code: true },
        ]}
      />
      <Async query={defect} skeleton={<PageSkeleton />}>
        {(data) => (
          <>
            <section className="glass pad stack" aria-labelledby="def-title">
              <div className="row row--between">
                <div className="stack-s">
                  <span className="id">{data.code}</span>
                  <h1 className="title" id="def-title">
                    {data.title}
                  </h1>
                </div>
                <div className="stack-s">
                  <span className="h3">Estado actual</span>
                  <StateChip state={data.state} current large />
                </div>
              </div>
              <dl className="meta">
                <MetaItem label="Severidad">
                  <SeverityChip severity={data.severity} />
                </MetaItem>
                <MetaItem label="En este estado">
                  <span className="num">{formatDaysLong(data.daysInState)}</span>
                </MetaItem>
                <MetaItem label="Requerimiento">
                  <Code code={data.requirement} />
                </MetaItem>
                <MetaItem label="RUN que lo reprodujo">
                  <Code code={data.run} />
                </MetaItem>
                <MetaItem label="Asignado a">{data.assignee?.displayName ?? <span className="muted">Sin asignar</span>}</MetaItem>
                <MetaItem label="Abierto por">
                  {data.createdBy.displayName} · {formatDate(data.createdAt)}
                </MetaItem>
              </dl>
              <div className="row row--between">
                <Transitions workflow={workflow} stateKey={data.state.key} can={can} onTransition={transition.mutateAsync} />
                {can('def.edit') && (
                  <button className="btn" type="button" onClick={() => setEditing(true)}>
                    <Icon name="pencil" />
                    Editar
                  </button>
                )}
              </div>
            </section>

            <section className="surface pad stack">
              <h2 className="h3">Descripción</h2>
              <p className="wrap">{data.description}</p>
            </section>

            <section className="stack" aria-labelledby="h-actividad">
              <h2 className="h2" id="h-actividad">
                Actividad
              </h2>
              {activity}
            </section>

            {editing && (
              <Dialog title={`Editar ${data.code}`} onClose={() => setEditing(false)}>
                <DefectForm defect={data} onClose={() => setEditing(false)} />
              </Dialog>
            )}
          </>
        )}
      </Async>
    </>
  )
}
