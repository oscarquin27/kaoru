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

### D46 — Vistas guardadas y vista previa de Office
- **Fecha:** 12 nov 2026
- **Estado:** `[SUPUESTO]` hasta que QA y dev lo revisen. La vista previa de Office responde a la pregunta 10, que sigue abierta: queda hecha pero **apagada por omisión**.
- **Vistas guardadas (§7.3).** Feature `savedview`, tabla `saved_view` (migración 00013). Una vista es el nombre que una persona le pone a los filtros de una lista (requerimientos, cambios, defectos, matriz, casos de prueba, auditoría). Cambia lo que decía `docs/modelo-datos.md` («no tienen tabla»).
  - Son **personales**: cada quien ve y quita solo las suyas. Compartir sigue siendo por URL.
  - **No exigen permiso y no se auditan**: son una preferencia de pantalla, no un dato del proceso.
  - Hasta 30 por lista y persona; el nombre no se repite dentro de una lista.
- **Vista previa de Office (§7.4).** Un `.docx` o `.xlsx` se ve dentro de Kaoru convertido a PDF, solo si el servidor tiene el servicio opcional `libreoffice`. El original es el que se descarga, el que lleva checksum y el que cuenta como evidencia; la pantalla lo dice.
  - **El conversor es Gotenberg 8.25** (un servicio HTTP que envuelve LibreOffice), con el navegador, los webhooks y las descargas apagados, sistema de archivos de solo lectura y sin puerto publicado: solo lo llama `kaoru-api`. Es una pieza nueva; se justifica porque LibreOffice no trae API y montarle una propia es más código que mantener.
  - **Pesa unos 2,7 GB.** Por eso no viaja en el paquete offline salvo que se pida (`OFFLINE_OFFICE=1 tools/offline-bundle.sh`) y se activa con dos líneas de `.env` (`COMPOSE_PROFILES=office` y `KAORU_CONVERTER_URL`).
  - **«Detectado al iniciar»** se resolvió por configuración, no sondeando: si `documents.converterUrl` está, las versiones de Office salen con `previewable` y existe `/preview`; si no, responde `NOT_SUPPORTED` y el front ofrece descargar. Si el conversor está configurado pero caído, la vista previa responde `SERVICE_UNAVAILABLE` y la descarga sigue funcionando.
  - **No se guarda el PDF convertido**: se convierte al pedirlo y el navegador lo conserva (la versión es inmutable). Guardarlo obligaría a relacionarlo con el original y a excluirlo de la limpieza de archivos sin registro.
  - La configuración admite ahora variables opcionales: `${VARIABLE:-}` queda vacía si falta.
- **Fallo corregido: el visor no mostraba ningún PDF.** Nginx mandaba `frame-ancestors 'none'` y `X-Frame-Options: DENY` en todas las respuestas, también en el contenido de los documentos, y el navegador se negaba a incrustarlo en el visor. No se había visto porque el front nunca se había probado en un navegador contra el despliegue (D44). Ahora el contenido y la vista previa de un documento (y solo esas dos rutas) se pueden incrustar desde el propio Kaoru; siguen exigiendo sesión, no aceptan un token puesto a mano, y `kaoru-api` los sirve aislados (`sandbox`). `tools/e2e-compose.sh` lo comprueba.
- **Cómo se verificó.** `make check`; un `.docx` real subido por el despliegue completo, convertido por el conversor real y visto en el navegador dentro del visor, igual que un PDF; el conversor arranca de solo lectura y con las rutas del navegador respondiendo 404. Las vistas guardadas, contra Postgres y con pruebas del front.
- **Lo que no se verificó:** `.xlsx` en el conversor real (solo `.docx`), documentos grandes o con macros, y el paquete offline con `OFFLINE_OFFICE=1`.

### D47 — Rendimiento con volumen, tema de login y manual de usuario
- **Fecha:** 12 nov 2026
- **Estado:** `[SUPUESTO]` hasta que QA y dev lo revisen.
- **Rendimiento (§11).** `make check-perf` carga 10 000 REQ, 20 000 TC, 100 000 RUN, 2 000 DEF y 30 000 registros de auditoría, y exige que la ficha del REQ y la búsqueda respondan en menos de 1 s. Medido en el equipo de desarrollo: ficha 40 ms, búsqueda 50 ms, listas entre 3 y 40 ms, matriz 170 ms. Es tiempo del servidor (API y base), sin red ni navegador, y **no es el servidor objetivo**, que sigue `[POR DEFINIR]` (pregunta 12): hay que repetirlo allí. Queda dentro de `make check`.
- **Fallo de rendimiento corregido.** Las listas con filtros opcionales («parámetro nulo O condición») tardaban hasta 0,5 s con volumen: tras cinco ejecuciones Postgres pasaba a un plan genérico que recorría la tabla de REQ una vez por cada estado del flujo. Ahora cada consulta se planifica con sus valores reales (`plan_cache_mode = force_custom_plan` en las conexiones de la aplicación, `corepg.NewPool`). Las listas quedaron entre 10 y 40 veces más rápidas. No hizo falta ningún índice nuevo.
- **Tema de login (§4, §7.0).** `infra/keycloak/themes/kaoru`: el tema de Keycloak (`keycloak.v2`) con una hoja de estilos, la marca y los textos de Kaoru. No cambia plantillas, para sobrevivir a actualizaciones de Keycloak. Sin recursos externos (`make check-generated` y `tools/e2e-compose.sh` lo comprueban).
  - Sigue el tema del sistema operativo (oscuro o claro), porque en el login todavía no hay preferencias de usuario.
  - Los colores repiten los tokens del boceto: Keycloak sirve sus temas aparte y no puede importar esa hoja. Es una copia consciente, anotada en el propio archivo.
  - **En una instalación que ya existe** el realm no se vuelve a importar: el tema se activa una vez en la consola de Keycloak (Realm settings → Themes → Login theme: `kaoru`). Tras cambiar archivos del tema hay que recrear el contenedor de Keycloak, que guarda los recursos en caché.
  - Políticas de contraseña, bloqueo por intentos fallidos y OTP opcional ya estaban en el realm desde D42; no cambian.
