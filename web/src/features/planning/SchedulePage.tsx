// Planificación por operador: qué hace cada uno en el rango, dónde se le cruzan dos actividades y qué
// CHG están en certificación. El rango vive en la URL (desde, semanas) para poder compartir el enlace.
import { useState } from 'react'
import { useCan, useUserOptions } from '../../shared/api/session.ts'
import type { ActivitySummary, Schedule, ScheduleWindow } from '../../shared/api/types.ts'
import { formatDay } from '../../shared/lib/format.ts'
import { ACTIVITY_KIND_LABELS, options } from '../../shared/lib/labels.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Chip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterSelect } from '../../shared/ui/Filters.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { KIND_LOOK } from './ActivityChips.tsx'
import { ActivityForm } from './ActivityForm.tsx'
import { useSchedule } from './api.ts'
import { addDays, mondayOf, spanText, today as todayOf } from './days.ts'
import { OverlapList } from './Overlaps.tsx'
import { Timeline, TimelineHead, TimelineRow } from './Timeline.tsx'

const WEEKS = [
  { value: '2', label: '2 semanas' },
  { value: '6', label: '6 semanas' },
  { value: '12', label: '12 semanas' },
]

/** El rango pedido en la URL; sin él, seis semanas desde el lunes de esta. */
export function useRange() {
  const { params, setFilter } = useUrlFilters()
  const today = todayOf()
  const asked = params.get('desde')
  const from = asked && /^\d{4}-\d{2}-\d{2}$/.test(asked) ? asked : mondayOf(today)
  const weeks = WEEKS.some((option) => option.value === params.get('semanas')) ? Number(params.get('semanas')) : 6
  return {
    today,
    from,
    to: addDays(from, weeks * 7 - 1),
    weeks,
    /** Mueve el rango; volver a la semana de hoy limpia la URL. */
    move: (days: number) => {
      const next = days === 0 ? mondayOf(today) : addDays(from, days)
      setFilter('desde', next === mondayOf(today) ? '' : next)
    },
  }
}

