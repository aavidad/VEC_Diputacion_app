#!/usr/bin/env bash
set -Eeuo pipefail

directorio="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
    pwd -P
)"

# Instala y acredita la línea base exacta CT-000041 sobre PostgreSQL 18.4.
# shellcheck disable=SC1091
source "$directorio/probar_o4_05_vocabulario_estados_publicacion_rrhh_pg18_4.sh"

paso() {
    printf '[O4-05:CT-000042:PG18.4] %s\n' "$1"
}

estado_ct42() {
    valor "SELECT pg_catalog.concat_ws('|',
        cobertura.version_esquema::text,
        consultas.version_esquema::text,
        (SELECT pg_catalog.count(*)::text
           FROM pg_catalog.pg_proc funcion
          WHERE funcion.pronamespace =
                'vec_contratacion_temporal'::regnamespace
            AND funcion.proname = ANY(ARRAY[
                'codificar_texto_utf8_rrhh_v1',
                'decodificar_texto_utf8_rrhh_v1',
                'texto_instante_canonico_rrhh_v1',
                'encuadrar_valor_rrhh_v1',
                'canon_resumen_publicacion_rrhh_v1',
                'canon_resultado_consulta_rrhh_puro_v1',
                'canon_recibo_lectura_rrhh_v2',
                'huella_material_consumo_rrhh_v3',
                'canon_contenido_cuadro_rrhh_v1',
                'canon_contenido_detalle_rrhh_v1'
            ]::name[])),
        (SELECT pg_catalog.count(*)::text
           FROM pg_catalog.pg_type tipo
          WHERE tipo.typnamespace =
                'vec_contratacion_temporal'::regnamespace
            AND tipo.typname = ANY(ARRAY[
                'resumen_publicacion_rrhh_v1',
                'solicitud_operativa_rrhh_v1',
                'analisis_operativo_rrhh_v1',
                'comprobacion_operativa_rrhh_v1',
                'cobertura_operativa_rrhh_v1',
                'asignacion_operativa_rrhh_v1',
                'hito_expediente_rrhh_v1',
                'entrada_detalle_expediente_rrhh_v1',
                'evidencia_recibo_lectura_rrhh_v2',
                '_resumen_publicacion_rrhh_v1',
                '_solicitud_operativa_rrhh_v1',
                '_analisis_operativo_rrhh_v1',
                '_comprobacion_operativa_rrhh_v1',
                '_cobertura_operativa_rrhh_v1',
                '_asignacion_operativa_rrhh_v1',
                '_hito_expediente_rrhh_v1',
                '_entrada_detalle_expediente_rrhh_v1',
                '_evidencia_recibo_lectura_rrhh_v2'
            ]::name[]))
    )
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4 cobertura
      CROSS JOIN
           vec_contratacion_temporal.control_migracion_consultas_rrhh consultas
     WHERE cobertura.control AND consultas.control"
}

comprobar_estado() {
    local esperado=$1
    local contexto=$2
    local obtenido
    obtenido="$(estado_ct42)"
    if [[ $obtenido != "$esperado" ]]; then
        printf 'estado CT-000042 alterado tras %s\n' "$contexto" >&2
        printf 'esperado=%s\nobtenido=%s\n' "$esperado" "$obtenido" >&2
        return 1
    fi
}

ejecutar_paquete_temporal() {
    # shellcheck disable=SC2154
    docker exec --workdir /tmp/ct000042 "$contenedor" \
        psql -X -v ON_ERROR_STOP=1 -v VERBOSITY=verbose \
        --username postgres --dbname postgres \
        --file 000042_canones_resultado_recibo_rrhh.up.sql
}

preparar_paquete_temporal() {
    docker exec "$contenedor" sh -eu -c '
        rm -rf /tmp/ct000042
        mkdir -p /tmp/ct000042
        cp /repo/contratacion_temporal/migraciones/000042_canones_resultado_recibo_rrhh.up.sql /tmp/ct000042/
        cp -a /repo/contratacion_temporal/migraciones/000042_componentes /tmp/ct000042/
    '
}

esperar_fallo_inclusion() {
    local componente=$1
    local salida
    salida="$(mktemp "${TMPDIR:-/tmp}/vec-ct42-inclusion.XXXXXX")"
    temporales+=("$salida")
    if ejecutar_paquete_temporal >"$salida" 2>&1; then
        printf 'se esperaba fallo por componente ausente: %s\n' \
            "$componente" >&2
        return 1
    fi
    if ! rg -Fq "$componente" "$salida"; then
        printf 'fallo inesperado al omitir %s\n' "$componente" >&2
        sed -n '1,20p' "$salida" >&2
        return 1
    fi
}

