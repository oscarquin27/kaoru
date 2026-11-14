// Defectos escapados a producción (requerimiento §10): un DEF escapó si se abrió después de que su REQ
// llegó a producción, es decir, después del primer despliegue exitoso en producción de un PKG que lo
// contiene. Se calcula: nadie lo marca a mano.
import type { EscapedDefectMetrics } from '../../shared/api/types.ts'
import { formatDate, formatDecimalDays } from '../../shared/lib/format.ts'
import { SeverityChip, StateChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { Pager } from '../../shared/ui/Filters.tsx'
import { Meter } from '../../shared/ui/layout.tsx'
import { useEscapedDefectMetrics } from './api.ts'
import { MetricsFilters, MetricsHead, PeriodNote, useMetricsQuery } from './common.tsx'

const MONTH = new Intl.DateTimeFormat('es', { month: 'short', year: 'numeric', timeZone: 'UTC' })

function Summary({ summary }: { summary: EscapedDefectMetrics['summary'] }) {
  return (
    <section className="glass kpis" aria-label="Resumen de escapados">
      <div className="kpi">
        <span className="kpi__label">DEF escapados</span>
        <span className="kpi__value">{summary.escaped}</span>
        <span className="kpi__note">De {summary.defects} creados en el período</span>
      </div>
      <div className="kpi">
        <span className="kpi__label">Tasa de escape</span>
        <span className="kpi__value">{summary.ratePercent.toLocaleString('es')} %</span>
        <span className="kpi__note">Escapados / DEF creados</span>
      </div>
    </section>
  )
}

/** Un renglón por mes: la barra es la parte de los DEF del mes que escaparon, con el número escrito. */
function Months({ months }: { months: EscapedDefectMetrics['months'] }) {
  return (
    <div className="surface table-wrap">
      <table className="table" aria-label="Escapados por mes">
        <thead>
          <tr>
            <th>Mes</th>
            <th className="num">DEF</th>
            <th className="num">Escapados</th>
            <th>Parte que escapó</th>
          </tr>
        </thead>
        <tbody>
          {months.map((month) => {
            const percent = month.defects === 0 ? 0 : Math.round((month.escaped * 100) / month.defects)
            return (
              <tr key={month.month}>
                <td>{MONTH.format(new Date(`${month.month}-01T00:00:00Z`))}</td>
                <td className="num">{month.defects}</td>
                <td className="num">{month.escaped}</td>
                <td>
                  <span className="cell-line">
                    <Meter percent={percent} />
                    <span className="num">{percent} %</span>
                  </span>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function EscapedTable({ items }: { items: EscapedDefectMetrics['items'] }) {
  if (items.length === 0) return <Empty>Ningún defecto del período se abrió después de que su REQ llegó a producción.</Empty>
  return (
    <div className="surface table-wrap">
      <table className="table" aria-label="Defectos escapados">
        <thead>
          <tr>
            <th className="col-fixed">DEF</th>
            <th>Severidad</th>
            <th>Estado</th>
            <th>REQ</th>
            <th>Llegó a producción</th>
            <th>Abierto</th>
            <th className="num">Después de producción</th>
          </tr>
        </thead>
        <tbody>
          {items.map((row) => (
            <tr key={row.defect.code}>
              <td className="col-fixed">
                <div className="cell-stack">
                  <Code code={row.defect.code} panel />
                  <span className="muted">{row.defect.title}</span>
                </div>
              </td>
              <td>
                <SeverityChip severity={row.severity} />
              </td>
              <td>
                <StateChip state={row.state} />
              </td>
              <td>
                <Code code={row.requirement} panel />
              </td>
              <td>
                <span className="cell-line">
                  <Code code={row.package} /> {formatDate(row.releasedAt)}
                </span>
              </td>
              <td>{formatDate(row.createdAt)}</td>
              <td className="num">{formatDecimalDays(row.daysAfterRelease)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export default function EscapedDefectsPage() {
  const metrics = useEscapedDefectMetrics(useMetricsQuery(['system', 'severity', 'from', 'to']))
  return (
    <>
      <MetricsHead
        title="Defectos escapados a producción"
        subtitle="Un defecto escapó si se abrió después de que su REQ llegó a producción: después del primer despliegue exitoso, en un ambiente de producción, de un PKG que contiene el REQ."
      />
      <MetricsFilters severity period />
      <Async query={metrics}>
        {(data) => (
          <>
            <PeriodNote period={data.period} what="defectos creados en esas fechas" />
            <Summary summary={data.summary} />
            <Months months={data.months} />
            <h2 className="h2">Escapados</h2>
            <EscapedTable items={data.items} />
            <Pager page={data.page} size={data.size} total={data.total} noun="escapados" />
          </>
        )}
      </Async>
    </>
  )
}
