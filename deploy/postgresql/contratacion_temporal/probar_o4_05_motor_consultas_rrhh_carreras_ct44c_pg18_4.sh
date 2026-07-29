#!/usr/bin/env bash
set -Eeuo pipefail

directorio="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
    pwd -P
)"
patron_cursor_argumento='--env VEC_''CURSOR[^ ]*='
estado_cursores_argumentos=0
rg -q -- "$patron_cursor_argumento" "${BASH_SOURCE[0]}" ||
    estado_cursores_argumentos=$?
if (( estado_cursores_argumentos != 1 )); then
    printf 'el guion no protege los cursores de los argumentos de Docker\n' >&2
    exit 1
fi
# CT44B deja vivo el PostgreSQL 18.4 efímero y el motor completo ya probado.
# shellcheck disable=SC1091
source "$directorio/probar_o4_05_motor_atomico_consultas_ct44b_pg18_4.sh"
: "${contenedor:?el ejecutor CT44B debe exponer PostgreSQL}"

paso() {
    printf '[O4-05:CT-000044C:PG18.4] %s\n' "$1"
}

crear_familia_motor() {
    local caso=$1
    local token
    preparar_vector "$caso" cuadro
    token="$(invocar_cuadro "$caso")"
    if [[ ! $token =~ ^[A-Za-z0-9_-]{43}$ ]]; then
        printf 'el motor no creó la familia CT44C %s\n' "$caso" >&2
        return 1
    fi
    printf '%s' "$token"
}

preparar_continuacion() {
    local caso=$1
    local token=$2
    preparar_vector "$caso" cuadro
    ajustar_cursor "$caso" "$token"
}

esperar_senal() {
    local senal=$1
    local _
    for _ in {1..200}; do
        if [[ "$(valor "SELECT pg_catalog.count(*)
              FROM pg_catalog.pg_locks
             WHERE locktype='advisory'
               AND granted
               AND classid=0
               AND objid=$senal")" == 1 ]]; then
            return 0
        fi
        sleep 0.02
    done
    printf 'no apareció la señal transaccional CT44C %s\n' "$senal" >&2
    return 1
}

motor_transaccion() {
    local caso=$1
    local token=$2
    local senal=$3
    local retencion=$4
    VEC_CURSOR="$token" docker exec --interactive \
        --env VEC_CASO="$caso" \
        --env VEC_CURSOR \
        --env VEC_SENAL="$senal" \
        --env VEC_RETENCION="$retencion" \
        "$contenedor" psql -XqAt \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_c2d2_registro_runtime --dbname postgres <<'SQL'
\getenv caso VEC_CASO
\getenv cursor VEC_CURSOR
\getenv senal VEC_SENAL
\getenv retencion VEC_RETENCION
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SELECT vec_contratacion_temporal
       .prueba_invocar_motor_cuadro_controlado_ct44b(
           $1, $2, 'organizacion:diputacion-granada', false
       )
\parse ejecutar_motor
\bind_named ejecutar_motor :caso :cursor
\g
SELECT pg_catalog.pg_advisory_xact_lock($1::bigint)
\parse marcar_motor
\bind_named marcar_motor :senal
\g /dev/null
SELECT pg_catalog.pg_sleep($1::double precision)
\parse retener_motor
\bind_named retener_motor :retencion
\g /dev/null
COMMIT;
SQL
}

revocar_transaccion() {
    local token=$1
    local etiqueta=$2
    local senal=$3
    local retencion=$4
    local confirmar=$5
    VEC_CURSOR="$token" docker exec --interactive \
        --env VEC_CURSOR \
        --env VEC_ETIQUETA="$etiqueta" \
        --env VEC_SENAL="$senal" \
        --env VEC_RETENCION="$retencion" \
        --env VEC_CONFIRMAR="$confirmar" \
        "$contenedor" psql -XqAt \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username postgres --dbname postgres <<'SQL'
\getenv cursor VEC_CURSOR
\getenv etiqueta VEC_ETIQUETA
\getenv senal VEC_SENAL
\getenv retencion VEC_RETENCION
\getenv confirmar VEC_CONFIRMAR
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL TimeZone='UTC';
SELECT vec_contratacion_temporal
       .prueba_revocar_cursor_cuadro_ct44c($1, $2)
\parse revocar_familia
\bind_named revocar_familia :cursor :etiqueta
\g
SELECT pg_catalog.pg_advisory_xact_lock($1::bigint)
\parse marcar_revocacion
\bind_named marcar_revocacion :senal
\g /dev/null
SELECT pg_catalog.pg_sleep($1::double precision)
\parse retener_revocacion
\bind_named retener_revocacion :retencion
\g /dev/null
\if :confirmar
COMMIT;
\else
ROLLBACK;
\endif
SQL
}