- **Fallo corregido: el símbolo de Kaoru no se pintaba.** `tools/boceto/img/kaoru.svg` tenía dos guiones seguidos dentro de un comentario, que XML no admite: el navegador descartaba la imagen. Afectaba al favicon de la aplicación y al login. Ahora `make check-generated` valida ese archivo.
- **Manual de usuario por rol** en `docs/manual-usuario.md` (§11).
- **Vista previa de Office:** probado también un `.xlsx` real (la matriz exportada por Kaoru) en el conversor real. Cierra una parte de lo no verificado en D46.
- **Lo que no se verificó:** el tema claro del login y sus pantallas de error, cambio de contraseña y OTP (solo se vio la pantalla de entrada en oscuro); el rendimiento en el servidor objetivo.
- **Sigue pendiente del MVP:** importación de la hoja (pregunta 5), visor de OpenAPI empaquetado, PDF.js empaquetado (hoy se usa el visor de PDF de Chrome) y resaltado de logs. Las columnas de PKG y RPT, el catálogo de componentes, los ambientes y el informe de conformidad son de las fases 3 y 4.

### D48 — Diseño de la fase 3: componentes, BUILD, ambientes, paquetes y regresión
- **Fecha:** 12 nov 2026
- **Estado:** `[SUPUESTO]` hasta que QA, dev y change management lo revisen. La fase 3 se empezó a pedido, sin que el MVP haya cumplido su criterio de salida (usarse en un CHG real de SyPago).
- **Esquema:** migraciones 00014 (tablas) y 00015 (flujo del PKG, permisos y reglas). **Contrato:** `api/openapi.yaml`, etiquetas `component`, `build`, `environment`, `bundle` y `regression`.
- **Features nuevos y sus tablas** (cada uno escribe solo las suyas):
  - `component`: `component`. El impacto lo guardan `requirement` (`requirement_component`) y `testdesign` (`test_case_component`), que son los dueños del REQ y del TC.
  - `build`: `build`, `build_requirement`. Solo inserción: un BUILD no se edita ni se borra.
  - `environment`: `environment`, `deployment`. El despliegue es de solo inserción.
  - `bundle`: `package`, `package_build`, `package_requirement`. Se llama `bundle` porque `package` es palabra reservada de Go.
  - `regression`: `regression_selection`, `regression_item`.
- **COMP.** Pertenece a un sistema; su nombre no se repite dentro del sistema. No se borra: se retira, y retirado ya no se ofrece para declarar impacto. Un REQ o un TC solo declara componentes de su mismo sistema.
- **BUILD.** Versión, checksum SHA-256 (único: el mismo artefacto no se registra dos veces) y manifiesto: REQ incluidos, migraciones y dependencias como texto, una por línea. `[SUPUESTO]`: el manifiesto es texto libre porque su formato real no está definido.
- **RUN y EXP sobre BUILD y ENV del catálogo.** El alta pide el código del BUILD y del ENV, del mismo sistema que el TC (o el REQ). La versión, el checksum y el nombre del ambiente **se siguen copiando en la fila**: son la evidencia tal como era al ejecutar. Esto se aparta de lo previsto en `docs/modelo-datos.md` (reemplazar esas columnas por llaves foráneas): así las filas anteriores siguen valiendo sin inventarles un BUILD, y un RUN no cambia si alguien retira o renombra un ambiente. Un ambiente de producción no admite RUN (regla de negocio 6: QA no ejecuta en producción).
- **ENV y DEP.** Un ambiente pertenece a un sistema y tiene tipo (limpio, línea base, offline, producción) y sistema operativo (Windows, Linux). Lo instalado se lee de su último despliegue exitoso; no hay campo de versión que mantener a mano. Un despliegue registra un BUILD **o** un PKG. En producción exige `dep.create_production` (change management); en los demás, `dep.create` (QA).
- **PKG.** Flujo de §8.4 con un paso más, `[SUPUESTO]`: «En pruebas → En armado» con motivo, para poder corregir el contenido. El contenido (BUILD y REQ) solo cambia en armado. Un REQ entra si está «Aprobado QA» o «En paquete»; si está «Aprobado QA», la misma acción lo pasa a «En paquete» (y por eso exige también ese permiso). Un REQ puede estar en más de un PKG: el diferido pasa al siguiente.
- **Reglas de bloqueo del PKG** (configurables como las demás):
  - A «En pruebas»: tiene al menos un BUILD y un REQ (`package_has_content`), y la regresión está confirmada (`regression_confirmed`). `[SUPUESTO]`.
  - A «Listo para liberar»: todos los TC de la regresión final tienen su último RUN sobre un BUILD del PKG aprobado (`regression_passed`, `[SUPUESTO]`), y ningún REQ del PKG tiene un DEF de severidad alta abierto (regla de negocio 2).
  - A «Desplegado»: existe un despliegue exitoso del PKG en un ambiente de producción (`deployed_to_production`, `[SUPUESTO]`).
- **Regresión por impacto (regla de negocio 4).** El sistema propone los TC activos que ejercitan algún componente tocado por los REQ del PKG, más los TC críticos del sistema. QA incluye o excluye TC; el criterio es obligatorio cuando la decisión difiere de lo propuesto, y queda con su autor y su fecha. Se guardan las dos cosas (`proposed` e `included`) para compararlas. Volver a proponer conserva los ajustes de QA. Confirmada, la selección no cambia.
- **Permisos nuevos y su reparto inicial:** `comp.manage` (dev, qa), `build.register` (dev), `pkg.build` (dev; quién arma el PKG sigue `[POR DEFINIR]`, regla 10), `pkg.start_testing`, `pkg.reopen`, `pkg.mark_ready`, `pkg.verify`, `regression.manage`, `env.manage`, `dep.create` (qa), `pkg.approve_production`, `pkg.deploy`, `dep.create_production` (change management).
- **Fuera de esta fase:** ingesta de JUnit XML y cobertura (regla 8), aceptación formal de DEF medios o bajos al pasar al siguiente paquete más allá del «Diferido» que ya existe, y todo lo de la fase 4 (checklist de producción, DIAG, SMOKE, informe de conformidad).

