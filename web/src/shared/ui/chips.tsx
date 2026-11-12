// Chip de estado: color + ícono + texto, siempre los tres (boceto, «Estados»).
import type { ReactNode } from 'react'
import type { RiskLevel, RunResult, Severity, StateRef } from '../api/types.ts'
import { RESULT_LABELS, RISK_LABELS, SEVERITY_LABELS } from '../lib/labels.ts'
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
