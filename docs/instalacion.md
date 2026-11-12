# Instalación offline de Kaoru

Kaoru corre en un solo servidor con Docker Engine y Docker Compose (Linux de preferencia). No necesita internet para funcionar ni para instalarse: todo viaja en un paquete.

## Qué trae el paquete

`kaoru-offline-<versión>.tar.gz` se arma donde hay internet con `make offline-bundle` y contiene:

| Archivo | Para qué |
|---|---|
| `images.tar.gz` y `SHA256SUMS` | Las cinco imágenes (`docker save`) y su suma de comprobación. |
| `docker-compose.yml` | Los servicios: `nginx`, `oauth2-proxy`, `kaoru-api`, `keycloak` y `postgres`. |
| `.env.example` | Variables de entorno con valores de laboratorio. |
| `nginx/`, `postgres/`, `keycloak/`, `web/` | Configuración de cada servicio y el realm `kaoru`. |
| `install.sh`, `backup.sh`, `restore.sh` | Instalar o actualizar, respaldar y restaurar. |

Las imágenes se fijan por tag y digest al construir el paquete. Dentro del paquete se referencian solo por tag, porque `docker load` no conserva el digest en todos los motores; la integridad la da la suma de `images.tar.gz`, que `install.sh` comprueba antes de cargar.

## Instalar

1. Copia el paquete al servidor y descomprímelo: `tar xzf kaoru-offline-<versión>.tar.gz`.
2. Dentro de la carpeta, copia `.env.example` a `.env` y **cambia todas las contraseñas y secretos**. `OAUTH2_PROXY_COOKIE_SECRET` debe tener exactamente 32 caracteres.
3. Pon en `KAORU_PUBLIC_URL` la dirección que escribirán los usuarios en Chrome, sin barra final (por ejemplo `https://kaoru.banco.local`), y en `KAORU_HTTPS_PORT` el puerto del servidor.
4. Pon el certificado del servidor en `tls/tls.crt` y `tls/tls.key`. `[POR DEFINIR]` si será el certificado interno del banco o uno autofirmado (pregunta 12).
5. Ejecuta `./install.sh`. Carga las imágenes, levanta los servicios en orden y espera a que todos estén sanos.

La primera vez, Postgres crea la base de Kaoru y la de Keycloak, `kaoru-api` aplica sus migraciones y Keycloak importa el realm `kaoru` con sus seis roles. No hay nada que configurar a mano.

## Crear los usuarios

Los usuarios son locales de Keycloak. La consola de administración **no está publicada** en la dirección de la aplicación: solo atiende en el propio servidor, en `http://localhost:8081/auth/admin/` (`KEYCLOAK_ADMIN_PORT`). Desde otro equipo se llega con un túnel SSH.

Entra con el administrador de `.env`, elige el realm `kaoru`, crea cada usuario con nombre, apellido y contraseña, y asígnale uno de los roles: `change-manager`, `qa-admin`, `qa`, `dev`, `ops` o `auditor`. El procedimiento de acceso de emergencia está en [keycloak.md](keycloak.md).

## Actualizar

Descomprime el paquete nuevo sobre la carpeta (conserva tu `.env` y tu `tls/`) y ejecuta `./install.sh` otra vez. Los datos viven en volúmenes de Docker y no se tocan; las migraciones nuevas se aplican al arrancar. Haz un [respaldo](respaldo.md) antes.

## Vista previa de Word y Excel (opcional)

Sin esto, los archivos de Office se descargan. Para verlos dentro de Kaoru hace falta un contenedor más (unos 2,7 GB), que solo viaja en el paquete si se armó con `OFFLINE_OFFICE=1 tools/offline-bundle.sh`. Para activarlo, quita el `#` de estas dos líneas de `.env` y vuelve a correr `./install.sh`:

```
COMPOSE_PROFILES=office
KAORU_CONVERTER_URL=http://libreoffice:3000
```

## Mantenimiento

Si una subida de documento falla después de guardar el archivo, el archivo queda en el volumen sin registro. Para verlos (no borra nada):

```bash
docker compose -p kaoru exec kaoru-api /kaoru-api prune-documents
```

Con `--apply` al final los borra. Solo cuenta archivos con más de un día, para no tocar una subida en curso.

## Comprobar que quedó bien

- `docker compose -p kaoru ps` muestra los cinco servicios como `healthy`.
- En Chrome, la dirección de `KAORU_PUBLIC_URL` lleva al inicio de sesión de Keycloak y, al entrar, a la aplicación.
- Los logs de `kaoru-api` son JSON: `docker compose -p kaoru logs kaoru-api`. Cada línea lleva el `trace_id` que también aparece en los mensajes de error que ve el usuario.

## Qué se verificó y qué no

`tools/e2e-offline.sh` instala desde el paquete, usa el sistema con sesión real, respalda, destruye todo, reinstala y restaura. Esa prueba corre en un equipo con internet: comprueba que el paquete basta para instalar, pero **no** se probó todavía en un servidor sin red. Esa es la demostración que pide la definición de terminado de la fase.
