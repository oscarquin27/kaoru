// Ficha de un BUILD con su manifiesto. Es de solo lectura: un BUILD no se edita.
import { useParams } from 'react-router'
import { formatDateTime } from '../../shared/lib/format.ts'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, PageSkeleton } from '../../shared/ui/feedback.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, MetaItem } from '../../shared/ui/layout.tsx'
import { useBuild } from './api.ts'

/** Texto del manifiesto, una entrada por línea. */
function Lines({ text, empty }: { text: string; empty: string }) {
  const lines = text.split('\n').map((line) => line.trim()).filter(Boolean)
  if (lines.length === 0) return <p className="muted">{empty}</p>
  return (
    <ul className="stack-s">
      {lines.map((line, index) => (
        <li key={index} className="mono">
          {line}
        </li>
      ))}
    </ul>
  )
}

export default function BuildPage() {
  const { code = '' } = useParams()
  const build = useBuild(code)

  return (
    <>
      <Crumbs items={[{ label: 'Paquetes', to: '/paquetes' }, { label: 'BUILD', to: '/paquetes/builds' }, { label: code, code: true }]} />
      <Async query={build} skeleton={<PageSkeleton />}>
        {(data) => (
          <>
            <section className="glass pad stack" aria-labelledby="build-title">
              <div className="stack-s">
                <span className="id">{data.code}</span>
                <h1 className="title" id="build-title">
                  Versión {data.version}
                </h1>
              </div>
              <dl className="meta">
                <MetaItem label="Sistema">{data.system.name}</MetaItem>
                <MetaItem label="Registrado por">
                  {data.builtBy.displayName} · {formatDateTime(data.createdAt)}
                </MetaItem>
                <MetaItem label="SHA-256">
                  <span className="mono wrap">{data.checksum}</span>
                </MetaItem>
              </dl>
              <div className="notice" role="note">
                <Icon name="lock" />
                <div className="notice__main">Un BUILD no se edita ni se borra. Si el artefacto cambia, se registra otro BUILD con su propio checksum.</div>
              </div>
            </section>

            <section className="stack" aria-labelledby="h-manifest">
              <h2 className="h2" id="h-manifest">
                Manifiesto
              </h2>
              <div className="surface pad stack">
                <h3 className="h3">REQ incluidos</h3>
                {data.requirements.length === 0 ? (
                  <p className="muted">Ningún requerimiento declarado.</p>
                ) : (
                  <ul className="stack-s" aria-label="REQ incluidos">
                    {data.requirements.map((requirement) => (
                      <li key={requirement.code}>
                        <Code code={requirement.code} /> {requirement.title}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
              <div className="grid grid--2">
                <div className="surface pad stack">
                  <h3 className="h3">Migraciones</h3>
                  <Lines text={data.migrations} empty="Sin migraciones." />
                </div>
                <div className="surface pad stack">
                  <h3 className="h3">Dependencias</h3>
                  <Lines text={data.dependencies} empty="Sin dependencias declaradas." />
                </div>
              </div>
              <div className="surface pad stack">
                <h3 className="h3">Notas</h3>
                <p className="wrap">{data.notes || <span className="muted">Sin notas.</span>}</p>
              </div>
            </section>
          </>
        )}
      </Async>
    </>
  )
}
