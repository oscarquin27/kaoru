// Alertas de atraso: hitos pendientes con la fecha vencida y ventanas de CHG que terminaron sin
// liberarlo. Las calcula la API al leer; lo más atrasado va primero.
import type { CalendarAlert } from '../../shared/api/types.ts'
import { formatDateTime, formatDay } from '../../shared/lib/format.ts'
import { Chip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Empty } from '../../shared/ui/feedback.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'

export const lateDays = (days: number) => (days === 1 ? '1 día de atraso' : `${days} días de atraso`)

export function LateAlerts({ alerts, empty = 'Nada atrasado.' }: { alerts: CalendarAlert[]; empty?: string }) {
  if (alerts.length === 0) return <Empty>{empty}</Empty>
  return (
    <ul className="list" aria-label="Alertas de atraso">
      {alerts.map((alert) => (
        <li key={alert.change ?? alert.milestone}>
          <span className="list__icon">
            <Icon name={alert.kind === 'milestone' ? 'flag' : 'calendar'} />
          </span>
          <div className="list__main">
            {alert.change && (
              <>
                <Code code={alert.change} />{' '}
              </>
            )}
            {alert.title}
            <br />
            <span className="muted">
              {alert.kind === 'milestone' ? `Hito para el ${formatDay(alert.dueAt)}` : `La ventana terminó el ${formatDateTime(alert.dueAt)} y el CHG no está liberado`}
            </span>
          </div>
          <Chip tone="bloq" icon="clock">
            {lateDays(alert.daysLate)}
          </Chip>
        </li>
      ))}
    </ul>
  )
}
