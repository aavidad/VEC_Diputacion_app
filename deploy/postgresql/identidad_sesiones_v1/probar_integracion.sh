#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-identidad-sesiones-pg-${USER:-usuario}-$$"
base=vec_identidad_sesiones_prueba

generar_clave() {
    local destino=$1 valor
    valor=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
    if [[ ${#valor} -ne 64 || $valor == *[!0-9a-f]* ]]; then
        echo 'no se pudo generar una clave efimera' >&2
        exit 1
    fi
    printf -v "$destino" '%s' "$valor"
}

clave_admin=
clave_provisionador=
clave_registrador=
clave_revalidador=
clave_revocador=
clave_mixto=
generar_clave clave_admin
generar_clave clave_provisionador
generar_clave clave_registrador
generar_clave clave_revalidador
generar_clave clave_revocador
generar_clave clave_mixto

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

docker run --detach --rm --name "$contenedor" \
    --publish 127.0.0.1::5432 \
    --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave_admin" \
    "$imagen" >/dev/null

listo=false
for _ in $(seq 1 100); do
    if docker exec "$contenedor" psql -X --username postgres \
        --dbname "$base" --command 'SELECT 1' >/dev/null 2>&1; then
        listo=true
        break
    fi
    sleep 0.2
done
if [[ $listo != true ]]; then
    echo 'PostgreSQL 18 no quedo disponible' >&2
    exit 1
fi

psql_archivo() {
    docker exec --interactive "$contenedor" psql -X --quiet \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        < "$raiz/$1"
}

psql_runtime() {
    local usuario=$1 clave=$2 consulta=$3
    docker exec --env PGPASSWORD="$clave" "$contenedor" \
        psql -X --quiet --no-align --tuples-only --set ON_ERROR_STOP=1 \
        --host 127.0.0.1 --username "$usuario" --dbname "$base" \
        --command "$consulta"
}

rechazar_runtime() {
    local usuario=$1 clave=$2 consulta=$3 descripcion=$4
    if psql_runtime "$usuario" "$clave" "$consulta" >/dev/null 2>&1; then
        echo "ACL invalida: $descripcion" >&2
        exit 1
    fi
}

docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    --command 'CREATE EXTENSION pgcrypto' >/dev/null
psql_archivo deploy/postgresql/autorizacion/roles_up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql
psql_archivo deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
psql_archivo deploy/postgresql/identidad_sesiones_v1/roles_up.sql

# La FK compuesta de consumo exige que la migracion del esquema propietario
# cree primero la UNIQUE nominal. El orden inverso debe fallar y revertirse.
if psql_archivo \
    deploy/postgresql/identidad_sesiones_v1/migraciones/000001_registro_base_v1.up.sql \
    >/dev/null 2>&1; then
    echo 'el orden inverso de migraciones fue aceptado' >&2
    exit 1
fi
if docker exec "$contenedor" psql -X --quiet --tuples-only \
    --username postgres --dbname "$base" \
    --command "SELECT to_regnamespace('vec_identidad_sesiones_v1') IS NULL" \
    | tr -d '[:space:]' | grep -qx t; then
    :
else
    echo 'el orden inverso dejo objetos parciales' >&2
    exit 1
fi

instalar_identidad() {
    psql_archivo deploy/postgresql/identidad_sesiones_v1/migraciones_autorizacion/000001_capacidad_tablas_v1.up.sql
    psql_archivo deploy/postgresql/identidad_sesiones_v1/migraciones/000001_registro_base_v1.up.sql
    psql_archivo deploy/postgresql/identidad_sesiones_v1/migraciones/000002_operaciones_v1.up.sql
}

instalar_identidad
psql_archivo deploy/postgresql/identidad_sesiones_v1/pruebas_sql/acl_y_modelo.sql

# Ciclo limpio de down antes de crear historia inmutable.
psql_archivo deploy/postgresql/identidad_sesiones_v1/migraciones/000002_operaciones_v1.down.sql
psql_archivo deploy/postgresql/identidad_sesiones_v1/migraciones/000001_registro_base_v1.down.sql
psql_archivo deploy/postgresql/identidad_sesiones_v1/migraciones_autorizacion/000001_capacidad_tablas_v1.down.sql
psql_archivo deploy/postgresql/identidad_sesiones_v1/roles_down.sql

psql_archivo deploy/postgresql/identidad_sesiones_v1/roles_up.sql
instalar_identidad

docker exec --interactive \
    --env CLAVE_PROVISIONADOR="$clave_provisionador" \
    --env CLAVE_REGISTRADOR="$clave_registrador" \
    --env CLAVE_REVALIDADOR="$clave_revalidador" \
    --env CLAVE_REVOCADOR="$clave_revocador" \
    --env CLAVE_MIXTO="$clave_mixto" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
\getenv clave_provisionador CLAVE_PROVISIONADOR
\getenv clave_registrador CLAVE_REGISTRADOR
\getenv clave_revalidador CLAVE_REVALIDADOR
\getenv clave_revocador CLAVE_REVOCADOR
\getenv clave_mixto CLAVE_MIXTO
CREATE ROLE vec_identidad_provisionador_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_provisionador';
CREATE ROLE vec_identidad_registrador_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_registrador';
CREATE ROLE vec_identidad_revalidador_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_revalidador';
CREATE ROLE vec_identidad_revocador_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_revocador';
CREATE ROLE vec_identidad_mixto_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_mixto';
GRANT vec_identidad_sesiones_v1_provisionador
    TO vec_identidad_provisionador_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_identidad_sesiones_v1_registrador
    TO vec_identidad_registrador_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_identidad_sesiones_v1_revalidador
    TO vec_identidad_revalidador_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_identidad_sesiones_v1_revocador
    TO vec_identidad_revocador_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_identidad_sesiones_v1_registrador
    TO vec_identidad_mixto_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_identidad_sesiones_v1_revalidador
    TO vec_identidad_mixto_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL

for identidad in \
    "vec_identidad_provisionador_prueba:$clave_provisionador:vec_identidad_sesiones_v1_provisionador" \
    "vec_identidad_registrador_prueba:$clave_registrador:vec_identidad_sesiones_v1_registrador" \
    "vec_identidad_revalidador_prueba:$clave_revalidador:vec_identidad_sesiones_v1_revalidador" \
    "vec_identidad_revocador_prueba:$clave_revocador:vec_identidad_sesiones_v1_revocador"; do
    usuario=${identidad%%:*}
    resto=${identidad#*:}
    clave=${resto%%:*}
    grupo=${resto#*:}
    if [[ $(psql_runtime "$usuario" "$clave" \
        'SELECT session_user = current_user') != t ]]; then
        echo "la capacidad $usuario suplanta mediante SET ROLE" >&2
        exit 1
    fi
    rechazar_runtime "$usuario" "$clave" "SET ROLE $grupo" \
        "$usuario puede activar SET ROLE"
    rechazar_runtime "$usuario" "$clave" \
        'SELECT * FROM vec_identidad_sesiones_v1.consumo_asercion' \
        "$usuario puede leer consumo_asercion"
done

rechazar_runtime vec_identidad_mixto_prueba "$clave_mixto" \
    'SET ROLE vec_identidad_sesiones_v1_registrador' \
    'el LOGIN mixto puede activar el rol registrador'
rechazar_runtime vec_identidad_mixto_prueba "$clave_mixto" \
    'SET ROLE vec_identidad_sesiones_v1_revalidador' \
    'el LOGIN mixto puede activar el rol revalidador'

rechazar_runtime vec_identidad_registrador_prueba "$clave_registrador" \
    "SELECT vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL)" \
    'el registrador puede revalidar'
rechazar_runtime vec_identidad_revalidador_prueba "$clave_revalidador" \
    "SELECT * FROM vec_identidad_sesiones_v1.registrar_sesion_v1(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL)" \
    'el revalidador puede registrar'
rechazar_runtime vec_identidad_provisionador_prueba "$clave_provisionador" \
    "SELECT * FROM vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL)" \
    'el provisionador puede reconciliar sesiones'

psql_archivo deploy/postgresql/identidad_sesiones_v1/pruebas_sql/acl_y_modelo.sql

direccion=$(docker port "$contenedor" 5432/tcp | head -n 1)
host=${direccion%:*}
puerto=${direccion##*:}
if [[ -z $host || -z $puerto || $puerto == *[!0-9]* ]]; then
    echo 'no se pudo resolver el puerto PostgreSQL efimero' >&2
    exit 1
fi
export VEC_POSTGRES_TEST_IDENTIDAD_REGISTRO_DSN="postgres://vec_identidad_registrador_prueba:${clave_registrador}@${host}:${puerto}/${base}?sslmode=disable"
export VEC_POSTGRES_TEST_IDENTIDAD_REVALIDACION_DSN="postgres://vec_identidad_revalidador_prueba:${clave_revalidador}@${host}:${puerto}/${base}?sslmode=disable"
export VEC_POSTGRES_TEST_IDENTIDAD_PROVISIONADOR_DSN="postgres://vec_identidad_provisionador_prueba:${clave_provisionador}@${host}:${puerto}/${base}?sslmode=disable"
export VEC_POSTGRES_TEST_IDENTIDAD_MIXTO_DSN="postgres://vec_identidad_mixto_prueba:${clave_mixto}@${host}:${puerto}/${base}?sslmode=disable"
(
    cd "$raiz"
    go test -count=1 -run '^TestIntegracionRegistroSesionesPostgreSQL18$' \
        ./internal/vec/adapters/httpseguridad/postgres
)
unset VEC_POSTGRES_TEST_IDENTIDAD_REGISTRO_DSN
unset VEC_POSTGRES_TEST_IDENTIDAD_REVALIDACION_DSN
unset VEC_POSTGRES_TEST_IDENTIDAD_PROVISIONADOR_DSN
unset VEC_POSTGRES_TEST_IDENTIDAD_MIXTO_DSN

psql_archivo deploy/postgresql/identidad_sesiones_v1/pruebas_sql/preparar_frontera_frescura.sql
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/identidad_sesiones_v1/pruebas_sql/retener_frontera_frescura.sql" \
    >/dev/null 2>&1 &
retenedor=$!
bloqueo_detectado=false
for _ in $(seq 1 300); do
    resultado=$(docker exec "$contenedor" psql -X --no-align --tuples-only \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        --command 'SELECT pg_try_advisory_lock(817263540118)')
    if [[ $resultado == f ]]; then
        bloqueo_detectado=true
        break
    fi
done
if [[ $bloqueo_detectado != true ]]; then
    wait "$retenedor" || true
    echo 'no se detecto el bloqueo de frontera de frescura' >&2
    exit 1
fi
psql_archivo deploy/postgresql/identidad_sesiones_v1/pruebas_sql/consultar_frontera_frescura.sql
wait "$retenedor"

psql_archivo deploy/postgresql/identidad_sesiones_v1/pruebas_sql/integracion_mecanica.sql

# Tras crear historia, ningun down puede retirar las APIs que la revalidan o
# revocan ni borrar cuentas, aserciones y sesiones auditables.
if psql_archivo \
    deploy/postgresql/identidad_sesiones_v1/migraciones/000002_operaciones_v1.down.sql \
    >/dev/null 2>&1; then
    echo 'el down retiro operaciones con historia de identidad' >&2
    exit 1
fi
if psql_archivo \
    deploy/postgresql/identidad_sesiones_v1/migraciones/000001_registro_base_v1.down.sql \
    >/dev/null 2>&1; then
    echo 'el down borro historia de identidad' >&2
    exit 1
fi
if ! docker exec "$contenedor" psql -X --quiet --tuples-only \
    --username postgres --dbname "$base" --command "
        SELECT count(*) = 3
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
         WHERE espacio.nspname = 'vec_identidad_sesiones_v1'
           AND funcion.proname IN (
               'registrar_sesion_v1',
               'revalidar_sesion_y_cuentas_v1',
               'revocar_sesion_v1'
           )" | tr -d '[:space:]' | grep -qx t; then
    echo 'un down fallido dejo las operaciones de identidad incompletas' >&2
    exit 1
fi

echo 'identidad_sesiones_v1: integracion PostgreSQL 18 superada'
