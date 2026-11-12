// Ficha del DEF: la pieza del feature de defectos más la actividad (auditoría y comentarios).
import { useParams } from 'react-router'
import { DefectDetail } from '../features/defects/DefectDetail.tsx'
import { Activity } from './Activity.tsx'

export default function DefectPage() {
  const { code = '' } = useParams()
  return <DefectDetail code={code} activity={<Activity entityType="defect" entityCode={code} />} />
}
