# Anexo A — Reglas del repositorio

> Adaptadas de las reglas de arquitectura de Aldía. Se quitó lo que solo aplica a Aldía (cobros, SyPago, checkout, templ/htmx, API para terceros, cola de trabajos, paletas) y se ajustó lo que cambia en un sistema de equipo con roles en lugar de datos por dueño. Al iniciar el repositorio, este anexo se copia a `docs/reglas.md` y desde ahí es la fuente única.

## A.1 Principios

Dos reglas mandan sobre todas: **KISS** y **DRY**. El resto de este anexo las aplica a la arquitectura. Si una regla de aquí las contradice, se cambia la regla.

### A.1.1 KISS: mantenlo simple
- **Solo lo que pide la fase.** Si ningún caso de uso lo usa hoy, no se escribe: ni interfaces, ni parámetros, ni configuración, ni carpetas, ni rutas «por si acaso».
- **Gana el diseño con menos piezas.** Entre dos que cumplen, el de menos paquetes, tipos, saltos al leerlo y opciones de configuración.
- **Primero lo que ya existe.** Biblioteca estándar, después una librería conocida y mantenida, y al final código propio. Nunca un framework propio.
- **Una interfaz por necesidad.** Un puerto aísla E/S (base, Keycloak, archivos) o algo que hay que falsear en pruebas. Un tipo puro no lleva interfaz, y una sola implementación no pide fábrica.
- **Regla de tres.** Una abstracción se escribe al tercer uso. Con dos, se aceptan dos copias si no expresan la misma regla.
- **Medir antes de optimizar.** No se agregan caché, índices extra ni concurrencia sin una medición que los pida.
- **Una herramienta por problema.** Un router (gin), un cliente de base (pgx + sqlc), una librería de caché de peticiones en el front (TanStack Query), un tema visual. Sumar otra se justifica en `docs/decisiones.md`.
- **Código que se lee de arriba abajo.** Funciones cortas, retornos tempranos y tipos concretos. Sin reflexión, y sin genéricos cuando un tipo concreto basta.
- **Las reglas también pesan.** Si una regla estorba más de lo que protege, se propone quitarla.

### A.1.2 DRY: no te repitas
DRY habla de **conocimiento**, no de texto: cada regla de negocio, contrato o dato de configuración tiene una sola fuente. Dos funciones parecidas que responden a reglas distintas no son repetición.

| Conocimiento | Fuente única | Lo que se deriva de ella |
|---|---|---|
| Reglas de datos de un feature | Su `domain`. | La validación del front. Las dos copias pasan la misma tabla de casos (abajo). |
| Contrato HTTP | `api/openapi.yaml`. | Los tipos TypeScript, generados. Una prueba compara las rutas registradas con el archivo. |
| Esquema de la base | `src/migrations/`. | El código de sqlc, generado. |
| Código de error → HTTP | `problem.StatusOf`. | Todas las respuestas de error y un solo lector de problem+json en el front. |
| Configuración | `bootstrap/config.go` y `config.json`. | Los módulos reciben valores por `Deps`. El front recibe del API lo que necesita (por ejemplo, los enlaces de Keycloak); no guarda su propia copia. |
| Flujos, estados y reglas de bloqueo | Las tablas de configuración del feature `workflow`, con sus valores iniciales en una migración. | El dominio las evalúa; el front las lee del API para pintar la línea de vida. |
| Apariencia y textos | `tools/boceto`. | `web/src/styles/tokens.css`, las clases de los componentes y los textos de las pantallas. |
| Qué se construye y en qué orden | Este requerimiento y el plan de fases. | Cada documento enlaza al otro en lugar de repetirlo. |

- **Generar antes que copiar.** Lo que sale de una fuente se genera y se versiona: sqlc y los tipos de OpenAPI.
- **Una regla en dos lenguajes, una tabla de casos.** Si el front repite una validación del dominio, las dos implementaciones pasan la misma tabla: `api/testdata/validation-cases.json`, con entrada, si vale y el valor normalizado. La leen las pruebas de Go y las de Vitest.
- **Lo repetido sube al lugar que le toca.** Lo técnico y genérico va a `core/`. El negocio con el mismo significado en dos features va a `shared/domain`. Un componente visual usado en dos pantallas va a `web/src/shared/ui`. Lo que solo se parece se queda donde está.
- **Un cambio, un lugar.** Si cambiar una regla obliga a tocar varios archivos que dicen lo mismo, falta una fuente única.

### A.1.3 Cuando chocan
- **En código, KISS le gana a un DRY prematuro.** Si quitar una repetición exige una abstracción más difícil de leer que las copias, se dejan las copias hasta el tercer uso.
- **En conocimiento, DRY gana siempre.** Una regla de negocio, un contrato o un valor de configuración nunca tienen dos fuentes.

