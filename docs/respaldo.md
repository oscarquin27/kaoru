# Respaldo y restauración

## Qué hay que respaldar

| Qué | Dónde vive | Cómo se respalda |
|---|---|---|
| Datos de Kaoru (REQ, RUN, DEF, auditoría…) | Base `kaoru` de Postgres | `pg_dump` en formato propio |
| Usuarios, roles y contraseñas | Base `keycloak` de Postgres | `pg_dump` en formato propio |
| Documentos adjuntos | Volumen `documents` | Un `tar.gz` del volumen |

Además guarda aparte, y fuera del servidor, tu `.env` y tu `tls/`: sin los secretos de `.env` las sesiones abiertas dejan de valer, y sin el certificado Nginx no arranca.

## Respaldar

```bash
./backup.sh /ruta/de/respaldos
```

Crea `/ruta/de/respaldos/kaoru-AAAAMMDD-HHMMSS/` con `kaoru.dump`, `keycloak.dump`, `documents.tar.gz` y `SHA256SUMS`. Se puede ejecutar con el sistema en uso. Prográmalo con `cron` y copia el directorio a otro equipo: un respaldo en el mismo disco no protege de perder el disco.

Los tres archivos no son una foto del mismo instante. Si alguien adjunta un documento justo mientras corre el respaldo, puede quedar el archivo sin su registro, que es inofensivo; nunca un registro sin archivo, porque las bases se respaldan antes que los documentos.

## Restaurar

```bash
./restore.sh /ruta/de/respaldos/kaoru-AAAAMMDD-HHMMSS
```

**Reemplaza** las dos bases y los documentos por los del respaldo. Antes comprueba las sumas de `SHA256SUMS` y se niega a seguir si no coinciden. Detiene la aplicación mientras restaura y la vuelve a levantar al terminar.

Sirve en los dos casos:

- **Servidor nuevo:** instala primero con `./install.sh` (con el mismo `.env`) y después restaura.
- **Misma instalación:** restaura directo.

Después de restaurar, comprueba la cadena de auditoría desde la aplicación (Administración › Auditoría › Verificar) o con `GET /api/audit/verification`: debe decir que está íntegra.

## Qué se verificó

`tools/e2e-offline.sh` crea datos reales (un CHG y un documento adjunto), respalda, borra los volúmenes, reinstala desde cero, restaura y comprueba que el mismo usuario entra con su misma contraseña, que el CHG está, que el documento descargado es idéntico byte a byte y que la cadena de auditoría sigue íntegra.
