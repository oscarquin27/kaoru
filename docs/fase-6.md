# Fase 6 — Planificación: encargo para quien la construya

Este documento es el punto de partida de la fase 6. Dice qué se pide, cómo encaja con lo que existe, el diseño y lo que falta decidir. Lo propuesto va marcado `[SUPUESTO]`; no es una decisión hasta que el coordinador de QA y change management lo revisen.

- **Requerimiento:** [kaoru-requerimiento-v2.md](kaoru-requerimiento-v2.md). Kaoru es un «Sistema de Planificación, Trazabilidad y Pruebas QA» (título y §1): hasta la fase 5 la planificación se limitaba a ventanas de CHG e hitos.
- **Reglas del repositorio:** [reglas.md](reglas.md). **Decisiones:** [decisiones.md](decisiones.md) (D1 a D53).
- **Cómo se trabaja, comandos y tropiezos:** los de [fase-5.md](fase-5.md) §3 siguen valiendo. Para levantar todo: `make demo` (ver [README](../README.md)).

## 1. Qué se pide

El coordinador de QA quiere dejar la parte operativa y usar Kaoru como su panel de control. Para eso necesita, en sus palabras:

1. **Tener reservado el período** de certificación de un proyecto **antes** de que entre en certificación.
2. **Ver qué están haciendo sus operadores**: quién tiene qué actividad y en qué fechas.
3. **Saber si alguna actividad se va a solapar** con otra del mismo operador.
4. **Reajustar las fechas** y ver el efecto.

Qué hay hoy (revisión del 14 nov 2026):

| Necesidad | Hoy | Falta |
|---|---|---|
| Fechas | Ventana de cambio del CHG (inicio y fin) e hitos con fecha y responsable. REQ, PKG y DEF no tienen fecha planificada. | El período de certificación de un CHG y las actividades de cada persona con sus fechas. |
| Reajustar | Ventanas e hitos se editan y se auditan, pero solo change management (`chg.edit`, `milestone.manage`). | Que el coordinador (`qa-admin`) pueda planificar. |
| Ver solapes | Nada. El calendario es una lista por fecha. | Detección de solapes por persona y una vista en el tiempo. |
| Asignar | Responsable del REQ, asignado del DEF, responsable del hito. | Asignar actividades con fechas a uno o varios operadores, y la carga de cada uno. |

Decisiones del coordinador al pedir esta fase (14 nov 2026):

- **El proyecto es el CHG.** El PKG no planifica fechas propias.
- **Solape = mismos días.** Dos actividades del mismo operador se solapan si sus fechas se cruzan al menos un día. No hay porcentaje de dedicación.
- **Al pasar a certificación sin período reservado ni operadores, Kaoru avisa** (regla de bloqueo en «solo avisa», editable en Administración). No bloquea.

## 2. Diseño `[SUPUESTO]`

### 2.1 Conceptos

- **Actividad (ACT-AAAA-NNNN):** un bloque de trabajo con fecha de inicio y de fin (días UTC, ambos incluidos), título, notas, una **clase** y cero o más **operadores**. Clases: `certification` (Certificación), `regression` (Regresión), `production` (Pase a producción) y `other` (Otra). La de certificación exige un CHG; las demás lo admiten.
- **Período de certificación de un CHG:** sus actividades de certificación no canceladas. Puede haber varias (rondas).
- **Operador:** cualquier usuario de Kaoru. Solo se puede asignar a quien ya entró una vez (D34).
- **Estado de una actividad:** se deduce de las fechas: *planificada* (empieza después de hoy), *en curso* (hoy está dentro) y *terminada* (terminó antes de hoy). Además, *cancelada*: no se borra, se cancela, y deja de contar para solapes y para la regla. Volver a activarla es posible.
- **Solape:** dos actividades no canceladas con un operador en común cuyas fechas se cruzan. Se informa con el operador, las dos actividades y los días que se cruzan. **No bloquea:** se avisa al guardar y se ve en la planificación.
- **Proyectos que se cruzan:** la vista de planificación muestra los períodos de certificación de todos los CHG en el tiempo, uno debajo de otro, para ver a simple vista qué proyectos coinciden. Kaoru no los marca como conflicto: dos proyectos a la vez son normales; el problema es el operador en dos sitios.

### 2.2 Regla al entrar en certificación

