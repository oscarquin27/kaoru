// Contrato de entrada dev → QA: quien entrega marca cada ítem cumplido.
import { ErrorNotice, Async } from '../../shared/ui/feedback.tsx'
import { formatDate } from '../../shared/lib/format.ts'
import { Icon } from '../../shared/ui/Icon.tsx'
import { useEntryContract, useToggleEntryContractItem } from './api.ts'

export function EntryContract({ code, canCheck }: { code: string; canCheck: boolean }) {
  const contract = useEntryContract(code)
  const toggle = useToggleEntryContractItem(code)
  return (
    <section className="surface pad stack" aria-labelledby="h-contrato">
      <div>
        <h3 className="h2" id="h-contrato">
          Contrato de entrada
        </h3>
        <p className="subtitle">Lo que desarrollo entrega a QA. Sin todo marcado, el REQ no pasa a «Entregado a QA».</p>
      </div>
      {toggle.error && <ErrorNotice error={toggle.error} />}
      <Async query={contract}>
        {(data) => (
          <ul className="list">
            {data.items.map((item) => (
              <li key={item.key}>
                {canCheck ? (
                  <input
                    type="checkbox"
                    aria-label={item.label}
                    checked={item.done}
                    disabled={toggle.isPending}
                    onChange={(event) => toggle.mutate({ key: item.key, done: event.target.checked })}
                  />
                ) : (
                  <span className={`list__icon${item.done ? ' list__icon--ok' : ''}`}>
                    <Icon name={item.done ? 'check' : 'circle'} />
                  </span>
                )}
                <div className="list__main">
                  {item.label}
                  <br />
                  <span className="muted">
                    {item.done ? `Cumplido${item.checkedBy ? ` · ${item.checkedBy.displayName}` : ''}${item.checkedAt ? ` · ${formatDate(item.checkedAt)}` : ''}` : 'Pendiente'}
                    {item.note && ` · ${item.note}`}
                  </span>
                </div>
              </li>
            ))}
          </ul>
        )}
      </Async>
    </section>
  )
}
