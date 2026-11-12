// Pestaña Trazabilidad de la ficha del REQ: la cadena y la tabla UC → TC → RUN → DEF.
import { Fragment, type ReactNode } from 'react'
import type { RequirementOverview } from '../../shared/api/types.ts'
import { ResultChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Empty } from '../../shared/ui/feedback.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'

interface TraceTabProps {
  overview: RequirementOverview
  /** Qué TC valida cada UC (lo sabe el diseño de pruebas). */
  functionalCases: { code: string; testCases: string[] }[]
  /** En qué RUN se reprodujo cada DEF (lo saben los defectos). */
  defects: { code: string; run: string }[]
}

export function TraceTab({ overview, functionalCases, defects }: TraceTabProps) {
  const { requirement, testCases } = overview
  const runs = testCases.filter((testCase) => testCase.lastRun).length
  const nodes: [label: string, value: ReactNode][] = [
    ['Cambio', requirement.change ? <Code code={requirement.change.code} /> : <span className="muted">Sin cambio</span>],
    ['Requerimiento', <span className="id">{requirement.code}</span>],
    ['Riesgo', requirement.risk ? <span className="id">{requirement.risk.code}</span> : <span className="muted">Sin evaluar</span>],
    ['Casos de uso', `${overview.functionalCases.length} UC`],
    ['Casos de prueba', `${testCases.length} TC`],
    ['Ejecuciones', `${runs} TC con RUN`],
    ['Defectos', `${overview.defects.length} DEF`],
  ]
  // Una fila por UC y TC; un UC sin TC ocupa una fila sola.
  const rows: { useCase: string; testCase?: string }[] = functionalCases.flatMap((useCase) =>
    useCase.testCases.length === 0 ? [{ useCase: useCase.code }] : useCase.testCases.map((testCase) => ({ useCase: useCase.code, testCase })),
  )

  return (
    <div className="stack">
      <div className="surface pad">
        <div className="chain">
          {nodes.map(([label, value], index) => (
            <Fragment key={label}>
              {index > 0 && <Icon name="arrow-right" />}
              <div className="chain__node">
                <span className="h3">{label}</span>
                <span>{value}</span>
              </div>
            </Fragment>
          ))}
        </div>
      </div>
      {rows.length === 0 ? (
        <Empty>Sin casos de uso todavía: la cadena empieza al definir el primero.</Empty>
      ) : (
        <div className="surface table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>UC</th>
                <th>TC</th>
                <th>Último RUN</th>
                <th>BUILD</th>
                <th>ENV</th>
                <th>DEF</th>
              </tr>
            </thead>
            <tbody>
              {rows.map(({ useCase, testCase }) => {
                const run = testCases.find((item) => item.code === testCase)?.lastRun
                const found = defects.filter((defect) => defect.run === run?.code)
                return (
                  <tr key={`${useCase}-${testCase}`}>
                    <td>
                      <Code code={useCase} />
                    </td>
                    <td>{testCase ? <Code code={testCase} /> : <span className="muted">Sin TC</span>}</td>
                    <td>
                      {run ? (
                        <span className="cell-line">
                          <Code code={run.code} />
                          <ResultChip result={run.result} />
                        </span>
                      ) : (
                        <span className="muted">Sin RUN</span>
                      )}
                    </td>
                    <td>{run ? <span className="mono">{run.buildVersion}</span> : <span className="muted">—</span>}</td>
                    <td>{run ? run.environmentName : <span className="muted">—</span>}</td>
                    <td>
                      <span className="cell-line">
                        {found.length === 0 ? <span className="muted">Ninguno</span> : found.map((defect) => <Code key={defect.code} code={defect.code} />)}
                      </span>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
