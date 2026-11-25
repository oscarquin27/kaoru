// Ficha del REQ, la pantalla central (boceto: requerimiento.html). Junta piezas de varios
// features y les pasa por props lo que necesitan unas de otras (reglas §A.16.2).
import { useState } from 'react'
import { useParams } from 'react-router'
import { DefectForm } from '../features/defects/DefectForm.tsx'
import { DefectsTable } from '../features/defects/DefectsTable.tsx'
import { useDefects } from '../features/defects/api.ts'
import { DocumentsTab } from '../features/documents/DocumentsTab.tsx'
import { useRequirementDocuments } from '../features/documents/api.ts'
import { Explorations } from '../features/executions/Explorations.tsx'
import { RunForm } from '../features/executions/RunForm.tsx'
import { useRequirementRuns } from '../features/executions/api.ts'
import { EntryContract } from '../features/requirements/EntryContract.tsx'
import { LinksTab } from '../features/requirements/LinksTab.tsx'
import { MissingList } from '../features/requirements/MissingList.tsx'
import { RequirementHeader } from '../features/requirements/RequirementHeader.tsx'
import { SummaryTab } from '../features/requirements/SummaryTab.tsx'
import { useLinks, useTransitionRequirement } from '../features/requirements/api.ts'
import { RiskCard } from '../features/risk/RiskCard.tsx'
import { TestDesign } from '../features/testdesign/TestDesign.tsx'
import { useFunctionalCases } from '../features/testdesign/api.ts'
import { TraceTab } from '../features/traceability/TraceTab.tsx'
import { useRequirementOverview } from '../features/traceability/api.ts'
import { requirementImpactPath, testCaseImpactPath } from '../shared/api/catalog.ts'
import { useCan, useWorkflow } from '../shared/api/session.ts'
import type { RequirementOverview } from '../shared/api/types.ts'
import { formatDateTime } from '../shared/lib/format.ts'
import { RESULT_LABELS, RISK_LABELS } from '../shared/lib/labels.ts'
import { Async, PageSkeleton } from '../shared/ui/feedback.tsx'
import { Icon } from '../shared/ui/Icon.tsx'
import { ImpactCard, ImpactDialog } from '../shared/ui/Impact.tsx'
import { Crumbs, Meter } from '../shared/ui/layout.tsx'
import { Dialog } from '../shared/ui/Modal.tsx'
import { Tabs } from '../shared/ui/Tabs.tsx'
import { Transitions } from '../shared/ui/Transitions.tsx'
import { Activity } from './Activity.tsx'

function Coverage({ overview }: { overview: RequirementOverview }) {
  const { required, passed } = overview.coverage
  const percent = required === 0 ? 0 : Math.round((passed / required) * 100)
  const risk = overview.requirement.risk
  return (
    <div className="surface pad row">
      <span className="h2 num">
        {passed} de {required}
      </span>
      <span className="soft">TC requeridos con RUN aprobado{risk ? ` (riesgo ${RISK_LABELS[risk.level].toLowerCase()})` : ''}</span>
      <Meter percent={percent} grow />
    </div>
  )
}