### D49 — Fase 3 construida: lo decidido al implementar y cómo se verificó
- **Fecha:** 12 nov 2026
- **Estado:** `[SUPUESTO]` hasta que QA, dev y change management lo revisen. Completa a D48.
- **Componentes.** Un nombre repetido en el sistema es error del campo `name`. Declarar impacto reemplaza la lista; declarar la misma no escribe ni audita. Un componente retirado que ya estaba declarado se conserva y se sigue mostrando, pero no se puede volver a declarar: para editar la lista hay que quitarlo. Un TC retirado sí admite declarar componentes.
- **BUILD.** El checksum repetido dice con qué BUILD choca. Los REQ repetidos del manifiesto cuentan una vez.
- **Ambientes y despliegues.**
  - Un ambiente **no pasa a producción ni deja de serlo** editándolo: si se pudiera, QA podría desplegar en producción con el permiso de los ambientes de prueba. Se retira y se registra otro.
  - Lo instalado es el último despliegue exitoso; uno fallido o revertido posterior no lo cambia.
  - Una **exploración sí se admite en producción**; lo prohibido es el RUN. `[SUPUESTO]`: la regla 6 dice que QA no ejecuta pasos en producción; conviene que QA confirme si observar en producción cuenta.
- **RUN y EXP.** Los errores de existencia (TC, BUILD, ENV) salen juntos y antes que los demás del formulario.
- **Regresión.**
  - Toda escritura bloquea la fila del PKG, también la primera propuesta.
  - Incluir exige que el TC exista, esté activo y sea del sistema; quitar no, para poder quitar un TC que se retiró después de propuesto.
  - Un TC agregado a mano que luego el sistema propone pasa a propuesto y pierde su criterio: la decisión ya coincide con la propuesta.
  - **Se puede confirmar una selección sin ningún TC**, y entonces la regla `regression_passed` se cumple. No se agregó una regla que nadie pidió: QA decide si quiere un mínimo.
  - **Los TC críticos de todo el sistema entran en cada propuesta** (regla de negocio 4); QA quita con criterio los que no apliquen.
  - Auditoría con `entity_type` `regression` sobre el PKG: proponer, cada ajuste (como decisión, con el criterio de motivo), confirmar y reabrir.
- **PKG.**
  - «En armado» es el estado inicial del flujo, no una clave fija; las claves `approved_qa` e `in_package` del REQ sí están fijas en `bundle` y en el front. Si QA renombra esas claves del flujo del REQ, hay que tocar código.
  - Cambiar el contenido, o volver a armado, **quita la confirmación de la regresión**: la selección ya no corresponde.
  - Fuera de armado, tocar el contenido responde `INVALID_ACTION` siempre.
  - Las reglas del PKG no miran la ruta del CHG: un paquete no tiene ruta. (Cambió en la fase 4: D50.)
  - Documentos en PKG y DEP, comentarios en PKG.
- **Matriz y búsqueda.** La matriz tiene columna PKG (también en CSV y XLSX) y filtros por PKG y por COMP. La búsqueda encuentra COMP, BUILD, PKG, ENV y DEP.
- **Front.** Paquetes y BUILD en la sección «Paquetes»; ambientes y despliegues en «Ambientes»; componentes en «Pruebas», porque los mantienen QA y dev y alimentan la regresión. Las listas nuevas no tienen vistas guardadas. Los selectores traen hasta 200 elementos: un sistema con más BUILD, REQ o componentes no los mostraría todos.
- **Fallo corregido en las herramientas.** `tools/smoke-api.sh` fallaba a veces: buscaba en los logs con `grep -q` dentro de una tubería y `docker logs` moría por SIGPIPE.
- **Cómo se verificó.**
  - `make check` completo, con 192 consultas preparadas contra Postgres, el recorrido de un PKG de punta a punta en integración (componentes → impacto → REQ aprobado → BUILD → PKG → regresión → RUN → producción → verificado), 12 confirmaciones simultáneas de las que vale una, y 105 pruebas del front.
  - En el navegador, contra la API real: ficha del PKG con lo que falta para avanzar; transición bloqueada por la regresión sin confirmar (409 real); quitar un TC propuesto exige criterio; confirmar; pasar a pruebas; ambiente con lo instalado; BUILD con su manifiesto; catálogo de componentes; matriz filtrada por PKG; y un RUN registrado sobre un BUILD y un ambiente del catálogo.
- **Lo que no se verificó en el navegador:** alta de BUILD, de ambiente y de componente desde la pantalla; registrar un despliegue; agregar y quitar contenido de un PKG; el resto del flujo del PKG (listo, producción, verificado); impacto en REQ y TC; y las pantallas de administración con el flujo de paquetes. Todo eso está cubierto por pruebas del front con respuestas simuladas y por la integración del backend, no a mano.
- **Sigue pendiente de la fase 3:** ingesta de JUnit XML y cobertura (regla 8); el piloto de regresión en SyPago comparado contra la hoja, que es el criterio de salida y no depende del código.

### D50 — Fase 4: checklist de producción, DIAG, SMOKE, informe de conformidad y calendario
- **Fecha:** 12 nov 2026
- **Estado:** `[SUPUESTO]` hasta que QA, dev, ops y change management lo revisen. Parte de la propuesta de `docs/fase-4.md`; aquí queda lo que se decidió al construirla y lo que cambió respecto de esa propuesta. Las preguntas abiertas de esa guía (§5) siguen sin respuesta: cada supuesto dice a cuál responde.
- **Esquema:** migraciones 00016 (tablas) y 00017 (permisos y reglas por ruta). **Contrato:** `api/openapi.yaml`, etiquetas `production`, `report` y `planning`; 122 de 122 operaciones implementadas.
- **Features y sus tablas** (cada uno escribe solo las suyas). Cambia respecto de la propuesta:
  - `production`: `checklist_template`, `checklist_template_step`, `checklist`, `checklist_signature`, `diagnostic`, `diagnostic_review`, `smoke_test`. La propuesta separaba `checklist` y `verification`; van juntos porque los tres bloquean la misma fila del PKG y dan juntos los hechos de sus reglas. DIAG y SMOKE quedan en tablas separadas, y la revisión del DIAG en otra (`diagnostic_review`), para que **todo sea de solo inserción** sin excepciones: ni la firma ni la revisión hacen `UPDATE`.
  - `report`: `conformance_report`, de solo inserción.
  - `planning`: `milestone`, la única tabla nueva que se edita. Se llama `planning` y no `calendar` porque el calendario es una lectura que junta hitos y ventanas; las ventanas siguen en el CHG y `planning` las pide a un caso de uso de `change` (`ListWindows`), no a su tabla.
