#!/usr/bin/env bash
set -Eeuo pipefail

directorio="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
    pwd -P
)"

# Reutiliza la línea base real CT-000039..CT-000042 y su contenedor PG18.4.
# shellcheck disable=SC1091
source "$directorio/probar_o4_05_canones_resultado_recibo_rrhh_pg18_4.sh"
: "${contenedor:?el runner CT42 debe exponer su contenedor PostgreSQL}"

paso() {
    printf '[O4-05:CT-000043:PG18.4] %s\n' "$1"
}

estado_ct43() {
    valor "SELECT pg_catalog.concat_ws('|',
        cobertura.version_esquema::text,
        consultas.version_esquema::text,
        (pg_catalog.to_regclass(
            'vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2'
        ) IS NOT NULL)::text,
        (SELECT pg_catalog.count(*)::text
           FROM pg_catalog.pg_proc funcion
          WHERE funcion.pronamespace =
                'vec_contratacion_temporal'::regnamespace
            AND funcion.proname =
                'cerrar_prueba_resultado_recibo_rrhh_v2'),
        (SELECT pg_catalog.count(*)::text
           FROM pg_catalog.pg_type tipo
          WHERE tipo.typnamespace =
                'vec_contratacion_temporal'::regnamespace
            AND tipo.typname = ANY(ARRAY[
                'contexto_cierre_prueba_rrhh_v2',
                'contenido_cierre_prueba_rrhh_v2',
                'evidencia_consumo_nuevo_rrhh_v3',
                'resultado_cierre_prueba_rrhh_v2',
                '_contexto_cierre_prueba_rrhh_v2',
                '_contenido_cierre_prueba_rrhh_v2',
                '_evidencia_consumo_nuevo_rrhh_v3',
                '_resultado_cierre_prueba_rrhh_v2',
                'prueba_resultado_recibo_rrhh_v2',
                '_prueba_resultado_recibo_rrhh_v2'
            ]::name[]))
      )
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4 cobertura
      CROSS JOIN
           vec_contratacion_temporal.control_migracion_consultas_rrhh consultas
     WHERE cobertura.control AND consultas.control"
}

comprobar_estado_ct43() {
    local esperado=$1
    local contexto=$2
    local obtenido
    obtenido="$(estado_ct43)"
    if [[ $obtenido != "$esperado" ]]; then
        printf 'estado CT43 alterado tras %s\n' "$contexto" >&2
        printf 'esperado=%s\nobtenido=%s\n' "$esperado" "$obtenido" >&2
        return 1
    fi
}

preparar_paquete_temporal() {
    docker exec "$contenedor" sh -eu -c '
        rm -rf /tmp/ct000043
        mkdir -p /tmp/ct000043
        cp /repo/contratacion_temporal/migraciones/000043_prueba_resultado_recibo_rrhh.up.sql /tmp/ct000043/
        cp -a /repo/contratacion_temporal/migraciones/000043_componentes /tmp/ct000043/
    '
}

ejecutar_paquete_temporal() {
    docker exec --workdir /tmp/ct000043 "$contenedor" \
        psql -X -v ON_ERROR_STOP=1 -v VERBOSITY=verbose \
        --username postgres --dbname postgres \
        --file 000043_prueba_resultado_recibo_rrhh.up.sql
}

esperar_fallo_inclusion() {
    local componente=$1
    local salida
    salida="$(mktemp "${TMPDIR:-/tmp}/vec-ct43-inclusion.XXXXXX")"
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

preparar_vector() {
    psql_admin --command \
        "SELECT public.preparar_vector_cierre_ct43('$1','$2')" \
        >/dev/null
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
           FROM vec_autorizacion_atestada_v3.consumo_decision_v3 consumo
          WHERE consumo.decision_ref =
                'decision:consulta-rrhh:$caso'),
        (SELECT pg_catalog.count(*)
           FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 auditoria
          WHERE auditoria.decision_ref =
                'decision:consulta-rrhh:$caso'),
        (SELECT pg_catalog.count(*)
           FROM vec_contratacion_temporal.registro_acceso_rrhh acceso
          WHERE acceso.decision_ref =
                'decision:consulta-rrhh:$caso'),
        (SELECT pg_catalog.count(*)
           FROM vec_contratacion_temporal
                .prueba_resultado_recibo_rrhh_v2 prueba
          WHERE prueba.decision_ref =
                'decision:consulta-rrhh:$caso')
    )"
}

probar_deriva() {
    local descripcion=$1
    local mensaje=$2
    local mutacion=$3
    local restauracion=$4
    psql_admin --command "$mutacion" >/dev/null
    esperar_fallo "$descripcion" 55000 "$mensaje" \
        archivo \
        contratacion_temporal/migraciones/000043_prueba_resultado_recibo_rrhh.down.sql
    comprobar_estado_ct43 '23|7|true|1|10' "$descripcion"
    psql_admin --command "$restauracion" >/dev/null
}

