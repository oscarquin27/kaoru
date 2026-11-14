// Ficha de un CHG: datos, transiciones según flujo y permisos, y sus requerimientos.
import { useState, type ReactNode } from 'react'
import { useParams } from 'react-router'
import { useCan, useWorkflow } from '../../shared/api/session.ts'
import type { Change } from '../../shared/api/types.ts'
import { formatDate, formatDateTime } from '../../shared/lib/format.ts'
import { ROUTE_LABELS } from '../../shared/lib/labels.ts'
import { StateChip } from '../../shared/ui/chips.tsx'
import { Async, PageSkeleton } from '../../shared/ui/feedback.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, MetaItem } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { RequirementsTable } from '../../shared/ui/RequirementsTable.tsx'
import { Transitions } from '../../shared/ui/Transitions.tsx'
import { useChange, useChangeRequirements, useTransitionChange } from './api.ts'
import { ChangeForm } from './ChangeForm.tsx'

/** `planning` agrega la sección de certificación; la arma app/ porque vive en otro feature. */
export default function ChangePage({ planning }: { planning?: (change: Change) => ReactNode }) {
  const { code = '' } = useParams()
  const can = useCan()
  const change = useChange(code)
  const requirements = useChangeRequirements(code)
  const workflow = useWorkflow('change').data
  const transition = useTransitionChange(code)
  const [editing, setEditing] = useState(false)

  return (
    <>
      <Crumbs items={[{ label: 'Cambios', to: '/cambios' }, { label: code, code: true }]} />
      <Async query={change} skeleton={<PageSkeleton />}>
        {(data) => (
          <>
            <section className="glass pad stack" aria-labelledby="chg-title">
              <div className="row row--between">
                <div className="stack-s">
                  <span className="id">{data.code}</span>
                  <h1 className="title" id="chg-title">
                    {data.title}
                  </h1>
                </div>
                <div className="stack-s">
                  <span className="h3">Estado actual</span>
                  <StateChip state={data.state} current large />
                </div>
              </div>
              <dl className="meta">
                <MetaItem label="Sistema">{data.system.name}</MetaItem>
                <MetaItem label="Ruta">{ROUTE_LABELS[data.route]}</MetaItem>
                <MetaItem label="Ventana de cambio">
                  {data.windowStart ? `${formatDateTime(data.windowStart)}${data.windowEnd ? ` a ${formatDateTime(data.windowEnd)}` : ''}` : 'Por definir'}
                </MetaItem>
                <MetaItem label="Registrado por">
                  {data.createdBy.displayName} · {formatDate(data.createdAt)}
                </MetaItem>
              </dl>
              <div className="row row--between">
                <Transitions workflow={workflow} stateKey={data.state.key} can={can} onTransition={transition.mutateAsync} />
                {can('chg.edit') && (
                  <button className="btn" type="button" onClick={() => setEditing(true)}>
                    <Icon name="pencil" />
                    Editar
                  </button>
                )}
              </div>
            </section>

            <section className="surface pad stack">
              <h2 className="h3">Descripción</h2>
              <p className="wrap">{data.description || <span className="muted">Sin descripción.</span>}</p>
            </section>

            {planning?.(data)}

            <section className="stack" aria-labelledby="h-reqs">
              <h2 className="h2" id="h-reqs">
                Requerimientos del cambio
              </h2>
              <Async query={requirements}>{(page) => <RequirementsTable items={page.items} />}</Async>
            </section>

            {editing && (
              <Dialog title={`Editar ${data.code}`} onClose={() => setEditing(false)}>
                <ChangeForm change={data} onClose={() => setEditing(false)} />
              </Dialog>
            )}
          </>
        )}
      </Async>
    </>
  )
}
