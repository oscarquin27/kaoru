# Correspondencia con las normas de referencia

> **Esto es una correspondencia de evidencia, no una declaración de cumplimiento.** Dice qué registro de Kaoru sirve como evidencia frente a cada tema de las normas. Kaoru no está certificado ni afirma cumplir ninguna de ellas, y tener el registro no equivale a cumplir la cláusula.

> `[SUPUESTO]` Las cláusulas y nombres de documentos se citan según las ediciones indicadas y sin tener a la vista el texto oficial. Antes de usar esta tabla en una auditoría, alguien con acceso a las normas debe verificar cada referencia.

Las columnas «Registro en Kaoru» apuntan a las tablas de [modelo-datos.md](modelo-datos.md). «Fase» dice cuándo existe el registro.

## ISO 9001:2015 — evidencia y auditoría

| Cláusula | Tema | Registro en Kaoru | Fase |
|---|---|---|---|
| 7.5.2 | Creación y actualización de información documentada (identificación, autor, fecha) | `document`, `document_version` (autor, fecha, checksum); `code` de cada entidad | 2 |
| 7.5.3 | Control de la información documentada (versiones, protección contra cambios) | Versiones que no se sobrescriben; `audit_entry` de solo inserción y encadenada por hash | 2 |
| 8.2.3 | Revisión de los requisitos | `requirement` (criterios de aceptación), `risk_assessment`, transiciones del REQ en la auditoría | 2 |
| 8.5.2 | Identificación y trazabilidad | Matriz REQ → RISK → UC → TC → RUN → DEF; IDs inmutables | 2 (PKG y RPT: 3 y 4) |
| 8.5.6 | Control de los cambios (quién autorizó, qué se revisó) | `change`, su ruta y sus transiciones con actor, fecha y motivo | 2 |
| 8.6 | Liberación (evidencia de conformidad y quién autorizó) | RUN aprobados por BUILD; reglas de bloqueo evaluadas; informe de conformidad RPT; firmas del checklist | 2 (RUN), 4 (RPT, CHK) |
| 8.7 | Control de las salidas no conformes | `defect` con severidad y estado; diferidos con quién aceptó y por qué | 2 |
| 9.1 | Seguimiento y medición | Cobertura por REQ, defectos por estado y tiempo en estado, tamaño de regresión | 2 a 4 |
| 9.2 | Auditoría interna | Rol `auditor` de solo lectura; consulta, exportación y verificación de la cadena de auditoría | 2 |

## ISO/IEC/IEEE 29119-3 — documentación de pruebas

Edición de referencia: 2013 `[SUPUESTO: confirmar si se usa la de 2021, que reagrupa algunos documentos]`. Se usan los nombres de sus documentos, no números de cláusula. La norma permite adaptarlos; lo que no aplica se omite.

| Documento de la norma | Registro en Kaoru | Fase |
|---|---|---|
| Plan de pruebas (alcance, riesgos, enfoque) | `change` con su ruta + `risk_assessment` de sus REQ | 2 |
| Especificación de diseño de pruebas | `functional_case` (UC-) y su relación con el REQ | 2 |
| Especificación de casos de prueba | `test_case` (precondiciones, resultado esperado, criticidad) | 2 |
| Especificación de procedimientos de prueba | `test_case.steps` | 2 |
| Requisitos de datos de prueba | Ítem «Datos de prueba» del contrato de entrada | 2 |
| Requisitos e informe de disponibilidad del ambiente | ENV- y DEP- | 3 |
| Registro de ejecución y resultados | `test_run` (TC, BUILD, ambiente, resultado, quién, cuándo) | 2 |
| Informe de incidentes | `defect` | 2 |
| Informe de estado de pruebas | Tablero e indicadores de la matriz | 2 |
| Informe de finalización de pruebas | Informe de conformidad RPT- | 4 |
| Política y estrategia de pruebas de la organización | Fuera de Kaoru: es un documento del departamento. Se puede adjuntar como DOC-. | — |

Los EXP- (pruebas exploratorias) se registran, pero no cuentan como evidencia de conformidad.

## ISO/IEC 25010 — modelo de calidad

Edición de referencia: 2011, con ocho características `[SUPUESTO: la edición de 2023 cambia la lista; confirmar cuál se adopta]`. Cada RISK indica cuáles están en juego (`risk_quality_characteristic`).

| Característica | Valor en Kaoru |
|---|---|
| Adecuación funcional | `functional_suitability` |
| Eficiencia de desempeño | `performance_efficiency` |
| Compatibilidad | `compatibility` |
| Usabilidad | `usability` |
| Fiabilidad | `reliability` |
| Seguridad | `security` |
| Mantenibilidad | `maintainability` |
| Portabilidad | `portability` |

## ISO/IEC 20000-1:2018 e ITIL 4 — gestión de cambios

| Referencia | Tema | Registro en Kaoru | Fase |
|---|---|---|---|
| ISO/IEC 20000-1 §8.5.1; ITIL 4, habilitación de cambios | Cada cambio se registra, se clasifica, se evalúa su riesgo y se aprueba | `change` (ruta alta, ligera o fix menor), `risk_assessment`, aprobación en la auditoría | 2 |
| ISO/IEC 20000-1 §8.5.3; ITIL 4, gestión de liberaciones y de despliegues | Qué se libera, dónde, quién y con qué resultado | PKG-, DEP-, checklist de producción firmado, DIAG- y SMOKE- | 3 y 4 |
| ITIL 4, validación y pruebas del servicio | Criterios de aceptación probados antes de liberar | Cobertura por REQ y reglas de bloqueo | 2 |
| ISO/IEC 20000-1 §8.2.6; ITIL 4, gestión de la configuración | Qué componentes existen y qué versión hay en cada ambiente | COMP-, BUILD-, ENV- | 3 |

El CHG- viaja por todo el flujo: desde el REQ hasta el informe de conformidad.

## Lo que Kaoru no cubre

- El sistema de gestión de la calidad en sí (política, objetivos, revisión por la dirección, competencia del personal).
- La ejecución de pruebas automatizadas: Kaoru registra resultados, no los produce.
- La gestión del código fuente y el despliegue automático a producción.
