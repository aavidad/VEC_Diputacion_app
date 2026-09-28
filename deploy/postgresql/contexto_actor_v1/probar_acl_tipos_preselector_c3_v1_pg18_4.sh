#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-acl-tipos-c3-${USER:-usuario}-$$"
clave=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
tmp=$(mktemp -d)
limpiar() { docker rm -f "$contenedor" >/dev/null 2>&1 || true; rm -rf -- "$tmp"; }
trap limpiar EXIT INT TERM

docker run -d --rm --network none --name "$contenedor" \
  -e POSTGRES_PASSWORD="$clave" "$imagen" >/dev/null
listo=false
for _ in $(seq 1 120); do
  if [[ $(docker exec "$contenedor" sh -c 'tr -d "\n" </proc/1/comm' 2>/dev/null || true) == postgres ]] \
      && docker exec "$contenedor" pg_isready -q -U postgres -d postgres >/dev/null 2>&1 \
      && docker exec "$contenedor" psql -XAtq -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then
    listo=true; break
  fi
  sleep 0.25
done
[[ $listo == true ]]
psql_archivo() {
  docker exec -i "$contenedor" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d postgres < "$raiz/$1"
}
psql_sql() {
  docker exec -i "$contenedor" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d postgres
}
valor() {
  docker exec "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"
}
[[ $(valor 'SHOW server_version_num') == 180004 ]]

# Preimagen real de las migraciones AD4 y B1 sobre datos sintéticos vacíos.
psql_archivo deploy/postgresql/autorizacion/roles_up.sql
psql_archivo deploy/postgresql/autorizacion/roles_v2_up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql
psql_archivo deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql
psql_archivo deploy/postgresql/bolsa_llamamientos/roles_up.sql
psql_archivo deploy/postgresql/bolsa_llamamientos/migraciones_autorizacion/000001_revalidacion_llamamientos.up.sql
paquete=deploy/postgresql/contexto_actor_v1/acl_tipos_preselector_c3_v1.up.sql
puerta=deploy/postgresql/contexto_actor_v1/pruebas_sql/acl_tipos_preselector_c3_v1.sql

