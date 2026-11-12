// Ficha de un diagnóstico posterior al despliegue (DIAG): lo que se entregó, la revisión de QA, la
// evidencia y la actividad. Junta producción, documentos y auditoría, por eso vive en app/.
import { useState } from 'react'
import { useParams } from 'react-router'
import { DocumentsTab } from '../features/documents/DocumentsTab.tsx'
import { useEntityDocuments } from '../features/documents/api.ts'
import { useDiagnostic } from '../features/production/api.ts'
import { ReviewForm } from '../features/production/forms.tsx'
import { useCan } from '../shared/api/session.ts'
import type { Diagnostic } from '../shared/api/types.ts'
import { formatDateTime } from '../shared/lib/format.ts'
import { DIAGNOSTIC_RESULT_LABELS } from '../shared/lib/labels.ts'
import { DiagnosticChip } from '../shared/ui/chips.tsx'
import { Code } from '../shared/ui/Code.tsx'
import { Async, PageSkeleton } from '../shared/ui/feedback.tsx'
import { Icon } from '../shared/ui/Icon.tsx'
import { Crumbs, MetaItem } from '../shared/ui/layout.tsx'
import { Dialog } from '../shared/ui/Modal.tsx'
import { Tabs } from '../shared/ui/Tabs.tsx'
import { Activity } from './Activity.tsx'

function Review({ diagnostic }: { diagnostic: Diagnostic }) {
  const can = useCan()
  const [reviewing, setReviewing] = useState(false)
  const review = diagnostic.review
  return (
    <section className="surface pad stack" aria-labelledby="diag-review">
      <div className="row row--between">
        <h2 className="h3" id="diag-review">
          Revisión de QA
        </h2>
        {!review && can('diag.review') && (
          <button className="btn btn--primary" type="button" onClick={() => setReviewing(true)}>
            <Icon name="check" />
            Revisar
          </button>
        )}
      </div>
      {review ? (
        <>
          <p>
            <strong>{DIAGNOSTIC_RESULT_LABELS[review.result]}</strong> · {review.reviewedBy.displayName} · {formatDateTime(review.reviewedAt)}
          </p>
          {review.notes && <p className="wrap">{review.notes}</p>}
          <p className="soft">La revisión no se cambia. Si hace falta otra, dev u ops entregan un DIAG nuevo.</p>
        </>
      ) : (
        <p className="muted">Todavía nadie de QA lo revisó.</p>
      )}
      {reviewing && (
        <Dialog title={`Revisar ${diagnostic.code}`} onClose={() => setReviewing(false)}>
          <ReviewForm diagnostic={diagnostic} onClose={() => setReviewing(false)} />
        </Dialog>
      )}
    </section>
  )
}

function DiagnosticView({ diagnostic }: { diagnostic: Diagnostic }) {
  const can = useCan()
  const documents = useEntityDocuments('diagnostic', diagnostic.code)
  return (
    <>
      <section className="glass pad stack" aria-labelledby="diag-title">
        <div className="row row--between">
          <div className="stack-s">
            <span className="id">{diagnostic.code}</span>
            <h1 className="title" id="diag-title">
              Diagnóstico de {diagnostic.package.code}
            </h1>
          </div>
          <DiagnosticChip review={diagnostic.review} />
        </div>
        <dl className="meta">
          <MetaItem label="Paquete">
            <Code code={diagnostic.package.code} /> {diagnostic.package.title}
          </MetaItem>
          <MetaItem label="Despliegue">
            <Code code={diagnostic.deployment.code} /> · {diagnostic.deployment.environment.name} · {formatDateTime(diagnostic.deployment.executedAt)}
          </MetaItem>
          <MetaItem label="Entregó">
            {diagnostic.submittedBy.displayName} · {formatDateTime(diagnostic.createdAt)}
          </MetaItem>
        </dl>
      </section>
      <section className="surface pad stack">
        <h2 className="h3">Notas</h2>
        <p className="wrap">{diagnostic.notes || <span className="muted">Sin notas.</span>}</p>
      </section>
      <Review diagnostic={diagnostic} />
      <Tabs
        label="Secciones del diagnóstico"
        tabs={[
          {
            id: 'evidencia',
            label: 'Evidencia',
            count: documents.data?.items.length,
            render: () => <DocumentsTab entityType="diagnostic" entityCode={diagnostic.code} canUpload={can('doc.upload')} />,
          },
          { id: 'actividad', label: 'Actividad', render: () => <Activity entityType="diagnostic" entityCode={diagnostic.code} /> },
        ]}
      />
    </>
  )
}

export default function DiagnosticPage() {
  const { code = '' } = useParams()
  const diagnostic = useDiagnostic(code)
  return (
    <>
      <Crumbs items={[{ label: 'Producción', to: '/produccion' }, { label: 'Diagnósticos', to: '/produccion/diagnosticos' }, { label: code, code: true }]} />
      <Async query={diagnostic} skeleton={<PageSkeleton />}>
        {(data) => <DiagnosticView diagnostic={data} />}
      </Async>
    </>
  )
}
