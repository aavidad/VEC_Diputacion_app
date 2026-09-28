#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-b1-rls-$(id -u)-$$"
base=vec_b1_rls_prueba
paquete=deploy/postgresql/bolsa_llamamientos/dba/20260928_b1_rls_propietario/01_cerrar_politicas.sql
clave=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')

limpiar() { docker rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT INT TERM
docker run --detach --rm --network none --name "$contenedor" \
    --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave" "$imagen" >/dev/null
for _ in $(seq 1 60); do
    if docker exec "$contenedor" pg_isready -U postgres -d "$base" >/dev/null 2>&1; then break; fi
    sleep 1
done
docker exec "$contenedor" pg_isready -U postgres -d "$base" >/dev/null

sql() {
    docker exec --interactive "$contenedor" psql -X -qAt \
        -v ON_ERROR_STOP=1 -U postgres -d "$base" "$@"
}
archivo() { sql < "$raiz/$1"; }
escalar() { sql -c "$1"; }
fallar() { printf '%s\n' "$1" >&2; exit 1; }

# Cadena auténtica mínima que instaló B1: roles, autorización y almacén.
archivo deploy/postgresql/autorizacion/roles_up.sql >/dev/null
archivo deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql >/dev/null
archivo deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql >/dev/null
archivo deploy/postgresql/bolsa_llamamientos/roles_up.sql >/dev/null
archivo deploy/postgresql/bolsa_llamamientos/migraciones_autorizacion/000001_revalidacion_llamamientos.up.sql >/dev/null
archivo deploy/postgresql/bolsa_llamamientos/migraciones/000001_almacen_llamamientos.up.sql >/dev/null

contar_roles() {
    escalar "SELECT count(*) FROM pg_catalog.pg_policy p JOIN pg_catalog.pg_class c ON c.oid=p.polrelid JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_llamamientos' AND p.polname='solo_propietario' AND p.polroles=$1"
}
[[ $(contar_roles 'ARRAY[0]::oid[]') == 14 ]] || fallar 'preimagen B1: no hay catorce políticas PUBLIC'
[[ $(escalar "SELECT count(*) FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_llamamientos' AND c.relkind='r' AND c.relrowsecurity AND c.relforcerowsecurity") == 14 ]] || fallar 'preimagen B1: RLS FORCE incompleta'

huella_predicados() {
    escalar "SELECT md5(string_agg(c.relname || ':' || pg_catalog.pg_get_expr(p.polqual,p.polrelid) || ':' || pg_catalog.pg_get_expr(p.polwithcheck,p.polrelid), '|' ORDER BY c.relname)) FROM pg_catalog.pg_policy p JOIN pg_catalog.pg_class c ON c.oid=p.polrelid JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_llamamientos' AND p.polname='solo_propietario'"
}
huella_acl() {
    escalar "SELECT md5(string_agg(c.relname || ':' || coalesce(c.relacl::text,'<NULL>'), '|' ORDER BY c.relname)) FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_llamamientos' AND c.relkind='r'"
}
antes_predicados=$(huella_predicados)
antes_acl=$(huella_acl)
antes_historia=$(escalar "SELECT md5(row_to_json(a)::text) FROM vec_bolsa_llamamientos.auditoria_actual a")

rechazar_deriva() {
    if archivo "$paquete" >/dev/null 2>&1; then
        fallar "B1 RLS: admitió deriva $1"
    fi
    [[ $(contar_roles 'ARRAY[0]::oid[]') == 14 ]] || fallar "B1 RLS: efecto parcial tras deriva $1"
}

escalar "ALTER POLICY solo_propietario ON vec_bolsa_llamamientos.bolsa_autoritativa USING (true)" >/dev/null
rechazar_deriva predicado
escalar "ALTER POLICY solo_propietario ON vec_bolsa_llamamientos.bolsa_autoritativa USING (current_user='vec_bolsa_llamamientos_propietario')" >/dev/null

escalar "ALTER POLICY solo_propietario ON vec_bolsa_llamamientos.bolsa_autoritativa TO vec_bolsa_llamamientos_ejecutor" >/dev/null
if archivo "$paquete" >/dev/null 2>&1; then fallar 'B1 RLS: admitió rol ajeno'; fi
[[ $(contar_roles 'ARRAY[0]::oid[]') == 13 ]] || fallar 'B1 RLS: alteró otras políticas al rechazar rol ajeno'
escalar "ALTER POLICY solo_propietario ON vec_bolsa_llamamientos.bolsa_autoritativa TO PUBLIC" >/dev/null

escalar "CREATE POLICY extra_ajena ON vec_bolsa_llamamientos.bolsa_autoritativa USING (true)" >/dev/null
rechazar_deriva politica_adicional
escalar "DROP POLICY extra_ajena ON vec_bolsa_llamamientos.bolsa_autoritativa" >/dev/null

escalar "GRANT SELECT ON vec_bolsa_llamamientos.bolsa_autoritativa TO vec_bolsa_llamamientos_ejecutor" >/dev/null
rechazar_deriva acl_directa
escalar "REVOKE SELECT ON vec_bolsa_llamamientos.bolsa_autoritativa FROM vec_bolsa_llamamientos_ejecutor" >/dev/null

escalar "ALTER TABLE vec_bolsa_llamamientos.bolsa_autoritativa NO FORCE ROW LEVEL SECURITY" >/dev/null
rechazar_deriva rls_no_forzada
escalar "ALTER TABLE vec_bolsa_llamamientos.bolsa_autoritativa FORCE ROW LEVEL SECURITY" >/dev/null

[[ $(huella_predicados) == "$antes_predicados" ]] || fallar 'restauración negativa alteró predicados'
[[ $(huella_acl) == "$antes_acl" ]] || fallar 'restauración negativa alteró ACL'

archivo "$paquete" >/dev/null
[[ $(contar_roles "ARRAY['vec_bolsa_llamamientos_propietario'::regrole::oid]") == 14 ]] || fallar 'postimagen: roles no cerrados'
[[ $(contar_roles 'ARRAY[0]::oid[]') == 0 ]] || fallar 'postimagen: quedan políticas PUBLIC B1'
[[ $(huella_predicados) == "$antes_predicados" ]] || fallar 'postimagen: predicados cambiados'
[[ $(huella_acl) == "$antes_acl" ]] || fallar 'postimagen: ACL cambiadas'
[[ $(escalar "SELECT md5(row_to_json(a)::text) FROM vec_bolsa_llamamientos.auditoria_actual a") == "$antes_historia" ]] || fallar 'postimagen: historia alterada'
[[ $(escalar "SELECT count(*) FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_llamamientos' AND c.relkind='r' AND c.relrowsecurity AND c.relforcerowsecurity") == 14 ]] || fallar 'postimagen: RLS FORCE alterada'
if archivo "$paquete" >/dev/null 2>&1; then fallar 'el paquete pudo reaplicarse'; fi
[[ $(contar_roles "ARRAY['vec_bolsa_llamamientos_propietario'::regrole::oid]") == 14 ]] || fallar 'reaplicación afectó postimagen'

printf '%s\n' 'B1 RLS PG18.4: preimagen, negativas, postimagen, historia y ACL OK'
