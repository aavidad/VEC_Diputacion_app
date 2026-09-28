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

sql_en() {
    local db=$1
    shift
    docker exec --interactive "$contenedor" psql -X -qAt \
        -v ON_ERROR_STOP=1 -U postgres -d "$db" "$@"
}
sql() { sql_en "$base" "$@"; }
archivo() { sql < "$raiz/$1"; }
escalar() { sql -c "$1"; }
archivo_en() { local db=$1; shift; sql_en "$db" < "$raiz/$1"; }
escalar_en() { local db=$1; shift; sql_en "$db" -c "$1"; }
fallar() { printf '%s\n' "$1" >&2; exit 1; }

# La ausencia total, también con esquema Bolsa de migraciones posteriores,
# debe informar NO_APLICA sin tocar políticas ni exigir SET ROLE.
salida_cero=$(archivo "$paquete")
[[ $salida_cero == *NO_APLICA* ]] || fallar 'ausencia B1 sin respuesta NO_APLICA'
[[ $(escalar "SELECT count(*) FROM pg_catalog.pg_policy p JOIN pg_catalog.pg_class c ON c.oid=p.polrelid JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_llamamientos'") == 0 ]] || fallar 'ausencia B1 alteró políticas'

# Cadena auténtica mínima que instaló B1: roles, autorización y almacén.
archivo deploy/postgresql/autorizacion/roles_up.sql >/dev/null
archivo deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql >/dev/null
archivo deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql >/dev/null
archivo deploy/postgresql/bolsa_llamamientos/roles_up.sql >/dev/null
archivo deploy/postgresql/bolsa_llamamientos/migraciones_autorizacion/000001_revalidacion_llamamientos.up.sql >/dev/null

parcial=vec_b1_rls_parcial
docker exec "$contenedor" createdb -U postgres "$parcial"
escalar_en "$parcial" "CREATE SCHEMA vec_bolsa_llamamientos AUTHORIZATION vec_bolsa_llamamientos_propietario" >/dev/null
escalar_en "$parcial" "CREATE TABLE vec_bolsa_llamamientos.tabla_posterior(id integer); CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.tabla_posterior TO vec_bolsa_llamamientos_propietario USING (true)" >/dev/null
salida_cero=$(archivo_en "$parcial" "$paquete")
[[ $salida_cero == *NO_APLICA* ]] || fallar 'esquema Bolsa sin B1 no devolvió NO_APLICA'
[[ $(escalar_en "$parcial" "SELECT polroles=ARRAY['vec_bolsa_llamamientos_propietario'::regrole::oid] FROM pg_catalog.pg_policy WHERE polrelid='vec_bolsa_llamamientos.tabla_posterior'::regclass") == t ]] || fallar 'NO_APLICA alteró política ajena'

tablas_b1=(bolsa_autoritativa necesidad_autoritativa necesidad_actual politica_autoritativa instantanea_autoritativa evaluacion_autoritativa atestacion_autorizacion_version atestacion_autorizacion_actual propuesta referencia_consumida uso_decision auditoria auditoria_actual outbox)
for indice in $(seq 0 12); do
    tabla=${tablas_b1[$indice]}
    escalar_en "$parcial" "CREATE TABLE vec_bolsa_llamamientos.$tabla(id integer); CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.$tabla USING (current_user='vec_bolsa_llamamientos_propietario') WITH CHECK (current_user='vec_bolsa_llamamientos_propietario')" >/dev/null
    if salida_parcial=$(archivo_en "$parcial" "$paquete" 2>&1); then
        fallar "B1 parcial $((indice+1))/14 fue aceptada"
    fi
    siguiente=${tablas_b1[$((indice+1))]}
    [[ $salida_parcial == *'preimagen parcial o derivada'* && $salida_parcial == *"$siguiente:tabla_ausente"* ]] || fallar "B1 parcial $((indice+1))/14 no nombró $siguiente"
    [[ $(escalar_en "$parcial" "SELECT count(*) FROM pg_catalog.pg_policy p JOIN pg_catalog.pg_class c ON c.oid=p.polrelid JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_llamamientos' AND c.relname <> 'tabla_posterior' AND p.polroles=ARRAY[0]::oid[]") == $((indice+1)) ]] || fallar "B1 parcial $((indice+1))/14 cambió políticas"
