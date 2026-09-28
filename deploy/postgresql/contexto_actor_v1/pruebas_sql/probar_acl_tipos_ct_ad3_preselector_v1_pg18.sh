#!/usr/bin/env bash
set -euo pipefail
raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
base=deploy/postgresql/contexto_actor_v1
delta="$raiz/$base/acl_tipos_ct_ad3_preselector_v1.up.sql"
fixture="$raiz/$base/pruebas_sql/fixture_acl_ct_ad3_preselector_v1.txt"
contenedor="vec-acl-ct-ad3-${USER:-operador}-$$"
tmp=$(mktemp -d)
limpiar() { docker rm -f "$contenedor" >/dev/null 2>&1 || true; rm -rf -- "$tmp"; }
trap limpiar EXIT INT TERM
docker run -d --name "$contenedor" --network none \
  --mount "type=bind,src=$raiz,dst=/repo,readonly" \
  -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4 >/dev/null
for _ in $(seq 1 120); do
  if docker exec "$contenedor" pg_isready -q -U postgres -d postgres >/dev/null 2>&1; then break; fi
  sleep 0.25
done
docker exec "$contenedor" pg_isready -q -U postgres -d postgres
sql() { docker exec "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
archivo() { docker exec -i "$contenedor" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d postgres < "$1"; }
fallo_55000() {
  local etiqueta=$1; shift
  if "$@" >"$tmp/$etiqueta" 2>&1; then echo "Aceptó $etiqueta" >&2; exit 1; fi
  grep -q '55000' "$tmp/$etiqueta"
}
[[ $(sql 'SHOW server_version_num') == 180004 ]]
sql 'REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC; REVOKE ALL ON SCHEMA public FROM PUBLIC; CREATE EXTENSION pgcrypto WITH SCHEMA public; REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;' >/dev/null
n=0
while IFS= read -r ruta; do
  [[ $ruta == deploy/postgresql/* ]] || continue
  n=$((n+1))
  archivo "$raiz/$ruta" >"$tmp/carga" 2>&1 || {
    echo "Fallo fixture SQL $n: $ruta" >&2; tail -12 "$tmp/carga" >&2; exit 1;
  }
done < "$fixture"
[[ $n == 39 ]]
abiertos="SELECT count(*) FROM pg_catalog.pg_type AS t JOIN pg_catalog.pg_namespace AS n
ON n.oid=t.typnamespace CROSS JOIN LATERAL pg_catalog.aclexplode(
coalesce(t.typacl,pg_catalog.acldefault('T',t.typowner))) AS a
WHERE n.nspname IN ('vec_contratacion_temporal','vec_autorizacion_atestada_v3')
AND t.typtype='c' AND t.typrelid<>0 AND a.grantee=0 AND a.privilege_type='USAGE'"
[[ $(sql "$abiertos") == 42 ]]
inventario="SELECT count(*) FROM pg_catalog.pg_type AS t JOIN pg_catalog.pg_namespace AS n
ON n.oid=t.typnamespace WHERE n.nspname IN
('vec_contratacion_temporal','vec_autorizacion_atestada_v3')
AND t.typtype='c' AND t.typrelid<>0"
[[ $(sql "$inventario") == 42 ]]
huella_ajena="SELECT pg_catalog.md5(coalesce(
 pg_catalog.string_agg(clase||':'||oid::text||':'||acl, '|' ORDER BY clase,oid),
 '')) FROM (
 SELECT 'tipo' AS clase,t.oid,coalesce(t.typacl::text,'NULL') AS acl
 FROM pg_catalog.pg_type AS t JOIN pg_catalog.pg_namespace AS n ON n.oid=t.typnamespace
 WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema'
 AND n.nspname NOT IN ('vec_contratacion_temporal','vec_autorizacion_atestada_v3')
 UNION ALL SELECT 'tipo_auxiliar',t.oid,coalesce(t.typacl::text,'NULL')
 FROM pg_catalog.pg_type AS t JOIN pg_catalog.pg_namespace AS n ON n.oid=t.typnamespace
 WHERE n.nspname IN ('vec_contratacion_temporal','vec_autorizacion_atestada_v3')
 AND NOT (t.typtype='c' AND t.typrelid<>0)
 UNION ALL SELECT 'tabla',c.oid,coalesce(c.relacl::text,'NULL')
 FROM pg_catalog.pg_class AS c JOIN pg_catalog.pg_namespace AS n ON n.oid=c.relnamespace
 WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema'
 UNION ALL SELECT 'funcion',p.oid,coalesce(p.proacl::text,'NULL')
 FROM pg_catalog.pg_proc AS p JOIN pg_catalog.pg_namespace AS n ON n.oid=p.pronamespace
 WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema'
 UNION ALL SELECT 'esquema',n.oid,coalesce(n.nspacl::text,'NULL')
 FROM pg_catalog.pg_namespace AS n
 WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema'
 UNION ALL SELECT 'default',d.oid,coalesce(d.defaclacl::text,'NULL')
 FROM pg_catalog.pg_default_acl AS d
 ) AS objetos"
antes=$(sql "$huella_ajena")

# El ROLLBACK completo deja los 42 tipos abiertos.
sed '$s/^COMMIT;$/ROLLBACK;/' "$delta" | archivo /dev/stdin >"$tmp/rollback" 2>&1 || {
  tail -12 "$tmp/rollback" >&2; exit 1;
}
[[ $(sql "$abiertos") == 42 ]]
# Un solo tipo cerrado es estado parcial: abortar y conservar la preimagen.
{
  echo 'BEGIN; REVOKE USAGE ON TYPE vec_contratacion_temporal.identidad_reserva_alta FROM PUBLIC;'
  sed -e 's/^BEGIN;$/-- BEGIN controlado por prueba/' -e 's/^COMMIT;$/-- COMMIT controlado por prueba/' "$delta"
  echo 'ROLLBACK;'
} >"$tmp/parcial.sql"
fallo_55000 parcial bash -c 'docker exec -i "$1" psql -Xq -v ON_ERROR_STOP=1 -v VERBOSITY=verbose -U postgres -d postgres < "$2"' bash "$contenedor" "$tmp/parcial.sql"
[[ $(sql "$abiertos") == 42 ]]
# Concesión ajena al tipo fila: rechazo sin revocación parcial.
{
  echo 'BEGIN; GRANT USAGE ON TYPE vec_contratacion_temporal.identidad_reserva_alta TO vec_autorizacion_atestada_v3_consumidor;'
  sed -e 's/^BEGIN;$/-- BEGIN controlado por prueba/' -e 's/^COMMIT;$/-- COMMIT controlado por prueba/' "$delta"
  echo 'ROLLBACK;'
} >"$tmp/ajeno.sql"
fallo_55000 ajeno bash -c 'docker exec -i "$1" psql -Xq -v ON_ERROR_STOP=1 -v VERBOSITY=verbose -U postgres -d postgres < "$2"' bash "$contenedor" "$tmp/ajeno.sql"
[[ $(sql "$abiertos") == 42 ]]

archivo "$delta" >"$tmp/apply" 2>&1
grep -q '42 tipos fila cerrados' "$tmp/apply"
[[ $(sql "$abiertos") == 0 ]]
[[ $(sql "$inventario") == 42 ]]
[[ $(sql "$huella_ajena") == "$antes" ]]
[[ $(sql "SELECT count(*) FROM pg_catalog.pg_type AS t
 JOIN pg_catalog.pg_namespace AS n ON n.oid=t.typnamespace
 WHERE n.nspname IN ('vec_contratacion_temporal','vec_autorizacion_atestada_v3')
 AND t.typtype='c' AND t.typrelid<>0
 AND (t.typacl IS NULL OR
      (SELECT count(*) FROM pg_catalog.aclexplode(t.typacl))<>1 OR
      NOT EXISTS (SELECT 1 FROM pg_catalog.aclexplode(t.typacl) AS a
      WHERE a.grantee=t.typowner AND a.grantor=t.typowner
      AND a.privilege_type='USAGE' AND NOT a.is_grantable))") == 0 ]]
archivo "$delta" >"$tmp/repeticion" 2>&1
grep -q 'NO_APLICA ACL CT/AD3' "$tmp/repeticion"
[[ $(sql "$abiertos") == 0 ]]
[[ $(sql "$huella_ajena") == "$antes" ]]
docker restart "$contenedor" >/dev/null
for _ in $(seq 1 120); do
  if docker exec "$contenedor" pg_isready -q -U postgres -d postgres >/dev/null 2>&1; then break; fi
  sleep 0.25
done
[[ $(sql "$abiertos") == 0 ]]
[[ $(sql "$huella_ajena") == "$antes" ]]
archivo "$raiz/$base/roles_contexto_corporativo_rrhh_selector_v1_up.sql" >"$tmp/selector" 2>&1 || {
  echo 'Selector rechazó postimagen' >&2; tail -12 "$tmp/selector" >&2; exit 1;
}
[[ $(sql "SELECT count(*) FROM pg_catalog.pg_roles WHERE rolname='vec_contexto_actor_corporativo_rrhh_selector'") == 1 ]]
archivo "$delta" >"$tmp/cerrado_con_selector" 2>&1
grep -q 'NO_APLICA ACL CT/AD3' "$tmp/cerrado_con_selector"
[[ $(sql "$abiertos") == 0 ]]
echo 'OK: 42 CT/AD3 abiertos→cerrados, rollback, parcial, deriva, repetición, reinicio y selector'
