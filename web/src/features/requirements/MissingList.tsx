// «Qué falta para avanzar»: las condiciones que calcula el servidor, separando lo que bloquea
// de lo que solo avisa.
import { Link } from 'react-router'
import type { Missing } from '../../shared/api/types.ts'
import { TextWithCodes } from '../../shared/ui/Code.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'

export type RuleTabs = Record<string, { tab: string; label: string }>

// A qué pestaña de la ficha del REQ lleva cada regla.
const RULE_TABS: RuleTabs = {
  min_coverage: { tab: 'pruebas', label: 'Ver pruebas' },
  no_open_defect_with_severity: { tab: 'defectos', label: 'Ver defectos' },
  links_have_snapshot: { tab: 'enlaces', label: 'Ver enlaces' },
  entry_contract_complete: { tab: 'resumen', label: 'Ver contrato de entrada' },
  risk_assessed: { tab: 'resumen', label: 'Ver riesgo' },
  risk_has_quality_characteristics: { tab: 'resumen', label: 'Ver riesgo' },
}

function plural(count: number, one: string, many: string): string {
  return `${count} ${count === 1 ? one : many}`
}

/** `ruleTabs` cambia a qué pestaña lleva cada regla (la ficha del PKG tiene otras). */
export function MissingList({ missing, ruleTabs = RULE_TABS }: { missing: Missing[]; ruleTabs?: RuleTabs }) {
  const steps = missing.filter((step) => step.conditions.length > 0)
  if (steps.length === 0) {
    return (
      <section className="surface pad stack" aria-labelledby="h-falta">
        <h2 className="h2" id="h-falta">
          Qué falta para avanzar
        </h2>
        <p className="soft">{missing.length === 0 ? 'Este estado no tiene un paso siguiente.' : 'Nada. Ninguna condición detiene el siguiente paso.'}</p>
      </section>
    )
  }
  return (
    <>
      {steps.map((step, index) => {
        const blocking = step.conditions.filter((condition) => condition.blocks).length
        const warnings = step.conditions.length - blocking
        const headingId = index === 0 ? 'h-falta' : `h-falta-${step.to.key}`
        return (
          <section key={step.to.key} className="surface pad stack" aria-labelledby={headingId}>
            <div className="row row--between">
              <h2 className="h2" id={headingId}>
                Qué falta para avanzar a «{step.to.name}»
              </h2>
              <span className="muted">
                {[blocking > 0 && plural(blocking, 'condición pendiente', 'condiciones pendientes'), warnings > 0 && plural(warnings, 'aviso', 'avisos')]
                  .filter(Boolean)
                  .join(' · ')}
              </span>
            </div>
            <ul className="list">
              {step.conditions.map((condition) => {
                const target = ruleTabs[condition.ruleKey]
                return (
                  <li key={condition.ruleKey} data-blocks={condition.blocks}>
                    <span className={`list__icon list__icon--${condition.blocks ? 'bloq' : 'aviso'}`}>
                      <Icon name={condition.blocks ? 'lock' : 'triangle'} />
                    </span>
                    <div className="list__main">
                      <TextWithCodes text={condition.message} />
                      <br />
                      <span className="muted">{condition.blocks ? 'Bloquea el avance.' : 'No bloquea: solo avisa.'}</span>
                    </div>
                    {target && (
                      <Link className="btn" to={{ hash: target.tab }} replace preventScrollReset>
                        {target.label}
                      </Link>
                    )}
                  </li>
                )
              })}
            </ul>
          </section>
        )
      })}
    </>
  )
}