done

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
huella_acl_columnas() {
    escalar "SELECT md5(coalesce(string_agg(c.relname || ':' || a.attnum::text || ':' || coalesce(a.attacl::text,'<NULL>'), '|' ORDER BY c.relname,a.attnum),'')) FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace JOIN pg_catalog.pg_attribute a ON a.attrelid=c.oid WHERE n.nspname='vec_bolsa_llamamientos' AND c.relkind='r' AND a.attnum>0 AND NOT a.attisdropped"
}
antes_predicados=$(huella_predicados)
antes_acl=$(huella_acl)
antes_acl_columnas=$(huella_acl_columnas)
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

escalar "GRANT SELECT (bolsa_ref) ON vec_bolsa_llamamientos.bolsa_autoritativa TO vec_bolsa_llamamientos_ejecutor" >/dev/null
[[ $(escalar "SELECT has_column_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.bolsa_autoritativa','bolsa_ref','SELECT')") == t ]] || fallar 'no se creó la deriva de ACL por columna'
rechazar_deriva acl_columna
escalar "REVOKE SELECT (bolsa_ref) ON vec_bolsa_llamamientos.bolsa_autoritativa FROM vec_bolsa_llamamientos_ejecutor" >/dev/null

escalar "ALTER TABLE vec_bolsa_llamamientos.bolsa_autoritativa NO FORCE ROW LEVEL SECURITY" >/dev/null
rechazar_deriva rls_no_forzada
escalar "ALTER TABLE vec_bolsa_llamamientos.bolsa_autoritativa FORCE ROW LEVEL SECURITY" >/dev/null

[[ $(huella_predicados) == "$antes_predicados" ]] || fallar 'restauración negativa alteró predicados'
[[ $(huella_acl) == "$antes_acl" ]] || fallar 'restauración negativa alteró ACL'
# GRANT/REVOKE puede materializar un ACL de columna equivalente al implícito;
# la comparación de postimagen toma la preimagen exacta justo antes del paquete.
antes_acl_columnas=$(huella_acl_columnas)

archivo "$paquete" >/dev/null
[[ $(contar_roles "ARRAY['vec_bolsa_llamamientos_propietario'::regrole::oid]") == 14 ]] || fallar 'postimagen: roles no cerrados'
[[ $(contar_roles 'ARRAY[0]::oid[]') == 0 ]] || fallar 'postimagen: quedan políticas PUBLIC B1'
[[ $(huella_predicados) == "$antes_predicados" ]] || fallar 'postimagen: predicados cambiados'
[[ $(huella_acl) == "$antes_acl" ]] || fallar 'postimagen: ACL cambiadas'
[[ $(huella_acl_columnas) == "$antes_acl_columnas" ]] || fallar 'postimagen: ACL de columna cambiadas'
[[ $(escalar "SELECT md5(row_to_json(a)::text) FROM vec_bolsa_llamamientos.auditoria_actual a") == "$antes_historia" ]] || fallar 'postimagen: historia alterada'
[[ $(escalar "SELECT count(*) FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_llamamientos' AND c.relkind='r' AND c.relrowsecurity AND c.relforcerowsecurity") == 14 ]] || fallar 'postimagen: RLS FORCE alterada'
if archivo "$paquete" >/dev/null 2>&1; then fallar 'el paquete pudo reaplicarse'; fi
[[ $(contar_roles "ARRAY['vec_bolsa_llamamientos_propietario'::regrole::oid]") == 14 ]] || fallar 'reaplicación afectó postimagen'

printf '%s\n' 'B1 RLS PG18.4: NO_APLICA 0/14, parciales 1..13 rechazados, 14/14, replay, ACL columna e historia OK'