### A.1.4 Arquitectura
- **El negocio en el centro.** Dominio y casos de uso no conocen HTTP, SQL, Keycloak ni React. Si una regla de negocio necesita un detalle técnico, falta un puerto.
- **El feature es la unidad.** Un feature es un bloque de producto (requerimientos, defectos, ejecuciones) con sus propias capas. Se lee, se prueba y se borra entero.
- **Las dependencias apuntan hacia adentro.** `delivery → application → domain`; los adaptadores implementan puertos de `application`. Las implementaciones concretas solo se ensamblan en la composición (§A.15).
- **Lo explícito gana.** Dependencias por constructor, identidad por parámetro, errores con código, tiempo e IDs inyectados. Sin variables globales ni `init()` con efectos.
- **Pragmatismo en los bordes.** Un adaptador o un handler puede ser plano y simple. La disciplina está en no filtrar detalles técnicos hacia el centro.
- **Lo que no se verifica, se rompe.** Formato, capas, código generado y pruebas se comprueban en CI (§A.18).

## A.2 Vocabulario: «caso de uso» tiene dos significados
- En el **código**, un caso de uso es una operación de la aplicación (`application/usecase/create_requirement.go`).
- En el **negocio**, un UC- es un caso de uso funcional que define QA. En el código se llama `FunctionalCase` y vive en el feature `testdesign`.
- Nunca se usa `UseCase` como nombre de entidad.

## A.3 Árbol del repositorio

```
kaoru/
├── docs/                          # fuente de verdad: reglas, decisiones, modelo de datos, guías
│   ├── reglas.md                  # este anexo
│   └── decisiones.md              # registro de decisiones (§A.21)
├── infra/                         # Compose: Nginx, oauth2-proxy, Keycloak (realm exportado), Postgres; paquete offline
├── api/
│   ├── openapi.yaml               # contrato HTTP con el front
│   └── testdata/validation-cases.json
├── src/                           # módulo Go
│   ├── go.mod
│   ├── sqlc.yaml                  # uno solo, un bloque por feature
│   ├── migrations/                # SQL versionado (goose); también es el esquema de sqlc
│   ├── cmd/
│   │   └── kaoru-api/main.go
│   └── internal/
│       ├── core/                  # técnico y genérico: no sabe de QA
│       ├── shared/                # negocio compartido entre features
│       ├── feature/               # negocio (lista en §A.4.1)
│       ├── bootstrap/             # composición: arma y conecta todo
│       └── archtest/              # prueba de arquitectura (§A.18)
├── web/                           # SPA React (Vite + TypeScript)
├── tools/                         # boceto y scripts; src/ nunca los importa
└── Makefile                       # make check, make generate, make offline-bundle
```

Las carpetas vacías «para después» no se crean: cada feature aparece en la fase que lo necesita.

## A.4 Las carpetas de `internal/`

Cada carpeta responde una sola pregunta. Para decidir dónde va algo, se pregunta en este orden:

| # | Pregunta | Carpeta | Contiene | Puede depender de |
|---|---|---|---|---|
| 1 | ¿Es negocio de un solo feature? | `feature/<f>/` | Un directorio por feature con sus capas (§A.5). | `core/`, `shared/` y, de otro feature, lo que permite §A.7. |
| 2 | ¿Es negocio compartido por dos o más features, con el mismo significado? | `shared/domain/` | Tipos y reglas puras: `UserId`, `Code` (los IDs legibles como `REQ-2026-0042`), `Severity`, `RiskLevel`, `QualityCharacteristic` (ISO 25010), `Checksum`, `Missing` (el error de un dato que falta). | Biblioteca estándar. |
| 3 | ¿Es técnico y serviría igual fuera de este sistema? | `core/` | Vocabulario de errores (axis), Unit of Work y helpers de Postgres, servidor HTTP con sus middlewares (Recovery, request-id, salud) y problem+json, verificación OIDC, almacenamiento de archivos, logging, métricas, configuración, `Clock` e `IdGenerator`. No menciona QA ni sus features. | Biblioteca estándar y librerías externas. |
| 4 | ¿Decide qué piezas existen y cómo se conectan? | `bootstrap/` | Struct de configuración, creación de logger, métricas, pool y servidores, construcción de los módulos y montaje de middlewares y rutas. | Todo. |

- `core/` no importa nada de `internal/` fuera de sí mismo.
- `bootstrap/` solo lo importa `cmd/`.
- No hay carpeta `platform/`, `utils/` ni `common/`: lo técnico va a `core/`, la composición a `bootstrap/`.
- **Origen del core** `[POR DEFINIR]`: si se reutiliza el core de la empresa (axis, scribe, configuro, postgres_db), se copia en una versión fija y conserva sus nombres; si no, se escribe un core mínimo con la misma forma y solo lo que use la fase en curso.

### A.4.1 Features previstos
Se crean solo en la fase que los necesita (sección 12 del requerimiento).

