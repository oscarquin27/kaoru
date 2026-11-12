// Ficha de un ambiente: qué tiene instalado y su historial de despliegues.
import { useState } from 'react'
import { useParams } from 'react-router'
import { useCan } from '../../shared/api/session.ts'
import { formatDate, formatDateTime } from '../../shared/lib/format.ts'
import { ENVIRONMENT_KIND_LABELS, OS_LABELS } from '../../shared/lib/labels.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { RetiredChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, ErrorNotice, PageSkeleton } from '../../shared/ui/feedback.tsx'
import { Pager } from '../../shared/ui/Filters.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, MetaItem } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { useDeployments, useEnvironment, useUpdateEnvironment } from './api.ts'
import { DeploymentForm } from './DeploymentForm.tsx'
import { Deployed, DeploymentsTable } from './DeploymentsTable.tsx'
import { EnvironmentForm } from './EnvironmentForm.tsx'

export default function EnvironmentPage() {
  const { code = '' } = useParams()
  const can = useCan()
  const { params } = useUrlFilters()
  const page = Number(params.get('page')) || 1
  const environment = useEnvironment(code)
  const deployments = useDeployments({ environment: code, page: page > 1 ? page : undefined })
  const update = useUpdateEnvironment(code)
  const [dialog, setDialog] = useState<'edit' | 'deploy' | null>(null)
  const close = () => setDialog(null)

  return (
    <>
      <Crumbs items={[{ label: 'Ambientes', to: '/ambientes' }, { label: code, code: true }]} />
      <Async query={environment} skeleton={<PageSkeleton />}>
        {(data) => {
          // Regla de negocio 6: en producción despliega change management.
          const canDeploy = !data.retired && can(data.kind === 'production' ? 'dep.create_production' : 'dep.create')
          return (
            <>
              <section className="glass pad stack" aria-labelledby="env-title">
                <div className="row row--between">
                  <div className="stack-s">
                    <span className="id">{data.code}</span>
                    <h1 className="title" id="env-title">
                      {data.name}
                    </h1>
                  </div>
                  <RetiredChip retired={data.retired} />
                </div>
                <dl className="meta">
                  <MetaItem label="Sistema">{data.system.name}</MetaItem>
                  <MetaItem label="Tipo">{ENVIRONMENT_KIND_LABELS[data.kind]}</MetaItem>
                  <MetaItem label="Sistema operativo">{OS_LABELS[data.os]}</MetaItem>
                  <MetaItem label="Instalado">
                    {data.installed ? (
                      <>
                        <Deployed build={data.installed.build} bundle={data.installed.package} />
                        <span className="muted">
                          {' '}
                          <Code code={data.installed.deployment} /> · {formatDateTime(data.installed.deployedAt)}
                        </span>
                      </>
                    ) : (
                      <span className="muted">Sin despliegues exitosos</span>
                    )}
                  </MetaItem>
                  <MetaItem label="Registrado por">
                    {data.createdBy.displayName} · {formatDate(data.createdAt)}
                  </MetaItem>
                </dl>
                {data.kind === 'production' && <p className="soft">Ambiente de producción: no admite RUN, y sus despliegues los registra change management.</p>}
                {(canDeploy || can('env.manage')) && (
                  <div className="row">
                    {canDeploy && (
                      <button className="btn btn--primary" type="button" onClick={() => setDialog('deploy')}>
                        <Icon name="plus" />
                        Registrar despliegue
                      </button>
                    )}
                    {can('env.manage') && (
                      <>
                        <button className="btn" type="button" onClick={() => setDialog('edit')}>
                          <Icon name="pencil" />
                          Editar
                        </button>
                        <button className="btn" type="button" disabled={update.isPending} onClick={() => update.mutate({ retired: !data.retired })}>
                          {data.retired ? 'Reactivar' : 'Retirar'}
                        </button>
                      </>
                    )}
                  </div>
                )}
                {update.error && <ErrorNotice error={update.error} />}
              </section>

              <section className="surface pad stack">
                <h2 className="h3">Notas</h2>
                <p className="wrap">{data.notes || <span className="muted">Sin notas.</span>}</p>
              </section>

              <section className="stack" aria-labelledby="h-deps">
                <h2 className="h2" id="h-deps">
                  Historial de despliegues
                </h2>
                <Async query={deployments}>
                  {(list) => (
                    <>
                      <DeploymentsTable items={list.items} empty="Este ambiente no tiene despliegues registrados." />
                      <Pager page={list.page} size={list.size} total={list.total} noun="despliegues" />
                    </>
                  )}
                </Async>
              </section>

              {dialog === 'edit' && (
                <Dialog title={`Editar ${data.code}`} onClose={close}>
                  <EnvironmentForm environment={data} onClose={close} />
                </Dialog>
              )}
              {dialog === 'deploy' && (
                <Dialog title={`Registrar despliegue en ${data.name}`} onClose={close}>
                  <DeploymentForm environment={data} onClose={close} />
                </Dialog>
              )}
            </>
          )
        }}
      </Async>
    </>
  )
}
