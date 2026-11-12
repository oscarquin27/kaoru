// Reglas de bloqueo de los flujos: cada una se puede desactivar o dejar en «solo avisa».
import { useCan, useWorkflow } from '../../shared/api/session.ts'
import type { Workflow } from '../../shared/api/types.ts'
import { ROUTE_LABELS, labelOf } from '../../shared/lib/labels.ts'
import { Async, ErrorNotice } from '../../shared/ui/feedback.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { useUpdateBlockingRule } from './api.ts'

// Qué comprueba cada regla. Una regla nueva del servidor sale con su clave hasta que se nombre aquí.
const RULE_LABELS: Record<string, string> = {
  risk_assessed: 'El riesgo está evaluado',
  risk_has_quality_characteristics: 'El riesgo indica características de calidad',
  entry_contract_complete: 'El contrato de entrada está completo',
  links_have_snapshot: 'Los enlaces de Google Sheets tienen instantánea',
  min_coverage: 'Cobertura mínima de TC con RUN aprobado (%)',
  no_open_defect_with_severity: 'Sin defectos abiertos de esa severidad',
  defect_severity_is_not: 'El defecto no es de esa severidad',
  package_has_content: 'El paquete tiene al menos un BUILD y un REQ',
  regression_confirmed: 'La selección de regresión está confirmada',
  regression_passed: 'Toda la regresión tiene su último RUN aprobado sobre un BUILD del paquete',
  deployed_to_production: 'Hay un despliegue exitoso del paquete en producción',
}

function WorkflowRules({ title, workflow, editable }: { title: string; workflow: Workflow; editable: boolean }) {
  const update = useUpdateBlockingRule()
  const stateName = (key: string) => workflow.states.find((state) => state.key === key)?.name ?? key
  const rows = workflow.transitions.flatMap((transition) => transition.rules.map((rule) => ({ transition, rule })))
  if (rows.length === 0) return null
  return (
    <section className="stack" aria-label={title}>
      <h2 className="h2">{title}</h2>
      {update.error && <ErrorNotice error={update.error} />}
      <div className="surface table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>Transición</th>
              <th>Regla</th>
              <th>Ruta</th>
              <th>Parámetros</th>
              <th>Bloquea</th>
              <th>Activa</th>
            </tr>
          </thead>
          <tbody>
            {rows.map(({ transition, rule }) => {
              const label = labelOf(RULE_LABELS, rule.ruleKey)
              const params = Object.entries(rule.params)
              return (
                <tr key={rule.id}>
                  <td>
                    {stateName(transition.from)} → {stateName(transition.to)}
                  </td>
                  <td>{label}</td>
                  <td>{rule.route ? ROUTE_LABELS[rule.route] : <span className="muted">Todas</span>}</td>
                  <td>{params.length === 0 ? <span className="muted">—</span> : <span className="mono">{params.map(([key, value]) => `${key}: ${String(value)}`).join(' · ')}</span>}</td>
                  <td>
                    <label className="check">
                      <input
                        type="checkbox"
                        aria-label={`Bloquea: ${label}`}
                        title="Sin marcar, la regla solo avisa"
                        checked={rule.blocks}
                        disabled={!editable || update.isPending}
                        onChange={(event) => update.mutate({ id: rule.id, patch: { blocks: event.target.checked } })}
                      />
                    </label>
                  </td>
                  <td>
                    <label className="check">
                      <input
                        type="checkbox"
                        aria-label={`Activa: ${label}`}
                        checked={rule.enabled}
                        disabled={!editable || update.isPending}
                        onChange={(event) => update.mutate({ id: rule.id, patch: { enabled: event.target.checked } })}
                      />
                    </label>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </section>
  )
}

export default function BlockingRulesPage() {
  const editable = useCan()('admin.configure')
  const requirement = useWorkflow('requirement')
  const defect = useWorkflow('defect')
  const change = useWorkflow('change')
  const bundle = useWorkflow('package')
  return (
    <>
      <Crumbs items={[{ label: 'Administración', to: '/administracion' }, { label: 'Reglas de bloqueo' }]} />
      <PageHead title="Reglas de bloqueo" subtitle="Condiciones que una transición exige. Una regla que no bloquea solo avisa." />
      <Async query={requirement}>{(data) => <WorkflowRules title="Requerimientos" workflow={data} editable={editable} />}</Async>
      <Async query={defect}>{(data) => <WorkflowRules title="Defectos" workflow={data} editable={editable} />}</Async>
      <Async query={change}>{(data) => <WorkflowRules title="Cambios" workflow={data} editable={editable} />}</Async>
      <Async query={bundle}>{(data) => <WorkflowRules title="Paquetes" workflow={data} editable={editable} />}</Async>
    </>
  )
}
