// Ficha de una actividad (ACT): fechas, CHG, operadores, sus solapes y la auditoría. Reajustar es
// editarla o cancelarla; nada se borra. Junta planificación y auditoría, por eso vive en app/.
import { useState } from 'react'
import { useParams } from 'react-router'
import { ActivityStatusChip, KindChip } from '../features/planning/ActivityChips.tsx'
import { ActivityForm } from '../features/planning/ActivityForm.tsx'
import { useActivity, useUpdateActivity } from '../features/planning/api.ts'
import { lengthText, spanText } from '../features/planning/days.ts'
import { OverlapList } from '../features/planning/Overlaps.tsx'
import { useCan, useUsers } from '../shared/api/session.ts'
import type { Activity as ActivityData } from '../shared/api/types.ts'
import { formatDate, formatDateTime } from '../shared/lib/format.ts'
import { ACTIVITY_KIND_LABELS } from '../shared/lib/labels.ts'
import { Code } from '../shared/ui/Code.tsx'
import { Async, PageSkeleton } from '../shared/ui/feedback.tsx'
import { Icon } from '../shared/ui/Icon.tsx'
import { Crumbs, MetaItem } from '../shared/ui/layout.tsx'
import { Dialog } from '../shared/ui/Modal.tsx'
import { Activity } from './Activity.tsx'

function CancelButton({ activity }: { activity: ActivityData }) {
  const update = useUpdateActivity(activity.code)
  const cancelled = activity.status === 'cancelled'
  return (
    <button className="btn" type="button" disabled={update.isPending} onClick={() => update.mutate({ cancelled: !cancelled })}>
      <Icon name={cancelled ? 'check' : 'x'} />
      {cancelled ? 'Reactivar' : 'Cancelar actividad'}
    </button>
  )
}

function ActivityView({ activity }: { activity: ActivityData }) {
  const can = useCan()
  const users = useUsers().data?.items ?? []
  const [editing, setEditing] = useState(false)
  const canManage = can('activity.manage')

  // La auditoría guarda ids de operadores y claves: aquí se leen con su nombre.
  const value = (field: string | undefined, raw: string | undefined) => {
    if (raw === undefined) return raw
    if (field === 'operators') return raw ? raw.split(',').map((id) => users.find((user) => user.id === id)?.displayName ?? id).join(', ') : 'ninguno'
    if (field === 'cancelled') return raw === 'true' ? 'cancelada' : 'vigente'
    if (field === 'kind') return ACTIVITY_KIND_LABELS[raw as keyof typeof ACTIVITY_KIND_LABELS] ?? raw
    return raw || 'nada'
  }

  return (
    <>
      <section className="glass pad stack" aria-labelledby="act-title">
        <div className="row row--between">
          <div className="stack-s">
            <span className="id">{activity.code}</span>
            <h1 className="title" id="act-title">
              {activity.title}
            </h1>
            <span className="row">
              <KindChip kind={activity.kind} />
              <ActivityStatusChip status={activity.status} />
            </span>
          </div>
          {canManage && (
            <span className="row">
              <CancelButton activity={activity} />
              <button className="btn" type="button" onClick={() => setEditing(true)}>
                <Icon name="pencil" />
                Editar
              </button>
            </span>
          )}
        </div>
        <dl className="meta">
          <MetaItem label="Fechas">
            {spanText(activity.startDate, activity.endDate)} · {lengthText(activity.startDate, activity.endDate)}
          </MetaItem>
          <MetaItem label="CHG">
            {activity.change ? (
              <span className="cell-line">
                <Code code={activity.change.code} /> {activity.change.title} · {activity.change.system}
              </span>
            ) : (
              'Ninguno'
            )}
          </MetaItem>
          <MetaItem label="Operadores">
            {activity.operators.length > 0 ? activity.operators.map((operator) => operator.displayName).join(', ') : <span className="muted">Nadie asignado</span>}
          </MetaItem>
          <MetaItem label="Registrada por">
            {activity.createdBy.displayName} · {formatDate(activity.createdAt)}
          </MetaItem>
          {activity.cancelledAt && <MetaItem label="Cancelada">{formatDateTime(activity.cancelledAt)}</MetaItem>}
        </dl>
      </section>
      {activity.status !== 'cancelled' && activity.operators.length === 0 && (
        <div className="notice" role="status">
          <Icon name="user" />
          <div className="notice__main">
            Nadie tiene asignada esta actividad.
            {activity.kind === 'certification' && ' Mientras no tenga operadores, pasar a pruebas los REQ de su CHG muestra un aviso.'}
          </div>
          {canManage && (
            <button className="btn" type="button" onClick={() => setEditing(true)}>
              Asignar operadores
            </button>
          )}
        </div>
      )}
      <OverlapList overlaps={activity.overlaps} label="Solapes de la actividad" />
      {activity.status !== 'cancelled' && activity.overlaps.length === 0 && activity.operators.length > 0 && (
        <p className="row soft">
          <Icon name="check" />
          Ninguno de sus operadores tiene otra actividad esos días.
        </p>
      )}
      <section className="surface pad stack">
        <h2 className="h3">Notas</h2>
        <p className="wrap">{activity.notes || <span className="muted">Sin notas.</span>}</p>
      </section>
      <section className="stack" aria-labelledby="act-history">
        <h2 className="h2" id="act-history">
          Actividad
        </h2>
        <Activity entityType="activity" entityCode={activity.code} value={value} />
      </section>
      {editing && (
        <Dialog title={`Editar ${activity.code}`} onClose={() => setEditing(false)}>
          <ActivityForm activity={activity} onClose={() => setEditing(false)} />
        </Dialog>
      )}
    </>
  )
}

export default function ActivityPage() {
  const { code = '' } = useParams()
  const activity = useActivity(code)
  return (
    <>
      <Crumbs items={[{ label: 'Planificación', to: '/planificacion' }, { label: 'Actividades', to: '/planificacion/actividades' }, { label: code, code: true }]} />
      <Async query={activity} skeleton={<PageSkeleton />}>
        {(data) => <ActivityView activity={data} />}
      </Async>
    </>
  )
}
