# Kaoru — Sistema de Planificación, Trazabilidad y Pruebas QA (Requerimiento v2)

> **Rol que debes asumir:** arquitecto de software senior y diseñador de producto. Diseña y construye un sistema web para el departamento de QA de una empresa pequeña del sector bancario, con cambios frecuentes y ambientes heterogéneos.
>
> **Forma de trabajo:** por fases con hitos verificables. No avances a la siguiente fase sin cumplir el criterio de salida de la actual. En cada entrega separa siempre: **Decisiones** (lo que se resolvió), **Supuestos** (marcados `[SUPUESTO]`, lo que asumiste sin confirmación) y **Pendientes** (lo que falta y quién debe resolverlo). No inventes datos del negocio: si falta algo, márcalo `[POR DEFINIR]` y pregúntalo.

## 0. Instrucciones para el agente que construye el sistema

1. **Antes de escribir código**, lee todo este documento, revisa la sección 15 (decisiones ya tomadas) y haz solo las preguntas de la sección 14 que siguen abiertas. No vuelvas a preguntar lo que ya está decidido.
2. **Trabaja en un repositorio git** que siga el **Anexo A (Reglas del repositorio)**: árbol de carpetas, capas, errores, persistencia, pruebas, calidad automática y flujo de git. Tu primer commit copia el Anexo A a `docs/reglas.md`; desde ahí, ese archivo es la fuente única y este documento lo enlaza en lugar de repetirlo.
3. **El Anexo A manda sobre los detalles técnicos.** Si algo de este documento choca con el Anexo A, señálalo y propone cuál se cambia; no elijas en silencio.
4. **Registra cada decisión de arquitectura** en `docs/decisiones.md` con número, fecha, contexto, decisión y motivo (Anexo A §A.21).
5. **Antes de dar por terminada cualquier tarea**, ejecuta `make check` (las mismas comprobaciones de CI del Anexo A §A.18) y el build de Docker. No reportes algo como hecho si no lo verificaste; si algo falla o no se pudo probar, dilo explícitamente.
6. **Al cerrar cada fase**, entrega el informe de la sección 13 y espera la aprobación del criterio de salida antes de continuar.
7. **Si encuentras una contradicción** en este documento, no la resuelvas en silencio: señálala, propone una opción y márcala `[SUPUESTO]` hasta que se confirme.

---

## 1. Visión del producto

**Nombre: Kaoru.** En honor a Kaoru Ishikawa, que defendió que la calidad es responsabilidad de toda la organización y no solo de un departamento: el mismo lenguaje común entre desarrollo y QA que busca este sistema. **Lema:** *Trazabilidad a prueba de errores*, por la filosofía poka-yoke que guía sus reglas de bloqueo.

Kaoru es un único lugar donde desarrollo, QA y change management planifican, prueban y dejan evidencia de cada cambio. El sistema reemplaza la hoja de cálculo actual y responde, en segundos y sin buscar en varios lugares:

- **¿Dónde está este requerimiento?** Su estado actual, qué falta para avanzar y quién lo tiene.
- **¿Qué se probó, sobre qué BUILD, en qué ambiente, quién y cuándo?**
- **¿Qué cambió y qué podría romperse?** Regresión propuesta por impacto.
- **¿Está listo para liberar?** Reglas de bloqueo explícitas y visibles.
- **¿Dónde está la documentación?** Enlaces y documentos del requerimiento visibles dentro del sistema.

**Meta de calidad:** robustez basada en buenas prácticas y estándares reconocidos. El objetivo no es prometer "cero fallos" (ningún proceso lo garantiza), sino **reducir el riesgo de forma medible**: ningún cambio sin prueba proporcional a su riesgo, ninguna liberación con defectos bloqueantes, ninguna evidencia perdida.

**Principios:**
1. Flexibilidad con estándares, no burocracia. El proceso pesa según el riesgo.
2. Un lenguaje común entre desarrollo y QA (vocabulario de ISO/IEC/IEEE 29119).
3. La evidencia se produce sola como efecto de trabajar en el sistema, no como tarea adicional.
4. Todo funciona offline.
5. Interfaz que se entiende sin manual: cualquier persona ubica un requerimiento y su estado en menos de 3 clics o con una búsqueda.

**Fuera de alcance (por ahora):** ejecutar pruebas automatizadas (el sistema registra resultados, no es un runner de CI), gestión de código fuente, despliegue automático a producción.

---

## 2. Contexto

- **Sistemas bajo prueba:**
  - **SIMF y SGLPAR:** alto riesgo, alcance nacional, muchos microservicios y componentes. Alrededor de 8 controles de cambio por año entre ambos; SIMF tiende a acumular muchos cambios por liberación.
  - **SyPago:** producto propio, cambios continuos.
