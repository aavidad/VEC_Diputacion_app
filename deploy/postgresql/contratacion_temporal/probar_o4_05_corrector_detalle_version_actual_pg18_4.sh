#!/usr/bin/env bash
set -Eeuo pipefail

directorio="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
    pwd -P
)"

# Reutiliza la línea base real CT-000039..CT-000042 y su PostgreSQL 18.4
# fijado por resumen. El contenedor queda bajo la misma limpieza automática.
# shellcheck disable=SC1091
source "$directorio/probar_o4_05_canones_resultado_recibo_rrhh_pg18_4.sh"
: "${contenedor:?el ejecutor CT42 debe exponer PostgreSQL}"

paso() {
    printf '[O4-05:CT-000043A:PG18.4] %s\n' "$1"
}

firma='vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v2(vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2,vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2,vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
huella_anterior='39f819926106443051d91a07f9158803cd58373ce9b60b050abe1a15aa6e5c7f'
huella_corregida='3822cb70994240e2c67784ce2b72472d1e4914c209e9166425cc1c3e3c575a46'

estado_corrector() {
    valor "SELECT pg_catalog.concat_ws('|',
        cobertura.version_esquema::text,
        consultas.version_esquema::text,
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            funcion.prosrc, 'UTF8'
        )), 'hex'),
        funcion.proowner::pg_catalog.regrole::text,
        funcion.prosecdef::text,
        funcion.provolatile::text,
        funcion.proparallel::text,
        funcion.proconfig::text
    )
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4 cobertura
      CROSS JOIN
           vec_contratacion_temporal.control_migracion_consultas_rrhh consultas
      CROSS JOIN pg_catalog.pg_proc funcion
     WHERE cobertura.control AND consultas.control
       AND funcion.oid = '$firma'::pg_catalog.regprocedure"
}

comprobar_estado_corrector() {
    local huella=$1
    local contexto=$2
    local esperado
    local obtenido
    esperado="23|7|$huella|vec_contratacion_temporal_propietario|true|v|u|{search_path=pg_catalog,row_security=on,TimeZone=UTC,lock_timeout=1s,statement_timeout=12s}"
    obtenido="$(estado_corrector)"
    if [[ $obtenido != "$esperado" ]]; then
        printf 'estado CT43A alterado tras %s\n' "$contexto" >&2
        printf 'esperado=%s\nobtenido=%s\n' "$esperado" "$obtenido" >&2
        return 1
    fi
}

preparar_detalle() {
    local caso=$1
    local version=$2
    local recalcular=${3:-true}
    psql_admin --command "
        SELECT public.preparar_vector_cierre_ct43('$caso','detalle');
        SELECT public.ajustar_version_observada_ct43a(
            '$caso', $version, $recalcular
        )" >/dev/null
}

invocar_cierre() {
    local caso=$1
    local configuracion=${2:-}
    psql_runtime --command "
        BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
        SET LOCAL TimeZone='UTC';
        SET LOCAL statement_timeout='15s';
        SET LOCAL idle_in_transaction_session_timeout='20s';
        ${configuracion}
        SELECT (
          vec_contratacion_temporal
          .prueba_cerrar_resultado_recibo_ct43('$caso')
        ).recibo_sello_sha256;
        COMMIT"
}

estado_efectos() {
    local caso=$1
    valor "SELECT pg_catalog.concat_ws('|',
        (SELECT pg_catalog.count(*)
           FROM vec_autorizacion_atestada_v3.consumo_decision_v3
          WHERE decision_ref = 'decision:consulta-rrhh:$caso'),
        (SELECT pg_catalog.count(*)
           FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
          WHERE decision_ref = 'decision:consulta-rrhh:$caso'),
        (SELECT pg_catalog.count(*)
           FROM vec_contratacion_temporal.registro_acceso_rrhh
          WHERE decision_ref = 'decision:consulta-rrhh:$caso'),
        (SELECT pg_catalog.count(*)
           FROM vec_contratacion_temporal
                .prueba_resultado_recibo_rrhh_v2
          WHERE decision_ref = 'decision:consulta-rrhh:$caso')
    )"
}

