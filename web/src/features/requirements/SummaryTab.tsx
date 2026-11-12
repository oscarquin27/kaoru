// Pestaña Resumen: descripción y criterios de aceptación, con edición.
import { useState } from 'react'
import type { RequirementOverview } from '../../shared/api/types.ts'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { RequirementForm } from './RequirementForm.tsx'

export function SummaryTab({ requirement, canEdit }: { requirement: RequirementOverview['requirement']; canEdit: boolean }) {
  const [editing, setEditing] = useState(false)
  const criteria = requirement.acceptanceCriteria
    .split('\n')
    .map((line) => line.replace(/^\s*[-*]\s*/, '').trim())
    .filter(Boolean)
  return (
    <div className="surface pad stack">
      <div className="row row--between">
        <h3 className="h3">Descripción</h3>
        {canEdit && (
          <button className="btn" type="button" onClick={() => setEditing(true)}>
            <Icon name="pencil" />
            Editar
          </button>
        )}
      </div>
      <p className="wrap">{requirement.description || <span className="muted">Sin descripción.</span>}</p>
      <h3 className="h3">Criterios de aceptación</h3>
      {criteria.length === 0 ? (
        <p className="muted">Sin criterios de aceptación.</p>
      ) : (
        <ul className="list">
          {criteria.map((line) => (
            <li key={line}>
              <span className="list__icon">
                <Icon name="circle" />
              </span>
              <div className="list__main">{line}</div>
            </li>
          ))}
        </ul>
      )}
      {editing && (
        <Dialog title={`Editar ${requirement.code}`} onClose={() => setEditing(false)}>
          <RequirementForm requirement={requirement} onClose={() => setEditing(false)} />
        </Dialog>
      )}
    </div>
  )
}