paso 'cada componente ausente revierte toda la instalación'
estado_base='21|5|0|0'
componentes=(
    010_tipos_nominales.sql
    015_codec_utf8_inmutable.sql
    016_instante_canonico.sql
    020_auxiliares_y_canones_base.sql
    030_canon_detalle.sql
    090_acl_catalogo_y_barrera.sql
)
for componente in "${componentes[@]}"; do
    preparar_paquete_temporal
    docker exec "$contenedor" rm \
        "/tmp/ct000042/000042_componentes/$componente"
    esperar_fallo_inclusion "$componente"
    comprobar_estado "$estado_base" "ausencia de $componente"
done

paso 'un componente alterado también mantiene barreras y catálogo'
preparar_paquete_temporal
docker exec "$contenedor" sh -eu -c \
    'printf "\\nSELECT 1 / 0;\\n" >> /tmp/ct000042/000042_componentes/020_auxiliares_y_canones_base.sql'
esperar_fallo 'componente alterado' 22012 'division by zero' \
    ejecutar_paquete_temporal
comprobar_estado "$estado_base" 'componente alterado'

paso 'UP real desde CWD distinto y vectores cruzados'
(
    cd /tmp
    psql_admin -X -v ON_ERROR_STOP=1 \
        --file /repo/contratacion_temporal/migraciones/000042_canones_resultado_recibo_rrhh.up.sql \
        >/dev/null
)
comprobar_estado '22|6|10|18' 'instalación'
archivo contratacion_temporal/pruebas_sql/o405_canones_resultado_recibo_rrhh.sql
archivo contratacion_temporal/pruebas_sql/o405_material_consumo_rrhh_v3.sql
archivo contratacion_temporal/pruebas_sql/o405_recibo_lectura_rrhh_v2.sql
archivo contratacion_temporal/pruebas_sql/o405_canones_rrhh_adversarial.sql

paso 'PUBLIC y roles runtime carecen de EXECUTE y USAGE'
estado_acl="$(valor "WITH
roles_runtime AS (
    SELECT rol.oid
      FROM pg_catalog.pg_roles rol
     WHERE rol.rolname = ANY(ARRAY[
        'vec_contratacion_temporal_migrador',
        'vec_contratacion_temporal_ejecutor',
        'vec_contratacion_temporal_confirmador_cobertura',
        'vec_contratacion_temporal_gobernador',
        'vec_contratacion_temporal_consultor_rrhh',
        'vec_contratacion_temporal_lector_resultado_cobertura'
     ])
),
funciones AS (
    SELECT funcion.oid, funcion.proacl, funcion.proowner
      FROM pg_catalog.pg_proc funcion
     WHERE funcion.pronamespace =
           'vec_contratacion_temporal'::regnamespace
       AND funcion.proname = ANY(ARRAY[
        'codificar_texto_utf8_rrhh_v1',
        'decodificar_texto_utf8_rrhh_v1',
        'texto_instante_canonico_rrhh_v1',
        'encuadrar_valor_rrhh_v1',
        'canon_resumen_publicacion_rrhh_v1',
        'canon_resultado_consulta_rrhh_puro_v1',
        'canon_recibo_lectura_rrhh_v2',
        'huella_material_consumo_rrhh_v3',
        'canon_contenido_cuadro_rrhh_v1',
        'canon_contenido_detalle_rrhh_v1'
       ]::name[])
),
tipos_base AS (
    SELECT tipo.oid, tipo.typacl, tipo.typowner
      FROM pg_catalog.pg_type tipo
     WHERE tipo.typnamespace =
           'vec_contratacion_temporal'::regnamespace
       AND tipo.typname = ANY(ARRAY[
        'resumen_publicacion_rrhh_v1',
        'solicitud_operativa_rrhh_v1',
        'analisis_operativo_rrhh_v1',
        'comprobacion_operativa_rrhh_v1',
        'cobertura_operativa_rrhh_v1',
        'asignacion_operativa_rrhh_v1',
        'hito_expediente_rrhh_v1',
        'entrada_detalle_expediente_rrhh_v1',
        'evidencia_recibo_lectura_rrhh_v2'
       ]::name[])
),
funciones_con_privilegio AS (
    SELECT funcion.oid, rol.oid AS rol_oid
      FROM funciones funcion CROSS JOIN roles_runtime rol
     WHERE pg_catalog.has_function_privilege(
        rol.oid, funcion.oid, 'EXECUTE'
     )
    UNION ALL
    SELECT funcion.oid, 0::oid
      FROM funciones funcion
      CROSS JOIN LATERAL pg_catalog.aclexplode(
        COALESCE(
            funcion.proacl,
            pg_catalog.acldefault('f', funcion.proowner)
        )
      ) privilegio
     WHERE privilegio.grantee = 0
       AND privilegio.privilege_type = 'EXECUTE'
),
tipos_con_privilegio AS (
    SELECT tipo.oid, rol.oid AS rol_oid
      FROM tipos_base tipo CROSS JOIN roles_runtime rol
     WHERE pg_catalog.has_type_privilege(
        rol.oid, tipo.oid, 'USAGE'
     )
    UNION ALL
    SELECT tipo.oid, 0::oid
      FROM tipos_base tipo
      CROSS JOIN LATERAL pg_catalog.aclexplode(
        COALESCE(
            tipo.typacl,
            pg_catalog.acldefault('T', tipo.typowner)
        )
      ) privilegio
     WHERE privilegio.grantee = 0
       AND privilegio.privilege_type = 'USAGE'
),
matrices AS (
    SELECT tipo.oid, tipo.typacl
      FROM pg_catalog.pg_type tipo
     WHERE tipo.typelem IN (SELECT oid FROM tipos_base)
)
SELECT pg_catalog.concat_ws('|',
    (SELECT pg_catalog.count(*) FROM roles_runtime),
    (SELECT pg_catalog.count(*) FROM funciones),
    (SELECT pg_catalog.count(*) FROM funciones_con_privilegio),
    (SELECT pg_catalog.count(*) FROM tipos_base),
    (SELECT pg_catalog.count(*) FROM tipos_con_privilegio),
    (SELECT pg_catalog.count(*) FROM matrices),
    (SELECT pg_catalog.count(*) FROM matrices WHERE typacl IS NOT NULL)
)")"
if [[ $estado_acl != '6|10|0|9|0|9|0' ]]; then
    printf 'ACL privadas CT-000042 divergentes: %s\n' "$estado_acl" >&2
    exit 1
