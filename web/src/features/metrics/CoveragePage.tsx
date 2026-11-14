// Cobertura REQ por REQ (requerimiento §10): cuántos TC exige cada REQ, cuántos tienen su último RUN
// aprobado y si eso alcanza el mínimo que fija la regla para su riesgo. Primero lo que hay que atender.
import { useChangeOptions, usePackageOptions } from '../../shared/api/catalog.ts'
import { useWorkflow } from '../../shared/api/session.ts'
import type { CoverageMetrics, CoverageRow, CoverageStatus } from '../../shared/api/types.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Chip, RiskChip, StateChip, type Tone } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { FilterSelect, Pager } from '../../shared/ui/Filters.tsx'
import type { IconName } from '../../shared/ui/Icon.tsx'
import { Meter } from '../../shared/ui/layout.tsx'
import { useCoverageMetrics } from './api.ts'
import { MetricsFilters, MetricsHead, share, useMetricsQuery } from './common.tsx'

const STATUS: Record<CoverageStatus, { label: string; tone: Tone; icon: IconName }> = {
  below: { label: 'Bajo el mínimo', tone: 'aviso', icon: 'triangle' },
  no_test_cases: { label: 'Sin TC', tone: 'neutro', icon: 'circle' },
  unassessed: { label: 'Sin riesgo', tone: 'neutro', icon: 'circle' },
  meets: { label: 'Cumple', tone: 'ok', icon: 'check' },
}

function CoverageStatusChip({ status }: { status: CoverageStatus }) {
  const look = STATUS[status]
  return (
    <Chip tone={look.tone} icon={look.icon}>
      {look.label}
    </Chip>
  )
}

function Summary({ summary }: { summary: CoverageMetrics['summary'] }) {
  const percent = summary.requiredTestCases === 0 ? 0 : Math.floor((summary.passedTestCases * 100) / summary.requiredTestCases)
  return (
    <section className="glass kpis" aria-label="Resumen de cobertura">
      <div className="kpi">
        <span className="kpi__label">Cobertura del filtro</span>
        <span className="kpi__value">{percent} %</span>
        <span className="kpi__note">
          {summary.passedTestCases} de {summary.requiredTestCases} TC con su último RUN aprobado
        </span>
      </div>
      <div className="kpi">
        <span className="kpi__label">REQ bajo el mínimo</span>
        <span className="kpi__value">{summary.below}</span>
        <span className="kpi__note">{share(summary.below, summary.requirements)} REQ</span>
      </div>
      <div className="kpi">
        <span className="kpi__label">Sin TC o sin riesgo</span>
        <span className="kpi__value">{summary.noTestCases + summary.unassessed}</span>
        <span className="kpi__note">
          {summary.noTestCases} sin TC · {summary.unassessed} sin riesgo evaluado
        </span>
      </div>
      <div className="kpi">
        <span className="kpi__label">Cumplen</span>
        <span className="kpi__value">{summary.meets}</span>
        <span className="kpi__note">{share(summary.meets, summary.requirements)} REQ</span>
      </div>
    </section>
  )
}

function CoverageTable({ rows }: { rows: CoverageRow[] }) {
  if (rows.length === 0) return <Empty>No hay requerimientos con este filtro.</Empty>
  return (
    <div className="surface table-wrap">
      <table className="table" aria-label="Cobertura por REQ">
        <thead>
          <tr>
            <th className="col-fixed">REQ</th>
            <th>Estado</th>
            <th>Riesgo</th>
            <th className="num">TC aprobados</th>
            <th>Cobertura</th>
            <th className="num">Mínimo</th>
            <th>Resultado</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.requirement.code}>
              <td className="col-fixed">
                <div className="cell-stack">
                  <Code code={row.requirement.code} panel />
                  <span className="muted">{row.requirement.title}</span>
                </div>
              </td>
              <td>
                <StateChip state={row.state} />
              </td>
              <td>{row.riskLevel ? <RiskChip level={row.riskLevel} /> : <span className="muted">Sin evaluar</span>}</td>
              <td className="num">
                {row.passed} de {row.required}
              </td>
              <td>
                {row.required === 0 ? (
                  <span className="muted">—</span>
                ) : (
                  <span className="cell-line">
                    <Meter percent={row.percent} />
                    <span className="num">{row.percent} %</span>
                  </span>
                )}
              </td>
              <td className="num">{row.minimumPercent === undefined ? <span className="muted">—</span> : `${row.minimumPercent} %`}</td>
              <td>
                <CoverageStatusChip status={row.status} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

const FILTERS = ['system', 'change', 'package', 'state', 'status']

export default function CoveragePage() {
  const { params } = useUrlFilters()
  const changes = useChangeOptions().data?.items ?? []
  const packages = usePackageOptions(params.get('system') ?? undefined).data?.items ?? []
  const states = useWorkflow('requirement').data?.states ?? []
  const coverage = useCoverageMetrics(useMetricsQuery(FILTERS))

  return (
    <>
      <MetricsHead
        title="Cobertura por REQ"
        subtitle="TC que exige cada REQ, cuántos tienen su último RUN aprobado y el mínimo que pide su riesgo. Primero lo que está por debajo."
      />
      <MetricsFilters>
        <FilterSelect name="change" label="CHG" options={changes.map((change) => ({ value: change.code, label: change.code }))} />
        <FilterSelect name="package" label="PKG" options={packages.map((item) => ({ value: item.code, label: `${item.code} · ${item.title}` }))} />
        <FilterSelect name="state" label="Estado" options={states.map((state) => ({ value: state.key, label: state.name }))} />
        <FilterSelect name="status" label="Resultado" options={Object.entries(STATUS).map(([value, look]) => ({ value, label: look.label }))} />
      </MetricsFilters>
      <Async query={coverage}>
        {(data) => (
          <>
            <Summary summary={data.summary} />
            <CoverageTable rows={data.items} />
            <Pager page={data.page} size={data.size} total={data.total} noun="REQ" />
          </>
        )}
      </Async>
    </>
  )
}
