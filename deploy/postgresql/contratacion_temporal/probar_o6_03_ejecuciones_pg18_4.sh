#!/usr/bin/env bash
set -Eeuo pipefail

readonly prefijo='ct-o6-rem02-pg-20260831'
readonly etiqueta='vec.prueba=ct-o6-rem02-pg-20260831'
readonly imagen='postgres@sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382'
readonly ajeno='d6217278d871'
readonly contenedor="${prefijo}-db"
readonly red="${prefijo}-net"
readonly volumen="${prefijo}-datos"

directorio="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd -P)"
raiz_repositorio="$(git -C "$directorio" rev-parse --show-toplevel)"
if [[ ${VEC_CT_O6_BD_DESECHABLE:-} != SI ]]; then
    printf 'REM-02 exige VEC_CT_O6_BD_DESECHABLE=SI\n' >&2
    exit 64
fi
command -v docker >/dev/null 2>&1 || { printf 'docker no disponible\n' >&2; exit 69; }
command -v rg >/dev/null 2>&1 || { printf 'rg no disponible\n' >&2; exit 69; }
command -v flock >/dev/null 2>&1 || { printf 'flock no disponible\n' >&2; exit 69; }

unset PGHOST PGHOSTADDR PGPORT PGDATABASE PGUSER PGPASSWORD PGPASSFILE PGSERVICE
unset PGSERVICEFILE PGOPTIONS PGAPPNAME PGSSLMODE PGCONNECT_TIMEOUT PGCLIENTENCODING
unset PGTARGETSESSIONATTRS PGLOADBALANCEHOSTS PGCHANNELBINDING PGREQUIREAUTH
unset PGSSLCERT PGSSLKEY PGSSLROOTCERT PGSSLCRL PGSSLCRLDIR PGREQUIREPEER
unset PGGSSENCMODE PGKRBSRVNAME PGGSSLIB PGSYSCONFDIR PGLOCALEDIR
unset DATABASE_URL DB_DSN DSN
unset VEC_CT_O6_PG_DSN VEC_DATABASE_URL

exec 9>/tmp/vec-postgres-dynamic-20260831.lock
if ! flock -w 900 9; then
    printf 'no se adquirio el bloqueo PostgreSQL dinamico\n' >&2
    exit 75
fi

if [[ -n $(docker ps -aq --filter "name=^/${contenedor}$") ]] ||
   docker network inspect "$red" >/dev/null 2>&1 ||
   [[ -n $(docker volume ls -q --filter "name=^${volumen}$") ]] ||
   find /tmp /var/tmp -maxdepth 1 -name "${prefijo}*" -print -quit | rg -q .; then
    printf 'colision de recursos propios REM-02\n' >&2
    exit 65
fi

imagen_id="$(docker image inspect --format '{{.Id}}' "$imagen")"
if [[ $imagen_id != 'sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382' ]]; then
    printf 'imagen local REM-02 divergente\n' >&2
    exit 65
fi
ajeno_antes="$(docker inspect --format '{{.Id}}|{{.State.Status}}|{{.Name}}|{{.Image}}' "$ajeno")"
printf 'ajeno antes: %s\n' "$ajeno_antes"

temporal="$(mktemp -d "/tmp/${prefijo}.XXXXXX")"
socket="$temporal/socket"
mkdir -m 0777 -- "$socket"
creado_contenedor=0
creada_red=0
creado_volumen=0

limpiar() {
    local estado=$?
    local residuo=0
    trap - EXIT
    if (( creado_contenedor )); then
        docker rm -f -v "$contenedor" >/dev/null 2>&1 || residuo=1
    fi
    if (( creada_red )); then
        docker network rm -- "$red" >/dev/null 2>&1 || residuo=1
    fi
    if (( creado_volumen )); then
        docker volume rm -- "$volumen" >/dev/null 2>&1 || residuo=1
    fi
    rm -rf -- "$temporal"
    if [[ -n $(docker ps -aq --filter "label=$etiqueta") ]] ||
       [[ -n $(docker network ls -q --filter "label=$etiqueta") ]] ||
       [[ -n $(docker volume ls -q --filter "label=$etiqueta") ]] ||
       find /tmp /var/tmp -maxdepth 1 -name "${prefijo}*" -print -quit | rg -q .; then
        residuo=1
    fi
    ajeno_despues="$(docker inspect --format '{{.Id}}|{{.State.Status}}|{{.Name}}|{{.Image}}' "$ajeno" 2>/dev/null || true)"
    printf 'ajeno despues: %s\n' "$ajeno_despues"
    if [[ $ajeno_despues != "$ajeno_antes" ]]; then
        printf 'el contenedor ajeno cambio durante REM-02\n' >&2
        residuo=1
    fi
    if (( residuo )); then
        printf 'limpieza REM-02 incompleta\n' >&2
        exit 1
    fi
    exit "$estado"
}
trap limpiar EXIT