function WeeksSelect() {
  const { params, setFilter } = useUrlFilters()
  const value = WEEKS.some((option) => option.value === params.get('semanas')) ? (params.get('semanas') ?? '6') : '6'
  return (
    <label className="field">
      Largo
      <select className="select" value={value} onChange={(event) => setFilter('semanas', event.target.value === '6' ? '' : event.target.value)}>
        {WEEKS.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  )
}

function RangeNav({ from, to, weeks, move }: ReturnType<typeof useRange>) {
  return (
    <div className="row" role="group" aria-label="Rango de fechas">
      <button className="btn" type="button" onClick={() => move(-7)}>
        Semana anterior
      </button>
      <button className="btn" type="button" onClick={() => move(0)}>
        Esta semana
      </button>
      <button className="btn" type="button" onClick={() => move(7)}>
        Semana siguiente
      </button>
      <span className="soft" aria-live="polite">
        {weeks} semanas, {spanText(from, to)}
      </span>
    </div>
  )
}

function Legend() {
  return (
    <p className="row soft" aria-label="Leyenda">
      {options(ACTIVITY_KIND_LABELS).map((option) => (
        <Chip key={option.value} {...KIND_LOOK[option.value as keyof typeof KIND_LOOK]}>
          {option.label}
        </Chip>
      ))}
      <span className="cell-line">
        <span className="legend-swatch legend-swatch--overlap" aria-hidden="true" />
        Días con solape
      </span>
      <span className="cell-line">
        <span className="legend-swatch legend-swatch--window" aria-hidden="true" />
        Ventana de cambio
      </span>
    </p>
  )
}

function Kpis({ schedule }: { schedule: Schedule }) {
  const { activities, operators, overlaps } = schedule.summary
  return (
    <section className="glass kpis" aria-label="Resumen del rango">
      <div className="kpi">
        <span className="kpi__label">Actividades</span>
        <span className="kpi__value">{activities}</span>
        <span className="kpi__note">No canceladas, que tocan el rango</span>
      </div>
      <div className="kpi">
        <span className="kpi__label">Operadores ocupados</span>
        <span className="kpi__value">{operators}</span>
        <span className="kpi__note">Con alguna actividad en el rango</span>
      </div>
      <div className="kpi">
        <span className="kpi__label">Solapes</span>
        <span className="kpi__value">{overlaps}</span>
        <span className="kpi__note">El mismo operador en dos actividades a la vez</span>
      </div>
      <div className="kpi">
        <span className="kpi__label">Sin operador</span>
        <span className="kpi__value">{schedule.unassigned.length}</span>
        <span className="kpi__note">Actividades que nadie tiene asignadas</span>
      </div>
    </section>
  )
}

function OperatorsTimeline({ schedule, today }: { schedule: Schedule; today: string }) {
  if (schedule.operators.length === 0) return <Empty>Nadie tiene actividades en este rango.</Empty>
  return (
    <Timeline from={schedule.from} to={schedule.to} label="Línea de tiempo por operador">
      <TimelineHead from={schedule.from} to={schedule.to} today={today} />
      {schedule.operators.map((row) => (
        <TimelineRow
          key={row.operator.id}
          from={schedule.from}
          to={schedule.to}
          today={today}
          name={row.overlaps.length > 0 ? `${row.operator.displayName}, ${row.overlaps.length} solapes` : row.operator.displayName}
          activities={row.activities}
          overlaps={row.overlaps}
          label={
            <>
              <strong>{row.operator.displayName}</strong>
              {row.overlaps.length > 0 ? (
                <Chip tone="bloq" icon="triangle">
                  {row.overlaps.length === 1 ? '1 solape' : `${row.overlaps.length} solapes`}
                </Chip>
              ) : (
                <span className="soft">{row.activities.length === 1 ? '1 actividad' : `${row.activities.length} actividades`}</span>
              )}
            </>
          }
        />
      ))}
    </Timeline>
  )
}

interface Project {
  code: string
  title: string
  system: string
  certifications: ActivitySummary[]
  windows: ScheduleWindow[]
}

/** Los CHG del rango: los que tienen certificación y los que solo tienen ventana (sin planificar). */
export function projectsOf(schedule: Schedule): Project[] {
  const byCode = new Map<string, Project>()
  for (const project of schedule.projects) {
    byCode.set(project.change.code, { ...project.change, certifications: project.certifications, windows: [] })
  }
  for (const window of schedule.windows) {
    const project = byCode.get(window.change.code) ?? { code: window.change.code, title: window.change.title, system: window.system, certifications: [], windows: [] }
    project.windows.push(window)
    byCode.set(window.change.code, project)
  }
  return [...byCode.values()]
}

function ProjectsTimeline({ schedule, today }: { schedule: Schedule; today: string }) {
  const projects = projectsOf(schedule)
  if (projects.length === 0) return <Empty>Ningún CHG tiene certificación ni ventana de cambio en este rango.</Empty>
  return (
    <Timeline from={schedule.from} to={schedule.to} label="Línea de tiempo por CHG">
      <TimelineHead from={schedule.from} to={schedule.to} today={today} />
      {projects.map((project) => (
        <TimelineRow
          key={project.code}
          from={schedule.from}
          to={schedule.to}
          today={today}
          name={`${project.code} ${project.title}${project.certifications.length === 0 ? ', sin certificación planificada' : ''}`}
          activities={project.certifications}
          marks={project.windows.map((window) => ({
            key: window.start,
            start: window.start.slice(0, 10),
            end: window.end.slice(0, 10),
            text: `Ventana ${spanText(window.start.slice(0, 10), window.end.slice(0, 10))}`,
          }))}
          label={
            <>
              <span className="cell-line">
                <Code code={project.code} />
                <span className="soft">{project.system}</span>
              </span>
              <span className="soft">{project.title}</span>
              {project.certifications.length === 0 && (
                <Chip tone="aviso" icon="triangle">
                  Sin certificación
                </Chip>
              )}
            </>
          }
        />
      ))}
    </Timeline>
  )
}

function Unassigned({ activities }: { activities: ActivitySummary[] }) {
  if (activities.length === 0) return null
  return (
    <section className="stack" aria-labelledby="h-sin-operador">
      <h2 className="h2" id="h-sin-operador">
        Sin operador
      </h2>
      <ul className="surface pad list">
        {activities.map((activity) => (
          <li key={activity.code}>
            <span className="list__icon list__icon--aviso">
              <Icon name="user" />
            </span>
            <div className="list__main cell-stack">
              <span className="cell-line">
                <Code code={activity.code} /> {activity.title}
                {activity.change && <Code code={activity.change.code} />}
              </span>
              <span className="soft">
                {ACTIVITY_KIND_LABELS[activity.kind]} · {spanText(activity.startDate, activity.endDate)}. Nadie la tiene asignada.
              </span>
            </div>
          </li>
        ))}
      </ul>
    </section>
  )
}

function Milestones({ schedule }: { schedule: Schedule }) {
  if (schedule.milestones.length === 0) return null
  return (
    <section className="stack" aria-labelledby="h-hitos">
      <h2 className="h2" id="h-hitos">
        Hitos del rango
      </h2>
      <ul className="surface pad list">
        {schedule.milestones.map((milestone) => (
          <li key={milestone.id}>
            <span className={`list__icon ${milestone.completed ? 'list__icon--ok' : milestone.daysLate > 0 ? 'list__icon--bloq' : ''}`}>
              <Icon name="flag" />
            </span>
            <div className="list__main cell-stack">
              <strong>{milestone.title}</strong>
              <span className="soft">
                {formatDay(milestone.dueDate)}
                {milestone.owner && ` · ${milestone.owner.displayName}`}
                {milestone.completed ? ' · cumplido' : milestone.daysLate > 0 ? ' · atrasado' : ''}
              </span>
            </div>
          </li>
        ))}
      </ul>
    </section>
  )
}

export default function SchedulePage() {
  const can = useCan()
  const range = useRange()
  const { params } = useUrlFilters()
  const operatorOptions = useUserOptions()
  const operator = params.get('operador') || undefined
  const schedule = useSchedule({ from: range.from, to: range.to, operator })
  const [creating, setCreating] = useState(false)

  return (
    <>
      <Crumbs items={[{ label: 'Planificación' }]} />
      <PageHead
        title="Planificación"
        subtitle="Qué hace cada operador, día por día. Las franjas rayadas son días en que alguien tiene dos actividades a la vez: se guardan igual, pero conviene reajustarlas."
        actions={
          can('activity.manage') && (
            <button className="btn btn--primary" type="button" onClick={() => setCreating(true)}>
              <Icon name="plus" />
              Nueva actividad
            </button>
          )
        }
      />
      <FilterBar actions={<RangeNav {...range} />}>
        <FilterSelect name="operador" label="Operador" options={operatorOptions} />
        <WeeksSelect />
      </FilterBar>
      <Async query={schedule}>
        {(data) => {
          const overlaps = data.operators.flatMap((row) => row.overlaps)
          return (
            <>
              <Kpis schedule={data} />
              <OverlapList overlaps={overlaps} />
              <section className="stack" aria-labelledby="h-operadores">
                <div className="row row--between">
                  <h2 className="h2" id="h-operadores">
                    Por operador
                  </h2>
                  <Legend />
                </div>
                <OperatorsTimeline schedule={data} today={range.today} />
              </section>
              <section className="stack" aria-labelledby="h-proyectos">
                <h2 className="h2" id="h-proyectos">
                  Proyectos en certificación
                </h2>
                <p className="soft">Cada CHG con su período de certificación reservado y su ventana de cambio. Un CHG con ventana y sin certificación todavía no tiene fechas para QA.</p>
                <ProjectsTimeline schedule={data} today={range.today} />
              </section>
              <Unassigned activities={data.unassigned} />
              <Milestones schedule={data} />
            </>
          )
        }}
      </Async>
      {creating && (
        <Dialog title="Nueva actividad" onClose={() => setCreating(false)}>
          <ActivityForm onClose={() => setCreating(false)} />
        </Dialog>
      )}
    </>
  )
}
