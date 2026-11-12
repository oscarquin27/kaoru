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

## Aprobación del diseño de la fase 1

### D28 — Esquema y boceto aprobados
- **Fecha:** 11 nov 2026
- **Contexto:** criterio de salida de la fase 1.
- **Decisión:** se aprueban el esquema y el boceto para pasar a la fase 2. Con el esquema quedan vigentes D16 a D21, D23, D25 y D26.
- **Motivo:** aprobación dada en la sesión de trabajo. `[SUPUESTO]` El requerimiento pide la aprobación de QA y dev: falta dejar constancia de quién firma por cada uno.
- **Siguen como propuesta:** D12, D13, D14, D15, D22, D24 y D27. De ellas, D12, D13, D15 y D22 ya están aplicadas en el código de la fase 2; si alguna se rechaza, hay que deshacerla.

## Decisiones de la fase 2 (cimientos del backend)

Tomadas el 11 nov 2026 al construir el core y el primer corte vertical (`workflow`, `audit`, `user`, `change`). `[SUPUESTO]` hasta que dev las revise.

### D29 — Core propio mínimo y sus librerías
- **Contexto:** sin respuesta a la pregunta 15 (D22). Reglas §A.1.1: sumar una herramienta se justifica aquí.
- **Decisión:** `internal/core/pkg/` con `axis` (errores con código), `problem` (RFC 9457), `httpserver`, `oidc`, `postgres`, `uow`, `clock`, `idgen` y `logging`. Librerías: gin (router), pgx v5 + sqlc (base), goose (migraciones), `coreos/go-oidc` (firma, emisor y vigencia del token, con JWKS en caché), `google/uuid` (UUIDv7) y `x/sync` (errgroup). Logs con `log/slog` de la biblioteca estándar.
- **Motivo:** una herramienta por problema y nada propio donde hay una librería conocida. Si se adopta el core de la empresa, el cambio queda dentro de `core/`.

### D30 — `audit` y `user` dependen de `workflow` para los permisos
- **Contexto:** las reglas piden construir `user`, `audit` y `workflow` primero, y que los permisos se comprueben con la tabla de `workflow`.
- **Decisión:** orden `workflow` → `audit` → `user` → resto. Hoy `workflow` solo lee, así que no audita nada y no hay ciclo.
- **Pendiente conocido:** cuando `qa-admin` pueda editar flujos y permisos, `workflow` tendrá que auditar y aparecerá el ciclo `workflow` ↔ `audit`. La salida prevista es la de reglas §A.7: un puerto que `bootstrap` conecta después de armar los dos.

### D31 — Cada feature reserva sus IDs legibles en `code_sequence`
- **Contexto:** `code_sequence` es una tabla compartida y las reglas prohíben escribir en la tabla de otro feature.
- **Decisión:** `code_sequence` no es de ningún feature. Cada uno tiene su consulta `NextCode` (dos líneas de SQL) y `archtest` la declara como tabla permitida.
- **Motivo:** regla de tres. Al tercer feature que la repita se evalúa subirla a un lugar común.

### D32 — El CHG no devuelve cuántos REQ agrupa
- **Contexto:** el contrato traía `requirementCount` en el CHG. La llave va de `requirement` a `change`, y las reglas (§A.7) solo permiten leer la tabla de otro feature en la dirección de la llave.
- **Decisión:** se quitó `requirementCount` del contrato. La ficha del CHG lista sus REQ con `/api/requirements?change=…`; el conteo consolidado, si hace falta, lo da `traceability`.
- **Motivo:** no romper la dependencia de un solo sentido entre features.

### D33 — Imagen `scratch` con healthcheck propio
- **Contexto:** todos los servicios del Compose necesitan healthcheck, y una imagen de un solo binario no trae `curl` ni shell.
- **Decisión:** `kaoru-api healthcheck` consulta `/readyz` del proceso local y sale con 0 o 1. La imagen corre como usuario sin privilegios.
- **Motivo:** menos superficie y nada que actualizar dentro de la imagen.

### D34 — El perfil del usuario se guarda al llegar la petición
- **Contexto:** la auditoría y los responsables referencian a `app_user`, pero los usuarios nacen en Keycloak.
- **Decisión:** un middleware de `user` guarda el perfil del token la primera vez que lo ve en el proceso, y de nuevo si cambia.
- **Motivo:** ningún feature tiene que preocuparse de que el usuario exista.