| Feature | Entidades | Fase |
|---|---|---|
| `user` | Perfil y roles leídos del token | 2 |
| `workflow` | Estados, transiciones, permisos por transición y reglas de bloqueo configurables | 2 |
| `audit` | Registro de auditoría encadenado por hash | 2 |
| `change` | CHG-, rutas alta, ligera y fix menor | 2 |
| `requirement` | REQ-, enlaces (incluido Google Sheets), contrato de entrada | 2 |
| `risk` | RISK- | 2 |
| `testdesign` | UC- (`FunctionalCase`), TC- | 2 |
| `execution` | RUN-, EXP- | 2 |
| `defect` | DEF- | 2 |
| `document` | DOC-, versiones y visor | 2 |
| `traceability` | Matriz, ficha consolidada, búsqueda global (solo lectura, §A.7) | 2 |
| `sheetimport` | Migración de la hoja | 2 |
| `component` | COMP- | 3 |
| `build` | BUILD- y manifiesto | 3 |
| `bundle` | PKG- (`package` es palabra reservada de Go) | 3 |
| `regression` | Selección propuesta y final, con justificación | 3 |
| `environment` | ENV-, DEP- | 3 |
| `production` | CHK- con firmas, DIAG-, SMOKE- | 4 |
| `report` | RPT- (informe de conformidad) | 4 |
| `planning` | Hitos, ventanas de cambio y alertas de atraso | 4 |
| `metrics` | Cobertura por REQ, tiempo en estado de los DEF, tamaño de regresión y DEF escapados (solo lectura, §A.7) | 5 |

Si un feature crece demasiado, se parte en features hermanos (`defect`, `defecttriage`), no en sub-features anidados.

## A.5 Anatomía de un feature

```
internal/feature/requirement/
├── module.go                     # composición del feature (§A.15)
├── domain/                       # entidades, value objects, reglas, errores de dominio
├── application/
│   ├── port/                     # interfaces que necesitan los casos de uso
│   └── usecase/                  # un archivo por caso de uso
├── adapter/
│   ├── postgres/                 # implementa los puertos con sqlc
│   │   ├── queries/*.sql
│   │   └── sqlcgen/              # generado; no se edita
│   └── <otro>/                   # un subpaquete por tecnología (keycloak, filestore…)
└── delivery/
    └── api/                      # handlers JSON, DTOs, rutas
```

Los nombres de carpeta son de tecnología o de rol, nunca de patrón genérico: `adapter/postgres`, `delivery/api`. No se usa `delivery/http` porque el paquete choca con `net/http`.

## A.6 Capas: qué puede importar cada una

| Capa | Contiene | Puede importar | No puede importar |
|---|---|---|---|
| `domain` | Entidades, value objects, políticas, errores de dominio. | Biblioteca estándar, `shared/domain`, el vocabulario de errores del core. | `application`, `adapter`, `delivery`, `bootstrap`, pgx, sqlc, gin, `net/http`. Tampoco lleva etiquetas `json` ni `db`. |
| `application/port` | Interfaces de lo que necesitan los casos de uso y los tipos que esas interfaces usan. | `domain`, `shared/domain`, interfaces del core. | `adapter`, `delivery`, drivers. |
| `application/usecase` | Casos de uso. | `domain`, `port`, `shared/domain`, interfaces del core (`Clock`, `IdGenerator`, `UnitOfWorkFactory`, `Logger`). | gin, `net/http`, pgx, sqlc, el logger concreto. |
| `adapter/*` | Implementaciones de puertos. | `port`, `domain`, `core/pkg/*`, drivers y SDKs; de otro feature, su `application/usecase` (§A.7). | Su propio `usecase`, `delivery`, otros adaptadores. |
| `delivery/*` | Handlers, DTOs y rutas. | `usecase`, `domain` (para leer tipos), problem+json del core, el `Principal` de OIDC. | `adapter`, `sqlcgen`, `port`. |
| `module.go` | Ensambla el feature. | Todo su feature, `core/`, `shared/`; de otro feature, su `application/usecase`. | El resto del interior de otro feature y `bootstrap/`. |