- **Stack de los sistemas bajo prueba:** principalmente Go, Kafka y PostgreSQL, con componentes en otras tecnologías (por ejemplo, SGLPAR tiene una API REST en .NET). El catálogo de componentes y la ingesta de resultados de pruebas deben ser **agnósticos de tecnología**.
- **Ambientes:** servidores bancarios offline (Windows y Linux) y algunos servidores en AWS. Las herramientas no pueden depender de servicios en la nube.
- **Situación actual:** el proceso vive en una hoja de cálculo. La documentación de los requerimientos vive en **Google Sheets** y seguirá ahí: el sistema la enlaza, no la reemplaza. Las pruebas funcionales y de regresión son manuales, los casos no siempre cubren lo que cambió y se pierden cambios. La instalación (DLLs en Windows, scripts `.sh` en Linux) la hace a veces dev y a veces QA, sin estándar.
- **Regulación:** no hay regulación vigente que lo exija. Se quiere estar preparado para auditorías ISO **sin afirmar certificación ni cumplimiento**.
- **Capacidad del equipo:** coordinador de QA y un recurso de QA, más el apoyo de desarrollo que se acuerde. Horas por semana: `[POR DEFINIR]`. Las fechas de la sección 12 asumen una capacidad que debe confirmarse en la fase 0.

---

## 3. Stack técnico del sistema

| Capa | Tecnología | Notas |
|---|---|---|
| Backend | **Go** con **gin** | API REST JSON. Un solo binario estático. Arquitectura por features (Anexo A). |
| Frontend | **React + TypeScript + Vite**, TanStack Query | Servido por Nginx como archivos estáticos. Cero CDN: fuentes, íconos y librerías empaquetados. Sin librería de componentes ni Tailwind (Anexo A §A.16). |
| Base de datos | **PostgreSQL** | Migraciones con **goose**, embebidas y aplicadas al arrancar; consultas con **sqlc** (Anexo A §A.11). |
| Identidad | **Keycloak dedicado** + **oauth2-proxy** + **Nginx** | Sesión por cookie; ningún token en el navegador. Ver sección 4. |
| Documentos | Volumen local con checksum SHA-256 | Interfaz de almacenamiento intercambiable, para poder usar S3 compatible (por ejemplo MinIO) más adelante sin reescribir. |
| Despliegue | **Docker + Docker Compose** | Servicios: `nginx` (TLS y front), `oauth2-proxy`, `kaoru-api` (Go), `postgres` (base de la app y base de Keycloak, separadas), `keycloak`, y opcionalmente `libreoffice` para previsualizar Office. |

**Despliegue offline con Docker:**
- Las imágenes se construyen donde hay internet y se trasladan al servidor offline con `docker save` / `docker load`. Entrega un script que genere el paquete (imágenes + `docker-compose.yml` + `.env.example` + realm de Keycloak) y otro que lo instale.
- Versiones de imagen fijadas por tag y digest; nada de `latest`.
- Datos persistentes en volúmenes nombrados: base de datos, documentos y datos de Keycloak.
- Healthchecks en todos los servicios y orden de arranque con `depends_on: condition: service_healthy`.
- TLS con certificado interno del banco o autofirmado, configurable `[POR DEFINIR]`.

**Reglas técnicas:**
- Sin dependencias externas en tiempo de ejecución. Las dependencias de compilación se versionan (`go.mod`, `package-lock.json`) y se pueden vendorizar para compilar sin internet.
- Configuración por archivo y variables de entorno, nunca valores fijos en el código.
- Respaldo: script de `pg_dump` y copia del directorio de documentos, con procedimiento de restauración probado.
- Logs estructurados (JSON) y endpoint `/healthz`.

---

## 4. Autenticación y autorización con Keycloak

- **Keycloak dedicado:** instancia propia en el mismo Docker Compose, solo para este sistema. Usa una base de datos PostgreSQL separada (otra base o esquema, nunca la H2 embebida en producción).
- **Login único (SSO):** un realm dedicado `kaoru`. La aplicación nunca guarda contraseñas.
- **Realm como código:** el realm (clientes, roles, tema, políticas de contraseña) se exporta a JSON, se versiona en `/deploy/keycloak/` y se importa al arrancar. Una instalación limpia queda lista sin configurar nada a mano.
- **Login master:** un rol `qa-admin` con control total de la configuración del sistema. La administración de Keycloak queda en el realm `master` y solo la usan los administradores designados. Documenta un procedimiento de acceso de emergencia (break-glass) que quede auditado.
- **Usuarios locales** en Keycloak, con política de contraseñas, bloqueo por intentos fallidos y OTP opcional. La federación con Active Directory o LDAP queda como opción futura.
- **Tema de login personalizado** en Keycloak con el mismo estilo visual de la aplicación (sección 7), empaquetado sin recursos externos.
- **Roles base** (realm roles, mapeados a permisos de la aplicación):

| Rol | Puede |
|---|---|
| `change-manager` | Crear y aprobar CHG, aceptar traslado de defectos, aprobar PKG para producción, ejecutar y firmar checklist de producción. |
| `qa-admin` | Todo lo de QA, más configurar flujos, reglas de bloqueo, catálogos, permisos y plantillas. |
| `qa` | Definir RISK, UC, TC; registrar RUN, EXP, DEF, SMOKE; desplegar en ambientes de prueba; generar informes. |
| `dev` | Registrar entregas (BUILD, manifiesto, unitarias), comentar y corregir DEF, aportar DIAG. |
| `ops` | Ejecutar y firmar pasos del checklist de producción, aportar DIAG. |
| `auditor` (solo lectura) | Ver todo, incluida la auditoría, sin modificar nada. |