retener_control_causal() {
    local token=$1
    local senal=$2
    local retencion=$3
    VEC_CURSOR="$token" docker exec --interactive \
        --env VEC_CURSOR \
        --env VEC_SENAL="$senal" \
        --env VEC_RETENCION="$retencion" \
        "$contenedor" psql -XqAt \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username postgres --dbname postgres <<'SQL'
\getenv cursor VEC_CURSOR
\getenv senal VEC_SENAL
\getenv retencion VEC_RETENCION
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL TimeZone='UTC';
WITH entrada AS (
    SELECT pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to($1, 'UTF8')
    ), 'hex') AS huella
)
SELECT causal.familia_ref
  FROM entrada
  JOIN vec_contratacion_temporal.cursor_cuadro_rrhh cursor
    ON cursor.token_huella_sha256 = entrada.huella
  JOIN vec_contratacion_temporal
       .control_causal_familia_cursor_rrhh causal
    USING (familia_ref)
 FOR UPDATE OF causal
\parse bloquear_control
\bind_named bloquear_control :cursor
\g /dev/null
SELECT pg_catalog.pg_advisory_xact_lock($1::bigint)
\parse marcar_control
\bind_named marcar_control :senal
\g /dev/null
SELECT pg_catalog.pg_sleep($1::double precision)
\parse retener_control
\bind_named retener_control :retencion
\g /dev/null
COMMIT;
SQL
}

estado_familia_cursor() {
    local token=$1
    VEC_CURSOR="$token" docker exec --interactive \
        --env VEC_CURSOR \
        "$contenedor" psql -XqAt \
        --set ON_ERROR_STOP=1 \
        --username postgres --dbname postgres <<'SQL'
\getenv cursor VEC_CURSOR
WITH entrada AS (
    SELECT pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to($1, 'UTF8')
    ), 'hex') AS huella
),
familia AS (
    SELECT cursor.familia_ref, entrada.huella
      FROM entrada
      JOIN vec_contratacion_temporal.cursor_cuadro_rrhh cursor
        ON cursor.token_huella_sha256 = entrada.huella
)
SELECT pg_catalog.concat_ws('|',
           causal.revision,
           (SELECT pg_catalog.count(*)
              FROM vec_contratacion_temporal
                   .revocacion_familia_cursor_rrhh revocacion
             WHERE revocacion.familia_ref = familia.familia_ref),
           (SELECT pg_catalog.count(*)
              FROM vec_contratacion_temporal
                   .consumo_cursor_cuadro_rrhh consumo
             WHERE consumo.token_huella_sha256 = familia.huella),
           (SELECT pg_catalog.count(*)
              FROM vec_contratacion_temporal.cursor_cuadro_rrhh hijo
             WHERE hijo.padre_token_huella_sha256 = familia.huella)
       )
  FROM familia
  JOIN vec_contratacion_temporal
       .control_causal_familia_cursor_rrhh causal
    USING (familia_ref)
\parse consultar_estado
\bind_named consultar_estado :cursor
\g
SQL
}

comprobar_serializacion_motor() {
    local archivo_salida=$1
    if ! rg -q \
        'ERROR: +40001: could not serialize access due to concurrent update' \
        "$archivo_salida"; then
        sed -n '1,30p' "$archivo_salida" >&2
        return 1
    fi
}

paso 'instalación de la ayuda mínima de prueba para revocación'
archivo \
    contratacion_temporal/pruebas_sql/o405_motor_consultas_rrhh_carreras_ct44c.sql
psql_admin <<'SQL' >/dev/null
DO $prueba$
DECLARE
    v_motor text;
    v_resolver text;
    v_efectos text;
