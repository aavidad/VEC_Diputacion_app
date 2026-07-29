#!/usr/bin/env bash
set -Eeuo pipefail

directorio="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
    pwd -P
)"

# Las puertas acotadas se repiten como procesos aislados antes de componer el
# paquete real. Cada una administra su propio PostgreSQL y su propia limpieza.
"$directorio/probar_o4_05_control_causal_cursores_ct44a_pg18_4.sh"
"$directorio/probar_o4_05_motor_atomico_consultas_ct44b_pg18_4.sh"

# Línea base real CT-000039..CT-000042; este proceso conserva su contenedor.
# shellcheck disable=SC1091
source "$directorio/probar_o4_05_canones_resultado_recibo_rrhh_pg18_4.sh"
: "${contenedor:?el ejecutor CT42 debe exponer PostgreSQL}"

paso() {
    printf '[O4-05:CT-000044C:PG18.4] %s\n' "$1"
}

estado_ct44() {
    valor "SELECT pg_catalog.concat_ws('|',
        cobertura.version_esquema::text,
        consultas.version_esquema::text,
        (pg_catalog.to_regclass(
            'vec_contratacion_temporal.'
            'control_causal_familia_cursor_rrhh'
        ) IS NOT NULL)::text,
        (SELECT pg_catalog.count(*)::text
           FROM pg_catalog.pg_proc funcion
          WHERE funcion.pronamespace =
                'vec_contratacion_temporal'::regnamespace
            AND funcion.proname = ANY(ARRAY[
                'acreditar_contexto_motor_consultas_rrhh_v1',
                'consumir_autorizacion_motor_consultas_rrhh_v1',
                'materializar_detalle_rrhh_v1',
                'materializar_cuadro_rrhh_v1',
                'validar_avance_control_causal_cursor_rrhh_v1',
                'avanzar_control_causal_revocacion_cursor_rrhh_v1',
                'resolver_estado_cursor_cuadro_rrhh_v1',
                'preparar_salida_cursor_cuadro_rrhh_v1',
                'aplicar_efectos_cursor_cuadro_rrhh_v1',
                'motor_consultar_cuadro_rrhh_v1',
                'motor_consultar_detalle_rrhh_v1'
            ]::name[])),
        (SELECT pg_catalog.count(*)::text
           FROM pg_catalog.pg_type tipo
          WHERE tipo.typnamespace =
                'vec_contratacion_temporal'::regnamespace
            AND tipo.typname = ANY(ARRAY[
                'material_autorizacion_consulta_rrhh_v3',
                'estado_cursor_entrada_cuadro_rrhh_v1',
                'materializacion_cuadro_rrhh_v1',
                'materializacion_detalle_rrhh_v1',
                'salida_cursor_cuadro_rrhh_v1',
                'resultado_motor_cuadro_rrhh_v1',
                'resultado_motor_detalle_rrhh_v1',
                '_material_autorizacion_consulta_rrhh_v3',
                '_estado_cursor_entrada_cuadro_rrhh_v1',
                '_materializacion_cuadro_rrhh_v1',
                '_materializacion_detalle_rrhh_v1',
                '_salida_cursor_cuadro_rrhh_v1',
                '_resultado_motor_cuadro_rrhh_v1',
                '_resultado_motor_detalle_rrhh_v1',
                'control_causal_familia_cursor_rrhh',
                '_control_causal_familia_cursor_rrhh'
            ]::name[]))
    )
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4 cobertura
      CROSS JOIN
           vec_contratacion_temporal.control_migracion_consultas_rrhh consultas
     WHERE cobertura.control AND consultas.control"
}

comprobar_estado_ct44() {
    local esperado=$1
    local contexto=$2
    local obtenido
    obtenido="$(estado_ct44)"
    if [[ $obtenido != "$esperado" ]]; then
        printf 'estado CT44 alterado tras %s\n' "$contexto" >&2
        printf 'esperado=%s\nobtenido=%s\n' "$esperado" "$obtenido" >&2
        return 1
    fi
}

