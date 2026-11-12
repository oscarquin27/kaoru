// Inicio: tablero por rol (boceto: index.html). Se arma con las listas que ya da la API y con el
// calendario, que trae las ventanas de cambio y las alertas de atraso.
import { useState } from 'react'
import { useChanges } from '../features/changes/api.ts'
import { useDefects } from '../features/defects/api.ts'
import { useRequirements } from '../features/requirements/api.ts'
import { LateAlerts } from '../features/planning/LateAlerts.tsx'
import { useCalendar } from '../features/planning/api.ts'
import { useMatrix } from '../features/traceability/api.ts'
import { useCan, useMe, useWorkflow } from '../shared/api/session.ts'
import type { CalendarWindow, Change, Defect, Requirement, StateRef, Workflow } from '../shared/api/types.ts'
import { formatDate, formatDays, formatDaysLong, formatLongDate, median } from '../shared/lib/format.ts'
import { ROLE_LABELS, ROUTE_LABELS } from '../shared/lib/labels.ts'
import { SeverityChip, StateChip } from '../shared/ui/chips.tsx'
import { Code } from '../shared/ui/Code.tsx'
import { Async, Empty } from '../shared/ui/feedback.tsx'
import { Icon } from '../shared/ui/Icon.tsx'
import { Crumbs, Meter, PageHead } from '../shared/ui/layout.tsx'

// El tablero mira la primera página grande de cada lista, no todo el historial.
const BOARD = { size: 200 }
const DAY = 86_400_000

/** Estados desde los que el usuario puede dar el siguiente paso de la línea de vida. */
function actionableStates(workflow: Workflow | undefined, can: (permission: string) => boolean): Set<string> {
  const onLifeline = new Set(workflow?.states.filter((state) => state.position !== undefined).map((state) => state.key))
  return new Set(workflow?.transitions.filter((transition) => onLifeline.has(transition.to) && can(transition.permission)).map((transition) => transition.from))
}

function Pending({ requirements, defects }: { requirements: Requirement[]; defects: Defect[] }) {
  const can = useCan()
  const me = useMe().data
  const requirementStates = actionableStates(useWorkflow('requirement').data, can)
  const defectStates = actionableStates(useWorkflow('defect').data, can)
  const myRequirements = requirements.filter((item) => item.owner?.id === me?.id || requirementStates.has(item.state.key))
  const myDefects = defects.filter((item) => item.assignee?.id === me?.id || defectStates.has(item.state.key))

  if (myRequirements.length + myDefects.length === 0) return <Empty>Nada pendiente para tu rol.</Empty>
  return (
    <ul className="list">
      {myRequirements.map((item) => (
        <li key={item.code}>
          <span className="list__icon">
            <Icon name="file" />
          </span>
          <div className="list__main">
            <Code code={item.code} /> {item.title}
            <br />
            <span className="muted">{item.owner?.id === me?.id ? 'Eres responsable' : 'Tu rol puede dar el siguiente paso'}</span>
          </div>
          <StateChip state={item.state} />
        </li>
      ))}
      {myDefects.map((item) => (
        <li key={item.code}>
          <span className="list__icon">
            <Icon name="bug" />
          </span>
          <div className="list__main">
            <Code code={item.code} /> {item.title}
            <br />
            <span className="muted">
              {item.assignee?.id === me?.id ? 'Asignado a ti' : 'Tu rol puede dar el siguiente paso'} · {formatDaysLong(item.daysInState)} en este estado
            </span>
          </div>
          <StateChip state={item.state} />
        </li>
      ))}
    </ul>
  )
}

function Windows({ windows }: { windows: CalendarWindow[] }) {
  const [now] = useState(() => Date.now())
  // El calendario ya las trae por fecha de inicio; aquí solo las que no empezaron.
  const upcoming = windows.filter((window) => new Date(window.start).getTime() > now)
  if (upcoming.length === 0) return <Empty>Ningún cambio tiene una ventana futura definida.</Empty>
  return (
    <ul className="list">
      {upcoming.map((window) => (
        <li key={window.change.code}>
          <span className="list__icon">
            <Icon name="calendar" />
          </span>
          <div className="list__main">
            <Code code={window.change.code} /> · {window.system}
            <br />
            <span className="muted">{formatLongDate(new Date(window.start))}</span>
          </div>
          <span className="muted">en {formatDaysLong((new Date(window.start).getTime() - now) / DAY)}</span>
        </li>
      ))}
    </ul>
  )
}