docker network create --internal --label "$etiqueta" "$red" >/dev/null
creada_red=1
docker volume create --label "$etiqueta" "$volumen" >/dev/null
creado_volumen=1
creado_contenedor=1
docker run -d --pull=never --name "$contenedor" --label "$etiqueta" \
    --network "$red" --memory=1280m --cpus=2 --pids-limit=256 \
    --mount "type=volume,src=$volumen,dst=/var/lib/postgresql" \
    --mount "type=bind,src=$socket,dst=/var/run/postgresql" \
    --mount "type=bind,src=$directorio,dst=/autoridad,readonly" \
    --env POSTGRES_HOST_AUTH_METHOD=trust --env POSTGRES_INITDB_ARGS='--encoding=UTF8' \
    "$imagen" -c listen_addresses= -c unix_socket_directories=/var/run/postgresql >/dev/null

listo=0
for _ in {1..80}; do
    if docker exec "$contenedor" pg_isready -q -h /var/run/postgresql -U postgres \
        2>/dev/null; then
        listo=1
        break
    fi
    if [[ $(docker inspect --format '{{.State.Running}}' "$contenedor") != true ]]; then
        printf 'PostgreSQL REM-02 termino durante la inicializacion\n' >&2
        break
    fi
    sleep 0.25
done
if (( ! listo )); then
    printf 'PostgreSQL REM-02 no quedo listo\n' >&2
    exit 1
fi

psql_focal() {
    docker exec -i "$contenedor" psql -X --no-psqlrc -h /var/run/postgresql \
        -U postgres -d postgres --set ON_ERROR_STOP=1 --set VERBOSITY=terse "$@"
}

psql_focal --file /autoridad/roles_up.sql >/dev/null
psql_focal >/dev/null <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
CREATE FUNCTION vec_contratacion_temporal.instante_utc_json_canonico_v2(
    p_valor jsonb, p_fecha_civil boolean
) RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE v_texto text; v_instante timestamptz;
BEGIN
    IF jsonb_typeof(p_valor) <> 'string' OR p_fecha_civil IS NULL THEN RETURN false; END IF;
    v_texto := p_valor #>> '{}';
    IF (p_fecha_civil AND v_texto !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T00:00:00Z$') OR
       (NOT p_fecha_civil AND v_texto !~ ('^[0-9]{4}-[0-9]{2}-[0-9]{2}T'
        || '[0-2][0-9]:[0-5][0-9]:[0-5][0-9]([.][0-9]{0,5}[1-9])?Z$')) THEN
        RETURN false;
    END IF;
    v_instante := v_texto::timestamptz;
    RETURN isfinite(v_instante) AND extract(microseconds FROM v_instante) =
        trunc(extract(microseconds FROM v_instante));
EXCEPTION WHEN data_exception OR datetime_field_overflow OR invalid_text_representation THEN
    RETURN false;
END $f$;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.instante_utc_json_canonico_v2(jsonb, boolean) FROM PUBLIC;
COMMIT;
SQL

preflight="$(psql_focal --tuples-only --no-align --command "SELECT concat_ws('|',
 current_setting('server_version_num'), getdatabaseencoding(),
 (SELECT rolsuper::text FROM pg_roles WHERE rolname=current_user),
 (to_regnamespace('vec_contratacion_temporal') IS NOT NULL)::text,
 (to_regrole('vec_contratacion_temporal_propietario') IS NOT NULL)::text,
 (to_regrole('vec_contratacion_temporal_ejecutor') IS NOT NULL)::text,
 (to_regclass('vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6') IS NULL)::text)")"
if [[ $preflight != '180004|UTF8|true|true|true|true|true' ]]; then
    printf 'preflight PostgreSQL REM-02 incompatible: %s\n' "$preflight" >&2
    exit 1
fi

ejecutar_go() {
    (
        cd -- "$raiz_repositorio"
        PGHOST="$socket" PGPORT=5432 PGDATABASE=postgres PGUSER=postgres PGSSLMODE=disable \
        VEC_CT_O6_INTEGRACION_PG=SI VEC_CT_O6_BD_DESECHABLE=SI \
        VEC_CT_O6_CONSERVAR_HISTORIA=SI VEC_CT_O6_SESIONES_CONCURRENTES=8 GOPROXY=off \
            go test ./internal/modules/contrataciontemporal/adapters/postgres \
            -run '^TestEjecucionesSeleccionO6PostgreSQLGoATerminal$' -count=1
    )
}

exigir_down_protegido() {
    local fase=$1
    if psql_focal --file /autoridad/migraciones/000046_ejecuciones_seleccion_llamamiento_o6.down.sql \
        >"$temporal/down-$fase.out" 2>&1; then
        printf 'down acepto historia durable en %s\n' "$fase" >&2
        exit 1
    fi
}

retirar_sin_historia() {
    psql_focal --command "SET ROLE vec_contratacion_temporal_propietario;
      DELETE FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6" >/dev/null
    psql_focal --file /autoridad/migraciones/000046_ejecuciones_seleccion_llamamiento_o6.down.sql >/dev/null
}

psql_focal --file /autoridad/migraciones/000046_ejecuciones_seleccion_llamamiento_o6.up.sql >/dev/null
ejecutar_go
exigir_down_protegido primera-instalacion
retirar_sin_historia
psql_focal --file /autoridad/migraciones/000046_ejecuciones_seleccion_llamamiento_o6.up.sql >/dev/null
ejecutar_go
exigir_down_protegido reinstalacion
retirar_sin_historia

printf '[CT-LITE-O6-REM-02:PG18.4] GO; lease/fencing/replay/ACL y UP-DOWN-UP\n'
