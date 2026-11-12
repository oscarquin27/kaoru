// Ficha de un despliegue (DEP). Es de solo lectura: un despliegue no se edita.
import { useParams } from 'react-router'
import { formatDateTime } from '../../shared/lib/format.ts'
import { ENVIRONMENT_KIND_LABELS } from '../../shared/lib/labels.ts'
import { DeploymentResultChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, PageSkeleton } from '../../shared/ui/feedback.tsx'
import { Crumbs, MetaItem } from '../../shared/ui/layout.tsx'
import { useDeployment } from './api.ts'
import { Deployed } from './DeploymentsTable.tsx'

export default function DeploymentPage() {
  const { code = '' } = useParams()
  const deployment = useDeployment(code)

  return (
    <>
      <Crumbs items={[{ label: 'Ambientes', to: '/ambientes' }, { label: 'Despliegues', to: '/ambientes/despliegues' }, { label: code, code: true }]} />
      <Async query={deployment} skeleton={<PageSkeleton />}>
        {(data) => (
          <>
            <section className="glass pad stack" aria-labelledby="dep-title">
              <div className="row row--between">
                <div className="stack-s">
                  <span className="id">{data.code}</span>
                  <h1 className="title" id="dep-title">
                    Despliegue en {data.environment.name}
                  </h1>
                </div>
                <DeploymentResultChip result={data.result} />
              </div>
              <dl className="meta">
                <MetaItem label="Ambiente">
                  <Code code={data.environment.code} /> · {ENVIRONMENT_KIND_LABELS[data.environment.kind]}
                </MetaItem>
                <MetaItem label="Desplegado">
                  <Deployed build={data.build} bundle={data.package} />
                </MetaItem>
                {data.build && (
                  <MetaItem label="SHA-256 del BUILD">
                    <span className="mono wrap">{data.build.checksum}</span>
                  </MetaItem>
                )}
                <MetaItem label="Desplegó">
                  {data.executedBy.displayName} · {formatDateTime(data.executedAt)}
                </MetaItem>
                <MetaItem label="Registrado">{formatDateTime(data.createdAt)}</MetaItem>
              </dl>
            </section>
            <section className="surface pad stack">
              <h2 className="h3">Notas</h2>
              <p className="wrap">{data.notes || <span className="muted">Sin notas.</span>}</p>
              <p className="soft">Un despliegue no se edita ni se borra. Si salió mal, se registra otro.</p>
            </section>
          </>
        )}
      </Async>
    </>
  )
}
