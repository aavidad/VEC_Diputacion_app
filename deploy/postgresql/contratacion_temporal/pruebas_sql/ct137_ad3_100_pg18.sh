#!/usr/bin/env bash
set -euo pipefail
raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-ct137-ad3100-${PPID}-${RANDOM}"
datos="/dev/shm/${contenedor}"
limpiar() {
 docker rm -f "$contenedor" >/dev/null 2>&1 || true
 docker run --rm -v /dev/shm:/limpiar --entrypoint rm "$imagen" -rf "/limpiar/$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT
mkdir -p "$datos"
docker run --detach --rm --network none --name "$contenedor" \
 -e POSTGRES_HOST_AUTH_METHOD=trust -v "$datos:/var/lib/postgresql" -v "$raiz:/repo:ro" "$imagen" >/dev/null
for _ in $(seq 1 60); do
 if docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then break; fi
 sleep 1
done
docker exec "$contenedor" pg_isready -q -U postgres -d postgres
psql_super() { docker exec -i "$contenedor" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
psql_login() { docker exec -i "$contenedor" psql -X -v ON_ERROR_STOP=1 -U vec_ct137_login -d postgres "$@"; }
ct=/repo/deploy/postgresql/contratacion_temporal
ad3=/repo/deploy/postgresql/autorizacion_atestada_v3
psql_super -f "$ct/pruebas_sql/ct137_fixture_pg18.sql" >/dev/null
# Roundtrip de definición sin dejar efectos, antes de cada UP efectivo.
for archivo in "$ad3/migraciones/000100_documental_tres_ambitos_ct.up.sql"; do
 { printf 'BEGIN;\n'; sed '/^BEGIN;$/d;/^COMMIT;$/d' "${raiz}${archivo#/repo}"; printf 'ROLLBACK;\n'; } | psql_super >/dev/null
done
psql_super -f "$ad3/migraciones/000100_documental_tres_ambitos_ct.up.sql" >/dev/null
{ printf 'BEGIN;\n'; sed '/^BEGIN;$/d;/^COMMIT;$/d' "${raiz}${ct#/repo}/migraciones/000137_documental_tres_ambitos.up.sql"; printf 'ROLLBACK;\n'; } | psql_super >/dev/null
psql_super -f "$ct/migraciones/000137_documental_tres_ambitos.up.sql" >/dev/null
if psql_super -f "$ad3/migraciones/000100_documental_tres_ambitos_ct.up.sql" >/dev/null 2>&1; then
 echo 'AD3-100 admitió doble UP' >&2; exit 1
fi
if psql_super -f "$ct/migraciones/000137_documental_tres_ambitos.up.sql" >/dev/null 2>&1; then
 echo 'CT-137 admitió doble UP' >&2; exit 1
fi
psql_super -f "$ad3/pruebas_sql/ad3_100_acl_pg18.sql" >/dev/null
psql_login -f "$ct/pruebas_sql/ct137_contrato_pg18.sql" >/dev/null
docker restart "$contenedor" >/dev/null
for _ in $(seq 1 60); do
 if docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then break; fi
 sleep 1
done
docker exec "$contenedor" pg_isready -q -U postgres -d postgres
psql_super -f "$ad3/pruebas_sql/ad3_100_acl_pg18.sql" >/dev/null
psql_login -f "$ct/pruebas_sql/ct137_contrato_pg18.sql" >/dev/null
printf 'CT137/AD3-100: rollback, UP, doble UP denegado, ACL, lectura, negativos y reinicio OK (PostgreSQL 18 sintético)\n'
