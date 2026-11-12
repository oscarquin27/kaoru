import type { Plugin } from 'vite'
import { mockResponse } from './handler.ts'

// Solo se engancha al servidor de desarrollo: el build y la vista previa no lo llevan.
export function mockApi(): Plugin {
  return {
    name: 'kaoru-mock-api',
    apply: 'serve',
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        if (!req.url?.startsWith('/api/')) return next()
        const response = mockResponse(req.method ?? 'GET', req.url)
        res.statusCode = response.status
        res.setHeader('Content-Type', response.contentType)
        res.end(response.body)
      })
    },
  }
}
