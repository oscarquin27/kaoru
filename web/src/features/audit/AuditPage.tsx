// Auditoría: consulta filtrable, verificación de la cadena de hash y exportación.
import { useState } from 'react'
import { formatDateTime, shortChecksum } from '../../shared/lib/format.ts'
import { ENTITY_LABELS, labelOf } from '../../shared/lib/labels.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Chip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty, ErrorNotice } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterInput, Pager } from '../../shared/ui/Filters.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { auditExportUrl, dayToInstant, useAudit, useAuditVerification } from './api.ts'
import { ACTION_LABELS } from './describe.ts'

function Verification() {
  const [asked, setAsked] = useState(false)
  const verification = useAuditVerification(asked)
  return (
    <section className="surface pad-s row" aria-label="Verificación de la cadena">
      <button className="btn" type="button" disabled={verification.isFetching} onClick={() => (asked ? void verification.refetch() : setAsked(true))}>
        <Icon name="lock" />
        {verification.isFetching ? 'Verificando…' : 'Verificar la cadena'}
      </button>
      <span aria-live="polite" className="row">
        {verification.error && <ErrorNotice error={verification.error} />}
        {verification.data &&
          (verification.data.intact ? (
            <Chip tone="ok" icon="check">
              Cadena íntegra · {verification.data.entries} registros
            </Chip>
          ) : (
            <Chip tone="bloq" icon="x">
              Cadena rota desde el registro {verification.data.firstBrokenSeq ?? '?'}
            </Chip>
          ))}
        {!asked && <span className="muted">Cada registro guarda el hash del anterior: si alguien altera uno, la cadena deja de cuadrar.</span>}
      </span>
    </section>
  )
}

export default function AuditPage() {
  const { params, pick, setFilter } = useUrlFilters()
  const { entityCode, actor, from, to } = pick(['entityCode', 'actor', 'from', 'to'])
  const range = { from: dayToInstant(from), to: dayToInstant(to, true) }
  const page = Number(params.get('page')) || 1
  const audit = useAudit({ entityCode, actor, ...range, page: page > 1 ? page : undefined })

  return (
    <>
      <Crumbs items={[{ label: 'Administración', to: '/administracion' }, { label: 'Auditoría' }]} />
      <PageHead
        title="Auditoría"
        subtitle="Quién hizo qué y cuándo. Los registros no se editan ni se borran."
        actions={
          <a className="btn" href={auditExportUrl(range)} download>
            <Icon name="download" />
            Exportar CSV
          </a>
        }
      />
      <Verification />
      <FilterBar>
        <FilterInput name="entityCode" label="ID de la entidad" placeholder="REQ-2026-0001" />
        <FilterInput name="actor" label="Actor (ID de usuario)" placeholder="Clic en un nombre de la tabla" />
        <FilterInput name="from" label="Desde" type="date" />
        <FilterInput name="to" label="Hasta" type="date" />
      </FilterBar>
      <Async query={audit}>
        {(data) => (
          <>
            {data.items.length === 0 ? (
              <Empty>No hay registros con este filtro.</Empty>
            ) : (
              <div className="surface table-wrap">
                <table className="table">
                  <thead>
                    <tr>
                      <th className="num">N.º</th>
                      <th>Fecha</th>
                      <th>Actor</th>
                      <th>Entidad</th>
                      <th>Acción</th>
                      <th>Campo</th>
                      <th>Antes</th>
                      <th>Después</th>
                      <th>Motivo</th>
                      <th>Hash</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.items.map((entry) => (
                      <tr key={entry.seq}>
                        <td className="num">{entry.seq}</td>
                        <td className="num">{formatDateTime(entry.occurredAt)}</td>
                        <td>
                          <button className="id" type="button" title="Filtrar por este actor" onClick={() => setFilter('actor', entry.actor.id)}>
                            {entry.actor.displayName}
                          </button>
                        </td>
                        <td>
                          <div className="cell-stack">
                            <span>{labelOf(ENTITY_LABELS, entry.entityType)}</span>
                            {entry.entityCode && <Code code={entry.entityCode} panel />}
                          </div>
                        </td>
                        <td>{ACTION_LABELS[entry.action]}</td>
                        <td>{entry.field ? <span className="mono">{entry.field}</span> : <span className="muted">—</span>}</td>
                        <td className="wrap">{entry.oldValue ?? <span className="muted">—</span>}</td>
                        <td className="wrap">{entry.newValue ?? <span className="muted">—</span>}</td>
                        <td className="wrap">{entry.reason ?? <span className="muted">—</span>}</td>
                        <td>
                          <span className="mono muted" title={`SHA-256: ${entry.hash}`}>
                            {shortChecksum(entry.hash)}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            <Pager page={data.page} size={data.size} total={data.total} noun="registros" />
          </>
        )}
      </Async>
    </>
  )
}
