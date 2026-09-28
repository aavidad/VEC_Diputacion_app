#!/usr/bin/env bash
set -euo pipefail

# PostgreSQL 18 efímero: no toca ninguna base conservada.
raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-ct136-auditoria-${PPID}-${RANDOM}"
base="ct136_auditoria"
vacia="ct136_auditoria_vacia"
datos="/dev/shm/${contenedor}"

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
    docker run --rm -v /dev/shm:/limpiar --entrypoint rm "$imagen" -rf "/limpiar/$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT
mkdir -p "$datos"

docker run --detach --rm --network none --name "$contenedor" \
    -e POSTGRES_HOST_AUTH_METHOD=trust \
    -v "$datos:/var/lib/postgresql" \
    -v "$raiz:/repo:ro" \
    "$imagen" >/dev/null

for _intento in $(seq 1 60); do
    if docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then
        break
    fi
    sleep 1
done
if ! docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then
    docker logs "$contenedor" >&2 || true
    exit 1
fi

psql_super() {
    docker exec -i "$contenedor" psql -X -v ON_ERROR_STOP=1 -U postgres -d "$1" "${@:2}"
}
psql_super postgres <<'SQL'
CREATE ROLE vec_contratacion_temporal_propietario NOLOGIN;
CREATE SCHEMA vec_contratacion_temporal AUTHORIZATION vec_contratacion_temporal_propietario;
CREATE DATABASE ct136_auditoria;
SQL
psql_super "$base" <<'SQL'
CREATE SCHEMA vec_contratacion_temporal AUTHORIZATION vec_contratacion_temporal_propietario;
SQL

psql_super "$base" -f /repo/deploy/postgresql/contratacion_temporal/roles_registrador_auditoria_up.sql >/dev/null
psql_super "$base" -c 'GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_contratacion_temporal_registrador_auditoria' >/dev/null
if psql_super "$base" -f /repo/deploy/postgresql/contratacion_temporal/migraciones/000136_auditoria_frontera_auditoria_ruta_exacta.up.sql >/dev/null 2>&1; then
    echo 'CT136: la preimagen con USAGE CT ajeno fue admitida' >&2
    exit 1
fi
psql_super "$base" -c 'REVOKE USAGE ON SCHEMA vec_contratacion_temporal FROM vec_contratacion_temporal_registrador_auditoria' >/dev/null
psql_super "$base" -f /repo/deploy/postgresql/contratacion_temporal/migraciones/000136_auditoria_frontera_auditoria_ruta_exacta.up.sql >/dev/null

psql_super "$base" <<'SQL'
CREATE ROLE vec_ct136_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_registrador_auditoria TO vec_ct136_login WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SQL

psql_login() {
    docker exec -i "$contenedor" psql -X -v ON_ERROR_STOP=1 -U vec_ct136_login -d "$base" "$@"
}
psql_login -At <<'SQL' | grep -qx 't'
SET TIME ZONE 'UTC';
SELECT vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1(
    'corr_0123456789abcdef0123456789abcdef', 'autenticacion_requerida',
    'api.auditoria.ruta_exacta', '/api/vec/auditoria/opciones', NULL
);
SQL
psql_login -At <<'SQL' | grep -qx 't'
SET TIME ZONE 'UTC';
SELECT vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1(
    'corr_0123456789abcdef0123456789abcdef', 'acceso_denegado',
    'api.auditoria.ruta_exacta', '/api/vec/auditoria/consultas', 'actor_sintetico_1'
);
SQL

if psql_login -c 'SELECT * FROM vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta' >/dev/null 2>&1; then
    echo 'CT136: el LOGIN registrador no puede leer la tabla' >&2
    exit 1
fi
if psql_login -c "SELECT vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1('corr_no_disponible','acceso_denegado','api.auditoria.ruta_exacta','/api/vec/auditoria/otra',NULL)" >/dev/null 2>&1; then
    echo 'CT136: se admitio una ruta fuera de la lista cerrada' >&2
    exit 1
fi
psql_super "$base" <<'SQL'
CREATE ROLE vec_ct136_deriva NOLOGIN;
GRANT vec_ct136_deriva TO vec_contratacion_temporal_registrador_auditoria;
SQL
if psql_login -c "SELECT vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1('corr_no_disponible','acceso_denegado','api.auditoria.ruta_exacta','/api/vec/auditoria/opciones',NULL)" >/dev/null 2>&1; then
    echo 'CT136: la deriva de membresia del registrador fue admitida' >&2
    exit 1
fi
psql_super "$base" <<'SQL'
REVOKE vec_ct136_deriva FROM vec_contratacion_temporal_registrador_auditoria;
DROP ROLE vec_ct136_deriva;
SQL
psql_super "$base" -c 'GRANT SELECT (actor_ref) ON vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta TO vec_contratacion_temporal_registrador_auditoria' >/dev/null
if psql_login -c "SELECT vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1('corr_no_disponible','acceso_denegado','api.auditoria.ruta_exacta','/api/vec/auditoria/opciones',NULL)" >/dev/null 2>&1; then
    echo 'CT136: el privilegio por columna del registrador fue admitido' >&2
    exit 1
fi
psql_super "$base" -c 'REVOKE SELECT (actor_ref) ON vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta FROM vec_contratacion_temporal_registrador_auditoria' >/dev/null
psql_super "$base" -At <<'SQL' | grep -qx '2'
SELECT count(*) FROM vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta;
SQL

# El DOWN conserva la historia: debe rechazarlo sin destruir nada.
if psql_super "$base" -f /repo/deploy/postgresql/contratacion_temporal/migraciones/000136_auditoria_frontera_auditoria_ruta_exacta.down.sql >/dev/null 2>&1; then
    echo 'CT136: el DOWN retiro historia de auditoria' >&2
    exit 1
fi
psql_super "$base" -At <<'SQL' | grep -qx '2'
SELECT count(*) FROM vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta;
SQL

# En otra base vacía se comprueba el orden de retirada sin usar historia conservada.
psql_super postgres -c "CREATE DATABASE ${vacia}" >/dev/null
psql_super "$vacia" <<'SQL'
CREATE SCHEMA vec_contratacion_temporal AUTHORIZATION vec_contratacion_temporal_propietario;
SQL
psql_super "$vacia" -f /repo/deploy/postgresql/contratacion_temporal/migraciones/000136_auditoria_frontera_auditoria_ruta_exacta.up.sql >/dev/null
psql_super "$vacia" -f /repo/deploy/postgresql/contratacion_temporal/migraciones/000136_auditoria_frontera_auditoria_ruta_exacta.down.sql >/dev/null
psql_super "$vacia" -At <<'SQL' | grep -qx 't'
SELECT pg_catalog.to_regclass(
    'vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta'
) IS NULL;
SQL

echo 'CT136 PG18: OK'
