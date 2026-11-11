-- +goose Up
-- El esquema `kaoru` y los roles `kaoru_owner` (migra) y `kaoru_app` (ejecuta)
-- los crea infra/postgres/init.sql. Las migraciones corren como `kaoru_owner`.
ALTER DEFAULT PRIVILEGES IN SCHEMA kaoru
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kaoru_app;

-- El id es el `sub` de Keycloak. Los roles no se guardan: llegan en el token.
CREATE TABLE app_user (
    id           text PRIMARY KEY,
    username     text        NOT NULL,
    display_name text        NOT NULL,
    email        text,
    created_at   timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL
);

CREATE TABLE system_under_test (
    id            uuid PRIMARY KEY,
    key           text        NOT NULL UNIQUE,
    name          text        NOT NULL,
    default_route text        NOT NULL CHECK (default_route IN ('high', 'light')),
    created_at    timestamptz NOT NULL,
    updated_at    timestamptz NOT NULL
);

-- Secuencia por prefijo y año de los IDs legibles (REQ-2026-0042).
CREATE TABLE code_sequence (
    prefix     text    NOT NULL,
    year       integer NOT NULL,
    last_value integer NOT NULL CHECK (last_value >= 0),
    PRIMARY KEY (prefix, year)
);

INSERT INTO system_under_test (id, key, name, default_route, created_at, updated_at)
VALUES ('01930000-0000-7000-8000-000000000001', 'SIMF', 'SIMF', 'high', now(), now()),
       ('01930000-0000-7000-8000-000000000002', 'SGLPAR', 'SGLPAR', 'high', now(), now()),
       ('01930000-0000-7000-8000-000000000003', 'SYPAGO', 'SyPago', 'light', now(), now());

-- +goose Down
DROP TABLE code_sequence;
DROP TABLE system_under_test;
DROP TABLE app_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA kaoru
    REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM kaoru_app;
