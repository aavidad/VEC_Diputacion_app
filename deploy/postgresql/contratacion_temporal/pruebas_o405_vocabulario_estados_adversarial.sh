#!/usr/bin/env bash

# Se carga desde el runner PG18.4, que aporta psql_admin, archivo,
# esperar_fallo y comprobar_estado_000041a.

down_vocabulario() {
    archivo \
        contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.down.sql
}

down_con_funcion_inmutable_noop() {
    psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
CREATE OR REPLACE FUNCTION
vec_contratacion_temporal.rechazar_mutacion_historia_v1()
RETURNS trigger
LANGUAGE plpgsql
AS $funcion$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END
$funcion$;
\ir /repo/contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.down.sql
COMMIT;
SQL
}

up_con_funcion_inmutable_noop() {
    psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
CREATE OR REPLACE FUNCTION
vec_contratacion_temporal.rechazar_mutacion_historia_v1()
RETURNS trigger
LANGUAGE plpgsql
AS $funcion$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END
$funcion$;
\ir /repo/contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.up.sql
COMMIT;
SQL
}

probar_derivas_vocabulario_instalado() {
    local estado_esperado=$1

    paso 'triggers, políticas, ACL y constraints extra no se pierden en DOWN'
    psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
CREATE TRIGGER control_vocabulario_estados_rrhh_derivado
BEFORE INSERT ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
FOR EACH ROW EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();
SQL
    esperar_fallo 'down con trigger derivado' 55000 \
        'estructura derivada impide revertir estados RRHH' \
        down_vocabulario
    psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DROP TRIGGER control_vocabulario_estados_rrhh_derivado ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1;
CREATE POLICY politica_derivada_ct000041a ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
TO vec_contratacion_temporal_propietario
USING (true) WITH CHECK (true);
SQL
    esperar_fallo 'down con política derivada' 55000 \
        'estructura derivada impide revertir estados RRHH' \
        down_vocabulario
    psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DROP POLICY politica_derivada_ct000041a ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1;
RESET ROLE;
CREATE ROLE vec_ct000041a_rol_futuro NOLOGIN;
SET ROLE vec_contratacion_temporal_propietario;
GRANT SELECT ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
TO vec_ct000041a_rol_futuro;
SQL
    esperar_fallo 'down con ACL de rol futuro' 55000 \
        'estructura derivada impide revertir estados RRHH' \
        down_vocabulario
    psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
REVOKE SELECT ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
FROM vec_ct000041a_rol_futuro;
RESET ROLE;
DROP ROLE vec_ct000041a_rol_futuro;
SET ROLE vec_contratacion_temporal_propietario;
ALTER TABLE
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
DROP CONSTRAINT control_vocabulario_estados_rrhh_creada_check;
ALTER TABLE
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
ADD CONSTRAINT control_vocabulario_estados_rrhh_creada_check
CHECK (true);
SQL
    esperar_fallo 'down con constraint homónima alterada' 55000 \
        'estructura derivada impide revertir estados RRHH' \
        down_vocabulario
    psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
ALTER TABLE
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
DROP CONSTRAINT control_vocabulario_estados_rrhh_creada_check;
ALTER TABLE
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
ADD CONSTRAINT control_vocabulario_estados_rrhh_creada_check
CHECK (
    creada_en = pg_catalog.date_trunc('microseconds', creada_en)
);
SQL
    comprobar_estado_000041a "$estado_esperado" \
        'derivas básicas restauradas'

    paso 'función no-op, columna, secuencia, tipo y TOAST bloquean el DOWN'
    esperar_fallo 'down con función de inmutabilidad no-op' 55000 \
        'estructura derivada impide revertir estados RRHH' \
        down_con_funcion_inmutable_noop
    comprobar_estado_000041a "$estado_esperado" \
        'función no-op revertida transaccionalmente'

    psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
ALTER TABLE
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
ADD COLUMN campo_futuro text;
SQL
    esperar_fallo 'down con columna futura' 55000 \
        'estructura derivada impide revertir estados RRHH' \
        down_vocabulario
    psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
ALTER TABLE
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
DROP COLUMN campo_futuro;
CREATE SEQUENCE
    vec_contratacion_temporal.secuencia_ct000041a_futura;
ALTER SEQUENCE
    vec_contratacion_temporal.secuencia_ct000041a_futura
OWNED BY
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1.version_esquema;
SQL
    esperar_fallo 'down con secuencia OWNED BY sin default' 55000 \
        'estructura derivada impide revertir estados RRHH' \
        down_vocabulario
    psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
ALTER SEQUENCE
    vec_contratacion_temporal.secuencia_ct000041a_futura
OWNED BY NONE;
DROP SEQUENCE
    vec_contratacion_temporal.secuencia_ct000041a_futura;
COMMENT ON TYPE
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
IS 'deriva de tipo compuesto';
SQL
    esperar_fallo 'down con comentario del tipo compuesto' 55000 \
        'estructura derivada impide revertir estados RRHH' \
        down_vocabulario
    psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
COMMENT ON TYPE
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
IS NULL;
ALTER TABLE
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
SET (toast.autovacuum_enabled = false);
SQL
    esperar_fallo 'down con configuración TOAST derivada' 55000 \
        'estructura derivada impide revertir estados RRHH' \
        down_vocabulario
    psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
ALTER TABLE
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
RESET (toast.autovacuum_enabled);
SQL
    comprobar_estado_000041a "$estado_esperado" \
        'objetos implícitos restaurados'
}

probar_deriva_funcion_antes_up() {
    esperar_fallo 'up con función de inmutabilidad no-op' 55000 \
        'estructura heredada de estados RRHH incompatible' \
        up_con_funcion_inmutable_noop
}
