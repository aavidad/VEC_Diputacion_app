#!/usr/bin/env bash

# Se carga desde el runner PG18.4, que aporta psql_admin, archivo,
# esperar_fallo y comprobar_estado_000041a.

down_vocabulario() {
    archivo \
        contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.down.sql
}

estado_antes_up_000041a() {
    valor "SELECT pg_catalog.concat_ws('|',
        cobertura.version_esquema::text,
        consultas.version_esquema::text,
        (pg_catalog.to_regclass(
            'vec_contratacion_temporal.control_vocabulario_estados_publicacion_rrhh_v1'
        ) IS NULL)::text,
        restriccion.convalidated::text,
        pg_catalog.pg_get_constraintdef(restriccion.oid, false)
    )
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4 cobertura
      CROSS JOIN
           vec_contratacion_temporal.control_migracion_consultas_rrhh consultas
      JOIN pg_catalog.pg_constraint restriccion
        ON restriccion.conrelid =
           'vec_contratacion_temporal.publicacion_version_rrhh'::regclass
       AND restriccion.conname =
           'publicacion_version_rrhh_estado_clave_check'
     WHERE cobertura.control AND consultas.control"
}

comprobar_estado_antes_up_000041a() {
    local esperado=$1
    local contexto=$2
    local obtenido
    obtenido="$(estado_antes_up_000041a)"
    if [[ $obtenido != "$esperado" ]]; then
        printf 'estado previo CT-000041A alterado tras %s\n' \
            "$contexto" >&2
        printf 'esperado=%s\nobtenido=%s\n' \
            "$esperado" "$obtenido" >&2
        return 1
    fi
}

conceder_acl_tipo_array() {
    psql_admin <<'SQL'
SET ROLE vec_contratacion_temporal_propietario;
GRANT USAGE ON TYPE
    vec_contratacion_temporal
        ._control_vocabulario_estados_publicacion_rrhh_v1
TO vec_contratacion_temporal_consultor_rrhh;
SQL
}

down_con_acl_tipo_fila() {
    psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
GRANT USAGE ON TYPE
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
TO vec_contratacion_temporal_consultor_rrhh;
\ir /repo/contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.down.sql
COMMIT;
SQL
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

up_con_disparador_publicacion_homonimo_debil() {
    psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DROP TRIGGER publicacion_version_rrhh_inmutable
ON vec_contratacion_temporal.publicacion_version_rrhh;
CREATE TRIGGER publicacion_version_rrhh_inmutable
BEFORE UPDATE
ON vec_contratacion_temporal.publicacion_version_rrhh
FOR EACH ROW
EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();
\ir /repo/contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.up.sql
COMMIT;
SQL
}

probar_derivas_vocabulario_instalado() {
    local estado_esperado=$1

    paso 'trigger y política homónimos debilitados no engañan al DOWN'
    psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DROP TRIGGER control_vocabulario_estados_rrhh_inmutable ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1;
CREATE TRIGGER control_vocabulario_estados_rrhh_inmutable
BEFORE UPDATE ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
FOR EACH ROW EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();
SQL
    esperar_fallo 'down con trigger homónimo UPDATE-only' 55000 \
        'estructura derivada impide revertir estados RRHH' \
        down_vocabulario
    comprobar_estado_000041a "$estado_esperado" \
        'rechazo transaccional del trigger homónimo'
    psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DROP TRIGGER control_vocabulario_estados_rrhh_inmutable ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1;
CREATE TRIGGER control_vocabulario_estados_rrhh_inmutable
BEFORE UPDATE OR DELETE ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
FOR EACH ROW EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();
DROP POLICY propietario_total ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1;
CREATE POLICY propietario_total ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
AS RESTRICTIVE
FOR UPDATE
TO vec_contratacion_temporal_consultor_rrhh
USING (false)
WITH CHECK (false);
SQL
    esperar_fallo 'down con política homónima alterada' 55000 \
        'estado incompatible para revertir estados RRHH' \
        down_vocabulario
    comprobar_estado_000041a "$estado_esperado" \
        'rechazo transaccional de la política homónima'
    psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DROP POLICY propietario_total ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1;
CREATE POLICY propietario_total ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
TO vec_contratacion_temporal_propietario
USING (true)
WITH CHECK (true);
SQL
    comprobar_estado_000041a "$estado_esperado" \
        'trigger y política homónimos restaurados'

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

    paso 'ACL de fila se detecta y ACL de array se prohíbe en PostgreSQL 18.4'
    esperar_fallo 'down con ACL del tipo compuesto de fila' 55000 \
        'estructura derivada impide revertir estados RRHH' \
        down_con_acl_tipo_fila
    comprobar_estado_000041a "$estado_esperado" \
        'ACL del tipo de fila revertida transaccionalmente'
    esperar_fallo 'GRANT directo sobre el tipo array dependiente' 0LP01 \
        'cannot set privileges of array types' \
        conceder_acl_tipo_array
    comprobar_estado_000041a "$estado_esperado" \
        'rechazo nativo de ACL del tipo array'

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

probar_derivas_antes_up() {
    local estado_esperado=$1

    esperar_fallo 'up con trigger de publicación homónimo UPDATE-only' 55000 \
        'estructura heredada de estados RRHH incompatible' \
        up_con_disparador_publicacion_homonimo_debil
    comprobar_estado_antes_up_000041a "$estado_esperado" \
        'trigger heredado homónimo revertido transaccionalmente'
    esperar_fallo 'up con función de inmutabilidad no-op' 55000 \
        'estructura heredada de estados RRHH incompatible' \
        up_con_funcion_inmutable_noop
    comprobar_estado_antes_up_000041a "$estado_esperado" \
        'función heredada no-op revertida transaccionalmente'
}
