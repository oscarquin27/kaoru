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
      { path: '/cambios/:code', page: lazy(() => import('./ChangePage.tsx')) },
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
      { path: '/pruebas/componentes', label: 'Componentes', page: lazy(() => import('../features/components/ComponentsPage.tsx')) },
      { path: '/pruebas/componentes/:code', page: lazy(() => import('../features/components/ComponentPage.tsx')) },
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
  {
    id: 'paquetes',
    label: 'Paquetes',
    icon: 'box',
    screens: [
      { path: '/paquetes', label: 'Paquetes', page: lazy(() => import('../features/bundle/PackagesPage.tsx')) },
      { path: '/paquetes/builds', label: 'BUILD', page: lazy(() => import('../features/builds/BuildsPage.tsx')) },
      { path: '/paquetes/builds/:code', page: lazy(() => import('../features/builds/BuildPage.tsx')) },
      { path: '/paquetes/:code', page: lazy(() => import('./PackagePage.tsx')) },
    ],
  },
  {
    id: 'ambientes',
    label: 'Ambientes',
    icon: 'server',
    screens: [
      { path: '/ambientes', label: 'Ambientes', page: lazy(() => import('../features/environments/EnvironmentsPage.tsx')) },
      { path: '/ambientes/despliegues', label: 'Despliegues', page: lazy(() => import('../features/environments/DeploymentsPage.tsx')) },
      { path: '/ambientes/despliegues/:code', page: lazy(() => import('../features/environments/DeploymentPage.tsx')) },
      { path: '/ambientes/:code', page: lazy(() => import('../features/environments/EnvironmentPage.tsx')) },
    ],
  },
  {
    id: 'produccion',
    label: 'Producción',
    icon: 'flag',
    screens: [
      { path: '/produccion', label: 'Checklists', page: lazy(() => import('../features/production/ChecklistsPage.tsx')) },
      { path: '/produccion/diagnosticos', label: 'Diagnósticos', page: lazy(() => import('../features/production/DiagnosticsPage.tsx')) },
      { path: '/produccion/humo', label: 'Pruebas de humo', page: lazy(() => import('../features/production/SmokeTestsPage.tsx')) },
      { path: '/produccion/plantillas', label: 'Plantillas', page: lazy(() => import('../features/production/TemplatesPage.tsx')) },
      { path: '/produccion/diagnosticos/:code', page: lazy(() => import('./DiagnosticPage.tsx')) },
      { path: '/produccion/humo/:code', page: lazy(() => import('./SmokeTestPage.tsx')) },
      { path: '/produccion/:code', page: lazy(() => import('./ChecklistPage.tsx')) },
    ],
  },
  {
    id: 'informes',
    label: 'Informes',
    icon: 'chart',
    screens: [
      { path: '/informes', label: 'Informes de conformidad', page: lazy(() => import('../features/report/ReportsPage.tsx')) },
      { path: '/informes/:code', page: lazy(() => import('./ReportPage.tsx')) },
    ],
  },
  {
    id: 'metricas',
    label: 'Métricas',
    icon: 'gauge',
    screens: [
      { path: '/metricas', label: 'Cobertura', page: lazy(() => import('../features/metrics/CoveragePage.tsx')) },
      { path: '/metricas/defectos', label: 'Tiempo en estado', page: lazy(() => import('../features/metrics/DefectStatesPage.tsx')) },
      { path: '/metricas/regresion', label: 'Regresión', page: lazy(() => import('../features/metrics/RegressionPage.tsx')) },
      { path: '/metricas/escapados', label: 'Defectos escapados', page: lazy(() => import('../features/metrics/EscapedDefectsPage.tsx')) },
    ],
  },
  {
    id: 'planificacion',
    label: 'Planificación',
    icon: 'calendar',
    screens: [
      { path: '/planificacion', label: 'Por operador', page: lazy(() => import('../features/planning/SchedulePage.tsx')) },
      { path: '/planificacion/actividades', label: 'Actividades', page: lazy(() => import('../features/planning/ActivitiesPage.tsx')) },
      // El calendario de hitos y ventanas conserva su ruta de la fase 4.
      { path: '/calendario', label: 'Calendario', page: lazy(() => import('../features/planning/CalendarPage.tsx')) },
      { path: '/planificacion/actividades/:code', page: lazy(() => import('./ActivityPage.tsx')) },
    ],
  },
  {
    id: 'administracion',
    label: 'Administración',
    icon: 'sliders',
    screens: [
      { path: '/administracion', label: 'Permisos por rol', page: lazy(() => import('../features/admin/PermissionsPage.tsx')) },
      { path: '/administracion/flujos', label: 'Flujos', page: lazy(() => import('../features/admin/WorkflowsPage.tsx')) },
      { path: '/administracion/reglas', label: 'Reglas de bloqueo', page: lazy(() => import('../features/admin/BlockingRulesPage.tsx')) },
      { path: '/administracion/sistemas', label: 'Sistemas', page: lazy(() => import('../features/admin/SystemsPage.tsx')) },
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