- **La ruta del PKG** (cierra el punto abierto de la guía y cambia D49, que decía que un paquete no tiene ruta): es la **más exigente** de las rutas de sus REQ. La ruta de un REQ es la de su CHG o, sin CHG, la ruta por omisión de su sistema. Un PKG sin REQ toma la de su sistema. Se muestra en la ficha del PKG. `[SUPUESTO]`.
- **Reglas de bloqueo por ruta** (sembradas en 00017, editables en Administración):
  - Aprobado para producción → Desplegado: `checklist_signed`, solo en ruta alta.
  - Desplegado → Verificado: `diagnostic_reviewed`, solo en ruta alta; `smoke_passed`, solo en ruta ligera.
  - **Cuenta lo último de cada cosa** `[SUPUESTO]`: el último CHK del PKG con todos sus pasos firmados; el último DIAG, revisado y conforme; el último SMOKE, aprobado. Un DIAG no conforme o un SMOKE fallido se superan entregando otro.
  - **Fix menor no exige ninguna de las tres** `[SUPUESTO]`, aunque se pueden registrar.
- **Checklist (CHK).**
  - **Una plantilla por sistema, versionada** (pregunta 1). Publicar crea la versión siguiente; las anteriores quedan y los CHK abiertos conservan la suya. Ninguna migración siembra pasos: una instalación nueva tiene que publicar la plantilla de cada sistema antes de abrir un CHK. `tools/dev-seed.sh` publica una de ejemplo para SyPago.
  - Un CHK es la ejecución de la última versión de la plantilla para un PKG en un **ambiente de producción** activo del sistema del PKG. Se pueden abrir varios; cuenta el último.
  - **Cualquiera con `chk.sign` firma cualquier paso, en cualquier orden** (pregunta 2). Un paso se firma una vez, con usuario, hora del servidor y comentario opcional; no se desfirma. QA no tiene `chk.sign` (regla de negocio 6).
  - La evidencia de un paso es un DOC adjunto al CHK (`document_link` admite `checklist`, `diagnostic` y `smoke_test`).
- **DIAG y SMOKE.**
  - Se registran sobre un **despliegue del PKG en un ambiente de producción**, cualquiera sea su resultado: un despliegue fallido también se diagnostica.
  - El DIAG lo entregan dev u ops; QA lo **revisa una sola vez** (regla 7): conforme o no conforme, con notas obligatorias si no lo está. Si el resultado cambia, se entrega otro DIAG. La salida real de los diagnósticos de SIMF y SGLPAR sigue sin conocerse (pregunta 3): el DIAG son notas y documentos adjuntos.
  - El SMOKE lo registra QA, aprobado o fallido, con notas obligatorias si falló.
- **Informe de conformidad (RPT).**
  - **Se emite por PKG** (pregunta 4), en cualquier estado del PKG `[SUPUESTO]`: QA decide cuándo lo necesita; el informe dice en qué estado estaba.
  - Emitir **bloquea la fila del PKG** y reúne la evidencia en la misma transacción, pidiéndola a los casos de uso de `bundle`, `traceability`, `regression` y `production`: `report` no repite reglas ni lee sus tablas.
  - Contenido (requerimiento §10, pregunta 5): datos del PKG; contenido (BUILD y REQ); trazabilidad REQ → CHG → RISK → UC → TC → último RUN → DEF; ejecuciones; regresión propuesta y final con los criterios de QA; DEF abiertos y diferidos con quién los difirió, cuándo y por qué; checklists con cada firma; DIAG con su revisión y SMOKE. Sin firma manuscrita.
  - **«Ejecuciones» lista todos los RUN sobre los BUILD del PKG**, también los de TC de REQ que no están en el paquete: el artefacto es el mismo. `[SUPUESTO]`, conviene que QA lo confirme.
  - **El Markdown es el formato canónico.** Se guarda tal como se emitió y su SHA-256 es la huella del informe: se guarda, se muestra en la ficha, viaja en la cabecera `X-Content-SHA256` de las descargas y va en el pie de cada página del PDF. La huella no puede ir dentro del texto que la produce.
  - **Inmutable:** emitir otra vez crea la versión siguiente, que referencia a la anterior (`previous_id`); la anterior se muestra como reemplazada. La matriz muestra en la columna RPT solo el informe vigente de cada PKG (también en CSV y XLSX, que ganan una columna).
  - Fechas del informe en UTC.
- **PDF sin dependencias.** Se escribió `core/pkg/mdpdf` con la biblioteca estándar, como el XLSX, y no con una librería de PDF como recomendaba la guía: el Markdown del informe usa un subconjunto pequeño (títulos, párrafos, listas, tablas, negrita y código), y con eso alcanza un escritor de unas 530 líneas, determinista y probado. Una herramienta por problema (reglas): el repositorio ya resuelve así el XLSX.
  - PDF 1.4, A4, fuentes estándar Helvetica y Helvetica-Bold con sus anchos AFM: no se incrusta ninguna fuente.
  - Encabezado con la marca y el lema de Kaoru y el título del informe; pie con el RPT, su versión, la fecha de emisión, la huella y «Página n de N».
  - Las tablas parten página repitiendo su encabezado, y una columna nunca es más angosta que su palabra más larga, para que un ID no se corte.
  - **Límite conocido:** las fuentes estándar solo cubren WinAnsi (cp1252). Un carácter fuera de ese juego (emoji, alfabetos no latinos) sale como «?» en el PDF; el Markdown lo conserva. `[SUPUESTO]` aceptable para textos en español.
  - El informe **se descarga**; no se ve como PDF dentro de Kaoru, así que Nginx no necesita una excepción a `frame-ancestors`. La ficha muestra el Markdown renderizado.
