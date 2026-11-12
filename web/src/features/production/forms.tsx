// Formularios de producción: abrir el checklist, firmar un paso, entregar y revisar un DIAG,
// registrar un SMOKE y publicar una versión de la plantilla. Nada de esto se edita después.
import { useEnvironmentOptions } from '../../shared/api/catalog.ts'
import type { FieldError } from '../../shared/api/errors.ts'
import { useApiQuery } from '../../shared/api/query.ts'
import { useSystems } from '../../shared/api/session.ts'
import type { Checklist, ChecklistStep, ChecklistTemplate, Deployment, Diagnostic, DiagnosticResult, Page, SmokeResult } from '../../shared/api/types.ts'
import { formatDateTime } from '../../shared/lib/format.ts'
import { DEPLOYMENT_RESULT_LABELS, DIAGNOSTIC_RESULT_LABELS, SMOKE_RESULT_LABELS, options } from '../../shared/lib/labels.ts'
import { Form, SelectField, TextAreaField, text } from '../../shared/ui/Form.tsx'
import { usePublishChecklistTemplate, useRegisterSmokeTest, useReviewDiagnostic, useSignChecklistStep, useStartChecklist, useSubmitDiagnostic } from './api.ts'

/** Despliegues del PKG en producción: son los únicos a los que se les hace DIAG o SMOKE. */
function useProductionDeployments(packageCode: string) {
  const deployments = useApiQuery<Page<Deployment>>('/api/deployments', { package: packageCode, size: 200 }).data?.items ?? []
  return deployments.filter((deployment) => deployment.environment.kind === 'production')
}

function DeploymentField({ packageCode }: { packageCode: string }) {
  const deployments = useProductionDeployments(packageCode)
  return (
    // La llave vuelve a montar el campo cuando llega la lista, para que tome el más reciente por omisión.
    <SelectField name="deployment" label="Despliegue" required defaultValue={deployments[0]?.code ?? ''} key={deployments.length} hint="Despliegues del PKG en producción, el más reciente primero.">
      <option value="">Elige el despliegue</option>
      {deployments.map((deployment) => (
        <option key={deployment.code} value={deployment.code}>
          {deployment.code} · {deployment.environment.name} · {formatDateTime(deployment.executedAt)} · {DEPLOYMENT_RESULT_LABELS[deployment.result]}
        </option>
      ))}
    </SelectField>
  )
}

export function StartChecklistForm({ packageCode, systemKey, onClose }: { packageCode: string; systemKey: string; onClose: () => void }) {
  const start = useStartChecklist()
  const environments = (useEnvironmentOptions(systemKey).data?.items ?? []).filter((environment) => environment.kind === 'production')
  return (
    <Form
      label="Abrir checklist"
      submitLabel="Abrir CHK"
      onSubmit={(data) => start.mutateAsync({ package: packageCode, environment: text(data, 'environment') })}
      onDone={onClose}
      onCancel={onClose}
    >
      <p className="soft">El checklist toma la última versión de la plantilla del sistema. Cada paso se firma una vez y la firma no se quita.</p>
      <SelectField name="environment" label="Ambiente de producción" required defaultValue={environments[0]?.code ?? ''} key={environments.length}>
        <option value="">Elige el ambiente</option>
        {environments.map((environment) => (
          <option key={environment.code} value={environment.code}>
            {environment.name} · {environment.code}
          </option>
        ))}
      </SelectField>
    </Form>
  )
}

export function SignStepForm({ checklist, step, onClose }: { checklist: Checklist; step: ChecklistStep; onClose: () => void }) {
  const sign = useSignChecklistStep(checklist.code)
  return (
    <Form label="Firmar paso" submitLabel="Firmar" onSubmit={(data) => sign.mutateAsync({ position: step.position, comment: text(data, 'comment') })} onDone={onClose} onCancel={onClose}>
      <p className="soft">
        Firmas el paso {step.position}, «{step.title}», con tu usuario y la hora del servidor. Una firma no se cambia ni se quita.
      </p>
      {step.description && <p className="wrap">{step.description}</p>}
      <TextAreaField name="comment" label="Comentario" hint="Opcional. La evidencia se adjunta como documento del checklist." />
    </Form>
  )
}

export function DiagnosticForm({ packageCode, onClose }: { packageCode: string; onClose: () => void }) {
  const submit = useSubmitDiagnostic()
  return (
    <Form label="Entregar diagnóstico" submitLabel="Entregar DIAG" onSubmit={(data) => submit.mutateAsync({ deployment: text(data, 'deployment'), notes: text(data, 'notes') })} onDone={onClose} onCancel={onClose}>
      <p className="soft">Lo que dev u ops vieron después del despliegue. QA lo revisa y marca si está conforme; un DIAG no se corrige, se entrega otro.</p>
      <DeploymentField packageCode={packageCode} />
      <TextAreaField name="notes" label="Notas" hint="Qué se revisó y qué se encontró. Los archivos se adjuntan como documentos del DIAG." />
    </Form>
  )
}

