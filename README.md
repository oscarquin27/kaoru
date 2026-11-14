# Kaoru

Trazabilidad a prueba de errores: requerimientos, pruebas, defectos, paquetes, producción y métricas, con auditoría encadenada. El requerimiento está en [docs/kaoru-requerimiento-v2.md](docs/kaoru-requerimiento-v2.md) y las decisiones en [docs/decisiones.md](docs/decisiones.md).

## Probarlo en este equipo

Requiere Docker en marcha, Go, Node y, una vez, `npm --prefix web ci`.

| Para | Comando |
|---|---|
| Levantar todo desde cero con datos de ejemplo (borra lo que había) | `make demo` |
| Levantar o actualizar con el código actual, conservando los datos | `make up` |
| Agregar otro juego de datos de ejemplo | `make seed` |
| Ver la contraseña de los usuarios de laboratorio | `make password` |
| Front con recarga al guardar, como un usuario | `make front COMO=lab-manager` |
| Apagar, conservando los datos | `make down` |
| Apagar y borrar los datos | `make reset` |

Kaoru queda en **https://localhost:8443**. El certificado es de laboratorio: Chrome avisa la primera vez («Configuración avanzada → Continuar»). `make front` sirve el front en http://localhost:5173 ya con la sesión del usuario, sin pasar por el login.

Usuarios de laboratorio, uno por rol, todos con la contraseña de `make password`:

| Usuario | Rol |
|---|---|
| `lab-manager` | change-manager |
| `lab-qa` | qa |
| `lab-admin` | qa-admin |
| `lab-dev` | dev |
| `lab-ops` | ops |
| `lab-auditor` | auditor |

Un usuario puede tener varios roles en Keycloak (consola en http://localhost:8081/auth/admin/, `KEYCLOAK_ADMIN_USER` y `KEYCLOAK_ADMIN_PASSWORD` de `infra/.env.example`); sus permisos son la suma.

## Comprobar e instalar

- `make check`: todo lo que corre la CI (apaga antes el entorno con `make down`: usa los mismos puertos).
- `make offline-bundle`: el paquete para instalar en un servidor; ver [docs/instalacion.md](docs/instalacion.md) y [docs/respaldo.md](docs/respaldo.md).
