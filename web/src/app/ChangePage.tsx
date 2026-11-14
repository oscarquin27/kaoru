// Ficha de un CHG con su sección de certificación. La ficha es de cambios y la certificación de
// planificación; como un feature no importa a otro, se juntan aquí.
import ChangeView from '../features/changes/ChangePage.tsx'
import { ChangeCertification } from '../features/planning/ChangeCertification.tsx'

export default function ChangePage() {
  return <ChangeView planning={(change) => <ChangeCertification code={change.code} />} />
}
