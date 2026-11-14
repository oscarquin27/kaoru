# Fase 5 — Métricas: encargo para quien la construya

Este documento es el punto de partida de la fase 5. Está escrito para una sesión que empieza sin contexto: dice qué pide el requerimiento, en qué estado está el repositorio, cómo se trabaja aquí, un diseño propuesto y qué falta decidir. Lo propuesto va marcado `[SUPUESTO]`; no es una decisión hasta que QA y change management lo revisen.

- **Requerimiento completo:** [kaoru-requerimiento-v2.md](kaoru-requerimiento-v2.md) (§7.7, §8.3, §9 reglas 3 y 4, §10, §12).
- **Reglas del repositorio:** [reglas.md](reglas.md). Se cumplen todas; `src/internal/archtest` verifica las de arquitectura.
- **Decisiones tomadas:** [decisiones.md](decisiones.md) (D1 a D51). Las de la fase 4 son D50 y D51.

## 1. Qué pide la fase 5

El requerimiento (§12) define la fase 5, «Ampliación», como continua: integración con YouTrack, métricas avanzadas y mejoras según el uso. Su alcance «se decide con change management y QA». **Para esta entrega se eligió solo métricas.** YouTrack, la importación de la hoja y la ingesta de JUnit XML quedan fuera (§6).

Las métricas que pide §10 y lo que ya existe de cada una:

| Métrica (§10) | Qué hay hoy | Qué falta |
|---|---|---|
| **Cobertura por REQ** | La ficha del REQ muestra la cobertura de ese REQ; la matriz y el inicio muestran la cobertura global del filtro (TC con RUN aprobado / TC requeridos). | Ver la cobertura **REQ por REQ** en una tabla: cuáles están por debajo de lo que exige su RISK y por cuánto. |
| **Defectos por estado y tiempo en estado** | El inicio agrupa los DEF **abiertos** por estado, con la mediana y el máximo del tiempo en el estado **actual** (lo calcula `defect` con la última transición de la auditoría). | El **histórico**: cuánto tiempo pasan los DEF en cada estado, incluidos los ya cerrados; tiempo de punta a punta (abierto → cerrado); reaperturas. Por período, sistema y severidad. |
| **Tamaño de regresión** | Cada PKG guarda la selección propuesta y la final con el origen de cada TC (`regression_item`), y la pestaña *Regresión* del PKG la muestra. | Comparar **entre paquetes**: TC propuestos, finales, agregados y quitados por QA, por origen (impacto, crítico, manual), y cuánto de eso tiene RUN aprobado. |
| **Defectos escapados a producción** | Nada. No hay forma de distinguir un DEF encontrado en pruebas de uno encontrado después de desplegar. | Definir qué es un DEF escapado (§5, pregunta 1), contarlos y listarlos. |

Regla que manda sobre todo lo anterior: **sin SLA impuesto** (regla 3, §8.3). Las métricas miden; no marcan nada en rojo por pasarse de un umbral que nadie definió.

**Criterio de salida** `[SUPUESTO]`: change management y QA ven la pantalla de métricas con los datos del laboratorio y aprueban las definiciones (sobre todo la de DEF escapado). Hasta que lo confirmen, es la demostración que se prepara.

## 2. Estado del repositorio

Todo está en `main`. `make check` y `make check-offline` pasan enteros.

- **Backend** (`src/`, Go 1.26, módulo `kaoru`): gin, pgx + sqlc, goose. Un feature por carpeta en `src/internal/feature/`: `workflow`, `audit`, `user`, `change`, `requirement`, `risk`, `testdesign`, `execution`, `defect`, `document`, `comment`, `traceability`, `savedview`, `component`, `build`, `environment`, `bundle` (PKG), `regression`, `production` (CHK, DIAG, SMOKE), `report` (RPT), `planning` (hitos y calendario).
- **Base:** migraciones `src/migrations/00001` a `00017`. Dos roles: `kaoru_owner` migra, `kaoru_app` ejecuta (sin `UPDATE` ni `DELETE` en las tablas de solo inserción).
- **Contrato:** `api/openapi.yaml` (82 rutas, 122 operaciones). Una prueba exige que las rutas registradas y las del contrato coincidan.
- **Front** (`web/`): React 19 + TypeScript + Vite, TanStack Query, sin librería de componentes ni de gráficos. La barra lateral (`web/src/app/sections.tsx`) **no tiene** sección de métricas todavía.
- **Despliegue** (`infra/`): Compose con Nginx, oauth2-proxy, Keycloak, Postgres y la API; servicio opcional `libreoffice`.

De dónde salen los datos de cada métrica, que la fase 5 debe reutilizar y no duplicar:

- **Cobertura:** `traceability` ya la calcula por REQ para la ficha y la matriz (TC requeridos según el nivel de RISK, TC con RUN aprobado). No se reescribe esa regla en otro lado.
- **Historia de estados:** no hay tabla de historia. Cada transición de REQ, DEF, CHG y PKG queda en `audit_entry` con `action = 'transition'`, el estado anterior y el nuevo, y `occurred_at`. `audit` ya expone `GetLastTransitions`; el histórico necesita un caso de uso hermano que devuelva **todas** las transiciones de un tipo de entidad en un período. El índice actual es `(entity_type, entity_id, seq)`: probablemente haga falta uno por `(entity_type, action, occurred_at)`.
- **Regresión:** `regression_selection` y `regression_item` (fase 3), dueño `regression`.
- **Producción:** `deployment` sobre un ambiente de `kind = 'production'` con `result = 'success'` (dueño `environment`), y `package_requirement` (dueño `bundle`) dicen desde cuándo un REQ está en producción.

## 3. Cómo se trabaja aquí

Igual que en la fase 4; lo que sigue es lo que hace que `make check` siga pasando.

- **Arquitectura por feature:** `domain`, `application/port`, `application/usecase` (un archivo por caso de uso), `adapter/<tecnología>`, `delivery/api`, `module.go`. Un feature escribe solo sus tablas. Para pedirle algo a otro feature: puerto + adaptador que llama a sus casos de uso. Lecturas SQL de tablas ajenas, solo en el sentido de la llave foránea. Si aparece un ciclo, se resuelve con enlace tardío en `bootstrap` (D30 y D36).
- **Sin `time.Now` ni `log/slog`** en dominio ni casos de uso: reloj inyectado, y el log solo en `bootstrap`. Las métricas dependen del «ahora» (tiempo en estado de lo que sigue abierto): el reloj inyectado lo hace probable.
- **Las métricas solo leen.** No hay escrituras nuevas, así que no hay permisos ni auditoría nuevos: leer no exige permiso en Kaoru.
- **Errores:** `axis` (códigos) y `problem` (RFC 9457). Un filtro inválido (fecha mal escrita, `from` después de `to`) es error de validación por campo.
- **Orden de trabajo que funcionó:** primero el contrato (y la migración del índice, si hace falta), después cada caso de uso con sus pruebas, después el front, y al final probar en el navegador contra la API real.
- **Pruebas obligatorias** (reglas §A.17): dominio (los cálculos: medianas, percentiles, duraciones por estado, quién escapó), casos de uso con dobles de puertos escritos a mano, e integración contra Postgres real en `src/internal/bootstrap/*_integration_test.go`. Las métricas se prestan a pruebas de tabla con historias de transiciones armadas a mano.
- **Rendimiento:** `make check-perf` mide medianas con volumen (`TestPerformanceWithVolume`). Cada endpoint de métricas entra ahí; el umbral de las demás consultas es 200 ms.
- **Al agregar un feature** hay que tocar: `src/sqlc.yaml` (si tiene consultas), `src/internal/archtest/arch_test.go`, `src/internal/bootstrap/router.go` (módulo y rutas).
- **Decisiones y supuestos** se registran en `docs/decisiones.md` con el formato de las entradas existentes. No se inventan datos de negocio: lo que falta va como `[POR DEFINIR]`.
- **Commits** en español, en imperativo, en una rama `feature/fase-5`, y se fusionan en `main` con `--no-ff` tras confirmación.

### Comandos

