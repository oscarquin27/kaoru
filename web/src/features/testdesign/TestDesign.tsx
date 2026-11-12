// Diseño de pruebas dentro de la ficha del REQ: casos de uso (UC) y casos de prueba (TC).
import { useState, type ReactNode } from 'react'
import type { FunctionalCase, TestCaseSummary } from '../../shared/api/types.ts'
import { ResultChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Empty } from '../../shared/ui/feedback.tsx'
import { CheckField, CheckGroupField, Form, TextAreaField, TextField, text } from '../../shared/ui/Form.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { RunTarget } from '../../shared/ui/RunTarget.tsx'
import { useCreateFunctionalCase, useCreateTestCase } from './api.ts'

function FunctionalCaseForm({ requirementCode, onClose }: { requirementCode: string; onClose: () => void }) {
  const create = useCreateFunctionalCase(requirementCode)
  return (
    <Form
      label="Definir caso de uso"
      submitLabel="Definir UC"
      onSubmit={(data) => create.mutateAsync({ title: text(data, 'title'), description: text(data, 'description') })}
      onDone={onClose}
      onCancel={onClose}
    >
      <TextField name="title" label="Título" required />
      <TextAreaField name="description" label="Descripción" />
    </Form>
  )
}

function TestCaseForm({ systemKey, functionalCases, onClose }: { systemKey: string; functionalCases: FunctionalCase[]; onClose: () => void }) {
  const create = useCreateTestCase()
  return (
    <Form
      label="Definir caso de prueba"
      submitLabel="Definir TC"
      validate={(data) =>
        data.getAll('functionalCases').length === 0 ? [{ field: 'functionalCases', code: 'required', message: 'Elige al menos un caso de uso que este TC valida.' }] : []
      }
      onSubmit={(data) =>
        create.mutateAsync({
          title: text(data, 'title'),
          system: systemKey,
          preconditions: text(data, 'preconditions'),
          steps: text(data, 'steps'),
          expectedResult: text(data, 'expectedResult'),
          isCritical: data.get('isCritical') === 'on',
          functionalCases: data.getAll('functionalCases') as string[],
        })
      }
      onDone={onClose}
      onCancel={onClose}
    >
      <TextField name="title" label="Título" required />
      <TextAreaField name="preconditions" label="Precondiciones" />
      <TextAreaField name="steps" label="Pasos" required />
      <TextAreaField name="expectedResult" label="Resultado esperado" required />
      <CheckGroupField
        name="functionalCases"
        label="Casos de uso que valida"
        choices={functionalCases.map((useCase) => ({ value: useCase.code, label: `${useCase.code} · ${useCase.title}` }))}
      />
      <CheckField name="isCritical" label="Es un caso crítico" />
    </Form>
  )
}

interface TestDesignProps {
  requirementCode: string
  systemKey: string
  functionalCases: FunctionalCase[]
  testCases: TestCaseSummary[]
  canManage: boolean
  /** Acción por TC que aporta otra pieza de la ficha (registrar un RUN). */
  testCaseAction?: (testCase: TestCaseSummary) => ReactNode
}

export function TestDesign({ requirementCode, systemKey, functionalCases, testCases, canManage, testCaseAction }: TestDesignProps) {
  const [form, setForm] = useState<'uc' | 'tc' | null>(null)
  const close = () => setForm(null)
  return (
    <>
      <section className="stack" aria-labelledby="h-uc">
        <div className="row row--between">
          <h3 className="h2" id="h-uc">
            Casos de uso
          </h3>
          {canManage && (
            <button className="btn" type="button" onClick={() => setForm('uc')}>
              <Icon name="plus" />
              Definir UC
            </button>
          )}
        </div>
        {functionalCases.length === 0 ? (
          <Empty>Sin casos de uso. Define el primero para poder escribir casos de prueba.</Empty>
        ) : (
          <div className="surface table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>UC</th>
                  <th>Caso de uso</th>
                  <th>TC que lo validan</th>
                </tr>
              </thead>
              <tbody>
                {functionalCases.map((useCase) => (
                  <tr key={useCase.code}>
                    <td>
                      <Code code={useCase.code} />
                    </td>
                    <td>{useCase.title}</td>
                    <td>
                      <span className="cell-line">
                        {useCase.testCases.length === 0 ? <span className="muted">Sin TC</span> : useCase.testCases.map((code) => <Code key={code} code={code} />)}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="stack" aria-labelledby="h-tc">
        <div className="row row--between">
          <h3 className="h2" id="h-tc">
            Casos de prueba
          </h3>
          {canManage && functionalCases.length > 0 && (
            <button className="btn" type="button" onClick={() => setForm('tc')}>
              <Icon name="plus" />
              Definir TC
            </button>
          )}
        </div>
        {testCases.length === 0 ? (
          <Empty>Sin casos de prueba.</Empty>
        ) : (
          <div className="surface table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>TC</th>
                  <th>Caso de prueba</th>
                  <th>Crítico</th>
                  <th>Último RUN</th>
                  <th>BUILD · ENV</th>
                  <th>Resultado</th>
                  {testCaseAction && (
                    <th className="col-actions">
                      <span className="sr-only">Acciones</span>
                    </th>
                  )}
                </tr>
              </thead>
              <tbody>
                {testCases.map((testCase) => (
                  <tr key={testCase.code}>
                    <td>
                      <Code code={testCase.code} />
                    </td>
                    <td>
                      {testCase.title}
                      {testCase.retired && <span className="muted"> · retirado</span>}
                    </td>
                    <td>{testCase.isCritical ? 'Sí' : 'No'}</td>
                    <td>{testCase.lastRun ? <Code code={testCase.lastRun.code} /> : <span className="muted">Sin RUN</span>}</td>
                    <td>
                      {testCase.lastRun ? (
                        <RunTarget target={testCase.lastRun} />
                      ) : (
                        <span className="muted">—</span>
                      )}
                    </td>
                    <td>
                      <ResultChip result={testCase.lastRun?.result} />
                    </td>
                    {testCaseAction && <td className="col-actions">{testCaseAction(testCase)}</td>}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {form === 'uc' && (
        <Dialog title="Definir caso de uso" onClose={close}>
          <FunctionalCaseForm requirementCode={requirementCode} onClose={close} />
        </Dialog>
      )}
      {form === 'tc' && (
        <Dialog title="Definir caso de prueba" onClose={close}>
          <TestCaseForm systemKey={systemKey} functionalCases={functionalCases} onClose={close} />
        </Dialog>
      )}
    </>
  )
}
