// Pestaña Enlaces. La documentación vive en Google Sheets: aquí se enlaza, no se copia.
import { useState } from 'react'
import type { FieldError } from '../../shared/api/errors.ts'
import type { Link, LinkKind } from '../../shared/api/types.ts'
import { formatDate } from '../../shared/lib/format.ts'
import { LINK_KIND_LABELS, options } from '../../shared/lib/labels.ts'
import { parseSheetLink, sheetPreviewUrl } from '../../shared/lib/sheetLink.ts'
import { Chip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { CheckField, Form, SelectField, TextField, optional, text } from '../../shared/ui/Form.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { useCreateLink, useDeleteLink, useLinks, useUpdateLink } from './api.ts'

interface DocumentOption {
  code: string
  title: string
}

function isSheetsUrl(url: string): boolean {
  return url.trim().startsWith('https://docs.google.com/')
}

/** Misma regla que el dominio: una URL que dice ser de Google Sheets tiene que tener su forma. */
function validateLink(data: FormData): FieldError[] {
  const url = text(data, 'url')
  const claimsSheets = text(data, 'kind') === 'google_sheets' || isSheetsUrl(url)
  if (url && claimsSheets && !parseSheetLink(url)) {
    return [{ field: 'url', code: 'invalid', message: 'No es una URL de Google Sheets.' }]
  }
  return []
}

function AddLinkForm({ code, onClose }: { code: string; onClose: () => void }) {
  const create = useCreateLink(code)
  const [url, setUrl] = useState('')
  const sheet = parseSheetLink(url)
  return (
    <Form
      label="Agregar enlace"
      submitLabel="Agregar enlace"
      validate={validateLink}
      onSubmit={(data) =>
        create.mutateAsync({
          url: text(data, 'url'),
          kind: optional(data, 'kind') as LinkKind | undefined,
          label: optional(data, 'label'),
          isOfficial: data.get('isOfficial') === 'on',
        })
      }
      onDone={onClose}
      onCancel={onClose}
    >
      <TextField
        name="url"
        label="URL"
        type="url"
        required
        value={url}
        onChange={(event) => setUrl(event.target.value)}
        placeholder="https://docs.google.com/spreadsheets/d/…"
        hint={
          sheet
            ? `Hoja de Google reconocida${sheet.gid ? ` · pestaña ${sheet.gid}` : ''}${sheet.range ? ` · ${sheet.range}` : ''}`
            : 'Pega la URL. No se comprueba que exista ni que tengas permiso.'
        }
      />
      <div className="form__row">
        <SelectField name="kind" label="Tipo" defaultValue="">
          <option value="">Deducir de la URL</option>
          {options(LINK_KIND_LABELS).map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </SelectField>
        <TextField name="label" label="Etiqueta" maxLength={200} hint="Por ejemplo «Especificación SyPago · pestaña Pagos»." />
      </div>
      <CheckField name="isOfficial" label="Es el documento oficial del requerimiento" />
    </Form>
  )
}

function EditLinkForm({ code, link, documents, onClose }: { code: string; link: Link; documents: DocumentOption[]; onClose: () => void }) {
  const update = useUpdateLink(code)
  return (
    <Form
      label="Editar enlace"
      submitLabel="Guardar"
      onSubmit={(data) =>
        update.mutateAsync({
          id: link.id,
          patch: { label: text(data, 'label'), isOfficial: data.get('isOfficial') === 'on', snapshotDocument: optional(data, 'snapshotDocument') ?? null },
        })
      }
      onDone={onClose}
      onCancel={onClose}
    >
      <TextField name="label" label="Etiqueta" required maxLength={200} defaultValue={link.label} />
      <SelectField
        name="snapshotDocument"
        label="Instantánea (DOC)"
        defaultValue={link.snapshotDocument ?? ''}
        hint="Una exportación adjunta en Documentación. Para el informe de conformidad cuenta la instantánea, no el enlace."
      >
        <option value="">Sin instantánea</option>
        {documents.map((document) => (
          <option key={document.code} value={document.code}>
            {document.code} · {document.title}
          </option>
        ))}
      </SelectField>
      <CheckField name="isOfficial" label="Es el documento oficial del requerimiento" defaultChecked={link.isOfficial} />
    </Form>
  )
}

function DeleteLinkForm({ code, link, onClose }: { code: string; link: Link; onClose: () => void }) {
  const remove = useDeleteLink(code)
  return (
    <Form label="Quitar enlace" submitLabel="Quitar enlace" onSubmit={() => remove.mutateAsync(link.id)} onDone={onClose} onCancel={onClose}>
      <p>
        Se quita «{link.label}» de este requerimiento. La hoja o el sitio enlazado no se toca.
      </p>
    </Form>
  )
}

function Snapshot({ link }: { link: Link }) {
  if (link.kind !== 'google_sheets') return <span className="muted">No aplica</span>
  if (!link.snapshotDocument) {
    return (
      <Chip tone="aviso" icon="triangle">
        Sin instantánea
      </Chip>
    )
  }
  return (
    <div className="cell-stack">
      <Chip tone="ok" icon="check">
        Adjunta
      </Chip>
      <Code code={link.snapshotDocument} />
    </div>
  )
}

type Action = { kind: 'add' } | { kind: 'edit' | 'delete'; link: Link }

export function LinksTab({ code, documents, canManage }: { code: string; documents: DocumentOption[]; canManage: boolean }) {
  const links = useLinks(code)
  const [action, setAction] = useState<Action | null>(null)
  const [preview, setPreview] = useState<string | null>(null)
  const close = () => setAction(null)

  return (
    <div className="stack">
      {canManage && (
        <div className="row">
          <button className="btn btn--primary" type="button" onClick={() => setAction({ kind: 'add' })}>
            <Icon name="plus" />
            Agregar enlace
          </button>
        </div>
      )}
      <Async query={links}>
        {(data) =>
          data.items.length === 0 ? (
            <Empty>Este requerimiento no tiene enlaces.</Empty>
          ) : (
            <div className="surface table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>Tipo</th>
                    <th>Enlace</th>
                    <th>Instantánea</th>
                    <th>Agregado por</th>
                    <th className="col-actions">
                      <span className="sr-only">Acciones</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {data.items.map((link) => {
                    const sheet = link.kind === 'google_sheets' ? link.sheet : undefined
                    return (
                      <tr key={link.id}>
                        <td>
                          <span className="cell-line">
                            <Icon name={sheet ? 'grid' : 'link'} />
                            {LINK_KIND_LABELS[link.kind]}
                          </span>
                        </td>
                        <td>
                          <div className="cell-stack">
                            <span>{link.label}</span>
                            {link.isOfficial && (
                              <Chip tone="ok" icon="check">
                                Documento oficial
                              </Chip>
                            )}
                          </div>
                        </td>
                        <td>
                          <Snapshot link={link} />
                        </td>
                        <td>
                          {link.addedBy.displayName} · {formatDate(link.createdAt)}
                        </td>
                        <td className="col-actions">
                          <span className="cell-line">
                            {sheet && (
                              <button className="btn" type="button" aria-pressed={preview === link.id} onClick={() => setPreview(preview === link.id ? null : link.id)}>
                                Vista previa
                              </button>
                            )}
                            <a
                              className="btn"
                              href={link.url}
                              target="_blank"
                              rel="noopener noreferrer"
                              title={sheet ? 'Se abre en otra pestaña, con tu sesión de Google' : 'Se abre en otra pestaña'}
                            >
                              <Icon name="arrow-out" />
                              {sheet ? 'Abrir en Google Sheets' : 'Abrir enlace'}
                            </a>
                            {canManage && (
                              <>
                                <button className="btn btn--icon" type="button" title="Editar enlace" aria-label={`Editar enlace ${link.label}`} onClick={() => setAction({ kind: 'edit', link })}>
                                  <Icon name="pencil" />
                                </button>
                                <button className="btn btn--icon" type="button" title="Quitar enlace" aria-label={`Quitar enlace ${link.label}`} onClick={() => setAction({ kind: 'delete', link })}>
                                  <Icon name="trash" />
                                </button>
                              </>
                            )}
                          </span>
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

      {links.data?.items.map(
        (link) =>
          link.id === preview &&
          link.sheet && (
            <section key={link.id} className="surface pad stack" aria-label={`Vista previa de ${link.label}`}>
              <p className="muted">
                Vista previa opcional: la sirve Google y necesita internet y tu sesión de Google. Si no aparece, usa «Abrir en Google Sheets».
              </p>
              <iframe className="sheet-preview" src={sheetPreviewUrl(link.sheet)} title={`Vista previa de ${link.label}`} loading="lazy" referrerPolicy="no-referrer" />
            </section>
          ),
      )}

      {action?.kind === 'add' && (
        <Dialog title="Agregar enlace" onClose={close}>
          <AddLinkForm code={code} onClose={close} />
        </Dialog>
      )}
      {action?.kind === 'edit' && (
        <Dialog title="Editar enlace" onClose={close}>
          <EditLinkForm code={code} link={action.link} documents={documents} onClose={close} />
        </Dialog>
      )}
      {action?.kind === 'delete' && (
        <Dialog title="Quitar enlace" onClose={close}>
          <DeleteLinkForm code={code} link={action.link} onClose={close} />
        </Dialog>
      )}
    </div>
  )
}
