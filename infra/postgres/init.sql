-- Se ejecuta una vez, como superusuario, sobre la base `kaoru` recién creada.
-- Las contraseñas llegan como variables de psql: -v owner_password=… -v app_password=…
--
-- kaoru_owner: dueño del esquema; aplica las migraciones al arrancar.
-- kaoru_app:   usuario de ejecución; no puede cambiar el esquema ni tocar la auditoría.
CREATE ROLE kaoru_owner LOGIN PASSWORD :'owner_password';
CREATE ROLE kaoru_app LOGIN PASSWORD :'app_password';

CREATE SCHEMA kaoru AUTHORIZATION kaoru_owner;
GRANT USAGE ON SCHEMA kaoru TO kaoru_app;

ALTER ROLE kaoru_owner SET search_path = kaoru;
ALTER ROLE kaoru_app SET search_path = kaoru;
