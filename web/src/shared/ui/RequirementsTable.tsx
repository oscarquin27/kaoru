// Tabla de REQ. La usan la lista de requerimientos, la ficha del cambio y el inicio.
import type { Requirement } from '../api/types.ts'
import { formatDate } from '../lib/format.ts'
import { StateChip } from './chips.tsx'
import { Code } from './Code.tsx'
import { Empty } from './feedback.tsx'

export function RequirementsTable({ items }: { items: Requirement[] }) {
  if (items.length === 0) return <Empty>No hay requerimientos con este filtro.</Empty>
  return (
    <div className="surface table-wrap">
      <table className="table">
        <thead>
          <tr>
            <th className="col-fixed">REQ</th>
            <th>Requerimiento</th>
            <th>Sistema</th>
            <th>Cambio</th>
            <th>Estado</th>
            <th>Responsable</th>
            <th>Actualizado</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.code}>
              <td className="col-fixed">
                <Code code={item.code} />
              </td>
              <td>{item.title}</td>
              <td>{item.system.name}</td>
              <td>{item.change ? <Code code={item.change.code} /> : <span className="muted">Sin cambio</span>}</td>
              <td>
                <StateChip state={item.state} />
              </td>
              <td>{item.owner?.displayName ?? <span className="muted">Sin asignar</span>}</td>
              <td className="num">{formatDate(item.updatedAt)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
