// «Acerca de»: la marca y la línea sobre Kaoru Ishikawa. Textos del boceto (acerca.html).
import { Chip } from '../shared/ui/chips.tsx'
import { Icon } from '../shared/ui/Icon.tsx'
import { Kana } from '../shared/ui/Kana.tsx'
import { Crumbs } from '../shared/ui/layout.tsx'

export default function AboutPage() {
  return (
    <>
      <Crumbs items={[{ label: 'Acerca de' }]} />
      <section className="glass pad stack">
        <div className="row">
          <Icon name="kaoru" className="icon brand__mark" size={48} />
          <span className="brand__name">
            <Kana className="brand__kana brand__kana--xl" />
            <h1 className="wordmark wordmark--xl">kaoru</h1>
          </span>
        </div>
        <p className="h2">Trazabilidad a prueba de errores</p>
        <p className="soft" style={{ maxWidth: '70ch' }}>
          Kaoru lleva el nombre de Kaoru Ishikawa, que defendió que la calidad es responsabilidad de toda la organización y no de un solo departamento.
        </p>
        <p className="soft" style={{ maxWidth: '70ch' }}>
          Aquí desarrollo, QA y change management planifican, prueban y dejan evidencia de cada cambio.
        </p>
      </section>

      <section className="surface pad stack" aria-labelledby="h-estados">
        <h2 className="h2" id="h-estados">
          Estados
        </h2>
        <p className="soft">Un estado nunca se comunica solo por color: siempre color, ícono y texto.</p>
        <div className="row">
          <Chip tone="actual" icon="target">Estado actual</Chip>
          <Chip tone="ok" icon="check">Aprobado</Chip>
          <Chip tone="curso" icon="clock">En curso</Chip>
          <Chip tone="aviso" icon="triangle">Con defectos</Chip>
          <Chip tone="bloq" icon="lock">Bloqueado</Chip>
          <Chip tone="bloq" icon="x">Fallido</Chip>
          <Chip tone="neutro" icon="circle">Pendiente</Chip>
        </div>
      </section>
    </>
  )
}
