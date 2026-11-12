// Tabla de defectos: la lista general y la pestaña Defectos de la ficha del REQ.
import type { Defect } from '../../shared/api/types.ts'
import { formatDays } from '../../shared/lib/format.ts'
import { SeverityChip, StateChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Empty } from '../../shared/ui/feedback.tsx'

export function DefectsTable({ items, showRequirement }: { items: Defect[]; showRequirement?: boolean }) {
  if (items.length === 0) return <Empty>No hay defectos{showRequirement ? ' con este filtro' : ''}.</Empty>
  return (
    <div className="surface table-wrap">
      <table className="table">
        <thead>
          <tr>
            <th className="col-fixed">DEF</th>
            <th>Defecto</th>
            <th>Severidad</th>
            <th>Estado</th>
            <th className="num">En estado</th>
            {showRequirement && <th>REQ</th>}
            <th>RUN</th>
          </tr>
        </thead>
        <tbody>
          {items.map((defect) => (
            <tr key={defect.code}>
              <td className="col-fixed">
                <Code code={defect.code} />
              </td>
              <td>{defect.title}</td>
              <td>
                <SeverityChip severity={defect.severity} />
              </td>
              <td>
                <StateChip state={defect.state} />
              </td>
              <td className="num">{formatDays(defect.daysInState)}</td>
              {showRequirement && (
                <td>
                  <Code code={defect.requirement} />
                </td>
              )}
              <td>
                <Code code={defect.run} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
