#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-selector-c3-local-$$"
contexto=$(docker context inspect --format '{{.Endpoints.docker.Host}}' 2>/dev/null || true)
[[ $contexto == unix:///var/run/docker.sock || $contexto =~ ^unix:///run/user/[0-9]+/docker\.sock$ ]] || {
  echo 'C3 ACL prueba: Docker no usa un socket local admitido' >&2; exit 1
}
docker_local() { docker --host "$contexto" "$@"; }
clave=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
tmp=$(mktemp -d)
limpiar() { docker_local rm -f "$contenedor" >/dev/null 2>&1 || true; rm -rf -- "$tmp"; }
trap limpiar EXIT INT TERM
docker_local run -d --rm --network none --name "$contenedor" \
  -e POSTGRES_PASSWORD="$clave" "$imagen" >/dev/null
listo=false
for _ in $(seq 1 120); do
  if [[ $(docker_local exec "$contenedor" sh -c 'tr -d "\n" </proc/1/comm' 2>/dev/null || true) == postgres ]] \
      && docker_local exec "$contenedor" pg_isready -q -U postgres -d postgres >/dev/null 2>&1 \
      && docker_local exec "$contenedor" psql -XAtq -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then
    listo=true; break
  fi
  sleep 0.25
done
[[ $listo == true ]]
psql_archivo() {
  docker_local exec -i "$contenedor" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d postgres < "$raiz/$1"
}
psql_sql() {
  docker_local exec -i "$contenedor" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d postgres
}
valor() {
  docker_local exec "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"
}
[[ $(valor 'SHOW server_version_num') == 180004 ]]
psql_sql <<'SQL'
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL

# ContextoActor real hasta 000003, AD4 y Bolsa B1, todos sin datos nominales.
psql_archivo deploy/postgresql/contexto_actor_v1/roles_up.sql
psql_archivo deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql
psql_archivo deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql
psql_archivo deploy/postgresql/contexto_actor_v1/migraciones/000003_organizacion_corporativa_v1.up.sql
psql_archivo deploy/postgresql/autorizacion/roles_up.sql
psql_archivo deploy/postgresql/autorizacion/roles_v2_up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql
psql_archivo deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql
instalador="$raiz/deploy/postgresql/contexto_actor_v1/instalar_selector_c3_v1.sh"
opciones=(--container "$contenedor" --database postgres --admin-user postgres)
if "$instalador" --inspect "${opciones[@]}" >"$tmp/sin_b1" 2>&1; then
  echo 'instalador aceptó la ausencia de las catorce tablas B1' >&2; exit 1
fi
grep -Fq 'falta el delta RLS B1' "$tmp/sin_b1"
psql_archivo deploy/postgresql/bolsa_llamamientos/roles_up.sql
psql_archivo deploy/postgresql/bolsa_llamamientos/migraciones_autorizacion/000001_revalidacion_llamamientos.up.sql
psql_archivo deploy/postgresql/bolsa_llamamientos/migraciones/000001_almacen_llamamientos.up.sql
psql_sql <<'SQL'
BEGIN;
LOCK TABLE pg_catalog.pg_type IN SHARE ROW EXCLUSIVE MODE;
ROLLBACK;
SQL

if "$instalador" --inspect "${opciones[@]}" >"$tmp/sin_rls" 2>&1; then
  echo 'instalador aceptó políticas B1 para PUBLIC' >&2; exit 1
fi
grep -Fq 'falta el delta RLS B1' "$tmp/sin_rls"
rls_sql=${VEC_C3_RLS_B1_SQL:-$raiz/deploy/postgresql/bolsa_llamamientos/dba/20260928_b1_rls_propietario/01_cerrar_politicas.sql}
[[ -f $rls_sql ]] || { echo 'falta el paquete RLS B1 revisado para esta prueba' >&2; exit 1; }
docker_local exec -i "$contenedor" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d postgres < "$rls_sql"
psql_sql <<'SQL'
ALTER DATABASE postgres SET default_transaction_isolation='repeatable read';
SQL
[[ $(valor 'SHOW default_transaction_isolation') == 'repeatable read' ]]
if ! huella=$("$instalador" --inspect "${opciones[@]}" 2>"$tmp/inspeccion"); then
  cat "$tmp/inspeccion" >&2
  psql_sql <<'SQL' >&2
SELECT c.relname,c.relkind,c.relrowsecurity,c.relforcerowsecurity,
       p.polname,p.polcmd,p.polpermissive,p.polroles,
       pg_get_expr(p.polqual,p.polrelid) AS uso,
       pg_get_expr(p.polwithcheck,p.polrelid) AS comprobacion
  FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
  JOIN pg_policy p ON p.polrelid=c.oid
 WHERE n.nspname='vec_bolsa_llamamientos'
 ORDER BY c.relname;
SQL
  exit 1
fi
psql_sql <<'SQL'
SET ROLE vec_bolsa_llamamientos_propietario;
ALTER POLICY solo_propietario ON vec_bolsa_llamamientos.propuesta
  USING (false) WITH CHECK (false);
RESET ROLE;
SQL
if "$instalador" --inspect "${opciones[@]}" >"$tmp/predicado" 2>&1; then
  echo 'instalador aceptó una política B1 de predicado alterado' >&2; exit 1
fi
grep -Fq 'falta el delta RLS B1' "$tmp/predicado"
psql_sql <<'SQL'
SET ROLE vec_bolsa_llamamientos_propietario;
ALTER POLICY solo_propietario ON vec_bolsa_llamamientos.propuesta
  USING (current_user = 'vec_bolsa_llamamientos_propietario')
  WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
RESET ROLE;
SQL
psql_sql <<'SQL'
GRANT SELECT (propuesta_ref) ON vec_bolsa_llamamientos.propuesta
  TO vec_bolsa_llamamientos_ejecutor;
SQL
if "$instalador" --inspect "${opciones[@]}" >"$tmp/columna" 2>&1; then
  echo 'instalador aceptó ACL ajena por columna B1' >&2; exit 1
fi
grep -Fq 'falta el delta RLS B1' "$tmp/columna"
psql_sql <<'SQL'
REVOKE SELECT (propuesta_ref) ON vec_bolsa_llamamientos.propuesta
  FROM vec_bolsa_llamamientos_ejecutor;
SQL
huella=$("$instalador" --inspect "${opciones[@]}")
[[ $huella =~ ^[0-9a-f]{64}$ ]]
if "$instalador" --apply "${opciones[@]}" --expected-preimage-sha256 "$(printf '0%.0s' {1..64})" >"$tmp/huella" 2>&1; then
  echo 'instalador aceptó otra preimagen' >&2; exit 1
fi
grep -Fq 'preimagen distinta' "$tmp/huella"
[[ $(valor "SELECT to_regrole('vec_contexto_actor_corporativo_rrhh_selector') IS NULL") == t ]]

# Carrera causal: una concesión por columna queda sin confirmar mientras la
# lectura previa ve B1 cerrada. El instalador espera el lock de propuesta;
# tras el COMMIT hostil debe revalidar bajo sus catorce locks y revertir todo.
docker_local exec -i "$contenedor" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d postgres \
  >"$tmp/escritor" 2>&1 <<'SQL' &
BEGIN;
LOCK TABLE vec_bolsa_llamamientos.propuesta IN ACCESS SHARE MODE;
GRANT SELECT (propuesta_ref) ON vec_bolsa_llamamientos.propuesta
  TO vec_bolsa_llamamientos_ejecutor;
SELECT pg_sleep(5);
COMMIT;
SQL
escritor_pid=$!
escritor_listo=false
for _ in $(seq 1 80); do
  if [[ $(valor "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE query LIKE 'SELECT pg_sleep(5)%' AND state='active')") == t ]]; then
    escritor_listo=true; break
  fi
  sleep 0.05
done
[[ $escritor_listo == true ]] || { echo 'no se observó escritor pendiente' >&2; exit 1; }
"$instalador" --apply "${opciones[@]}" --expected-preimage-sha256 "$huella" >"$tmp/carrera" 2>&1 &
instalador_pid=$!
espera_lock=false
for _ in $(seq 1 80); do
  if [[ $(valor "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE wait_event_type='Lock' AND query LIKE 'LOCK TABLE%')") == t ]]; then
    espera_lock=true; break
  fi
  sleep 0.05
done
[[ $espera_lock == true ]] || { echo 'instalador no esperó el lock B1' >&2; exit 1; }
wait "$escritor_pid"
if wait "$instalador_pid"; then
  echo 'instalador aceptó concesión concurrente por columna' >&2; exit 1
fi
grep -Eq 'preimagen cambió bajo lock|RLS B1 cambió bajo lock' "$tmp/carrera"
[[ $(valor "SELECT to_regrole('vec_contexto_actor_corporativo_rrhh_selector') IS NULL") == t ]]
[[ $(valor "SELECT count(*) FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace
  CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
  WHERE n.nspname='vec_bolsa_llamamientos' AND t.typtype='c'
    AND a.grantee=0 AND a.privilege_type='USAGE'") == 14 ]]
psql_sql <<'SQL'
REVOKE SELECT (propuesta_ref) ON vec_bolsa_llamamientos.propuesta
  FROM vec_bolsa_llamamientos_ejecutor;
SQL
huella=$("$instalador" --inspect "${opciones[@]}")

# La tabla AD4 también forma parte de la huella y del conjunto bloqueado.
docker_local exec -i "$contenedor" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d postgres \
  >"$tmp/escritor_ad4" 2>&1 <<'SQL' &
BEGIN;
LOCK TABLE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2 IN ACCESS SHARE MODE;
GRANT SELECT (decision_ref) ON vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
  TO vec_bolsa_llamamientos_ejecutor;
SELECT pg_sleep(5);
COMMIT;
SQL
escritor_ad4_pid=$!
ad4_listo=false
for _ in $(seq 1 80); do
  if [[ $(valor "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE query LIKE 'SELECT pg_sleep(5)%' AND state='active')") == t ]]; then
    ad4_listo=true; break
  fi
  sleep 0.05
done
[[ $ad4_listo == true ]] || { echo 'no se observó escritor AD4 pendiente' >&2; exit 1; }
"$instalador" --apply "${opciones[@]}" --expected-preimage-sha256 "$huella" >"$tmp/carrera_ad4" 2>&1 &
instalador_ad4_pid=$!
ad4_espera=false
for _ in $(seq 1 80); do
  if [[ $(valor "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE wait_event_type='Lock' AND query LIKE 'LOCK TABLE%')") == t ]]; then
    ad4_espera=true; break
  fi
  sleep 0.05
done
[[ $ad4_espera == true ]] || { echo 'instalador no esperó tabla AD4' >&2; exit 1; }
wait "$escritor_ad4_pid"
if wait "$instalador_ad4_pid"; then
  echo 'instalador aceptó ACL AD4 concurrente' >&2; exit 1
fi
grep -Eq 'preimagen cambió bajo lock|RLS B1 cambió bajo lock' "$tmp/carrera_ad4"
[[ $(valor "SELECT to_regrole('vec_contexto_actor_corporativo_rrhh_selector') IS NULL") == t ]]
psql_sql <<'SQL'
REVOKE SELECT (decision_ref) ON vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
  FROM vec_bolsa_llamamientos_ejecutor;
SQL
huella=$("$instalador" --inspect "${opciones[@]}")

# Un tipo ajeno PUBLIC hace fallar al selector DESPUÉS del cuerpo ACL. El
# único COMMIT exterior debe revertir también los quince REVOKE previos.
psql_sql <<'SQL'
CREATE DOMAIN public.c3_tipo_hostil AS text;
SQL
if "$instalador" --apply "${opciones[@]}" --expected-preimage-sha256 "$huella" >"$tmp/rollback" 2>&1; then
  echo 'instalador aceptó un tipo ajeno abierto' >&2; exit 1
fi
grep -Fq 'alta del selector corporativo RRHH rechazada' "$tmp/rollback"
[[ $(valor "SELECT to_regrole('vec_contexto_actor_corporativo_rrhh_selector') IS NULL") == t ]]
[[ $(valor "SELECT count(*) FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace
  CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
  WHERE ((n.nspname='vec_bolsa_llamamientos' AND t.typtype='c')
      OR (n.nspname='vec_autorizacion' AND t.typname='decision_autorizacion_solicitud_ligada_v2'))
    AND a.grantee=0 AND a.privilege_type='USAGE'") == 15 ]]
psql_sql <<'SQL'
DROP DOMAIN public.c3_tipo_hostil;
SQL

# Bloqueo tardío en la guarda del selector: el instalador ya debe retener
# ShareRowExclusive sobre pg_type. Un GRANT ON TYPE concurrente espera; al
# liberar la barrera, el instalador confirma y el GRANT se revierte.
mkfifo "$tmp/barrera_entrada"
docker_local exec -i -e PGAPPNAME=vec-c3-barrera "$contenedor" \
  psql -Xq -v ON_ERROR_STOP=1 -U postgres -d postgres \
  <"$tmp/barrera_entrada" >"$tmp/barrera" 2>&1 &
barrera_pid=$!
exec {barrera_fd}>"$tmp/barrera_entrada"
printf 'BEGIN;\nLOCK TABLE pg_catalog.pg_shdescription IN ACCESS EXCLUSIVE MODE;\n' >&"$barrera_fd"
barrera_lista=false
for _ in $(seq 1 100); do
  if [[ $(valor "SELECT EXISTS(SELECT 1 FROM pg_stat_activity a JOIN pg_locks l ON l.pid=a.pid WHERE a.application_name='vec-c3-barrera' AND l.relation='pg_shdescription'::regclass AND l.mode='AccessExclusiveLock' AND l.granted)") == t ]]; then
    barrera_lista=true; break
  fi
  sleep 0.05
done
[[ $barrera_lista == true ]] || { echo 'no se observó barrera tardía' >&2; exit 1; }
"$instalador" --apply "${opciones[@]}" --expected-preimage-sha256 "$huella" >"$tmp/instalar" 2>&1 &
instalador_pid=$!
tipo_protegido=false
for _ in $(seq 1 100); do
  if [[ $(valor "SELECT EXISTS(SELECT 1 FROM pg_stat_activity a JOIN pg_locks l ON l.pid=a.pid WHERE a.application_name='vec-c3-acl-installer' AND a.wait_event_type='Lock' AND l.relation='pg_type'::regclass AND l.mode='ShareRowExclusiveLock' AND l.granted)") == t ]]; then
    tipo_protegido=true; break
  fi
  sleep 0.05
done
[[ $tipo_protegido == true ]] || { echo 'instalador no retuvo pg_type bajo barrera' >&2; exit 1; }
docker_local exec -i -e PGAPPNAME=vec-c3-grant-tipo "$contenedor" \
  psql -Xq -v ON_ERROR_STOP=1 -U postgres -d postgres >"$tmp/grant_tipo" 2>&1 <<'SQL' &
BEGIN;
GRANT USAGE ON TYPE vec_bolsa_llamamientos.propuesta
  TO vec_bolsa_llamamientos_ejecutor;
ROLLBACK;
SQL
grant_tipo_pid=$!
grant_espera=false
for _ in $(seq 1 100); do
  if [[ $(valor "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name='vec-c3-grant-tipo' AND wait_event_type='Lock')") == t ]]; then
    grant_espera=true; break
  fi
  sleep 0.05
done
[[ $grant_espera == true ]] || { echo 'GRANT ON TYPE no esperó pg_type' >&2; exit 1; }
printf 'COMMIT;\n' >&"$barrera_fd"
exec {barrera_fd}>&-
wait "$barrera_pid"
wait "$grant_tipo_pid"
if ! wait "$instalador_pid"; then
  cat "$tmp/instalar" >&2
  psql_sql <<'SQL' >&2
SELECT 'base',d.datname FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a WHERE NOT d.datistemplate AND a.grantee=0
UNION ALL
SELECT 'relacion',n.nspname||'.'||c.relname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN LATERAL aclexplode(c.relacl) a WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema' AND a.grantee=0
UNION ALL
SELECT 'columna',n.nspname||'.'||c.relname||'.'||x.attname FROM pg_attribute x JOIN pg_class c ON c.oid=x.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN LATERAL aclexplode(x.attacl) a WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema' AND a.grantee=0
UNION ALL
SELECT 'tipo_base',n.nspname||'.'||t.typname FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema' AND t.typtype='b' AND NOT EXISTS (SELECT 1 FROM pg_type e WHERE e.oid=t.typelem AND e.typarray=t.oid) AND a.grantee=0
UNION ALL
SELECT 'default_acl',d.defaclrole::regrole::text FROM pg_default_acl d CROSS JOIN LATERAL aclexplode(d.defaclacl) a WHERE a.grantee=0
UNION ALL
SELECT 'politica',p.polname FROM pg_policy p WHERE 0::oid=ANY(p.polroles)
UNION ALL
SELECT 'parametro',p.parname FROM pg_parameter_acl p CROSS JOIN LATERAL aclexplode(p.paracl) a WHERE a.grantee=0
ORDER BY 1,2;
SELECT 'tipo',n.nspname||'.'||t.typname FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace
 CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
 WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema' AND t.typtype IN ('c','d','e','m','r') AND a.grantee=0
UNION ALL
SELECT 'funcion',n.nspname||'.'||p.proname FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
 WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema' AND a.grantee=0
UNION ALL
SELECT 'esquema',n.nspname FROM pg_namespace n
 CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
 WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema' AND a.grantee=0
ORDER BY 1,2;
SQL
  exit 1
fi
[[ $(valor "SELECT to_regrole('vec_contexto_actor_corporativo_rrhh_selector') IS NOT NULL") == t ]]
if "$instalador" --apply "${opciones[@]}" --expected-preimage-sha256 "$huella" >"$tmp/repetir" 2>&1; then
  echo 'instalador aceptó repetición' >&2; exit 1
fi
grep -Fq 'preimagen distinta' "$tmp/repetir"

docker_local restart "$contenedor" >/dev/null
listo=false
for _ in $(seq 1 120); do
  if docker_local exec "$contenedor" pg_isready -q -U postgres -d postgres >/dev/null 2>&1 \
      && docker_local exec "$contenedor" psql -XAtq -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then
    listo=true; break
  fi
  sleep 0.25
done
[[ $listo == true ]]
psql_archivo deploy/postgresql/contexto_actor_v1/pruebas_sql/acl_tipos_preselector_c3_v1.sql
psql_sql <<'SQL'
SET ROLE vec_bolsa_llamamientos_propietario;
CREATE TABLE vec_bolsa_llamamientos.c3_tipo_posterior (id integer PRIMARY KEY);
RESET ROLE;
SQL
if psql_archivo deploy/postgresql/contexto_actor_v1/pruebas_sql/acl_tipos_preselector_c3_v1.sql >"$tmp/deriva" 2>&1; then
  echo 'puerta final aceptó un tipo posterior abierto' >&2; exit 1
fi
grep -Fq 'USAGE PUBLIC recuperado' "$tmp/deriva"
echo 'OK C3 instalador: preimagen explícita, ACL→gate→selector, reinicio y deriva posterior'
