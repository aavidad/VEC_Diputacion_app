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
        (pg_catalog.to_regprocedure(
            'vec_contratacion_temporal.huella_solicitud_seleccion_llamamiento_o6_v1(jsonb)'
        ) IS NULL)::text
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

psql_focal --file "$migracion_up" >/dev/null

# La prueba focal genera solicitud, huella, recibo y artefacto con el canon Go,
# cruza el adaptador real, reinicia el pool, prueba replay terminal y acredita
# PUBLIC/rol ajeno/SECURITY DEFINER. No usa JSON ni huellas sintetizados en Bash.
(
    cd -- "$raiz_repositorio"
    VEC_CT_O6_INTEGRACION_PG=SI \
    VEC_CT_O6_CONSERVAR_HISTORIA=SI \
        go test ./internal/modules/contrataciontemporal/adapters/postgres \
        -run '^TestEjecucionesSeleccionO6PostgreSQLGoATerminal$' -count=1
)

if psql_focal --file "$migracion_down" \
    >"$directorio_temporal_o6/down-con-historia.out" 2>&1; then
    printf 'down O6 acepto historia durable\n' >&2
    exit 1
fi

psql_focal --command "
    SET ROLE vec_contratacion_temporal_propietario;
    DELETE FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6;
" >/dev/null
psql_focal --file "$migracion_down" >/dev/null
rm -rf -- "$directorio_temporal_o6"
trap - EXIT
printf '[CT-LITE-O6-03:PG18.4] GO focal Go-PostgreSQL-Go\n'