## A.7 Entre features
- Un feature puede importar de otro solo **tipos de su `domain` que sean identificadores o value objects de lectura**. Si dos features necesitan el mismo tipo, se promueve a `shared/domain`.
- Para **usar** a otro feature, el consumidor declara un puerto en su `application/port` y un adaptador en su `adapter/` lo implementa llamando a los casos de uso del otro (que expone su `module.go`). Ejemplo: `defect/application/port.Executions`, implementado por `defect/adapter/executions`, que llama a los casos de uso de `execution` para validar que el RUN existe.
- **Leer la tabla de otro feature, solo en la dirección de la llave.** Una consulta puede leer columnas de la tabla de otro feature cuando su propia tabla tiene una llave foránea hacia ella; **nunca escribe** en ella. La revisión del PR pregunta: «¿escribe en la tabla de otro feature?».
- **Excepción declarada: `traceability`.** Es un feature de solo lectura que cruza todas las tablas para armar la matriz, la ficha consolidada y la búsqueda global. No escribe nada, no tiene entidades propias y devuelve lecturas planas. archtest verifica que su adaptador no ejecute `INSERT`, `UPDATE` ni `DELETE`. `metrics` (fase 5) es la segunda excepción, con las mismas condiciones: cruza las tablas para medir y no escribe nada.
- **Auditoría:** cada feature declara un puerto `AuditTrail` y su adaptador llama al caso de uso `Record` de `audit` **dentro de la misma transacción** (Within reutiliza la del context). Si la auditoría falla, la operación no se guarda.
- **Estados y bloqueos:** cada feature con ciclo de vida declara un puerto `Workflow` que su adaptador implementa con los casos de uso de `workflow`. El feature pregunta «¿vale esta transición para este rol?» y aplica la respuesta en su dominio.
- El adaptador **conserva el código de error** del otro feature. Nunca hace `errors.New(err.Error())`.
- Nunca se importa `adapter`, `delivery`, `port` ni `sqlcgen` de otro feature.
- La dependencia entre features es de **un solo sentido**. Si aparece un ciclo, el corte de features está mal; la única salida aceptada es un puerto que `bootstrap` conecta después de armar los dos (y se registra en `docs/decisiones.md`).

## A.8 Dominio
- **Value objects:** tipos inmutables con campos privados y un constructor que valida: `ParseCode(raw)`, `ParseSeverity`, `ParseChecksum`, `ParseSheetLink` (extrae ID, `gid` y rango de una URL de Google Sheets). Tienen métodos para leer (`String()`, etc.).
- **Entidades:** se crean con `NewX(...) (X, error)`, que valida invariantes. Los cambios de estado son métodos con nombre de negocio (`req.DeliverToQA(...)`, `def.Triage(...)`, `pkg.MarkReadyToRelease(...)`) que devuelven error si la transición no vale. Solo el adaptador de persistencia asigna los campos directo, al rehidratar.
- **Inmutables por regla de negocio:** BUILD-, RUN- y RPT- no tienen métodos para cambiar su contenido una vez creados.
- **Tiempo e IDs llegan como parámetros.** El dominio no llama a `time.Now()`, `uuid.New()` ni `rand`.
- **Errores de dominio:** variables con código, en `domain/errors.go`. El mensaje es el que puede ver el usuario.
- Un tipo de dominio **nunca se serializa directo** (`json.Marshal`, etiquetas `db`). El adaptador lo convierte en un registro explícito.
- **Nada especulativo:** no se escriben políticas, estados ni errores que ningún caso de uso usa todavía.

## A.9 Casos de uso
- **Forma:** un paquete `usecase` por feature y un archivo por caso de uso, con su struct, su `Input`, su `Output` y `Execute(ctx, in) (Output, error)`. Un `usecase.go` declara `Deps`, `UseCases` y `New(d Deps) (*UseCases, error)`, que falla si falta una dependencia.
- **La identidad entra en el Input** (`ActorId` y `Roles`, sacados del token por el handler), no se lee del context. El caso de uso no sabe de tokens ni cabeceras.
- **La autorización se decide aquí:** el caso de uso comprueba el permiso (`req.deliver`, `pkg.build`, `def.defer`…) contra los roles del Input, con la tabla de permisos de `workflow`. Un permiso negado es `PERMISSION_DENIED`.
- `Input` y `Output` no llevan etiquetas `json` ni `validate`. Los DTOs de transporte viven en `delivery`.
- El caso de uso **valida a través del dominio**. No repite reglas con `strings.TrimSpace` ni etiquetas de validador.
- El caso de uso **decide la transacción** con `UnitOfWorkFactory` y `Within`. La entrega nunca abre transacciones.
- **Un caso de uso no llama a otro** del mismo feature. La lógica común va al dominio o a una función del paquete `usecase`. Entre features se aplica §A.7.
- **Toda escritura registra auditoría** por su puerto `AuditTrail`, en la misma transacción: actor, fecha, entidad, campo, valor anterior, valor nuevo y motivo.
- **Sin observabilidad propia por caso de uso.** Cada ruta llama a un solo caso de uso, así que el log de la petición y las métricas HTTP ya registran cada uno.
- Lecturas y escrituras son casos de uso por igual. Si una lectura arma una respuesta de varios agregados, devuelve un `Output` plano propio.

## A.10 Errores

### A.10.1 Modelo
- Las funciones devuelven `(T, error)` o `error`. El error que viaja es un error con código (`*axis.GeneralError` si se reutiliza el core).
- Se crea con código y mensaje para el usuario; la causa técnica va en `cause`, nunca en el mensaje.
- Se pregunta por código (`CodeOf(err)`, `IsCode(err, …)`), aunque venga envuelto con `fmt.Errorf("…: %w", err)`.
- Los errores de dominio son variables y se comparan con `errors.Is`.
- Los errores de validación llevan la lista de campos (`field`, `code`, `message`); un formulario junta todos sus errores. El campo lleva `required` si falta el dato e `invalid` si no; el código nunca se deduce del texto.
- Un error sin código (de cualquier librería) llega a la entrega como `INTERNAL_SERVER_ERROR`. Los adaptadores traducen los errores técnicos que conocen.
- No se usan tipos `Result[T]`.

