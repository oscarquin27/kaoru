# Registro de decisiones

Reglas del registro en [reglas.md §A.21](reglas.md). Una decisión no se borra: si cambia, se escribe otra que la reemplaza y la anterior se marca como reemplazada.

D1 a D10 vienen de la sección 15 del [requerimiento](kaoru-requerimiento-v2.md). Donde el requerimiento no dice el motivo, queda como «no registrado» en lugar de inventarlo.

## Vigentes

### D1 — Documentación de requerimientos en Google Sheets
- **Fecha:** 11 nov 2026
- **Contexto:** la documentación de los requerimientos ya vive en Google Sheets y el servidor de Kaoru está offline.
- **Decisión:** sigue en Google Sheets. Kaoru la enlaza con enlaces estructurados y permite adjuntar instantáneas como evidencia (requerimiento §7.5.1). Sin API de Google.
- **Motivo:** no reemplazar lo que el equipo ya usa; el contenido vivo puede cambiar, así que para el informe de conformidad cuenta la instantánea.

### D2 — Keycloak dedicado
- **Fecha:** 11 nov 2026
- **Contexto:** el sistema necesita login único, roles y política de contraseñas sin guardar contraseñas en la aplicación.
- **Decisión:** una instancia de Keycloak solo para Kaoru, en el mismo Docker Compose, con usuarios locales y base PostgreSQL separada.
- **Motivo:** no registrado en el requerimiento.

### D3 — Despliegue con Docker y Docker Compose
- **Fecha:** 11 nov 2026
- **Contexto:** servidores bancarios offline, Windows y Linux, y algunos en AWS.
- **Decisión:** Docker y Docker Compose, con un paquete de imágenes (`docker save` / `docker load`) para instalar sin internet.
- **Motivo:** las herramientas no pueden depender de servicios en la nube.

### D4 — Solo Google Chrome
- **Fecha:** 11 nov 2026
- **Contexto:** navegador de los equipos de usuario.
- **Decisión:** se soporta únicamente Google Chrome estable de escritorio.
- **Motivo:** permite usar CSS moderno sin polyfills (requerimiento §7.2).

### D5 — Stack
- **Fecha:** 10 nov 2026
- **Contexto:** elección de tecnologías del sistema.
- **Decisión:** Go + React (TypeScript) + PostgreSQL + Keycloak.
- **Motivo:** no registrado en el requerimiento.

### D6 — Estilo visual
- **Fecha:** 10 nov 2026
- **Contexto:** dirección visual de la interfaz.
- **Decisión:** glass, elegante y moderno, con tema oscuro predeterminado (requerimiento §7).
- **Motivo:** no registrado en el requerimiento.

### D7 — Reglas del repositorio
- **Fecha:** 11 nov 2026
- **Contexto:** la empresa ya tiene reglas de arquitectura en Aldía.
- **Decisión:** se adoptan esas reglas, adaptadas a este sistema, en [reglas.md](reglas.md). KISS y DRY mandan.
- **Motivo:** no registrado en el requerimiento.

### D8 — Sesión por cookie en el perímetro
- **Fecha:** 11 nov 2026
- **Contexto:** cómo mantiene la sesión el navegador.
- **Decisión:** Nginx + oauth2-proxy con cookie, en lugar de PKCE en la SPA.
- **Motivo:** el navegador nunca ve un token.

### D9 — Front sin librería de componentes
- **Fecha:** 11 nov 2026
- **Contexto:** origen de la apariencia y los textos del front.
- **Decisión:** el boceto propio (`tools/boceto`) es la fuente de apariencia; sin librería de componentes ni Tailwind.
- **Motivo:** una sola fuente de apariencia y textos (reglas §A.1.2).

### D10 — Nombre
- **Fecha:** 11 nov 2026
- **Contexto:** nombre del sistema y de sus piezas.
- **Decisión:** el sistema se llama Kaoru. Repositorio `kaoru`, realm `kaoru`, binario `kaoru-api`, esquema `kaoru`.
- **Motivo:** en honor a Kaoru Ishikawa (requerimiento §1 y §7.0).

### D11 — El Anexo A se movió a `docs/reglas.md`
- **Fecha:** 11 nov 2026
- **Contexto:** el requerimiento (§0, punto 2) pide que el primer commit copie el Anexo A a `docs/reglas.md` y que el requerimiento lo enlace en lugar de repetirlo.
- **Decisión:** el texto del anexo se movió sin cambios a `docs/reglas.md`; el requerimiento conserva solo el enlace.
- **Motivo:** una sola fuente por conocimiento (DRY).

## Propuestas `[SUPUESTO]` — contradicciones encontradas, por confirmar

El requerimiento (§0, puntos 3 y 7) pide señalar las contradicciones y proponer una salida. En las cuatro se propone que mande `reglas.md`. Ninguna está aplicada como definitiva hasta que QA la confirme.

### D12 — Carpeta de despliegue: `infra/` `[SUPUESTO]`
- **Fecha:** 11 nov 2026
- **Contexto:** el requerimiento §4 versiona el realm en `/deploy/keycloak/`; reglas §A.3 pone Compose, Nginx, oauth2-proxy y el realm en `infra/`.
- **Propuesta:** usar `infra/keycloak/` y corregir §4.
- **Motivo:** las reglas mandan sobre los detalles técnicos.

### D13 — Endpoints de salud: `/livez` y `/readyz` `[SUPUESTO]`
- **Fecha:** 11 nov 2026
- **Contexto:** el requerimiento §3 pide un endpoint `/healthz`; reglas §A.14 define `/livez` y `/readyz` en un puerto interno.
- **Propuesta:** solo `/livez` y `/readyz`; los healthchecks de Compose usan `/readyz`. Corregir §3.
- **Motivo:** las reglas mandan; un tercer endpoint no agrega nada (KISS).

### D14 — Accesibilidad: WCAG 2.2 AA `[SUPUESTO]`
- **Fecha:** 11 nov 2026
- **Contexto:** el requerimiento §7.2 pide WCAG 2.1 AA; reglas §A.16.2 pide WCAG 2.2 AA.
- **Propuesta:** WCAG 2.2 AA, que contiene a la 2.1. Corregir §7.2.
- **Motivo:** las reglas mandan y la 2.2 es la más exigente de las dos.

### D15 — `go.mod` en `src/` `[SUPUESTO]`
- **Fecha:** 11 nov 2026
- **Contexto:** la carpeta traía un `go.mod` en la raíz (`module kaoru`, sin versión de Go); reglas §A.3 pone el módulo Go en `src/go.mod`.
- **Propuesta:** crear el módulo en `src/` al empezar el código y quitar el de la raíz. Mientras tanto, el de la raíz no se versiona ni se toca.
- **Motivo:** las reglas mandan; la raíz queda para `docs/`, `infra/`, `api/`, `web/` y `tools/`.