- **Permisos como configuración:** cada acción (por ejemplo `pkg.build`, `def.defer`, `run.create`) es un permiso asignable a roles desde la interfaz de administración. Así se resuelve "quién construye el PKG" (dev o devops) sin tocar código.
- **Perímetro y sesión:** Nginx + oauth2-proxy exigen sesión con Keycloak; la sesión es una cookie `SameSite=Lax` y **el navegador nunca ve un token**. Nginx inyecta el token al backend y sobrescribe la cabecera para que el navegador no pueda falsificarla. Si la API responde 401, el front redirige al inicio de sesión. Cierre por inactividad configurable en Keycloak.
- **Validación en backend:** aunque venga por el perímetro, el backend verifica el token (firma con JWKS en caché, emisor, expiración, cliente emisor y tipo). Los permisos se comprueban en el **caso de uso**, no en el handler. El frontend solo oculta botones; la seguridad real está en el backend. Detalle en Anexo A §A.13.

---

## 5. Estándares de referencia (adaptados, no impuestos)

No implementes un estándar completo al pie de la letra. Úsalos como marco:

- **ISO/IEC/IEEE 29119:** vocabulario común de pruebas. La parte 3 define la documentación de pruebas. La norma permite adaptarla al contexto: usa sus términos y omite lo que no aplique.
- **ISO 9001:** marco de evidencia y auditoría. Define qué registros deben existir (quién, cuándo, qué se probó, con qué resultado) sin imponer cómo trabajar. El sistema produce esa evidencia de forma automática.
- **ISO/IEC 25010:** modelo de calidad. Sus características (adecuación funcional, eficiencia de desempeño, compatibilidad, seguridad, fiabilidad, mantenibilidad, etc.) alimentan la matriz de riesgo: cada RISK- indica qué características están en juego.
- **ITIL / ISO/IEC 20000 (gestión de cambios):** cada cambio tiene un CHG- que viaja por todo el flujo, desde el requerimiento hasta el informe de conformidad.
- **Riesgo proporcional:** cada REQ lleva un nivel de riesgo y ese nivel define cuánta prueba necesita. Un fix menor en un módulo estable no sigue el mismo proceso que una certificación nueva.
- **Entregable (fase 1):** tabla de correspondencia entre cláusulas de estas normas y los campos o registros del sistema. Debe decir explícitamente que es una correspondencia de evidencia y no una declaración de cumplimiento.

---

## 6. Entidades y IDs

**Formato de ID:** prefijo + año + secuencia, por ejemplo `REQ-2026-0042`. Los IDs son inmutables, únicos y se pueden buscar en todo el sistema.

| Entidad | Qué es | Quién la define |
|---|---|---|
| **REQ-** | Requerimiento con criterio de aceptación, enlaces externos y documentación adjunta. | Change management la origina; dev y QA la enriquecen. |
| **CHG-** | Cambio aprobado que agrupa requerimientos. Ruta **alta** (SIMF, SGLPAR) o **ligera** (SyPago y fixes). | Change management |
| **RISK-** | Nivel de riesgo del cambio (alto, medio, bajo), características ISO 25010 en juego y justificación. Define la profundidad de prueba. | QA |
| **UC-** | Caso de uso funcional. | QA |
| **TC-** | Caso de prueba que valida uno o más UC. Declara componentes que toca y si es crítico. Dev puede aportar pruebas unitarias enlazadas a REQ. | QA (dev aporta) |
| **COMP-** | Componente del catálogo (microservicio, DLL, script, tópico de Kafka, esquema de base de datos). Pertenece a un sistema. | QA y dev |
| **BUILD-** | Artefacto inmutable: versión, checksum, manifiesto (REQ incluidos, migraciones, dependencias). No se modifica después de construirse. | Dev |
| **PKG-** | Conjunto de BUILDs y REQs listo para desplegar. | `[POR DEFINIR]`: dev o devops (permiso configurable) |
| **ENV-** | Ambiente con tipo (limpio, línea base, offline, producción), sistema operativo, versión registrada e historial de despliegues. | QA |
| **DEP-** | Despliegue registrado: fecha, BUILD o PKG, ENV, responsable, resultado. | QA (pruebas) o change management (producción) |
| **RUN-** | Ejecución **oficial** de un TC sobre un BUILD sin modificar y con el procedimiento estándar. Es la única que cuenta como evidencia de conformidad. | QA |
| **EXP-** | Ejecución **exploratoria**. Puede usar cambios manuales registrados. No cuenta como evidencia. | QA |
| **DEF-** | Defecto ligado a REQ y RUN. Severidad y estado. | QA lo abre; dev lo corrige. |
| **CHK-** | Checklist de producción versionado, con firma por paso y evidencia adjunta. | Change management |
| **DIAG-** | Resultado de diagnóstico posterior al despliegue (SIMF, SGLPAR). | Dev u ops lo entregan; QA lo revisa. |
| **SMOKE-** | Resultado de prueba de humo posterior al despliegue (SyPago). | QA |
| **DOC-** | Documento adjunto versionado (especificación, manual, evidencia, minuta), con checksum. Se enlaza a cualquier entidad. | Cualquier rol con permiso |
| **RPT-** | Informe de conformidad generado (inmutable una vez emitido, con hash). | QA |

---

## 7. Experiencia de usuario y diseño visual

### 7.0 Identidad de Kaoru
La identidad toma la estética japonesa como guía de diseño, no como decoración: nada de abanicos, cerezos ni tipografías "orientales".
- **Principios de diseño:**
  - ***Kanso* (simplicidad):** cada pantalla muestra solo lo necesario para decidir. Es KISS aplicado a la interfaz.
  - ***Ma* (espacio):** el espacio vacío es parte del diseño; separa en lugar de llenar con bordes y cajas.
  - ***Shibui* (elegancia sobria):** pocos colores, un solo acento, detalles finos que se notan al usarlo y no al primer vistazo.