### A.10.2 Traducción a HTTP
Una sola tabla, en `problem.StatusOf`. Ningún handler escribe su propio mapeo.

**Campo o conflicto.** Si el usuario corrige el error en un campo, es `INVALID_REQUEST` (422) con `errors[]`, aunque choque con datos guardados. Los 409 quedan para lo que no tiene campo: una acción que no vale en el estado actual (`INVALID_ACTION`: aprobar un PKG con un DEF alto abierto) o una escritura que chocó con otra (`CONFLICT`).

| Código | HTTP |
|---|---|
| `INVALID_FORMAT` (JSON ilegible, tipos incorrectos) | 400 |
| `INVALID_REQUEST` (reglas de datos, con `errors[]`) | 422 |
| `NOT_AUTHENTICATED`, `INVALID_TOKEN` | 401 |
| `PERMISSION_DENIED` | 403 |
| `NOT_FOUND_RESOURCE` | 404 |
| `DUPLICATE_RESOURCE`, `CONFLICT`, `PRECONDITION_FAILED`, `INVALID_ACTION` | 409 |
| `RESOURCE_EXHAUSTED` (archivo demasiado grande) | 413 |
| `NOT_SUPPORTED` (tipo de archivo) | 415 |
| `SERVICE_UNAVAILABLE` | 503 |
| `TIMEOUT` | 504 |
| `CANCELED` (el cliente se fue) | 499; no cuenta como 500 en logs ni métricas |
| Cualquier otro, o un error sin código | 500 |

### A.10.3 Formato: RFC 9457 (`application/problem+json`)
```json
{
  "type": "about:blank",
  "title": "Acción no permitida",
  "status": 409,
  "code": "INVALID_ACTION",
  "detail": "El paquete tiene 1 defecto de severidad alta abierto (DEF-2026-0091).",
  "traceId": "4bf92f3577b34da6a3ce929d0e0e4736"
}
```
- En un 4xx, `detail` lleva el mensaje del error (apto para el usuario) y, en un 422, la lista `errors`.
- En un 5xx, un texto genérico del código, nunca el mensaje interno.
- Nunca se envía `err.Error()` de un error interno, ni textos de pgx, Keycloak o SQL. El detalle técnico va al log con el mismo `traceId`.
- Un 5xx se registra con el detalle completo; un 4xx solo con su código, porque su causa puede traer datos del usuario.

## A.11 Persistencia
- **Una base, un esquema** (`kaoru`). La base de Keycloak es otra y se configura en `infra/`.
- **Migraciones con goose** en `src/migrations/`, embebidas en el binario y aplicadas al arrancar con un bloqueo para que dos procesos no migren a la vez. Nombres en inglés: `00001_create_app_user.sql`. **Una migración aplicada no se edita: se escribe otra.** Los datos de prueba van en `migrations/dev/` y nunca se aplican en producción.
- **sqlc lee el esquema de `migrations/`.** No hay un segundo DDL a mano. Un solo `src/sqlc.yaml` con un bloque por feature.
- **Tipos:** `uuid` para llaves (UUIDv7, generadas por la aplicación; la excepción es `app_user.id`, que es el `sub` de Keycloak); `timestamptz` en UTC; `text` con `CHECK` en lugar de `ENUM` para valores fijos. Los estados configurables no son `CHECK`: son filas de las tablas de `workflow`, referenciadas por llave foránea.
- **IDs legibles:** cada entidad tiene además una columna `code` única e inmutable (`REQ-2026-0042`), generada por una secuencia por tipo y año. La llave técnica sigue siendo el `uuid`.
- **Nombres:** tablas en singular y snake_case (`requirement`, `requirement_link`).
- **Marcas de tiempo:** `created_at` y `updated_at` en toda tabla de entidad. Una tabla hija que se escribe siempre con su dueño no las lleva. Las entidades inmutables (BUILD, RUN, RPT, auditoría) solo llevan `created_at`.
- **Borrado:** lo que tiene historia o forma parte de la evidencia **no se borra**; se anula o se cierra con un estado y queda en la auditoría. Lo que no deja historia (un borrador nunca usado) se borra de verdad, y lo que lo usa lo referencia con `on delete restrict`.
- **Acceso:** los datos son del equipo, no de un usuario. Lo que cada rol puede ver o cambiar lo decide el caso de uso (§A.9); los repositorios no filtran por dueño. Si más adelante se limita el acceso por sistema (SIMF, SGLPAR, SyPago), el filtro entra como parámetro explícito del repositorio y una prueba de integración lo verifica.
- **Repositorios:** devuelven tipos de dominio y mapean a mano entre fila y entidad; toman la transacción del context; llevan `var _ port.X = (*X)(nil)`; traducen errores de Postgres a códigos (23505 → `DUPLICATE_RESOURCE`; 23503, 23514, 23502 → `INVALID_REQUEST`; sin filas → `NOT_FOUND_RESOURCE`; 40001/40P01 y sin conexión → `SERVICE_UNAVAILABLE`; plazo vencido → `TIMEOUT`; cancelado → `CANCELED`). Una restricción con significado de negocio se traduce a su error de dominio.
- **Las reglas de estado no viven en SQL.** Una consulta guarda lo que el dominio decidió. Lo que depende del día (tiempo en estado, atraso de un hito) **se calcula al leer**, no con una tarea que reescribe estados.
- **Auditoría:** tabla de solo inserción; el usuario de base de la aplicación no tiene `UPDATE` ni `DELETE` sobre ella. Cada fila guarda el hash de la anterior.
- **Archivos:** el contenido de DOC- y de la evidencia vive en el almacenamiento de archivos (puerto `FileStore`), nombrado por su SHA-256; la base guarda solo los metadatos y el checksum.

