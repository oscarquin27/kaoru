// Riesgo del REQ dentro de la ficha: lo vigente y el formulario para evaluar o reevaluar.
import { useState } from 'react'
import type { QualityCharacteristic, Risk, RiskLevel } from '../../shared/api/types.ts'
import { formatDate } from '../../shared/lib/format.ts'
import { QUALITY_LABELS, RISK_LABELS, options } from '../../shared/lib/labels.ts'
import { RiskChip } from '../../shared/ui/chips.tsx'
import { Async } from '../../shared/ui/feedback.tsx'
import { CheckGroupField, Form, SelectField, TextAreaField, text } from '../../shared/ui/Form.tsx'
import { MetaItem } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { useAssessRisk, useRisk } from './api.ts'

function RiskForm({ code, risk, onClose }: { code: string; risk?: Risk; onClose: () => void }) {
  const assess = useAssessRisk(code)
  return (
    <Form
      label="Evaluar riesgo"
      submitLabel={risk ? 'Guardar reevaluación' : 'Guardar evaluación'}
      onSubmit={(data) =>
        assess.mutateAsync({
          level: text(data, 'level') as RiskLevel,
          justification: text(data, 'justification'),
          characteristics: data.getAll('characteristics') as QualityCharacteristic[],
        })
      }
      onDone={onClose}
      onCancel={onClose}
    >
      <SelectField name="level" label="Nivel de riesgo" required defaultValue={risk?.level ?? ''} hint="El nivel fija cuánta cobertura de pruebas se exige.">
        <option value="">Elige un nivel</option>
        {options(RISK_LABELS).map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </SelectField>
      <TextAreaField name="justification" label="Justificación" required defaultValue={risk?.justification} />
      <CheckGroupField name="characteristics" label="Características de calidad afectadas (ISO/IEC 25010)" choices={options(QUALITY_LABELS)} selected={risk?.characteristics} />
    </Form>
  )
}

export function RiskCard({ code, assessed, canAssess }: { code: string; assessed: boolean; canAssess: boolean }) {
  const risk = useRisk(code, assessed)
  const [editing, setEditing] = useState(false)
  return (
    <section className="surface pad stack" aria-labelledby="h-riesgo">
      <div className="row row--between">
        <h3 className="h2" id="h-riesgo">
          Riesgo
        </h3>
        {canAssess && (
          <button className="btn" type="button" onClick={() => setEditing(true)}>
            {assessed ? 'Reevaluar riesgo' : 'Evaluar riesgo'}
          </button>
        )}
      </div>
      {!assessed ? (
        <p className="soft">Sin evaluar. El REQ no avanza a «Riesgo evaluado» hasta que alguien lo evalúe.</p>
      ) : (
        <Async query={risk}>
          {(data) => (
            <dl className="meta stack">
              <MetaItem label="Nivel">
                <RiskChip level={data.level} /> <span className="id">{data.code}</span>
              </MetaItem>
              <MetaItem label="Justificación">{data.justification}</MetaItem>
              <MetaItem label="Características de calidad">
                {data.characteristics.length > 0 ? data.characteristics.map((item) => QUALITY_LABELS[item]).join(' · ') : <span className="muted">Ninguna</span>}
              </MetaItem>
              <MetaItem label="Evaluó">
                {data.assessedBy.displayName} · {formatDate(data.updatedAt)}
              </MetaItem>
            </dl>
          )}
        </Async>
      )}
      {editing && (
        <Dialog title={assessed ? 'Reevaluar riesgo' : 'Evaluar riesgo'} onClose={() => setEditing(false)}>
          <RiskForm code={code} risk={risk.data} onClose={() => setEditing(false)} />
        </Dialog>
      )}
    </section>
  )
}