preparar_paquete_temporal() {
    docker exec "$contenedor" sh -eu -c '
        rm -rf /tmp/ct000044
        mkdir -p /tmp/ct000044
        cp /repo/contratacion_temporal/migraciones/000044_motor_consultas_rrhh.up.sql /tmp/ct000044/
        cp -a /repo/contratacion_temporal/migraciones/000044_componentes /tmp/ct000044/
    '
}

ejecutar_paquete_temporal() {
    docker exec --workdir /tmp/ct000044 "$contenedor" \
        psql -X -v ON_ERROR_STOP=1 -v VERBOSITY=verbose \
        --username postgres --dbname postgres \
        --file 000044_motor_consultas_rrhh.up.sql
}

esperar_fallo_inclusion() {
    local componente=$1
    local salida
    salida="$(mktemp "${TMPDIR:-/tmp}/vec-ct44c-inclusion.XXXXXX")"
    temporales+=("$salida")
    if ejecutar_paquete_temporal >"$salida" 2>&1; then
        printf 'se esperaba fallo por componente ausente: %s\n' \
            "$componente" >&2
        return 1
    fi
    if ! rg -Fq "$componente" "$salida"; then
        printf 'fallo inesperado al omitir %s\n' "$componente" >&2
        tail -20 "$salida" >&2
        return 1
    fi
}

probar_deriva() {
    local descripcion=$1
    local mensaje=$2
    local mutacion=$3
    local restauracion=$4
    psql_admin --command "$mutacion" >/dev/null
    esperar_fallo "$descripcion" 55000 "$mensaje" \
        archivo \
        contratacion_temporal/migraciones/000044_motor_consultas_rrhh.down.sql
    comprobar_estado_ct44 '24|8|true|11|16' "$descripcion"
    psql_admin --command "$restauracion" >/dev/null
}

paso 'dependencias VEC-AD-3, prueba CT43 y corrector CT43A'
for ruta in \
    autorizacion_atestada_v3/migraciones/000003_consumidor_consulta_cuadro_rrhh_v3.up.sql \
    autorizacion_atestada_v3/migraciones/000004_consumidor_consulta_detalle_rrhh_v3.up.sql \
    autorizacion_atestada_v3/migraciones/000005_revalidacion_final_consultas_rrhh_v3.up.sql \
    autorizacion_atestada_v3/migraciones/000006_prueba_consumo_consultas_rrhh_v3.up.sql
do
    archivo "$ruta"
done
archivo \
    contratacion_temporal/migraciones/000043_prueba_resultado_recibo_rrhh.up.sql
archivo \
    contratacion_temporal/migraciones/000043a_detalle_version_actual.up.sql

paso 'cada componente ausente o mutado revierte el paquete completo'
estado_base='23|7|false|0|0'
componentes=(
    010_tipos_resultado.sql
    020_guardas_y_contexto.sql
    030_materializacion_detalle.sql
    040_materializacion_cuadro.sql
    050_control_causal_y_cursores.sql
    055_efectos_cursor.sql
    060_motor_atomico_y_efectos.sql
    090_acl_catalogo_y_barrera.sql
    095_avance_barreras.sql
)
for componente in "${componentes[@]}"; do
    preparar_paquete_temporal
    docker exec "$contenedor" rm \
        "/tmp/ct000044/000044_componentes/$componente"
    esperar_fallo_inclusion "$componente"
    comprobar_estado_ct44 "$estado_base" "ausencia de $componente"
    preparar_paquete_temporal
    docker exec \
        --env VEC_COMPONENTE="$componente" \
        "$contenedor" sh -eu -c \
        'printf "\\nSELECT 1 / 0;\\n" >> "/tmp/ct000044/000044_componentes/$VEC_COMPONENTE"'
    esperar_fallo "mutación de $componente CT44" 22012 \
        'division by zero' ejecutar_paquete_temporal
    comprobar_estado_ct44 "$estado_base" "mutación de $componente"
done