BEGIN
    SELECT prosrc INTO STRICT v_motor
      FROM pg_catalog.pg_proc
     WHERE oid =
       'vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3)'::regprocedure;
    SELECT prosrc INTO STRICT v_resolver
      FROM pg_catalog.pg_proc
     WHERE oid =
       'vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text,text,numeric,text,text)'::regprocedure;
    SELECT prosrc INTO STRICT v_efectos
      FROM pg_catalog.pg_proc
     WHERE oid =
       'vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1,vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,bytea,vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2)'::regprocedure;
    IF pg_catalog.strpos(
           v_motor, 'resolver_estado_cursor_cuadro_rrhh_v1'
       ) = 0
       OR pg_catalog.strpos(
           v_resolver,
           'WHERE causal.familia_ref = v_familia_ref'
       ) = 0
       OR pg_catalog.strpos(v_resolver, 'FOR UPDATE') = 0
       OR pg_catalog.strpos(
           v_efectos,
           'WHERE familia.familia_ref = p_estado.familia_ref'
       ) = 0
       OR pg_catalog.strpos(v_efectos, 'FOR UPDATE OF causal') = 0 THEN
        RAISE EXCEPTION 'cerrojo causal CT44C no acreditado';
    END IF;
END
$prueba$;
SQL

tokens_ct44c=()

paso 'revocación primero: el motor espera el COMMIT y deniega sin efectos'
token_revocacion_primero="$(
    crear_familia_motor ct44c_revocacion_primero_padre
)"
tokens_ct44c+=("$token_revocacion_primero")
preparar_continuacion \
    ct44c_revocacion_primero_motor "$token_revocacion_primero"
salida_revocacion_primero="$(
    mktemp "${TMPDIR:-/tmp}/vec-ct44c-revocacion-primero.XXXXXX"
)"
salida_motor_rechazado="$(
    mktemp "${TMPDIR:-/tmp}/vec-ct44c-motor-rechazado.XXXXXX"
)"
temporales+=(
    "$salida_revocacion_primero" "$salida_motor_rechazado"
)
revocar_transaccion "$token_revocacion_primero" primero \
    440401 0.65 true >"$salida_revocacion_primero" 2>&1 &
pid_revocacion_primero=$!
if ! esperar_senal 440401; then
    wait "$pid_revocacion_primero" || true
    sed -n '1,40p' "$salida_revocacion_primero" >&2
    exit 1
fi
inicio_ns="$(date +%s%N)"
estado_motor_rechazado=0
motor_transaccion ct44c_revocacion_primero_motor \
    "$token_revocacion_primero" 440402 0 \
    >"$salida_motor_rechazado" 2>&1 || estado_motor_rechazado=$?
fin_ns="$(date +%s%N)"
wait "$pid_revocacion_primero"
if (( estado_motor_rechazado == 0 || fin_ns - inicio_ns < 200000000 )); then
    sed -n '1,30p' "$salida_motor_rechazado" >&2
    printf 'el motor CT44C no quedó ordenado tras la revocación\n' >&2
    exit 1
fi
comprobar_serializacion_motor "$salida_motor_rechazado"
[[ "$(estado_familia_cursor "$token_revocacion_primero")" == '1|1|0|0' ]]
[[ "$(efectos_decision ct44c_revocacion_primero_motor)" == '0|0|0|0' ]]

paso 'motor primero: la revocación espera su COMMIT y ambos dejan un orden'
token_motor_primero="$(
    crear_familia_motor ct44c_motor_primero_padre
)"
tokens_ct44c+=("$token_motor_primero")
preparar_continuacion ct44c_motor_primero "$token_motor_primero"
salida_motor_primero="$(
    mktemp "${TMPDIR:-/tmp}/vec-ct44c-motor-primero.XXXXXX"
)"
temporales+=("$salida_motor_primero")
motor_transaccion ct44c_motor_primero "$token_motor_primero" \
    440403 0.65 >"$salida_motor_primero" 2>&1 &
pid_motor_primero=$!
esperar_senal 440403
inicio_ns="$(date +%s%N)"
revocar_transaccion "$token_motor_primero" motor_primero \
    440404 0 true >/dev/null
