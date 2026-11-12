// Ejemplos de la fase 3 para las pruebas de las pantallas nuevas, tipados contra el contrato.
// Las grabaciones de mocks/fixtures.json todavía no traen estas rutas.
import type { Build, Component, Deployment, Environment, Package, Regression, System, UserRef, Workflow } from './shared/api/types.ts'

export const SYPAGO: System = { key: 'SYPAGO', name: 'SyPago', defaultRoute: 'light' }
export const ANA: UserRef = { id: 'eb0a2839-b99a-41c5-884d-60a37e4a3127', displayName: 'Ana Rojas' }
export const JUAN: UserRef = { id: 'dc4e769a-8eb2-42d0-a7fa-30ecd347d566', displayName: 'Juan Díaz' }
const AT = '2026-11-12T10:00:00Z'
export const CHECKSUM = 'a3f1c2d4e5b60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90'

export const COMPONENT: Component = {
  code: 'COMP-2026-0001',
  system: SYPAGO,
  kind: 'microservice',
  name: 'motor-pagos',
  description: 'Valida y ejecuta los pagos',
  retired: false,
  createdBy: JUAN,
  createdAt: AT,
  updatedAt: AT,
}
export const COMPONENT_2: Component = { ...COMPONENT, code: 'COMP-2026-0002', kind: 'kafka_topic', name: 'pagos.eventos', description: '' }

export const BUILD: Build = {
  code: 'BUILD-2026-0001',
  system: SYPAGO,
  version: '2.14.0',
  checksum: CHECKSUM,
  requirements: [{ code: 'REQ-2026-0001', title: 'Límite diario por ordenante' }],
  migrations: '00031_limite_diario.sql',
  dependencies: 'kafka-client 3.7.0\npgx 5.6.0',
  notes: 'Compilado en CI',
  builtBy: JUAN,
  createdAt: AT,
}
export const BUILD_2: Build = { ...BUILD, code: 'BUILD-2026-0002', version: '2.14.1', checksum: CHECKSUM.replace('a3f1', 'b4e2'), requirements: [] }

export const ENVIRONMENT: Environment = {
  code: 'ENV-2026-0001',
  system: SYPAGO,
  name: 'QA-1',
  kind: 'baseline',
  os: 'linux',
  notes: '',
  retired: false,
  installed: { deployment: 'DEP-2026-0001', build: { code: BUILD.code, version: BUILD.version, checksum: BUILD.checksum }, deployedAt: AT },
  createdBy: ANA,
  createdAt: AT,
  updatedAt: AT,
}
export const PRODUCTION: Environment = { ...ENVIRONMENT, code: 'ENV-2026-0002', name: 'PROD', kind: 'production', installed: undefined }

export const DEPLOYMENT: Deployment = {
  code: 'DEP-2026-0001',
  environment: { code: ENVIRONMENT.code, name: ENVIRONMENT.name, kind: ENVIRONMENT.kind },
  build: { code: BUILD.code, version: BUILD.version, checksum: BUILD.checksum },
  result: 'success',
  notes: 'Sin novedades',
  executedBy: ANA,
  executedAt: AT,
  createdAt: AT,
}

export const PACKAGE_WORKFLOW: Workflow = {
  entityType: 'package',
  states: [
    { key: 'in_assembly', name: 'En armado', position: 1, isInitial: true, isFinal: false },
    { key: 'in_testing', name: 'En pruebas', position: 2, isInitial: false, isFinal: false },
    { key: 'ready_to_release', name: 'Listo para liberar', position: 3, isInitial: false, isFinal: false },
    { key: 'approved_for_production', name: 'Aprobado para producción', position: 4, isInitial: false, isFinal: false },
    { key: 'deployed', name: 'Desplegado', position: 5, isInitial: false, isFinal: false },
    { key: 'verified', name: 'Verificado', position: 6, isInitial: false, isFinal: true },
  ],
  transitions: [
    { from: 'in_assembly', to: 'in_testing', permission: 'pkg.start_testing', requiresReason: false, rules: [] },
    { from: 'in_testing', to: 'in_assembly', permission: 'pkg.reopen', requiresReason: true, rules: [] },
    { from: 'in_testing', to: 'ready_to_release', permission: 'pkg.mark_ready', requiresReason: false, rules: [] },
    { from: 'ready_to_release', to: 'approved_for_production', permission: 'pkg.approve_production', requiresReason: false, rules: [] },
    { from: 'approved_for_production', to: 'deployed', permission: 'pkg.deploy', requiresReason: false, rules: [] },
    { from: 'deployed', to: 'verified', permission: 'pkg.verify', requiresReason: false, rules: [] },
  ],
}

