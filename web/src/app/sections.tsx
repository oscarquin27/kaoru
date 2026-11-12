// ÚNICA lista de secciones (reglas §A.16.2): de aquí salen el menú, las rutas, la navegación
// secundaria y el título de cada pantalla. Cada pantalla se carga aparte con React.lazy.
import { lazy, type ComponentType, type LazyExoticComponent } from 'react'
import type { IconName } from '../shared/ui/Icon.tsx'

export interface Screen {
  path: string
  /** Con etiqueta, la pantalla aparece en la navegación del área. Sin ella es una ficha. */
  label?: string
  page: LazyExoticComponent<ComponentType>
}

export interface Section {
  id: string
  label: string
  icon: IconName
  /** Sin pantallas: el área es de una fase posterior y aparece deshabilitada. */
  screens: Screen[]
}

export const SECTIONS: Section[] = [
  { id: 'inicio', label: 'Inicio', icon: 'house', screens: [{ path: '/', label: 'Inicio', page: lazy(() => import('./HomePage.tsx')) }] },
  {
    id: 'cambios',
    label: 'Cambios',
    icon: 'swap',
    screens: [
      { path: '/cambios', label: 'Cambios', page: lazy(() => import('../features/changes/ChangesPage.tsx')) },
      { path: '/cambios/:code', page: lazy(() => import('../features/changes/ChangePage.tsx')) },
    ],
  },
  {
    id: 'requerimientos',
    label: 'Requerimientos',
    icon: 'file',
    screens: [
      { path: '/requerimientos', label: 'Requerimientos', page: lazy(() => import('../features/requirements/RequirementsPage.tsx')) },
      { path: '/requerimientos/:code', page: lazy(() => import('./RequirementPage.tsx')) },
      { path: '/documentos/:code', page: lazy(() => import('../features/documents/DocumentPage.tsx')) },
    ],
  },
  {
    id: 'pruebas',
    label: 'Pruebas',
    icon: 'flask',
    screens: [
      { path: '/pruebas', label: 'Matriz de trazabilidad', page: lazy(() => import('../features/traceability/MatrixPage.tsx')) },
      { path: '/pruebas/casos', label: 'Casos de prueba', page: lazy(() => import('../features/testdesign/TestCasesPage.tsx')) },
    ],
  },
  {
    id: 'defectos',
    label: 'Defectos',
    icon: 'bug',
    screens: [
      { path: '/defectos', label: 'Defectos', page: lazy(() => import('../features/defects/DefectsPage.tsx')) },
      { path: '/defectos/:code', page: lazy(() => import('./DefectPage.tsx')) },
    ],
  },
  { id: 'paquetes', label: 'Paquetes', icon: 'box', screens: [] },
  { id: 'ambientes', label: 'Ambientes', icon: 'server', screens: [] },
  { id: 'produccion', label: 'Producción', icon: 'flag', screens: [] },
  { id: 'informes', label: 'Informes', icon: 'chart', screens: [] },
  { id: 'calendario', label: 'Calendario', icon: 'calendar', screens: [] },
  {
    id: 'administracion',
    label: 'Administración',
    icon: 'sliders',
    screens: [
      { path: '/administracion', label: 'Permisos por rol', page: lazy(() => import('../features/admin/PermissionsPage.tsx')) },
      { path: '/administracion/flujos', label: 'Flujos', page: lazy(() => import('../features/admin/WorkflowsPage.tsx')) },
      { path: '/administracion/reglas', label: 'Reglas de bloqueo', page: lazy(() => import('../features/admin/BlockingRulesPage.tsx')) },
      { path: '/administracion/auditoria', label: 'Auditoría', page: lazy(() => import('../features/audit/AuditPage.tsx')) },
    ],
  },
]

/** Va al pie de la barra lateral, como en el boceto. */
export const ABOUT: Section = {
  id: 'acerca',
  label: 'Acerca de',
  icon: 'info',
  screens: [{ path: '/acerca', label: 'Acerca de', page: lazy(() => import('./AboutPage.tsx')) }],
}

export const ALL_SECTIONS = [...SECTIONS, ABOUT]
