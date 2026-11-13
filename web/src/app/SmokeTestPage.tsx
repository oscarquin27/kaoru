// Ficha de una prueba de humo (SMOKE): su resultado, la evidencia y la actividad. Junta producción,
// documentos y auditoría, por eso vive en app/.
import { useParams } from 'react-router'
import { DocumentsTab } from '../features/documents/DocumentsTab.tsx'
import { useEntityDocuments } from '../features/documents/api.ts'
import { useSmokeTest } from '../features/production/api.ts'
import { useCan } from '../shared/api/session.ts'
import type { SmokeTest } from '../shared/api/types.ts'
import { formatDateTime } from '../shared/lib/format.ts'
import { SmokeChip } from '../shared/ui/chips.tsx'
import { Code } from '../shared/ui/Code.tsx'
import { Async, PageSkeleton } from '../shared/ui/feedback.tsx'
import { Crumbs, MetaItem } from '../shared/ui/layout.tsx'
import { Tabs } from '../shared/ui/Tabs.tsx'
import { Activity } from './Activity.tsx'

function SmokeTestView({ smoke }: { smoke: SmokeTest }) {
  const can = useCan()
  const documents = useEntityDocuments('smoke_test', smoke.code)
  return (
    <>
      <section className="glass pad stack" aria-labelledby="smoke-title">
        <div className="row row--between">
          <div className="stack-s">
            <span className="id">{smoke.code}</span>
            <h1 className="title" id="smoke-title">
              Prueba de humo de {smoke.package.code}
            </h1>
          </div>
          <SmokeChip result={smoke.result} />
        </div>
        <dl className="meta">
          <MetaItem label="Paquete">
            <Code code={smoke.package.code} /> {smoke.package.title}
          </MetaItem>
          <MetaItem label="Despliegue">
            <Code code={smoke.deployment.code} /> · {smoke.deployment.environment.name} · {formatDateTime(smoke.deployment.executedAt)}
          </MetaItem>
          <MetaItem label="Registró">
            {smoke.executedBy.displayName} · {formatDateTime(smoke.createdAt)}
          </MetaItem>
        </dl>
      </section>
      <section className="surface pad stack">
        <h2 className="h3">Notas</h2>
        <p className="wrap">{smoke.notes || <span className="muted">Sin notas.</span>}</p>
        <p className="soft">Una prueba de humo no se corrige. Si se repite, se registra otra y cuenta la última.</p>
      </section>
      <Tabs
        label="Secciones de la prueba de humo"
        tabs={[
          {
            id: 'evidencia',
            label: 'Evidencia',
            count: documents.data?.items.length,
            render: () => <DocumentsTab entityType="smoke_test" entityCode={smoke.code} canUpload={can('doc.upload')} />,
          },
          { id: 'actividad', label: 'Actividad', render: () => <Activity entityType="smoke_test" entityCode={smoke.code} /> },
        ]}
      />
    </>
  )
}

export default function SmokeTestPage() {
  const { code = '' } = useParams()
  const smoke = useSmokeTest(code)
  return (
    <>
      <Crumbs items={[{ label: 'Producción', to: '/produccion' }, { label: 'Pruebas de humo', to: '/produccion/humo' }, { label: code, code: true }]} />
      <Async query={smoke} skeleton={<PageSkeleton />}>
        {(data) => <SmokeTestView smoke={data} />}
      </Async>
    </>
  )
}
