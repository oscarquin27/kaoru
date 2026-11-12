# Fase 4 — Producción y conformidad: encargo para quien la construya

Este documento es el punto de partida de la fase 4. Está escrito para una sesión que empieza sin contexto: dice qué pide el requerimiento, en qué estado está el repositorio, cómo se trabaja aquí, un diseño propuesto y qué falta decidir. Lo propuesto va marcado `[SUPUESTO]`; no es una decisión hasta que QA, dev y change management la revisen.

- **Requerimiento completo:** [kaoru-requerimiento-v2.md](kaoru-requerimiento-v2.md) (§6, §8.2, §9 reglas 6, 7 y 11, §10, §12).
- **Reglas del repositorio:** [reglas.md](reglas.md). Se cumplen todas; `src/internal/archtest` verifica las de arquitectura.
- **Decisiones tomadas:** [decisiones.md](decisiones.md) (D1 a D49). Las de la fase 3 son D48 y D49.

## 1. Qué pide la fase 4

| Entregable | Qué es (requerimiento) |
|---|---|
| **CHK-** Checklist de producción | Versionado, con firma por paso (usuario autenticado + fecha + comentario) y evidencia adjunta. Lo ejecuta change management; cada ejecutor firma su paso. QA no ejecuta pasos en producción (regla 6). Obligatorio en la ruta alta (§8.2). |
| **DIAG-** Diagnóstico | Resultado de diagnóstico posterior al despliegue (SIMF, SGLPAR). Lo entregan dev u ops; QA lo revisa y marca el resultado (regla 7). Obligatorio en la ruta alta. |
| **SMOKE-** Prueba de humo | Resultado de la prueba de humo posterior al despliegue (SyPago). Lo registra QA (regla 7). Obligatorio en la ruta ligera. |
| **RPT-** Informe de conformidad | Exportable en PDF y Markdown: trazabilidad, ejecuciones, regresión, defectos abiertos y diferidos, firmas del checklist y hash del documento. Generación 100 % offline. Inmutable una vez emitido; si algo cambia, se emite una versión nueva que referencia la anterior (regla 11). Lleva la marca de Kaoru en encabezado y pie (§7.0). |
| **Calendario y alertas** | Hitos, ventanas de cambio y fases, con alertas de atraso dentro de la aplicación; correo opcional si hay SMTP interno (pregunta 13, sin responder). |
| **Inicio (§7.7)** | Lo que falta del tablero: próximas ventanas de cambio y alertas de atraso de hitos. |
| **Matriz (§7.6)** | Columna RPT, que cierra la cadena REQ → … → PKG → RPT. |

**Criterio de salida:** informe generado para un CHG de ruta alta, en la ventana que defina change management. Eso no depende solo del código.

**Fuera de esta fase:** integración con YouTrack y métricas avanzadas (fase 5).

## 2. Estado del repositorio

Todo está en `main`. `make check` pasa entero.

- **Backend** (`src/`, Go 1.26, módulo `kaoru`): gin, pgx + sqlc, goose. Un feature por carpeta en `src/internal/feature/`: `workflow`, `audit`, `user`, `change`, `requirement`, `risk`, `testdesign`, `execution`, `defect`, `document`, `comment`, `traceability`, `savedview`, `component`, `build`, `environment`, `bundle` (PKG), `regression`.
- **Base:** migraciones `src/migrations/00001` a `00015`. Dos roles: `kaoru_owner` migra, `kaoru_app` ejecuta (sin `UPDATE` ni `DELETE` en las tablas de solo inserción).
- **Contrato:** `api/openapi.yaml` (67 rutas, 101 operaciones). Una prueba exige que las rutas registradas y las del contrato coincidan.
- **Front** (`web/`): React 19 + TypeScript + Vite, TanStack Query, sin librería de componentes. Las secciones «Producción», «Informes» y «Calendario» ya están en la barra lateral, **vacías** (`web/src/app/sections.tsx`).
- **Despliegue** (`infra/`): Compose con Nginx, oauth2-proxy, Keycloak, Postgres y la API; servicio opcional `libreoffice` para previsualizar Office.

