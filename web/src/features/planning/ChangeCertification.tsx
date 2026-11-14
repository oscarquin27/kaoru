// La certificación de un CHG en su ficha: si QA ya tiene reservado el período, quiénes lo hacen y qué
// más tiene ese CHG planificado. Es lo mismo que mira la regla certification_planned al pasar a pruebas.
import { useState } from 'react'
import { useCan } from '../../shared/api/session.ts'
import type { Activity, ActivitySummary } from '../../shared/api/types.ts'
import { Chip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async } from '../../shared/ui/feedback.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { ActivityStatusChip, KindChip } from './ActivityChips.tsx'
import { ActivityForm } from './ActivityForm.tsx'
import { useActivities } from './api.ts'
import { spanText, today as todayOf } from './days.ts'
import { OverlapList } from './Overlaps.tsx'

export type CertificationState = 'none' | 'no_operators' | 'outside' | 'ready'

/** El mismo criterio que la regla: un período no cancelado, con operadores, que cubra hoy. */
export function certificationState(activities: ActivitySummary[], today = todayOf()): { state: CertificationState; period?: ActivitySummary } {
  const periods = activities.filter((activity) => activity.kind === 'certification' && activity.status !== 'cancelled')
  if (periods.length === 0) return { state: 'none' }
  const staffed = periods.filter((activity) => activity.operators.length > 0)
  if (staffed.length === 0) return { state: 'no_operators', period: periods[0] }
  const current = staffed.find((activity) => activity.startDate <= today && today <= activity.endDate)
  if (current) return { state: 'ready', period: current }
  // Fuera de fechas: el más cercano, primero el que viene.
  const next = staffed.find((activity) => activity.startDate > today)
  return { state: 'outside', period: next ?? staffed[staffed.length - 1] }
}

function StateNotice({ code, state, period }: { code: string; state: CertificationState; period?: ActivitySummary }) {
  switch (state) {
    case 'none':
      return (
        <Chip tone="aviso" icon="triangle" large>
          Sin período de certificación
        </Chip>
      )
    case 'no_operators':
      return (
        <Chip tone="aviso" icon="user" large>
          Período sin operadores
        </Chip>
      )
    case 'outside':
      return (
        <Chip tone="curso" icon="calendar" large>
          {period && period.startDate > todayOf() ? `Certifica ${spanText(period.startDate, period.endDate)}` : `Fuera del período (${period ? spanText(period.startDate, period.endDate) : code})`}
        </Chip>
      )
    case 'ready':
      return (
        <Chip tone="ok" icon="check" large>
          En período de certificación
        </Chip>
      )
  }
}

const HINTS: Record<CertificationState, string> = {
  none: 'Reserva las fechas de QA antes de que el CHG entre en certificación. Sin ellas, pasar a pruebas sus REQ muestra un aviso.',
  no_operators: 'El período está reservado pero nadie lo tiene asignado.',
  outside: 'Hoy está fuera del período reservado. Pasar a pruebas sus REQ ahora muestra un aviso.',
  ready: 'El período está reservado, tiene operadores y cubre hoy.',
}

export function ChangeCertification({ code }: { code: string }) {
  const can = useCan()
  const activities = useActivities({ change: code, size: 100 })
  const [planning, setPlanning] = useState(false)
  // La que se acaba de planificar aquí: sus solapes se avisan sin salir de la ficha.
  const [created, setCreated] = useState<Activity>()

  return (
    <section className="surface pad stack" aria-labelledby="h-certificacion">
      <div className="row row--between">
        <h2 className="h3" id="h-certificacion">
          Certificación
        </h2>
        {can('activity.manage') && (
          <button className="btn" type="button" onClick={() => setPlanning(true)}>
            <Icon name="calendar" />
            Planificar certificación
          </button>
        )}
      </div>
      {created && <OverlapList overlaps={created.overlaps} label={`Solapes de ${created.code}`} />}
      <Async query={activities}>
        {(page) => {
          const { state, period } = certificationState(page.items)
          return (
            <>
              <div className="stack-s">
                <StateNotice code={code} state={state} period={period} />
                <p className="soft">{HINTS[state]}</p>
              </div>
              {page.items.length > 0 && (
                <ul className="list" aria-label={`Actividades de ${code}`}>
                  {page.items.map((activity) => (
                    <li key={activity.code}>
                      <div className="list__main cell-stack">
                        <span className="cell-line">
                          <Code code={activity.code} /> {activity.title}
                        </span>
                        <span className="soft">
                          {spanText(activity.startDate, activity.endDate)} ·{' '}
                          {activity.operators.length > 0 ? activity.operators.map((operator) => operator.displayName).join(', ') : 'sin operador'}
                        </span>
                      </div>
                      <KindChip kind={activity.kind} />
                      <ActivityStatusChip status={activity.status} />
                    </li>
                  ))}
                </ul>
              )}
            </>
          )
        }}
      </Async>
      {planning && (
        <Dialog title={`Planificar la certificación de ${code}`} onClose={() => setPlanning(false)}>
          <ActivityForm onCreated={setCreated} defaults={{ kind: 'certification', change: code, title: `Certificación ${code}` }} onClose={() => setPlanning(false)} />
        </Dialog>
      )}
    </section>
  )
}
