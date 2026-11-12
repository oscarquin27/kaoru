// Lista de casos de prueba (TC) de todos los requerimientos.
import { useSystems } from '../../shared/api/session.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Chip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterInput, FilterSelect, Pager } from '../../shared/ui/Filters.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { useTestCases } from './api.ts'

const YES_NO = [
  { value: 'true', label: 'Sí' },
  { value: 'false', label: 'No' },
]

export default function TestCasesPage() {
  const { params, pick } = useUrlFilters()
  const systems = useSystems().data?.items ?? []
  const page = Number(params.get('page')) || 1
  const testCases = useTestCases({ ...pick(['system', 'critical', 'requirement', 'q']), page: page > 1 ? page : undefined })

  return (
    <>
      <Crumbs items={[{ label: 'Pruebas', to: '/pruebas' }, { label: 'Casos de prueba' }]} />
      <PageHead title="Casos de prueba" subtitle="Los TC se definen en la ficha de cada requerimiento, sobre sus casos de uso." />
      <FilterBar views={{ screen: 'test-cases', names: ['system', 'critical', 'requirement', 'q'] }}>
        <FilterSelect name="system" label="Sistema" options={systems.map((system) => ({ value: system.key, label: system.name }))} />
        <FilterSelect name="critical" label="Crítico" options={YES_NO} />
        <FilterInput name="requirement" label="REQ" placeholder="REQ-2026-0001" />
        <FilterInput name="q" label="Texto en el título" />
      </FilterBar>
      <Async query={testCases}>
        {(data) => (
          <>
            {data.items.length === 0 ? (
              <Empty>No hay casos de prueba con este filtro.</Empty>
            ) : (
              <div className="surface table-wrap">
                <table className="table">
                  <thead>
                    <tr>
                      <th className="col-fixed">TC</th>
                      <th>Caso de prueba</th>
                      <th>Sistema</th>
                      <th>Crítico</th>
                      <th>Casos de uso</th>
                      <th>Vigencia</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.items.map((testCase) => (
                      <tr key={testCase.code}>
                        <td className="col-fixed">
                          <Code code={testCase.code} />
                        </td>
                        <td>{testCase.title}</td>
                        <td>{testCase.system.name}</td>
                        <td>{testCase.isCritical ? 'Sí' : 'No'}</td>
                        <td>
                          <span className="cell-line">
                            {testCase.functionalCases.map((code) => (
                              <Code key={code} code={code} />
                            ))}
                          </span>
                        </td>
                        <td>
                          {testCase.retired ? (
                            <Chip tone="neutro" icon="minus">
                              Retirado
                            </Chip>
                          ) : (
                            <Chip tone="ok" icon="check">
                              Activo
                            </Chip>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            <Pager page={data.page} size={data.size} total={data.total} noun="casos de prueba" />
          </>
        )}
      </Async>
    </>
  )
}