paso 'tres altas frescas y dos retiradas conservan la misma huella'
archivo \
    contratacion_temporal/migraciones/000044_motor_consultas_rrhh.up.sql
comprobar_estado_ct44 '24|8|true|11|16' 'primer UP'
esperar_fallo 'reentrada UP CT44' 55000 \
    'estado incompatible para motor de consultas RRHH' \
    archivo \
    contratacion_temporal/migraciones/000044_motor_consultas_rrhh.up.sql
archivo \
    contratacion_temporal/migraciones/000044_motor_consultas_rrhh.down.sql
comprobar_estado_ct44 "$estado_base" 'primer DOWN'
esperar_fallo 'reentrada DOWN CT44' 55000 \
    'estado incompatible para revertir motor RRHH' \
    archivo \
    contratacion_temporal/migraciones/000044_motor_consultas_rrhh.down.sql
archivo \
    contratacion_temporal/migraciones/000044_motor_consultas_rrhh.up.sql
comprobar_estado_ct44 '24|8|true|11|16' 'segundo UP'
archivo \
    contratacion_temporal/migraciones/000044_motor_consultas_rrhh.down.sql
archivo \
    contratacion_temporal/migraciones/000044_motor_consultas_rrhh.up.sql
comprobar_estado_ct44 '24|8|true|11|16' 'tercer UP'

paso 'una barrera futura bloquea DOWN sin alterar el paquete'
psql_admin <<'SQL' >/dev/null
BEGIN;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 9
 WHERE control AND version_esquema = 8;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 25
 WHERE control AND version_esquema = 24;
COMMIT;
SQL
esperar_fallo 'barrera futura CT45' 55000 \
    'estado incompatible para revertir motor RRHH' \
    archivo \
    contratacion_temporal/migraciones/000044_motor_consultas_rrhh.down.sql
comprobar_estado_ct44 '25|9|true|11|16' 'barrera futura'
psql_admin <<'SQL' >/dev/null
BEGIN;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 8
 WHERE control AND version_esquema = 9;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 24
 WHERE control AND version_esquema = 25;
COMMIT;
SQL
comprobar_estado_ct44 '24|8|true|11|16' 'restauración de barrera'

paso 'ACL privada y retirada segura frente a derivas semánticas'
firma='vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3)'
esperar_fallo 'el rol de ejecución intenta usar el motor privado CT44' 42501 \
    'permission denied' \
    psql_runtime --command "
    SELECT vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v1(
        NULL::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
        NULL::vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
        NULL::vec_contratacion_temporal
            .material_autorizacion_consulta_rrhh_v3
    )"
probar_deriva 'ACL de función CT44' \
    'ACL del motor de consultas RRHH incompatible' \
    "GRANT EXECUTE ON FUNCTION $firma TO vec_contratacion_temporal_consultor_rrhh" \
    "REVOKE EXECUTE ON FUNCTION $firma FROM vec_contratacion_temporal_consultor_rrhh"
probar_deriva 'comentario de tabla CT44' \
    'catálogo del motor de consultas RRHH incompatible' \
    "COMMENT ON TABLE vec_contratacion_temporal.control_causal_familia_cursor_rrhh IS 'deriva'" \
    "COMMENT ON TABLE vec_contratacion_temporal.control_causal_familia_cursor_rrhh IS 'Punto de control causal privado y mutable de una única familia de cursores RRHH.'"
probar_deriva 'comentario de columna CT44' \
    'catálogo del motor de consultas RRHH incompatible' \
    "COMMENT ON COLUMN vec_contratacion_temporal.control_causal_familia_cursor_rrhh.revision IS 'deriva'" \
    "COMMENT ON COLUMN vec_contratacion_temporal.control_causal_familia_cursor_rrhh.revision IS NULL"
probar_deriva 'índice añadido CT44' \
    'catálogo del motor de consultas RRHH incompatible' \
    "CREATE INDEX deriva_ct44_idx ON vec_contratacion_temporal.control_causal_familia_cursor_rrhh(revision)" \
    "DROP INDEX vec_contratacion_temporal.deriva_ct44_idx"