- **Calendario y alertas.**
  - El calendario junta las ventanas de los CHG vigentes (no finales) y los hitos. Las fases del plan no entran (pregunta 7): si change management las quiere, se cargan como hitos.
  - **Alerta de atraso:** un hito pendiente con la fecha vencida, o una ventana que terminó sin que el CHG esté «Liberado». Se calcula al leer, sin procesos en segundo plano; lo más atrasado primero.
  - Fechas en UTC `[SUPUESTO]`: un hito vence al terminar su día en UTC y los días de atraso de una ventana se redondean hacia arriba. Con la hora de Venezuela (UTC−4) un hito puede salir atrasado cuatro horas antes de la medianoche local.
  - Hitos: título, fecha, responsable opcional, notas y cumplido o pendiente. Sin ID legible. Los mantiene change management (`milestone.manage`); crear y editar se auditan.
  - **Sin correo** (pregunta 6): las alertas solo se ven en Kaoru (Inicio y Calendario).
- **Permisos nuevos y su reparto inicial:** el de la guía (§4.5), sin cambios: `chk.manage_template`, `chk.start`, `chk.sign` (change-manager; `chk.sign` también ops), `diag.submit` (dev, ops), `diag.review`, `smoke.create`, `rpt.issue` (qa, qa-admin), `milestone.manage` (change-manager).
- **Front.**
  - Producción: checklists, diagnósticos, pruebas de humo y plantillas, con la ficha de cada CHK, DIAG y SMOKE (evidencia y actividad). Informes: lista y ficha con el Markdown, la huella, la cadena de versiones y las descargas. Calendario: alertas arriba y una sola lista por fecha con hitos y ventanas; sin vista de mes.
  - La ficha del PKG gana las pestañas «Producción» (con lo que exige su ruta y las acciones según permisos) e «Informes», y su ruta en la cabecera. Inicio toma las ventanas y las alertas del calendario.
  - CHK, DIAG, SMOKE y RPT no admiten comentarios: su actividad es solo la auditoría. La auditoría dice las firmas y las revisiones con palabras («marcó el paso 2 como firmado. Comentario: …»).
  - Ninguna pantalla nueva tiene estilos nuevos: se componen con las clases del boceto.

### D51 — Fase 4: cómo se verificó y qué falta
- **Fecha:** 12 nov 2026
- **Cómo se verificó.**
  - `make check` completo y `make check-offline`. La integración recorre un PKG de SIMF (ruta alta) de punta a punta: plantilla, CHK con firmas, desplegado, DIAG revisado, verificado, inmutabilidad de firmas y revisiones, RPT v1 y v2, exportaciones, búsqueda, documentos enlazados y auditoría; y el calendario con ventanas, hitos y alertas. La ruta ligera del PKG de la fase 3 ahora exige su SMOKE. 128 pruebas del front.
  - `tools/dev-seed.sh` lleva el PKG de SyPago hasta verificado (plantilla, CHK con 3 de 4 firmas y un DOC de evidencia, despliegue en producción, DIAG revisado, un SMOKE fallido y otro aprobado, dos versiones del RPT, tres hitos y una ventana vencida), y `web/mocks/fixtures.json` se volvió a grabar de esos datos.
  - En el navegador, contra la API real: Inicio con próximas ventanas y alertas; pestañas Producción e Informes del PKG; registrar un SMOKE (con la validación de notas); emitir un RPT nuevo y abrir su ficha; descargar el PDF y el Markdown y comprobar que el SHA-256 del Markdown coincide con la huella mostrada; ver el PDF real (encabezado, pie, tablas); ficha del CHK con su actividad; como change management: firmar el último paso, publicar la versión 2 de la plantilla, abrir un CHK nuevo que la toma, y crear, editar (quitar el responsable) y cumplir hitos, con la alerta que desaparece; matriz con la columna RPT; búsqueda de un DIAG.
- **Lo que no se verificó en el navegador:** entregar y revisar un DIAG desde la pantalla (las dos acciones tienen pruebas del front y de integración), adjuntar evidencia desde la ficha del CHK, el tema claro y el ancho de teléfono de las pantallas nuevas.
- **Corregido al probar:** los selectores de despliegue, ambiente y sistema de los formularios de producción no tomaban el valor por omisión porque se pintaban antes de que llegara la lista.
- **Pendiente de la fase 4:**
  - Respuestas a las preguntas 1 a 8 de `docs/fase-4.md` §5; cada supuesto de D50 depende de una.
  - El criterio de salida: un informe emitido para un PKG de ruta alta en una ventana real que defina change management. No depende del código.
  - Correo de alertas, si aparece un SMTP interno.
  - El pie de la aplicación dice «Servidor de datos de ejemplo» en cualquier servidor de desarrollo, también cuando habla con la API real (`tools/dev-front.sh`). Es de una fase anterior y no afecta a producción.

### D52 — Fase 5: métricas de cobertura, tiempo en estado, regresión y defectos escapados
- **Fecha:** 14 nov 2026
- **Estado:** `[SUPUESTO]` hasta que QA y change management lo revisen. Parte de la propuesta de `docs/fase-5.md`; aquí queda lo que se decidió al construirla y lo que cambió respecto de esa propuesta. Las preguntas de esa guía (§5) siguen sin respuesta: cada supuesto dice a cuál responde.
- **Alcance:** solo métricas (requerimiento §10). YouTrack, la importación de la hoja y JUnit XML quedan fuera (pregunta 8).
- **Contrato:** `api/openapi.yaml`, etiqueta `metrics`: `GET /api/metrics/coverage`, `/defect-states`, `/regression` y `/escaped-defects`. 86 rutas y 126 de 126 operaciones implementadas.
- **Esquema:** sin cambios. Ninguna tabla ni índice nuevos.
- **`metrics` es un feature de solo lectura, la segunda excepción de reglas §A.7** (la primera es `traceability`). Cambia respecto de la propuesta, que repartía cada cálculo entre los features dueños de los datos y los juntaba por puertos. Se descartó porque cada métrica es una agregación sobre varias tablas (REQ, UC, TC, RUN, RISK y reglas; DEF y auditoría; PKG, regresión, RUN y BUILD; DEF, PKG, despliegues y ambientes): repartirla obliga a traer filas por puerto y agregarlas en memoria, o a escribir en cada feature una consulta que solo sirve a la pantalla de métricas. `metrics` no tiene tablas ni escribe; lee con una consulta por métrica (no una por fila) y los cálculos (medianas, percentiles, tramos por estado, quién escapó) viven en su dominio con pruebas de tabla. `archtest` lo declara sin tablas propias.
  - **Costo:** la regla de cobertura mínima (`min_coverage`, que evalúa `requirement` en Go) está repetida en el SQL de la métrica. Si la regla cambia, cambian las dos; la prueba de integración la ancla (un REQ de riesgo bajo con 1 de 2 TC aprobados queda bajo el mínimo del 100 %).
