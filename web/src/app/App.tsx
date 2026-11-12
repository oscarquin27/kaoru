// Proveedores y rutas. Las rutas salen de la lista única de secciones.
import { QueryClientProvider, type QueryClient } from '@tanstack/react-query'
import { Route, Routes } from 'react-router'
import { Layout } from './Layout.tsx'
import { NotFoundPage } from './NotFoundPage.tsx'
import { ALL_SECTIONS } from './sections.tsx'

export function App({ queryClient }: { queryClient: QueryClient }) {
  return (
    <QueryClientProvider client={queryClient}>
      <Routes>
        <Route element={<Layout />}>
          {ALL_SECTIONS.flatMap((section) => section.screens).map((screen) => (
            <Route key={screen.path} path={screen.path} element={<screen.page />} />
          ))}
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </QueryClientProvider>
  )
}