fin_ns="$(date +%s%N)"
wait "$pid_motor_primero"
if (( fin_ns - inicio_ns < 200000000 )); then
    printf 'la revocación CT44C no esperó el COMMIT del motor\n' >&2
    exit 1
fi
[[ "$(estado_familia_cursor "$token_motor_primero")" == '1|1|1|1' ]]
[[ "$(efectos_decision ct44c_motor_primero)" == '1|1|1|1' ]]

paso 'rollback de revocación: el motor espera y no observa rechazo falso'
token_revocacion_rollback="$(
    crear_familia_motor ct44c_revocacion_rollback_padre
)"
tokens_ct44c+=("$token_revocacion_rollback")
preparar_continuacion \
    ct44c_revocacion_rollback_motor "$token_revocacion_rollback"
salida_revocacion_rollback="$(
    mktemp "${TMPDIR:-/tmp}/vec-ct44c-revocacion-rollback.XXXXXX"
)"
temporales+=("$salida_revocacion_rollback")
revocar_transaccion "$token_revocacion_rollback" rollback \
    440405 0.65 false >"$salida_revocacion_rollback" 2>&1 &
pid_revocacion_rollback=$!
esperar_senal 440405
inicio_ns="$(date +%s%N)"
motor_transaccion ct44c_revocacion_rollback_motor \
    "$token_revocacion_rollback" 440406 0 >/dev/null
fin_ns="$(date +%s%N)"
wait "$pid_revocacion_rollback"
if (( fin_ns - inicio_ns < 200000000 )); then
    printf 'el motor CT44C no esperó el rollback de revocación\n' >&2
    exit 1
fi
[[ "$(estado_familia_cursor "$token_revocacion_rollback")" == '0|0|1|1' ]]
[[ "$(efectos_decision ct44c_revocacion_rollback_motor)" == '1|1|1|1' ]]

paso 'dos familias distintas progresan sin un cerrojo causal común'
token_paralelo_a="$(crear_familia_motor ct44c_paralelo_a_padre)"
token_paralelo_b="$(crear_familia_motor ct44c_paralelo_b_padre)"
tokens_ct44c+=("$token_paralelo_a" "$token_paralelo_b")
preparar_continuacion ct44c_paralelo_b "$token_paralelo_b"
salida_paralelo_a="$(
    mktemp "${TMPDIR:-/tmp}/vec-ct44c-paralelo-a.XXXXXX"
)"
temporales+=("$salida_paralelo_a")
retener_control_causal "$token_paralelo_a" \
    440407 2.0 >"$salida_paralelo_a" 2>&1 &
pid_paralelo_a=$!
esperar_senal 440407
motor_transaccion ct44c_paralelo_b "$token_paralelo_b" \
    440408 0 >/dev/null
if ! kill -0 "$pid_paralelo_a" 2>/dev/null; then
    printf 'familias CT44C distintas compartieron espera causal\n' >&2
    exit 1
fi
wait "$pid_paralelo_a"
[[ "$(estado_familia_cursor "$token_paralelo_a")" == '0|0|0|0' ]]
[[ "$(estado_familia_cursor "$token_paralelo_b")" == '0|0|1|1' ]]
[[ "$(efectos_decision ct44c_paralelo_b)" == '1|1|1|1' ]]

paso 'tokens claros ausentes de persistencia y registros PostgreSQL'
for token_ct44c in "${tokens_ct44c[@]}"; do
    [[ "$(contar_apariciones_tokens "$token_ct44c" "$token_ct44c")" == 0 ]]
    if docker logs "$contenedor" 2>&1 |
        rg -Fq -- "$token_ct44c"; then
        printf 'un token CT44C apareció en el registro PostgreSQL\n' >&2
        exit 1
    fi
done

paso 'retirada de la ayuda de prueba y cierre verde'
psql_admin <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DROP FUNCTION
vec_contratacion_temporal.prueba_revocar_cursor_cuadro_ct44c(text, text);
COMMIT;
SQL
unset token_ct44c tokens_ct44c
unset token_revocacion_primero token_motor_primero
unset token_revocacion_rollback token_paralelo_a token_paralelo_b

paso 'carreras del motor completo CT-000044C superadas'