# El clon principal no tiene las catorce tablas B1. Incluso si el rol selector
# existe, el delta de la cadena nueva debe informar NO_APLICA sin tocar AD4.
psql_sql <<'SQL'
CREATE ROLE vec_contexto_actor_corporativo_rrhh_selector NOLOGIN;
SQL
psql_archivo "$paquete" >"$tmp/sin_b1" 2>&1
grep -Fq 'NO_APLICA C3 ACL tipos: B1 ausente (0/14 tablas y tipos)' "$tmp/sin_b1"
[[ $(valor "SELECT count(*) FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace
  CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
  WHERE n.nspname='vec_autorizacion'
    AND t.typname='decision_autorizacion_solicitud_ligada_v2'
    AND a.grantee=0 AND a.privilege_type='USAGE'") == 1 ]]
[[ $(valor "SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
  WHERE n.nspname='vec_bolsa_llamamientos' AND c.relname='bolsa_autoritativa'") == 0 ]]
psql_sql <<'SQL'
DROP ROLE vec_contexto_actor_corporativo_rrhh_selector;
SET ROLE vec_bolsa_llamamientos_propietario;
CREATE SCHEMA vec_bolsa_llamamientos AUTHORIZATION vec_bolsa_llamamientos_propietario;
CREATE TABLE vec_bolsa_llamamientos.bolsa_autoritativa (id integer PRIMARY KEY);
RESET ROLE;
SQL
if psql_archivo "$paquete" >"$tmp/b1_parcial" 2>&1; then
  echo 'paquete aceptó B1 parcial' >&2; exit 1
fi
grep -Fq 'B1 parcial: 1/14 tablas, 1/14 tipos' "$tmp/b1_parcial"
[[ $(valor "SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
  WHERE n.nspname='vec_bolsa_llamamientos' AND c.relname='bolsa_autoritativa'") == 1 ]]
psql_sql <<'SQL'
DROP SCHEMA vec_bolsa_llamamientos CASCADE;
SQL
psql_archivo deploy/postgresql/bolsa_llamamientos/migraciones/000001_almacen_llamamientos.up.sql

abiertos=$(valor "SELECT count(*) FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace
  CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
  WHERE ((n.nspname='vec_bolsa_llamamientos' AND t.typtype='c')
      OR (n.nspname='vec_autorizacion' AND t.typname='decision_autorizacion_solicitud_ligada_v2'))
    AND a.grantee=0 AND a.privilege_type='USAGE'")
[[ $abiertos == 15 ]] || { echo "preimagen esperada: 15 tipos abiertos; obtenidos $abiertos" >&2; exit 1; }

# Una concesión ajena adicional debe detener toda la transacción sin cerrar
# parcialmente los quince tipos.
psql_sql <<'SQL'
GRANT USAGE ON TYPE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2 TO vec_bolsa_llamamientos_ejecutor;
SQL
if psql_archivo "$paquete" >"$tmp/deriva" 2>&1; then
  echo 'paquete aceptó una ACL ajena' >&2; exit 1
fi
grep -Fq 'ACL previa incompatible' "$tmp/deriva"
[[ $(valor "SELECT count(*) FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace
  CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
  WHERE n.nspname='vec_bolsa_llamamientos' AND t.typtype='c' AND a.grantee=0 AND a.privilege_type='USAGE'") == 14 ]]
psql_sql <<'SQL'
REVOKE USAGE ON TYPE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2 FROM vec_bolsa_llamamientos_ejecutor;
SQL

# Renombrar al propietario conserva el OID de tablas y tipos. La preimagen
# debe exigir también que exista el rol nominal esperado.
psql_sql <<'SQL'
ALTER ROLE vec_bolsa_llamamientos_propietario RENAME TO vec_bolsa_llamamientos_propietario_renombrado;
SQL
if psql_archivo "$paquete" >"$tmp/owner" 2>&1; then
  echo 'paquete aceptó un propietario nominal ausente' >&2; exit 1
fi
grep -Fq 'tipo u owner incompatible' "$tmp/owner"
psql_sql <<'SQL'
ALTER ROLE vec_bolsa_llamamientos_propietario_renombrado RENAME TO vec_bolsa_llamamientos_propietario;
SQL

psql_archivo "$paquete"
psql_archivo "$puerta"
[[ $(valor "SELECT count(*) FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace
  CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
  WHERE ((n.nspname='vec_bolsa_llamamientos' AND t.typtype='c')
      OR (n.nspname='vec_autorizacion' AND t.typname='decision_autorizacion_solicitud_ligada_v2'))
    AND a.grantee=0 AND a.privilege_type='USAGE'") == 0 ]]
if psql_archivo "$paquete" >"$tmp/repeticion" 2>&1; then
  echo 'paquete aceptó reaplicación' >&2; exit 1
fi

# Un tipo creado después del parche vuelve a ser visible a PUBLIC: la puerta
# final debe rechazarlo hasta que su propia migración cierre esa ACL.
psql_sql <<'SQL'
SET ROLE vec_bolsa_llamamientos_propietario;
CREATE TABLE vec_bolsa_llamamientos.c3_tipo_posterior (id integer PRIMARY KEY);
RESET ROLE;
SQL
if psql_archivo "$puerta" >"$tmp/tipo_posterior" 2>&1; then
  echo 'puerta aceptó un tipo fila posterior abierto' >&2; exit 1
fi
grep -Fq 'USAGE PUBLIC recuperado' "$tmp/tipo_posterior"
psql_sql <<'SQL'
REVOKE USAGE ON TYPE vec_bolsa_llamamientos.c3_tipo_posterior FROM PUBLIC;
SQL
psql_archivo "$puerta"
echo 'OK C3 ACL tipos: NO_APLICA sin B1, parcial rechazado, AD4+B1 reales y puerta posterior cerrada'
