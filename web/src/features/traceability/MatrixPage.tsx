// Matriz de trazabilidad (boceto: matriz.html). Una fila por REQ y TC; las celdas del REQ se
// agrupan. Clic en un ID abre su ficha en el panel lateral sin perder la posición.
import { useState } from 'react'
import { useUserOptions, useSystems, useWorkflow } from '../../shared/api/session.ts'
import type { Matrix, MatrixRow } from '../../shared/api/types.ts'
import { RISK_LABELS, options } from '../../shared/lib/labels.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { ResultChip, RiskChip, SeverityChip, StateChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterInput, FilterSelect, Pager } from '../../shared/ui/Filters.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { matrixExportUrl, useChangeOptions, useMatrix } from './api.ts'

const FILTERS = ['change', 'system', 'state', 'risk', 'owner', 'from', 'to']

function Indicators({ indicators }: { indicators: Matrix['indicators'] }) {
  const { high, medium, low } = indicators.openDefects
  return (
    <section className="glass kpis" aria-label="Indicadores del filtro actual">
      <div className="kpi">
        <span className="kpi__label">Cobertura</span>
        <span className="kpi__value">{indicators.coveragePercent} %</span>
        <span className="kpi__note">
          {indicators.passedTestCases} de {indicators.requiredTestCases} TC requeridos con RUN aprobado
        </span>
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

/** Cuántas filas seguidas comparten el REQ de la fila `index` (0 si no es la primera del grupo). */
function groupSpan(rows: MatrixRow[], index: number): number {
  const code = rows[index]?.requirement.code
  if (index > 0 && rows[index - 1]?.requirement.code === code) return 0
  let span = 1
  while (rows[index + span]?.requirement.code === code) span++
  return span
}

function MatrixTable({ rows, systemNames }: { rows: MatrixRow[]; systemNames: Record<string, string> }) {
  if (rows.length === 0) return <Empty>No hay requerimientos con este filtro.</Empty>
  return (
    <section className="surface table-wrap" aria-label="Matriz">
      <table className="table">
        <thead>
          <tr>
            <th className="col-fixed">REQ</th>
            <th>Estado</th>
            <th>RISK</th>
            <th>UC</th>
            <th>TC</th>
            <th>RUN · BUILD · ENV</th>
            <th>DEF</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row, index) => {
            const span = groupSpan(rows, index)
            const run = row.testCase?.lastRun
            return (
              <tr key={`${row.requirement.code}-${row.testCase?.code ?? 'sin-tc'}`}>
                {span > 0 && (
                  <>
                    <td className="col-fixed" rowSpan={span}>
                      <div className="cell-stack">
                        <Code code={row.requirement.code} panel />
                        <span className="muted">{systemNames[row.system] ?? row.system}</span>
                      </div>
                    </td>
                    <td rowSpan={span}>
                      <StateChip state={row.state} />
                    </td>
                    <td rowSpan={span}>{row.riskLevel ? <RiskChip level={row.riskLevel} /> : <span className="muted">Sin evaluar</span>}</td>
                  </>
                )}
                <td>
                  <span className="cell-stack">
                    {row.functionalCases.length === 0 ? <span className="muted">Sin UC</span> : row.functionalCases.map((code) => <Code key={code} code={code} />)}
                  </span>
                </td>
                <td>
                  {row.testCase ? (
                    <div className="cell-stack">
                      <Code code={row.testCase.code} />
                      {row.testCase.isCritical && <span className="muted">Crítico</span>}
                    </div>
                  ) : (
                    <span className="muted">Sin TC</span>
                  )}
                </td>
                <td>
                  {run ? (
                    <div className="cell-stack">
                      <span className="cell-line">
                        <Code code={run.code} />
                        <ResultChip result={run.result} />
                      </span>
                      <span className="muted mono">
                        {run.buildVersion} · {run.environmentName}
                      </span>
                    </div>
                  ) : (
                    <span className="muted">Sin RUN</span>
                  )}
                </td>
                <td>
                  {row.defects.length === 0 ? (
                    <span className="muted">Ninguno</span>
                  ) : (
                    <div className="cell-stack">
                      {row.defects.map((defect) => (
                        <span key={defect.code} className="cell-line">
                          <Code code={defect.code} panel />
                          <SeverityChip severity={defect.severity} />
                        </span>
                      ))}
                    </div>
                  )}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </section>
  )
}

export default function MatrixPage() {
  const users = useUserOptions()
  const { params, pick } = useUrlFilters()
  const [copied, setCopied] = useState(false)
  const systems = useSystems().data?.items ?? []
  const states = useWorkflow('requirement').data?.states ?? []
  const changes = useChangeOptions().data?.items ?? []
  const filters = pick(FILTERS)
  const page = Number(params.get('page')) || 1
  const matrix = useMatrix({ ...filters, page: page > 1 ? page : undefined })

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(window.location.href)
      setCopied(true)
    } catch {
      setCopied(false)
    }
  }

  return (
    <>
      <Crumbs items={[{ label: 'Pruebas' }, { label: 'Matriz de trazabilidad' }]} />
      <PageHead
        title="Matriz de trazabilidad"
        subtitle="REQ → RISK → UC → TC → RUN → DEF"
        actions={
          <>
            <a className="btn" href={matrixExportUrl('csv', filters)} download>
              <Icon name="download" />
              Exportar CSV
            </a>
            <a className="btn" href={matrixExportUrl('xlsx', filters)} download>
              <Icon name="download" />
              Exportar XLSX
            </a>
          </>
        }
      />
      {matrix.data && <Indicators indicators={matrix.data.indicators} />}
      <FilterBar
        actions={
          <button className="btn" type="button" onClick={() => void copyLink()}>
            <Icon name={copied ? 'check' : 'link'} />
            {copied ? 'Enlace copiado' : 'Copiar enlace'}
          </button>
        }
      >
        <FilterSelect name="change" label="CHG" options={changes.map((change) => ({ value: change.code, label: change.code }))} />
        <FilterSelect name="system" label="Sistema" options={systems.map((system) => ({ value: system.key, label: system.name }))} />
        <FilterSelect name="state" label="Estado" options={states.map((state) => ({ value: state.key, label: state.name }))} />
        <FilterSelect name="risk" label="Riesgo" options={options(RISK_LABELS)} />
        <FilterSelect name="owner" label="Responsable" options={users} />
        <FilterInput name="from" label="Desde" type="date" />
        <FilterInput name="to" label="Hasta" type="date" />
      </FilterBar>
      <Async query={matrix}>
        {(data) => (
          <>
            <MatrixTable rows={data.items} systemNames={Object.fromEntries(systems.map((system) => [system.key, system.name]))} />
            <Pager page={data.page} size={data.size} total={data.total} noun="filas (una por REQ y TC)" />
          </>
        )}
      </Async>
    </>
  )
}
