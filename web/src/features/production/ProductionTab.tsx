// Pestaña «Producción» del PKG: su checklist, sus diagnósticos y sus pruebas de humo, con las
// acciones de cada uno. Qué exige el flujo depende de la ruta del PKG (requerimiento §8.2).
import { useState, type ReactNode } from 'react'
import { useCan } from '../../shared/api/session.ts'
import type { Route } from '../../shared/api/types.ts'
import { Async } from '../../shared/ui/feedback.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { useChecklists, useDiagnostics, useSmokeTests } from './api.ts'
import { ChecklistsTable } from './ChecklistsTable.tsx'
import { DiagnosticForm, SmokeForm, StartChecklistForm } from './forms.tsx'
import { DiagnosticsTable, SmokeTestsTable } from './VerificationTables.tsx'

const ROUTE_RULES: Record<Route, string> = {
  high: 'Ruta alta: el PKG pasa a desplegado con el último checklist firmado entero, y a verificado con el último diagnóstico conforme.',
  light: 'Ruta ligera: el PKG pasa a verificado con la última prueba de humo aprobada.',
  minor_fix: 'Fix menor: el flujo no exige checklist, diagnóstico ni prueba de humo, aunque se pueden registrar.',
}

type Action = 'checklist' | 'diagnostic' | 'smoke'

function Part({ id, title, action, children }: { id: string; title: string; action?: ReactNode; children: ReactNode }) {
  return (
    <section className="stack" aria-labelledby={id}>
      <div className="row row--between">
        <h2 className="h3" id={id}>
          {title}
        </h2>
        {action}
      </div>
      {children}
    </section>
  )
}

export function ProductionTab({ packageCode, systemKey, route }: { packageCode: string; systemKey: string; route?: Route }) {
  const can = useCan()
  const [action, setAction] = useState<Action | null>(null)
  const filter = { package: packageCode, size: 50 }
  const checklists = useChecklists(filter)
  const diagnostics = useDiagnostics(filter)
  const smokeTests = useSmokeTests(filter)
  const close = () => setAction(null)
  const button = (kind: Action, label: string) => (
    <button className="btn" type="button" onClick={() => setAction(kind)}>
      <Icon name="plus" />
      {label}
    </button>
  )

  return (
    <div className="stack">
      {route && (
        <div className="notice" role="note">
          <Icon name="info" />
          <div className="notice__main">{ROUTE_RULES[route]} Cuenta siempre lo último de cada cosa.</div>
        </div>
      )}
      <Part id="h-chk" title="Checklist de producción" action={can('chk.start') && button('checklist', 'Abrir checklist')}>
        <Async query={checklists}>{(data) => <ChecklistsTable items={data.items} empty="El PKG no tiene checklist de producción." />}</Async>
      </Part>
      <Part id="h-diag" title="Diagnósticos" action={can('diag.submit') && button('diagnostic', 'Entregar DIAG')}>
        <Async query={diagnostics}>{(data) => <DiagnosticsTable items={data.items} empty="El PKG no tiene diagnósticos." />}</Async>
      </Part>
      <Part id="h-smoke" title="Pruebas de humo" action={can('smoke.create') && button('smoke', 'Registrar SMOKE')}>
        <Async query={smokeTests}>{(data) => <SmokeTestsTable items={data.items} empty="El PKG no tiene pruebas de humo." />}</Async>
      </Part>
      {action === 'checklist' && (
        <Dialog title={`Abrir checklist de ${packageCode}`} onClose={close}>
          <StartChecklistForm packageCode={packageCode} systemKey={systemKey} onClose={close} />
        </Dialog>
      )}
      {action === 'diagnostic' && (
        <Dialog title={`Entregar diagnóstico de ${packageCode}`} onClose={close}>
          <DiagnosticForm packageCode={packageCode} onClose={close} />
        </Dialog>
      )}
      {action === 'smoke' && (
        <Dialog title={`Registrar prueba de humo de ${packageCode}`} onClose={close}>
          <SmokeForm packageCode={packageCode} onClose={close} />
        </Dialog>
      )}
    </div>
  )
}
