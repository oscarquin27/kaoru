// Encabezado de la ficha del REQ y su línea de vida (boceto: requerimiento.html).
import type { ReactNode } from 'react'
import type { RequirementOverview } from '../../shared/api/types.ts'
import { formatDays, formatDaysLong } from '../../shared/lib/format.ts'
import { ROUTE_LABELS } from '../../shared/lib/labels.ts'
import { RiskChip, StateChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Icon, type IconName } from '../../shared/ui/Icon.tsx'
import { MetaItem } from '../../shared/ui/layout.tsx'

type Stage = RequirementOverview['lifecycle'][number]

const STAGE_ICONS: Record<Stage['status'], IconName> = { done: 'check', current: 'target', blocked: 'lock', pending: 'circle' }

function stageTime(stage: Stage): string {
  switch (stage.status) {
    case 'done':
      return formatDays(stage.days)
    case 'current':
      return `${formatDays(stage.days)} · actual`
    case 'blocked':
      return 'bloqueada'
    case 'pending':
      return 'pendiente'
  }
}

export function Lifecycle({ stages }: { stages: Stage[] }) {
  return (
    <ol className="life" aria-label="Línea de vida">
      {stages.map((stage) => (
        <li key={stage.state.key} className="life__step" data-state={stage.status} aria-current={stage.status === 'current' ? 'step' : undefined}>
          <span className="life__dot">
            <Icon name={STAGE_ICONS[stage.status]} />
          </span>
          <span className="life__name">{stage.state.name}</span>
          <span className="life__time">{stageTime(stage)}</span>
          {stage.status === 'done' && <span className="sr-only">completada</span>}
        </li>
      ))}
    </ol>
  )
}

export function RequirementHeader({ overview, actions }: { overview: RequirementOverview; actions?: ReactNode }) {
  const { requirement, lifecycle } = overview
  const current = lifecycle.find((stage) => stage.status === 'current')
  return (
    <section className="glass pad stack" aria-labelledby="req-title">
      <div className="row row--between">
        <div className="stack-s">
          <span className="id">{requirement.code}</span>
          <h1 className="title" id="req-title">
            {requirement.title}
          </h1>
        </div>
        <div className="stack-s">
          <span className="h3">Estado actual</span>
          <StateChip state={requirement.state} current large />
        </div>
      </div>
      <dl className="meta">
        <MetaItem label="Sistema">{requirement.system.name}</MetaItem>
        <MetaItem label="Cambio">
          {requirement.change ? <Code code={requirement.change.code} /> : <span className="muted">Sin cambio</span>} · ruta {ROUTE_LABELS[requirement.route].toLowerCase()}
        </MetaItem>
        <MetaItem label="Riesgo">
          {requirement.risk ? (
            <>
              <RiskChip level={requirement.risk.level} /> <span className="id">{requirement.risk.code}</span>
            </>
          ) : (
            <span className="muted">Sin evaluar</span>
          )}
        </MetaItem>
        <MetaItem label="Responsable actual">{requirement.owner?.displayName ?? <span className="muted">Sin asignar</span>}</MetaItem>
        {current && (
          <MetaItem label="En este estado">
            <span className="num">{formatDaysLong(current.days)}</span>
          </MetaItem>
        )}
      </dl>
      <Lifecycle stages={lifecycle} />
      {actions}
    </section>
  )
}
