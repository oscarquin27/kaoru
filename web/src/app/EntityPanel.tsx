// Panel lateral: abre la ficha de cualquier ID sin perder la posición. Se abre con `?ficha=ID`,
// así que el enlace se puede compartir. Junta lecturas de varios features, por eso vive en app/.
import type { ReactNode } from 'react'
import { Link, useSearchParams } from 'react-router'
import { testCaseImpactPath, useImpact } from '../shared/api/catalog.ts'
import { useApiQuery } from '../shared/api/query.ts'
import type { Change, Defect, Document, Exploration, FunctionalCase, RequirementOverview, Run, TestCase } from '../shared/api/types.ts'
import { prefixOf, routeOf } from '../shared/lib/codes.ts'
import { formatDate, formatDateTime, formatDaysLong, shortChecksum } from '../shared/lib/format.ts'
import { DOCUMENT_KIND_LABELS, ROUTE_LABELS } from '../shared/lib/labels.ts'
import { PANEL_PARAM } from '../shared/lib/urlFilters.ts'
import { ComponentChips, ResultChip, RiskChip, SeverityChip, StateChip } from '../shared/ui/chips.tsx'
import { Code, TextWithCodes } from '../shared/ui/Code.tsx'
import { Async } from '../shared/ui/feedback.tsx'
import { MetaItem } from '../shared/ui/layout.tsx'
import { Drawer } from '../shared/ui/Modal.tsx'
import { BuildLink, EnvironmentLink } from '../shared/ui/RunTarget.tsx'

type Fields = [label: string, value: ReactNode][]

function Codes({ codes }: { codes: string[] }) {
  if (codes.length === 0) return <span className="muted">Ninguno</span>
  return codes.map((code) => <Code key={code} code={code} />)
}

function TestCaseComponents({ code }: { code: string }) {
  const impact = useImpact(testCaseImpactPath(code))
  return <Async query={impact}>{(data) => <ComponentChips components={data.items} empty="Sin componentes declarados" />}</Async>
}

/** Lee la entidad y la convierte en pares etiqueta-valor. */
function Sheet<T>({ path, fields }: { path: string; fields: (data: T) => Fields }) {
  const query = useApiQuery<T>(path)
  return (
    <Async query={query}>
      {(data) => (
        <dl className="meta stack">
          {fields(data).map(([label, value]) => (
            <MetaItem key={label} label={label}>
              {value}
            </MetaItem>
          ))}
        </dl>
      )}
    </Async>
  )
}

function requirementFields({ requirement, missing }: RequirementOverview): Fields {
  const blocking = missing.flatMap((step) => step.conditions.filter((condition) => condition.blocks))
  return [
    ['Título', requirement.title],
    ['Estado', <StateChip state={requirement.state} />],
    ['Sistema', requirement.system.name],
    ['Riesgo', requirement.risk ? <RiskChip level={requirement.risk.level} /> : <span className="muted">Sin evaluar</span>],
    ['Responsable', requirement.owner?.displayName ?? <span className="muted">Sin asignar</span>],
    [
      'Qué falta',
      blocking.length === 0 ? (
        'Nada bloquea el siguiente paso.'
      ) : (
        <ul className="stack-s">
          {blocking.map((condition) => (
            <li key={condition.ruleKey}>
              <TextWithCodes text={condition.message} />
            </li>
          ))}
        </ul>
      ),
    ],
  ]
}

const runFields = (run: Run): Fields => [
  ['Tipo', 'Ejecución oficial'],
  ['TC', <><Code code={run.testCase.code} /> {run.testCase.title}</>],
  ['BUILD', <><BuildLink target={run} /> <span className="mono muted" title={run.buildChecksum}>{shortChecksum(run.buildChecksum)}</span></>],
  ['Ambiente', <EnvironmentLink target={run} />],
  ['Resultado', <ResultChip result={run.result} />],
  ['Ejecutó', `${run.executedBy.displayName} · ${formatDateTime(run.executedAt)}`],
  ['Notas', run.notes || <span className="muted">Sin notas</span>],
]

const defectFields = (defect: Defect): Fields => [
  ['Defecto', defect.title],
  ['Severidad', <SeverityChip severity={defect.severity} />],
  ['Estado', <StateChip state={defect.state} />],
  ['En estado', formatDaysLong(defect.daysInState)],
  ['RUN', <Code code={defect.run} />],
  ['Requerimiento', <Code code={defect.requirement} />],
]

