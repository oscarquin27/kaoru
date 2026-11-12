// Tabla de despliegues (DEP), del más reciente al más antiguo.
import type { Deployment } from '../../shared/api/types.ts'
import { formatDateTime } from '../../shared/lib/format.ts'
import { DeploymentResultChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Empty } from '../../shared/ui/feedback.tsx'

/** Lo desplegado: un BUILD con su versión o un PKG con su título. */
export function Deployed({ build, bundle }: { build?: Deployment['build']; bundle?: Deployment['package'] }) {
  if (build) {
    return (
      <span className="cell-line">
        <Code code={build.code} /> <span className="mono">{build.version}</span>
      </span>
    )
  }
  if (bundle) {
    return (
      <span className="cell-line">
        <Code code={bundle.code} /> {bundle.title}
      </span>
    )
  }
  return <span className="muted">—</span>
}

export function DeploymentsTable({ items, showEnvironment, empty }: { items: Deployment[]; showEnvironment?: boolean; empty: string }) {
  if (items.length === 0) return <Empty>{empty}</Empty>
  return (
    <div className="surface table-wrap">
      <table className="table" aria-label="Despliegues">
        <thead>
          <tr>
            <th className="col-fixed">DEP</th>
            {showEnvironment && <th>Ambiente</th>}
            <th>Desplegado</th>
            <th>Resultado</th>
            <th>Notas</th>
            <th>Desplegó</th>
          </tr>
        </thead>
        <tbody>
          {items.map((deployment) => (
            <tr key={deployment.code}>
              <td className="col-fixed">
                <Code code={deployment.code} />
              </td>
              {showEnvironment && (
                <td>
                  <span className="cell-line">
                    <Code code={deployment.environment.code} /> {deployment.environment.name}
                  </span>
                </td>
              )}
              <td>
                <Deployed build={deployment.build} bundle={deployment.package} />
              </td>
              <td>
                <DeploymentResultChip result={deployment.result} />
              </td>
              <td className="wrap">{deployment.notes || <span className="muted">Sin notas</span>}</td>
              <td>
                {deployment.executedBy.displayName} · {formatDateTime(deployment.executedAt)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