- **Wordmark:** "kaoru" en minúsculas, con una sans geométrica de trazo fino (empaquetada localmente) y espaciado amplio entre letras. Tamaño mínimo y zona de respeto definidos en el boceto.
- **Símbolo:** un monograma propio y abstracto. Propuesta para el boceto: una "K" formada por una línea continua que conecta puntos, como una cadena de trazabilidad (REQ → TC → RUN → PKG). Debe leerse bien a 16 px (favicon) y en un solo color. Opcionalmente, el kanji 馨 (*kaoru*, el que lleva el nombre de Ishikawa) como marca de agua discreta en la pantalla de inicio de sesión y en la página "Acerca de" `[SUPUESTO: validar el kanji con alguien que lea japonés antes de usarlo]`.
- **Paleta base:**
  - Fondo oscuro tipo tinta (*sumi*): negro azulado profundo, nunca negro puro.
  - Superficies de vidrio: blanco o tinta con baja opacidad y desenfoque.
  - **Un solo color de acento** para acciones principales y el estado actual (propuesta: índigo, en referencia al *aizome*); se define en el boceto con su contraste validado.
  - Colores de estado (aprobado, en curso, bloqueado, defecto alto) separados del acento, siempre con ícono y texto.
  - Tema claro equivalente, tipo papel *washi*: blanco cálido, no blanco puro.
- **Tono de los textos:** sobrio, directo y en español; frases cortas y sin jerga de marketing. Los estados y errores dicen qué pasó y qué hacer ("Falta el manifiesto del BUILD para entregar a QA").
- **Dónde aparece la marca:** login de Keycloak (tema propio), barra lateral, favicon, encabezado y pie de los informes de conformidad en PDF y Markdown, y página "Acerca de" con una línea sobre Kaoru Ishikawa.
- **Fuente única:** el boceto (`tools/boceto`) define el wordmark, el símbolo, la paleta y los textos; todo lo demás se deriva de él (Anexo A §A.1.2).

### 7.1 Dirección visual: glass, elegante y moderno
- **Glassmorphism sobrio:** paneles translúcidos (`backdrop-filter: blur`) sobre un fondo con degradado sutil y profundo, bordes de 1 px semitransparentes, sombras suaves, esquinas redondeadas consistentes. El vidrio enmarca el contenido; **las tablas y textos densos van sobre superficies casi opacas** para que se lean bien.
- **Tema oscuro y claro**, con el oscuro como predeterminado.
- **Tipografía** sans moderna empaquetada localmente (por ejemplo Inter), con números tabulares en tablas e IDs en monoespaciada.
- **Sistema de diseño propio con tokens** (colores, espaciados, radios, desenfoque, elevación) definidos en un solo lugar. El **boceto** de la fase 1 (`tools/boceto`, HTML y CSS navegable) es la fuente única de apariencia y textos; de él salen `web/src/styles/tokens.css` y los componentes de `web/src/shared/ui`. Sin librería de componentes ni Tailwind (Anexo A §A.16).
- **Movimiento discreto:** transiciones de 150 a 250 ms, sin animaciones decorativas que distraigan.

### 7.2 Restricciones que el diseño debe respetar
- **Accesibilidad:** contraste WCAG 2.1 AA en todo texto, también sobre vidrio. El estado nunca se comunica solo por color: siempre color + ícono + texto.
- **Navegador objetivo: Google Chrome** (versión estable de escritorio). No hace falta soportar otros navegadores; puedes usar CSS moderno (`backdrop-filter`, container queries, `:has()`) sin polyfills.
- **Rendimiento en equipos del banco:** el desenfoque cuesta GPU. Limita el número de capas con desenfoque por pantalla, ofrece la opción de usuario "reducir transparencia" (pasa a superficies sólidas) y respeta `prefers-reduced-motion`.
- **Densidad:** modo cómodo y modo compacto para tablas grandes.

### 7.3 Navegación: ubicarse fácilmente
- **Búsqueda global (Ctrl+K):** escribir cualquier ID (`REQ-2026-0042`, `DEF-…`, `BUILD-…`) o texto lleva directo a la ficha. Incluye acciones rápidas ("nuevo DEF", "ir a matriz").
- **Barra lateral** por áreas: Inicio, Cambios, Requerimientos, Pruebas, Defectos, Paquetes, Ambientes, Producción, Informes, Calendario, Administración.
- **Migas de pan** que muestran la cadena de trazabilidad: `CHG-2026-0007 › REQ-2026-0042 › TC-2026-0110 › RUN-2026-0533`.
- **Vistas guardadas** y filtros compartibles por URL.

