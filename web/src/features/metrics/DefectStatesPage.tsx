// Tiempo en estado de los DEF (requerimiento §8.3 y §10), con los cerrados incluidos: cuántos pasaron
// por cada estado y cuánto se quedaron. Lo que sigue abierto cuenta hasta hoy. Sin SLA: solo se mide.
import type { DefectStateMetrics } from '../../shared/api/types.ts'
import { formatDecimalDays } from '../../shared/lib/format.ts'
import { StateChip } from '../../shared/ui/chips.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { useDefectStateMetrics } from './api.ts'
import { MetricsFilters, MetricsHead, PeriodNote, useMetricsQuery } from './common.tsx'

function days(value: number | undefined) {
  return value === undefined ? <span className="muted">—</span> : formatDecimalDays(value)
}

function Summary({ data }: { data: DefectStateMetrics }) {
  const { resolution } = data
  return (
    <section className="glass kpis" aria-label="Resumen de defectos">
      <div className="kpi">
        <span className="kpi__label">DEF medidos</span>
        <span className="kpi__value">{data.total}</span>
        <span className="kpi__note">Creados en el período</span>
      </div>
      <div className="kpi">
        <span className="kpi__label">Resueltos</span>
        <span className="kpi__value">{resolution.defects}</span>
        <span className="kpi__note">Hoy en un estado final</span>
      </div>
      <div className="kpi">
        <span className="kpi__label">De abierto a resuelto (mediana)</span>
        <span className="kpi__value">{days(resolution.medianDays)}</span>
        <span className="kpi__note">Percentil 90: {resolution.p90Days === undefined ? '—' : formatDecimalDays(resolution.p90Days)}</span>
      </div>
    </section>
  )
}

function StatesTable({ states }: { states: DefectStateMetrics['states'] }) {
  return (
    <div className="surface table-wrap">
      <table className="table" aria-label="Tiempo en estado">
        <thead>
          <tr>
            <th>Estado</th>
            <th className="num">Hoy</th>
            <th className="num">Pasaron</th>
            <th className="num">Entradas</th>
            <th className="num">Mediana</th>
            <th className="num">Percentil 90</th>
          </tr>
        </thead>
        <tbody>
          {states.map((row) => (
            <tr key={row.state.key}>
              <td>
                <StateChip state={row.state} />
              </td>
              <td className="num">{row.current}</td>
              <td className="num">{row.defects}</td>
              <td className="num">{row.visits}</td>
              <td className="num">{row.isFinal ? <span className="muted">Final</span> : days(row.medianDays)}</td>
              <td className="num">{row.isFinal ? <span className="muted">Final</span> : days(row.p90Days)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export default function DefectStatesPage() {
  const metrics = useDefectStateMetrics(useMetricsQuery(['system', 'severity', 'from', 'to']))
  return (
    <>
      <MetricsHead
        title="Tiempo en estado"
        subtitle="Cuánto pasan los defectos en cada estado, con los cerrados incluidos. Mediana y percentil 90 del tiempo total de cada DEF en el estado; lo abierto cuenta hasta hoy. No hay SLA: solo se mide."
      />
      <MetricsFilters severity period />
      <Async query={metrics}>
        {(data) => (
          <>
            <PeriodNote period={data.period} what="defectos creados en esas fechas" />
            <Summary data={data} />
            {data.total === 0 ? <Empty>No hay defectos creados en el período con este filtro.</Empty> : <StatesTable states={data.states} />}
            <p className="muted">«Pasaron» cuenta defectos; «Entradas», las veces que se entró al estado: un DEF que vuelve suma una entrada más.</p>
          </>
        )}
      </Async>
    </>
  )
}
