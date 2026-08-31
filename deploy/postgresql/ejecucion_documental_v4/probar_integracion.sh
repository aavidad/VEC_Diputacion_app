#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
prefijo=${VEC_POSTGRES_TEST_PREFIX:-vecdoc-r5-${UID}-$$}
propietario=ejecucion_documental_v4
tarea=VEC-DOC-RUNNER-AISLADO-R5
etiqueta_propietario="vec.propietario=$propietario"
etiqueta_tarea="vec.tarea=$tarea"
etiqueta_prefijo="vec.prefijo=$prefijo"
contenedor="vec-ejecucion-v4-pg-${prefijo}"
red="${prefijo}-internal"
socket_contenedor=/run/vec-postgresql
socket_entrypoint=/var/run/postgresql
directorio_socket=
firma_directorio=
contenedor_id=
red_id=
base=vec_ejecucion_v4_prueba
clave_admin="admin-v4-$$"
clave_fuente="fuente-v4-$$"
clave_registro="registro-v4-$$"
clave_emisor="emisor-v4-$$"
clave_ejecucion="ejecucion-v4-$$"

if (( ${#prefijo} < 8 || ${#prefijo} > 48 )) \
    || [[ ! "$prefijo" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]]; then
    echo "VEC_POSTGRES_TEST_PREFIX debe tener 8..48 caracteres [a-z0-9-], sin guiones consecutivos ni extremos" >&2
    exit 1
fi
if [[ ! "$imagen" =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]]; then
    echo "VEC_POSTGRES_TEST_IMAGE debe ser una referencia exacta nombre@sha256" >&2
    exit 1
fi
if ! imagen_id=$(docker image inspect --format '{{.Id}}' "$imagen" 2>/dev/null); then
    echo "la imagen PostgreSQL exacta no existe localmente" >&2
    exit 1
fi
if [[ ! "$imagen_id" =~ ^sha256:[0-9a-f]{64}$ ]]; then
    echo "la imagen PostgreSQL local no resolvio a un identificador exacto" >&2
    exit 1
fi
if docker container inspect "$contenedor" >/dev/null 2>&1 \
    || docker network inspect "$red" >/dev/null 2>&1; then
    echo "el prefijo solicitado colisiona con un nombre Docker preexistente" >&2
    exit 1
fi
if [[ -n "$(docker container ls --all --quiet --filter "label=$etiqueta_prefijo")" \
    || -n "$(docker network ls --quiet --filter "label=$etiqueta_prefijo")" \
    || -n "$(docker volume ls --quiet --filter "label=$etiqueta_prefijo")" ]]; then
    echo "el prefijo solicitado ya etiqueta recursos Docker preexistentes" >&2
    exit 1
fi

limpiar() {
    local estado=$1 error_limpieza=0 firma firma_actual residuos
    trap - EXIT INT TERM
    set +e

    if [[ -n "$contenedor_id" ]] \
        && docker container inspect "$contenedor_id" >/dev/null 2>&1; then
        firma=$(docker container inspect --format \
            '{{.Name}}|{{index .Config.Labels "vec.propietario"}}|{{index .Config.Labels "vec.tarea"}}|{{index .Config.Labels "vec.prefijo"}}' \
            "$contenedor_id" 2>/dev/null)
        if [[ "$firma" != "/$contenedor|$propietario|$tarea|$prefijo" ]]; then
            echo "limpieza denegada: el contenedor exacto no conserva su identidad" >&2
            error_limpieza=1
        elif ! docker container rm --force --volumes "$contenedor_id" >/dev/null; then
            echo "no se pudo retirar el contenedor propio exacto" >&2
            error_limpieza=1
        fi
    fi

    if [[ -n "$red_id" ]] && docker network inspect "$red_id" >/dev/null 2>&1; then
        firma=$(docker network inspect --format \
            '{{.Name}}|{{index .Labels "vec.propietario"}}|{{index .Labels "vec.tarea"}}|{{index .Labels "vec.prefijo"}}' \
            "$red_id" 2>/dev/null)
        if [[ "$firma" != "$red|$propietario|$tarea|$prefijo" ]]; then
            echo "limpieza denegada: la red exacta no conserva su identidad" >&2
            error_limpieza=1
        elif ! docker network rm "$red_id" >/dev/null; then
            echo "no se pudo retirar la red propia exacta" >&2
            error_limpieza=1
        fi
    fi

    if [[ -n "$directorio_socket" && -e "$directorio_socket" ]]; then
        firma_actual=$(stat -c '%d:%i:%u' -- "$directorio_socket" 2>/dev/null)
        if [[ ! -d "$directorio_socket" || -L "$directorio_socket" \
            || "$firma_actual" != "$firma_directorio" ]]; then
            echo "limpieza denegada: el directorio de socket cambio de identidad" >&2
            error_limpieza=1
        else
            rm -f -- "$directorio_socket/.s.PGSQL.5432" \
                "$directorio_socket/.s.PGSQL.5432.lock" || error_limpieza=1
            rmdir -- "$directorio_socket" || error_limpieza=1
        fi
    fi

    if ! residuos=$(
        docker container ls --all --quiet --filter "name=^${contenedor}$" &&
            docker container ls --all --quiet \
                --filter "label=$etiqueta_prefijo" &&
            docker network ls --quiet --filter "name=^${red}$" &&
            docker network ls --quiet --filter "label=$etiqueta_prefijo" &&
            docker volume ls --quiet --filter "label=$etiqueta_prefijo"
    ); then
        echo "la limpieza propia no pudo consultar los residuos Docker" >&2
        error_limpieza=1
    elif [[ -n "$residuos" \
        || -n "$directorio_socket" && -e "$directorio_socket" ]]; then
        echo "la limpieza propia no pudo acreditar residuos cero" >&2
        error_limpieza=1
    elif (( error_limpieza == 0 )); then
        echo "limpieza $prefijo: contenedor, red y socket con residuos cero"
    fi

    if (( estado == 0 && error_limpieza != 0 )); then
        estado=1
    fi
    exit "$estado"
}
trap 'limpiar $?' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

redactar_logs_arranque() {
    local logs=$1 secreto
    for secreto in "$clave_admin" "$clave_fuente" "$clave_registro" \
        "$clave_emisor" "$clave_ejecucion"
    do
        logs=${logs//"$secreto"/[REDACTADO]}
    done
    logs=$(sed -E \
        -e 's/((PGPASSWORD|POSTGRES_PASSWORD|[Pp][Aa][Ss][Ss][Ww][Oo][Rr][Dd])[=:])[^[:space:]]+/\1[REDACTADO]/g' \
        -e 's#(postgres(ql)?://[^:/[:space:]]+:)[^@/[:space:]]+#\1[REDACTADO]#g' \
        <<<"$logs")
    printf '%.16384s' "$logs"
}

diagnosticar_contenedor_detenido() {
    local estado codigo_salida oom logs
    estado=$(docker container inspect --format '{{.State.Status}}' \
        "$contenedor_id" 2>/dev/null || printf 'no_disponible')
    codigo_salida=$(docker container inspect --format '{{.State.ExitCode}}' \
        "$contenedor_id" 2>/dev/null || printf 'no_disponible')
    oom=$(docker container inspect --format '{{.State.OOMKilled}}' \
        "$contenedor_id" 2>/dev/null || printf 'no_disponible')
    logs=$(docker container logs --tail 80 "$contenedor_id" 2>&1 || true)
    logs=$(redactar_logs_arranque "$logs")
    echo "PostgreSQL se detuvo durante el arranque: estado=$estado codigo_salida=$codigo_salida oom=$oom" >&2
    if [[ -n "$logs" ]]; then
        echo "ultimos logs de arranque (maximo 80 lineas/16384 caracteres, credenciales redactadas):" >&2
        printf '%s\n' "$logs" >&2
    fi
}

exigir_contenedor_en_ejecucion() {
    local ejecutando
    if ! ejecutando=$(docker container inspect --format '{{.State.Running}}' \
        "$contenedor_id" 2>/dev/null); then
        echo "no se pudo inspeccionar el estado del contenedor propio" >&2
        return 1
    fi
    if [[ "$ejecutando" == "true" ]]; then
        return 0
    fi
    diagnosticar_contenedor_detenido
    return 1
}

directorio_socket=$(mktemp -d -p /tmp "${prefijo}.socket.XXXXXXXX")
if [[ ! -O "$directorio_socket" || -L "$directorio_socket" ]]; then
    echo "mktemp no creo un directorio de socket propio" >&2
    exit 1
fi
firma_directorio=$(stat -c '%d:%i:%u' -- "$directorio_socket")
chmod 1777 "$directorio_socket"

red_id=$(docker network create --driver bridge --internal \
    --label "$etiqueta_propietario" --label "$etiqueta_tarea" \
    --label "$etiqueta_prefijo" "$red")

psql_admin() {
    docker exec --interactive --env PGPASSWORD="$clave_admin" \
        "$contenedor_id" psql --no-psqlrc --quiet --set ON_ERROR_STOP=1 \
        --host "$socket_contenedor" --username postgres --dbname "$base" "$@"
}

aplicar() {
    psql_admin < "$raiz/$1"
}

psql_login() {
    local usuario=$1
    local clave=$2
    shift 2
    docker exec --env PGPASSWORD="$clave" "$contenedor_id" \
        psql --no-psqlrc --quiet --set ON_ERROR_STOP=1 \
        --host "$socket_contenedor" \
        --username "$usuario" --dbname "$base" "$@"
}

exigir_rechazo_login() {
    local usuario=$1
    local clave=$2
    local consulta=$3
    local descripcion=$4
    if psql_login "$usuario" "$clave" --command "$consulta" \
        >/dev/null 2>&1; then
        echo "ACL invalida: $descripcion" >&2
        exit 1
    fi
}

exigir_sin_uso_tipo() {
    local usuario=$1 clave=$2 etiqueta=$3 privilegio
    privilegio=$(psql_login "$usuario" "$clave" --tuples-only --no-align \
        --command "SELECT has_type_privilege(current_user, 'vec_ejecucion_documental_v4.atestacion_pdp', 'USAGE')")
    if [[ "$privilegio" != "f" ]]; then
        echo "ACL invalida: $etiqueta conserva USAGE sobre un tipo V4" >&2
        exit 1
    fi
}

if ! contenedor_id=$(docker run --detach --name "$contenedor" --pull=never \
    --network "$red" --read-only --cpus 2 --memory 1280m --pids-limit 256 \
    --label "$etiqueta_propietario" --label "$etiqueta_tarea" \
    --label "$etiqueta_prefijo" \
    --tmpfs /var/lib/postgresql:rw,noexec,nosuid,nodev,size=768m,mode=1777 \
    --tmpfs /tmp:rw,noexec,nosuid,nodev,size=64m,mode=1777 \
    --tmpfs "$socket_entrypoint:rw,noexec,nosuid,nodev,size=8m,mode=3775" \
    --mount "type=bind,src=$directorio_socket,dst=$socket_contenedor" \
    --env PGDATA=/var/lib/postgresql/18/docker \
    --env PGHOST="$socket_contenedor" \
    --env PGPASSWORD="$clave_admin" \
    --env POSTGRES_INITDB_ARGS="--auth-local=scram-sha-256 --auth-host=scram-sha-256" \
    --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave_admin" \
    "$imagen" -c listen_addresses= \
    -c "unix_socket_directories=$socket_contenedor,$socket_entrypoint"); then
    if firma=$(docker container inspect --format \
        '{{.Id}}|{{index .Config.Labels "vec.propietario"}}|{{index .Config.Labels "vec.tarea"}}|{{index .Config.Labels "vec.prefijo"}}' \
        "$contenedor" 2>/dev/null) \
        && [[ "$firma" == *"|$propietario|$tarea|$prefijo" ]]; then
        contenedor_id=${firma%%|*}
    fi
    echo "no se pudo iniciar el contenedor PostgreSQL aislado" >&2
    exit 1
fi

configuracion=$(docker container inspect --format \
    '{{.Image}}|{{.HostConfig.ReadonlyRootfs}}|{{.HostConfig.NanoCpus}}|{{.HostConfig.Memory}}|{{.HostConfig.PidsLimit}}|{{.HostConfig.NetworkMode}}|{{json .HostConfig.PortBindings}}' \
    "$contenedor_id")
if [[ "$configuracion" != "$imagen_id|true|2000000000|1342177280|256|$red|{}" ]]; then
    echo "el contenedor no conserva imagen, aislamiento o limites exactos" >&2
    exit 1
fi
firma=$(docker container inspect --format \
    '{{.Name}}|{{index .Config.Labels "vec.propietario"}}|{{index .Config.Labels "vec.tarea"}}|{{index .Config.Labels "vec.prefijo"}}' \
    "$contenedor_id")
if [[ "$firma" != "/$contenedor|$propietario|$tarea|$prefijo" ]]; then
    echo "el contenedor no conserva nombre y etiquetas propios" >&2
    exit 1
fi
firma=$(docker network inspect --format \
    '{{.Name}}|{{.Driver}}|{{.Internal}}|{{index .Labels "vec.propietario"}}|{{index .Labels "vec.tarea"}}|{{index .Labels "vec.prefijo"}}' \
    "$red_id")
if [[ "$firma" != "$red|bridge|true|$propietario|$tarea|$prefijo" ]]; then
    echo "la red no conserva nombre, aislamiento y etiquetas propios" >&2
    exit 1
fi
redes_contenedor=$(docker container inspect --format \
    '{{range $nombre, $datos := .NetworkSettings.Networks}}{{$nombre}}{{"\n"}}{{end}}' \
    "$contenedor_id")
if [[ "$redes_contenedor" != "$red" ]]; then
    echo "el contenedor tiene una conexion de red distinta de la interna propia" >&2
    exit 1
fi
tmpfs=$(docker container inspect --format \
    '{{index .HostConfig.Tmpfs "/var/lib/postgresql"}}|{{index .HostConfig.Tmpfs "/tmp"}}|{{index .HostConfig.Tmpfs "/var/run/postgresql"}}' \
    "$contenedor_id")
if [[ "$tmpfs" != "rw,noexec,nosuid,nodev,size=768m,mode=1777|rw,noexec,nosuid,nodev,size=64m,mode=1777|rw,noexec,nosuid,nodev,size=8m,mode=3775" ]]; then
    echo "el contenedor no conserva los tmpfs minimos exactos" >&2
    exit 1
fi
montaje_socket=$(docker container inspect --format \
    "{{range .Mounts}}{{if eq .Destination \"$socket_contenedor\"}}{{.Type}}|{{.Source}}|{{.RW}}{{end}}{{end}}" \
    "$contenedor_id")
if [[ "$montaje_socket" != "bind|$directorio_socket|true" ]]; then
    echo "el socket no usa el montaje temporal propio exacto" >&2
    exit 1
fi
if [[ -n "$(docker container inspect --format \
    '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' \
    "$contenedor_id")" ]]; then
    echo "el contenedor creo un volumen Docker no autorizado" >&2
    exit 1
fi
postgres_listo=false
for _ in $(seq 1 60); do
    exigir_contenedor_en_ejecucion || exit 1
    if docker exec "$contenedor_id" pg_isready --host "$socket_contenedor" \
        --username postgres \
        --dbname "$base" >/dev/null 2>&1; then
        postgres_listo=true
        break
    fi
    exigir_contenedor_en_ejecucion || exit 1
    sleep 1
done
if [[ "$postgres_listo" != "true" ]]; then
    echo "PostgreSQL no quedo disponible antes del limite de 60 segundos" >&2
    exit 1
fi
if [[ ! -S "$directorio_socket/.s.PGSQL.5432" ]]; then
    echo "PostgreSQL no publico el socket Unix en el directorio propio" >&2
    exit 1
fi
for socket in "$socket_contenedor" "$socket_entrypoint"; do
    if ! docker exec "$contenedor_id" test -S "$socket/.s.PGSQL.5432"; then
        exigir_contenedor_en_ejecucion || exit 1
        echo "PostgreSQL no publico el socket Unix esperado en $socket" >&2
        exit 1
    fi
    if ! docker exec "$contenedor_id" pg_isready --host "$socket" \
        --username postgres --dbname "$base" >/dev/null 2>&1; then
        exigir_contenedor_en_ejecucion || exit 1
        echo "PostgreSQL no acepta conexiones por el socket esperado en $socket" >&2
        exit 1
    fi
done
version_postgresql=$(psql_admin --tuples-only --no-align \
    --command 'SHOW server_version_num')
if [[ "$version_postgresql" != "180004" ]]; then
    echo "se requiere PostgreSQL 18.4, no ${version_postgresql}" >&2
    exit 1
fi
directorios_socket_postgresql=$(psql_admin --tuples-only --no-align \
    --command 'SHOW unix_socket_directories')
if [[ "$directorios_socket_postgresql" != "$socket_contenedor,$socket_entrypoint" ]]; then
    echo "PostgreSQL no conserva los dos directorios de socket exactos" >&2
    exit 1
fi

psql_admin <<'SQL'
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;
SQL
aplicar deploy/postgresql/autorizacion/roles_up.sql
aplicar deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql

# El bootstrap V4 sigue reservado al superusuario. Una cuenta CREATEROLE que
# sea propietaria de la base debe fallar sin dejar roles ni guarda parciales.
psql_admin <<'SQL'
CREATE ROLE vec_v4_instalador_no_super_prueba NOLOGIN NOSUPERUSER
    NOCREATEDB CREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
DO $propietario$
BEGIN
    EXECUTE format(
        'ALTER DATABASE %I OWNER TO vec_v4_instalador_no_super_prueba',
        current_database()
    );
END
$propietario$;
SQL
if psql_admin \
    --command 'SET SESSION AUTHORIZATION vec_v4_instalador_no_super_prueba' \
    --file=- \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_up.sql" \
    >/dev/null 2>&1; then
    echo "roles_up V4 acepto un CREATEROLE no superusuario" >&2
    exit 1
fi
psql_admin <<'SQL'
DO $sin_estado$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname LIKE 'vec_ejecucion_documental_v4_%'
    ) OR to_regnamespace('vec_ejecucion_documental_v4_guardia') IS NOT NULL
      OR EXISTS (
          SELECT 1 FROM pg_catalog.pg_event_trigger
           WHERE evtname = 'vec_ejecucion_documental_v4_cerrar_acl_tipos'
      ) THEN
        RAISE EXCEPTION 'el bootstrap no superusuario dejo estado';
    END IF;
    EXECUTE format('ALTER DATABASE %I OWNER TO postgres', current_database());
END
$sin_estado$;
DROP ROLE vec_v4_instalador_no_super_prueba;
SQL

# Un bootstrap interrumpido antes de la migracion debe retirarse y reinstalarse.
aplicar deploy/postgresql/ejecucion_documental_v4/roles_up.sql
aplicar deploy/postgresql/ejecucion_documental_v4/roles_down.sql
psql_admin <<'SQL'
DO $bootstrap_parcial_retirado$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname LIKE 'vec_ejecucion_documental_v4_%'
    ) OR to_regnamespace('vec_ejecucion_documental_v4_guardia') IS NOT NULL
      OR EXISTS (
          SELECT 1 FROM pg_catalog.pg_event_trigger
           WHERE evtname = 'vec_ejecucion_documental_v4_cerrar_acl_tipos'
      ) OR EXISTS (
          SELECT 1 FROM pg_catalog.pg_default_acl AS defecto
          JOIN pg_catalog.pg_roles AS rol ON rol.oid = defecto.defaclrole
           WHERE rol.rolname LIKE 'vec_ejecucion_documental_v4_%'
      ) THEN
        RAISE EXCEPTION 'la retirada del bootstrap V4 parcial dejo estado';
    END IF;
END
$bootstrap_parcial_retirado$;
SQL
aplicar deploy/postgresql/ejecucion_documental_v4/roles_up.sql

# Conserva una decision historica, aplica la autoridad actual de identidad y
# monta despues el esquema V4. El orden es parte del contrato reproducible.
aplicar deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
aplicar deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.down.sql
aplicar deploy/postgresql/ejecucion_documental_v4/pruebas_sql/sembrar_decision_legacy_v1.sql
aplicar deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
aplicar deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000003_revalidacion_ejecucion_documental_v4.up.sql
aplicar deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.up.sql
down_v4=deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.down.sql
if aplicar "$down_v4" >/dev/null 2>&1; then
    echo "el down V4 vacio acepto retirar el esquema sin opt-in" >&2
    exit 1
fi
psql_admin <<'SQL'
DO $conservada_vacia$
BEGIN
    IF to_regnamespace('vec_ejecucion_documental_v4') IS NULL THEN
        RAISE EXCEPTION 'el down vacio fallido no conservo el esquema';
    END IF;
END
$conservada_vacia$;
SET ROLE vec_ejecucion_documental_v4_propietario;
CREATE TABLE vec_ejecucion_documental_v4.evidencia_futura_prueba (
    evidencia_id bigint PRIMARY KEY
);
INSERT INTO vec_ejecucion_documental_v4.evidencia_futura_prueba VALUES (1);
RESET ROLE;
DO $acl_tipo_futuro$
BEGIN
    IF has_type_privilege(
        'vec_ejecucion_documental_v4_emisor_capacidad',
        'vec_ejecucion_documental_v4.evidencia_futura_prueba', 'USAGE'
    ) OR has_type_privilege(
        'vec_ejecucion_documental_v4_ejecutor_atestado',
        'vec_ejecucion_documental_v4.evidencia_futura_prueba', 'USAGE'
    ) THEN
        RAISE EXCEPTION 'la guarda DDL no cerro el tipo fila futuro';
    END IF;
END
$acl_tipo_futuro$;
SQL
if aplicar "$down_v4" >/dev/null 2>&1; then
    echo "una relacion V4 futura eludio el opt-in destructivo" >&2
    exit 1
fi
psql_admin <<'SQL'
DO $conservada_futura$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM vec_ejecucion_documental_v4.evidencia_futura_prueba
         WHERE evidencia_id = 1
    ) THEN
        RAISE EXCEPTION 'el down fallido no conservo la evidencia futura';
    END IF;
END
$conservada_futura$;
CREATE SCHEMA vec_v4_dependencia_externa_prueba;
CREATE VIEW vec_v4_dependencia_externa_prueba.vista_evidencia AS
    SELECT evidencia_id
      FROM vec_ejecucion_documental_v4.evidencia_futura_prueba;
SQL
if docker exec --interactive \
    --env PGPASSWORD="$clave_admin" \
    --env PGOPTIONS="-c vec.confirmar_destruccion_ejecucion_documental_v4=DESTRUIR_EVIDENCIA_V4_IRREVERSIBLE" \
    "$contenedor_id" psql --no-psqlrc --quiet --set ON_ERROR_STOP=1 \
    --host "$socket_contenedor" \
    --username postgres --dbname "$base" < "$raiz/$down_v4" \
    >/dev/null 2>&1; then
    echo "el down V4 destruyo una dependencia externa" >&2
    exit 1
fi
psql_admin <<'SQL'
DO $dependencia_conservada$
BEGIN
    IF to_regclass(
        'vec_v4_dependencia_externa_prueba.vista_evidencia'
    ) IS NULL OR NOT EXISTS (
        SELECT 1 FROM vec_ejecucion_documental_v4.evidencia_futura_prueba
         WHERE evidencia_id = 1
    ) THEN
        RAISE EXCEPTION 'el down fallido no conservo la dependencia externa';
    END IF;
END
$dependencia_conservada$;
DROP VIEW vec_v4_dependencia_externa_prueba.vista_evidencia;
DROP SCHEMA vec_v4_dependencia_externa_prueba;
SQL
docker exec --interactive \
    --env PGPASSWORD="$clave_admin" \
    --env PGOPTIONS="-c vec.confirmar_destruccion_ejecucion_documental_v4=DESTRUIR_EVIDENCIA_V4_IRREVERSIBLE" \
    "$contenedor_id" psql --no-psqlrc --quiet --set ON_ERROR_STOP=1 \
    --host "$socket_contenedor" \
    --username postgres --dbname "$base" < "$raiz/$down_v4"
aplicar deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.up.sql
aplicar deploy/postgresql/ejecucion_documental_v4/pruebas_sql/revalidacion_identidad_v1.sql
aplicar deploy/postgresql/ejecucion_documental_v4/pruebas_sql/privilegios_minimos_v2.sql

psql_admin \
    --set clave_fuente="$clave_fuente" \
    --set clave_registro="$clave_registro" \
    --set clave_emisor="$clave_emisor" \
    --set clave_ejecucion="$clave_ejecucion" <<'SQL'
CREATE ROLE vec_v4_fuente_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS PASSWORD :'clave_fuente';
CREATE ROLE vec_v4_registro_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS PASSWORD :'clave_registro';
CREATE ROLE vec_v4_emisor_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS PASSWORD :'clave_emisor';
CREATE ROLE vec_v4_ejecucion_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS PASSWORD :'clave_ejecucion';
GRANT vec_autorizacion_fuente TO vec_v4_fuente_prueba;
GRANT vec_autorizacion_registro TO vec_v4_registro_prueba;
GRANT vec_ejecucion_documental_v4_emisor_capacidad TO vec_v4_emisor_prueba;
GRANT vec_ejecucion_documental_v4_ejecutor_atestado TO vec_v4_ejecucion_prueba;
SQL

# Las dos identidades runtime siguen separadas y no ejecutan pgcrypto.
for identidad in \
    "vec_v4_emisor_prueba:$clave_emisor:emisor" \
    "vec_v4_ejecucion_prueba:$clave_ejecucion:ejecutor"
do
    IFS=: read -r usuario clave etiqueta <<<"$identidad"
    exigir_rechazo_login "$usuario" "$clave" \
        "SELECT public.hmac(decode('00','hex'),decode('00','hex'),'sha256')" \
        "$etiqueta ejecuto HMAC directamente"
    exigir_rechazo_login "$usuario" "$clave" \
        "SELECT public.digest(decode('00','hex'),'sha256')" \
        "$etiqueta ejecuto otra funcion pgcrypto"
    exigir_rechazo_login "$usuario" "$clave" \
        "SELECT * FROM vec_ejecucion_documental_v4.atestacion_pdp" \
        "$etiqueta leyo evidencia V4"
    exigir_sin_uso_tipo "$usuario" "$clave" "$etiqueta"
done

# El propietario solo ejecuta el overload HMAC bytea exacto.
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_ejecucion_documental_v4_propietario;
SELECT octet_length(public.hmac(
    decode('00', 'hex'), decode('00', 'hex'), 'sha256'
));
ROLLBACK;
SQL
if psql_admin --command "SET ROLE vec_ejecucion_documental_v4_propietario; SELECT public.hmac('dato'::text,'clave'::text,'sha256')" \
    >/dev/null 2>&1; then
    echo "el propietario ejecuto el overload HMAC text no autorizado" >&2
    exit 1
fi
if psql_admin --command "SET ROLE vec_ejecucion_documental_v4_propietario; SELECT public.digest(decode('00','hex'),'sha256')" \
    >/dev/null 2>&1; then
    echo "el propietario ejecuto una funcion pgcrypto distinta de HMAC bytea" >&2
    exit 1
fi

export VEC_POSTGRES_TEST_FUENTE_DSN="host=$directorio_socket port=5432 user=vec_v4_fuente_prueba password=$clave_fuente dbname=$base sslmode=disable"
export VEC_POSTGRES_TEST_REGISTRO_DSN="host=$directorio_socket port=5432 user=vec_v4_registro_prueba password=$clave_registro dbname=$base sslmode=disable"
export VEC_POSTGRES_TEST_ADMIN_DSN="host=$directorio_socket port=5432 user=postgres password=$clave_admin dbname=$base sslmode=disable"
export VEC_POSTGRES_TEST_V4_EMISOR_DSN="host=$directorio_socket port=5432 user=vec_v4_emisor_prueba password=$clave_emisor dbname=$base sslmode=disable"
export VEC_POSTGRES_TEST_V4_EJECUCION_DSN="host=$directorio_socket port=5432 user=vec_v4_ejecucion_prueba password=$clave_ejecucion dbname=$base sslmode=disable"

if [[ "${VEC_POSTGRES_PRUEBA_SOLO_SQL:-0}" != "1" ]]; then
    (cd "$raiz" && go test \
        ./internal/vec/adapters/postgres/confianzadocumental \
        -run '^TestIntegracionEjecucionDocumentalV4PostgreSQLReal$' -count=1)
fi

aplicar deploy/postgresql/ejecucion_documental_v4/migraciones/000002_registro_autoridad_objeto_esperado_v1.up.sql
prueba_autoridad=deploy/postgresql/ejecucion_documental_v4/pruebas_sql/registro_autoridad_objeto_esperado_v1.sql

if [[ "${VEC_POSTGRES_PRUEBA_SOLO_SQL:-0}" == "1" ]]; then
    aplicar "$prueba_autoridad"
else
    psql_admin --set exigir_autoridad=1 --set solo_registro=1 \
        --set retener_registro=1 < "$raiz/$prueba_autoridad" &
    pid_registro_uno=$!
    registro_retenido=false
    for _ in $(seq 1 50); do
        estado=$(psql_admin --tuples-only --no-align --command \
            "SELECT count(*) FROM pg_catalog.pg_stat_activity WHERE pid <> pg_backend_pid() AND state = 'active' AND query LIKE 'SELECT pg_sleep(3)%'")
        if [[ "$estado" == "1" ]]; then
            registro_retenido=true
            break
        fi
        sleep 0.1
    done
    if [[ "$registro_retenido" != true ]]; then
        wait "$pid_registro_uno" || true
        echo "no se observo el registro de autoridad retenido" >&2
        exit 1
    fi
    psql_admin --set exigir_autoridad=1 --set solo_registro=1 \
        < "$raiz/$prueba_autoridad"
    wait "$pid_registro_uno"
    psql_admin --set exigir_autoridad=1 < "$raiz/$prueba_autoridad"
fi

# Ninguna identidad existente puede fabricar el nuevo registro ni leerlo.
for identidad in \
    "vec_v4_fuente_prueba:$clave_fuente:fuente" \
    "vec_v4_registro_prueba:$clave_registro:registro" \
    "vec_v4_emisor_prueba:$clave_emisor:emisor" \
    "vec_v4_ejecucion_prueba:$clave_ejecucion:ejecutor"
do
    IFS=: read -r usuario clave etiqueta <<<"$identidad"
    exigir_rechazo_login "$usuario" "$clave" \
        "SELECT * FROM vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(0,convert_to('{}','UTF8'))" \
        "$etiqueta fabrico una autoridad de objeto"
    exigir_rechazo_login "$usuario" "$clave" \
        "SELECT * FROM vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1" \
        "$etiqueta leyo el registro de autoridad"
done

historia=$(psql_admin --tuples-only --no-align --command \
    "SELECT count(*) FROM vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1")
down_autoridad=deploy/postgresql/ejecucion_documental_v4/migraciones/000002_registro_autoridad_objeto_esperado_v1.down.sql
if [[ "$historia" != "0" ]]; then
    if aplicar "$down_autoridad" >/dev/null 2>&1; then
        echo "el down de autoridad destruyo historia sin limpieza" >&2
        exit 1
    fi
    docker exec --interactive \
        --env PGPASSWORD="$clave_admin" \
        --env PGOPTIONS="-c vec.limpiar_registro_autoridad_objeto_esperado_v1_prueba=LIMPIAR_REGISTRO_AUTORIDAD_OBJETO_ESPERADO_V1_PRUEBA" \
        "$contenedor_id" psql --no-psqlrc --quiet --set ON_ERROR_STOP=1 \
        --host "$socket_contenedor" \
        --username postgres --dbname "$base" < "$raiz/$down_autoridad"
else
    aplicar "$down_autoridad"
fi

# Reinstalacion limpia y retirada final sin historia.
aplicar deploy/postgresql/ejecucion_documental_v4/migraciones/000002_registro_autoridad_objeto_esperado_v1.up.sql
psql_admin <<'SQL'
DO $reinstalada$
BEGIN
    IF to_regprocedure(
        'vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(numeric,bytea)'
    ) IS NULL OR EXISTS (
        SELECT 1
          FROM vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1
    ) THEN
        RAISE EXCEPTION 'la reinstalacion no quedo limpia';
    END IF;
END
$reinstalada$;
SQL
aplicar "$down_autoridad"

# Incluso en modo solo SQL se conserva evidencia para probar que un rechazo de
# 000001.down no pierde material ni muta el esquema.
psql_admin <<'SQL'
DO $marca_baseline$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM vec_ejecucion_documental_v4.atestacion_pdp)
       AND NOT EXISTS (
           SELECT 1 FROM vec_ejecucion_documental_v4.orden_generacion_documental
       ) AND NOT EXISTS (SELECT 1 FROM vec_ejecucion_documental_v4.auditoria) THEN
        UPDATE vec_ejecucion_documental_v4.control_cadena_auditoria
           SET ultima_secuencia = 1,
               ultima_huella_sha256 = repeat('1', 64)
         WHERE control_id = true;
    END IF;
END
$marca_baseline$;
SQL
if aplicar "$down_v4" >/dev/null 2>&1; then
    echo "el down V4 destruyo evidencia sin opt-in explicito" >&2
    exit 1
fi
psql_admin <<'SQL'
DO $evidencia_conservada$
BEGIN
    IF to_regnamespace('vec_ejecucion_documental_v4') IS NULL
       OR NOT EXISTS (
           SELECT 1
             FROM vec_ejecucion_documental_v4.control_cadena_auditoria
            WHERE ultima_secuencia > 0
       ) THEN
        RAISE EXCEPTION 'el down fallido no conservo esquema y evidencia';
    END IF;
END
$evidencia_conservada$;
SQL
docker exec --interactive \
    --env PGPASSWORD="$clave_admin" \
    --env PGOPTIONS="-c vec.confirmar_destruccion_ejecucion_documental_v4=DESTRUIR_EVIDENCIA_V4_IRREVERSIBLE" \
    "$contenedor_id" psql --no-psqlrc --quiet --set ON_ERROR_STOP=1 \
    --host "$socket_contenedor" \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.down.sql"
aplicar deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000003_revalidacion_ejecucion_documental_v4.down.sql

psql_admin <<'SQL'
REVOKE vec_autorizacion_fuente FROM vec_v4_fuente_prueba;
REVOKE vec_autorizacion_registro FROM vec_v4_registro_prueba;
REVOKE vec_ejecucion_documental_v4_emisor_capacidad
    FROM vec_v4_emisor_prueba;
REVOKE vec_ejecucion_documental_v4_ejecutor_atestado
    FROM vec_v4_ejecucion_prueba;
DROP ROLE vec_v4_fuente_prueba;
DROP ROLE vec_v4_registro_prueba;
DROP ROLE vec_v4_emisor_prueba;
DROP ROLE vec_v4_ejecucion_prueba;
SQL
helper="$raiz/deploy/postgresql/ejecucion_documental_v4/probar_baseline_000001.sh"
if [[ ! -x "$helper" ]]; then
    echo "falta el helper ejecutable del baseline 000001" >&2
    exit 1
fi
"$helper" "$contenedor" "$base" "$raiz"

psql_admin <<'SQL'
DO $retirada_final$
BEGIN
    IF to_regnamespace('vec_ejecucion_documental_v4') IS NOT NULL
       OR to_regnamespace('vec_ejecucion_documental_v4_guardia') IS NOT NULL
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname LIKE 'vec_ejecucion_documental_v4_%'
       ) THEN
        RAISE EXCEPTION 'la retirada final V4 dejo residuos';
    END IF;
END
$retirada_final$;
SQL

echo "integracion autoridad objeto esperado V1/PostgreSQL 18.4: correcta"