Lo que la fase 4 encuentra hecho y debe reutilizar:

- **Flujos configurables con reglas de bloqueo** (`workflow`). Cada feature dueño de una entidad evalúa sus reglas en el dominio: ver `requirement/domain/rules.go` y `bundle/domain/rules.go`.
- **El PKG ya tiene su flujo**: `En armado → En pruebas → Listo para liberar → Aprobado para producción → Desplegado → Verificado`. La transición `Desplegado → Verificado` hoy no tiene regla: la fase 4 le pone las suyas.
- **Despliegues** (`environment`): un DEP registra un BUILD o un PKG en un ambiente; en producción exige `dep.create_production`.
- **Ruta del CHG** (`high`, `light`, `minor_fix`) en `change.route`, y **ventana** en `change.window_start` y `change.window_end`.
- **Documentos versionados con checksum** (`document`), enlazables a una entidad por tipo y código.
- **Auditoría** de solo inserción, encadenada por hash (`audit`).
- **Exportación sin librerías**: CSV y XLSX de la matriz en `traceability/delivery/api/export.go`.

## 3. Cómo se trabaja aquí

Lo que sigue no es opcional; es lo que hace que `make check` siga pasando.

- **Arquitectura por feature:** `domain`, `application/port`, `application/usecase` (un archivo por caso de uso), `adapter/<tecnología>`, `delivery/api`, `module.go`. Un feature escribe solo sus tablas. Para pedirle algo a otro feature: puerto + adaptador que llama a sus casos de uso. Lecturas SQL de tablas ajenas, solo en el sentido de la llave foránea. Si aparece un ciclo, se resuelve con enlace tardío en `bootstrap` (decisiones D30 y D36).
- **Sin `time.Now` ni `log/slog`** en dominio ni casos de uso: reloj inyectado, y el log solo en `bootstrap`.
- **Toda escritura** comprueba el permiso, bloquea la fila de la entidad (`SELECT … FOR UPDATE`) al empezar la transacción y audita dentro de ella. Si la auditoría falla, no se guarda nada.
- **Errores:** `axis` (códigos) y `problem` (RFC 9457). Los de validación van por campo y el front los pinta bajo cada campo.
- **Orden de trabajo que funcionó en la fase 3:** primero la migración y el contrato (son la especificación), después cada feature con sus pruebas, después el front, y al final probar en el navegador contra la API real.
- **Pruebas obligatorias** (reglas §A.17): dominio; casos de uso con dobles de puertos escritos a mano (cada permiso negado, cada error de negocio, camino feliz con lo guardado y lo auditado, «nada se guarda si la auditoría falla»); e integración contra Postgres real en `src/internal/bootstrap/*_integration_test.go`.
- **Al agregar un feature** hay que tocar: `src/sqlc.yaml` (un bloque), `src/internal/archtest/arch_test.go` (sus tablas), `src/internal/bootstrap/router.go` (módulo y rutas) y, si tiene ID legible, `src/internal/shared/domain/code.go` (prefijo).
- **Decisiones y supuestos** se registran en `docs/decisiones.md` con el formato de las entradas existentes. No se inventan datos de negocio: lo que falta va como `[POR DEFINIR]`.
- **Commits** en español, en imperativo, en una rama `feature/…`, y se fusionan en `main` con `--no-ff`.

### Comandos

