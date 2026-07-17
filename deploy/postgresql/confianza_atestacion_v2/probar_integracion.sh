#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-confianza-atestacion-v2-pg-${USER:-usuario}-$$"
base=vec_confianza_atestacion_v2_prueba

generar_clave() {
    local destino=$1 valor
    valor=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
    if [[ ${#valor} -ne 64 || $valor == *[!0-9a-f]* ]]; then
        echo 'no se pudo generar una clave efimera de prueba' >&2
        exit 1
    fi
    printf -v "$destino" '%s' "$valor"
}

clave_admin=
clave_lector=
clave_ajeno=
clave_privilegiado=
clave_extra=
clave_delegador=
generar_clave clave_admin
generar_clave clave_lector
generar_clave clave_ajeno
generar_clave clave_privilegiado
generar_clave clave_extra
generar_clave clave_delegador

retenedor=
limpiar() {
    if [[ -n ${retenedor:-} ]]; then
        kill "$retenedor" >/dev/null 2>&1 || true
        wait "$retenedor" >/dev/null 2>&1 || true
    fi
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

docker run --detach --rm --name "$contenedor" \
    --publish 127.0.0.1::5432 \
    --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave_admin" \
    "$imagen" >/dev/null
for _ in $(seq 1 60); do
    if docker exec "$contenedor" pg_isready \
        --username postgres --dbname "$base" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done
docker exec "$contenedor" pg_isready \
    --username postgres --dbname "$base" >/dev/null

psql_archivo() {
    docker exec --interactive "$contenedor" psql -X --quiet \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        < "$raiz/$1"
}

psql_archivo_optin_down() {
    docker exec --interactive \
        --env PGOPTIONS='-c vec.confirmar_destruccion_confianza_atestacion_v2=DESTRUIR_CONFIANZA_V2_IRREVERSIBLE' \
        "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
        --username postgres --dbname "$base" \
        < "$raiz/deploy/postgresql/confianza_atestacion_v2/migraciones/000001_catalogo_confianza_v2.down.sql"
}

consulta_login() {
    local usuario=$1 clave=$2 consulta=$3
    docker exec --env PGPASSWORD="$clave" "$contenedor" \
        psql -X --no-align --tuples-only --set ON_ERROR_STOP=1 \
        --host 127.0.0.1 --username "$usuario" --dbname "$base" \
        --command "$consulta"
}

rechazar_sql() {
    local usuario=$1 clave=$2 consulta=$3 patron=$4 descripcion=$5
    local salida estado
    set +e
    salida=$(consulta_login "$usuario" "$clave" "$consulta" 2>&1)
    estado=$?
    set -e
    if [[ $estado -eq 0 || $salida != *"$patron"* ]]; then
        echo "fallo cerrado no demostrado: $descripcion" >&2
        exit 1
    fi
}

crear_logins_prueba() {
    docker exec --interactive \
        --env CLAVE_LECTOR="$clave_lector" \
        --env CLAVE_AJENO="$clave_ajeno" \
        --env CLAVE_PRIVILEGIADO="$clave_privilegiado" \
        --env CLAVE_EXTRA="$clave_extra" \
        --env CLAVE_DELEGADOR="$clave_delegador" \
        "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
        --username postgres --dbname "$base" <<'SQL'
\getenv clave_lector CLAVE_LECTOR
\getenv clave_ajeno CLAVE_AJENO
\getenv clave_privilegiado CLAVE_PRIVILEGIADO
\getenv clave_extra CLAVE_EXTRA
\getenv clave_delegador CLAVE_DELEGADOR
CREATE ROLE vec_confianza_v2_lector_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_lector';
CREATE ROLE vec_confianza_v2_ajeno_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_ajeno';
CREATE ROLE vec_confianza_v2_privilegiado_prueba LOGIN NOSUPERUSER NOCREATEDB
    CREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_privilegiado';
CREATE ROLE vec_confianza_v2_grupo_extra_prueba NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_confianza_v2_extra_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_extra';
CREATE ROLE vec_confianza_v2_delegador_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_delegador';
GRANT vec_confianza_atestacion_v2_lector_autoridad
    TO vec_confianza_v2_lector_prueba;
GRANT vec_confianza_atestacion_v2_lector_autoridad
    TO vec_confianza_v2_privilegiado_prueba;
GRANT vec_confianza_atestacion_v2_lector_autoridad,
      vec_confianza_v2_grupo_extra_prueba
    TO vec_confianza_v2_extra_prueba;
GRANT vec_confianza_atestacion_v2_lector_autoridad
    TO vec_confianza_v2_delegador_prueba WITH ADMIN OPTION;
SQL
}

retirar_logins_prueba() {
    docker exec --interactive "$contenedor" psql -X --quiet \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
REVOKE vec_confianza_atestacion_v2_lector_autoridad
    FROM vec_confianza_v2_lector_prueba;
REVOKE vec_confianza_atestacion_v2_lector_autoridad
    FROM vec_confianza_v2_privilegiado_prueba;
REVOKE vec_confianza_atestacion_v2_lector_autoridad,
       vec_confianza_v2_grupo_extra_prueba
    FROM vec_confianza_v2_extra_prueba;
REVOKE vec_confianza_atestacion_v2_lector_autoridad
    FROM vec_confianza_v2_delegador_prueba;
DROP ROLE vec_confianza_v2_lector_prueba;
DROP ROLE vec_confianza_v2_ajeno_prueba;
DROP ROLE vec_confianza_v2_privilegiado_prueba;
DROP ROLE vec_confianza_v2_extra_prueba;
DROP ROLE vec_confianza_v2_delegador_prueba;
DROP ROLE vec_confianza_v2_grupo_extra_prueba;
SQL
}

psql_archivo deploy/postgresql/autorizacion/roles_up.sql

# La presencia de cualquier privilegio de PUBLIC en la base dedicada aborta
# antes de crear roles u objetos del modulo.
docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    --command "GRANT CONNECT ON DATABASE $base TO PUBLIC"
if psql_archivo deploy/postgresql/confianza_atestacion_v2/roles_up.sql \
    >/dev/null 2>&1; then
    echo 'roles_up acepto CONNECT de PUBLIC' >&2
    exit 1
fi
roles_residuales=$(docker exec "$contenedor" psql -X --no-align \
    --tuples-only --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --command "SELECT count(*) FROM pg_catalog.pg_roles WHERE rolname LIKE 'vec_confianza_atestacion_v2_%'")
if [[ $roles_residuales != 0 ]]; then
    echo 'un bootstrap rechazado dejo roles residuales' >&2
    exit 1
fi
docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    --command "REVOKE ALL PRIVILEGES ON DATABASE $base FROM PUBLIC"

psql_archivo deploy/postgresql/confianza_atestacion_v2/roles_up.sql
psql_archivo deploy/postgresql/confianza_atestacion_v2/migraciones/000001_catalogo_confianza_v2.up.sql
psql_archivo deploy/postgresql/confianza_atestacion_v2/pruebas_sql/acl_y_contrato.sql
psql_archivo deploy/postgresql/confianza_atestacion_v2/pruebas_sql/integracion_catalogo.sql
crear_logins_prueba

numero=$(consulta_login vec_confianza_v2_lector_prueba "$clave_lector" \
    'SELECT count(*) FROM vec_confianza_atestacion_v2.obtener_confianza_actual()')
if [[ $numero != 2 ]]; then
    echo 'la funcion no devolvio el conjunto exacto de dos raices' >&2
    exit 1
fi
estados=$(consulta_login vec_confianza_v2_lector_prueba "$clave_lector" \
    "SELECT string_agg(raiz_estado, ',' ORDER BY clave_id) FROM vec_confianza_atestacion_v2.obtener_confianza_actual()")
if [[ $estados != activa,activa ]]; then
    echo 'el conjunto inicial de raices no esta activo' >&2
    exit 1
fi

rechazar_sql vec_confianza_v2_lector_prueba "$clave_lector" \
    'SELECT * FROM vec_confianza_atestacion_v2.acto_gobierno' \
    'permission denied' 'el lector accedio directamente a tablas'
rechazar_sql vec_confianza_v2_lector_prueba "$clave_lector" \
    "SELECT vec_confianza_atestacion_v2.calcular_huella_configuracion('x')" \
    'permission denied' 'el lector ejecuto una funcion auxiliar'
rechazar_sql vec_confianza_v2_privilegiado_prueba "$clave_privilegiado" \
    'SELECT count(*) FROM vec_confianza_atestacion_v2.obtener_confianza_actual()' \
    'identidad de lectura de confianza V2 rechazada' \
    'un LOGIN con CREATEROLE fue aceptado'
rechazar_sql vec_confianza_v2_extra_prueba "$clave_extra" \
    'SELECT count(*) FROM vec_confianza_atestacion_v2.obtener_confianza_actual()' \
    'identidad de lectura de confianza V2 rechazada' \
    'un LOGIN con membresia adicional fue aceptado'
rechazar_sql vec_confianza_v2_delegador_prueba "$clave_delegador" \
    'SELECT count(*) FROM vec_confianza_atestacion_v2.obtener_confianza_actual()' \
    'identidad de lectura de confianza V2 rechazada' \
    'un LOGIN con ADMIN OPTION fue aceptado'
rechazar_sql vec_confianza_v2_ajeno_prueba "$clave_ajeno" \
    'SELECT 1' 'permission denied for database' \
    'un rol ajeno heredo CONNECT de PUBLIC'

puerto=$(docker port "$contenedor" 5432/tcp | head -n 1)
puerto=${puerto##*:}
dsn="postgres://vec_confianza_v2_lector_prueba:${clave_lector}@127.0.0.1:${puerto}/${base}?sslmode=disable"
VEC_CONFIANZA_ATESTACION_V2_POSTGRES_DSN="$dsn" \
    go test ./internal/vec/adapters/postgres/confianzaatestacionv2 \
        -run '^TestIntegracionPostgreSQLCargaConfianzaV2Real$' -count=1
unset dsn

psql_archivo deploy/postgresql/confianza_atestacion_v2/pruebas_sql/revocar_raiz.sql
estados=$(consulta_login vec_confianza_v2_lector_prueba "$clave_lector" \
    "SELECT string_agg(raiz_estado, ',' ORDER BY clave_id) FROM vec_confianza_atestacion_v2.obtener_confianza_actual()")
if [[ $estados != revocada,activa ]]; then
    echo 'la funcion filtro una raiz revocada o perdio la raiz activa' >&2
    exit 1
fi
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/confianza_atestacion_v2/pruebas_sql/revocar_configuracion_con_bloqueo.sql" \
    >/dev/null 2>&1 &
retenedor=$!
bloqueo_detectado=false
for _ in $(seq 1 100); do
    marca=$(docker exec "$contenedor" psql -X --no-align --tuples-only \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        --command 'SELECT pg_catalog.pg_try_advisory_lock(721702027)')
    if [[ $marca == f ]]; then
        bloqueo_detectado=true
        break
    fi
    sleep 0.05
done
if [[ $bloqueo_detectado != true ]]; then
    echo 'no se confirmo el lock de revocacion concurrente' >&2
    exit 1
fi
numero=$(consulta_login vec_confianza_v2_lector_prueba "$clave_lector" \
    'SELECT count(*) FROM vec_confianza_atestacion_v2.obtener_confianza_actual()')
wait "$retenedor"
retenedor=
if [[ $numero != 0 ]]; then
    echo 'una revocacion confirmada durante la espera no se observo fresca' >&2
    exit 1
fi

# El down es destructivo incluso con fixtures y exige opt-in incondicional.
if psql_archivo \
    deploy/postgresql/confianza_atestacion_v2/migraciones/000001_catalogo_confianza_v2.down.sql \
    >/dev/null 2>&1; then
    echo 'el down destructivo se ejecuto sin confirmacion' >&2
    exit 1
fi
psql_archivo_optin_down
# Con una membresia LOGIN viva, roles_down debe abortar antes de mutar.
if psql_archivo deploy/postgresql/confianza_atestacion_v2/roles_down.sql \
    >/dev/null 2>&1; then
    echo 'roles_down retiro autoridad con miembros LOGIN vivos' >&2
    exit 1
fi
retirar_logins_prueba
psql_archivo deploy/postgresql/confianza_atestacion_v2/roles_down.sql

# Segundo ciclo: la consulta empieza con una configuracion valida, espera un
# lock exclusivo hasta que caduca y debe devolver cero usando reloj posterior.
psql_archivo deploy/postgresql/confianza_atestacion_v2/roles_up.sql
psql_archivo deploy/postgresql/confianza_atestacion_v2/migraciones/000001_catalogo_confianza_v2.up.sql
psql_archivo deploy/postgresql/confianza_atestacion_v2/pruebas_sql/integracion_catalogo.sql
docker exec --interactive --env CLAVE_LECTOR="$clave_lector" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
\getenv clave_lector CLAVE_LECTOR
CREATE ROLE vec_confianza_v2_lector_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_lector';
GRANT vec_confianza_atestacion_v2_lector_autoridad
    TO vec_confianza_v2_lector_prueba;
SQL
psql_archivo \
    deploy/postgresql/confianza_atestacion_v2/pruebas_sql/preparar_expiracion_bajo_bloqueo.sql
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/confianza_atestacion_v2/pruebas_sql/retener_gobierno_hasta_expiracion.sql" \
    >/dev/null 2>&1 &
retenedor=$!

bloqueo_detectado=false
for _ in $(seq 1 100); do
    marca=$(docker exec "$contenedor" psql -X --no-align --tuples-only \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        --command 'SELECT pg_catalog.pg_try_advisory_lock(721702026)')
    if [[ $marca == f ]]; then
        bloqueo_detectado=true
        break
    fi
    sleep 0.05
done
if [[ $bloqueo_detectado != true ]]; then
    echo 'no se confirmo el lock exclusivo de gobierno' >&2
    exit 1
fi
numero=$(consulta_login vec_confianza_v2_lector_prueba "$clave_lector" \
    'SELECT count(*) FROM vec_confianza_atestacion_v2.obtener_confianza_actual()')
wait "$retenedor"
retenedor=
if [[ $numero != 0 ]]; then
    echo 'se acepto una configuracion caducada durante la espera' >&2
    exit 1
fi

psql_archivo_optin_down
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
REVOKE vec_confianza_atestacion_v2_lector_autoridad
    FROM vec_confianza_v2_lector_prueba;
DROP ROLE vec_confianza_v2_lector_prueba;
SQL
psql_archivo deploy/postgresql/confianza_atestacion_v2/roles_down.sql

restos=$(docker exec "$contenedor" psql -X --no-align --tuples-only \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --command "SELECT count(*) FROM pg_catalog.pg_roles WHERE rolname LIKE 'vec_confianza_atestacion_v2_%'")
if [[ $restos != 0 ]]; then
    echo 'el ciclo descendente dejo roles residuales' >&2
    exit 1
fi

echo 'integracion PostgreSQL de confianza de atestacion V2: OK'
