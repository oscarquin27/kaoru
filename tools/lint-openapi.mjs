// Lint completo del contrato con las reglas recomendadas de Redocly. Usa @redocly/openapi-core,
// que ya está instalado en web/ como dependencia de openapi-typescript: no descarga nada.
//   (cd web && node ../tools/lint-openapi.mjs)
import { createRequire } from 'node:module'
import { resolve } from 'node:path'

const require = createRequire(resolve(process.cwd(), 'package.json'))
const { createConfig, lint } = require('@redocly/openapi-core')

const config = await createConfig({
  extends: ['recommended'],
  rules: {
    // La API es interna, detrás del perímetro: sin licencia ni servidores públicos que declarar.
    'info-license': 'off',
    'no-server-example.com': 'off',
    // La sesión la pone oauth2-proxy para todas las rutas; no se repite operación por operación.
    'security-defined': 'off',
    // Los errores van todos por la respuesta `default` (problem+json).
    'operation-4xx-response': 'off',
  },
})
const problems = await lint({ ref: resolve(process.cwd(), '../api/openapi.yaml'), config })
for (const problem of problems) {
  const where = problem.location[0]
  console.log(`${problem.severity}: ${problem.ruleId}: ${problem.message}\n    en ${where?.pointer ?? '?'}`)
}
const errors = problems.filter((problem) => problem.severity === 'error').length
console.log(`Lint de OpenAPI: ${errors} errores, ${problems.length - errors} avisos`)
process.exit(problems.length > 0 ? 1 : 0)
