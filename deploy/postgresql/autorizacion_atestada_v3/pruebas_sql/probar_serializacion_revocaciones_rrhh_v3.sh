#!/usr/bin/env bash

# Matriz causal cargada por el runner PostgreSQL 18.4 de prueba de consumo.
probar_serializacion_revocaciones_rrhh_v3() {
    local revision_antes
    local raices_antes
    local entrada_rollback
    local salida_rollback
    local salida_prueba_rollback
    local pid_revocador_rollback
    local pid_prueba_rollback
    local entrada_prueba_primero
    local salida_prueba_primero
    local salida_revocador_despues
    local pid_prueba_primero
    local pid_revocador_despues
    local entrada_revocador_primero
    local salida_revocador_primero
    local salida_prueba_obsoleta
    local pid_revocador_primero
    local pid_prueba_obsoleta

    : "${contenedor:?contenedor PostgreSQL no definido}"
    : "${temporales:?directorio temporal no definido}"

    paso 'vectores válidos previos a la matriz causal de revocación'
    preparar rollback_revocacion detalle
    preparar prueba_primero cuadro
    preparar revocador_primero detalle
    consumir rollback_revocacion detalle >/dev/null
    consumir prueba_primero cuadro >/dev/null
    consumir revocador_primero detalle >/dev/null

    paso 'revocador encolado primero y ROLLBACK no deniega la prueba'
    revision_antes="$(valor "SELECT revision
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno
     WHERE control_id")"
    raices_antes="$(valor "SELECT count(*)
      FROM vec_autorizacion_atestada_v3.revocacion_raiz")"
    entrada_rollback="${temporales}/revocacion-rollback.in"
    salida_rollback="${temporales}/revocacion-rollback.out"
    salida_prueba_rollback="${temporales}/prueba-rollback.out"
    mkfifo "${entrada_rollback}"
    docker exec --interactive "${contenedor}" \
        psql -XAtq --set ON_ERROR_STOP=1 --username postgres \
        --dbname postgres <"${entrada_rollback}" \
        >"${salida_rollback}" 2>&1 &
    pid_revocador_rollback=$!
    exec 9>"${entrada_rollback}"
    printf '%s\n' \
        "BEGIN;" \
        "SET application_name='revocador_rollback_raiz';" \
        "SET LOCAL statement_timeout='15s';" \
        "SET LOCAL idle_in_transaction_session_timeout='20s';" \
        "SET ROLE vec_autorizacion_atestada_v3_propietario;" \
        "INSERT INTO vec_autorizacion_atestada_v3.revocacion_raiz VALUES ('raiz-o205-1',1,clock_timestamp(),'motivo:rollback-causal','acto:revocacion:rollback-raiz',clock_timestamp());" \
        >&9
    esperar_sesion revocador_rollback_raiz 'idle in transaction'
    docker exec --interactive "${contenedor}" \
        psql -XAtq --set ON_ERROR_STOP=1 \
        --username vec_consulta_rrhh_runtime --dbname postgres \
        >"${salida_prueba_rollback}" 2>&1 <<'SQL' &
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET application_name='prueba_tras_rollback';
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT count(*)
  FROM vec_contratacion_temporal
       .prueba_evidencia_consumo_consulta_rrhh_v3(
    'rollback_revocacion','detalle',''
  );
COMMIT;
SELECT 'prueba_tras_rollback_confirmada';
SQL
    pid_prueba_rollback=$!
    esperar_sesion prueba_tras_rollback active Lock
    printf '%s\n' \
        "ROLLBACK;" \
        "SELECT 'revocacion_revertida';" \
        >&9
    exec 9>&-
    wait "${pid_revocador_rollback}"
    wait "${pid_prueba_rollback}"
    grep -qx 'revocacion_revertida' "${salida_rollback}"
    grep -qx 'prueba_tras_rollback_confirmada' \
        "${salida_prueba_rollback}"
    [[ "$(valor "SELECT revision
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno
     WHERE control_id")" == "${revision_antes}" ]]
    [[ "$(valor "SELECT count(*)
      FROM vec_autorizacion_atestada_v3.revocacion_raiz")" == \
        "${raices_antes}" ]]

    paso 'prueba primero: la revocación espera hasta el COMMIT'
    revision_antes="$(valor "SELECT revision
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno
     WHERE control_id")"
    entrada_prueba_primero="${temporales}/prueba-primero.in"
    salida_prueba_primero="${temporales}/prueba-primero.out"
    salida_revocador_despues="${temporales}/revocador-despues.out"
    mkfifo "${entrada_prueba_primero}"
    docker exec --interactive "${contenedor}" \
        psql -XAtq --set ON_ERROR_STOP=1 \
        --username vec_consulta_rrhh_runtime --dbname postgres \
        <"${entrada_prueba_primero}" \
        >"${salida_prueba_primero}" 2>&1 &
    pid_prueba_primero=$!
    exec 9>"${entrada_prueba_primero}"
    printf '%s\n' \
        "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;" \
        "SET application_name='orden_prueba_primero';" \
        "SET LOCAL TimeZone='UTC';" \
        "SET LOCAL statement_timeout='15s';" \
        "SET LOCAL idle_in_transaction_session_timeout='20s';" \
        "SELECT count(*) FROM vec_contratacion_temporal.prueba_evidencia_consumo_consulta_rrhh_v3('prueba_primero','cuadro','');" \
        >&9
    esperar_sesion orden_prueba_primero 'idle in transaction'
    docker exec --interactive "${contenedor}" \
        psql -XAtq --set ON_ERROR_STOP=1 --username postgres \
        --dbname postgres >"${salida_revocador_despues}" 2>&1 <<'SQL' &
SET application_name='revocador_despues_prueba';
BEGIN;
SET LOCAL statement_timeout='15s';
SET ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.revocacion_clave_capacidad
VALUES (
  'clave-consulta-cuadro-rrhh-v3',2,clock_timestamp(),
  'motivo:orden-prueba-primero',
  'acto:revocacion:orden-prueba-primero',clock_timestamp()
);
COMMIT;
SELECT 'revocacion_confirmada';
SQL
    pid_revocador_despues=$!
    esperar_sesion revocador_despues_prueba active Lock
    kill -0 "${pid_revocador_despues}"
    printf '%s\n' \
        "COMMIT;" \
        "SELECT 'prueba_confirmada';" \
        >&9
    exec 9>&-
    wait "${pid_prueba_primero}"
    wait "${pid_revocador_despues}"
    grep -qx 'prueba_confirmada' "${salida_prueba_primero}"
    grep -qx 'revocacion_confirmada' "${salida_revocador_despues}"
    [[ "$(valor "SELECT revision
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno
     WHERE control_id")" == "$((revision_antes + 1))" ]]
    [[ "$(valor "SELECT count(*)
      FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad
     WHERE clave_id = concat('clave-', 'consulta-cuadro-rrhh-v3')
       AND version=2")" == '1' ]]
    esperar_fallo 'revocación confirmada impide una prueba posterior' \
        probar prueba_primero cuadro

    paso 'revocador primero: COMMIT causal fuerza SQLSTATE 40001'
    revision_antes="$(valor "SELECT revision
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno
     WHERE control_id")"
    entrada_revocador_primero="${temporales}/revocador-primero.in"
    salida_revocador_primero="${temporales}/revocador-primero.out"
    salida_prueba_obsoleta="${temporales}/prueba-obsoleta.out"
    mkfifo "${entrada_revocador_primero}"
    docker exec --interactive "${contenedor}" \
        psql -XAtq --set ON_ERROR_STOP=1 --username postgres \
        --dbname postgres <"${entrada_revocador_primero}" \
        >"${salida_revocador_primero}" 2>&1 &
    pid_revocador_primero=$!
    exec 9>"${entrada_revocador_primero}"
    printf '%s\n' \
        "SET application_name='orden_revocador_primero';" \
        "BEGIN;" \
        "SET LOCAL statement_timeout='15s';" \
        "SET LOCAL idle_in_transaction_session_timeout='20s';" \
        "SET ROLE vec_autorizacion_atestada_v3_propietario;" \
        "INSERT INTO vec_autorizacion_atestada_v3.revocacion_configuracion VALUES ('configuracion-o205-1',clock_timestamp(),'motivo:orden-revocador-primero','acto:revocacion:orden-revocador-primero',clock_timestamp());" \
        "SELECT 'revocacion_preparada';" \
        >&9
    esperar_sesion orden_revocador_primero 'idle in transaction'
    docker exec --interactive "${contenedor}" \
        psql -XAtq --set ON_ERROR_STOP=1 \
        --username vec_consulta_rrhh_runtime --dbname postgres \
        >"${salida_prueba_obsoleta}" 2>&1 <<'SQL' &
\set VERBOSITY verbose
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET application_name='prueba_snapshot_obsoleto';
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT count(*)
  FROM vec_contratacion_temporal
       .prueba_evidencia_consumo_consulta_rrhh_v3(
    'revocador_primero','detalle',''
  );
COMMIT;
SQL
    pid_prueba_obsoleta=$!
    esperar_sesion prueba_snapshot_obsoleto active Lock
    printf '%s\n' \
        "COMMIT;" \
        "SELECT 'revocacion_confirmada';" \
        >&9
    exec 9>&-
    wait "${pid_revocador_primero}"
    if wait "${pid_prueba_obsoleta}"; then
        printf 'el snapshot obsoleto produjo un falso éxito\n' >&2
        cat "${salida_prueba_obsoleta}" >&2
        return 1
    fi
    grep -Eq '(^|[^0-9])40001([^0-9]|$)' "${salida_prueba_obsoleta}"
    if grep -Eq '55P03|40P01' "${salida_prueba_obsoleta}"; then
        printf 'la prueba causal cayó por timeout o deadlock\n' >&2
        cat "${salida_prueba_obsoleta}" >&2
        return 1
    fi
    grep -qx 'revocacion_preparada' "${salida_revocador_primero}"
    grep -qx 'revocacion_confirmada' "${salida_revocador_primero}"
    [[ "$(valor "SELECT revision
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno
     WHERE control_id")" == "$((revision_antes + 1))" ]]
    [[ "$(valor "SELECT count(*)
      FROM vec_autorizacion_atestada_v3.revocacion_configuracion
     WHERE configuracion_revision='configuracion-o205-1'")" == '1' ]]
}
