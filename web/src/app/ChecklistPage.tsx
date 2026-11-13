// Ficha de un checklist de producción (CHK): sus pasos con quién firmó y cuándo, la evidencia y la
// actividad. Junta producción, documentos y auditoría, por eso vive en app/.
import { useState } from 'react'
import { useParams } from 'react-router'
import { DocumentsTab } from '../features/documents/DocumentsTab.tsx'
import { useEntityDocuments } from '../features/documents/api.ts'
import { useChecklist } from '../features/production/api.ts'
import { SignStepForm } from '../features/production/forms.tsx'
import { useCan } from '../shared/api/session.ts'
import type { Checklist, ChecklistStep } from '../shared/api/types.ts'
import { formatDateTime } from '../shared/lib/format.ts'
import { ENVIRONMENT_KIND_LABELS } from '../shared/lib/labels.ts'
import { ChecklistChip } from '../shared/ui/chips.tsx'
import { Code } from '../shared/ui/Code.tsx'
import { Async, PageSkeleton } from '../shared/ui/feedback.tsx'
import { Icon } from '../shared/ui/Icon.tsx'
import { Crumbs, MetaItem } from '../shared/ui/layout.tsx'
import { Dialog } from '../shared/ui/Modal.tsx'
import { Tabs } from '../shared/ui/Tabs.tsx'
import { Activity } from './Activity.tsx'

function Steps({ checklist }: { checklist: Checklist }) {
  const can = useCan()
  const [signing, setSigning] = useState<ChecklistStep | null>(null)
  return (
    <>
      <div className="surface table-wrap">
        <table className="table" aria-label="Pasos del checklist">
          <thead>
            <tr>
              <th className="num">#</th>
              <th>Paso</th>
              <th>Firma</th>
            </tr>
          </thead>
          <tbody>
            {checklist.steps.map((step) => (
              <tr key={step.position}>
                <td className="num">{step.position}</td>
                <td>
                  <strong>{step.title}</strong>
                  {step.description && <p className="soft wrap">{step.description}</p>}
                </td>
                <td>
                  {step.signature ? (
                    <div className="stack-s">
                      <span>
                        <Icon name="check" /> {step.signature.signedBy.displayName} · {formatDateTime(step.signature.signedAt)}
                      </span>
                      {step.signature.comment && <span className="soft wrap">{step.signature.comment}</span>}
                    </div>
                  ) : can('chk.sign') ? (
                    <button className="btn" type="button" onClick={() => setSigning(step)}>
                      <Icon name="pencil" />
                      Firmar
                    </button>
                  ) : (
                    <span className="muted">Sin firmar</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {signing && (
        <Dialog title={`Firmar el paso ${signing.position} de ${checklist.code}`} onClose={() => setSigning(null)}>
          <SignStepForm checklist={checklist} step={signing} onClose={() => setSigning(null)} />
        </Dialog>
      )}
    </>
  )
}

function ChecklistView({ checklist }: { checklist: Checklist }) {
  const can = useCan()
  const documents = useEntityDocuments('checklist', checklist.code)
  return (
    <>
      <section className="glass pad stack" aria-labelledby="chk-title">
        <div className="row row--between">
          <div className="stack-s">
            <span className="id">{checklist.code}</span>
            <h1 className="title" id="chk-title">
              Checklist de producción de {checklist.package.code}
            </h1>
          </div>
          <ChecklistChip signed={checklist.signedSteps} total={checklist.steps.length} />
        </div>
        <dl className="meta">
          <MetaItem label="Paquete">
            <Code code={checklist.package.code} /> {checklist.package.title}
          </MetaItem>
          <MetaItem label="Ambiente">
            <Code code={checklist.environment.code} /> · {checklist.environment.name} · {ENVIRONMENT_KIND_LABELS[checklist.environment.kind]}
          </MetaItem>
          <MetaItem label="Plantilla">v{checklist.templateVersion}</MetaItem>
          <MetaItem label="Abrió">
            {checklist.startedBy.displayName} · {formatDateTime(checklist.createdAt)}
          </MetaItem>
        </dl>
        <p className="soft">
          {checklist.complete
            ? 'Todos los pasos están firmados. En ruta alta, el PKG ya puede pasar a desplegado.'
            : 'Cada paso se firma una vez, en cualquier orden, y la firma no se quita. En ruta alta, el PKG no pasa a desplegado hasta que estén todos.'}
        </p>
      </section>
      <Tabs
        label="Secciones del checklist"
        tabs={[
          { id: 'pasos', label: 'Pasos', count: `${checklist.signedSteps}/${checklist.steps.length}`, render: () => <Steps checklist={checklist} /> },
          {
            id: 'evidencia',
            label: 'Evidencia',
            count: documents.data?.items.length,
            render: () => <DocumentsTab entityType="checklist" entityCode={checklist.code} canUpload={can('doc.upload')} />,
          },
          { id: 'actividad', label: 'Actividad', render: () => <Activity entityType="checklist" entityCode={checklist.code} /> },
        ]}
      />
    </>
  )
}

export default function ChecklistPage() {
  const { code = '' } = useParams()
  const checklist = useChecklist(code)
  return (
    <>
      <Crumbs items={[{ label: 'Producción', to: '/produccion' }, { label: code, code: true }]} />
      <Async query={checklist} skeleton={<PageSkeleton />}>
        {(data) => <ChecklistView checklist={data} />}
      </Async>
    </>
  )
}
