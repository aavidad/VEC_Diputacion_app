#!/usr/bin/env bash
set -Eeuo pipefail

directorio="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
    pwd -P
)"
raiz_repositorio="$(git -C "$directorio" rev-parse --show-toplevel)"
migracion_up="$directorio/migraciones/000046_ejecuciones_seleccion_llamamiento_o6.up.sql"
migracion_down="$directorio/migraciones/000046_ejecuciones_seleccion_llamamiento_o6.down.sql"

if [[ ${VEC_CT_O6_BD_DESECHABLE:-} != SI ]]; then
    printf 'O6-03 exige VEC_CT_O6_BD_DESECHABLE=SI y una base exclusiva\n' >&2
    exit 64
fi
command -v psql >/dev/null 2>&1 || {
    printf 'psql no disponible\n' >&2
    exit 69
}

psql_focal() {
    psql -X --no-psqlrc --set ON_ERROR_STOP=1 --set VERBOSITY=terse "$@"
}

preflight="$(psql_focal --tuples-only --no-align --command "
    SELECT pg_catalog.concat_ws('|',
        pg_catalog.current_setting('server_version_num'),
        pg_catalog.getdatabaseencoding(),
        (SELECT rol.rolsuper::text
           FROM pg_catalog.pg_roles rol
          WHERE rol.rolname = pg_catalog.current_user),
        (pg_catalog.to_regnamespace('vec_contratacion_temporal') IS NOT NULL)::text,
        (pg_catalog.to_regrole('vec_contratacion_temporal_propietario') IS NOT NULL)::text,
        (pg_catalog.to_regrole('vec_contratacion_temporal_ejecutor') IS NOT NULL)::text,
        (pg_catalog.to_regclass(
            'vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'
        ) IS NULL)::text,
        (NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_proc funcion
             WHERE funcion.pronamespace =
                   'vec_contratacion_temporal'::pg_catalog.regnamespace
               AND funcion.proname = ANY(ARRAY[
                   'campo_canonico_seleccion_llamamiento_o6_v1',
                   'entero_json_seleccion_llamamiento_o6_v1',
                   'referencia_json_seleccion_llamamiento_o6_v1',
                   'huella_solicitud_seleccion_llamamiento_o6_v1',
                   'solicitud_json_seleccion_llamamiento_o6_v1',
                   'solicitud_desde_texto_seleccion_llamamiento_o6_v1',
                   'recibo_json_seleccion_llamamiento_o6_v1',
                   'recibo_desde_texto_seleccion_llamamiento_o6_v1',
                   'artefacto_json_seleccion_llamamiento_o6_v1',
                   'referencia_material_seleccion_llamamiento_o6_v1',
                   'contexto_material_seleccion_llamamiento_o6_v1',
                   'huellas_materiales_seleccion_llamamiento_o6_v1',
                   'materiales_ligados_seleccion_llamamiento_o6_v1',
                   'confirmacion_canonica_seleccion_llamamiento_o6_v1',
                   'resolver_terminal_seleccion_llamamiento_o6_v1',
                   'reservar_seleccion_llamamiento_o6_v1',
                   'abrir_ventana_seleccion_llamamiento_o6_v1',
                   'marcar_indeterminada_seleccion_llamamiento_o6_v1',
                   'liberar_seleccion_llamamiento_o6_v1',
                   'confirmar_seleccion_llamamiento_o6_v1',
                   'consultar_seleccion_llamamiento_o6_v1'
               ]::name[])
        ))::text
    )")"
if [[ $preflight != '180004|UTF8|true|true|true|true|true|true' ]]; then
    printf 'preflight O6-03 incompatible; se exige PG18.4 y base exclusiva preparada\n' >&2
    exit 65
fi
directorio_temporal_o6="$(mktemp -d)"

limpiar() {
    local estado=$?
    trap - EXIT
    if psql_focal --tuples-only --no-align --command "
        SELECT (pg_catalog.to_regclass(
            'vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'
        ) IS NOT NULL)::text" 2>/dev/null | rg -qx true; then
        psql_focal --command "
            SET ROLE vec_contratacion_temporal_propietario;
            DELETE FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6;
        " >/dev/null 2>&1 || true
        psql_focal --file "$migracion_down" >/dev/null 2>&1 || true
    fi
    rm -rf -- "$directorio_temporal_o6"
    exit "$estado"
}
trap limpiar EXIT

ejecutar_focal() {
    # Genera el canon Go, cruza el adaptador, usa ocho backends distintos,
    # reinicia el pool y acredita replay y minimo privilegio.
    (
        cd -- "$raiz_repositorio"
        VEC_CT_O6_INTEGRACION_PG=SI \
        VEC_CT_O6_CONSERVAR_HISTORIA=SI \
        VEC_CT_O6_SESIONES_CONCURRENTES=8 \
        GOPROXY=off \
            go test ./internal/modules/contrataciontemporal/adapters/postgres \
            -run '^TestEjecucionesSeleccionO6PostgreSQLGoATerminal$' -count=1
    )
}

exigir_down_protegido() {
    local fase=$1
    local conservada
    if psql_focal --file "$migracion_down" \
        >"$directorio_temporal_o6/down-$fase.out" 2>&1; then
        printf 'down O6 acepto historia durable en %s\n' "$fase" >&2
        exit 1
    fi
    conservada="$(psql_focal --tuples-only --no-align --command "
        SELECT pg_catalog.concat_ws('|',
            (pg_catalog.to_regclass(
                'vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'
            ) IS NOT NULL)::text,
            (EXISTS (SELECT 1 FROM
                vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6))::text
        )")"
    if [[ $conservada != 'true|true' ]]; then
        printf 'down protegido no conservo esquema e historia en %s\n' "$fase" >&2
        exit 1
    fi
}

retirar_sin_historia() {
    local fase=$1
    local retirada
    psql_focal --command "
        SET ROLE vec_contratacion_temporal_propietario;
        DELETE FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6;
    " >/dev/null
    psql_focal --file "$migracion_down" >/dev/null
    retirada="$(psql_focal --tuples-only --no-align --command "
        SELECT pg_catalog.concat_ws('|',
            (pg_catalog.to_regclass(
                'vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'
            ) IS NULL)::text,
            (NOT EXISTS (
                SELECT 1 FROM pg_catalog.pg_proc funcion
                 WHERE funcion.pronamespace =
                       'vec_contratacion_temporal'::pg_catalog.regnamespace
                   AND funcion.proname LIKE '%seleccion_llamamiento_o6_v1'
            ))::text
        )")"
    if [[ $retirada != 'true|true' ]]; then
        printf 'retirada O6 incompleta en %s\n' "$fase" >&2
        exit 1
    fi
}

psql_focal --file "$migracion_up" >/dev/null
ejecutar_focal
exigir_down_protegido primera-instalacion
retirar_sin_historia primera-instalacion
psql_focal --file "$migracion_up" >/dev/null
ejecutar_focal
exigir_down_protegido reinstalacion
retirar_sin_historia reinstalacion
rm -rf -- "$directorio_temporal_o6"
trap - EXIT
printf '[CT-LITE-O6-03:PG18.4] GO ocho sesiones y UP/DOWN protegido/UP repetido\n'
