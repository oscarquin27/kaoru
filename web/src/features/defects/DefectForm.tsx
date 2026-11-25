// Alta y edición de un DEF. Abrir uno exige el REQ y el RUN que lo reprodujo.
// Desde la ficha del REQ, si el RUN todavía no está registrado, se elige «Registrar un RUN nuevo»
// (y, si hace falta, su BUILD): se registran al guardar, antes del DEF.
import { useRef, useState } from 'react'
import { useNavigate } from 'react-router'
import { api } from '../../shared/api/client.ts'
import { ApiError, toApiError, type FieldError } from '../../shared/api/errors.ts'
import type { Defect, Run, RunResult, Severity } from '../../shared/api/types.ts'
import { fromLocalInput, toLocalInput } from '../../shared/lib/format.ts'
import { RESULT_LABELS, SEVERITY_LABELS, options } from '../../shared/lib/labels.ts'
import { Form, SelectField, TextAreaField, TextField, optional, text } from '../../shared/ui/Form.tsx'
import { UserField } from '../../shared/ui/UserField.tsx'
import { TargetFields, resolveBuild, validateNewBuild, type Registered } from '../executions/Target.tsx'
import { useCreateDefect, useUpdateDefect } from './api.ts'

/** Valor del selector que pide registrar un RUN nuevo. */
const NEW_RUN = '__new__'
const RUN_FIELDS: Record<string, string> = { testCase: 'newRunTestCase', result: 'newRunResult', executedAt: 'newRunExecutedAt', notes: 'description' }

interface DefectFormProps {
  /** Con `defect`, edita. */
  defect?: Defect
  /** REQ ya conocido (alta desde la ficha del REQ). */
  requirement?: string
  /** RUN del REQ entre los que elegir; sin lista, se escribe el código. */
  runs?: { code: string; label: string }[]
  /** TC activos del REQ y su sistema: con ellos se puede registrar el RUN al vuelo. */
  testCases?: { code: string; label: string }[]
  systemKey?: string
  onClose: () => void
}

function validateCodes(data: FormData): FieldError[] {
  if (text(data, 'run') === NEW_RUN) {
    const errors = validateNewBuild(data)
    if (new Date(text(data, 'newRunExecutedAt')).getTime() > Date.now()) {
      errors.push({ field: 'newRunExecutedAt', code: 'invalid', message: 'La fecha de ejecución no puede ser futura.' })
    }
    return errors
  }
  const errors: FieldError[] = []
  for (const [field, prefix] of [['requirement', 'REQ'], ['run', 'RUN']] as const) {
    const value = text(data, field)
    if (data.has(field) && value && !new RegExp(`^${prefix}-\\d{4}-\\d{4,}$`).test(value)) {
      errors.push({ field, code: 'invalid', message: `Escribe el ID completo, por ejemplo ${prefix}-2026-0001.` })
    }
  }
  return errors
}

