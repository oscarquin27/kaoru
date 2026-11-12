// Regresión por impacto de un paquete (regla de negocio 4): el sistema propone, QA ajusta con su
// criterio y confirma. Confirmada, la selección no cambia.
import { useState } from 'react'
import type { Regression, RegressionItem } from '../../shared/api/types.ts'
import { formatDateTime } from '../../shared/lib/format.ts'
import { REGRESSION_ORIGIN_LABELS } from '../../shared/lib/labels.ts'
import { Chip, ComponentChips, ResultChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty, ErrorNotice } from '../../shared/ui/feedback.tsx'
import { Form, TextAreaField, TextField, text } from '../../shared/ui/Form.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { RunTarget } from '../../shared/ui/RunTarget.tsx'
import { useAdjustRegressionItem, useConfirmRegression, useProposeRegression, useRegression } from './api.ts'

const CODE = /^[A-Z]+-\d{4}-\d{4,}$/

function Summary({ summary }: { summary: Regression['summary'] }) {
  return (
    <section className="glass kpis" aria-label="Resumen de la regresión">
      <div className="kpi">
        <span className="kpi__label">Propuestos</span>
        <span className="kpi__value">{summary.proposed}</span>
        <span className="kpi__note">TC que propuso el sistema</span>
      </div>
      <div className="kpi">
        <span className="kpi__label">Incluidos</span>
        <span className="kpi__value">{summary.included}</span>
        <span className="kpi__note">
          {summary.added} agregados · {summary.removed} quitados por QA
        </span>
      </div>
      <div className="kpi">
        <span className="kpi__label">Avance</span>
        <span className="kpi__value">
          {summary.passed} de {summary.included}
        </span>
        <span className="row">
          <Chip tone="ok" icon="check">
            {summary.passed} aprobados
          </Chip>
          <Chip tone="bloq" icon="x">
            {summary.failed} fallidos
          </Chip>
          <Chip tone="neutro" icon="circle">
            {summary.pending} pendientes
          </Chip>
        </span>
      </div>
    </section>
  )
}

type Asking = { kind: 'adjust'; item: RegressionItem } | { kind: 'add' } | { kind: 'confirm' } | null