fi

paso 'reentrada rechazada sin efectos'
esperar_fallo 'reentrada CT-000042' 55000 \
    'estado incompatible para instalar cánones RRHH' \
    archivo \
    contratacion_temporal/migraciones/000042_canones_resultado_recibo_rrhh.up.sql
comprobar_estado '22|6|10|18' 'reentrada'

paso 'barrera futura y derivas semánticas bloquean la retirada'
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 23
 WHERE control AND version_esquema = 22;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 7
 WHERE control AND version_esquema = 6;
SQL
esperar_fallo 'down con barrera futura' 55000 \
    'estado incompatible para revertir cánones RRHH' \
    archivo \
    contratacion_temporal/migraciones/000042_canones_resultado_recibo_rrhh.down.sql
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 22
 WHERE control AND version_esquema = 23;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 6
 WHERE control AND version_esquema = 7;
CREATE FUNCTION
vec_contratacion_temporal.encuadrar_valor_rrhh_v1(integer)
RETURNS bytea
LANGUAGE sql
IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog
AS 'SELECT pg_catalog.int4send($1)';
SQL
esperar_fallo 'down con sobrecarga futura' 55000 \
    'catálogo canónico RRHH incompatible' \
    archivo \
    contratacion_temporal/migraciones/000042_canones_resultado_recibo_rrhh.down.sql
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DROP FUNCTION
vec_contratacion_temporal.encuadrar_valor_rrhh_v1(integer);
SQL

definicion_encuadrar="$(valor "SELECT pg_catalog.pg_get_functiondef(
    'vec_contratacion_temporal.encuadrar_valor_rrhh_v1(bytea)'::regprocedure
)")"
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
CREATE OR REPLACE FUNCTION
vec_contratacion_temporal.encuadrar_valor_rrhh_v1(p_valor bytea)
RETURNS bytea
LANGUAGE sql
IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog
AS 'SELECT p_valor';
SQL
esperar_fallo 'down con cuerpo alterado' 55000 \
    'catálogo canónico RRHH incompatible' \
    archivo \
    contratacion_temporal/migraciones/000042_canones_resultado_recibo_rrhh.down.sql