paso 'dependencia VEC-AD-3 causal 000003..000006'
for ruta in \
    autorizacion_atestada_v3/migraciones/000003_consumidor_consulta_cuadro_rrhh_v3.up.sql \
    autorizacion_atestada_v3/migraciones/000004_consumidor_consulta_detalle_rrhh_v3.up.sql \
    autorizacion_atestada_v3/migraciones/000005_revalidacion_final_consultas_rrhh_v3.up.sql \
    autorizacion_atestada_v3/migraciones/000006_prueba_consumo_consultas_rrhh_v3.up.sql
do
    archivo "$ruta"
done

paso 'componentes ausentes o mutados conservan barreras y objetos'
estado_base='22|6|false|0|0'
componentes=(
    010_tipos_cierre.sql
    020_relaciones_y_prueba.sql
    030_primitiva_cierre.sql
    085_guardia_columnas_padre.sql
    090_acl_catalogo_y_barrera.sql
    095_avance_barreras.sql
)
for componente in "${componentes[@]}"; do
    preparar_paquete_temporal
    docker exec "$contenedor" rm \
        "/tmp/ct000043/000043_componentes/$componente"
    esperar_fallo_inclusion "$componente"
    comprobar_estado_ct43 "$estado_base" "ausencia de $componente"
done
preparar_paquete_temporal
docker exec "$contenedor" sh -eu -c \
    'printf "\\nSELECT 1 / 0;\\n" >> /tmp/ct000043/000043_componentes/020_relaciones_y_prueba.sql'
esperar_fallo 'componente CT43 mutado' 22012 'division by zero' \
    ejecutar_paquete_temporal
comprobar_estado_ct43 "$estado_base" 'componente mutado'

paso 'instalación real, reentrada y ciclo limpio DOWN/UP'
(
    cd /tmp
    psql_admin --file \
        /repo/contratacion_temporal/migraciones/000043_prueba_resultado_recibo_rrhh.up.sql \
        >/dev/null
)
comprobar_estado_ct43 '23|7|true|1|10' 'primer UP'
esperar_fallo 'reentrada UP CT43' 55000 \
    'estado incompatible para prueba durable RRHH' \
    archivo \
    contratacion_temporal/migraciones/000043_prueba_resultado_recibo_rrhh.up.sql
comprobar_estado_ct43 '23|7|true|1|10' 'reentrada UP'
archivo \
    contratacion_temporal/migraciones/000043_prueba_resultado_recibo_rrhh.down.sql
comprobar_estado_ct43 "$estado_base" 'primer DOWN'
esperar_fallo 'reentrada DOWN CT43' 55000 \
    'estado incompatible para revertir prueba RRHH' \
    archivo \
    contratacion_temporal/migraciones/000043_prueba_resultado_recibo_rrhh.down.sql
archivo \
    contratacion_temporal/migraciones/000043_prueba_resultado_recibo_rrhh.up.sql
comprobar_estado_ct43 '23|7|true|1|10' 'segundo UP'

paso 'retirada segura detecta ACL, comentarios, catálogo y dependencias'
firma='vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v2(vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2,vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2,vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
probar_deriva 'ACL de función CT43' \
    'ACL de prueba durable RRHH incompatible' \
    "GRANT EXECUTE ON FUNCTION $firma TO vec_contratacion_temporal_consultor_rrhh" \
    "REVOKE EXECUTE ON FUNCTION $firma FROM vec_contratacion_temporal_consultor_rrhh"
probar_deriva 'ACL de columna preexistente CT43' \
    'catálogo de prueba durable RRHH incompatible' \
    "GRANT SELECT (expediente_ref_prueba_v2) ON vec_contratacion_temporal.registro_acceso_rrhh TO vec_contratacion_temporal_consultor_rrhh" \
    "REVOKE SELECT (expediente_ref_prueba_v2) ON vec_contratacion_temporal.registro_acceso_rrhh FROM vec_contratacion_temporal_consultor_rrhh"
probar_deriva 'almacenamiento de columna preexistente CT43' \
    'catálogo de prueba durable RRHH incompatible' \
    "ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh ALTER COLUMN expediente_ref_prueba_v2 SET STORAGE EXTERNAL" \
    "ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh ALTER COLUMN expediente_ref_prueba_v2 SET STORAGE EXTENDED"
probar_deriva 'estadística de columna preexistente CT43' \
    'catálogo de prueba durable RRHH incompatible' \
    "ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh ALTER COLUMN expediente_ref_prueba_v2 SET STATISTICS 37" \
    "ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh ALTER COLUMN expediente_ref_prueba_v2 SET STATISTICS -1"
