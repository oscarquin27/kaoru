# Keycloak: realm, roles y sesión

El realm `kaoru` está versionado en [`infra/keycloak/kaoru-realm.json`](../infra/keycloak/kaoru-realm.json) y se importa al arrancar (`--import-realm`). Una instalación limpia no requiere configurar nada a mano.

## Qué define el realm

| Tema | Valor | Estado |
|---|---|---|
| Roles de realm | `change-manager`, `qa-admin`, `qa`, `dev`, `ops`, `auditor` | Decidido (requerimiento §4) |
| Cliente | `kaoru-web`, confidencial, solo flujo de código de autorización. Lo usa oauth2-proxy. | Decidido (D8) |
| Registro propio y recuperación por correo | Desactivados (no hay SMTP) | `[SUPUESTO]` hasta la pregunta 13 |
| Política de contraseñas | 12 caracteres, mayúscula, minúscula, dígito, distinta del usuario, historial de 5 | `[POR DEFINIR]`: política del banco |
| Bloqueo por intentos fallidos | 5 intentos; espera creciente de 1 a 15 minutos | `[POR DEFINIR]` |
| OTP | TOTP disponible, opcional por usuario | Decidido (requerimiento §4) |
| Cierre por inactividad | 30 minutos; sesión máxima de 10 horas | `[POR DEFINIR]` |
| Vida del token de acceso | 5 minutos | `[SUPUESTO]` |
| Eventos | De usuario (90 días) y de administración, con detalle | Necesario para auditar el acceso de emergencia |
| Idioma | Español | Decidido |

El JSON usa dos variables que llegan por entorno: `KAORU_PUBLIC_URL` (origen público de la app) y `KAORU_OIDC_CLIENT_SECRET`. Ningún secreto está en el repositorio.

El tema de login propio todavía no está en el realm: se agrega cuando exista, derivado del boceto (fase 2).

## Tema de login

El login usa el tema `kaoru` (`infra/keycloak/themes/kaoru`), que viaja completo en el paquete. Una instalación nueva lo trae activo. En una que ya existía se activa una vez en la consola: **Realm settings → Themes → Login theme → kaoru**. Keycloak guarda una copia comprimida de cada archivo del tema (`/opt/keycloak/data/tmp/kc-gzip-cache`) y no la renueva si el archivo cambia, ni al reiniciarse: los navegadores, que piden gzip, seguirían recibiendo la versión anterior. `install.sh` y `tools/dev-up.sh` la borran en cada arranque y Keycloak la rehace sola, sin reiniciarse. Si se cambia el tema a mano con todo arriba: `docker compose exec keycloak rm -rf /opt/keycloak/data/tmp/kc-gzip-cache`. Además, Keycloak deja que el navegador guarde esos archivos 30 días en la misma dirección: quien ya había abierto el login ve el cambio tras recargar sin caché (Cmd+Shift+R).

## Roles y permisos

Keycloak solo dice qué roles tiene el usuario (`realm_access.roles` en el token). Qué puede hacer cada rol es configuración de Kaoru: tabla `role_permission`, editable por `qa-admin`. Los valores predeterminados están en [flujos.md](flujos.md#permisos-por-rol-predeterminados).

## Sesión

1. Nginx recibe la petición y pregunta a oauth2-proxy si hay sesión.
2. Sin sesión, oauth2-proxy redirige a Keycloak. Al volver, guarda la sesión en una cookie `SameSite=Lax`.
3. Con sesión, Nginx pone el token de acceso en la cabecera `Authorization` y **sobrescribe** lo que haya mandado el navegador.
4. `kaoru-api` verifica el token (firma con JWKS en caché, `iss`, `exp`, `nbf`, `azp = kaoru-web`, `typ = Bearer`) y pasa `sub` y roles al caso de uso.

El navegador nunca ve un token.

## Administración y acceso de emergencia

- La consola de administración de Keycloak vive en el realm `master` y solo la usan los administradores designados. No se publica en la misma ruta que la aplicación.
- **Acceso de emergencia (break-glass)**, propuesta `[SUPUESTO]`:
  1. Existe una cuenta de administrador de `master` que no se usa en el día a día. Su contraseña está en un sobre sellado o en la bóveda del banco, en custodia de change management.
  2. Usarla exige registrar un CHG de emergencia con el motivo.
  3. Lo que se haga con ella queda en los eventos de administración de Keycloak (activados en el realm), que se exportan y se adjuntan al CHG como DOC.
  4. Al terminar, se cambia la contraseña y se vuelve a sellar.
- Quiénes son los administradores designados y dónde se custodia la contraseña: `[POR DEFINIR]` con change management.

## Verificado

`tools/e2e-compose.sh` levanta el Compose completo desde cero con Keycloak 26.4, crea dos usuarios y hace lo que haría Chrome. Comprueba que:

- Sin sesión, `/api/` responde 401 y `/` redirige al inicio de sesión de Keycloak.
- Tras iniciar sesión, el navegador solo tiene cookies. La de la aplicación (`_oauth2_proxy`) va cifrada y es `HttpOnly`; no trae un token legible.
- Un token puesto por el navegador en `Authorization` no vale: Nginx lo sobrescribe. La cookie de sesión propia de Keycloak tampoco sirve como token de la API.
- `kaoru-api` recibe el token, lo verifica y aplica los permisos del rol (`qa` no crea un CHG; `change-manager` sí).
- Una escritura sin la cabecera `Origin` de la aplicación se rechaza (CSRF).
- La consola de administración de Keycloak no está publicada en la dirección de la aplicación.
- Al cerrar sesión se cierra también la de Keycloak: volver a entrar pide la contraseña.
- Con `E2E_TOKEN_REFRESH=1`, que la sesión sigue viva y se puede escribir después de que vence el token de acceso (5 minutos).

El realm incluye un mapeo de audiencia para que el token de acceso nombre al cliente `kaoru-web`, que oauth2-proxy exige.

No verificado: el tema de login propio (no existe todavía), OTP, el bloqueo por intentos fallidos y la federación con LDAP.
