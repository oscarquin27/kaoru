// Calendario: las alertas de atraso arriba y, debajo, hitos y ventanas de cambio en una sola lista
// por fecha. Las ventanas se leen de los CHG vigentes; los hitos los carga change management.
import { useState } from 'react'
import { useCan } from '../../shared/api/session.ts'
import type { CalendarWindow, Milestone } from '../../shared/api/types.ts'
import { formatDateTime, formatDay } from '../../shared/lib/format.ts'
import { ROUTE_LABELS } from '../../shared/lib/labels.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Chip, StateChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterCheck } from '../../shared/ui/Filters.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { useCalendar, useMilestones, useUpdateMilestone } from './api.ts'
import { LateAlerts, lateDays } from './LateAlerts.tsx'
import { MilestoneForm } from './MilestoneForm.tsx'

// El calendario mira la primera página grande de hitos, no todo el historial.
const MILESTONES = { size: 200 }

type Entry = { kind: 'window'; at: string; window: CalendarWindow } | { kind: 'milestone'; at: string; milestone: Milestone }

function MilestoneStatus({ milestone }: { milestone: Milestone }) {
  if (milestone.completed) {
    return (
      <Chip tone="ok" icon="check">
        Cumplido
      </Chip>
    )
  }
  if (milestone.daysLate > 0) {
    return (
      <Chip tone="bloq" icon="clock">
        {lateDays(milestone.daysLate)}
      </Chip>
    )
  }
  return (
    <Chip tone="curso" icon="clock">
      Pendiente
    </Chip>
  )
}

function CompleteButton({ milestone }: { milestone: Milestone }) {
  const update = useUpdateMilestone(milestone.id)
  return (
    <button className="btn" type="button" disabled={update.isPending} onClick={() => update.mutate({ completed: !milestone.completed })}>
      <Icon name={milestone.completed ? 'x' : 'check'} />
      {milestone.completed ? 'Reabrir' : 'Marcar cumplido'}
    </button>
  )
}

function MilestoneRow({ milestone, canManage, onEdit }: { milestone: Milestone; canManage: boolean; onEdit: (milestone: Milestone) => void }) {
  return (
    <tr>
      <td className="num">{formatDay(milestone.dueDate)}</td>
      <td>
        <span className="cell-line">
          <Icon name="flag" />
          Hito
        </span>
      </td>
      <td>
        <strong>{milestone.title}</strong>
        {milestone.notes && <p className="soft wrap">{milestone.notes}</p>}
      </td>
      <td>{milestone.owner?.displayName ?? <span className="muted">Sin responsable</span>}</td>
      <td>
        <MilestoneStatus milestone={milestone} />
      </td>
      <td className="col-actions">
        {canManage && (
          <span className="row">
            <CompleteButton milestone={milestone} />
            <button className="btn btn--icon" type="button" title="Editar hito" onClick={() => onEdit(milestone)} aria-label={`Editar ${milestone.title}`}>
              <Icon name="pencil" />
            </button>
          </span>
        )}
      </td>
    </tr>
  )
}

function WindowRow({ window }: { window: CalendarWindow }) {
  return (
    <tr>
      <td className="num">{formatDateTime(window.start)}</td>
      <td>
        <span className="cell-line">
          <Icon name="calendar" />
          Ventana
        </span>
      </td>
      <td>
        <span className="cell-line">
          <Code code={window.change.code} /> {window.change.title}
        </span>
        <p className="soft">
          {window.system} · ruta {ROUTE_LABELS[window.route].toLowerCase()} · hasta {formatDateTime(window.end)}
        </p>
      </td>
      <td>
        <span className="muted">—</span>
      </td>
      <td>
        {window.daysLate > 0 ? (
          <Chip tone="bloq" icon="clock">
            {lateDays(window.daysLate)}
          </Chip>
        ) : (
          <StateChip state={window.state} />
        )}
      </td>
      <td className="col-actions" />
    </tr>
  )
}

function Agenda({ windows, milestones, canManage, onEdit }: { windows: CalendarWindow[]; milestones: Milestone[]; canManage: boolean; onEdit: (milestone: Milestone) => void }) {
  const entries: Entry[] = [
    ...windows.map((window): Entry => ({ kind: 'window', at: window.start, window })),
    ...milestones.map((milestone): Entry => ({ kind: 'milestone', at: `${milestone.dueDate}T00:00:00Z`, milestone })),
  ].sort((a, b) => new Date(a.at).getTime() - new Date(b.at).getTime())

  if (entries.length === 0) return <Empty>No hay hitos ni ventanas de cambio en el calendario.</Empty>
  return (
    <div className="surface table-wrap">
      <table className="table" aria-label="Calendario">
        <thead>
          <tr>
            <th className="num">Fecha</th>
            <th>Tipo</th>
            <th>Qué</th>
            <th>Responsable</th>
            <th>Estado</th>
            <th className="col-actions">
              <span className="sr-only">Acciones</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {entries.map((entry) =>
            entry.kind === 'window' ? (
              <WindowRow key={`w-${entry.window.change.code}`} window={entry.window} />
            ) : (
              <MilestoneRow key={`m-${entry.milestone.id}`} milestone={entry.milestone} canManage={canManage} onEdit={onEdit} />
            ),
          )}
        </tbody>
      </table>
    </div>
  )
}

export default function CalendarPage() {
  const can = useCan()
  const { params } = useUrlFilters()
  const calendar = useCalendar()
  const withCompleted = params.get('completed') === 'true'
  const milestones = useMilestones(withCompleted ? MILESTONES : { ...MILESTONES, status: 'pending' })
  // undefined: cerrado; null: un hito nuevo; un hito: su edición.
  const [editing, setEditing] = useState<Milestone | null | undefined>(undefined)
  const canManage = can('milestone.manage')

  return (
    <>
      <Crumbs items={[{ label: 'Planificación', to: '/planificacion' }, { label: 'Calendario' }]} />
      <PageHead
        title="Calendario"
        subtitle="Hitos y ventanas de cambio por fecha. Lo atrasado sale arriba: un hito pendiente con la fecha vencida o una ventana que terminó sin liberar el CHG."
        actions={
          canManage && (
            <button className="btn btn--primary" type="button" onClick={() => setEditing(null)}>
              <Icon name="plus" />
              Nuevo hito
            </button>
          )
        }
      />
      <section className="surface pad stack" aria-labelledby="h-atrasos">
        <h2 className="h2" id="h-atrasos">
          Alertas de atraso
        </h2>
        <Async query={calendar}>{(data) => <LateAlerts alerts={data.alerts} />}</Async>
      </section>
      <section className="stack" aria-labelledby="h-agenda">
        <h2 className="h2" id="h-agenda">
          Hitos y ventanas
        </h2>
        <FilterBar>
          <FilterCheck name="completed" label="Incluir hitos cumplidos" />
        </FilterBar>
        <Async query={calendar}>
          {(data) => <Async query={milestones}>{(page) => <Agenda windows={data.windows} milestones={page.items} canManage={canManage} onEdit={setEditing} />}</Async>}
        </Async>
      </section>
      {editing !== undefined && (
        <Dialog title={editing ? 'Editar hito' : 'Nuevo hito'} onClose={() => setEditing(undefined)}>
          <MilestoneForm milestone={editing ?? undefined} onClose={() => setEditing(undefined)} />
        </Dialog>
      )}
    </>
  )
}