### 7.4 Ficha del requerimiento (pantalla central)
Al abrir un REQ se ve, sin desplazarse:
1. **Encabezado:** ID, título, sistema, CHG, nivel de riesgo, responsable actual y **estado actual** destacado.
2. **Línea de vida (stepper)** con todas las etapas del ciclo (sección 8.1): completadas, actual, pendientes y bloqueadas, con el tiempo en cada una.
3. **"Qué falta para avanzar":** lista de condiciones pendientes calculada por el sistema (por ejemplo "faltan 2 TC sin RUN aprobado", "DEF-…-0091 severidad alta abierto").
4. **Pestañas:**
   - **Resumen:** descripción y criterios de aceptación.
   - **Enlaces:** links al requerimiento. El tipo principal es **Google Sheets** (sección 7.5); también se aceptan otros (YouTrack, carpeta compartida, wiki). Cada enlace tiene tipo, etiqueta, URL y quién lo agregó.
   - **Documentación:** documentos adjuntos con **visor integrado** (sección 7.5), versiones e historial.
   - **Trazabilidad:** grafo y tabla de UC, TC, RUN, DEF, BUILD y PKG relacionados.
   - **Pruebas:** cobertura (TC con RUN aprobado / TC requeridos según riesgo).
   - **Defectos.**
   - **Actividad:** auditoría y comentarios en orden cronológico.

### 7.5 Visor de documentación
- **Visualización dentro del sistema, sin descargar:** PDF (PDF.js empaquetado), Markdown (renderizado y sanitizado), imágenes, texto plano y logs con resaltado.
- **Office (docx, xlsx):** descarga directa; conversión a PDF para previsualizar si LibreOffice está instalado en el servidor (opcional, detectado al iniciar).
- **Versionado:** subir una nueva versión conserva las anteriores; cada versión guarda checksum SHA-256, autor y fecha. Se puede comparar texto entre versiones de Markdown.
- **Seguridad:** límite de tamaño configurable, lista de tipos permitidos y validación del tipo real del archivo (no solo la extensión).
- **Enlaces externos:** se abren en otra pestaña; el servidor nunca intenta descargarlos. Se puede marcar un enlace como "documento oficial" del REQ.

### 7.5.1 Documentación en Google Sheets
La documentación de los requerimientos vive en Google Sheets y seguirá ahí. El sistema la **enlaza**:
- **Enlace estructurado:** al pegar una URL de Google Sheets, el sistema extrae y guarda el ID de la hoja, la pestaña (`gid`) y, si existe, el rango. Muestra una etiqueta legible ("Especificación SyPago · pestaña Pagos · A1:F40") en lugar de la URL cruda.
- **Apertura directa:** botón "Abrir en Google Sheets" que lleva a la pestaña y rango exactos, en otra pestaña de Chrome, con la sesión de Google del usuario.
- **Vista previa embebida opcional:** si el equipo del usuario tiene internet y sesión de Google, se puede mostrar la hoja dentro de la ficha en un `iframe`. Si no carga, se muestra el botón de apertura sin errores ni bloqueos. La aplicación funciona igual sin esta vista previa.
- **Validación de forma:** el sistema valida que la URL tenga el formato de Google Sheets, pero **no** verifica que exista ni que el usuario tenga permisos (el servidor está offline).
- **Instantánea como evidencia:** como el contenido de una hoja de Google puede cambiar, el usuario puede adjuntar una exportación (PDF o XLSX) como DOC- versionado con checksum. **Para el informe de conformidad cuenta la instantánea, no el enlace vivo.** El sistema avisa si un REQ que llega a "Aprobado QA" tiene enlaces sin instantánea.
- **Sin API de Google** en esta versión: el servidor no se conecta a Google. Una sincronización por API queda como opción futura, solo si el servidor llega a tener salida a internet.

### 7.6 Matriz de trazabilidad
- Tabla virtualizada (soporta miles de filas) con columnas REQ → RISK → UC → TC → RUN (por BUILD y ENV) → DEF → PKG → RPT.
- Celdas con chips de estado (color + ícono + texto); clic en cualquier ID abre su ficha en un panel lateral sin perder la posición.
- Filtros por CHG, PKG, sistema, componente, estado, riesgo, responsable y rango de fechas.
- Indicadores arriba: cobertura, REQ bloqueados, defectos abiertos por severidad.
- Exportación a CSV y XLSX.

### 7.7 Inicio (tablero)
Por rol: "mis pendientes", cambios en curso con su avance, defectos por estado con tiempo en estado (sin SLA impuesto), próximas ventanas de cambio y alertas de atraso de hitos.

---

## 8. Flujos y máquinas de estados (configurables)

Los estados, transiciones, permisos por transición y reglas de bloqueo se definen como **configuración** (tablas en base de datos editables por `qa-admin`), con valores predeterminados. Cada transición queda auditada.

### 8.1 Ciclo de vida del REQ (predeterminado)
`Registrado → Riesgo evaluado → Diseño de pruebas → En desarrollo → Entregado a QA → En pruebas → (Con defectos ↔ En pruebas) → Aprobado QA → En paquete → Liberado → Verificado post-despliegue → Cerrado`

Además: `Rechazado` y `Diferido a otro paquete` (requiere aceptación de change management).

- **Entregado a QA** solo se alcanza si el contrato de entrada (regla 8) está completo; el sistema muestra qué ítem falta.
- **Aprobado QA** exige la cobertura mínima que define su RISK y cero DEF de severidad alta abiertos.

### 8.2 Ruta del CHG
- **Ruta alta (SIMF, SGLPAR):** RISK obligatorio con características ISO 25010, regresión completa por impacto + TC críticos, checklist de producción firmado, DIAG obligatorio.
- **Ruta ligera (SyPago y fixes):** RISK simplificado, regresión por impacto, SMOKE obligatorio.
- **Fix menor:** subruta de la ligera para cambios de bajo riesgo en módulos estables, con TC mínimos definidos por QA. Hoy no existe atajo para fixes; este es el atajo controlado.

