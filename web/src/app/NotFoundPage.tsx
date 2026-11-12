import { Link } from 'react-router'
import { Crumbs, PageHead } from '../shared/ui/layout.tsx'

export function NotFoundPage() {
  return (
    <>
      <Crumbs items={[{ label: 'No encontrado' }]} />
      <PageHead title="Esta página no existe" subtitle="Revisa la dirección o busca el ID con Ctrl K." />
      <p>
        <Link className="btn" to="/">
          Ir a inicio
        </Link>
      </p>
    </>
  )
}