probar_deriva() {
    local descripcion=$1
    local mutacion=$2
    local restauracion=$3
    psql_admin --command "$mutacion" >/dev/null
    esperar_fallo "$descripcion" 55000 \
        'definición corregida de detalle RRHH incompatible' \
        archivo \
        contratacion_temporal/migraciones/000043a_detalle_version_actual.down.sql
    psql_admin --command "$restauracion" >/dev/null
    comprobar_estado_corrector "$huella_corregida" "$descripcion"
}

paso 'dependencia causal VEC-AD-3 000003..000006'
for ruta in \
    autorizacion_atestada_v3/migraciones/000003_consumidor_consulta_cuadro_rrhh_v3.up.sql \
    autorizacion_atestada_v3/migraciones/000004_consumidor_consulta_detalle_rrhh_v3.up.sql \
    autorizacion_atestada_v3/migraciones/000005_revalidacion_final_consultas_rrhh_v3.up.sql \
    autorizacion_atestada_v3/migraciones/000006_prueba_consumo_consultas_rrhh_v3.up.sql
do
    archivo "$ruta"
done

paso 'instalación CT-000043 sobre barreras 22/6'
(
    cd /tmp
    psql_admin --file \
        /repo/contratacion_temporal/migraciones/000043_prueba_resultado_recibo_rrhh.up.sql \
        >/dev/null
)
comprobar_estado_corrector "$huella_anterior" 'instalación CT43'

paso 'UP, reentrada, bloqueo del DOWN padre y ciclo DOWN/UP'
archivo \
    contratacion_temporal/migraciones/000043a_detalle_version_actual.up.sql
comprobar_estado_corrector "$huella_corregida" 'primer UP'
esperar_fallo 'reentrada UP CT43A' 55000 \
    'definición anterior de detalle RRHH incompatible' \
    archivo \
    contratacion_temporal/migraciones/000043a_detalle_version_actual.up.sql
comprobar_estado_corrector "$huella_corregida" 'reentrada UP'
esperar_fallo 'DOWN CT43 con CT43A vigente' 55000 \
    'catálogo de prueba durable RRHH incompatible' \
    archivo \
    contratacion_temporal/migraciones/000043_prueba_resultado_recibo_rrhh.down.sql
comprobar_estado_corrector "$huella_corregida" 'DOWN padre bloqueado'
archivo \
    contratacion_temporal/migraciones/000043a_detalle_version_actual.down.sql
comprobar_estado_corrector "$huella_anterior" 'primer DOWN'
esperar_fallo 'reentrada DOWN CT43A' 55000 \
    'definición corregida de detalle RRHH incompatible' \
    archivo \
    contratacion_temporal/migraciones/000043a_detalle_version_actual.down.sql
comprobar_estado_corrector "$huella_anterior" 'reentrada DOWN'
archivo \
    contratacion_temporal/migraciones/000043a_detalle_version_actual.up.sql
comprobar_estado_corrector "$huella_corregida" 'segundo UP'

paso 'cuerpo, atributos, ACL, comentarios y dependencias quedan sellados'
probar_deriva 'coste de función CT43A' \
    "ALTER FUNCTION $firma COST 37" \
    "ALTER FUNCTION $firma COST 100"
probar_deriva 'ACL de función CT43A' \
    "GRANT EXECUTE ON FUNCTION $firma TO vec_contratacion_temporal_consultor_rrhh" \
    "REVOKE EXECUTE ON FUNCTION $firma FROM vec_contratacion_temporal_consultor_rrhh"
probar_deriva 'comentario de función CT43A' \
    "COMMENT ON FUNCTION $firma IS 'deriva sintética CT43A'" \
    "COMMENT ON FUNCTION $firma IS NULL"
psql_admin <<SQL >/dev/null
CREATE VIEW public.dependencia_futura_ct43a AS
SELECT vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v2(
    NULL::vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2,
    NULL::vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2,
    NULL::vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,
    NULL::bytea, NULL::bytea, NULL::bytea, NULL::bytea,
    NULL::numeric, NULL::numeric, NULL::bytea, NULL::bytea,
    NULL::bytea, NULL::bytea
) AS resultado;
SQL
esperar_fallo 'dependencia futura CT43A' 55000 \
    'definición corregida de detalle RRHH incompatible' \
    archivo \
    contratacion_temporal/migraciones/000043a_detalle_version_actual.down.sql
psql_admin --command \
    'DROP VIEW public.dependencia_futura_ct43a' >/dev/null
comprobar_estado_corrector "$huella_corregida" 'dependencia futura'