function Selection({ packageCode, regression, canManage }: { packageCode: string; regression: Regression; canManage: boolean }) {
  const propose = useProposeRegression(packageCode)
  // El ajuste directo pinta su error en la página; el que pide criterio, dentro de su formulario.
  const adjust = useAdjustRegressionItem(packageCode)
  const adjustWithReason = useAdjustRegressionItem(packageCode)
  const confirm = useConfirmRegression(packageCode)
  const [asking, setAsking] = useState<Asking>(null)
  const confirmed = regression.confirmedAt !== undefined
  const editable = canManage && !confirmed
  const busy = propose.isPending || adjust.isPending
  const error = propose.error ?? adjust.error
  const close = () => setAsking(null)

  /** Volver a lo propuesto no pide criterio; apartarse de lo propuesto, sí. */
  function toggle(item: RegressionItem) {
    const included = !item.included
    if (included === item.proposed) adjust.mutate({ testCase: item.testCase.code, included })
    else setAsking({ kind: 'adjust', item })
  }

  return (
    <div className="stack">
      <Summary summary={regression.summary} />
      <div className="surface pad-s row row--between">
        <span className="soft">
          {confirmed ? (
            <>
              Confirmada por {regression.confirmedBy?.displayName ?? 'QA'} el {formatDateTime(regression.confirmedAt ?? '')}. Ya no cambia.
            </>
          ) : (
            <>
              Propuesta por {regression.proposedBy.displayName} el {formatDateTime(regression.proposedAt)}. Sin confirmar: todavía se puede ajustar.
            </>
          )}
        </span>
        {editable && (
          <span className="row">
            <button className="btn" type="button" onClick={() => setAsking({ kind: 'add' })}>
              <Icon name="plus" />
              Agregar TC
            </button>
            <button className="btn" type="button" disabled={busy} title="Conserva los ajustes de QA" onClick={() => propose.mutate()}>
              Volver a proponer
            </button>
            <button className="btn btn--primary" type="button" onClick={() => setAsking({ kind: 'confirm' })}>
              <Icon name="lock" />
              Confirmar
            </button>
          </span>
        )}
      </div>
      {error && <ErrorNotice error={error} />}
      {regression.items.length === 0 ? (
        <Empty>La propuesta no trae ningún TC: ningún TC activo ejercita los componentes que tocan los REQ del paquete, y el sistema no tiene TC críticos.</Empty>
      ) : (
        <div className="surface table-wrap">
          <table className="table" aria-label="TC de la regresión">
            <thead>
              <tr>
                <th className="col-fixed">TC</th>
                <th>Caso de prueba</th>
                <th>Origen</th>
                <th>Componentes que cruzan</th>
                <th>Incluido</th>
                <th>Criterio</th>
                <th>Último RUN sobre un BUILD del paquete</th>
                {editable && (
                  <th className="col-actions">
                    <span className="sr-only">Acciones</span>
                  </th>
                )}
              </tr>
            </thead>
            <tbody>
              {regression.items.map((item) => (
                <tr key={item.testCase.code} data-included={item.included}>
                  <td className="col-fixed">
                    <Code code={item.testCase.code} />
                  </td>
                  <td>
                    <div className="cell-stack">
                      <span>{item.testCase.title}</span>
                      {item.isCritical && <span className="muted">Crítico</span>}
                    </div>
                  </td>
                  <td>{REGRESSION_ORIGIN_LABELS[item.origin]}</td>
                  <td>
                    <ComponentChips components={item.components} empty="Ninguno" />
                  </td>
                  <td>
                    {item.included ? (
                      <Chip tone="ok" icon="check">
                        Sí
                      </Chip>
                    ) : (
                      <Chip tone="neutro" icon="minus">
                        No
                      </Chip>
                    )}
                  </td>
                  <td className="wrap">
                    {item.reason ? (
                      <div className="cell-stack">
                        <span>{item.reason}</span>
                        {item.decidedBy && (
                          <span className="muted">
                            {item.decidedBy.displayName}
                            {item.decidedAt ? ` · ${formatDateTime(item.decidedAt)}` : ''}
                          </span>
                        )}
                      </div>
                    ) : (
                      <span className="muted">Como se propuso</span>
                    )}
                  </td>
                  <td>
                    {item.lastRun ? (
                      <div className="cell-stack">
                        <span className="cell-line">
                          <Code code={item.lastRun.code} />
                          <ResultChip result={item.lastRun.result} />
                        </span>
                        <RunTarget target={item.lastRun} />
                      </div>
                    ) : item.included ? (
                      <ResultChip result={undefined} />
                    ) : (
                      <span className="muted">—</span>
                    )}
                  </td>
                  {editable && (
                    <td className="col-actions">
                      <button className="btn" type="button" aria-label={`${item.included ? 'Quitar' : 'Incluir'} ${item.testCase.code}`} disabled={busy} onClick={() => toggle(item)}>
                        {item.included ? 'Quitar' : 'Incluir'}
                      </button>
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {asking?.kind === 'adjust' && (
        <Dialog title={`${asking.item.included ? 'Quitar' : 'Incluir'} ${asking.item.testCase.code}`} onClose={close}>
          <Form
            label="Criterio del ajuste"
            submitLabel={asking.item.included ? 'Quitar de la regresión' : 'Incluir en la regresión'}
            onSubmit={(data) => adjustWithReason.mutateAsync({ testCase: asking.item.testCase.code, included: !asking.item.included, reason: text(data, 'reason') })}
            onDone={close}
            onCancel={close}
          >
            <p className="soft">
              {asking.item.included ? 'El sistema propuso este TC. Para quitarlo' : 'El sistema no propuso este TC. Para incluirlo'}, deja el criterio: queda con tu nombre y la fecha.
            </p>
            <TextAreaField name="reason" label="Criterio" required />
          </Form>
        </Dialog>
      )}
      {asking?.kind === 'add' && (
        <Dialog title="Agregar TC a la regresión" onClose={close}>
          <Form
            label="Agregar TC"
            submitLabel="Agregar TC"
            validate={(data) => (CODE.test(text(data, 'testCase')) ? [] : [{ field: 'testCase', code: 'invalid', message: 'Escribe el código completo del TC, por ejemplo TC-2026-0001.' }])}
            onSubmit={(data) => adjustWithReason.mutateAsync({ testCase: text(data, 'testCase'), included: true, reason: text(data, 'reason') })}
            onDone={close}
            onCancel={close}
          >
            <TextField name="testCase" label="Código del TC" required placeholder="TC-2026-0001" spellCheck={false} autoComplete="off" />
            <TextAreaField name="reason" label="Criterio" required hint="Por qué entra un TC que el sistema no propuso." />
          </Form>
        </Dialog>
      )}
      {asking?.kind === 'confirm' && (
        <Dialog title="Confirmar la regresión" onClose={close}>
          <Form label="Confirmar la regresión" submitLabel="Confirmar regresión" onSubmit={() => confirm.mutateAsync()} onDone={close} onCancel={close}>
            <p>
              Quedan {regression.summary.included} TC en la selección final. Desde que se confirma, la selección no cambia: no se puede volver a proponer, agregar ni quitar TC.
            </p>
          </Form>
        </Dialog>
      )}
    </div>
  )
}

export function RegressionTab({ packageCode, canManage }: { packageCode: string; canManage: boolean }) {
  const regression = useRegression(packageCode)
  const propose = useProposeRegression(packageCode)

  // Un 404 no es un fallo: la regresión todavía no se propuso.
  if (regression.data === undefined && regression.error?.status === 404) {
    return (
      <div className="surface pad stack">
        <p>Este paquete todavía no tiene una selección de regresión.</p>
        <p className="soft">Al proponerla, el sistema toma los TC activos que ejercitan algún componente tocado por los REQ del paquete, más los TC críticos del sistema.</p>
        {canManage && (
          <div className="row">
            <button className="btn btn--primary" type="button" disabled={propose.isPending} onClick={() => propose.mutate()}>
              {propose.isPending ? 'Proponiendo…' : 'Proponer regresión'}
            </button>
          </div>
        )}
        {propose.error && <ErrorNotice error={propose.error} />}
      </div>
    )
  }
  return <Async query={regression}>{(data) => <Selection packageCode={packageCode} regression={data} canManage={canManage} />}</Async>
}