### D35 — Lo que `make check` todavía no cubre
- **Contexto:** reglas §A.14 y §A.18.
- **Pendiente:** `golangci-lint` (no está instalado; hoy corren `gofmt` y `go vet`), `sqlc vet`, el lint completo de OpenAPI y las métricas Prometheus de §A.14. El `staticcheck` instalado es de Go 1.24 y no analiza un módulo de Go 1.26.
- **Motivo de registrarlo:** no dar por cumplido lo que no se ejecutó.

### D36 — Los hechos de las reglas de bloqueo se conectan al arrancar
- **Fecha:** 11 nov 2026
- **Contexto:** aprobar o entregar un REQ exige mirar su RISK, su cobertura y sus defectos, pero `risk`, `testdesign`, `execution` y `defect` dependen de `requirement`, no al revés. Las reglas (§A.7) piden dependencias de un solo sentido y dan una única salida para un ciclo.
- **Decisión:** `requirement` declara el puerto `Facts` y evalúa las reglas en su dominio. Su adaptador `adapter/facts` llama a los casos de uso de los otros features, y `bootstrap` conecta cada fuente después de armar todos los módulos (`requirementModule.ConnectRisk(…)`). Una fuente sin conectar responde «no hay»: sin RISK, sin TC y sin defectos, que es lo que bloquea.
- **Motivo:** es la salida que dan las reglas, deja la evaluación en el dominio del REQ y mantiene a `traceability` como solo lectura. Costo: `requirement` y esos features se conocen en los dos sentidos, solo a través de casos de uso.
- **Estado:** `[SUPUESTO]` hasta que dev la revise. Conectado hoy: `risk`.

### D37 — El nivel de riesgo no viaja dentro del REQ
- **Fecha:** 11 nov 2026
- **Contexto:** el contrato traía `riskLevel` en el REQ. La llave va de `risk_assessment` a `requirement`, así que `requirement` no puede leer esa tabla (reglas §A.7).
- **Decisión:** se quitó `riskLevel` del esquema `Requirement`. El riesgo se lee en `/api/requirements/{code}/risk`; la ficha consolidada y la matriz, que arma `traceability`, lo incluyen.
- **Motivo:** el mismo de D32.

### D38 — Decisiones de `testdesign`, `execution` y `defect`
- **Fecha:** 11 nov 2026
- **Estado:** `[SUPUESTO]` hasta que QA y dev las revisen.
- **Un DEF exige un RUN que no esté aprobado.** La regla de negocio 5 pide reproducir en un RUN lo que se encuentra en una exploración; un RUN aprobado no reproduce nada, así que solo valen `failed` o `blocked`.
- **Un TC retirado no se exige en la cobertura ni admite RUN nuevos**, pero conserva los que tiene.
- **La cobertura cuenta el último RUN de cada TC.** Un TC aprobado ayer y fallido hoy está pendiente.
- **«Defecto abierto» es cualquier DEF que no esté en un estado final.** Un DEF diferido cuenta como abierto, pero la regla de bloqueo solo mira los de severidad alta, que no se pueden diferir.
- **El tiempo en estado de un DEF sale de la auditoría** (D19) con una consulta por lista, no una por defecto.
- **`execution` y `defect` leen `test_case`, `requirement` y `test_run` por SQL** en la dirección de su llave foránea (reglas §A.7), en lugar de un puerto por cada lectura. La única llamada entre features es la de `execution` a `testdesign` para saber qué TC validan un REQ, porque esa relación no está en sus tablas.
- **Se quitó `lastRun` del TC en el contrato**, por el mismo motivo de D32 y D37: lo da `/api/runs?testCase=…` y lo dará la ficha consolidada.

### D39 — Deuda conocida al cerrar el flujo central del backend
- **Fecha:** 11 nov 2026
- **Sin pruebas unitarias de casos de uso** en `risk`, `testdesign`, `execution`, `defect` y `user`: están cubiertos por pruebas de dominio y por las de integración contra Postgres, pero falta lo que pide reglas §A.17 (cada permiso negado, cada error de negocio y «nada se guarda si la auditoría falla» con dobles de puertos).
- **Sin comprobar que el RUN de un DEF sea de un TC de ese REQ.** Hoy se puede abrir un DEF en un REQ con el RUN de otro.
- **Sin control de escrituras simultáneas** sobre la misma entidad: gana la última.
- **Sin la ruta de la ficha consolidada** (`/api/requirements/{code}/overview`): el caso de uso `GetReadiness` existe y está probado, pero lo expone `traceability`, que no está hecho.