psql_admin <<'SQL' >/dev/null
DO $deriva$
DECLARE
    v_firma regprocedure :=
        'vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v2(vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2,vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2,vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    v_definicion text := pg_catalog.pg_get_functiondef(v_firma);
BEGIN
    v_definicion := pg_catalog.replace(
        v_definicion,
        'cierre de prueba RRHH inválido',
        'cierre de prueba RRHH inválido alterado'
    );
    EXECUTE v_definicion;
END
$deriva$;
SQL
esperar_fallo 'cuerpo de función CT43A' 55000 \
    'definición corregida de detalle RRHH incompatible' \
    archivo \
    contratacion_temporal/migraciones/000043a_detalle_version_actual.down.sql
psql_admin <<'SQL' >/dev/null
DO $restauracion$
DECLARE
    v_firma regprocedure :=
        'vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v2(vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2,vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2,vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    v_definicion text := pg_catalog.pg_get_functiondef(v_firma);
BEGIN
    v_definicion := pg_catalog.replace(
        v_definicion,
        'cierre de prueba RRHH inválido alterado',
        'cierre de prueba RRHH inválido'
    );
    EXECUTE v_definicion;
END
$restauracion$;
SQL
comprobar_estado_corrector "$huella_corregida" 'cuerpo restaurado'

paso 'datos sintéticos VEC, Identidad y auxiliares de CT43A'
psql_admin --command \
    'GRANT EXECUTE ON FUNCTION public.gen_random_bytes(integer) TO vec_autorizacion_atestada_v3_propietario' \
    >/dev/null
archivo autorizacion_atestada_v3/pruebas_sql/preparar_entorno_o2_05.sql
archivo autorizacion_atestada_v3/pruebas_sql/consultas_rrhh_v3.sql
psql_admin --command \
    'REVOKE EXECUTE ON FUNCTION public.gen_random_bytes(integer) FROM vec_autorizacion_atestada_v3_propietario' \
    >/dev/null
archivo \
    autorizacion_atestada_v3/pruebas_sql/revalidacion_final_consultas_rrhh_v3.sql
archivo \
    autorizacion_atestada_v3/pruebas_sql/prueba_consumo_consultas_rrhh_v3.sql
archivo \
    contratacion_temporal/pruebas_sql/o405_prueba_resultado_recibo_rrhh_datos_sinteticos.sql
archivo \
    contratacion_temporal/pruebas_sql/o405_corrector_detalle_version_actual.sql

paso 'versión 0 conserva canon VEC y registra la versión actual positiva'
preparar_detalle ct43a_actual 0
invocar_cierre ct43a_actual >/dev/null
[[ "$(estado_efectos ct43a_actual)" == '1|1|1|1' ]]
psql_admin --command \
    "SELECT public.comprobar_version_actual_ct43a('ct43a_actual')" \
    >/dev/null

paso 'versión N coincide; otra versión se rechaza sin efectos'
preparar_detalle ct43a_exacta 1
invocar_cierre ct43a_exacta >/dev/null
[[ "$(estado_efectos ct43a_exacta)" == '1|1|1|1' ]]
preparar_detalle ct43a_distinta 2
esperar_fallo 'versión distinta CT43A' 42501 \
    'cierre de prueba RRHH rechazado' \
    invocar_cierre ct43a_distinta
[[ "$(estado_efectos ct43a_distinta)" == '0|0|0|0' ]]

paso 'mutación y cruce VEC se rechazan sin oráculo ni efectos'
preparar_detalle ct43a_vec_mutado 0
psql_admin --command "
    SELECT public.ajustar_version_observada_ct43a(
        'ct43a_vec_mutado', 1, false
    )" >/dev/null
esperar_fallo 'contexto mutado CT43A' 42501 \
    'cierre de prueba RRHH rechazado' \
    invocar_cierre ct43a_vec_mutado
[[ "$(estado_efectos ct43a_vec_mutado)" == '0|0|0|0' ]]
preparar_detalle ct43a_vec_cruce_a 0
preparar_detalle ct43a_vec_cruce_b 0
psql_admin --command "
    UPDATE public.vectores_consulta_rrhh_v3 a
       SET capacidad = b.capacidad
      FROM public.vectores_consulta_rrhh_v3 b
     WHERE a.caso = 'ct43a_vec_cruce_a'
       AND b.caso = 'ct43a_vec_cruce_b'" >/dev/null