### 8.3 DEF
`Abierto → Triado → En corrección → Listo para reprueba → Cerrado`, más `Reabierto` y `Diferido` (requiere aceptación de change management). El sistema mide el tiempo en cada estado, sin SLA.

### 8.4 PKG
`En armado → En pruebas → Listo para liberar → Aprobado para producción → Desplegado → Verificado`.

---

## 9. Reglas de negocio

1. **La matriz de trazabilidad es el núcleo:** REQ → RISK → UC → TC → RUN (por BUILD y ENV) → DEF → PKG → informe de conformidad. Debe responder en una sola consulta qué pasó en cada cambio, quién lo hizo y cuándo.
2. **Bloqueo por severidad:** un DEF de severidad alta bloquea que el PKG pase a "Listo para liberar". Los de severidad media o baja pueden pasar al siguiente paquete si change management lo acepta (queda registrado quién aceptó y por qué).
   - Criterios de severidad: `[POR DEFINIR en fase 0]`. Propuesta inicial para discutir:
     - **Alta:** pérdida o corrupción de datos o dinero, caída de un servicio, vulnerabilidad de seguridad, incumplimiento de un criterio de aceptación crítico, sin alternativa operativa.
     - **Media:** función afectada con alternativa operativa razonable.
     - **Baja:** cosmético o de usabilidad sin impacto en el resultado.
3. **Sin SLA impuesto:** se mide por estado. El sistema muestra el tiempo en cada estado.
4. **Regresión por impacto:** cada REQ y cada TC declaran los componentes (COMP-) que tocan. Ante un PKG nuevo, el sistema propone los TC que cruzan con los componentes cambiados, más el conjunto fijo de TC críticos del sistema. QA puede añadir o quitar TC y **el criterio de cada ajuste queda registrado**. Se guarda la selección propuesta y la final para poder compararlas.
5. **RUN vs EXP:** RUN exige BUILD inmutable (checksum verificado) y procedimiento estándar. EXP permite cambios manuales registrados. Un defecto encontrado en EXP debe reproducirse en RUN antes de abrirse como DEF.
6. **Despliegues:**
   - **Ambientes de prueba:** QA despliega el BUILD exacto sin aprobación de change management. Cada despliegue se registra (fecha, BUILD, ENV, responsable, resultado).
   - **Producción:** lo ejecuta change management con un checklist versionado, firmado paso a paso por cada ejecutor. La ventana la define change management para cada CHG. QA no ejecuta pasos en producción.
7. **Verificación posterior:**
   - **SIMF y SGLPAR:** se registran DIAG entregados por dev u operaciones; QA los revisa y marca el resultado.
   - **SyPago:** QA registra SMOKE.
8. **Contrato dev-QA (vigente siempre):**
   - **Entrada (dev → QA):** IDs de REQ y CHG, componentes impactados, BUILD con manifiesto, scripts de instalación para Windows y Linux probados en ambiente limpio, pruebas unitarias y cobertura, notas de rollback, datos de prueba y known issues. **Sin todo esto, QA no inicia.** El sistema lo modela como una checklist de entrega por REQ o PKG y muestra qué falta.
   - **Salida (QA → dev y change management):** RUN por BUILD, registro de regresión, defectos con severidad y estado, e informe de conformidad con trazabilidad completa.
   - **Ingesta de resultados de dev:** se aceptan reportes en formato JUnit XML y cobertura en Cobertura XML o LCOV, que cubren Go, .NET y otros lenguajes con las herramientas habituales.
9. **Auditoría inmutable:** cada creación, cambio de estado, edición o decisión registra quién, cuándo, qué entidad, campo, valor anterior y nuevo, y motivo cuando aplique. La tabla de auditoría es de solo inserción (sin permisos de `UPDATE` ni `DELETE` para el usuario de la aplicación) y cada registro incluye el hash del anterior para detectar alteraciones.
10. **Quién construye el PKG:** `[POR DEFINIR]`. Se resuelve asignando el permiso `pkg.build` al rol que corresponda, sin reescribir código.
11. **Informes emitidos son inmutables:** un RPT emitido no se edita; si algo cambia, se emite una nueva versión que referencia la anterior.

---

## 10. Requisitos funcionales