- **Cobertura por REQ.**
  - TC requeridos: los TC no retirados enlazados al REQ por sus UC. Aprobados: los que tienen su **último RUN** aprobado (por fecha de ejecución, luego de registro). Porcentaje truncado hacia abajo, para no mostrar 100 % con uno pendiente.
  - **El mínimo** sale de la regla `min_coverage` activa del REQ: la de su nivel de RISK, filtrada por la ruta del REQ como el resto de las reglas; si la regla no dice porcentaje para ese nivel, es 100 %. Sin RISK o sin regla activa no hay mínimo.
  - Cada REQ cae en un resultado: **bajo el mínimo**, **sin TC**, **sin riesgo** (sin RISK evaluado: no es 0 %) o **cumple**. Se ordena por ese orden, después por menor cobertura y por más TC pendientes. El resumen cuenta todo el filtro; el filtro por resultado solo recorta la tabla.
  - Filtros: sistema, CHG, PKG, estado del REQ y resultado. Sin período: la cobertura es la de hoy.
- **Tiempo en estado (histórico de DEF).**
  - La historia sale de `audit_entry` (transiciones de `defect`, con el estado anterior y el nuevo). El índice existente `(entity_type, entity_id, seq)` alcanza: la consulta trae las transiciones de los DEF del filtro por su ID. La propuesta preveía un índice por fecha; no hizo falta (`make check-perf`).
  - Cada DEF se parte en tramos: el primero va de su creación a la primera transición; el último, hasta hoy si sigue abierto. Por estado: DEF que pasaron, **entradas** (un DEF que vuelve a un estado suma otra: así se ven las reaperturas, que no tienen contador propio), mediana y percentil 90 del tiempo **total** de cada DEF en el estado. En los estados finales no se mide la estadía.
  - **De abierto a resuelto:** de la creación a la última transición, para los DEF que hoy están en un estado final.
  - **Días naturales** (pregunta 2), con un decimal. Mediana común; percentil 90 por rango más cercano.
  - Filtros: sistema, severidad y período (DEF **creados** en el período).
  - Inicio no cambia (sigue mostrando el tiempo en el estado actual de los abiertos) y gana un enlace a esta métrica.
- **Tamaño de regresión** (pregunta 3: en TC, el RUN no registra duración).
  - Una fila por PKG con la regresión confirmada: propuestos, finales, agregados y quitados por QA, finales por origen (impacto, crítico, manual), finales con su último RUN sobre un BUILD del PKG aprobado, y los TC activos del sistema hoy para comparar.
  - Resumen: PKG, totales y la mediana de TC finales por PKG. Filtros: sistema y período (fecha de confirmación).
- **Defecto escapado** (pregunta 1): **calculado**, no marcado a mano. Un DEF escapó si se creó después del primer despliegue **exitoso**, en un ambiente de `kind = 'production'`, de un PKG que contiene su REQ. Vale para los DEF ya registrados. Tasa: escapados / DEF creados en el período, con un decimal. Se muestra por mes y la lista de escapados con su REQ, el PKG que lo llevó a producción, esa fecha y los días que pasaron.
- **Período** (pregunta 4): por omisión, desde el primer día del mes de hace 11 meses (UTC), es decir, los últimos 12 meses calendario con el actual, sin fecha final. `from` y `to` son días en UTC y `to` se incluye. `to` antes de `from` es error de validación del campo `to`.
- **Sin metas ni SLA** (pregunta 5, regla 3): nada se pinta en rojo por un umbral. La única comparación es la cobertura frente al mínimo que ya fija la regla de bloqueo del REQ.
- **Lo ve todo el mundo** (pregunta 6): leer no exige permiso. Sin permisos ni auditoría nuevos.
- **Cambia respecto de la propuesta, además:**
  - **Sin exportación CSV** en esta entrega: ninguna pregunta la pide y la tabla se puede copiar. Se agrega si change management o QA la piden.
  - **Sin vistas guardadas** en métricas: los filtros quedan en la URL y se comparte el enlace.
- **Front.** Sección «Métricas» en la barra lateral con cuatro pantallas (Cobertura, Tiempo en estado, Regresión y Defectos escapados), filtros en la URL, `kpis` y tablas. Las barras son el `meter` que ya existía, siempre con su número escrito. Sin librería de gráficos ni estilos nuevos.
- **Semilla:** `tools/dev-seed.sh` registra un RUN fallido de TC-4 y abre un DEF sobre REQ-2 después del despliegue en producción del PKG de SyPago: es el DEF escapado del laboratorio, y deja a REQ-2 bajo el mínimo.

### D53 — Fase 5: cómo se verificó y qué falta
- **Fecha:** 14 nov 2026
- **Cómo se verificó.**
  - Pruebas de tabla del dominio (tramos por estado con reaperturas y estados desconocidos, mediana y percentil, resultado y orden de la cobertura, escapados por mes y período); casos de uso con dobles escritos a mano; integración contra Postgres (`metrics_integration_test.go`) que arma con los casos de uso de los otros features un CHG con REQ, RISK y RUN, transiciones de un DEF, una regresión confirmada y despliegues en producción (uno fallido, que no cuenta) y comprueba cada métrica y sus validaciones.
  - `make check-perf` con volumen: cobertura 139 ms, cobertura filtrada por resultado 121 ms, tiempo en estado 7 ms, regresión 1 ms y escapados 6 ms (umbral 200 ms).
  - `make check` y `make check-offline`. 136 pruebas del front.
  - En el navegador, contra la API real y los datos de `tools/dev-seed.sh`: las cuatro pantallas; el filtro por resultado de la cobertura; el filtro por severidad y período de los escapados; el error de un período al revés; el enlace desde Inicio.