function ChangesInProgress({ changes, requirements }: { changes: Change[]; requirements: Requirement[] }) {
  const workflow = useWorkflow('requirement').data
  const changeWorkflow = useWorkflow('change').data
  const position = (key: string) => workflow?.states.find((state) => state.key === key)?.position ?? 0
  const approvedFrom = position('approved_qa')
  const finals = new Set(changeWorkflow?.states.filter((state) => state.isFinal).map((state) => state.key))
  const open = changes.filter((change) => !finals.has(change.state.key))

  if (open.length === 0) return <Empty>No hay cambios en curso.</Empty>
  return (
    <div className="surface table-wrap">
      <table className="table">
        <thead>
          <tr>
            <th>CHG</th>
            <th>Sistema</th>
            <th>Ruta</th>
            <th>Estado</th>
            <th>Avance</th>
            <th>Ventana</th>
          </tr>
        </thead>
        <tbody>
          {open.map((change) => {
            const own = requirements.filter((item) => item.change?.code === change.code)
            const approved = own.filter((item) => approvedFrom > 0 && position(item.state.key) >= approvedFrom).length
            return (
              <tr key={change.code}>
                <td>
                  <Code code={change.code} />
                </td>
                <td>{change.system.name}</td>
                <td>{ROUTE_LABELS[change.route]}</td>
                <td>
                  <StateChip state={change.state} />
                </td>
                <td>
                  {own.length === 0 ? (
                    <span className="muted">Sin REQ</span>
                  ) : (
                    <div className="cell-line">
                      <Meter percent={Math.round((approved / own.length) * 100)} />
                      <span className="num soft">
                        {approved} de {own.length} REQ aprobados
                      </span>
                    </div>
                  )}
                </td>
                <td className="num">{formatDate(change.windowStart)}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function DefectsByState({ defects }: { defects: Defect[] }) {
  const finals = new Set(useWorkflow('defect').data?.states.filter((state) => state.isFinal).map((state) => state.key))
  const groups = new Map<string, { state: StateRef; items: Defect[] }>()
  for (const defect of defects) {
    if (finals.has(defect.state.key)) continue
    const group = groups.get(defect.state.key) ?? { state: defect.state, items: [] }
    group.items.push(defect)
    groups.set(defect.state.key, group)
  }
  if (groups.size === 0) return <Empty>No hay defectos abiertos.</Empty>
  return (
    <div className="surface table-wrap">
      <table className="table">
        <thead>
          <tr>
            <th>Estado</th>
            <th className="num">Defectos</th>
            <th className="num">Alta</th>
            <th className="num">Tiempo en estado (mediana)</th>
            <th className="num">Más antiguo</th>
          </tr>
        </thead>
        <tbody>
          {[...groups.values()].map(({ state, items }) => {
            const days = items.map((item) => item.daysInState)
            return (
              <tr key={state.key}>
                <td>
                  <StateChip state={state} />
                </td>
                <td className="num">{items.length}</td>
                <td className="num">{items.filter((item) => item.severity === 'high').length}</td>
                <td className="num">{formatDays(median(days))}</td>
                <td className="num">{formatDays(Math.max(...days))}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function Indicators() {
  const indicators = useMatrix().data?.indicators
  if (!indicators) return null
  const { high, medium, low } = indicators.openDefects
  return (
    <section className="glass kpis" aria-label="Indicadores">
      <div className="kpi">
        <span className="kpi__label">Cobertura</span>
        <span className="kpi__value">{indicators.coveragePercent} %</span>
        <span className="kpi__note">TC con RUN aprobado / TC requeridos</span>
      </div>
      <div className="kpi">
        <span className="kpi__label">REQ bloqueados</span>
        <span className="kpi__value">{indicators.blockedRequirements}</span>
        <span className="kpi__note">Con algún defecto de severidad alta abierto</span>
      </div>
      <div className="kpi">
        <span className="kpi__label">Defectos abiertos</span>
        <span className="kpi__value">{high + medium + low}</span>
        <span className="row">
          <SeverityChip severity="high" count={high} />
          <SeverityChip severity="medium" count={medium} />
          <SeverityChip severity="low" count={low} />
        </span>
      </div>
    </section>
  )
}

export default function HomePage() {
  const me = useMe().data
  const requirements = useRequirements(BOARD)
  const defects = useDefects(BOARD)
  const changes = useChanges(BOARD)
  const calendar = useCalendar()
  const role = me?.roles.map((item) => ROLE_LABELS[item]).join(' · ')

  return (
    <>
      <Crumbs items={[{ label: 'Inicio' }]} />
      <PageHead title="Inicio" subtitle={`${role ? `Vista de ${role} · ` : ''}${formatLongDate(new Date())}`} />
      <Indicators />

      <div className="grid grid--2">
        <section className="surface pad stack" aria-labelledby="h-pendientes">
          <h2 className="h2" id="h-pendientes">
            Mis pendientes
          </h2>
          <Async query={requirements}>{(reqs) => <Async query={defects}>{(defs) => <Pending requirements={reqs.items} defects={defs.items} />}</Async>}</Async>
        </section>
        <section className="surface pad stack" aria-labelledby="h-ventanas">
          <h2 className="h2" id="h-ventanas">
            Próximas ventanas de cambio
          </h2>
          <Async query={calendar}>{(data) => <Windows windows={data.windows} />}</Async>
        </section>
      </div>

      <section className="surface pad stack" aria-labelledby="h-atrasos">
        <h2 className="h2" id="h-atrasos">
          Alertas de atraso
        </h2>
        <Async query={calendar}>{(data) => <LateAlerts alerts={data.alerts} empty="Ningún hito ni ventana de cambio está atrasado." />}</Async>
      </section>

      <section className="stack" aria-labelledby="h-cambios">
        <h2 className="h2" id="h-cambios">
          Cambios en curso
        </h2>
        <Async query={changes}>{(data) => <ChangesInProgress changes={data.items} requirements={requirements.data?.items ?? []} />}</Async>
      </section>

      <section className="stack" aria-labelledby="h-defectos">
        <div>
          <h2 className="h2" id="h-defectos">
            Defectos por estado
          </h2>
          <p className="subtitle">Tiempo en estado, sin SLA.</p>
        </div>
        <Async query={defects}>{(data) => <DefectsByState defects={data.items} />}</Async>
      </section>
    </>
  )
}