export function DefectForm({ defect, requirement, runs, testCases, systemKey, onClose }: DefectFormProps) {
  const navigate = useNavigate()
  const create = useCreateDefect()
  const update = useUpdateDefect(defect?.code ?? '')
  // Registrar el RUN aquí solo se puede con el REQ conocido: de él salen los TC y el sistema.
  const canRegisterRun = !defect && requirement !== undefined && runs !== undefined && testCases !== undefined && systemKey !== undefined
  const [run, setRun] = useState(canRegisterRun && runs.length === 0 ? NEW_RUN : '')
  const registered = useRef<Registered>(new Map())

  /** El código del RUN elegido; si se pidió uno nuevo, lo registra (con su BUILD) y devuelve su código. */
  async function resolveRun(data: FormData): Promise<string> {
    const chosen = text(data, 'run')
    if (chosen !== NEW_RUN || !systemKey) return chosen
    const build = await resolveBuild(data, { system: systemKey, requirement, registered: registered.current })
    const input = {
      testCase: text(data, 'newRunTestCase'),
      build,
      environment: text(data, 'environment'),
      result: text(data, 'newRunResult') as RunResult,
      notes: text(data, 'description'),
      executedAt: fromLocalInput(text(data, 'newRunExecutedAt')) ?? '',
    }
    const key = `run|${JSON.stringify(input)}`
    const known = registered.current.get(key)
    if (known) return known
    try {
      const created = await api.post<Run>('/api/runs', input)
      registered.current.set(key, created.code)
      return created.code
    } catch (caught) {
      // Los errores del RUN se pintan bajo sus campos de este formulario.
      const error = toApiError(caught)
      const fieldErrors: FieldError[] = error.fieldErrors.map((item) => ({ ...item, field: RUN_FIELDS[item.field] ?? item.field }))
      throw new ApiError(error.status, error.code, error.message, fieldErrors, error.traceId)
    }
  }

  async function submit(data: FormData) {
    const common = { title: text(data, 'title'), description: text(data, 'description'), severity: text(data, 'severity') as Severity }
    if (defect) {
      await update.mutateAsync({ ...common, assignee: optional(data, 'assignee') ?? null })
      return
    }
    const runCode = await resolveRun(data)
    const created = await create.mutateAsync({ ...common, assignee: optional(data, 'assignee'), requirement: requirement ?? text(data, 'requirement'), run: runCode })
    void navigate(`/defectos/${created.code}`)
  }

  return (
    <Form label={defect ? 'Editar defecto' : 'Abrir defecto'} submitLabel={defect ? 'Guardar' : 'Abrir DEF'} validate={validateCodes} onSubmit={submit} onDone={onClose} onCancel={onClose}>
      <TextField name="title" label="Título" required defaultValue={defect?.title} />
      <TextAreaField name="description" label="Descripción" required defaultValue={defect?.description} hint="Qué pasó, qué se esperaba y cómo reproducirlo." />
      <div className="form__row">
        <SelectField name="severity" label="Severidad" required defaultValue={defect?.severity ?? ''}>
          <option value="">Elige la severidad</option>
          {options(SEVERITY_LABELS).map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </SelectField>
        {!defect && !requirement && <TextField name="requirement" label="Requerimiento (REQ)" required placeholder="REQ-2026-0001" />}
        {!defect &&
          (runs ? (
            <SelectField name="run" label="RUN que lo reprodujo" required value={run} onChange={(event) => setRun(event.target.value)}>
              <option value="">Elige un RUN</option>
              {runs.map((item) => (
                <option key={item.code} value={item.code}>
                  {item.label}
                </option>
              ))}
              {canRegisterRun && <option value={NEW_RUN}>＋ Registrar un RUN nuevo…</option>}
            </SelectField>
          ) : (
            <TextField name="run" label="RUN que lo reprodujo" required placeholder="RUN-2026-0001" />
          ))}
      </div>
      {canRegisterRun && run === NEW_RUN && (
        <fieldset className="subform stack-s">
          <legend>RUN nuevo</legend>
          {testCases.length === 0 ? (
            <p className="muted">Este REQ no tiene TC activos: define uno en la pestaña Pruebas para poder reproducir el defecto.</p>
          ) : (
            <>
              <div className="form__row">
                <SelectField name="newRunTestCase" label="TC que lo reprodujo" required defaultValue={testCases.length === 1 ? (testCases[0]?.code ?? '') : ''}>
                  <option value="">Elige el TC</option>
                  {testCases.map((testCase) => (
                    <option key={testCase.code} value={testCase.code}>
                      {testCase.label}
                    </option>
                  ))}
                </SelectField>
                <SelectField name="newRunResult" label="Resultado del RUN" required defaultValue="failed">
                  {options(RESULT_LABELS)
                    .filter((option) => option.value !== 'passed')
                    .map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                </SelectField>
                <TextField name="newRunExecutedAt" label="Fecha de ejecución" type="datetime-local" required defaultValue={toLocalInput(new Date())} />
              </div>
              <TargetFields systemKey={systemKey} official />
              <p className="muted">El RUN se registra al guardar, con la descripción del defecto como notas, y queda como evidencia: no se edita después.</p>
            </>
          )}
        </fieldset>
      )}
      <UserField name="assignee" label="Asignado a" current={defect?.assignee} />
      {!defect && !canRegisterRun && <p className="muted">Un hallazgo de una exploración no se abre como defecto sin reproducirlo antes en un RUN. Para registrar el RUN aquí mismo, abre el DEF desde la ficha del REQ, pestaña Defectos.</p>}
    </Form>
  )
}
