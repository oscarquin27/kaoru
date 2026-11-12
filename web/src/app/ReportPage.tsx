// Ficha de un informe de conformidad (RPT): el Markdown tal como se emitió, su hash, la cadena de
// versiones y las descargas. Junta informes, documentos (el visor de Markdown) y auditoría: vive en app/.
import { useParams } from 'react-router'
import { Markdown } from '../features/documents/Markdown.tsx'
import { reportExportUrl, useReport } from '../features/report/api.ts'
import { ReportStatus } from '../features/report/ReportsTable.tsx'
import type { Report } from '../shared/api/types.ts'
import { formatDateTime } from '../shared/lib/format.ts'
import { Code } from '../shared/ui/Code.tsx'
import { Async, PageSkeleton } from '../shared/ui/feedback.tsx'
import { Icon } from '../shared/ui/Icon.tsx'
import { Crumbs, MetaItem } from '../shared/ui/layout.tsx'
import { Tabs } from '../shared/ui/Tabs.tsx'
import { Activity } from './Activity.tsx'

function ReportView({ report }: { report: Report }) {
  const file = `informe-${report.code}-v${report.version}`
  return (
    <>
      <section className="glass pad stack" aria-labelledby="rpt-title">
        <div className="row row--between">
          <div className="stack-s">
            <span className="id">{report.code}</span>
            <h1 className="title" id="rpt-title">
              Informe de conformidad de {report.package.code}
            </h1>
          </div>
          <ReportStatus report={report} />
        </div>
        <dl className="meta">
          <MetaItem label="Paquete">
            <Code code={report.package.code} /> {report.package.title}
          </MetaItem>
          <MetaItem label="Versión">v{report.version}</MetaItem>
          <MetaItem label="Emitió">
            {report.issuedBy.displayName} · {formatDateTime(report.createdAt)}
          </MetaItem>
          <MetaItem label="Reemplaza a">{report.previous ? <Code code={report.previous} /> : <span className="muted">Es la primera versión</span>}</MetaItem>
          <MetaItem label="SHA-256 del Markdown">
            <span className="mono wrap">{report.checksum}</span>
          </MetaItem>
        </dl>
        {report.supersededBy && (
          <div className="notice" role="note">
            <Icon name="info" />
            <div className="notice__main">
              Este informe fue reemplazado por <Code code={report.supersededBy} />. Se conserva tal como se emitió.
            </div>
          </div>
        )}
        <div className="row">
          <a className="btn btn--primary" href={reportExportUrl(report.code, 'pdf')} download={`${file}.pdf`}>
            <Icon name="download" />
            Descargar PDF
          </a>
          <a className="btn" href={reportExportUrl(report.code, 'markdown')} download={`${file}.md`}>
            <Icon name="download" />
            Descargar Markdown
          </a>
        </div>
        <p className="soft">El informe no se edita. El PDF se genera del mismo Markdown, que es lo que firma el hash.</p>
      </section>
      <Tabs
        label="Secciones del informe"
        tabs={[
          {
            id: 'informe',
            label: 'Informe',
            render: () => (
              <div className="surface pad">
                <Markdown source={report.content ?? ''} />
              </div>
            ),
          },
          { id: 'actividad', label: 'Actividad', render: () => <Activity entityType="conformance_report" entityCode={report.code} /> },
        ]}
      />
    </>
  )
}

export default function ReportPage() {
  const { code = '' } = useParams()
  const report = useReport(code)
  return (
    <>
      <Crumbs items={[{ label: 'Informes', to: '/informes' }, { label: code, code: true }]} />
      <Async query={report} skeleton={<PageSkeleton />}>
        {(data) => <ReportView report={data} />}
      </Async>
    </>
  )
}