| Para | Comando |
|---|---|
| Todo lo que corre la CI | `make check` |
| Solo Go (vet, lint, pruebas unitarias) | `make check-go` |
| Regenerar sqlc tras tocar consultas | `make generate` |
| Integración contra Postgres real | `make check-integration` |
| Rendimiento con volumen | `make check-perf` |
| Paquete offline con respaldo y restauración | `make check-offline` |
| Levantar el entorno de desarrollo | `KAORU_WEB_DIR="$PWD/web/dist" tools/dev-up.sh` |
| Datos de ejemplo | `tools/dev-seed.sh` |
| Regrabar las respuestas de ejemplo del front | `tools/dev-capture.sh` |
| Front contra la API real, en el navegador | `npm run dev:real --prefix web -- <usuario>` (http://localhost:5173) |
| Apagar el entorno (con `-v` borra los datos) | `tools/dev-down.sh` |

### Tropiezos conocidos

- **Las pruebas de despliegue de `make check` usan los puertos del entorno de desarrollo.** Hay que apagarlo antes con `tools/dev-down.sh`.
- **`web/mocks/fixtures.json` son respuestas reales grabadas.** No se editan a mano. Si se siembra dos veces, los ID cambian y varias pruebas del front fallan: `tools/dev-down.sh -v`, levantar, sembrar una vez y regrabar.
- **La semilla ocurre en segundos.** Todas las transiciones de `tools/dev-seed.sh` caen en el mismo minuto, así que los tiempos en estado del laboratorio son casi cero. Para que la demostración diga algo, la semilla tiene que fechar hacia atrás lo que pueda (como hace con la ventana del CHG-2) o el front tiene que probarse con datos armados en `test-data.ts`. La auditoría **no** se puede fechar hacia atrás: es de solo inserción y encadenada.
- **eslint prohíbe que un feature del front importe otro.** Una pantalla que combina datos de varios features va en `web/src/app/`.
- **Un `<select>` que recibe sus opciones tarde** no toma el valor por omisión: se le pone `key={opciones.length}` (fase 4).

## 4. Diseño propuesto `[SUPUESTO]`

Es una propuesta para arrancar. Quien construya la fase puede cambiarla; lo que cambie se registra en `docs/decisiones.md`.

### 4.1 Dónde vive cada cálculo

Cada métrica la calcula el feature dueño de sus datos, con su propio caso de uso y su consulta agregada (una consulta por métrica, no una por fila). Un feature nuevo `metrics`, **sin tablas**, junta lo que pide la pantalla a través de puertos, igual que `report` junta el informe.

| Métrica | Caso de uso dueño | Notas |
|---|---|---|
| Cobertura por REQ | `traceability` | Reutiliza el cálculo de la ficha, en lote y paginado. |
| Historia de estados de DEF | `audit` (transiciones en un período) + `defect` (duraciones por estado) | `audit` devuelve las transiciones crudas; el dominio de `defect` las convierte en tramos por estado. Lo que sigue abierto cuenta hasta «ahora». |
| Tamaño de regresión | `regression` | Una fila por PKG con selección confirmada. |
| DEF escapados | `defect`, con la fecha de primer despliegue exitoso en producción de cada REQ que le da `environment` o `bundle` por puerto | Quién calcula esa fecha se decide al construir, respetando la regla de lecturas ajenas (§3). |

Sin tablas nuevas. La única migración probable es el índice de `audit_entry` para leer transiciones por tipo y fecha.

### 4.2 Definiciones

- **Cobertura por REQ:** TC requeridos (según el nivel de su RISK), TC con RUN aprobado, porcentaje y cuántos faltan. Un REQ sin RISK aparece como «sin evaluar», no como 0 %. Filtros: sistema, CHG, PKG, estado; orden por menor cobertura.
- **Tiempo en estado (histórico):** para cada DEF, los tramos entre una transición y la siguiente (el primero empieza al crearse). Por estado: cuántos DEF pasaron, mediana y percentil 90 en días naturales. Además: tiempo de punta a punta de los DEF que llegaron a un estado final, y cuántos se reabrieron. Filtros: sistema, severidad y período (DEF **creados** en el período).
- **Tamaño de regresión:** por PKG con selección confirmada: TC propuestos, TC finales, agregados y quitados por QA, desglose por origen (`impact`, `critical`, `manual`), y TC finales con RUN aprobado sobre un BUILD del paquete. Además, el tamaño final frente al total de TC del sistema. Filtros: sistema y período (fecha de confirmación).
- **DEF escapado** `[SUPUESTO]`: un DEF abierto **después** de que su REQ llegó a producción, es decir, después del primer despliegue exitoso en un ambiente de producción de un PKG que contiene ese REQ. Se calcula; no se captura a mano y sirve para los DEF ya registrados. Tasa de escape: escapados / todos los DEF del período. Filtros: sistema, severidad y período. Ver la pregunta 1.

### 4.3 Contrato

`[SUPUESTO]` Cuatro rutas de solo lectura, una por métrica, con filtros en la consulta como el resto de la API:

| Ruta | Devuelve |
|---|---|
| `GET /metrics/coverage` | Una fila por REQ, paginada, más el resumen del filtro. |
| `GET /metrics/defect-states` | Por estado: cantidad, mediana, p90; más punta a punta y reaperturas. Opcionalmente por severidad. |
| `GET /metrics/regression` | Una fila por PKG, paginada, más totales. |
| `GET /metrics/escaped-defects` | Totales, tasa por mes y la lista de DEF escapados con su REQ, su PKG y la fecha en que llegó a producción. |

Los períodos agrupan por mes calendario en UTC. Sin período, los últimos 12 meses `[SUPUESTO]`.

**Exportar:** CSV de cada tabla, reutilizando lo que hace `traceability/delivery/api/export.go`. Es lo que pide quien quiere llevar el número a otro lado; un XLSX se agrega si se pide.

### 4.4 Front

- **Sección nueva «Métricas»** en la barra lateral, con cuatro pantallas (o una pantalla con cuatro pestañas): cobertura, defectos, regresión y escapados. Los filtros quedan en la URL y se pueden guardar como vista, como en las demás listas.
- **Sin librería de gráficos** (D9 y la regla de una herramienta por problema). Tablas, los `kpis` que ya existen y, si una tendencia por mes se lee mejor como barras, barras hechas con CSS o un SVG propio pequeño. Cada barra lleva su número escrito: el color no es la única señal.
- **Enlaces:** cada REQ, DEF y PKG de las tablas es un enlace a su ficha, como en la matriz.
- **Inicio:** se queda como está (cumple §7.7). A lo sumo, un enlace de «Defectos por estado» a la métrica histórica.
- El boceto (`tools/boceto`) no dibuja estas pantallas: se componen con las clases que ya existen, sin estilo nuevo.

## 5. Lo que falta decidir antes o durante

Ninguna de estas preguntas está respondida. Se puede avanzar con el supuesto indicado, marcándolo.

| # | Pregunta | A quién | Supuesto para avanzar |
|---|---|---|---|
| 1 | ¿Qué es un defecto escapado a producción? ¿Calculado (abierto después de que su REQ llegó a producción) o marcado a mano al abrir el DEF? | QA y change management | Calculado, como en §4.2. Si se prefiere marcarlo, es un campo nuevo en `defect` y solo vale de ahí en adelante. |
| 2 | ¿El tiempo en estado se mide en días naturales o hábiles? | QA | Naturales: no hay calendario de feriados en Kaoru. |
| 3 | ¿El tamaño de regresión se mide en TC o en tiempo de ejecución? | QA | En TC: el RUN no registra cuánto duró. |
| 4 | ¿Qué período y qué agrupación quieren ver por omisión? | Change management y QA | Últimos 12 meses, por mes. |
| 5 | ¿Hace falta comparar contra metas? | Change management y QA | No: sin SLA impuesto (regla 3). Se mide y se muestra. |
| 6 | ¿Las métricas las ve todo el mundo? | QA | Sí: leer no exige permiso en Kaoru. |
| 7 | ¿Cuál es el criterio de salida de esta entrega? | Change management y QA | Demostración con datos del laboratorio y definiciones aprobadas (§1). |
| 8 | ¿Se agregan después YouTrack, la importación de la hoja o JUnit XML a la fase 5? | Change management y QA | No en esta entrega; ver §6. |

## 6. Deuda que la fase 5 hereda

No es parte de esta entrega, pero conviene saberlo:

- **Integración con YouTrack** (fase 5, opcional y desactivable): sin hacer. Necesita versión, URL interna y token.
- **Importación de la hoja** (MVP): sin hacer; espera el formato de la hoja (pregunta 5 del requerimiento).
- **Ingesta de JUnit XML y cobertura** (fase 3, regla 8): sin hacer.
- **Visor de OpenAPI empaquetado, PDF.js empaquetado y resaltado de logs** (MVP): sin hacer.
- **Pantallas que nadie probó a mano**: de la fase 3, alta de BUILD, ambiente y componente, registrar un despliegue, contenido del PKG, impacto en REQ y TC (D49); de la fase 4, entregar y revisar un DIAG desde el front, subir evidencia a un CHK, el tema claro y el ancho de teléfono (D51).
- **Nunca se verificó:** login y logout en un navegador real, la instalación en un servidor sin red, y el rendimiento en el servidor objetivo.
- **Correo de alertas:** espera saber si hay SMTP interno (pregunta 13).
- **Los criterios de salida del MVP y de las fases 3 y 4 no se cumplieron**: dependen de usar el sistema en CHG reales. Las fases se construyen a pedido, por delante de eso.

## 7. Qué significa «terminado»

Del requerimiento (§12): pruebas automatizadas en verde, migraciones aplicadas sobre una base limpia, `docker compose up` probado a partir del paquete offline, documentación actualizada y demostración a quien firma el criterio de salida.

En este repositorio eso es:

1. `make check` y `make check-offline` pasan, con los endpoints de métricas en `make check-perf`.
2. Lo nuevo se probó en el navegador contra la API real, y lo que no, se dice.
3. `docs/decisiones.md` tiene las decisiones y los supuestos; `docs/flujos.md` (definiciones de cada métrica) y `docs/manual-usuario.md` están al día, y `docs/modelo-datos.md` si cambia el esquema.
4. La entrega separa Decisiones, Supuestos y Pendientes (requerimiento §13).