export const PACKAGE: Package = {
  code: 'PKG-2026-0001',
  title: 'SyPago 2.14',
  description: 'Límites por ordenante',
  system: SYPAGO,
  state: { key: 'in_assembly', name: 'En armado' },
  builds: [{ code: BUILD.code, version: BUILD.version, checksum: BUILD.checksum }],
  requirements: [{ code: 'REQ-2026-0002', title: 'Comprobante con referencia', state: { key: 'in_package', name: 'En paquete' } }],
  missing: [
    {
      to: { key: 'in_testing', name: 'En pruebas' },
      conditions: [{ ruleKey: 'regression_confirmed', message: 'La selección de regresión no está confirmada.', blocks: true, related: [] }],
    },
  ],
  createdBy: JUAN,
  createdAt: AT,
  updatedAt: AT,
}
export const PACKAGE_IN_TESTING: Package = {
  ...PACKAGE,
  state: { key: 'in_testing', name: 'En pruebas' },
  missing: [
    { to: { key: 'in_assembly', name: 'En armado' }, conditions: [] },
    {
      to: { key: 'ready_to_release', name: 'Listo para liberar' },
      conditions: [
        { ruleKey: 'regression_passed', message: 'Falta 1 TC de la regresión sin RUN aprobado: TC-2026-0003.', blocks: true, related: ['TC-2026-0003'] },
        { ruleKey: 'no_open_defect_with_severity', message: 'Hay 1 DEF abierto de severidad alta: DEF-2026-0001.', blocks: false, related: ['DEF-2026-0001'] },
      ],
    },
  ],
}

const component = { code: COMPONENT.code, name: COMPONENT.name, kind: COMPONENT.kind }

export const REGRESSION: Regression = {
  package: PACKAGE.code,
  proposedBy: ANA,
  proposedAt: AT,
  summary: { proposed: 2, included: 2, added: 1, removed: 1, passed: 1, failed: 0, pending: 1 },
  items: [
    {
      testCase: { code: 'TC-2026-0001', title: 'Rechaza el pago que supera el límite' },
      isCritical: true,
      origin: 'impact',
      proposed: true,
      included: true,
      reason: '',
      components: [component],
      lastRun: { code: 'RUN-2026-0009', result: 'passed', buildVersion: '2.14.0', environmentName: 'QA-1', build: BUILD.code, environment: ENVIRONMENT.code, executedAt: AT },
    },
    {
      testCase: { code: 'TC-2026-0002', title: 'Acepta el pago dentro del límite' },
      isCritical: false,
      origin: 'impact',
      proposed: true,
      included: false,
      reason: 'Lo cubre TC-2026-0001',
      decidedBy: ANA,
      decidedAt: AT,
      components: [component],
    },
    {
      testCase: { code: 'TC-2026-0003', title: 'Un pago reversado libera el acumulado' },
      isCritical: false,
      origin: 'manual',
      proposed: false,
      included: true,
      reason: 'Falló en el paquete anterior',
      decidedBy: ANA,
      decidedAt: AT,
      components: [],
    },
  ],
}
export const REGRESSION_CONFIRMED: Regression = { ...REGRESSION, confirmedBy: ANA, confirmedAt: AT }
