// Tablas de la verificación posterior al despliegue: diagnósticos (DIAG) y pruebas de humo (SMOKE).
import type { Diagnostic, SmokeTest } from '../../shared/api/types.ts'
import { formatDateTime } from '../../shared/lib/format.ts'
import { DiagnosticChip, SmokeChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Empty } from '../../shared/ui/feedback.tsx'

export function DiagnosticsTable({ items, showPackage, empty }: { items: Diagnostic[]; showPackage?: boolean; empty: string }) {
  if (items.length === 0) return <Empty>{empty}</Empty>
  return (
    <div className="surface table-wrap">
      <table className="table" aria-label="Diagnósticos">
        <thead>
          <tr>
            <th className="col-fixed">DIAG</th>
            {showPackage && <th>Paquete</th>}
            <th>Despliegue</th>
            <th>Revisión de QA</th>
            <th>Notas</th>
            <th>Entregó</th>
          </tr>
        </thead>
        <tbody>
          {items.map((diagnostic) => (
            <tr key={diagnostic.code}>
              <td className="col-fixed">
                <Code code={diagnostic.code} />
              </td>
              {showPackage && (
                <td>
                  <Code code={diagnostic.package.code} />
                </td>
              )}
              <td>
                <span className="cell-line">
                  <Code code={diagnostic.deployment.code} /> {diagnostic.deployment.environment.name}
                </span>
              </td>
              <td>
                <DiagnosticChip review={diagnostic.review} />
              </td>
              <td className="wrap">{diagnostic.notes || <span className="muted">Sin notas</span>}</td>
              <td>
                {diagnostic.submittedBy.displayName} · {formatDateTime(diagnostic.createdAt)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function SmokeTestsTable({ items, showPackage, empty }: { items: SmokeTest[]; showPackage?: boolean; empty: string }) {
  if (items.length === 0) return <Empty>{empty}</Empty>
  return (
    <div className="surface table-wrap">
      <table className="table" aria-label="Pruebas de humo">
        <thead>
          <tr>
            <th className="col-fixed">SMOKE</th>
            {showPackage && <th>Paquete</th>}
            <th>Despliegue</th>
            <th>Resultado</th>
            <th>Notas</th>
            <th>Registró</th>
          </tr>
        </thead>
        <tbody>
          {items.map((smoke) => (
            <tr key={smoke.code}>
              <td className="col-fixed">
                <Code code={smoke.code} />
              </td>
              {showPackage && (
                <td>
                  <Code code={smoke.package.code} />
                </td>
              )}
              <td>
                <span className="cell-line">
                  <Code code={smoke.deployment.code} /> {smoke.deployment.environment.name}
                </span>
              </td>
              <td>
                <SmokeChip result={smoke.result} />
              </td>
              <td className="wrap">{smoke.notes || <span className="muted">Sin notas</span>}</td>
              <td>
                {smoke.executedBy.displayName} · {formatDateTime(smoke.createdAt)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