### D40 — Decisiones al terminar el backend del MVP
- **Fecha:** 11 nov 2026
- **Estado:** `[SUPUESTO]` hasta que QA y dev las revisen.
- **Documentos.** El contenido vive en un directorio (un volumen), nombrado por su SHA-256; la base guarda solo metadatos. El tipo se valida **antes de escribir en disco** y por el contenido real, no por el nombre: la extensión debe estar permitida (`documents.allowedExtensions`) y coincidir con lo que se detecta. El tamaño máximo es `documents.maxBytes` (25 MB). El contenido se sirve con `nosniff` y `Content-Security-Policy: sandbox`; el texto y el Markdown salen como texto plano y el front los pinta.
- **Migración 00012.** `document_link` guarda también el código de la entidad, para que `document` no lea tablas de otros features. Por eso el contrato devuelve `linkedTo` con tipo y código, sin título.
- **Comentarios.** Son de solo inserción y quedan además en la auditoría de la entidad comentada. Admiten REQ, CHG y DEF.
- **`traceability` no repite reglas.** La cobertura la pregunta a `execution` y «qué falta para avanzar» a `requirement`; solo las listas y la matriz salen de sus propias consultas de lectura.
- **«REQ bloqueados» en la matriz** son los que tienen algún DEF de severidad alta abierto. Evaluar todas las reglas de cada REQ en una lista sería caro; el detalle completo está en la ficha.
- **Línea de vida.** Solo los estados con posición son etapas. Mientras un REQ está en un estado lateral (con defectos, diferido, rechazado), ninguna etapa aparece como actual; el estado va en el encabezado.
- **Exportaciones.** CSV y XLSX se escriben con la biblioteca estándar (un XLSX de una hoja es un zip con cinco XML), sin librería. Topes: 50 000 filas de matriz y 200 000 registros de auditoría. Las celdas que empiezan con `=`, `+`, `-` o `@` se neutralizan para que Excel no las ejecute como fórmula.
- **Administración.** `qa-admin` cambia los permisos de cada rol y ajusta o desactiva reglas de bloqueo; todo queda auditado. `qa-admin` no puede quedarse sin `admin.configure`. El ciclo `workflow` ↔ `audit` previsto en D30 se resolvió como dicen las reglas: `bootstrap` conecta la auditoría a `workflow` después de armar los dos.
- **Fuera de esta entrega:** editar estados y transiciones desde la administración (hoy se cambian por migración), las vistas guardadas, y la importación de la hoja, que sigue esperando su formato (pregunta 5).

### D41 — Deuda conocida del backend
- **Fecha:** 11 nov 2026
- Se mantiene lo de D39: faltan pruebas unitarias de casos de uso con dobles de puertos en `risk`, `testdesign`, `execution`, `defect`, `user`, y ahora también en `document`, `comment`, `traceability` y la administración de `workflow`. Están cubiertos por pruebas de dominio e integración.
- Un archivo subido cuyo registro falla después queda en disco sin referencia. No hay tarea de limpieza.
- La previsualización de Office con LibreOffice no está hecha: esos archivos se descargan.
- No hay medición de rendimiento con 10 000 REQ y 100 000 RUN; la matriz y la búsqueda no tienen índices de texto.

