// Pestaña Documentación de la ficha del REQ (boceto: requerimiento.html).
import { useState } from 'react'
import { Link } from 'react-router'
import { formatDate, shortChecksum } from '../../shared/lib/format.ts'
import { DOCUMENT_KIND_LABELS } from '../../shared/lib/labels.ts'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { useRequirementDocuments } from './api.ts'
import { UploadForm } from './UploadForm.tsx'

export function DocumentsTab({ requirementCode, canUpload }: { requirementCode: string; canUpload: boolean }) {
  const documents = useRequirementDocuments(requirementCode)
  const [uploading, setUploading] = useState(false)
  return (
    <div className="stack">
      {canUpload && (
        <div className="row">
          <button className="btn btn--primary" type="button" onClick={() => setUploading(true)}>
            <Icon name="plus" />
            Subir documento
          </button>
        </div>
      )}
      <Async query={documents}>
        {(data) =>
          data.items.length === 0 ? (
            <Empty>Este requerimiento no tiene documentos adjuntos.</Empty>
          ) : (
            <div className="surface table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>DOC</th>
                    <th>Documento</th>
                    <th>Tipo</th>
                    <th>Versión</th>
                    <th>SHA-256</th>
                    <th>Autor y fecha</th>
                    <th className="col-actions">
                      <span className="sr-only">Acciones</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {data.items.map((document) => {
                    const latest = document.versions[0]
                    return (
                      <tr key={document.code}>
                        <td>
                          <Code code={document.code} />
                        </td>
                        <td>{document.title}</td>
                        <td>{DOCUMENT_KIND_LABELS[document.kind]}</td>
                        <td className="num">{latest ? `v${latest.version}` : '—'}</td>
                        <td>{latest && <span className="mono" title={latest.checksum}>{shortChecksum(latest.checksum)}</span>}</td>
                        <td>{latest && `${latest.uploadedBy.displayName} · ${formatDate(latest.createdAt)}`}</td>
                        <td className="col-actions">
                          <Link className="btn" to={`/documentos/${document.code}`}>
                            Ver
                          </Link>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )
        }
      </Async>
      {uploading && (
        <Dialog title="Subir documento" onClose={() => setUploading(false)}>
          <UploadForm requirementCode={requirementCode} onClose={() => setUploading(false)} />
        </Dialog>
      )}
    </div>
  )
}
