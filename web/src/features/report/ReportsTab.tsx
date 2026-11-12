// Pestaña «Informes» del PKG: sus versiones del informe de conformidad y el botón para emitir otra.
import { useNavigate } from 'react-router'
import { useCan } from '../../shared/api/session.ts'
import { Async, ErrorNotice } from '../../shared/ui/feedback.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { useIssueReport, useReports } from './api.ts'
import { ReportsTable } from './ReportsTable.tsx'

export function ReportsTab({ packageCode }: { packageCode: string }) {
  const can = useCan()
  const navigate = useNavigate()
  const reports = useReports({ package: packageCode, size: 50 })
  const issue = useIssueReport()
  const hasReport = (reports.data?.items.length ?? 0) > 0

  async function onIssue() {
    const report = await issue.mutateAsync(packageCode).catch(() => undefined)
    if (report) void navigate(`/informes/${report.code}`)
  }

  return (
    <div className="stack">
      <div className="row row--between">
        <p className="soft">
          El informe congela la evidencia del PKG tal como está al emitirlo. {hasReport ? 'Emitir otro crea una versión nueva que reemplaza a la vigente; las anteriores se conservan.' : ''}
        </p>
        {can('rpt.issue') && (
          <button className="btn btn--primary" type="button" disabled={issue.isPending} onClick={() => void onIssue()}>
            <Icon name="file" />
            {hasReport ? 'Emitir nueva versión' : 'Emitir informe'}
          </button>
        )}
      </div>
      {issue.error && <ErrorNotice error={issue.error} />}
      <Async query={reports}>{(data) => <ReportsTable items={data.items} empty="El PKG no tiene informes de conformidad." />}</Async>
    </div>
  )
}