## A.12 Transacciones
- La transacción la abre el caso de uso con `uow.Within(ctx, fn)`. Dentro de `fn` solo se usa el `ctx` que recibe `fn`.
- **Nunca hay E/S externa dentro de una transacción** (Keycloak, conversión con LibreOffice, escritura de archivos grandes). Para adjuntar un documento: primero se guarda el archivo por su checksum, después se registra en la base. Un archivo sin registro se puede limpiar; un registro sin archivo no puede existir.
- Los casos de uso anidados solo usan `Within`, nunca `Begin`/`Commit` a mano.
- **Sin cola de trabajos por ahora.** Las alertas de atraso se calculan al leer. Si una fase pide trabajo en segundo plano (por ejemplo, correo), se decide en `docs/decisiones.md` antes de agregarlo.

## A.13 Identidad y seguridad
- **El perímetro** (Nginx + oauth2-proxy) exige sesión. Nginx sobrescribe la cabecera del token, así que el navegador no puede inyectarla. El backend solo escucha en la red interna de Docker.
- **El backend verifica el token igual:** firma con el JWKS del realm (en caché), `iss`, `exp`, `nbf`, `azp` igual al cliente de la app y `typ = Bearer`. Nunca usa una lectura sin verificar. Si no puede bajar las llaves del JWKS (plazo de 5 s), responde 503 y no 401.
- **El handler pasa la identidad al caso de uso** (`ActorId = principal.Sub`, `Roles = principal.Roles`). Ningún identificador de actor llega en el cuerpo ni en una cabecera propia.
- **Roles:** salen de `realm_access.roles` y se comprueban en el caso de uso con la tabla de permisos de `workflow`.
- **Firmas del checklist:** una firma es el usuario autenticado + fecha del servidor + paso + versión del checklist + comentario. Nunca se acepta un nombre o una fecha que venga del cuerpo.
- **CSRF:** la cookie de sesión es `SameSite=Lax` y toda petición `POST`, `PUT`, `PATCH` o `DELETE` debe traer `Origin` igual al origen público de la app; si falta o es otro, 403.
- **Logs sin datos sensibles:** nunca se registran tokens, cookies ni cuerpos de petición. Se registran IDs.
- **Secretos solo por entorno.** `.env` nunca se versiona; `.env.example` sí, con valores de laboratorio.
- **Límites:** `http.MaxBytesReader` en toda ruta (1 MB por defecto; el límite de adjuntos es configurable), `ReadHeaderTimeout` en el servidor y proxies de confianza limitados a la red de Nginx.
- **Adjuntos:** lista de tipos permitidos y validación del tipo real del contenido; Markdown sanitizado al pintarse.

## A.14 Observabilidad
- **Logs** siempre a través del `Logger` inyectado en `Deps`. Nadie llama a un logger global fuera de `bootstrap/`. En contenedores, JSON a stdout; en desarrollo, legible.
- **Campos del context:** `trace_id`, `user_sub`, `use_case`. Mensajes en español y cortos; campos en inglés y snake_case.
- **Traza:** Nginx genera `X-Request-Id`; el backend lo acepta si viene, lo genera si no y lo devuelve. Es el `traceId` del problem+json.
- **Métricas Prometheus**, siempre con las mismas labels: `http_server_requests_total{method,route,status}` y `http_server_request_duration_seconds{method,route}`, donde `route` es la plantilla (`/api/requirements/:id`), nunca la URL; y las estadísticas del pool de Postgres. Nada más hasta que una medición lo pida.
- **Salud:** `/livez` (el proceso responde) y `/readyz` (la base responde y no se está apagando). Junto con `/metrics`, viven en un puerto interno que Nginx no publica.
- **Recuperación:** un middleware convierte un pánico en 500 problem+json y registra la pila.

