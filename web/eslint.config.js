import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import tseslint from 'typescript-eslint'

export default tseslint.config(
  { ignores: ['dist', 'src/shared/api/schema.d.ts'] },
  js.configs.recommended,
  tseslint.configs.recommended,
  reactHooks.configs.flat.recommended,
  {
    languageOptions: { globals: { ...globals.browser, ...globals.node } },
    rules: {
      // Reglas §A.16.2: los features no se importan entre sí; lo común va a shared/.
      'no-restricted-imports': ['error', { patterns: [{ group: ['**/features/*', '../../features/*'], message: 'Un feature no importa a otro: sube lo común a shared/ o arma la pantalla en app/.' }] }],
    },
  },
  { files: ['src/app/**', 'src/main.tsx'], rules: { 'no-restricted-imports': 'off' } },
)
