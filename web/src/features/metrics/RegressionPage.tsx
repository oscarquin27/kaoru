// Tamaño de la regresión (requerimiento §10): una fila por PKG con la regresión confirmada, para
// comparar lo que propuso el sistema, lo que QA ajustó y lo que quedó aprobado.
import type { RegressionMetrics } from '../../shared/api/types.ts'
import { formatDate } from '../../shared/lib/format.ts'
import { StateChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { Pager } from '../../shared/ui/Filters.tsx'
import { useRegressionMetrics } from './api.ts'
import { MetricsFilters, MetricsHead, PeriodNote, useMetricsQuery } from './common.tsx'

function Summary({ summary }: { summary: RegressionMetrics['summary'] }) {
  return (
    <section className="glass kpis" aria-label="Resumen de regresión">
      <div className="kpi">
        <span className="kpi__label">PKG con regresión confirmada</span>
        <span className="kpi__value">{summary.packages}</span>
      </div>
      <div className="kpi">
        <span className="kpi__label">TC por PKG (mediana)</span>
        <span className="kpi__value">{summary.medianFinal?.toLocaleString('es') ?? '—'}</span>
        <span className="kpi__note">
          {summary.final} TC finales de {summary.proposed} propuestos
        </span>
      </div>
      <div className="kpi">
        <span className="kpi__label">Ajustes de QA</span>
        <span className="kpi__value">{summary.added + summary.removed}</span>
        <span className="kpi__note">
          {summary.added} agregados · {summary.removed} quitados
        </span>
      </div>
    </section>
  )
}

function RegressionTable({ items }: { items: RegressionMetrics['items'] }) {
  if (items.length === 0) return <Empty>No hay paquetes con la regresión confirmada en el período.</Empty>
  return (
    <div className="surface table-wrap">
      <table className="table" aria-label="Tamaño de regresión por PKG">
        <thead>
          <tr>
            <th className="col-fixed">PKG</th>
            <th>Estado</th>
            <th>Confirmada</th>
            <th className="num">Propuestos</th>
            <th className="num">Finales</th>
            <th className="num">Agregados</th>
            <th className="num">Quitados</th>
            <th>Por origen</th>
            <th className="num">Aprobados</th>
            <th className="num">TC del sistema</th>
          </tr>
        </thead>
        <tbody>
          {items.map((row) => (
            <tr key={row.package.code}>
              <td className="col-fixed">
                <div className="cell-stack">
                  <Code code={row.package.code} />
                  <span className="muted">{row.package.title}</span>
                </div>
              </td>
              <td>
                <StateChip state={row.state} />
              </td>
              <td>{formatDate(row.confirmedAt)}</td>
              <td className="num">{row.proposed}</td>
              <td className="num">{row.final}</td>
              <td className="num">{row.added}</td>
              <td className="num">{row.removed}</td>
              <td>
                {row.byOrigin.impact} por impacto · {row.byOrigin.critical} críticos · {row.byOrigin.manual} manuales
              </td>
              <td className="num">
                {row.passed} de {row.final}
              </td>
              <td className="num">{row.systemTestCases}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export default function RegressionPage() {
  const metrics = useRegressionMetrics(useMetricsQuery(['system', 'from', 'to']))
  return (
    <>
      <MetricsHead
        title="Tamaño de regresión"
        subtitle="Por paquete: los TC que propuso el sistema por impacto y críticos, los que QA agregó o quitó, y cuántos de los finales tienen su último RUN sobre un BUILD del PKG aprobado."
      />
      <MetricsFilters period />
      <Async query={metrics}>
        {(data) => (
          <>
            <PeriodNote period={data.period} what="regresiones confirmadas en esas fechas" />
            <Summary summary={data.summary} />
            <RegressionTable items={data.items} />
            <Pager page={data.page} size={data.size} total={data.total} noun="paquetes" />
            <p className="muted">«TC del sistema» son los TC activos del sistema hoy, para comparar el tamaño de la regresión con el total.</p>
          </>
        )}
      </Async>
    </>
  )
}
