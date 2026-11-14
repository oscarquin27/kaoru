// Los solapes dichos con palabras: el color de la línea de tiempo nunca es la única señal.
import type { ActivityOverlap } from '../../shared/api/types.ts'
import { TextWithCodes } from '../../shared/ui/Code.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { lengthText, spanText } from './days.ts'

export function overlapText(overlap: ActivityOverlap): string {
  return `${overlap.operator.displayName}: ${overlap.activity.code} y ${overlap.other.code} se cruzan ${spanText(overlap.from, overlap.to)} (${lengthText(overlap.from, overlap.to)}).`
}

export function OverlapList({ overlaps, label = 'Solapes' }: { overlaps: ActivityOverlap[]; label?: string }) {
  if (overlaps.length === 0) return null
  return (
    <div className="notice" role="status" aria-label={label}>
      <Icon name="triangle" />
      <div className="stack-s">
        <strong>{overlaps.length === 1 ? 'Hay 1 solape' : `Hay ${overlaps.length} solapes`}: el mismo operador en dos actividades los mismos días.</strong>
        <ul className="stack-s">
          {overlaps.map((overlap) => (
            <li key={`${overlap.operator.id}-${overlap.activity.code}-${overlap.other.code}`}>
              <TextWithCodes text={overlapText(overlap)} />
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}
