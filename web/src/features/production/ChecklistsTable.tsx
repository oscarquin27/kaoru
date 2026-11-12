// Tabla de checklists de producción (CHK), del más reciente al más antiguo.
import type { Checklist } from '../../shared/api/types.ts'
import { formatDateTime } from '../../shared/lib/format.ts'
import { ChecklistChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Empty } from '../../shared/ui/feedback.tsx'

export function ChecklistsTable({ items, showPackage, empty }: { items: Checklist[]; showPackage?: boolean; empty: string }) {
  if (items.length === 0) return <Empty>{empty}</Empty>
  return (
    <div className="surface table-wrap">
      <table className="table" aria-label="Checklists de producción">
        <thead>
          <tr>
            <th className="col-fixed">CHK</th>
            {showPackage && <th>Paquete</th>}
            <th>Ambiente</th>
            <th>Avance</th>
            <th>Plantilla</th>
            <th>Abrió</th>
          </tr>
        </thead>
        <tbody>
          {items.map((checklist) => (
            <tr key={checklist.code}>
              <td className="col-fixed">
                <Code code={checklist.code} />
              </td>
              {showPackage && (
                <td>
                  <span className="cell-line">
                    <Code code={checklist.package.code} /> {checklist.package.title}
                  </span>
                </td>
              )}
              <td>{checklist.environment.name}</td>
              <td>
                <ChecklistChip signed={checklist.signedSteps} total={checklist.steps.length} />
              </td>
              <td className="num">v{checklist.templateVersion}</td>
              <td>
                {checklist.startedBy.displayName} · {formatDateTime(checklist.createdAt)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