printf '%s\n' "$definicion_encuadrar" | psql_admin >/dev/null

psql_admin <<'SQL' >/dev/null
ALTER FUNCTION
vec_contratacion_temporal.encuadrar_valor_rrhh_v1(bytea)
OWNER TO vec_contratacion_temporal_migrador;
SQL
esperar_fallo 'down con propietario alterado' 55000 \
    'catálogo canónico RRHH incompatible' \
    archivo \
    contratacion_temporal/migraciones/000042_canones_resultado_recibo_rrhh.down.sql
psql_admin <<'SQL' >/dev/null
ALTER FUNCTION
vec_contratacion_temporal.encuadrar_valor_rrhh_v1(bytea)
OWNER TO vec_contratacion_temporal_propietario;
SET ROLE vec_contratacion_temporal_propietario;
ALTER FUNCTION
vec_contratacion_temporal.encuadrar_valor_rrhh_v1(bytea)
SET search_path = public;
SQL
esperar_fallo 'down con proconfig alterada' 55000 \
    'catálogo canónico RRHH incompatible' \
    archivo \
    contratacion_temporal/migraciones/000042_canones_resultado_recibo_rrhh.down.sql
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
ALTER FUNCTION
vec_contratacion_temporal.encuadrar_valor_rrhh_v1(bytea)
SET search_path = pg_catalog;
SQL
comprobar_estado '22|6|10|18' 'derivas semánticas rechazadas'

paso 'ACL y dependencia futuras bloquean la retirada'
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION
vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(text)
TO vec_contratacion_temporal_ejecutor;
SQL
esperar_fallo 'down con ACL derivada' 55000 \
    'catálogo canónico RRHH incompatible' \
    archivo \
    contratacion_temporal/migraciones/000042_canones_resultado_recibo_rrhh.down.sql
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
REVOKE EXECUTE ON FUNCTION
vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(text)
FROM vec_contratacion_temporal_ejecutor;
CREATE VIEW vec_contratacion_temporal.dependencia_ct000042 AS
SELECT vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
    'dependencia'
) AS contenido;
SQL
esperar_fallo 'down con dependencia futura' 55000 \
    'catálogo canónico RRHH incompatible' \
    archivo \
    contratacion_temporal/migraciones/000042_canones_resultado_recibo_rrhh.down.sql
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DROP VIEW vec_contratacion_temporal.dependencia_ct000042;
SQL
comprobar_estado '22|6|10|18' 'derivas rechazadas'

paso 'un homónimo exterior no altera el catálogo objetivo'
psql_admin <<'SQL' >/dev/null
CREATE FUNCTION public.encuadrar_valor_rrhh_v1(p_valor bytea)
RETURNS bytea
LANGUAGE sql
IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog
AS 'SELECT p_valor';
SQL
archivo \
    contratacion_temporal/migraciones/000042_canones_resultado_recibo_rrhh.down.sql
comprobar_estado "$estado_base" 'down con homónimo exterior'
archivo \
    contratacion_temporal/migraciones/000042_canones_resultado_recibo_rrhh.up.sql
psql_admin --command \
    'DROP FUNCTION public.encuadrar_valor_rrhh_v1(bytea)' \
    >/dev/null
comprobar_estado '22|6|10|18' 'up tras homónimo exterior'

paso 'una segunda base con OID coincidentes no contamina la huella'
psql_admin --command \
    'CREATE DATABASE ct42_regresion_multibase TEMPLATE postgres' \
    >/dev/null