function RequirementView({ overview }: { overview: RequirementOverview }) {
  const { requirement } = overview
  const code = requirement.code
  const can = useCan()
  const workflow = useWorkflow('requirement').data
  const transition = useTransitionRequirement(code)
  const links = useLinks(code)
  const documents = useRequirementDocuments(code)
  const functionalCases = useFunctionalCases(code)
  const defects = useDefects({ requirement: code })
  const runs = useRequirementRuns(code)
  const [runFor, setRunFor] = useState<string | null>(null)
  const [impactFor, setImpactFor] = useState<string | null>(null)
  const systemKey = requirement.system.key
  const [openingDefect, setOpeningDefect] = useState(false)

  const blocked = overview.missing.filter((step) => step.conditions.some((condition) => condition.blocks)).map((step) => step.to.key)

  return (
    <>
      <RequirementHeader
        overview={overview}
        actions={<Transitions workflow={workflow} stateKey={requirement.state.key} can={can} blocked={blocked} onTransition={transition.mutateAsync} />}
      />
      <MissingList missing={overview.missing} />
      <Tabs
        label="Secciones del requerimiento"
        tabs={[
          {
            id: 'resumen',
            label: 'Resumen',
            render: () => (
              <div className="stack">
                <SummaryTab requirement={requirement} canEdit={can('req.edit')} />
                <ImpactCard
                  path={requirementImpactPath(code)}
                  systemKey={systemKey}
                  title="Componentes que toca"
                  hint="De aquí sale la regresión que se propone para el paquete que lo incluya."
                  canEdit={can('req.edit')}
                />
                <div className="grid grid--2">
                  <RiskCard code={code} assessed={requirement.risk !== undefined} canAssess={can('risk.assess')} />
                  <EntryContract code={code} canCheck={can('entry_contract.check')} />
                </div>
              </div>
            ),
          },
          {
            id: 'enlaces',
            label: 'Enlaces',
            count: links.data?.items.length,
            render: () => <LinksTab code={code} documents={documents.data?.items ?? []} canManage={can('link.manage')} />,
          },
          {
            id: 'documentacion',
            label: 'Documentación',
            count: documents.data?.items.length,
            render: () => <DocumentsTab entityType="requirement" entityCode={code} canUpload={can('doc.upload')} />,
          },
          {
            id: 'trazabilidad',
            label: 'Trazabilidad',
            render: () => <TraceTab overview={overview} functionalCases={functionalCases.data?.items ?? []} defects={defects.data?.items ?? []} />,
          },
          {
            id: 'pruebas',
            label: 'Pruebas',
            count: `${overview.coverage.passed}/${overview.coverage.required}`,
            render: () => (
              <div className="stack">
                <Coverage overview={overview} />
                <Async query={functionalCases}>
                  {(data) => (
                    <TestDesign
                      requirementCode={code}
                      systemKey={requirement.system.key}
                      functionalCases={data.items}
                      testCases={overview.testCases}
                      canManage={can('testdesign.manage')}
                      testCaseAction={
                        can('run.create') || can('testdesign.manage')
                          ? (testCase) =>
                              !testCase.retired && (
                                <span className="cell-line">
                                  {can('testdesign.manage') && (
                                    <button className="btn" type="button" aria-label={`Componentes de ${testCase.code}`} title="Componentes que ejercita" onClick={() => setImpactFor(testCase.code)}>
                                      Componentes
                                    </button>
                                  )}
                                  {can('run.create') && (
                                    <button className="btn" type="button" aria-label={`Registrar RUN de ${testCase.code}`} onClick={() => setRunFor(testCase.code)}>
                                      Registrar RUN
                                    </button>
                                  )}
                                </span>
                              )
                          : undefined
                      }
                    />
                  )}
                </Async>
                <Explorations requirementCode={code} systemKey={systemKey} canCreate={can('exp.create')} />
              </div>
            ),
          },
          {
            id: 'defectos',
            label: 'Defectos',
            count: overview.defects.length,
            render: () => (
              <div className="stack">
                {can('def.create') && (
                  <div className="row">
                    <button className="btn btn--primary" type="button" onClick={() => setOpeningDefect(true)}>
                      <Icon name="plus" />
                      Abrir DEF
                    </button>
                  </div>
                )}
                <Async query={defects}>{(data) => <DefectsTable items={data.items} />}</Async>
              </div>
            ),
          },
          { id: 'actividad', label: 'Actividad', render: () => <Activity entityType="requirement" entityCode={code} /> },
        ]}
      />

      {runFor && (
        <Dialog title={`Registrar RUN de ${runFor}`} onClose={() => setRunFor(null)}>
          <RunForm testCase={runFor} systemKey={systemKey} requirement={code} onClose={() => setRunFor(null)} />
        </Dialog>
      )}
      {impactFor && (
        <ImpactDialog path={testCaseImpactPath(impactFor)} systemKey={systemKey} title={`Componentes que ejercita ${impactFor}`} onClose={() => setImpactFor(null)} />
      )}
      {openingDefect && (
        <Dialog title={`Abrir defecto en ${code}`} onClose={() => setOpeningDefect(false)}>
          <DefectForm
            requirement={code}
            runs={(runs.data?.items ?? []).map((run) => ({
              code: run.code,
              label: `${run.code} · ${run.testCase.code} · ${RESULT_LABELS[run.result]} · ${formatDateTime(run.executedAt)}`,
            }))}
            testCases={overview.testCases
              .filter((testCase) => !testCase.retired)
              .map((testCase) => ({ code: testCase.code, label: `${testCase.code} · ${testCase.title}` }))}
            systemKey={systemKey}
            onClose={() => setOpeningDefect(false)}
          />
        </Dialog>
      )}
    </>
  )
}

export default function RequirementPage() {
  const { code = '' } = useParams()
  const overview = useRequirementOverview(code)
  const change = overview.data?.requirement.change
  return (
    <>
      <Crumbs
        items={[
          change ? { label: change.code, to: `/cambios/${change.code}`, code: true } : { label: 'Requerimientos', to: '/requerimientos' },
          { label: code, code: true },
        ]}
      />
      <Async query={overview} skeleton={<PageSkeleton />}>
        {(data) => <RequirementView overview={data} />}
      </Async>
    </>
  )
}
