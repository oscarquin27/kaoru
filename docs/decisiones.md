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

## Propuestas `[SUPUESTO]` — diseño de la fase 1, por aprobar por QA y dev

Tomadas el 11 nov 2026 para poder avanzar con el diseño mientras la fase 0 sigue abierta. Ninguna está aprobada.

### D16 — El esquema de la fase 1 cubre solo el MVP `[SUPUESTO]`
- **Contexto:** la fase 1 pide el esquema SQL; las reglas (§A.1.1, §A.3) piden no escribir lo que ninguna fase usa todavía.
- **Propuesta:** las migraciones crean las tablas de la fase 2. Las entidades de las fases 3 y 4 quedan diseñadas en `docs/modelo-datos.md` y `docs/flujos.md`, sin tablas.
- **Motivo:** KISS; una migración aplicada no se edita, así que conviene no adelantar tablas que van a cambiar.

### D17 — Estados del CHG `[SUPUESTO]`
- **Contexto:** el requerimiento pide la máquina de estados del CHG pero solo define sus rutas (§8.2).
- **Propuesta:** `Registrado → Aprobado → En curso → Liberado → Cerrado`, más `Rechazado`. La ruta no cambia los estados, sino las reglas que aplican.
- **Motivo:** lo mínimo que permite aprobar y seguir un cambio. Lo decide change management.

### D18 — Dos roles de base y tablas de solo inserción `[SUPUESTO]`
- **Contexto:** la regla de negocio 9 pide que el usuario de la aplicación no tenga `UPDATE` ni `DELETE` sobre la auditoría, pero la aplicación aplica sus migraciones al arrancar, y el dueño de una tabla siempre puede darse permisos.
- **Propuesta:** `kaoru_owner` migra y `kaoru_app` ejecuta. `kaoru_app` solo inserta y lee en `audit_entry`, `test_run`, `document_version` y `comment`. `kaoru-api` recibe dos cadenas de conexión.
- **Motivo:** es la única forma de que la restricción sea real. `make check` la comprueba.

### D19 — El tiempo en cada estado sale de la auditoría `[SUPUESTO]`
- **Contexto:** la línea de vida y los defectos muestran el tiempo en cada estado.
- **Propuesta:** se calcula al leer, con las filas `action = 'transition'` de `audit_entry`. No hay tabla de historial de estados.
- **Motivo:** una sola fuente (DRY) y reglas §A.11: lo que depende del día se calcula al leer. Si la medición de la fase 2 lo pide, se agrega un índice o una tabla.

### D20 — La API direcciona por código `[SUPUESTO]`
- **Contexto:** los IDs legibles son únicos e inmutables, y las migas de pan y la búsqueda los usan.
- **Propuesta:** `/api/requirements/REQ-2026-0042`. El `uuid` sigue siendo la llave interna y solo se expone en recursos hijos sin código (enlaces, comentarios).
- **Motivo:** las URL del front y de la API coinciden y se ahorra una consulta de traducción.

### D21 — RUN y EXP guardan BUILD y ambiente como texto hasta la fase 3 `[SUPUESTO]`
- **Contexto:** el MVP registra RUN «por BUILD y ENV», pero BUILD- y ENV- son de la fase 3.
- **Propuesta:** `build_version`, `build_checksum` y `environment_name` como texto; la fase 3 los cambia por llaves foráneas con una migración.
- **Motivo:** no adelantar dos features. Costo: hasta la fase 3 el checksum del BUILD se registra, pero no se verifica contra un artefacto.

### D22 — Core mínimo propio `[SUPUESTO]`
- **Contexto:** la pregunta 15 (reutilizar el core de Aldía) sigue abierta.
- **Propuesta:** si no hay respuesta al empezar la fase 2, se escribe un core mínimo con la misma forma (reglas §A.4).
- **Motivo:** no bloquear la fase 2. Si después se adopta el core de la empresa, el cambio queda dentro de `core/`.

### D23 — Comentarios como feature propio `[SUPUESTO]`
- **Contexto:** la pestaña Actividad muestra comentarios y `dev` comenta los DEF, pero la lista de features (§A.4.1) no dice quién es dueño de los comentarios.
- **Propuesta:** un feature `comment` en la fase 2, con la tabla `comment` (hoy creada en la migración 00006).
- **Motivo:** los comentarios aplican a REQ, CHG y DEF; ponerlos dentro de uno obligaría a los otros a depender de él.

### D24 — Ediciones de las normas `[SUPUESTO]`
- **Contexto:** el requerimiento cita las normas sin edición.
- **Propuesta:** ISO/IEC 25010:2011 (ocho características), ISO/IEC/IEEE 29119-3:2013, ISO 9001:2015 e ISO/IEC 20000-1:2018.
- **Motivo:** son las que usa `docs/correspondencia-normas.md`. Las referencias deben verificarse contra el texto oficial.

### D25 — Valores en inglés, etiquetas en español `[SUPUESTO]`
- **Contexto:** reglas §A.19: identificadores en inglés, textos al usuario en español.
- **Propuesta:** la base y la API guardan `high`, `passed`, `in_testing`; el nombre visible («Alta», «En pruebas») sale de `workflow_state.name` o del front.
- **Motivo:** un valor de `CHECK` o de un enum es un identificador.

### D26 — Supuestos del modelo `[SUPUESTO]`
- **Contexto:** el requerimiento no fija estas cardinalidades.
- **Propuesta:** un CHG pertenece a un solo sistema y sus REQ son de ese sistema; un REQ puede existir sin CHG; hay un RISK vigente por REQ (reevaluar lo actualiza); un UC pertenece a un solo REQ; un TC valida uno o más UC; un DEF siempre lleva el RUN que lo reprodujo.
- **Motivo:** la lectura más simple de las secciones 6 y 9.

### D27 — Versiones y comprobaciones provisionales `[SUPUESTO]`
- **Contexto:** `make check` necesita versiones concretas, y el lint de OpenAPI no se pudo ejecutar en esta sesión.
- **Propuesta:** PostgreSQL 17 y goose 3.24.1 para probar migraciones; el realm se probó en Keycloak 26.0. El contrato se comprueba con `tools/check-openapi.rb` (sintaxis, referencias, `operationId`), que no reemplaza al lint de §A.18. Las versiones definitivas, con digest, se fijan en el Compose de la fase 2.
- **Motivo:** tener una comprobación real hoy sin fijar todavía la infraestructura.
