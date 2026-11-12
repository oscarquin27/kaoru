// Nombres en español de los valores fijos del contrato (enums de api/openapi.yaml).
import type { DocumentKind, EntityType, LinkKind, QualityCharacteristic, RiskLevel, Role, Route, RunResult, SearchHit, Severity } from '../api/types.ts'

export const ROUTE_LABELS: Record<Route, string> = { high: 'Alta', light: 'Ligera', minor_fix: 'Ligera · fix menor' }
export const SEVERITY_LABELS: Record<Severity, string> = { high: 'Alta', medium: 'Media', low: 'Baja' }
export const RISK_LABELS: Record<RiskLevel, string> = { high: 'Alto', medium: 'Medio', low: 'Bajo' }
export const RESULT_LABELS: Record<RunResult, string> = { passed: 'Aprobado', failed: 'Fallido', blocked: 'Bloqueado' }
export const LINK_KIND_LABELS: Record<LinkKind, string> = {
  google_sheets: 'Google Sheets',
  youtrack: 'YouTrack',
  shared_folder: 'Carpeta compartida',
  wiki: 'Wiki',
  other: 'Otro',
}
export const DOCUMENT_KIND_LABELS: Record<DocumentKind, string> = {
  specification: 'Especificación',
  manual: 'Manual',
  evidence: 'Evidencia',
  minutes: 'Minuta',
  snapshot: 'Instantánea',
  other: 'Otro',
}
export const QUALITY_LABELS: Record<QualityCharacteristic, string> = {
  functional_suitability: 'Adecuación funcional',
  performance_efficiency: 'Eficiencia de desempeño',
  compatibility: 'Compatibilidad',
  usability: 'Usabilidad',
  reliability: 'Fiabilidad',
  security: 'Seguridad',
  maintainability: 'Mantenibilidad',
  portability: 'Portabilidad',
}
export const ROLE_LABELS: Record<Role, string> = {
  'change-manager': 'Change manager',
  'qa-admin': 'QA admin',
  qa: 'QA',
  dev: 'Desarrollo',
  ops: 'Operaciones',
  auditor: 'Auditoría',
}
export const ENTITY_LABELS: Record<EntityType | SearchHit['type'], string> = {
  requirement: 'Requerimiento',
  change: 'Cambio',
  risk_assessment: 'Riesgo',
  functional_case: 'Caso de uso',
  test_case: 'Caso de prueba',
  test_run: 'Ejecución',
  exploratory_session: 'Exploración',
  defect: 'Defecto',
  document: 'Documento',
}

/** Etiqueta de un valor que puede no estar en la tabla (la API manda texto libre). */
export function labelOf(labels: Record<string, string>, key: string): string {
  return labels[key] ?? key
}

export function options(labels: Record<string, string>): { value: string; label: string }[] {
  return Object.entries(labels).map(([value, label]) => ({ value, label }))
}
