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

## Paquetes y regresión (fase 3)

Además de lo anterior, cada rol participa en el paquete (PKG):

- **Desarrollo** mantiene el catálogo de componentes (*Pruebas → Componentes*), registra cada **BUILD** con su checksum y su manifiesto (*Paquetes → BUILD*; un BUILD no se edita) y **arma el PKG**: lo crea y le agrega sus BUILD y sus REQ aprobados por QA. El contenido solo cambia mientras el PKG está «En armado».
- **Desarrollo y QA declaran el impacto**: en la ficha del REQ, los componentes que toca; en cada TC, los que ejercita. De ese cruce sale la regresión.
- **QA** mantiene los **ambientes** (*Ambientes*) y registra cada **despliegue** en los de prueba. Cada RUN se registra sobre un BUILD y un ambiente del catálogo.
- **QA define la regresión** en la pestaña *Regresión* del PKG: *Proponer* trae los TC que cruzan con los componentes tocados más los críticos del sistema; puedes quitar o agregar TC, y cuando te apartas de lo propuesto escribes el criterio; *Confirmar* la congela. Sin regresión confirmada el PKG no pasa a pruebas, y no queda «Listo para liberar» mientras un TC de la regresión no tenga un RUN aprobado sobre un BUILD del paquete o haya un defecto de severidad alta abierto.
- **Change manager** aprueba el PKG para producción, registra el despliegue en producción y lo marca desplegado. QA lo verifica después.

## Producción, informes y calendario (fase 4)

Lo que exige el paso a producción depende de la **ruta del PKG**, que se ve en su cabecera: la más exigente de las de sus REQ. La pestaña *Producción* del PKG lo explica y reúne su checklist, sus diagnósticos y sus pruebas de humo. En todos los casos cuenta lo último: si un diagnóstico no estuvo conforme o una prueba de humo falló, se registra otra.

- **Ruta alta** (SIMF, SGLPAR): para pasar a «Desplegado», el checklist firmado entero; para pasar a «Verificado», un diagnóstico revisado y conforme.
- **Ruta ligera** (SyPago): para pasar a «Verificado», una prueba de humo aprobada.
- **Fix menor:** no exige ninguno de los tres.

**Change manager**

1. **Publica la plantilla del checklist** de cada sistema en *Producción → Plantillas*: un paso por renglón, y la descripción después de una barra («Respaldar la base | Con pg_dump»). Una plantilla no se edita: *Nueva versión* parte de la actual. Sin plantilla no se abre ningún checklist.
2. **Abre el checklist** desde la pestaña *Producción* del PKG (*Abrir checklist*), en el ambiente de producción. Toma la última versión de la plantilla.
3. **Firma los pasos** en la ficha del checklist (*Firmar*), con un comentario si hace falta. Firman change management y ops, cada uno el paso que ejecutó, en cualquier orden. Una firma no se quita. La evidencia (salidas, capturas) se adjunta en la pestaña *Evidencia*.
4. **Mantén los hitos** en *Calendario*: *Nuevo hito* con fecha y responsable; *Marcar cumplido* cuando se cumple.

**Desarrollo y operaciones**

- **Entrega el diagnóstico** después del despliegue: pestaña *Producción* del PKG → *Entregar DIAG*, eligiendo el despliegue y escribiendo lo que se revisó. Los archivos de salida se adjuntan en la ficha del DIAG.
- **Operaciones firma** los pasos del checklist que ejecuta.

**QA**

- **Revisa el diagnóstico** en su ficha (*Revisar*): conforme o no conforme, con notas si no lo está. Se revisa una sola vez.
- **Registra la prueba de humo** desde la pestaña *Producción* del PKG (*Registrar SMOKE*): aprobada o fallida, con notas si falló.
- **Emite el informe de conformidad** desde la pestaña *Informes* del PKG. El informe congela la evidencia de ese momento: contenido, trazabilidad, ejecuciones, regresión con sus criterios, defectos abiertos y diferidos, firmas del checklist, diagnóstico y prueba de humo. Si algo cambia, *Emitir nueva versión*: la anterior queda, marcada como reemplazada.
- En la ficha del informe lo lees, ves su **huella** (SHA-256) y lo **descargas en PDF o en Markdown**. La huella es la del Markdown; el PDF la lleva en el pie de cada página. Para comprobar que un informe no cambió, calcula el SHA-256 del Markdown descargado y compáralo con la huella.

**Todos**

- *Inicio* muestra las **próximas ventanas de cambio** y las **alertas de atraso**: un hito vencido sin cumplir, o un CHG cuya ventana terminó sin liberarse. *Calendario* las pone arriba y, debajo, hitos y ventanas por fecha. Las fechas de los hitos son días en UTC. No llegan correos.
- La matriz de trazabilidad tiene la columna **RPT**: el informe vigente del paquete de cada REQ.

## Operaciones

Consultas el estado de lo que vas a desplegar, comentas y adjuntas documentos (por ejemplo, la evidencia de un despliegue). Ves los paquetes, sus BUILD, los ambientes y sus despliegues. En producción firmas los pasos del checklist y entregas diagnósticos (ver la sección de la fase 4).

## Auditoría

Consultas y exportas la auditoría (*Administración → Auditoría*): quién hizo qué, cuándo, con el valor anterior y el nuevo. **Verificar la cadena** comprueba que ningún registro fue alterado. Además lees todo lo demás; no escribes nada.

## QA admin

Además de lo de QA, configuras el sistema en *Administración*. Cada cambio vale de inmediato y queda auditado.

- **Permisos por rol:** qué puede hacer cada rol. Tu rol no puede quedarse sin el permiso de configurar.
- **Flujos:** estados y transiciones de REQ, DEF y CHG. Puedes agregar un estado, renombrarlo, moverlo en la línea de vida, y agregar o quitar transiciones con el permiso que exigen. La clave de un estado no cambia, y un estado con registros o con transiciones no se quita.
- **Reglas de bloqueo:** activar, desactivar o dejar en «solo avisa» cada condición de una transición.
- Los **usuarios y sus roles** no se administran aquí sino en Keycloak; ver [keycloak.md](keycloak.md).

## Lo que este manual no cubre todavía

La importación de la hoja actual (todavía no existe) y el correo de alertas (no hay servidor de correo).