if [[ $(valor "WITH objetos AS (
    SELECT 'pg_proc'::regclass AS clase, funcion.oid AS objeto
      FROM pg_catalog.pg_proc funcion
     WHERE funcion.pronamespace =
           'vec_contratacion_temporal'::regnamespace
       AND funcion.proname = ANY(ARRAY[
        'codificar_texto_utf8_rrhh_v1',
        'decodificar_texto_utf8_rrhh_v1',
        'texto_instante_canonico_rrhh_v1',
        'encuadrar_valor_rrhh_v1',
        'canon_resumen_publicacion_rrhh_v1',
        'canon_resultado_consulta_rrhh_puro_v1',
        'canon_recibo_lectura_rrhh_v2',
        'huella_material_consumo_rrhh_v3',
        'canon_contenido_cuadro_rrhh_v1',
        'canon_contenido_detalle_rrhh_v1'
       ]::name[])
    UNION ALL
    SELECT 'pg_type'::regclass, tipo.oid
     FROM pg_catalog.pg_type tipo
     WHERE tipo.typnamespace =
           'vec_contratacion_temporal'::regnamespace
       AND tipo.typname = ANY(ARRAY[
        'resumen_publicacion_rrhh_v1',
        'solicitud_operativa_rrhh_v1',
        'analisis_operativo_rrhh_v1',
        'comprobacion_operativa_rrhh_v1',
        'cobertura_operativa_rrhh_v1',
        'asignacion_operativa_rrhh_v1',
        'hito_expediente_rrhh_v1',
        'entrada_detalle_expediente_rrhh_v1',
        'evidencia_recibo_lectura_rrhh_v2',
        '_resumen_publicacion_rrhh_v1',
        '_solicitud_operativa_rrhh_v1',
        '_analisis_operativo_rrhh_v1',
        '_comprobacion_operativa_rrhh_v1',
        '_cobertura_operativa_rrhh_v1',
        '_asignacion_operativa_rrhh_v1',
        '_hito_expediente_rrhh_v1',
        '_entrada_detalle_expediente_rrhh_v1',
        '_evidencia_recibo_lectura_rrhh_v2'
       ]::name[])
    UNION ALL
    SELECT 'pg_class'::regclass, tipo.typrelid
      FROM pg_catalog.pg_type tipo
     WHERE tipo.typnamespace =
           'vec_contratacion_temporal'::regnamespace
       AND tipo.typrelid <> 0
       AND tipo.typname = ANY(ARRAY[
        'resumen_publicacion_rrhh_v1',
        'solicitud_operativa_rrhh_v1',
        'analisis_operativo_rrhh_v1',
        'comprobacion_operativa_rrhh_v1',
        'cobertura_operativa_rrhh_v1',
        'asignacion_operativa_rrhh_v1',
        'hito_expediente_rrhh_v1',
        'entrada_detalle_expediente_rrhh_v1',
        'evidencia_recibo_lectura_rrhh_v2'
       ]::name[])
), dependencias AS (
    SELECT dependencia.dbid
      FROM pg_catalog.pg_shdepend dependencia
     WHERE EXISTS (
        SELECT 1 FROM objetos
         WHERE clase = dependencia.classid
           AND objeto = dependencia.objid
     )
)
SELECT (
    (SELECT pg_catalog.count(DISTINCT dbid) FROM dependencias) = 2
    AND (SELECT pg_catalog.count(*) FROM dependencias
          WHERE dbid = (
            SELECT oid FROM pg_catalog.pg_database
             WHERE datname = pg_catalog.current_database()
          )) > 0
    AND (SELECT pg_catalog.count(*) FROM dependencias
          WHERE dbid = (
            SELECT oid FROM pg_catalog.pg_database
             WHERE datname = 'ct42_regresion_multibase'
          )) = (
            SELECT pg_catalog.count(*) FROM dependencias
             WHERE dbid = (
               SELECT oid FROM pg_catalog.pg_database
                WHERE datname = pg_catalog.current_database()
             )
          )
)") != t ]]; then
    printf 'la regresión no produjo OID locales coincidentes\\n' >&2
    exit 1
fi

paso 'ciclo multibase limpio DOWN/UP y vectores finales'
archivo \
    contratacion_temporal/migraciones/000042_canones_resultado_recibo_rrhh.down.sql
comprobar_estado "$estado_base" 'retirada'
esperar_fallo 'down reentrante' 55000 \
    'estado incompatible para revertir cánones RRHH' \
    archivo \
    contratacion_temporal/migraciones/000042_canones_resultado_recibo_rrhh.down.sql
comprobar_estado "$estado_base" 'down reentrante'
archivo \
    contratacion_temporal/migraciones/000042_canones_resultado_recibo_rrhh.up.sql
archivo contratacion_temporal/pruebas_sql/o405_canones_resultado_recibo_rrhh.sql
archivo contratacion_temporal/pruebas_sql/o405_material_consumo_rrhh_v3.sql
archivo contratacion_temporal/pruebas_sql/o405_recibo_lectura_rrhh_v2.sql
archivo contratacion_temporal/pruebas_sql/o405_canones_rrhh_adversarial.sql
comprobar_estado '22|6|10|18' 'segundo UP'
psql_admin --command \
    'DROP DATABASE ct42_regresion_multibase WITH (FORCE)' \
    >/dev/null

paso 'cánones y Recibo V2 CT-000042 superados'