- Gestión de REQ, CHG, RISK, UC, TC, COMP, BUILD, PKG, ENV, DEP, RUN, EXP, DEF, DIAG, SMOKE, DOC con enlaces entre ellos.
- Enlaces externos y documentos adjuntos en cualquier entidad, con visor integrado (sección 7.5).
- Ficha de REQ con línea de vida, "qué falta para avanzar" y pestañas (sección 7.4).
- Búsqueda global por ID y texto.
- Matriz de trazabilidad filtrable y exportable (sección 7.6).
- Selección de regresión por impacto con justificación y comparación propuesta vs final.
- Registro de RUN y EXP por BUILD y ENV, con evidencia adjunta (capturas, logs).
- Gestión de ambientes: tipo, sistema operativo, versión instalada e historial de despliegues.
- Flujos configurables: rutas alta, ligera y fix menor; estados; reglas de bloqueo; permisos por transición.
- Checklist de producción versionado con firma por paso (usuario autenticado + fecha + comentario) y evidencia adjunta.
- Registro de DIAG y SMOKE.
- Informe de conformidad exportable en PDF y Markdown: trazabilidad, ejecuciones, regresión, defectos abiertos y diferidos, firmas del checklist y hash del documento. Generación 100 % offline.
- Calendario de hitos, ventanas de cambio y fases, con alertas de atraso (dentro de la aplicación; correo opcional si hay SMTP interno `[POR DEFINIR]`).
- Tablero por rol (sección 7.7) y métricas: cobertura por REQ, defectos por estado y tiempo en estado, tamaño de regresión, defectos escapados a producción.
- Auditoría completa consultable y exportable.
- Migración de la hoja actual (Google Sheets exportado a XLSX o CSV, sin conexión a Google): importación con mapeo de columnas, conversión de las URLs de documentación en enlaces estructurados, **ejecución de prueba** (dry-run) con reporte de errores, e importación final auditada.
- Integración opcional con YouTrack mediante su API REST para sincronizar REQ y DEF (fase 5), desactivable sin afectar el resto.

---

## 11. Requisitos no funcionales

- **Offline:** sin dependencias externas en tiempo de ejecución. Corre en Docker sobre cualquier servidor con Docker Engine (Linux de preferencia). Lo único que usa internet son los enlaces a Google Sheets, y eso pasa en el Chrome del usuario, no en el servidor.
- **Los sistemas bajo prueba siguen siendo Windows y Linux:** el sistema registra BUILDs, scripts y despliegues de ambas plataformas aunque él mismo corra en Docker.
- **Backend en Go con PostgreSQL**, migraciones versionadas y reversibles.
- **Pruebas automatizadas** según el mínimo del Anexo A §A.17. Además, cobertura explícita de: lógica de trazabilidad, cálculo de "qué falta para avanzar", selección de regresión, reglas de bloqueo, máquinas de estados, permisos por rol y cadena de hash de auditoría.
- **Documentación:** API en OpenAPI 3 (visor empaquetado), diagrama del modelo de datos, guía de instalación offline con Docker, guía de respaldo y restauración, manual de usuario corto por rol.
- **Seguridad:** permisos mínimos por rol, validación de entradas, protección CSRF y cabeceras de seguridad, sanitización de Markdown, sin secretos en el repositorio.
- **Rendimiento:** la ficha de REQ y la búsqueda responden en menos de 1 s con 10 000 REQ y 100 000 RUN en el servidor objetivo `[SUPUESTO: dimensionar en fase 1]`.
- **Respaldo y recuperación** documentados y probados.

---

## 12. Fases, fechas y criterios de salida

Fecha de inicio supuesta: **lunes 16 de noviembre de 2026**. Confirmar o ajustar. Cada fase avanza solo cuando se cumple su criterio de salida.

| Fase | Fechas | Duración | Entregable | Criterio de salida |
|---|---|---|---|---|
| **0. Decisiones** | 16 nov – 27 nov 2026 | 2 semanas | Severidades, cadencia de triaje, quién construye el PKG, diagnósticos de SIMF y SGLPAR, formato de la hoja, servidor que aloja el sistema, acceso a internet desde los equipos de usuario, capacidad real | Decisiones firmadas por change management y QA |
| **1. Diseño** | 30 nov – 18 dic 2026 | 3 semanas | Esquema SQL, diagrama de entidades, máquinas de estados (REQ, CHG, DEF, PKG), contrato de API (OpenAPI), configuración de Keycloak y roles, tabla de correspondencia con las normas, **sistema de diseño y prototipo navegable** de inicio, ficha de REQ, matriz y visor de documentos | Esquema y prototipo aprobados por QA y dev |
| **2. MVP** | 21 dic 2026 – 22 ene 2027 | 5 semanas (con capacidad reducida en fin de año) | Login con Keycloak, REQ, CHG, RISK, UC, TC, RUN, DEF, enlaces y documentos con visor, búsqueda global, ficha de REQ, matriz de trazabilidad, auditoría, migración de la hoja | MVP usado en un CHG real de SyPago, con la hoja retirada para ese CHG |
| **3. Regresión y ambientes** | 25 ene – 26 feb 2027 | 5 semanas | COMP, selección de regresión por impacto, BUILD, PKG, ENV y DEP | Piloto de regresión en SyPago con resultados comparados contra la hoja |
| **4. Producción y conformidad** | 1 mar – 26 mar 2027 | 4 semanas | Checklist de producción, DIAG, SMOKE, informe de conformidad, calendario y alertas | Informe generado para un CHG de ruta alta, en la ventana que defina change management |
| **5. Ampliación** | desde abr 2027 | Continuo | Integración con YouTrack, métricas avanzadas, mejoras según uso | Se decide con change management y QA |

**Hitos:**
- H1, 27 nov 2026: decisiones cerradas.
- H2, 18 dic 2026: esquema y prototipo aprobados.
- H3, 22 ene 2027: MVP en uso.
- H4, 26 feb 2027: piloto de regresión completado.
- H5, 26 mar 2027: primer informe de conformidad de ruta alta.

