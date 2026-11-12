import { expect, it } from 'vitest'
import { mockResponse } from './handler.ts'

it('responde lo grabado, sin importar el orden ni la codificación de la consulta', () => {
  expect(mockResponse('GET', '/api/search?q=límite').status).toBe(200)
  expect(mockResponse('GET', '/api/documents?entityCode=REQ-2026-0001&entityType=requirement&size=200').status).toBe(200)
  expect(mockResponse('GET', '/api/documents/DOC-2026-0001/versions/2/content').contentType).toContain('text/markdown')
})

it('lo que no está grabado es un 404 problem+json', () => {
  const response = mockResponse('GET', '/api/requirements?state=closed')
  expect(response).toMatchObject({ status: 404, contentType: 'application/problem+json' })
})

it('cualquier escritura es un 409 que dice que no guarda cambios', () => {
  const response = mockResponse('POST', '/api/requirements')
  expect(response.status).toBe(409)
  expect(JSON.parse(response.body)).toMatchObject({ code: 'INVALID_ACTION', detail: 'Servidor de datos de ejemplo: no guarda cambios.' })
})