## A.15 Composición, configuración y arranque
- **`module.go` por feature:** declara sus `Deps`, construye adaptadores y casos de uso, expone `UseCases` (lo que otros features pueden usar) y `RegisterAPI(r gin.IRouter)`. Es el único archivo del feature que conoce sus adaptadores concretos.
- **`bootstrap/`**, en orden: lee la configuración; crea logger, métricas, pool, Unit of Work y servidores; construye los módulos en orden de dependencia (`user`, `audit` y `workflow` primero); monta los middlewares globales (Recovery, request-id, OIDC, Origin); registra las rutas.
- `cmd/kaoru-api/main.go` queda en unas líneas: `bootstrap.RunAPI(ctx)`.
- **Configuración** en `config.json`; los secretos se referencian como `${POSTGRES_PASSWORD}` y llegan por entorno. La ruta sale de `KAORU_CONFIG_PATH`. Cada sección se lee una vez al arrancar en un struct de `bootstrap/config.go`, se valida y se pasa por `Deps`. Si falta algo, el proceso sale con un mensaje claro y código distinto de 0. Ningún adaptador lee la configuración en cada llamada. Sin `init()` con `panic`.
- **Arranque y apagado:** `signal.NotifyContext` y `errgroup` para el servidor público y el interno. Si `ListenAndServe` falla, el proceso termina con error. Al apagar, en orden: `/readyz` pasa a falso, `server.Shutdown(timeout)`, cierre del pool y vaciado de logs.

## A.16 Entrega: API JSON y React

### A.16.1 API JSON (`delivery/api`)
- Todo bajo `/api/`, sin versión: el único consumidor es nuestro front y se despliegan juntos. (La integración con YouTrack de la fase 5 es un adaptador saliente, no otra superficie.)
- **Contrato:** `api/openapi.yaml` es la fuente. Los tipos TypeScript se generan de él. Los handlers Go se escriben a mano, y una prueba compara las rutas registradas con las del archivo.
- **DTOs propios** con etiquetas `json` en camelCase. El handler convierte DTO → Input y Output → DTO. No se exponen entidades.
- **Respuestas:** crear → 201 con el recurso; borrar → 204; listas → `{ "items": [...], "page": 1, "size": 25, "total": 24 }`; errores → problem+json.
- **El handler hace cuatro cosas:** leer el `Principal`, decodificar el DTO, llamar al caso de uso y escribir la respuesta. Si hace más, la lógica está en el lugar equivocado.

### A.16.2 React (`web/`)
```
web/src/
├── app/            # router, proveedores, layout (barra lateral, búsqueda global, migas de pan)
├── features/
│   ├── changes/
│   ├── requirements/   # ficha de REQ, enlaces, visor de documentos
│   ├── testdesign/
│   ├── executions/
│   ├── defects/
│   ├── traceability/   # matriz
│   └── …               # uno por feature del backend que tenga pantallas
├── shared/
│   ├── api/        # cliente HTTP, tipos generados de OpenAPI, problem+json y 401
│   ├── ui/         # componentes base con los tokens del boceto
│   └── lib/        # formato de fechas, IDs, checksums
└── styles/tokens.css
```
- `features/*` no se importan entre sí; lo común va a `shared/`. Una pantalla que junta piezas de varios features (la ficha de REQ con sus defectos y ejecuciones) se arma en `app/` y recibe esas piezas por props.
- **Estado del servidor** con TanStack Query. No se copia a un store global.
- **Sin tokens en el navegador.** Si `/api` responde 401, el cliente redirige al inicio de sesión de oauth2-proxy con la ruta actual.
- **Un solo lector de errores** en `shared/api`: convierte también un fallo de red o una respuesta que no es problem+json en un error con texto del boceto. Ninguna pantalla muestra el mensaje técnico del navegador. Una lectura se repite hasta dos veces, solo si puede cambiar al repetirla (sin conexión, 502, 503, 504). Una pantalla que falla al pintarse muestra el texto del boceto dentro del armazón.
- Las validaciones replican las del dominio para dar feedback inmediato (misma tabla de casos, §A.1.2). Los `errors[]` del servidor se pintan bajo cada campo.
- **Secciones:** `app/sections.tsx` es la única lista de módulos: de ahí salen el menú, el título de la pestaña y la ayuda.
- **Cada pantalla, un archivo del paquete,** cargada con `React.lazy` y un esqueleto con la geometría del contenido mientras espera (solo la primera vez).
- **Accesibilidad (WCAG 2.2 AA):** paneles y diálogos devuelven el foco a quien los abrió; campos con etiqueta y `required`; al guardar con errores, foco al primero; avisos que se anuncian; texto con contraste 4,5:1 también sobre vidrio; nunca menos de 12 px.
- **Iconos:** todos desde `shared/ui/Icon`, con un solo trazo y tamaño; el nombre dice la forma (`calendar`), no el uso.
- **Tablas:** el selector de fila a la izquierda y las acciones a la derecha quedan fijos al desplazarse; las acciones son iconos con `title` y `aria-label`.
- **`tools/boceto` es la referencia visual y de textos.** Su CSS se reutiliza con sus mismas clases. No se agrega librería de componentes ni Tailwind. Un color nuevo es una variable de `tokens.css` con valor en los dos temas; ninguna regla lleva un color fijo.