function validateReview(data: FormData): FieldError[] {
  if (text(data, 'result') === 'non_conforming' && !text(data, 'notes')) {
    return [{ field: 'notes', code: 'required', message: 'Di por qué no está conforme.' }]
  }
  return []
}

export function ReviewForm({ diagnostic, onClose }: { diagnostic: Diagnostic; onClose: () => void }) {
  const review = useReviewDiagnostic(diagnostic.code)
  return (
    <Form
      label="Revisar diagnóstico"
      submitLabel="Guardar revisión"
      validate={validateReview}
      onSubmit={(data) => review.mutateAsync({ result: text(data, 'result') as DiagnosticResult, notes: text(data, 'notes') })}
      onDone={onClose}
      onCancel={onClose}
    >
      <p className="soft">La revisión se marca una vez. Si el resultado cambia, dev u ops entregan otro DIAG.</p>
      <SelectField name="result" label="Resultado" required defaultValue="">
        <option value="">Elige el resultado</option>
        {options(DIAGNOSTIC_RESULT_LABELS).map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </SelectField>
      <TextAreaField name="notes" label="Notas" hint="Obligatorias si no está conforme." />
    </Form>
  )
}

function validateSmoke(data: FormData): FieldError[] {
  if (text(data, 'result') === 'failed' && !text(data, 'notes')) {
    return [{ field: 'notes', code: 'required', message: 'Di qué falló.' }]
  }
  return []
}

export function SmokeForm({ packageCode, onClose }: { packageCode: string; onClose: () => void }) {
  const register = useRegisterSmokeTest()
  return (
    <Form
      label="Registrar prueba de humo"
      submitLabel="Registrar SMOKE"
      validate={validateSmoke}
      onSubmit={(data) => register.mutateAsync({ deployment: text(data, 'deployment'), result: text(data, 'result') as SmokeResult, notes: text(data, 'notes') })}
      onDone={onClose}
      onCancel={onClose}
    >
      <p className="soft">Cuenta la última prueba de humo de cada PKG. No se corrige: si se repite, se registra otra.</p>
      <DeploymentField packageCode={packageCode} />
      <SelectField name="result" label="Resultado" required defaultValue="">
        <option value="">Elige el resultado</option>
        {options(SMOKE_RESULT_LABELS).map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </SelectField>
      <TextAreaField name="notes" label="Notas" hint="Obligatorias si falló." />
    </Form>
  )
}

/** Un paso por renglón; la descripción, si la hay, va después de una barra: «Respaldar la base | Con pg_dump». */
export function parseSteps(raw: string): { title: string; description: string }[] {
  return raw
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const [title = '', ...rest] = line.split('|')
      return { title: title.trim(), description: rest.join('|').trim() }
    })
}

function stepsText(template: ChecklistTemplate | undefined): string {
  return (template?.steps ?? []).map((step) => (step.description ? `${step.title} | ${step.description}` : step.title)).join('\n')
}

export function TemplateForm({ current, onClose }: { current?: ChecklistTemplate; onClose: () => void }) {
  const publish = usePublishChecklistTemplate()
  const systems = useSystems().data?.items ?? []
  return (
    <Form
      label="Publicar plantilla"
      submitLabel="Publicar versión"
      validate={(data) => (parseSteps(text(data, 'steps')).length === 0 ? [{ field: 'steps', code: 'required', message: 'Escribe al menos un paso.' }] : [])}
      onSubmit={(data) => publish.mutateAsync({ system: text(data, 'system'), steps: parseSteps(text(data, 'steps')) })}
      onDone={onClose}
      onCancel={onClose}
    >
      <p className="soft">Una plantilla no se edita: se publica otra versión. Los checklists ya abiertos conservan la suya.</p>
      <SelectField name="system" label="Sistema" required defaultValue={current?.system.key ?? ''} key={systems.length}>
        <option value="">Elige el sistema</option>
        {systems.map((system) => (
          <option key={system.key} value={system.key}>
            {system.name}
          </option>
        ))}
      </SelectField>
      <TextAreaField name="steps" label="Pasos" required rows={8} defaultValue={stepsText(current)} hint="Un paso por renglón. Para una descripción, sepárala con «|»." />
    </Form>
  )
}
