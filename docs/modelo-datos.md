# Modelo de datos

La fuente única del esquema son las migraciones de [`src/migrations/`](../src/migrations). Este documento las explica; si difieren, mandan las migraciones.

El esquema cubre lo que construye la fase 2 (MVP). Las entidades de las fases 3 y 4 aparecen al final como previstas y no tienen tablas todavía (reglas §A.1.1: solo lo que pide la fase).

## Convenciones

- Una base, un esquema (`kaoru`). Tablas en singular y snake_case.
- Llave técnica `uuid` (UUIDv7, generada por la aplicación). La excepción es `app_user.id`, que es el `sub` de Keycloak.
- Cada entidad con ID legible tiene además `code` único e inmutable (`REQ-2026-0042`), que sale de `code_sequence` (prefijo + año).
- Valores fijos como `text` con `CHECK`, en inglés. Los estados no son `CHECK`: son filas de `workflow_state`.
- `timestamptz` en UTC. Las tablas inmutables solo llevan `created_at`.
- Lo que tiene historia no se borra: llaves foráneas con `ON DELETE RESTRICT`.

## Roles de base de datos

| Rol | Uso | Puede |
|---|---|---|
| `kaoru_owner` | Aplica las migraciones al arrancar. | Todo dentro del esquema `kaoru`. |
| `kaoru_app` | Conexión de ejecución de `kaoru-api`. | `SELECT`, `INSERT`, `UPDATE`, `DELETE` sobre las tablas, salvo las inmutables. No cambia el esquema. |

Tablas de solo inserción para `kaoru_app` (sin `UPDATE` ni `DELETE`): `audit_entry`, `test_run`, `document_version` y `comment`. Lo comprueba [`tools/check-migrations.sh`](../tools/check-migrations.sh).

Los crea [`infra/postgres/init.sql`](../infra/postgres/init.sql).

## Diagrama (fase 2)

```mermaid
erDiagram
    system_under_test ||--o{ change : "tiene"
    system_under_test ||--o{ requirement : "tiene"
    system_under_test ||--o{ test_case : "tiene"
    change ||--o{ requirement : "agrupa"
    workflow_state ||--o{ change : "estado"
    workflow_state ||--o{ requirement : "estado"
    workflow_state ||--o{ defect : "estado"
    workflow_state ||--o{ workflow_transition : "desde / hacia"
    permission ||--o{ workflow_transition : "exige"
    permission ||--o{ role_permission : "asignado a rol"
    workflow_transition ||--o{ blocking_rule : "condiciona"
    requirement ||--o| risk_assessment : "RISK vigente"
    risk_assessment ||--o{ risk_quality_characteristic : "ISO 25010"
    requirement ||--o{ requirement_link : "enlaces"
    document ||--o{ requirement_link : "instantánea"
    requirement ||--o{ requirement_entry_check : "contrato de entrada"
    entry_contract_item ||--o{ requirement_entry_check : "ítem"
    requirement ||--o{ functional_case : "UC"
    functional_case ||--o{ test_case_functional_case : ""
    test_case ||--o{ test_case_functional_case : "valida"
    test_case ||--o{ test_run : "RUN"
    requirement ||--o{ exploratory_session : "EXP"
    requirement ||--o{ defect : "DEF"
    test_run ||--o{ defect : "reproducido en"
    document ||--o{ document_version : "versiones"
    document ||--o{ document_link : "enlazado a"
    app_user ||--o{ audit_entry : "actor"

    requirement {
        uuid id PK
        text code UK
        text title
        text acceptance_criteria
        uuid system_id FK
        uuid change_id FK "opcional"
        uuid state_id FK
        text owner_id FK "responsable actual"
    }
    change {
        uuid id PK
        text code UK
        uuid system_id FK
        text route "high, light, minor_fix"
        uuid state_id FK
        timestamptz window_start
        timestamptz window_end
    }
    risk_assessment {
        uuid id PK
        text code UK
        uuid requirement_id FK "único"
        text level "high, medium, low"
        text justification
    }
    requirement_link {
        uuid id PK
        text kind "google_sheets, youtrack…"
        text url
        text sheet_id
        text sheet_gid
        text sheet_range
        boolean is_official
        uuid snapshot_document_id FK
    }
    test_case {
        uuid id PK
        text code UK
        text steps
        text expected_result
        boolean is_critical
        timestamptz retired_at
    }
    test_run {
        uuid id PK
        text code UK
        uuid test_case_id FK
        text build_version
        text build_checksum "SHA-256"
        text environment_name
        text result "passed, failed, blocked"
    }
    defect {
        uuid id PK
        text code UK
        uuid requirement_id FK
        uuid test_run_id FK
        text severity "high, medium, low"
        uuid state_id FK
    }
    document_version {
        uuid id PK
        uuid document_id FK
        integer version
        text media_type
        text checksum "SHA-256"
    }
    audit_entry {
        uuid id PK
        bigint seq UK
        text actor_id FK
        text entity_type
        uuid entity_id
        text action
        text field
        text old_value
        text new_value
        text reason
        text prev_hash
        text hash UK
    }
    workflow_state {
        uuid id PK
        text entity_type
        text key
        text name
        integer position "NULL: estado lateral"
    }
    blocking_rule {
        uuid id PK
        uuid transition_id FK
        text rule_key
        jsonb params
        text route "NULL: todas"
        boolean blocks "false: solo avisa"
    }
```