## A.17 Pruebas
- Las pruebas van **junto al código** (`create_requirement_test.go` al lado de `create_requirement.go`), nunca en subcarpetas `test/`. `package x_test` cuando la prueba es de caja negra.
- Biblioteca estándar `testing` con **tablas de casos**. Los dobles de los puertos se escriben a mano; no se usan frameworks de mocks.
- El tiempo y los IDs de las pruebas son fijos (`Clock` e `IdGenerator` falsos).

| Capa | Qué se prueba | Cómo |
|---|---|---|
| `domain` | Cada regla de datos y cada transición de estado. | Unitarias con tabla. |
| `usecase` | Camino feliz, cada error de negocio, cada permiso negado, que registra auditoría y que no deja cambios si algo falla. | Unitarias con dobles de puertos y un UnitOfWork falso. |
| `adapter/postgres` | Ida y vuelta de cada entidad, traducción de errores e inmutabilidad (BUILD, RUN, RPT, auditoría). | Integración contra Postgres real (etiqueta de build `integration`). |
| `traceability` | Que la matriz y la ficha devuelven la cadena completa en escenarios armados de punta a punta. | Integración. |
| `delivery/api` | Mapeo DTO ↔ caso de uso, códigos HTTP y problem+json. | `httptest`. |
| `web/` | Formularios, reglas de UX y manejo de errores del servidor. | Vitest + Testing Library. |

## A.18 Calidad automática
Cada PR (y `make check` en local) pasa por:
- `gofmt`/`goimports`, `go vet` y `golangci-lint` (con staticcheck, errcheck, gosec y revive; configuración en `src/.golangci.yml`).
- `go test ./...` y las pruebas de integración (`-tags integration -p 1`: un paquete a la vez, porque comparten la base).
- **Código generado al día:** sqlc y los tipos de OpenAPI están versionados; CI ejecuta `go generate ./...` y `npm run generate`, y falla si `git diff` no queda vacío.
- **Test de arquitectura** (`src/internal/archtest`): recorre los imports con `go list` y falla si se rompe una regla de §A.4, §A.6 o §A.7, o si alguien fuera de `bootstrap/` importa el logger concreto. Con `go/ast` verifica que dominio, casos de uso y `shared/` no llamen a `time.Now`, y que las consultas de `traceability` no escriban.
- `sqlc vet` y el lint del contrato OpenAPI.
- En `web/`: lint, chequeo de tipos, pruebas y build.
- No entra código sin uso ni código especulativo; nada de binarios, `.env` ni logs en git. `go mod tidy` se corre en desarrollo, nunca en el Dockerfile.

## A.19 Nombres y estilo
- **Identificadores en inglés; mensajes al usuario, comentarios y documentación en español.**
- Paquetes: una palabra en minúsculas, sin guion bajo (`postgres`, `usecase`, `sheetimport`), igual que su carpeta.
- Archivos: `snake_case.go`; un caso de uso por archivo con su nombre (`deliver_requirement.go`).
- Casos de uso con verbo + sustantivo en presente (`CreateRequirement`, `TriageDefect`, `ProposeRegression`); puertos con nombre de rol (`RequirementRepository`, `FileStore`, `AuditTrail`); errores con `Err` (`ErrBlockingDefectOpen`).
- Identificadores de entidades con sufijo `Id` (`RequirementId`, `BuildId`).
- `context.Context` siempre como primer parámetro, y se propaga.
- Los constructores reciben `Deps` y validan; nunca devuelven un objeto a medio armar.

## A.20 Git y flujo de trabajo
- `main` siempre compila y pasa CI. Se trabaja en ramas cortas (`feature/ficha-requerimiento`, `fix/enlace-sheets-gid`).
- **Commits en español, en imperativo y pequeños:** «Agrega ficha de requerimiento con línea de vida».
- La descripción del PR responde dos preguntas: **¿qué se puede quitar sin perder un requisito?** (KISS) y **¿algo quedó dicho en dos lugares?** (DRY).
- Un PR que cambia una regla de este anexo o un contrato (OpenAPI, esquema) actualiza el documento en el mismo PR.

## A.21 Registro de decisiones
- Cada decisión de arquitectura nueva se agrega a `docs/decisiones.md` con número (`D1`, `D2`…), fecha, contexto, decisión y motivo. Las decisiones D1 a D10 de la sección 15 del requerimiento son las primeras entradas.
- Una decisión no se borra: si cambia, se escribe otra que la reemplaza y se marca la anterior como reemplazada.
