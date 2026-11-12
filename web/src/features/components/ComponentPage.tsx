// Ficha de un componente (COMP).
import { useState } from 'react'
import { useParams } from 'react-router'
import { useCan } from '../../shared/api/session.ts'
import { formatDate } from '../../shared/lib/format.ts'
import { COMPONENT_KIND_LABELS } from '../../shared/lib/labels.ts'
import { RetiredChip } from '../../shared/ui/chips.tsx'
import { Async, ErrorNotice, PageSkeleton } from '../../shared/ui/feedback.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, MetaItem } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { useComponent, useUpdateComponent } from './api.ts'
import { ComponentForm } from './ComponentForm.tsx'

export default function ComponentPage() {
  const { code = '' } = useParams()
  const canManage = useCan()('comp.manage')
  const component = useComponent(code)
  const update = useUpdateComponent()
  const [editing, setEditing] = useState(false)

  return (
    <>
      <Crumbs items={[{ label: 'Componentes', to: '/pruebas/componentes' }, { label: code, code: true }]} />
      <Async query={component} skeleton={<PageSkeleton />}>
        {(data) => (
          <>
            <section className="glass pad stack" aria-labelledby="comp-title">
              <div className="row row--between">
                <div className="stack-s">
                  <span className="id">{data.code}</span>
                  <h1 className="title" id="comp-title">
                    {data.name}
                  </h1>
                </div>
                <RetiredChip retired={data.retired} />
              </div>
              <dl className="meta">
                <MetaItem label="Sistema">{data.system.name}</MetaItem>
                <MetaItem label="Clase">{COMPONENT_KIND_LABELS[data.kind]}</MetaItem>
                <MetaItem label="Registrado por">
                  {data.createdBy.displayName} · {formatDate(data.createdAt)}
                </MetaItem>
              </dl>
              {canManage && (
                <div className="row">
                  <button className="btn" type="button" onClick={() => setEditing(true)}>
                    <Icon name="pencil" />
                    Editar
                  </button>
                  <button className="btn" type="button" disabled={update.isPending} onClick={() => update.mutate({ code: data.code, patch: { retired: !data.retired } })}>
                    {data.retired ? 'Reactivar' : 'Retirar'}
                  </button>
                </div>
              )}
              {update.error && <ErrorNotice error={update.error} />}
            </section>
            <section className="surface pad stack">
              <h2 className="h3">Descripción</h2>
              <p className="wrap">{data.description || <span className="muted">Sin descripción.</span>}</p>
              {data.retired && <p className="soft">Está retirado: ya no se ofrece para declarar el impacto de un REQ o de un TC.</p>}
            </section>
            {editing && (
              <Dialog title={`Editar ${data.code}`} onClose={() => setEditing(false)}>
                <ComponentForm component={data} onClose={() => setEditing(false)} />
              </Dialog>
            )}
          </>
        )}
      </Async>
    </>
  )
}