- **Lo que no se verificó en el navegador:** los filtros por CHG, PKG y estado de la cobertura, el tema claro y el ancho de teléfono. Los filtros tienen pruebas del front que comprueban que viajan en la consulta.
- **Límite de la demostración:** la semilla ocurre en segundos y la auditoría no se puede fechar hacia atrás, así que los tiempos en estado del laboratorio son todos «< 0,1 d». Las pruebas del front usan datos armados para los valores grandes.
- **Pendiente de la fase 5:**
  - Respuestas a las preguntas 1 a 8 de `docs/fase-5.md` §5; sobre todo la definición de DEF escapado.
  - El criterio de salida: change management y QA ven las métricas con los datos del laboratorio y aprueban las definiciones. No depende del código.
  - Exportación CSV, si se pide.
  - La deuda heredada que lista `docs/fase-5.md` §6 sigue igual.

### D54 — Fase 6: planificación de actividades por operador, solapes y período de certificación
- **Fecha:** 14 nov 2026
- **Estado:** `[SUPUESTO]` hasta que el coordinador de QA y change management lo revisen. Parte de `docs/fase-6.md`. Las tres decisiones del coordinador de esa guía (§1) sí están tomadas: el proyecto es el CHG, solape es compartir al menos un día y la regla solo avisa. Las preguntas 1 a 4 de su §4 siguen abiertas; cada supuesto dice a cuál responde.
- **Contrato:** `api/openapi.yaml`, etiqueta `planning`: `GET` y `POST /api/activities`, `GET` y `PATCH /api/activities/{code}`, `GET /api/planning/schedule`. 89 rutas y 131 de 131 operaciones implementadas.
- **Esquema:** migración `00018`. Tablas `activity` y `activity_operator`, dueño `planning` (ver `modelo-datos.md`). Prefijo `ACT` en `code_sequence`. `blocking_rule.rule_key` admite `certification_planned`. Permiso `activity.manage` para `qa-admin` y `change-manager`.
- **Actividad.**
  - Clase (`certification`, `regression`, `production`, `other`; pregunta 1), título, notas, CHG opcional (obligatorio en la certificación, por `CHECK` y en el dominio), fecha de inicio y de fin en días UTC, las dos incluidas, y cero o más operadores.
  - **Largo máximo de un año** (366 días incluidos). No lo pedía la guía: evita que una fecha mal escrita (2062 en vez de 2026) deje a un operador ocupado décadas y ensucie todos los solapes.
  - **El estado no se guarda:** sale de las fechas al leer (*planificada*, *en curso*, *terminada*); *cancelada* es la única marca guardada (`cancelled_at`). Cancelar no borra y se puede revertir. Editar los operadores reemplaza la lista. `kaoru_app` tiene sobre las dos tablas los permisos por omisión, como `milestone`: la base no impide borrar una actividad, solo la aplicación (no hay ruta para hacerlo).
  - **Operadores:** usuarios que ya entraron una vez (`app_user`, D34). Uno que no existe es error del campo `operators`.
  - **Auditoría:** el alta registra las fechas; cada campo editado, su valor anterior y nuevo; los operadores, como la lista de ids anterior y la nueva (la ficha los muestra con su nombre).
- **Solape** (decisión del coordinador). Dos actividades no canceladas con un operador en común que comparten al menos un día. Se informa por operador, con las dos actividades y los días que se cruzan. **No bloquea:** guardar responde 201 o 200 con los solapes que produce, y la ficha y la planificación los muestran. Lo calcula el dominio, con pruebas de tabla; no hay tabla de solapes.
- **Planificación de un rango** (`GET /api/planning/schedule`). Por omisión, seis semanas desde el lunes de la semana actual; con `from` o `to` solo, seis semanas desde o hasta esa fecha; como máximo 92 días (`to` es el campo del error). Trae: por operador, sus actividades del rango y sus solapes; las actividades sin operador; los CHG con certificación en el rango; las ventanas de cambio de los CHG vigentes que tocan el rango (por el caso de uso de `change`, como el calendario); y los hitos con fecha en el rango. El filtro por operador recorta las filas de operadores; los proyectos, las ventanas y los hitos se siguen viendo enteros.
- **Regla `certification_planned`** (decisión del coordinador). En la transición del REQ *Entregado a QA → En pruebas*, en «solo avisa» (`blocks = false`), editable en *Administración → Reglas de bloqueo*. Se cumple si el CHG del REQ tiene una certificación no cancelada, con operadores, que cubre hoy (UTC). Si no, dice cuál de las tres cosas falta; cuando hoy está fuera, cita el período con operadores más cercano. Un REQ sin CHG la cumple. `requirement` lee el estado por un puerto conectado en `bootstrap` (D36).
- **La guía no valida el período contra la ventana del CHG** (pregunta 2): se ven juntos en la planificación, uno al lado del otro.
- **Quién planifica** (pregunta 3): `qa-admin` y `change-manager`. **Quién ve** (pregunta 4): todos, como el resto de Kaoru.
- **Front.**
  - La sección *Calendario* pasa a *Planificación*: *Por operador* (`/planificacion`), *Actividades* (`/planificacion/actividades`) y *Calendario*, que conserva su ruta `/calendario` de la fase 4.
  - **Por operador:** grilla CSS, una columna por día y una fila por operador, con desplazamiento horizontal. Las actividades que se pisan van en carriles distintos de la misma fila. Cada barra lleva su código escrito y, para lectores de pantalla y al pasar el ratón, título, clase y fechas. Un solape se ve de tres maneras, para que el color no sea la única señal: la franja rayada detrás de los días que se cruzan con el ícono de aviso y el número de actividades arriba, el chip *N solapes* junto al nombre del operador y la lista de solapes en palabras sobre la grilla. Debajo, los CHG del rango con su certificación y su ventana: un CHG con ventana y sin certificación queda marcado *Sin certificación*. El rango (desde, 2, 6 o 12 semanas) y el operador viven en la URL.
  - **Ficha del CHG:** sección *Certificación* con el mismo criterio que la regla y *Planificar certificación* con la clase y el CHG puestos. La ficha pasa a `web/src/app/ChangePage.tsx`, que envuelve la de `changes` (un feature no importa a otro).
  - **Inicio:** *Tus actividades* (las que terminan hoy o después) y, con `activity.manage`, el número de solapes de las próximas seis semanas.
  - Sin librería de gráficos ni de fechas nueva.