probar_deriva 'política añadida CT44' \
    'catálogo del motor de consultas RRHH incompatible' \
    "CREATE POLICY deriva_ct44 ON vec_contratacion_temporal.control_causal_familia_cursor_rrhh AS RESTRICTIVE TO vec_contratacion_temporal_propietario USING (true)" \
    "DROP POLICY deriva_ct44 ON vec_contratacion_temporal.control_causal_familia_cursor_rrhh"
probar_deriva 'publicación añadida CT44' \
    'catálogo del motor de consultas RRHH incompatible' \
    "CREATE PUBLICATION deriva_ct44_pub FOR TABLE vec_contratacion_temporal.control_causal_familia_cursor_rrhh" \
    "DROP PUBLICATION deriva_ct44_pub"

paso 'una dependencia futura hace fallar DROP RESTRICT y revierte todo'
psql_admin <<'SQL' >/dev/null
CREATE VIEW public.deriva_ct44_vista AS
SELECT vec_contratacion_temporal.materializar_detalle_rrhh_v1(
    NULL::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    NULL::vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    NULL::numeric
) AS material;
SQL
esperar_fallo 'dependencia futura CT44' 2BP01 \
    'cannot drop function' \
    archivo \
    contratacion_temporal/migraciones/000044_motor_consultas_rrhh.down.sql
comprobar_estado_ct44 '24|8|true|11|16' 'dependencia futura'
psql_admin --command 'DROP VIEW public.deriva_ct44_vista' >/dev/null

paso 'estado causal creado por el motor bloquea la retirada'
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
archivo \
    contratacion_temporal/pruebas_sql/o405_motor_atomico_consultas_ct44b_datos_sinteticos.sql
psql_admin --command 'SELECT public.ampliar_corpus_cuadro_ct44b()' \
    >/dev/null
psql_admin --command \
    "SELECT public.preparar_vector_cierre_ct43('ct44c_estado_causal','cuadro')" \
    >/dev/null
psql_runtime --command "
    BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
    SET LOCAL TimeZone='UTC';
    SELECT vec_contratacion_temporal
           .prueba_invocar_motor_cuadro_ct44b(
               'ct44c_estado_causal', ''
           );
    COMMIT" >/dev/null
[[ "$(valor 'SELECT pg_catalog.count(*) FROM vec_contratacion_temporal.control_causal_familia_cursor_rrhh')" -ge 1 ]]
esperar_fallo 'DOWN con estado causal CT44' 55000 \
    'existe estado causal RRHH; reversión prohibida' \
    archivo \
    contratacion_temporal/migraciones/000044_motor_consultas_rrhh.down.sql
comprobar_estado_ct44 '24|8|true|11|16' 'DOWN con estado causal'

paso 'carreras integrales de revocación sobre motor completo'
"$directorio/probar_o4_05_motor_consultas_rrhh_carreras_ct44c_pg18_4.sh"

paso 'retirada de privilegios auxiliares de prueba'
psql_admin <<'SQL' >/dev/null
REVOKE ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA vec_contratacion_temporal
    FROM vec_contratacion_temporal_consultor_rrhh;
REVOKE USAGE ON SCHEMA vec_contratacion_temporal
    FROM vec_contratacion_temporal_consultor_rrhh;
SQL
[[ "$(valor "SELECT pg_catalog.count(*)::text || '|' ||
    pg_catalog.has_schema_privilege(
        'vec_contratacion_temporal_consultor_rrhh',
        'vec_contratacion_temporal', 'USAGE'
    )::text
  FROM pg_catalog.pg_proc
 WHERE pronamespace = 'vec_contratacion_temporal'::regnamespace
   AND pg_catalog.has_function_privilege(
       'vec_contratacion_temporal_consultor_rrhh', oid, 'EXECUTE'
   )")" == '0|false' ]]

paso 'paquete integral del motor CT-000044 superado'