### D42 — Despliegue: Compose, sesión y paquete offline
- **Fecha:** 11 nov 2026
- **Estado:** `[SUPUESTO]` hasta que dev y quien administre el servidor lo revisen.
- **Rutas.** Nginx atiende todo en un solo origen: `/` el front, `/api/` la API, `/oauth2/` la sesión y `/auth/realms/` el login de Keycloak. Keycloak corre con ruta base `/auth`. Así no hace falta un segundo nombre de host ni un segundo certificado.
- **Keycloak por dentro y por fuera.** El emisor de los tokens es la URL pública (`KC_HOSTNAME`). oauth2-proxy y `kaoru-api` le hablan por la red interna de Docker en HTTP (canje del código y llaves), sin depender de que el servidor resuelva su propio nombre público.
- **Consola de administración.** No se publica en Nginx. Atiende solo en `127.0.0.1` del servidor (`KEYCLOAK_ADMIN_PORT`).
- **Sesión.** Cookie cifrada de oauth2-proxy, `HttpOnly`, `Secure`, `SameSite=Lax`, 10 horas. El token de acceso dura 5 minutos y oauth2-proxy lo renueva a los 3. Al cerrar sesión se cierra también la de Keycloak.
- **oauth2-proxy en su variante `-alpine`.** La imagen normal no trae shell y no admite healthcheck dentro del contenedor, que el requerimiento pide para todos los servicios.
- **Digest.** El Compose del repositorio fija cada imagen por tag y digest. El del paquete offline las referencia solo por tag, porque `docker load` no conserva el digest en todos los motores; la integridad la da la suma de `images.tar.gz`, que `install.sh` comprueba. Esto resuelve D27 para las versiones: PostgreSQL 17, Keycloak 26.4, oauth2-proxy 7.8.1 y Nginx 1.28.
- **Respaldo.** `pg_dump` de las dos bases y un `tar.gz` del volumen de documentos, con sumas de comprobación. La restauración reemplaza todo y se niega a correr con un respaldo dañado.
- **TLS.** `tools/make-lab-cert.sh` genera un certificado autofirmado solo para laboratorio. Cuál se usa en el banco sigue `[POR DEFINIR]` (pregunta 12).
- **El front es un marcador de posición.** Nginx sirve una página mínima detrás de la sesión hasta que exista la SPA.

### D43 — Deuda cerrada al preparar la entrega
- **Fecha:** 11 nov 2026
- **Un DEF exige un RUN de un TC de su REQ.** Cierra la deuda de D39: ya no se puede abrir un defecto en un REQ con la evidencia de otro.
- **Escrituras simultáneas.** Todo caso de uso que escribe sobre un CHG, un REQ o un DEF bloquea su fila (`SELECT … FOR UPDATE`) al empezar la transacción. Dos cambios sobre la misma entidad van uno tras otro y el segundo ve lo que dejó el primero. Cierra la deuda de D39.
- **Métricas Prometheus** en el puerto interno (`/metrics`): peticiones y duración por método, plantilla de ruta y código, y las estadísticas del pool de Postgres (reglas §A.14). Cierra parte de D35.
- **Pruebas unitarias de casos de uso** agregadas en `risk` y `defect`. Siguen pendientes en `user`, `testdesign`, `execution`, `document`, `comment`, `traceability` y la administración de `workflow` (D41).
- **Siguen sin correr:** `golangci-lint`, `sqlc vet` y el lint completo de OpenAPI (D35).

### D44 — El front
- **Fecha:** 11 nov 2026
- **Estado:** `[SUPUESTO]` hasta que QA y dev lo revisen en Chrome con sesión real.
- **Qué es.** SPA de React + TypeScript + Vite con TanStack Query y react-router, en `web/`. Sin librería de componentes ni Tailwind: reutiliza el CSS del boceto con sus mismas clases (`web/src/styles` lo importa de `tools/boceto/css`). Los tipos salen de `api/openapi.yaml` con `openapi-typescript`.
- **TypeScript 5**, no 7: lo exige `openapi-typescript`.
- **Markdown** con un renderizador propio pequeño que genera nodos React; no inserta HTML del documento.
- **Rutas.** Los IDs legibles son las rutas (`/requerimientos/REQ-2026-0042`). La matriz vive en `/pruebas` y la auditoría en `/administracion/auditoria`, porque el boceto no les da sección propia.
- **Lo que el contrato no permite todavía:** elegir a otro usuario como responsable o asignado (no hay endpoint de usuarios; solo «yo»), columnas y filtros de PKG y RPT en la matriz, alertas de atraso en Inicio, comparar versiones de un documento y editar los parámetros de una regla de bloqueo desde la pantalla.
- **Cómo se verificó.** Lint, tipos, 52 pruebas y build; visto en el navegador contra respuestas reales de la API grabadas (`web/mocks/fixtures.json`); y `tools/e2e-front.sh` comprueba que el front compilado se sirve solo con sesión, que sus archivos cargan con la política de seguridad de Nginx y que las rutas abren directo.
- **Lo que no se verificó:** el front escribiendo contra la API real en un navegador (alta, transición, subida de archivos, el 401 real), porque el navegador de la sesión de trabajo no confía en el certificado autofirmado de laboratorio. Los cuerpos que envía están tipados contra el contrato y las lecturas se probaron con respuestas reales, pero falta la prueba manual en Chrome.

