# Despliegue en el servidor de pruebas (contabo)

Kaoru de prueba corre en **https://kaoru.aldia.services**, en un servidor Ubuntu con Docker que ya atiende otras aplicaciones. Lo despliega un playbook de Ansible desde el equipo de desarrollo: `infra/contabo/deploy.yml`.

## Requisitos en este equipo

- Ansible, Docker con `buildx` y `npm --prefix web ci` hecho una vez.
- Un alias `contabo` en `~/.ssh/config` que entre como `root` al servidor.

## Desplegar

```bash
make deploy-contabo
```

Hace, en orden:

1. **Comprobaciones, sin cambiar nada:** los puertos `127.0.0.1:8445` (Nginx de Kaoru) y `127.0.0.1:8182` (consola de Keycloak) están libres o ya son de Kaoru; el proyecto de Compose `kaoru` no lo usa otra carpeta; ningún otro sitio del nginx del servidor atiende el dominio.
2. **En este equipo:** compila el front, construye la imagen de `kaoru-api` para `linux/amd64` (el servidor es x86_64) y empaqueta la configuración de `infra/`. El build corre con `tools/docker-sin-llavero.sh`: sin credenciales guardadas, porque las imágenes base son públicas, y sin el aviso del llavero de macOS, así el despliegue no se detiene si nadie está frente al equipo.
3. **En el servidor**, en `/app/kaoru`: sube y carga la imagen, reemplaza la configuración y el front, escribe `.env` con secretos al azar **solo la primera vez**, crea un certificado interno autofirmado y corre `install.sh`. Las demás imágenes las baja Docker por su digest. La primera vez crea además el usuario `kaoru-admin`, con los roles `qa-admin`, `change-manager`, `dev` y `ops`: así una sola persona puede cargar CHG y REQ, planificar y llevar un REQ por todo el flujo (de «Diseño de pruebas» a «En desarrollo» y «Entregado a QA» hace falta `dev`). Si el usuario ya existe, solo le asegura esos roles.
4. **Sitio público:** agrega `/etc/nginx/sites-available/kaoru.aldia.services`, comprueba que `http://kaoru.aldia.services` llegue a este servidor (deja un archivo en la carpeta del desafío de Let's Encrypt y lo pide por el dominio), pide el certificado con el certbot del sistema y comprueba que `https://kaoru.aldia.services/healthz` responda.

Por partes: `make deploy-contabo TAGS=app` (solo la aplicación, sin tocar el nginx del servidor) o `TAGS=sitio` (solo el sitio y el certificado). Para actualizar, el mismo comando: los datos viven en volúmenes de Docker y no se tocan.

## Cómo convive con lo que ya hay

```
Internet ─▶ Cloudflare (proxy) ─▶ nginx del servidor (:443, certificado de Let's Encrypt)
               └─ kaoru.aldia.services ─▶ 127.0.0.1:8445  Nginx de Kaoru (TLS interno)
                                                ├─ front, /api → kaoru-api
                                                ├─ /oauth2 → oauth2-proxy
                                                └─ /auth/realms → Keycloak
```

- El DNS de `aldia.services` está en Cloudflare y el registro `kaoru` va con proxy (nube naranja): el nombre resuelve a IP de Cloudflare, no a la del servidor. Por eso el playbook no compara IP sino que prueba el camino completo. Funciona porque Cloudflare deja pasar `/.well-known/acme-challenge/` por HTTP y habla con el servidor por HTTPS; el modo SSL de la zona debe ser *Full (strict)* o *Full*. En *Flexible* habría un bucle de redirecciones, porque el puerto 80 de Kaoru redirige a https.
- Kaoru no publica nada fuera de `127.0.0.1`. El 8443 del servidor es de otra aplicación; por eso Kaoru usa el 8445.
- Del nginx del servidor solo se agrega el sitio de Kaoru, y se recarga únicamente si `nginx -t` pasa. Si no pasa, se quita el enlace y nginx sigue con lo que tenía.
- El certificado lo renueva el `certbot.timer` del sistema, que recarga nginx al renovarlo.
- La vista previa de Word y Excel (perfil `office`) no está activa: su contenedor pesa unos 2,7 GB y el servidor comparte memoria con las otras aplicaciones.

## Credenciales

Todo vive en el servidor, legible solo por root:

| Qué | Dónde |
|---|---|
| Contraseña de `kaoru-admin` | `ssh contabo cat /app/kaoru/.admin-password` |
| Administrador de Keycloak y secretos | `/app/kaoru/.env` |

La consola de Keycloak no se publica. Para crear usuarios, abre un túnel y entra con `KEYCLOAK_ADMIN_USER` y `KEYCLOAK_ADMIN_PASSWORD` de `.env`:

```bash
ssh -L 8182:localhost:8182 contabo
```

Luego, en este equipo: `http://localhost:8182/auth/admin/`, realm `kaoru` (procedimiento en [instalacion.md](instalacion.md#crear-los-usuarios)).

## Límites conocidos

- **Cambiar de dominio** no basta con cambiar `kaoru_domain`: Keycloak importó el realm con la dirección de la primera instalación, y las URI de redirección del cliente `kaoru-web` hay que corregirlas en la consola.
- **La IP del usuario** no llega a Kaoru: pasa por Cloudflare y dos nginx, y el de Kaoru no confía en las cabeceras de los anteriores, así que registra la IP interna de Docker.
- Las imágenes `kaoru-api:<versión>` anteriores quedan en el servidor (unos 11 MB cada una). No se borran solas porque `docker image prune` tocaría las de las otras aplicaciones.