esperar_fallo 'capacidad cruzada CT43A' 22023 \
    'ligadura VEC-AD-3 inválida' \
    invocar_cierre ct43a_vec_cruce_a
[[ "$(estado_efectos ct43a_vec_cruce_a)" == '0|0|0|0' ]]

paso 'rollback, replay y concurrencia mantienen atomicidad'
preparar_detalle ct43a_rollback 0
psql_runtime --command "
    BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
    SET LOCAL TimeZone='UTC';
    SET LOCAL statement_timeout='15s';
    SET LOCAL idle_in_transaction_session_timeout='20s';
    SELECT vec_contratacion_temporal
           .prueba_cerrar_resultado_recibo_ct43('ct43a_rollback');
    ROLLBACK" >/dev/null
[[ "$(estado_efectos ct43a_rollback)" == '0|0|0|0' ]]
invocar_cierre ct43a_rollback >/dev/null
[[ "$(estado_efectos ct43a_rollback)" == '1|1|1|1' ]]
esperar_fallo 'replay CT43A' 42501 \
    'cierre de prueba RRHH rechazado' \
    invocar_cierre ct43a_rollback
[[ "$(estado_efectos ct43a_rollback)" == '1|1|1|1' ]]

preparar_detalle ct43a_concurrente 0
salida_a="$(mktemp "${TMPDIR:-/tmp}/vec-ct43a-a.XXXXXX")"
salida_b="$(mktemp "${TMPDIR:-/tmp}/vec-ct43a-b.XXXXXX")"
temporales+=("$salida_a" "$salida_b")
invocar_cierre ct43a_concurrente >"$salida_a" 2>&1 &
pid_a=$!
invocar_cierre ct43a_concurrente >"$salida_b" 2>&1 &
pid_b=$!
estado_a=0
estado_b=0
wait "$pid_a" || estado_a=$?
wait "$pid_b" || estado_b=$?
if (( (estado_a == 0) + (estado_b == 0) != 1 )); then
    sed -n '1,20p' "$salida_a" >&2
    sed -n '1,20p' "$salida_b" >&2
    exit 1
fi
[[ "$(estado_efectos ct43a_concurrente)" == '1|1|1|1' ]]

paso 'los cuatro estados transitorios conservan SQLSTATE y rollback total'
psql_admin --command \
    'CREATE TRIGGER forzar_sqlstate_ct43a BEFORE INSERT ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 FOR EACH ROW EXECUTE FUNCTION public.forzar_sqlstate_prueba_ct43()' \
    >/dev/null
for estado in 40001 40P01 55P03 57014; do
    caso="ct43a_estado_${estado,,}"
    preparar_detalle "$caso" 0
    esperar_fallo "propagación $estado CT43A" "$estado" \
        'estado transitorio sintético CT43' \
        invocar_cierre "$caso" \
        "SET LOCAL vec.prueba_ct43_sqlstate='$estado';"
    [[ "$(estado_efectos "$caso")" == '0|0|0|0' ]]
done
psql_admin --command \
    'DROP TRIGGER forzar_sqlstate_ct43a ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2' \
    >/dev/null

paso 'revocación viva impide cierre y no deja efectos'
preparar_detalle ct43a_revocada 0
psql_admin <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
INSERT INTO vec_autorizacion.control_sesion_v1(
    control_sesion_ref, revision, sesion_ref, estado, huella_sha256,
    sesion_revalidada_en, sesion_valida_hasta
)
SELECT control_sesion_ref, 2, sesion_ref, 'revocada',
       pg_catalog.repeat('b', 64), sesion_revalidada_en,
       sesion_valida_hasta
  FROM vec_autorizacion.control_sesion_v1
 WHERE control_sesion_ref =
       'cse_registro_v3_0000000000000000000000'
   AND revision = 1;
UPDATE vec_autorizacion.control_sesion_actual_v1
   SET revision = 2,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       ),
       acto_ref = 'acto:sesion:revocada-ct43a'
 WHERE sesion_ref = 'ses_registro_v3_0000000000000000000000';
COMMIT;
SQL
esperar_fallo 'sesión revocada CT43A' 42501 \
    'decisión VEC-AD-3 rechazada' \
    invocar_cierre ct43a_revocada
[[ "$(estado_efectos ct43a_revocada)" == '0|0|0|0' ]]

paso 'corrector ascendente CT-000043A superado'