probar_deriva 'índice futuro sobre columna del padre CT43' \
    'dependencias de columnas de prueba RRHH incompatibles' \
    "CREATE INDEX deriva_ct43_padre_idx ON vec_contratacion_temporal.registro_acceso_rrhh(expediente_ref_prueba_v2)" \
    "DROP INDEX vec_contratacion_temporal.deriva_ct43_padre_idx"
probar_deriva 'restricción futura sobre columna del padre CT43' \
    'dependencias de columnas de prueba RRHH incompatibles' \
    "ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh ADD CONSTRAINT deriva_ct43_padre_chk CHECK (pg_catalog.octet_length(expediente_ref_prueba_v2) >= 0)" \
    "ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh DROP CONSTRAINT deriva_ct43_padre_chk"
probar_deriva 'estadística futura sobre columnas del padre CT43' \
    'dependencias de columnas de prueba RRHH incompatibles' \
    "CREATE STATISTICS vec_contratacion_temporal.deriva_ct43_padre_stat ON expediente_ref_prueba_v2,version_expediente_prueba_v2 FROM vec_contratacion_temporal.registro_acceso_rrhh" \
    "DROP STATISTICS vec_contratacion_temporal.deriva_ct43_padre_stat"
probar_deriva 'publicación futura sobre columna del padre CT43' \
    'dependencias de columnas de prueba RRHH incompatibles' \
    "CREATE PUBLICATION deriva_ct43_padre_pub FOR TABLE vec_contratacion_temporal.registro_acceso_rrhh(expediente_ref_prueba_v2)" \
    "DROP PUBLICATION deriva_ct43_padre_pub"
probar_deriva 'publicación futura completa del padre CT43' \
    'dependencias de columnas de prueba RRHH incompatibles' \
    "CREATE PUBLICATION deriva_ct43_padre_todo_pub FOR TABLE vec_contratacion_temporal.registro_acceso_rrhh" \
    "DROP PUBLICATION deriva_ct43_padre_todo_pub"
probar_deriva 'tabla hija futura del padre CT43' \
    'dependencias de columnas de prueba RRHH incompatibles' \
    "CREATE TABLE public.deriva_ct43_hija() INHERITS (vec_contratacion_temporal.registro_acceso_rrhh)" \
    "DROP TABLE public.deriva_ct43_hija"
probar_deriva 'comentario de restricción CT43' \
    'catálogo de prueba durable RRHH incompatible' \
    "COMMENT ON CONSTRAINT prueba_resultado_recibo_rrhh_v2_pkey ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 IS 'deriva'" \
    "COMMENT ON CONSTRAINT prueba_resultado_recibo_rrhh_v2_pkey ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 IS NULL"
psql_admin <<'SQL' >/dev/null
CREATE FUNCTION public.disparador_deriva_ct43()
RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog
AS $f$ BEGIN RETURN NEW; END $f$;
SQL
probar_deriva 'trigger añadido CT43' \
    'catálogo de prueba durable RRHH incompatible' \
    "CREATE TRIGGER deriva_ct43 BEFORE INSERT ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 FOR EACH ROW EXECUTE FUNCTION public.disparador_deriva_ct43()" \
    "DROP TRIGGER deriva_ct43 ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2"
probar_deriva 'índice añadido CT43' \
    'catálogo de prueba durable RRHH incompatible' \
    "CREATE INDEX deriva_ct43_idx ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2(tipo_consulta)" \
    "DROP INDEX vec_contratacion_temporal.deriva_ct43_idx"
probar_deriva 'estadística añadida CT43' \
    'catálogo de prueba durable RRHH incompatible' \
    "CREATE STATISTICS vec_contratacion_temporal.deriva_ct43_stat ON tipo_consulta,total FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2" \
    "DROP STATISTICS vec_contratacion_temporal.deriva_ct43_stat"
probar_deriva 'dependencia futura CT43' \
    'catálogo de prueba durable RRHH incompatible' \
    "CREATE VIEW public.deriva_ct43_vista AS SELECT acceso_ref FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2" \
    "DROP VIEW public.deriva_ct43_vista"
psql_admin --command \
    'DROP FUNCTION public.disparador_deriva_ct43()' >/dev/null

paso 'datos sintéticos VEC, identidad y cierre causal cuadro/detalle'
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

esperar_fallo 'rol de ejecución lee prueba CT43' 42501 \
    'permission denied for table prueba_resultado_recibo_rrhh_v2' \
    psql_runtime --command \
    'SELECT count(*) FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2'
esperar_fallo 'rol de ejecución inserta prueba CT43' 42501 \
    'permission denied for table prueba_resultado_recibo_rrhh_v2' \
    psql_runtime --command \
    "INSERT INTO vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2(acceso_ref) VALUES ('acceso:prohibido')"

