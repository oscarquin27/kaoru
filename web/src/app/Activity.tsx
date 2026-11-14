// Actividad de una entidad: auditoría y comentarios en un solo orden cronológico, lo más
// reciente arriba. Junta dos features, por eso vive en app/.
import { useCan, useWorkflow } from '../shared/api/session.ts'
import type { EntityType } from '../shared/api/types.ts'
import { formatDateTime } from '../shared/lib/format.ts'
import { TextWithCodes } from '../shared/ui/Code.tsx'
import { BlockSkeleton, ErrorNotice } from '../shared/ui/feedback.tsx'
import { useAudit } from '../features/audit/api.ts'
import { describeEntry } from '../features/audit/describe.ts'
import { useComments } from '../features/comments/api.ts'
import { CommentForm } from '../features/comments/CommentForm.tsx'

// Las que tienen flujo admiten comentarios; las de producción y el informe solo traen su auditoría.
const COMMENTED = ['requirement', 'defect', 'package'] as const
type Commented = (typeof COMMENTED)[number]
type ActivityEntity = Commented | 'checklist' | 'diagnostic' | 'smoke_test' | 'conformance_report' | 'activity'

const isCommented = (entityType: ActivityEntity): entityType is Commented => (COMMENTED as readonly string[]).includes(entityType)

/** `value` traduce un valor guardado al texto de pantalla (por ejemplo, ids de usuario a nombres). */
export function Activity({ entityType, entityCode, value }: { entityType: ActivityEntity; entityCode: string; value?: (field: string | undefined, raw: string | undefined) => string | undefined }) {
  const can = useCan()
  const commented = isCommented(entityType)
  const audit = useAudit({ entityCode })
  const comments = useComments(entityType as EntityType, entityCode, commented)
  const states = useWorkflow(commented ? entityType : 'package', commented).data?.states ?? []
  // La auditoría guarda claves de estado; aquí se muestran con su nombre.
  const named = (value: string | undefined) => states.find((state) => state.key === value)?.name ?? value

  const events = [
    // Los comentarios ya salen con su texto: su registro de auditoría sería una repetición.
    ...(audit.data?.items ?? []).filter((entry) => entry.field !== 'comment').map((entry) => ({
      key: `a-${entry.seq}`,
      at: entry.occurredAt,
      text: describeEntry(
        entry.action === 'transition'
          ? { ...entry, oldValue: named(entry.oldValue), newValue: named(entry.newValue) }
          : value
            ? { ...entry, oldValue: value(entry.field, entry.oldValue), newValue: value(entry.field, entry.newValue) }
            : entry,
      ),
      quote: '',
    })),
    ...(comments.data?.items ?? []).map((comment) => ({ key: `c-${comment.id}`, at: comment.createdAt, text: `${comment.author.displayName} comentó:`, quote: comment.body })),
  ].sort((a, b) => b.at.localeCompare(a.at))

  return (
    <div className="stack">
      {commented && can('comment.create') && (
        <div className="surface pad">
          <CommentForm entityType={entityType as EntityType} entityCode={entityCode} />
        </div>
      )}
      {audit.error && <ErrorNotice error={audit.error} />}
      {comments.error && <ErrorNotice error={comments.error} />}
      {audit.isLoading || comments.isLoading ? (
        <BlockSkeleton />
      ) : (
        <div className="surface pad">
          {events.length === 0 ? (
            <p className="muted">Sin actividad registrada.</p>
          ) : (
            <ol className="timeline">
              {events.map((event, index) => (
                <li key={event.key} aria-current={index === 0 ? 'true' : undefined}>
                  <span className="num muted">{formatDateTime(event.at)}</span>
                  <br />
                  <TextWithCodes text={event.text} />
                  {event.quote && <p className="soft wrap">{event.quote}</p>}
                </li>
              ))}
            </ol>
          )}
        </div>
      )}
    </div>
  )
}