const testCaseFields = (testCase: TestCase): Fields => [
  ['Caso de prueba', testCase.title],
  ['Sistema', testCase.system.name],
  ['Crítico', testCase.isCritical ? 'Sí' : 'No'],
  ['Precondiciones', <span className="wrap">{testCase.preconditions || 'Ninguna'}</span>],
  ['Pasos', <span className="wrap">{testCase.steps}</span>],
  ['Resultado esperado', <span className="wrap">{testCase.expectedResult}</span>],
  ['Casos de uso', <Codes codes={testCase.functionalCases} />],
  ['Componentes que ejercita', <TestCaseComponents code={testCase.code} />],
]

const functionalCaseFields = (useCase: FunctionalCase): Fields => [
  ['Caso de uso', useCase.title],
  ['Descripción', useCase.description || <span className="muted">Sin descripción</span>],
  ['Requerimiento', <Code code={useCase.requirement} />],
  ['Casos de prueba', <Codes codes={useCase.testCases} />],
]

const explorationFields = (exploration: Exploration): Fields => [
  ['Tipo', 'Sesión exploratoria'],
  ['Requerimiento', <Code code={exploration.requirement} />],
  ['BUILD', <BuildLink target={exploration} />],
  ['Ambiente', <EnvironmentLink target={exploration} />],
  ['Cambios manuales', <span className="wrap">{exploration.manualChanges || 'Ninguno'}</span>],
  ['Hallazgos', <span className="wrap">{exploration.findings || 'Ninguno'}</span>],
  ['Ejecutó', `${exploration.executedBy.displayName} · ${formatDateTime(exploration.executedAt)}`],
]

const changeFields = (change: Change): Fields => [
  ['Cambio', change.title],
  ['Sistema', change.system.name],
  ['Ruta', ROUTE_LABELS[change.route]],
  ['Estado', <StateChip state={change.state} />],
  ['Ventana', formatDate(change.windowStart)],
]

function documentFields(document: Document): Fields {
  const latest = document.versions[0]
  return [
    ['Documento', document.title],
    ['Tipo', DOCUMENT_KIND_LABELS[document.kind]],
    ['Versión', latest ? `v${latest.version} · ${latest.uploadedBy.displayName} · ${formatDate(latest.createdAt)}` : 'Sin versiones'],
    ['SHA-256', latest ? <span className="mono" title={latest.checksum}>{shortChecksum(latest.checksum)}</span> : '—'],
    ['Enlazado a', <Codes codes={document.linkedTo.map((entity) => entity.code)} />],
  ]
}

function PanelBody({ code }: { code: string }) {
  switch (prefixOf(code)) {
    case 'REQ':
      return <Sheet path={`/api/requirements/${code}/overview`} fields={requirementFields} />
    case 'RUN':
      return <Sheet path={`/api/runs/${code}`} fields={runFields} />
    case 'DEF':
      return <Sheet path={`/api/defects/${code}`} fields={defectFields} />
    case 'TC':
      return <Sheet path={`/api/test-cases/${code}`} fields={testCaseFields} />
    case 'UC':
      return <Sheet path={`/api/functional-cases/${code}`} fields={functionalCaseFields} />
    case 'EXP':
      return <Sheet path={`/api/explorations/${code}`} fields={explorationFields} />
    case 'CHG':
      return <Sheet path={`/api/changes/${code}`} fields={changeFields} />
    case 'DOC':
      return <Sheet path={`/api/documents/${code}`} fields={documentFields} />
    default:
      return <p className="soft">Este ID no tiene ficha propia. Si es un riesgo, se consulta en la ficha de su requerimiento.</p>
  }
}

export function EntityPanel() {
  const [params, setParams] = useSearchParams()
  const code = params.get(PANEL_PARAM)
  if (!code) return null

  function close() {
    setParams(
      (previous) => {
        const next = new URLSearchParams(previous)
        next.delete(PANEL_PARAM)
        return next
      },
      { preventScrollReset: true },
    )
  }
  const route = routeOf(code)
  return (
    <Drawer title={code} onClose={close}>
      <PanelBody code={code} />
      {route && (
        <Link className="btn btn--primary" to={route}>
          Abrir ficha completa
        </Link>
      )}
    </Drawer>
  )
}