### D45 — Responsables, edición de flujos y deuda cerrada
- **Fecha:** 11 nov 2026
- **Estado:** `[SUPUESTO]` hasta que QA y dev lo revisen.
- **Lista de usuarios (`GET /api/users`).** Devuelve a quienes ya entraron a Kaoru alguna vez (id, usuario y nombre; sin correo), hasta 50, con búsqueda por nombre o usuario. El directorio vive en Keycloak y Kaoru no lo consulta: alguien que nunca entró no aparece y no se le puede asignar nada todavía. Cualquier usuario con sesión puede leerla, porque hace falta para asignar. Con esto el front deja elegir a cualquier persona como responsable de un REQ o asignada a un DEF, y filtrar por ella. Cierra un límite de D44.
- **Estados y transiciones se editan desde la administración.** Con `admin.configure` se puede agregar un estado, cambiarle el nombre o su posición en la línea de vida, quitarlo, agregar o quitar una transición y cambiar el permiso que exige o si pide motivo. Vale de inmediato y queda auditado con el antes y el después. Cierra lo que D40 dejó fuera. Límites, a propósito:
  - La **clave** de un estado no cambia: la referencian las transiciones y el historial.
  - El **estado inicial** no se cambia ni se quita, y un estado nuevo nunca nace como inicial.
  - Un estado **con transiciones o con registros dentro** no se quita (lo segundo lo garantiza la base).
  - Quitar una transición **quita sus reglas de bloqueo**. Agregar reglas nuevas a una transición sigue siendo por migración: desde la pantalla solo se activan, desactivan o ajustan las que existen.
  - No se valida que el flujo quede «bien formado» (que todo estado sea alcanzable): es responsabilidad de quien configura.
- **Archivos sin registro.** `kaoru-api prune-documents` lista los archivos del almacén que ninguna versión referencia y tienen más de un día; con `--apply` los borra. No se limpia al fallar la subida porque el almacén deduplica por contenido: borrar en ese momento podría quitarle el archivo a otra subida simultánea del mismo contenido. Cierra una deuda de D41.
- **Pruebas unitarias de casos de uso** con dobles de puertos en todos los features. Cierra D39 y D41 en ese punto. En `document` queda explícito que «nada se guarda si la auditoría falla» vale para el registro, no para el archivo (que es lo que limpia la orden anterior).
- **`sqlc vet`** prepara cada consulta contra un Postgres real ya migrado (`make check-sqlvet`). La configuración con la base se arma al vuelo para que `sqlc generate` siga sin necesitar una.
- **Lint completo de OpenAPI** con las reglas recomendadas de Redocly, usando `@redocly/openapi-core`, que ya estaba instalado en `web/` como dependencia de `openapi-typescript`; no suma herramienta. Cuatro reglas van apagadas con su motivo en `tools/lint-openapi.mjs`. Cierra D35.
- **Front contra la API real en desarrollo.** `tools/dev-front.sh [usuario]` sirve el front con recarga y reenvía `/api` al Kaoru de desarrollo con la sesión de un usuario de laboratorio. Solo existe en `npm run dev`; el build no lo lleva.
- **Cómo se verificó.** Además de `make check`: en el navegador, contra la API real y con tres usuarios distintos, se agregó, ajustó y quitó un estado y una transición (y se vio en la auditoría); se dio de alta un REQ con otra persona como responsable; una transición bloqueada mostró qué faltaba (409 real); se evaluó el riesgo y la transición pasó; se subió un documento (y se rechazó uno cuyo contenido no correspondía a su extensión, 415 real) y se comentó. Cierra «lo que no se verificó» de D44, salvo el login y el logout en el navegador, que siguen probados solo por `tools/e2e-compose.sh`.
- **Sigue pendiente:** importación de la hoja (pregunta 5), vistas guardadas, previsualización de Office, tema de login de Keycloak, medición de rendimiento con volumen, columnas de PKG y RPT, y la instalación en un servidor realmente sin red.
