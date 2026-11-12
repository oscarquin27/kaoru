// Subida de un documento nuevo o de una versión nueva. El servidor valida el tipo real y el tamaño.
import type { Document } from '../../shared/api/types.ts'
import { DOCUMENT_KIND_LABELS, options } from '../../shared/lib/labels.ts'
import { Form, SelectField, TextField } from '../../shared/ui/Form.tsx'
import { useCreateDocument, useCreateDocumentVersion } from './api.ts'

/** Con `document` sube una versión; sin él, adjunta un documento nuevo al requerimiento. */
export function UploadForm({ document, requirementCode, onClose }: { document?: Document; requirementCode?: string; onClose: () => void }) {
  const create = useCreateDocument()
  const addVersion = useCreateDocumentVersion(document?.code ?? '')

  function submit(data: FormData) {
    if (document) return addVersion.mutateAsync(data)
    data.set('entityType', 'requirement')
    data.set('entityCode', requirementCode ?? '')
    return create.mutateAsync(data)
  }

  return (
    <Form label={document ? 'Subir nueva versión' : 'Subir documento'} submitLabel="Subir" onSubmit={submit} onDone={onClose} onCancel={onClose}>
      {!document && (
        <div className="form__row">
          <TextField name="title" label="Título" required />
          <SelectField name="kind" label="Tipo" required defaultValue="specification">
            {options(DOCUMENT_KIND_LABELS).map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </SelectField>
        </div>
      )}
      <TextField
        name="file"
        label="Archivo"
        type="file"
        required
        hint={document ? 'Las versiones anteriores se conservan.' : 'Se guarda con su SHA-256, autor y fecha.'}
      />
    </Form>
  )
}