| Para | Comando |
|---|---|
| Todo lo que corre la CI | `make check` |
| Solo Go (vet, lint, pruebas unitarias) | `make check-go` |
| Regenerar sqlc tras tocar consultas | `make generate` |
| Integración contra Postgres real | `make check-integration` |
| Paquete offline con respaldo y restauración | `make check-offline` |
| Levantar el entorno de desarrollo | `KAORU_WEB_DIR="$PWD/web/dist" tools/dev-up.sh` |
| Datos de ejemplo | `tools/dev-seed.sh` |
| Regrabar las respuestas de ejemplo del front | `tools/dev-capture.sh` |
| Front contra la API real, en el navegador | `tools/dev-front.sh [usuario]` (http://localhost:5173) |
| Apagar el entorno (con `-v` borra los datos) | `tools/dev-down.sh` |

### Tropiezos conocidos

- **Las pruebas de despliegue de `make check` usan los puertos del entorno de desarrollo.** Hay que apagarlo antes con `tools/dev-down.sh`.
- **`web/mocks/fixtures.json` son respuestas reales grabadas.** No se editan a mano. Si se siembra dos veces, los ID cambian y varias pruebas del front fallan: `tools/dev-down.sh -v`, levantar, sembrar una vez y regrabar.
- **El navegador de la sesión de trabajo no confía en el certificado de laboratorio** de `https://localhost:8443`. Para probar el front contra la API real se usa `tools/dev-front.sh`, que no necesita el certificado.
- **Nginx prohíbe incrustar respuestas** (`frame-ancestors 'none'`) salvo el contenido y la vista previa de documentos. Si el informe en PDF se va a ver dentro de la aplicación, su ruta necesita la misma excepción en `infra/nginx/kaoru.conf`.
- **Las claves de regla de bloqueo están en un `CHECK`** de la tabla `blocking_rule`: una regla nueva exige cambiarlo en una migración (ver cómo lo hace la 00014).
- **Los tipos de entidad de documentos y comentarios también están en `CHECK`**, y además en los adaptadores `document/adapter/entities` y `comment/adapter/entities`.

## 4. Diseño propuesto `[SUPUESTO]`

Es una propuesta para arrancar, hecha con el mismo criterio que la fase 3. Quien construya la fase puede cambiarla; lo que cambie se registra en `docs/decisiones.md`.

### 4.1 Features y tablas

| Feature | Tablas | Notas |
|---|---|---|
| `checklist` | `checklist_template`, `checklist_template_step`, `checklist`, `checklist_step` | La plantilla es el checklist «versionado»: se edita creando una versión nueva y la anterior queda. Un CHK- es una ejecución de una versión de la plantilla para un PKG y un ambiente de producción. Cada paso se firma una vez (usuario, fecha, comentario) y no se desfirma: `checklist_step` de solo inserción para la firma, o con `REVOKE UPDATE` tras firmar. La evidencia de un paso es un DOC enlazado al CHK. |
| `verification` | `diagnostic`, `smoke_test` | DIAG y SMOKE son casi iguales: código, PKG, despliegue al que se refieren, resultado, notas, quién y cuándo. El DIAG tiene dos momentos (quién lo entrega y quién lo revisa); el SMOKE, uno. Puede ser una sola tabla con tipo, si simplifica. De solo inserción, salvo la revisión del DIAG. |
| `report` | `conformance_report` | RPT- con su versión, el RPT anterior al que reemplaza, el CHG o PKG al que pertenece, el contenido en Markdown tal como se emitió, su SHA-256, quién lo emitió y cuándo. De solo inserción. |
| `calendar` | `milestone` | Hitos con fecha, responsable y estado (pendiente, cumplido). Las ventanas de cambio se leen del CHG; no se duplican. |

Prefijos nuevos: `CHK`, `DIAG`, `SMOKE`, `RPT`. Los hitos no necesitan ID legible.

### 4.2 Reglas de bloqueo nuevas

Sobre el flujo del PKG, configurables como las demás:

| Transición | Regla | Qué exige |
|---|---|---|
| Aprobado para producción → Desplegado | `checklist_signed` | El CHK del PKG tiene todos sus pasos firmados. Solo en la ruta alta. |
| Desplegado → Verificado | `diagnostic_reviewed` | Hay un DIAG del PKG revisado por QA con resultado conforme. Solo en la ruta alta. |
| Desplegado → Verificado | `smoke_passed` | Hay un SMOKE del PKG aprobado. Solo en la ruta ligera. |

**Punto abierto:** un PKG no tiene ruta (D49: sus reglas ignoran `route`). Para aplicar reglas «solo en la ruta alta» hay que definir la ruta del paquete. Propuesta: la más exigente de las rutas de los CHG de sus REQ.

### 4.3 Informe de conformidad

- **Qué contiene** (requerimiento §10): trazabilidad (REQ → RISK → UC → TC → RUN → DEF → PKG), ejecuciones, regresión propuesta y final con sus criterios, defectos abiertos y diferidos con quién aceptó diferirlos, firmas del checklist, DIAG o SMOKE, y el hash del documento.
- **De qué se emite:** de un PKG `[SUPUESTO]`. El criterio de salida habla de «un CHG de ruta alta», pero lo que se despliega y se verifica es el paquete. Hay que confirmarlo con change management.
- **Cómo se arma:** `report` no repite reglas; le pide los datos a `traceability`, `regression`, `bundle`, `checklist` y `verification`, como hace `traceability` con la ficha del REQ.
- **Markdown** es el formato canónico: se guarda tal como se emitió y su SHA-256 es el hash del informe.
- **PDF 100 % offline:** es la decisión técnica más delicada de la fase. Opciones:
  1. Escribir el PDF con la biblioteca estándar, como se hizo con el XLSX. Sin dependencias, pero un PDF con tablas y paginación es bastante más trabajo.
  2. Una librería de PDF en Go. Suma una dependencia, que hay que justificar en `docs/decisiones.md` (reglas: «una herramienta por problema»).
  3. Convertir con el servicio `libreoffice` (Gotenberg). Ya existe, pero es opcional y pesa 2,7 GB: el informe no puede depender de un servicio opcional.

  Recomendación: la opción 2, con una librería pequeña y sin CGO (la imagen de la API es `scratch`).
- **Inmutable:** emitir es insertar. Una versión nueva referencia la anterior; nada se actualiza ni se borra.
- **Ver el PDF dentro de Kaoru** exige la excepción de Nginx mencionada arriba.

### 4.4 Calendario y alertas

- El calendario junta tres cosas: las ventanas de los CHG (ya existen), los hitos (tabla nueva) y las fases del plan. Las fases están en el requerimiento §12 con fechas supuestas: `[POR DEFINIR]` si se cargan como hitos o quedan fuera.
- **Alerta de atraso:** un hito pendiente con la fecha vencida, o un CHG cuya ventana terminó sin estar liberado. Se calcula al leer; no hace falta un proceso en segundo plano.
- **Correo:** no se hace hasta saber si hay SMTP interno (pregunta 13).
- **Inicio:** agrega «próximas ventanas de cambio» y «alertas de atraso».

### 4.5 Permisos nuevos y reparto inicial

Según la tabla de roles del requerimiento (§4):

| Permiso | Rol inicial |
|---|---|
| `chk.manage_template` (editar la plantilla del checklist) | `change-manager` |
| `chk.start` (abrir el checklist de un PKG) | `change-manager` |
| `chk.sign` (firmar un paso) | `change-manager`, `ops` |
| `diag.submit` (entregar un DIAG) | `dev`, `ops` |
| `diag.review` (revisar un DIAG) | `qa`, `qa-admin` |
| `smoke.create` (registrar un SMOKE) | `qa`, `qa-admin` |
| `rpt.issue` (emitir un informe) | `qa`, `qa-admin` |
| `milestone.manage` (hitos) | `change-manager` |

### 4.6 Front

- **Producción:** checklists en curso y su ficha (pasos, quién firmó cada uno, evidencia, botón «Firmar» con comentario), más DIAG y SMOKE por PKG.
- **Informes:** lista de RPT y su ficha con el Markdown renderizado (ya hay un renderizador propio en `web/src/features/documents/Markdown.tsx`), descarga en PDF y Markdown, el hash y la cadena de versiones.
- **Calendario:** hitos y ventanas en una lista por fecha con las alertas arriba. Una vista de mes es opcional.
- **Ficha del PKG:** pestañas o tarjetas para su checklist, sus DIAG o SMOKE y sus informes.
- El boceto (`tools/boceto`) no dibuja ninguna de estas pantallas: se componen con las clases que ya existen, sin estilo nuevo.

## 5. Lo que falta decidir antes o durante

Ninguna de estas preguntas está respondida. Se puede avanzar con el supuesto indicado, marcándolo.

| # | Pregunta | A quién | Supuesto para avanzar |
|---|---|---|---|
| 1 | ¿El checklist es uno por sistema, uno por ruta, o uno solo? ¿Qué pasos tiene hoy? | Change management | Una plantilla por sistema, con pasos de ejemplo marcados como tales. |
| 2 | ¿Quién firma cada paso: cualquiera con el permiso, o cada paso tiene un rol? | Change management | Cualquiera con `chk.sign`. |
| 3 | ¿Qué salidas tienen hoy los diagnósticos de SIMF y SGLPAR y en qué formato? (pregunta 4 del requerimiento) | Dev y ops | Resultado conforme o no conforme, notas y un documento adjunto. |
| 4 | ¿El informe se emite por PKG o por CHG? | Change management y QA | Por PKG. |
| 5 | ¿Qué debe decir exactamente el informe y quién lo firma? | QA | Las secciones del requerimiento §10, sin firma manuscrita. |
| 6 | ¿Hay SMTP interno? (pregunta 13) | Infraestructura | No hay correo. |
| 7 | ¿Las fases y sus fechas van al calendario? | Change management | No; solo hitos que alguien cargue y las ventanas de los CHG. |
| 8 | Fechas de las próximas ventanas de cambio (pregunta 6) | Change management | Se cargan en cada CHG cuando existan. |

## 6. Deuda que la fase 4 hereda

No es parte de la fase, pero conviene saberlo:

- **Importación de la hoja** (MVP): sin hacer; espera el formato de la hoja (pregunta 5).
- **Ingesta de JUnit XML y cobertura** (fase 3, regla 8): sin hacer.
- **Visor de OpenAPI empaquetado, PDF.js empaquetado y resaltado de logs** (MVP): sin hacer. Hoy los PDF se ven con el visor de Chrome.
- **Pantallas de la fase 3 que nadie probó a mano**: alta de BUILD, de ambiente y de componente; registrar un despliegue; contenido del PKG; impacto en REQ y TC. Están cubiertas por pruebas con respuestas simuladas y por la integración del backend (D49).
- **Nunca se verificó:** login y logout en un navegador real, la instalación en un servidor sin red, y el rendimiento en el servidor objetivo.
- **El criterio de salida del MVP y el de la fase 3 no se cumplieron**: dependen de usar el sistema en un CHG real de SyPago y de un piloto de regresión. Las fases 3 y 4 se construyen a pedido, por delante de eso.

## 7. Qué significa «terminado»

Del requerimiento (§12): pruebas automatizadas en verde, migraciones aplicadas sobre una base limpia, `docker compose up` probado a partir del paquete offline, documentación actualizada y demostración a quien firma el criterio de salida.

En este repositorio eso es:

1. `make check` y `make check-offline` pasan.
2. Lo nuevo se probó en el navegador contra la API real, y lo que no, se dice.
3. `docs/decisiones.md` tiene las decisiones y los supuestos; `docs/modelo-datos.md`, `docs/flujos.md` y `docs/manual-usuario.md` están al día.
4. La entrega separa Decisiones, Supuestos y Pendientes (requerimiento §13).
