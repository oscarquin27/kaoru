// Visor de documentos (boceto: documento.html): se ve dentro del sistema, sin descargar,
// lo que el navegador sabe mostrar. Office y otros tipos: descarga.
import { useState } from 'react'
import { useParams } from 'react-router'
import { useCan } from '../../shared/api/session.ts'
import type { Document, DocumentVersion } from '../../shared/api/types.ts'
import { routeOf } from '../../shared/lib/codes.ts'
import { formatBytes, formatDate, formatDateTime, shortChecksum } from '../../shared/lib/format.ts'
import { DOCUMENT_KIND_LABELS } from '../../shared/lib/labels.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Chip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, PageSkeleton } from '../../shared/ui/feedback.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, MetaItem, PageHead } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { contentUrl, useDocument, useTextContent } from './api.ts'
import { Markdown } from './Markdown.tsx'
import { UploadForm } from './UploadForm.tsx'

function isText(mediaType: string): boolean {
  return mediaType.startsWith('text/')
}

function Content({ code, version }: { code: string; version: DocumentVersion }) {
  const text = useTextContent(code, version.version, isText(version.mediaType))
  const url = contentUrl(code, version.version)
  if (isText(version.mediaType)) {
    return (
      <Async query={text}>
        {(data) => (
          <>
            <div>
              {data.checksum === version.checksum ? (
                <Chip tone="ok" icon="check">
                  Checksum verificado
                </Chip>
              ) : (
                <Chip tone="bloq" icon="x">
                  El checksum no coincide
                </Chip>
              )}
            </div>
            {version.mediaType === 'text/markdown' ? <Markdown source={data.text} /> : <pre className="pre">{data.text}</pre>}
          </>
        )}
      </Async>
    )
  }
  if (version.mediaType === 'application/pdf') {
    return (
      <object className="viewer__object" data={url} type="application/pdf" aria-label={version.fileName}>
        <p className="soft">El PDF no se pudo mostrar aquí. Descárgalo para verlo.</p>
      </object>
    )
  }
  if (version.mediaType.startsWith('image/')) return <img className="viewer__image" src={url} alt={version.fileName} />
  return <p className="soft">Este tipo de archivo ({version.mediaType}) no se ve dentro del sistema. Descárgalo para abrirlo.</p>
}

function DocumentView({ document }: { document: Document }) {
  const can = useCan()
  const { params, setFilter } = useUrlFilters()
  const [uploading, setUploading] = useState(false)
  const latest = document.versions[0]
  const version = document.versions.find((item) => String(item.version) === params.get('v')) ?? latest

  return (
    <>
      <PageHead
        code={document.code}
        title={document.title}
        subtitle={DOCUMENT_KIND_LABELS[document.kind]}
        actions={
          <>
            {version && (
              <a className="btn" href={contentUrl(document.code, version.version, true)} download={version.fileName}>
                <Icon name="download" />
                Descargar
              </a>
            )}
            {can('doc.upload') && (
              <button className="btn btn--primary" type="button" onClick={() => setUploading(true)}>
                <Icon name="plus" />
                Subir nueva versión
              </button>
            )}
          </>
        }
      />
      {!version ? (
        <p className="soft">Este documento no tiene versiones.</p>
      ) : (
        <div className="grid grid--side">
          <section className="surface viewer" aria-label="Visor">
            <div className="viewer__bar">
              <label className="field">
                Versión
                <select className="select" value={version.version} onChange={(event) => setFilter('v', event.target.value === String(latest?.version) ? '' : event.target.value)}>
                  {document.versions.map((item) => (
                    <option key={item.version} value={item.version}>
                      v{item.version} · {formatDate(item.createdAt)}
                      {item === latest ? ' (actual)' : ''}
                    </option>
                  ))}
                </select>
              </label>
              <span className="spacer" />
              <span className="muted">{version.fileName}</span>
            </div>
            <div className="viewer__body stack">
              <Content key={version.version} code={document.code} version={version} />
            </div>
          </section>

          <aside className="stack" aria-label="Datos del documento">
            <section className="surface pad stack">
              <h2 className="h2">Datos</h2>
              <dl className="meta stack">
                <MetaItem label="Tipo">{version.mediaType} · tipo real verificado</MetaItem>
                <MetaItem label="Tamaño">
                  <span className="num">{formatBytes(version.sizeBytes)}</span>
                </MetaItem>
                <MetaItem label="Enlazado a">
                  {document.linkedTo.length > 0 ? document.linkedTo.map((entity) => <Code key={entity.code} code={entity.code} />) : <span className="muted">Nada</span>}
                </MetaItem>
              </dl>
            </section>
            <section className="surface pad stack">
              <h2 className="h2">Versiones</h2>
              <ol className="timeline">
                {document.versions.map((item) => (
                  <li key={item.version} aria-current={item === version ? 'true' : undefined}>
                    <div className="row row--between">
                      <strong>v{item.version}</strong>
                      {item === latest && (
                        <Chip tone="actual" icon="target">
                          Actual
                        </Chip>
                      )}
                    </div>
                    <span className="soft">
                      {item.uploadedBy.displayName} · {formatDateTime(item.createdAt)}
                    </span>
                    <br />
                    <span className="mono muted" title={`SHA-256: ${item.checksum}`}>
                      {shortChecksum(item.checksum)}
                    </span>
                  </li>
                ))}
              </ol>
              <p className="muted">Subir una versión conserva las anteriores. El informe de conformidad cita la versión y su SHA-256.</p>
            </section>
          </aside>
        </div>
      )}
      {uploading && (
        <Dialog title={`Nueva versión de ${document.code}`} onClose={() => setUploading(false)}>
          <UploadForm document={document} onClose={() => setUploading(false)} />
        </Dialog>
      )}
    </>
  )
}

export default function DocumentPage() {
  const { code = '' } = useParams()
  const document = useDocument(code)
  // Migas de trazabilidad: el REQ al que está enlazado y el documento.
  const parent = document.data?.linkedTo.find((entity) => routeOf(entity.code))
  return (
    <>
      <Crumbs
        items={[
          ...(parent ? [{ label: parent.code, to: `${routeOf(parent.code)}#documentacion`, code: true }] : [{ label: 'Requerimientos', to: '/requerimientos' }]),
          { label: code, code: true },
        ]}
      />
      <Async query={document} skeleton={<PageSkeleton />}>
        {(data) => <DocumentView document={data} />}
      </Async>
    </>
  )
}
