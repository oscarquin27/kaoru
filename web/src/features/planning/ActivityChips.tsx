// Clase y estado de una actividad: color, ícono y texto, como los demás chips.
import type { ActivityKind, ActivityStatus } from '../../shared/api/types.ts'
import { ACTIVITY_KIND_LABELS, ACTIVITY_STATUS_LABELS } from '../../shared/lib/labels.ts'
import { Chip, type Tone } from '../../shared/ui/chips.tsx'
import type { IconName } from '../../shared/ui/Icon.tsx'

export const KIND_LOOK: Record<ActivityKind, { tone: Tone; icon: IconName }> = {
  certification: { tone: 'actual', icon: 'target' },
  regression: { tone: 'curso', icon: 'flask' },
  production: { tone: 'ok', icon: 'server' },
  other: { tone: 'neutro', icon: 'circle' },
}

const STATUS_LOOK: Record<ActivityStatus, { tone: Tone; icon: IconName }> = {
  planned: { tone: 'neutro', icon: 'calendar' },
  active: { tone: 'curso', icon: 'clock' },
  done: { tone: 'ok', icon: 'check' },
  cancelled: { tone: 'bloq', icon: 'x' },
}

export function KindChip({ kind }: { kind: ActivityKind }) {
  return <Chip {...KIND_LOOK[kind]}>{ACTIVITY_KIND_LABELS[kind]}</Chip>
}

export function ActivityStatusChip({ status }: { status: ActivityStatus }) {
  return <Chip {...STATUS_LOOK[status]}>{ACTIVITY_STATUS_LABELS[status]}</Chip>
}
