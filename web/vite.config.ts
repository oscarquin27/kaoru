import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'
import { mockApi } from './mocks/vite-plugin.ts'

// Con KAORU_DEV_API, `npm run dev` habla con un Kaoru real en vez de las respuestas grabadas:
// reenvía /api con la sesión de laboratorio que deja tools/dev-front.sh. Solo para desarrollo.
const realApi = process.env.KAORU_DEV_API
const proxy = realApi
  ? { '/api': { target: realApi, changeOrigin: true, headers: { Cookie: process.env.KAORU_DEV_COOKIE ?? '', Origin: realApi } } }
  : undefined

// El build sale con rutas absolutas desde «/»: Nginx sirve dist/ y hace try_files a /index.html.
export default defineConfig({
  plugins: [react(), ...(realApi ? [] : [mockApi()])],
  // El favicon es el símbolo del boceto: su carpeta de imágenes se sirve en la raíz.
  publicDir: '../tools/boceto/img',
  // Fuera de web/ solo se lee lo que es fuente única: el boceto y la tabla de casos compartida.
  server: { port: 5173, strictPort: true, proxy, fs: { allow: ['.', '../tools/boceto', '../api/testdata'] } },
  build: { target: 'chrome120' },
  test: {
    environment: 'jsdom',
    setupFiles: ['src/test-setup.ts'],
    restoreMocks: true,
    unstubGlobals: true,
  },
})