- **Sin vistas guardadas** en *Actividades*: las vistas guardadas solo admiten las pantallas que lista `savedview` y agregar una es un cambio del dominio de ese feature. Los filtros quedan en la URL.
- **Fuera de esta entrega** (guía §3): horas o porcentaje de dedicación, feriados y días hábiles, correo, arrastrar barras, la regla en el PKG y fechas planificadas en REQ, PKG o DEF.
- **Semilla:** `tools/dev-seed.sh` planifica la certificación del CHG de SyPago desde hace dos días hasta dentro de ocho, con Ana Rojas y Luis Paz; una regresión de SIMF de Luis que se le cruza tres días; y el pase a producción del CHG de SIMF sin operador.

### D55 — Fase 6: cómo se verificó y qué falta
- **Fecha:** 14 nov 2026
- **Cómo se verificó.**
  - Pruebas de tabla del dominio (validación, operadores, estado por fechas, edición y cancelación, solapes, criterio de la certificación, rango de la planificación); casos de uso con dobles escritos a mano; integración contra Postgres (`planning_integration_test.go`): el aviso de la regla en sus tres variantes y sin aviso con todo en orden, permisos, CHG obligatorio, operador inexistente, solape de dos días, planificación filtrada por operador, rango al revés, cancelar que quita el solape, filtros de la lista y auditoría.
  - `make check-perf` con 30 operadores y 3 000 actividades: planificación de seis semanas 6 ms, de 92 días 8 ms, actividades de un operador 2 ms, ficha 1 ms (umbral 200 ms).
  - `tools/e2e-compose.sh`: QA no planifica (403), change management crea, la planificación la lee QA por el perímetro y más de 92 días es 422. `tools/e2e-front.sh` entrega `/planificacion` y `/planificacion/actividades`.
  - `make check` y `make check-offline`. 149 pruebas del front.
  - En el navegador, contra la API real (`make demo` y `tools/dev-front.sh` como `lab-admin`): la línea de tiempo con el solape de Luis en la lista, en la franja con su ícono y en el chip; el CHG de SIMF marcado *Sin certificación*; abrir una actividad desde su barra; reajustar sus fechas y ver desaparecer el solape, con la auditoría; planificar la certificación desde la ficha del CHG (estado *Certifica del … al …*), una segunda ronda y una tercera que se cruza, con el aviso en la misma ficha; cancelarla; la lista filtrada por *Cancelada*; *Tus actividades* y el aviso de solapes en Inicio; la regla con su nombre en *Reglas de bloqueo*; el tema claro.
- **Lo que no se verificó en el navegador:** el aviso de la regla al pasar un REQ a pruebas (ningún REQ de la semilla está en *Entregado a QA*; lo cubre la integración) y el filtro por operador y el cambio de largo de la planificación (los cubren las pruebas del front).
- **Corregido al probar:** planificar desde la ficha del CHG no mostraba el solape que producía la actividad nueva, porque se queda en la ficha en vez de ir a la de la actividad; ahora lo avisa ahí. La cuenta de reglas del REQ en `api_integration_test.go` pasó de 6 a 7.
- **Pendiente de la fase 6:**
  - Respuestas a las preguntas 1 a 4 de `docs/fase-6.md` §4.
  - El criterio de salida: el coordinador planifica una certificación real con sus operadores y la revisa con change management. No depende del código.
  - **El ancho de teléfono no funciona en ninguna pantalla**, tampoco en las de esta fase: la barra lateral ocupa 232 px fijos y el contenido se desborda. Viene del armazón de la aplicación (fase 2); la línea de tiempo ya se desplaza sola, pero hace falta que el armazón se adapte.
  - Vistas guardadas en *Actividades*, si se piden (cambio en `savedview`).
  - Lo que la guía dejó fuera (§3).

### D56 — Marca: sello con la K y かおる como furigana
- **Fecha:** 14 nov 2026
- **Decisión:** el símbolo pasa a ser un sello índigo (el token `accent`) con la K de línea continua en blanco (`on-accent`). Sobre el wordmark *kaoru* va かおる, su lectura en hiragana, pequeña y en `accent-text`, como furigana. Se descartó el kanji 馨: se mantiene la K y la referencia japonesa va en kana.
- **Por qué en trazos:** かおる está dibujada a mano como trazos SVG del mismo grosor y remate que la K (`KANA` en `tools/boceto/js/boceto.js`, `web/src/shared/ui/Kana.tsx`), no como texto: Kaoru se instala sin conexión y no puede contar con una fuente japonesa en el equipo.
- **Dónde:** barra lateral y *Acerca de* (boceto y front), favicon (`tools/boceto/img/kaoru.svg`, copiado al tema de Keycloak) y `favicon.ico` del login, que `tools/make-favicon.py` dibuja ahora como sello. El login de Keycloak muestra el sello por `kaoru.svg`; la kana no se agregó allí.
- **Verificación:** `npm run check` (la prueba de íconos compara también el `rect` del sello con el boceto), `make check-generated` y el navegador en tema oscuro y claro. El tema de Keycloak no se miró en el navegador.

### D57 — Servidor de pruebas: contabo con Ansible, detrás del nginx que ya tiene
- **Fecha:** 16 nov 2026
- **Decisión:** Kaoru de prueba corre en `https://kaoru.aldia.services`, en un servidor que ya atiende otras aplicaciones. Lo despliega `infra/contabo/deploy.yml` (Ansible) con el mismo `install.sh` del paquete offline. El nginx del servidor termina el TLS público (Let's Encrypt, con su certbot) y pasa todo al Nginx de Kaoru en `127.0.0.1:8445`.
- **Por qué así:** el servidor tiene internet, así que no hace falta el paquete offline completo: viajan solo la imagen de `kaoru-api` (construida aquí para `linux/amd64`) y la configuración; las demás imágenes se bajan por digest. El playbook comprueba puertos, proyecto de Compose y dominio antes de tocar nada, y solo recarga el nginx del servidor si `nginx -t` pasa.
- **DNS:** `aldia.services` ya está en Cloudflare (plan gratuito); `kaoru` es un registro A con proxy. Como el nombre resuelve a Cloudflare y no al servidor, el playbook comprueba que el dominio llega al servidor pidiendo un archivo por la ruta del desafío de Let's Encrypt, en lugar de comparar IP.
- **Fuera:** vista previa de Office (memoria compartida), IP real del usuario en Kaoru y cambio automático de dominio (docs/despliegue-contabo.md, «Límites conocidos»).