**Reglas de planificación:**
- Las ventanas de despliegue de SIMF y SGLPAR las define change management. Si una ventana cae dentro de una fase, la fase no se detiene, pero el piloto de ruta alta se mueve a la siguiente ventana disponible.
- Las semanas de fin de año tienen capacidad reducida. Se reflejan en el plan con menos horas, no con menos fases.
- Si la capacidad real es menor a la supuesta, cada fase se extiende en proporción. **El plan se recalcula, no se recorta.** Con un equipo de dos personas en QA, valida en la fase 0 si estas fechas son realistas y presenta el plan recalculado.
- Cualquier atraso mayor a una semana se reporta a QA y change management con su impacto en los hitos.

**Definición de terminado (para cada fase con código):** pruebas automatizadas en verde, migraciones aplicadas sobre una base limpia, `docker compose up` probado en un servidor limpio **sin internet** a partir del paquete offline, documentación actualizada y demostración al rol que firma el criterio de salida.

---

## 13. Formato de cada entrega

En cada fase entrega, en este orden:
1. **Resumen** de lo hecho (máximo 10 líneas).
2. **Decisiones** tomadas y quién las aprobó.
3. **Supuestos** `[SUPUESTO]` que requieren validación.
4. **Pendientes** `[POR DEFINIR]` con responsable sugerido.
5. **Artefactos** (esquema, código, diagramas, documentos) con su ubicación.
6. **Evidencia del criterio de salida** o qué falta para cumplirlo.
7. **Riesgos** detectados para la fase siguiente.

---

## 14. Preguntas abiertas (hazlas antes de empezar)

**Proceso**
1. Severidades exactas y criterios de cada una (partiendo de la propuesta de la regla 2).
2. Cadencia de triaje de defectos y quién participa.
3. Quién construye el PKG: dev o devops.
4. Qué salidas tienen hoy los diagnósticos de SIMF y SGLPAR, en qué formato, y si QA puede verlas.
5. Formato de la hoja actual (columnas, pestañas, ejemplos) para migrar sus datos.
6. Fechas de las próximas ventanas de cambio de SIMF, SGLPAR y SyPago.
7. Qué TC se consideran críticos hoy en cada sistema.

**Documentación**
8. ¿Los equipos de los usuarios tienen internet para abrir Google Sheets mientras usan el sistema? Si no lo tienen, los enlaces solo se pueden abrir desde otro equipo y la instantánea adjunta pasa a ser obligatoria.
9. ¿La documentación es una sola hoja de Google con una pestaña por requerimiento, o una hoja por requerimiento?
10. Además de Google Sheets, ¿qué otros formatos se adjuntan como evidencia (PDF, capturas, logs, Word)? ¿Se incluye el contenedor de LibreOffice para previsualizar Office?

**Infraestructura**
11. ¿En qué servidor corre el Docker que aloja el sistema (Linux o Windows, offline o AWS)? ¿Quién lo administra?
12. ¿Certificado TLS interno del banco o autofirmado?
13. ¿Hay servidor SMTP interno para alertas por correo?

**Equipo**
14. Capacidad real: personas y horas por semana de QA, y cuánto apoyo de desarrollo hay para construir el sistema.

**Repositorio**
15. ¿Se reutiliza el `core` de la empresa que usa Aldía (axis, scribe, configuro, postgres_db)? Si sí, ¿de dónde se copia y en qué versión? Si no, se escribe un core mínimo con la misma forma (Anexo A §A.4).
16. ¿Dónde corre la CI (GitHub Actions, GitLab, otra)? Mientras tanto, `make check` ejecuta lo mismo en local.

---

## 15. Decisiones ya tomadas (no volver a preguntar)

| # | Tema | Decisión | Fecha |
|---|---|---|---|
| D1 | Documentación de requerimientos | Sigue viviendo en **Google Sheets**. El sistema la enlaza con enlaces estructurados y permite adjuntar instantáneas como evidencia (sección 7.5.1). | 11 nov 2026 |
| D2 | Identidad | **Keycloak dedicado** solo para este sistema, dentro del mismo Docker Compose, con usuarios locales. | 11 nov 2026 |
| D3 | Despliegue | **Docker y Docker Compose**, con paquete de imágenes para instalar offline. | 11 nov 2026 |
| D4 | Navegador | **Google Chrome** únicamente. | 11 nov 2026 |
| D5 | Stack | Go + React (TypeScript) + PostgreSQL + Keycloak. | 10 nov 2026 |
| D6 | Estilo visual | Glass, elegante y moderno, con tema oscuro predeterminado (sección 7). | 10 nov 2026 |
| D7 | Reglas del repositorio | Se adoptan las reglas de arquitectura de Aldía, adaptadas a este sistema (Anexo A). KISS y DRY mandan. | 11 nov 2026 |
| D8 | Sesión | Nginx + oauth2-proxy con cookie; sin tokens en el navegador (en lugar de PKCE en la SPA). | 11 nov 2026 |
| D9 | Front | Boceto propio como fuente de apariencia; sin librería de componentes ni Tailwind. | 11 nov 2026 |
| D10 | Nombre | El sistema se llama **Kaoru**, en honor a Kaoru Ishikawa. Repositorio `kaoru`, realm `kaoru`, binario `kaoru-api`, esquema `kaoru`. Identidad en la sección 7.0. | 11 nov 2026 |

---

# Anexo A — Reglas del repositorio

El Anexo A vive en [`docs/reglas.md`](reglas.md), que es su fuente única desde el primer commit (sección 0, punto 2). Las referencias «Anexo A §A.n» de este documento apuntan a ese archivo.