`document_link` y `comment` apuntan a cualquier entidad con `entity_type` + `entity_id`, sin llave foránea; el caso de uso comprueba que la entidad existe.

## Tablas por feature

| Feature | Tablas | Migración |
|---|---|---|
| `user` | `app_user` | 00001 |
| (compartido) | `system_under_test`, `code_sequence` | 00001 |
| `workflow` | `workflow_state`, `workflow_transition`, `permission`, `role_permission`, `blocking_rule` | 00002, 00011 |
| `audit` | `audit_entry` | 00003 |
| `change` | `change` | 00004 |
| `document` | `document`, `document_version`, `document_link` | 00005 |
| `requirement` | `requirement`, `requirement_link`, `entry_contract_item`, `requirement_entry_check` | 00006 |
| `comment` (propuesto, D23) | `comment` | 00006 |
| `risk` | `risk_assessment`, `risk_quality_characteristic` | 00007 |
| `testdesign` | `functional_case`, `test_case`, `test_case_functional_case` | 00008 |
| `execution` | `test_run`, `exploratory_session` | 00009 |
| `defect` | `defect` | 00010 |
| `traceability` | Ninguna: solo lee. | — |

## Cómo responde el modelo a las preguntas del producto

- **Matriz (regla de negocio 1):** `requirement` → `risk_assessment` → `functional_case` → `test_case_functional_case` → `test_case` → último `test_run` → `defect`. Es una sola consulta de `traceability`.
- **Tiempo en cada estado:** se calcula al leer, con las filas de `audit_entry` donde `action = 'transition'`. No hay una tabla aparte de historial de estados.
- **Cobertura:** TC del REQ cuyo último RUN es `passed` sobre el total de TC activos del REQ.
- **Qué falta para avanzar:** las `blocking_rule` de las transiciones que salen del estado actual, evaluadas por el dominio.
- **Cadena de auditoría:** `hash = SHA-256(prev_hash + contenido de la fila)`. El orden lo da `seq`. La inserción se serializa con un bloqueo para que dos escrituras no lean el mismo `prev_hash`.

## Previsto para fases posteriores (sin tablas todavía)

| Fase | Entidades | Cambio sobre lo existente |
|---|---|---|
| 3 | COMP-, BUILD-, PKG-, ENV-, DEP-, selección de regresión | `test_run` y `exploratory_session` cambian `build_version`, `build_checksum` y `environment_name` por llaves foráneas a BUILD y ENV. REQ y TC declaran componentes. Se agrega el flujo del PKG a `workflow_state`. |
| 4 | CHK- con firmas, DIAG-, SMOKE-, RPT-, hitos y ventanas | Las ventanas del CHG pasan de `change.window_*` al calendario. |

## Pendiente

- **Vistas guardadas (§7.3):** tabla `saved_view` (dueño, lista, nombre, filtros). Son personales y no se auditan; compartir sigue siendo por URL (decisión D46).
- **Importación de la hoja:** sin tablas hasta conocer su formato (pregunta 5).
- **Índices de búsqueda de texto:** no se agregan sin una medición (reglas §A.1.1). El objetivo de menos de 1 s con 10 000 REQ y 100 000 RUN se mide en la fase 2 con datos generados.
