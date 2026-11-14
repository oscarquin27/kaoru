// Línea de tiempo de un rango de días: una fila por operador (o por CHG), las actividades como barras
// en carriles para que no se tapen, y los días de solape marcados detrás. Cada barra lleva su código y
// sus fechas en texto: el color nunca es la única señal.
import { Fragment, type CSSProperties, type ReactNode } from 'react'
import { Link } from 'react-router'
import type { ActivityOverlap, ActivitySummary } from '../../shared/api/types.ts'
import { routeOf } from '../../shared/lib/codes.ts'
import { ACTIVITY_KIND_LABELS } from '../../shared/lib/labels.ts'
import { Icon } from '../../shared/ui/Icon.tsx'
import { daysBetween, daysOf, spanText } from './days.ts'

const WEEKDAY = new Intl.DateTimeFormat('es', { weekday: 'narrow', timeZone: 'UTC' })
const MONTH = new Intl.DateTimeFormat('es', { month: 'short', timeZone: 'UTC' })

export interface Span {
  key: string
  start: string
  end: string
}

/** Reparte las barras en carriles: cada una va al primero donde no pisa a la anterior. */
export function packLanes<T extends Span>(items: T[]): T[][] {
  const lanes: T[][] = []
  for (const item of [...items].sort((a, b) => a.start.localeCompare(b.start) || a.end.localeCompare(b.end) || a.key.localeCompare(b.key))) {
    const lane = lanes.find((candidate) => (candidate.at(-1)?.end ?? '') < item.start)
    if (lane) lane.push(item)
    else lanes.push([item])
  }
  return lanes
}

interface Columns {
  from: string
  to: string
  days: number
}

/** Las columnas de la grilla (la 1 es la etiqueta) que ocupa un tramo recortado al rango. */
function columns(range: Columns, start: string, end: string): { style: CSSProperties; cutStart: boolean; cutEnd: boolean } {
  const first = Math.max(0, daysBetween(range.from, start))
  const last = Math.min(range.days - 1, daysBetween(range.from, end))
  return { style: { gridColumn: `${first + 2} / ${last + 3}` }, cutStart: start < range.from, cutEnd: end > range.to }
}

function Bar({ range, activity, lane }: { range: Columns; activity: ActivitySummary; lane: number }) {
  const { style, cutStart, cutEnd } = columns(range, activity.startDate, activity.endDate)
  const label = `${activity.code} · ${activity.title} · ${ACTIVITY_KIND_LABELS[activity.kind]} · ${spanText(activity.startDate, activity.endDate)}`
  const className = ['gantt__bar', `gantt__bar--${activity.kind}`, cutStart && 'gantt__bar--cut-start', cutEnd && 'gantt__bar--cut-end'].filter(Boolean).join(' ')
  return (
    <Link className={className} style={{ ...style, gridRow: lane + 2 }} to={routeOf(activity.code) ?? '/planificacion/actividades'} title={label} aria-label={label}>
      <span className="gantt__code">{activity.code}</span>
      <span className="gantt__text">{activity.title}</span>
    </Link>
  )
}

export function TimelineHead({ from, to, today }: { from: string; to: string; today: string }) {
  const days = daysOf(from, to)
  return (
    <div className="gantt__row gantt__row--head" aria-hidden="true">
      <span className="gantt__label" />
      {days.map((day, index) => {
        const date = new Date(`${day}T00:00:00Z`)
        const weekend = date.getUTCDay() === 0 || date.getUTCDay() === 6
        const showMonth = index === 0 || day.endsWith('-01')
        return (
          <span key={day} className={['gantt__day', weekend && 'gantt__day--weekend', day === today && 'gantt__day--today'].filter(Boolean).join(' ')} style={{ gridColumn: index + 2 }}>
            <span className="gantt__month">{showMonth ? MONTH.format(date) : ' '}</span>
            <span>{WEEKDAY.format(date)}</span>
            <span className="num">{date.getUTCDate()}</span>
          </span>
        )
      })}
    </div>
  )
}

interface RowProps {
  from: string
  to: string
  today: string
  label: ReactNode
  /** Nombre de la fila para lectores de pantalla. */
  name: string
  activities: ActivitySummary[]
  overlaps?: ActivityOverlap[]
  /** Tramos de fondo que no son actividades: las ventanas de cambio de un CHG. */
  marks?: { key: string; start: string; end: string; text: string }[]
}

export function TimelineRow({ from, to, today, label, name, activities, overlaps = [], marks = [] }: RowProps) {
  const range = { from, to, days: daysBetween(from, to) + 1 }
  const lanes = packLanes(activities.map((activity) => ({ key: activity.code, start: activity.startDate, end: activity.endDate, activity })))
  const rows = Math.max(1, lanes.length)
  const todayIndex = daysBetween(from, today)
  const full = { gridRow: '1 / -1' }
  return (
    <div className="gantt__row" role="group" aria-label={name} style={{ '--lanes': rows } as CSSProperties}>
      <div className="gantt__label" style={full}>
        {label}
      </div>
      {daysOf(from, to).map((day, index) => {
        const weekday = new Date(`${day}T00:00:00Z`).getUTCDay()
        return weekday === 0 || weekday === 6 ? <span key={day} className="gantt__shade" style={{ ...full, gridColumn: index + 2 }} aria-hidden="true" /> : null
      })}
      {todayIndex >= 0 && todayIndex < range.days && <span className="gantt__today" style={{ ...full, gridColumn: todayIndex + 2 }} aria-hidden="true" />}
      {marks.map((mark) => (
        <span key={mark.key} className="gantt__mark" style={{ ...columns(range, mark.start, mark.end).style, gridRow: 1 }} title={mark.text}>
          {mark.text}
        </span>
      ))}
      {overlaps.map((overlap) => {
        const key = `${overlap.activity.code}-${overlap.other.code}`
        const where = columns(range, overlap.from, overlap.to).style
        const title = `Solape de ${overlap.activity.code} y ${overlap.other.code} ${spanText(overlap.from, overlap.to)}`
        // Detrás de las barras, la franja; arriba, el ícono con las dos actividades que se cruzan.
        return (
          <Fragment key={key}>
            <span className="gantt__overlap" style={{ ...where, ...full }} title={title} aria-hidden="true" />
            <span className="gantt__flag" style={{ ...where, gridRow: 1 }} title={title} aria-hidden="true">
              <Icon name="triangle" />2
            </span>
          </Fragment>
        )
      })}
      {lanes.map((lane, index) => lane.map((item) => <Bar key={item.key} range={range} activity={item.activity} lane={index} />))}
    </div>
  )
}

export function Timeline({ from, to, children, label }: { from: string; to: string; children: ReactNode; label: string }) {
  return (
    <div className="surface gantt-scroll">
      <div className="gantt" role="region" aria-label={label} style={{ '--days': daysBetween(from, to) + 1 } as CSSProperties}>
        {children}
      </div>
    </div>
  )
}