Nueva regla `certification_planned` en la transición del REQ **Entregado a QA → En pruebas**, en «solo avisa». Se cumple si el CHG del REQ tiene una actividad de certificación no cancelada, con al menos un operador, cuyo período incluye hoy. Si no se cumple, dice por qué:

- el CHG no tiene período de certificación planificado;
- la certificación no tiene operadores asignados;
- hoy está fuera del período (del … al …).

Un REQ sin CHG la cumple: no hay proyecto que planificar.

### 2.3 Dónde vive

En el feature `planning` (que ya tiene hitos y calendario): tablas `activity` y `activity_operator`, migración `00018`. `requirement` pide el estado de la certificación por un puerto, conectado en `bootstrap` como las demás fuentes de hechos (D36). `planning` sigue leyendo los CHG a través de los casos de uso de `change`.

Permiso nuevo `activity.manage` («Planificar actividades»), para `qa-admin` y `change-manager`. Leer no exige permiso. Crear y cada campo editado se auditan; los operadores se auditan como la lista anterior y la nueva.

### 2.4 Contrato

| Ruta | Para |
|---|---|
| `GET /api/activities` | Lista con filtros: operador, CHG, clase, estado y fechas. |
| `POST /api/activities` | Crear; la respuesta trae los solapes que produce. |
| `GET /api/activities/{code}` | Ficha con sus solapes. |
| `PATCH /api/activities/{code}` | Editar fechas, título, notas, clase, operadores; cancelar o reactivar. |
| `GET /api/planning/schedule` | La planificación de un rango: por operador, sus actividades y solapes; las actividades sin operador; los períodos de certificación por CHG; las ventanas de cambio y los hitos del rango. Por omisión, seis semanas desde el lunes de esta semana; como máximo 92 días. |

### 2.5 Front

- La sección **Calendario** pasa a llamarse **Planificación**, con tres pantallas: *Por operador* (línea de tiempo), *Actividades* (lista) y *Calendario* (la de la fase 4).
- **Por operador:** una fila por operador y una columna por día, con las actividades como barras con su código escrito; los días con solape se marcan con un icono y el número de actividades (el color no es la única señal). Arriba, la lista de solapes; abajo, los proyectos (períodos de certificación por CHG), las ventanas de cambio y los hitos. Se mueve por semanas y se filtra por operador. Sin librería de gráficos: grilla CSS dentro de `table-wrap`, con desplazamiento horizontal en el teléfono.
- **Ficha de la actividad:** datos, operadores, solapes, edición y actividad (auditoría).
- **Ficha del CHG:** sección *Certificación* con sus períodos y operadores, y *Planificar certificación* con el CHG ya elegido. La ficha pasa a `web/src/app/` porque combina dos features.
- **Inicio:** *Tus actividades* (las próximas de quien entra) y, para quien planifica, cuántos solapes hay en las próximas seis semanas.

## 3. Fuera de esta entrega

- Horas o porcentaje de dedicación, feriados y días hábiles (no hay calendario laboral en Kaoru).
- Avisos por correo (no hay SMTP).
- Arrastrar barras para cambiar fechas: se editan en el formulario.
- La regla en el PKG: un PKG puede mezclar REQ de varios CHG.
- Fechas planificadas en REQ, PKG o DEF.

## 4. Preguntas abiertas

| # | Pregunta | Supuesto para avanzar |
|---|---|---|
| 1 | ¿Las clases de actividad (certificación, regresión, pase a producción, otra) alcanzan? | Sí; se agregan otras si aparecen. |
| 2 | ¿El período de certificación debe caber en la ventana del CHG o terminar antes de ella? | No se valida: se ven juntos en la línea de tiempo. |
| 3 | ¿Quién más planifica, además del coordinador y change management? | Nadie: `activity.manage` es de `qa-admin` y `change-manager`, editable en Administración. |
| 4 | ¿Un operador ve la planificación de todos o solo la suya? | La de todos: leer no exige permiso en Kaoru. |

## 5. Qué significa «terminado»

1. `make check` y `make check-offline` pasan, con `GET /api/planning/schedule` en `make check-perf`.
2. Lo nuevo se probó en el navegador contra la API real, y lo que no, se dice.
3. `docs/decisiones.md`, `docs/flujos.md`, `docs/manual-usuario.md` y `docs/modelo-datos.md` al día.
4. La entrega separa Decisiones, Supuestos y Pendientes.
