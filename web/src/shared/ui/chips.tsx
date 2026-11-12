// Chip de estado: color + ícono + texto, siempre los tres (boceto, «Estados»).
import type { ReactNode } from 'react'
import type { ComponentRef, DeploymentResult, Diagnostic, RiskLevel, RunResult, Severity, SmokeResult, StateRef } from '../api/types.ts'
import { COMPONENT_KIND_LABELS, DEPLOYMENT_RESULT_LABELS, DIAGNOSTIC_RESULT_LABELS, RESULT_LABELS, RISK_LABELS, SEVERITY_LABELS, SMOKE_RESULT_LABELS } from '../lib/labels.ts'
import { Icon, type IconName } from './Icon.tsx'

export type Tone = 'ok' | 'curso' | 'aviso' | 'bloq' | 'neutro' | 'actual'

interface Look {
  tone: Tone
  icon: IconName
}

export function Chip({ tone, icon, large, children }: Look & { large?: boolean; children: ReactNode }) {
  return (
    <span className={`chip chip--${tone}${large ? ' chip--l' : ''}`}>
      <Icon name={icon} />
      {children}
    </span>
  )
}

const OK: Look = { tone: 'ok', icon: 'check' }
const IN_PROGRESS: Look = { tone: 'curso', icon: 'clock' }
const WARNING: Look = { tone: 'aviso', icon: 'triangle' }
const NEUTRAL: Look = { tone: 'neutro', icon: 'circle' }
const CURRENT: Look = { tone: 'actual', icon: 'target' }

// Cómo se ve cada estado de los flujos de REQ, CHG y DEF. Uno que no esté aquí sale neutro.
const STATE_LOOKS: Record<string, Look> = {
  in_testing: CURRENT,
  approved_qa: OK,
  approved: OK,
  released: OK,
  verified: OK,
  closed: OK,
  in_development: IN_PROGRESS,
  delivered_to_qa: IN_PROGRESS,
  in_package: IN_PROGRESS,
  in_progress: IN_PROGRESS,
  in_fix: IN_PROGRESS,
  ready_for_retest: IN_PROGRESS,
  with_defects: WARNING,
  reopened: WARNING,
  deferred: WARNING,
  rejected: { tone: 'bloq', icon: 'x' },
  // PKG
  in_assembly: IN_PROGRESS,
  ready_to_release: OK,
  approved_for_production: OK,
  deployed: OK,
}

export function StateChip({ state, current, large }: { state: StateRef; current?: boolean; large?: boolean }) {
  const look = current ? CURRENT : (STATE_LOOKS[state.key] ?? NEUTRAL)
  return (
    <Chip {...look} large={large}>
      {state.name}
    </Chip>
  )
}

const LEVEL_LOOKS: Record<Severity | RiskLevel, Look> = {
  high: { tone: 'bloq', icon: 'triangle' },
  medium: { tone: 'aviso', icon: 'minus' },
  low: NEUTRAL,
}

export function SeverityChip({ severity, count }: { severity: Severity; count?: number }) {
  const label = SEVERITY_LABELS[severity]
  return <Chip {...LEVEL_LOOKS[severity]}>{count === undefined ? label : `${count} ${label.toLowerCase()}`}</Chip>
}

export function RiskChip({ level }: { level: RiskLevel }) {
  return <Chip {...LEVEL_LOOKS[level]}>{RISK_LABELS[level]}</Chip>
}

const RESULT_LOOKS: Record<RunResult, Look> = {
  passed: OK,
  failed: { tone: 'bloq', icon: 'x' },
  blocked: { tone: 'bloq', icon: 'lock' },
}

/** Resultado del último RUN; sin RUN, «Pendiente». */
export function ResultChip({ result }: { result: RunResult | undefined }) {
  if (!result) return <Chip {...NEUTRAL}>Pendiente</Chip>
  return <Chip {...RESULT_LOOKS[result]}>{RESULT_LABELS[result]}</Chip>
}

const DEPLOYMENT_LOOKS: Record<DeploymentResult, Look> = {
  success: OK,
  failed: { tone: 'bloq', icon: 'x' },
  rolled_back: WARNING,
}

export function DeploymentResultChip({ result }: { result: DeploymentResult }) {
  return <Chip {...DEPLOYMENT_LOOKS[result]}>{DEPLOYMENT_RESULT_LABELS[result]}</Chip>
}

/** Revisión de QA de un DIAG; sin revisar, «Sin revisar». */
export function DiagnosticChip({ review }: { review: Diagnostic['review'] }) {
  if (!review) return <Chip {...IN_PROGRESS}>Sin revisar</Chip>
  return review.result === 'conforming' ? <Chip {...OK}>{DIAGNOSTIC_RESULT_LABELS.conforming}</Chip> : <Chip tone="bloq" icon="x">{DIAGNOSTIC_RESULT_LABELS.non_conforming}</Chip>
}

export function SmokeChip({ result }: { result: SmokeResult }) {
  return result === 'passed' ? <Chip {...OK}>{SMOKE_RESULT_LABELS.passed}</Chip> : <Chip tone="bloq" icon="x">{SMOKE_RESULT_LABELS.failed}</Chip>
}

/** Avance de un checklist: completo, o cuántos pasos lleva firmados. */
export function ChecklistChip({ signed, total }: { signed: number; total: number }) {
  if (signed === total) return <Chip {...OK}>Firmado</Chip>
  return (
    <Chip {...IN_PROGRESS}>
      {signed} de {total} firmados
    </Chip>
  )
}

/** Vigencia de algo que no se borra, solo se retira (COMP, ENV). */
export function RetiredChip({ retired }: { retired: boolean }) {
  return retired ? (
    <Chip tone="neutro" icon="minus">
      Retirado
    </Chip>
  ) : (
    <Chip tone="ok" icon="check">
      Activo
    </Chip>
  )
}

/** Componentes (COMP) como chips; el nombre dice qué es y la clase va en el título. */
export function ComponentChips({ components, empty = 'Ninguno' }: { components: ComponentRef[]; empty?: string }) {
  if (components.length === 0) return <span className="muted">{empty}</span>
  return (
    <span className="cell-line">
      {components.map((component) => (
        <span key={component.code} className="chip chip--neutro" title={`${component.code} · ${COMPONENT_KIND_LABELS[component.kind]}`}>
          <Icon name="box" />
          {component.name}
        </span>
      ))}
    </span>
  )
}
