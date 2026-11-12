// Ficha del PKG: estado, qué falta para avanzar, contenido, regresión, producción, informes,
// documentos y actividad.
// Junta piezas de varios features, por eso vive en app/ (reglas §A.16.2).
import { useState } from 'react'
import { useParams } from 'react-router'
import { PackageContent } from '../features/bundle/PackageContent.tsx'
import { PackageForm } from '../features/bundle/PackageForm.tsx'
import { usePackage, useTransitionPackage } from '../features/bundle/api.ts'
import { DocumentsTab } from '../features/documents/DocumentsTab.tsx'
import { useEntityDocuments } from '../features/documents/api.ts'
import { ProductionTab } from '../features/production/ProductionTab.tsx'
import { RegressionTab } from '../features/regression/RegressionTab.tsx'
import { ReportsTab } from '../features/report/ReportsTab.tsx'
import { useReports } from '../features/report/api.ts'
import { useRegression } from '../features/regression/api.ts'
import { MissingList, type RuleTabs } from '../features/requirements/MissingList.tsx'
import { Lifecycle, type Stage } from '../features/requirements/RequirementHeader.tsx'
import { useCan, useWorkflow } from '../shared/api/session.ts'
import type { Package, Workflow } from '../shared/api/types.ts'
import { formatDate } from '../shared/lib/format.ts'
import { ROUTE_LABELS } from '../shared/lib/labels.ts'
import { StateChip } from '../shared/ui/chips.tsx'
import { Async, PageSkeleton } from '../shared/ui/feedback.tsx'
import { Icon } from '../shared/ui/Icon.tsx'
import { Crumbs, MetaItem } from '../shared/ui/layout.tsx'
import { Dialog } from '../shared/ui/Modal.tsx'
import { Tabs } from '../shared/ui/Tabs.tsx'
import { Transitions } from '../shared/ui/Transitions.tsx'
import { Activity } from './Activity.tsx'

// A qué pestaña de esta ficha lleva cada regla de bloqueo del PKG.
const RULE_TABS: RuleTabs = {
  package_has_content: { tab: 'contenido', label: 'Ver contenido' },
  regression_confirmed: { tab: 'regresion', label: 'Ver regresión' },
  regression_passed: { tab: 'regresion', label: 'Ver regresión' },
  no_open_defect_with_severity: { tab: 'contenido', label: 'Ver los REQ' },
  checklist_signed: { tab: 'produccion', label: 'Ver checklist' },
  diagnostic_reviewed: { tab: 'produccion', label: 'Ver diagnósticos' },
  smoke_passed: { tab: 'produccion', label: 'Ver pruebas de humo' },
}

/** La línea de estados sale del flujo configurado: el PKG no trae tiempos por etapa. */
function stagesOf(workflow: Workflow | undefined, bundle: Package): Stage[] {
  const line = (workflow?.states ?? []).flatMap((state) => (state.position === undefined ? [] : [{ ...state, position: state.position }])).sort((a, b) => a.position - b.position)
  const current = line.find((state) => state.key === bundle.state.key)
  if (!current) return []
  const blocked = bundle.missing.filter((step) => step.conditions.some((condition) => condition.blocks)).map((step) => step.to.key)
  return line.map((state) => ({
    state: { key: state.key, name: state.name },
    status: state.position < current.position ? 'done' : state.key === current.key ? 'current' : blocked.includes(state.key) ? 'blocked' : 'pending',
  }))
}

function PackageView({ bundle }: { bundle: Package }) {
  const code = bundle.code
  const can = useCan()
  const workflow = useWorkflow('package').data
  const transition = useTransitionPackage(code)
  const documents = useEntityDocuments('package', code)
  const regression = useRegression(code)
  const reports = useReports({ package: code, size: 50 })
  const [editing, setEditing] = useState(false)

  const stages = stagesOf(workflow, bundle)
  const blocked = bundle.missing.filter((step) => step.conditions.some((condition) => condition.blocks)).map((step) => step.to.key)
  // El contenido solo cambia en el estado inicial del flujo («En armado»).
  const initial = workflow?.states.find((state) => state.isInitial)?.key ?? 'in_assembly'
  const editable = bundle.state.key === initial && can('pkg.build')
  const summary = regression.data?.summary

  return (
    <>
      <section className="glass pad stack" aria-labelledby="pkg-title">
        <div className="row row--between">
          <div className="stack-s">
            <span className="id">{bundle.code}</span>
            <h1 className="title" id="pkg-title">
              {bundle.title}
            </h1>
          </div>
          <div className="stack-s">
            <span className="h3">Estado actual</span>
            <StateChip state={bundle.state} current large />
          </div>
        </div>
        <dl className="meta">
          <MetaItem label="Sistema">{bundle.system.name}</MetaItem>
          {bundle.route && <MetaItem label="Ruta">{ROUTE_LABELS[bundle.route]}</MetaItem>}
          <MetaItem label="Contenido">
            <span className="num">
              {bundle.builds.length} BUILD · {bundle.requirements.length} REQ
            </span>
          </MetaItem>
          <MetaItem label="Creado por">
            {bundle.createdBy.displayName} · {formatDate(bundle.createdAt)}
          </MetaItem>
        </dl>
        {bundle.description && <p className="wrap">{bundle.description}</p>}
        {stages.length > 0 && <Lifecycle stages={stages} />}
        <div className="row row--between">
          <Transitions workflow={workflow} stateKey={bundle.state.key} can={can} blocked={blocked} onTransition={transition.mutateAsync} />
          {can('pkg.build') && (
            <button className="btn" type="button" onClick={() => setEditing(true)}>
              <Icon name="pencil" />
              Editar
            </button>
          )}
        </div>
      </section>
      <MissingList missing={bundle.missing} ruleTabs={RULE_TABS} />
      <Tabs
        label="Secciones del paquete"
        tabs={[
          {
            id: 'contenido',
            label: 'Contenido',
            count: bundle.builds.length + bundle.requirements.length,
            render: () => <PackageContent bundle={bundle} editable={editable} />,
          },
          {
            id: 'regresion',
            label: 'Regresión',
            count: summary ? `${summary.passed}/${summary.included}` : undefined,
            render: () => <RegressionTab packageCode={code} canManage={can('regression.manage')} />,
          },
          {
            id: 'produccion',
            label: 'Producción',
            render: () => <ProductionTab packageCode={code} systemKey={bundle.system.key} route={bundle.route} />,
          },
          {
            id: 'informes',
            label: 'Informes',
            count: reports.data?.total,
            render: () => <ReportsTab packageCode={code} />,
          },
          {
            id: 'documentacion',
            label: 'Documentación',
            count: documents.data?.items.length,
            render: () => <DocumentsTab entityType="package" entityCode={code} canUpload={can('doc.upload')} />,
          },
          { id: 'actividad', label: 'Actividad', render: () => <Activity entityType="package" entityCode={code} /> },
        ]}
      />
      {editing && (
        <Dialog title={`Editar ${code}`} onClose={() => setEditing(false)}>
          <PackageForm bundle={bundle} onClose={() => setEditing(false)} />
        </Dialog>
      )}
    </>
  )
}

export default function PackagePage() {
  const { code = '' } = useParams()
  const bundle = usePackage(code)
  return (
    <>
      <Crumbs items={[{ label: 'Paquetes', to: '/paquetes' }, { label: code, code: true }]} />
      <Async query={bundle} skeleton={<PageSkeleton />}>
        {(data) => <PackageView bundle={data} />}
      </Async>
    </>
  )
}
