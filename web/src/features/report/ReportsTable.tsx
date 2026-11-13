// Tabla de informes de conformidad (RPT): cada versión con su hash y si sigue vigente.
import type { Report } from '../../shared/api/types.ts'
import { formatDateTime, shortChecksum } from '../../shared/lib/format.ts'
import { Chip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Empty } from '../../shared/ui/feedback.tsx'

export function ReportStatus({ report }: { report: Report }) {
  if (report.supersededBy) {
    return (
      <Chip tone="neutro" icon="minus">
        Reemplazado
      </Chip>
    )
  }
  return (
    <Chip tone="ok" icon="check">
      Vigente
    </Chip>
  )
}

export function ReportsTable({ items, showPackage, empty }: { items: Report[]; showPackage?: boolean; empty: string }) {
  if (items.length === 0) return <Empty>{empty}</Empty>
  return (
    <div className="surface table-wrap">
      <table className="table" aria-label="Informes de conformidad">
        <thead>
          <tr>
            <th className="col-fixed">RPT</th>
            {showPackage && <th>Paquete</th>}
            <th className="num">Versión</th>
            <th>Vigencia</th>
            <th>SHA-256</th>
            <th>Emitió</th>
          </tr>
        </thead>
        <tbody>
          {items.map((report) => (
            <tr key={report.code}>
              <td className="col-fixed">
                <Code code={report.code} />
              </td>
              {showPackage && (
                <td>
                  <span className="cell-line">
                    <Code code={report.package.code} /> {report.package.title}
                  </span>
                </td>
              )}
              <td className="num">v{report.version}</td>
              <td>
                <span className="cell-line">
                  <ReportStatus report={report} />
                  {report.supersededBy && <Code code={report.supersededBy} />}
                </span>
              </td>
              <td>
                <span className="mono" title={report.checksum}>
                  {shortChecksum(report.checksum)}
                </span>
              </td>
              <td>
                {report.issuedBy.displayName} · {formatDateTime(report.createdAt)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