preparar_vector ct43_cuadro_a cuadro
esperar_fallo 'aislamiento inferior CT43' 42501 \
    'consulta RRHH VEC-AD-3 rechazada' psql_runtime --command "
    SET statement_timeout='15s';
    SET idle_in_transaction_session_timeout='20s';
    SELECT vec_contratacion_temporal
           .prueba_cerrar_resultado_recibo_ct43('ct43_cuadro_a')"
esperar_fallo 'solo lectura CT43' 42501 \
    'consulta RRHH VEC-AD-3 rechazada' psql_runtime --command "
    BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ ONLY;
    SET LOCAL statement_timeout='15s';
    SET LOCAL idle_in_transaction_session_timeout='20s';
    SELECT vec_contratacion_temporal
           .prueba_cerrar_resultado_recibo_ct43('ct43_cuadro_a')"
preparar_vector ct43_cuadro_a cuadro
invocar_cierre ct43_cuadro_a >/dev/null
[[ "$(estado_efectos ct43_cuadro_a)" == '1|1|1|1' ]]
esperar_fallo 'repetición CT43' 42501 \
    'cierre de prueba RRHH rechazado' \
    invocar_cierre ct43_cuadro_a
[[ "$(estado_efectos ct43_cuadro_a)" == '1|1|1|1' ]]

preparar_vector ct43_cuadro_b cuadro
invocar_cierre ct43_cuadro_b >/dev/null
preparar_vector ct43_detalle_a detalle
invocar_cierre ct43_detalle_a >/dev/null

paso 'reversión causal y preservación de SQLSTATE transitorios'
preparar_vector ct43_reversion cuadro
psql_runtime --command "
    BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
    SET LOCAL TimeZone='UTC';
    SET LOCAL statement_timeout='15s';
    SET LOCAL idle_in_transaction_session_timeout='20s';
    SELECT vec_contratacion_temporal
           .prueba_cerrar_resultado_recibo_ct43('ct43_reversion');
    ROLLBACK" >/dev/null
[[ "$(estado_efectos ct43_reversion)" == '0|0|0|0' ]]
invocar_cierre ct43_reversion >/dev/null
[[ "$(estado_efectos ct43_reversion)" == '1|1|1|1' ]]

psql_admin --command \
    'CREATE TRIGGER forzar_sqlstate_ct43 BEFORE INSERT ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 FOR EACH ROW EXECUTE FUNCTION public.forzar_sqlstate_prueba_ct43()' \
    >/dev/null
for estado in 40001 40P01 55P03 57014; do
    caso="ct43_estado_${estado,,}"
    preparar_vector "$caso" cuadro
    esperar_fallo "propagación $estado CT43" "$estado" \
        'estado transitorio sintético CT43' \
        invocar_cierre "$caso" \
        "SET LOCAL vec.prueba_ct43_sqlstate='$estado';"
    [[ "$(estado_efectos "$caso")" == '0|0|0|0' ]]
done
psql_admin --command \
    'DROP TRIGGER forzar_sqlstate_ct43 ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2' \
    >/dev/null

paso 'dos cierres concurrentes consumen y prueban una sola vez'
preparar_vector ct43_concurrente cuadro
salida_a="$(mktemp "${TMPDIR:-/tmp}/vec-ct43-a.XXXXXX")"
salida_b="$(mktemp "${TMPDIR:-/tmp}/vec-ct43-b.XXXXXX")"
temporales+=("$salida_a" "$salida_b")
invocar_cierre ct43_concurrente >"$salida_a" 2>&1 &
pid_a=$!
invocar_cierre ct43_concurrente >"$salida_b" 2>&1 &
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
[[ "$(estado_efectos ct43_concurrente)" == '1|1|1|1' ]]

paso 'material durable, FKs cruzadas, inmutabilidad y límites'
archivo \
    contratacion_temporal/pruebas_sql/o405_prueba_resultado_recibo_rrhh.sql

paso 'revocación viva falla cerrada sin consumo ni prueba'
preparar_vector ct43_revocada cuadro
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
       acto_ref = 'acto:sesion:revocada-ct43'
 WHERE sesion_ref = 'ses_registro_v3_0000000000000000000000';
COMMIT;
SQL
esperar_fallo 'sesión revocada CT43' 42501 \
    'decisión VEC-AD-3 rechazada' \
    invocar_cierre ct43_revocada
[[ "$(estado_efectos ct43_revocada)" == '0|0|0|0' ]]

paso 'evidencia durable bloquea la retirada segura sin mutar estado'
estado_con_pruebas="$(estado_ct43)"
esperar_fallo 'DOWN con evidencia CT43' 55000 \
    'existe prueba durable RRHH; reversión prohibida' \
    archivo \
    contratacion_temporal/migraciones/000043_prueba_resultado_recibo_rrhh.down.sql
comprobar_estado_ct43 "$estado_con_pruebas" 'DOWN con evidencia'

paso 'prueba durable y Recibo V2 CT-000043 superados'
