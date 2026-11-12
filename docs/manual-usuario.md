# Manual de usuario de Kaoru, por rol

Lo que cada rol hace en Kaoru con la configuración inicial del MVP. Quién puede hacer cada acción es configuración: QA admin la cambia en **Administración → Permisos por rol**, así que tu instalación puede diferir.

## Lo que vale para todos

- **Entrar.** Abre Kaoru en Chrome y entra con tu usuario. La sesión dura la jornada (10 horas); al cerrar sesión se cierra también en Keycloak. Tu rol te lo asigna un administrador en Keycloak.
- **Buscar.** `Ctrl+K` busca por ID (`REQ-2026-0042`) o por texto en cualquier pantalla.
- **Los ID son enlaces.** Un clic abre la ficha; el enlace de cualquier pantalla se puede copiar y compartir.
- **Filtros y vistas guardadas.** Cada lista se filtra, y los filtros quedan en la URL. Con **Guardar vista** les pones nombre para volver a ellos; las vistas son tuyas, nadie más las ve.
- **La ficha del REQ** es la pantalla central: arriba el estado y la línea de vida, debajo **«Qué falta para avanzar»** (lo que bloquea y lo que solo avisa) y las pestañas de enlaces, documentación, trazabilidad, pruebas, defectos y actividad.
- **Si un botón no aparece**, tu rol no tiene ese permiso. Si una transición se rechaza, el mensaje dice qué falta.
- **Todo queda registrado.** Cada alta, cambio y transición guarda quién lo hizo y cuándo. Nada de eso se edita ni se borra.
- **Leer no exige permiso:** cualquier persona con sesión ve los REQ, las pruebas, los defectos y los documentos.

## Change manager

Abres y cierras el ciclo de un cambio.

1. **Registra el CHG** en *Cambios → Nuevo CHG*: título y sistema. La ruta (alta o ligera) sale del sistema.
2. **Registra sus REQ** en *Requerimientos → Nuevo REQ*, con su CHG y su responsable. En el selector de responsable aparece quien ya entró a Kaoru alguna vez.
3. **Mueve el CHG**: aprobar, poner en curso, liberar y cerrar. Rechazar pide un motivo.
4. **Decide las excepciones**: aceptar que un REQ o un DEF se difiera, rechazar un REQ, marcar un REQ como liberado y cerrarlo. Diferir y rechazar piden motivo.
5. **Consulta la auditoría** en *Administración → Auditoría* y expórtala a CSV.

## QA

Evalúas el riesgo, diseñas las pruebas, las ejecutas y decides si el REQ pasa.

1. **Evalúa el riesgo** en la ficha del REQ (*Evaluar riesgo*): nivel, justificación y características de calidad. El nivel fija la cobertura que se exige. Sin riesgo evaluado, el REQ no avanza.
2. **Diseña** en la pestaña *Pruebas*: casos de uso (UC) y casos de prueba (TC) enlazados a ellos. Pasa el REQ a *Diseño de pruebas*.
3. **Ejecuta.** Cuando dev entrega, pasa el REQ a *En pruebas* y registra cada RUN: TC, versión y checksum del build, ambiente y resultado. Un RUN no se corrige: si te equivocaste, registras otro.
4. **Explora.** Una exploración (EXP) deja lo que probaste fuera de guion. Un hallazgo de una exploración se reproduce en un RUN antes de abrirlo como defecto.
5. **Abre defectos** desde el RUN que falló, con su severidad. Luego los trías, los cierras tras la reprueba o los reabres.
6. **Aprueba.** *Aprobado QA* exige la cobertura mínima de TC con RUN aprobado y que no haya defectos abiertos de severidad alta. La ficha te dice exactamente qué TC y qué DEF faltan.
7. **Adjunta evidencia** en *Documentación*. Si un enlace a Google Sheets importa como evidencia, adjunta su exportación (PDF o XLSX) como instantánea: para el informe cuenta la instantánea, no el enlace vivo.

## Desarrollo

Entregas a QA con el contrato de entrada completo y corriges defectos.

1. **Pasa el REQ a *En desarrollo*** cuando empieces.
2. **Completa el contrato de entrada** en la ficha: cada ítem que marcas queda con tu nombre y la fecha. *Entregado a QA* no se alcanza con ítems sin marcar, y la ficha dice cuál falta.
3. **Agrega enlaces y documentos**: la especificación en Google Sheets (pega la URL; Kaoru reconoce la hoja, la pestaña y el rango), manuales y notas.
4. **Corrige defectos**: los que tienes asignados salen en *Inicio*. Al terminar, déjalos *Listos para reprueba*.
5. **Incluye el REQ en un paquete** cuando QA lo aprobó.

## Operaciones

Consultas el estado de lo que vas a desplegar, comentas y adjuntas documentos (por ejemplo, la evidencia de un despliegue). Las pantallas de paquetes, ambientes y producción llegan en fases posteriores.

## Auditoría

Consultas y exportas la auditoría (*Administración → Auditoría*): quién hizo qué, cuándo, con el valor anterior y el nuevo. **Verificar la cadena** comprueba que ningún registro fue alterado. Además lees todo lo demás; no escribes nada.

## QA admin

Además de lo de QA, configuras el sistema en *Administración*. Cada cambio vale de inmediato y queda auditado.

- **Permisos por rol:** qué puede hacer cada rol. Tu rol no puede quedarse sin el permiso de configurar.
- **Flujos:** estados y transiciones de REQ, DEF y CHG. Puedes agregar un estado, renombrarlo, moverlo en la línea de vida, y agregar o quitar transiciones con el permiso que exigen. La clave de un estado no cambia, y un estado con registros o con transiciones no se quita.
- **Reglas de bloqueo:** activar, desactivar o dejar en «solo avisa» cada condición de una transición.
- Los **usuarios y sus roles** no se administran aquí sino en Keycloak; ver [keycloak.md](keycloak.md).

## Lo que este manual no cubre todavía

La importación de la hoja actual, paquetes, ambientes, producción, informes y calendario: no están en el MVP.
