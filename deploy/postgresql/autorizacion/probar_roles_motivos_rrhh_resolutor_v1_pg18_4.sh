#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-rol-motivos-rrhh-resolutor-${USER:-usuario}-$$"
base=vec_rol_motivos_rrhh_resolutor
rol=vec_autorizacion_motivos_rrhh_resolutor
clave=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
salidas=$(mktemp -d)
up=deploy/postgresql/autorizacion/roles_motivos_rrhh_resolutor_v1_up.sql
down=deploy/postgresql/autorizacion/roles_motivos_rrhh_resolutor_v1_down.sql

limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  rm -rf "$salidas"
}
trap limpiar EXIT INT TERM

docker run --detach --rm --name "$contenedor" \
  --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave" \
  "$imagen" >/dev/null
disponible=false
for _ in $(seq 1 120); do
  if docker exec "$contenedor" pg_isready -U postgres -d "$base" \
      >/dev/null 2>&1; then
    disponible=true
    break
  fi
  sleep 1
done
[[ $disponible == true ]] || {
  docker logs "$contenedor" >&2 || true
  echo 'PostgreSQL 18.4 no quedó disponible' >&2
  exit 1
}
[[ $(docker exec "$contenedor" psql -XAtq -U postgres -d "$base" \
  -c "SELECT current_setting('server_version_num')") == 180004 ]]

psql_archivo() {
  docker exec --interactive "$contenedor" psql -Xq \
    --set ON_ERROR_STOP=1 -U postgres -d "$base" <"$raiz/$1"
}
psql_valor() {
  docker exec "$contenedor" psql -XAtq --set ON_ERROR_STOP=1 \
    -U postgres -d "$base" -c "$1"
}
psql_sql() {
  docker exec --interactive "$contenedor" psql -Xq \
    --set ON_ERROR_STOP=1 -U postgres -d "$base"
}
psql_archivo_como() {
  local principal=$1 archivo=$2
  {
    printf 'SET ROLE %s;\n' "$principal"
    cat "$raiz/$archivo"
  } | docker exec --interactive "$contenedor" psql -Xq \
    --set ON_ERROR_STOP=1 -U postgres -d "$base"
}
exigir_fallo_up() {
  local caso=$1
  if psql_archivo "$up" >"$salidas/fallo_up" 2>&1; then
    echo "el alta aceptó un estado hostil: $caso" >&2
    exit 1
  fi
  grep -Fq 'alta del resolutor RRHH rechazada' "$salidas/fallo_up"
}
exigir_fallo_down() {
  local caso=$1
  if psql_archivo "$down" >"$salidas/fallo_down" 2>&1; then
    echo "la retirada aceptó un estado hostil: $caso" >&2
    exit 1
  fi
}
exigir_rol() {
  [[ $(psql_valor "SELECT (count(*)=1)::text FROM pg_authid WHERE rolname='$rol'") == true ]]
}
exigir_ausencia() {
  [[ $(psql_valor "SELECT (count(*)=0)::text FROM pg_authid WHERE rolname='$rol'") == true ]]
}
exigir_ausencia_oid() {
  local oid_anterior=$1
  exigir_ausencia
  [[ $(psql_valor "SELECT (NOT EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole=$oid_anterior) AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE roleid=$oid_anterior OR member=$oid_anterior OR grantor=$oid_anterior))::text") == true ]]
  [[ $(psql_valor "SELECT (NOT EXISTS(SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(d.datacl) a WHERE a.grantee=$oid_anterior OR a.grantor=$oid_anterior) AND NOT EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a WHERE a.grantee=$oid_anterior OR a.grantor=$oid_anterior) AND NOT EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(c.relacl) a WHERE a.grantee=$oid_anterior OR a.grantor=$oid_anterior))::text") == true ]]
  [[ $(psql_valor "SELECT (NOT EXISTS(SELECT 1 FROM pg_attribute a CROSS JOIN LATERAL aclexplode(a.attacl) acl WHERE acl.grantee=$oid_anterior OR acl.grantor=$oid_anterior) AND NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE a.grantee=$oid_anterior OR a.grantor=$oid_anterior) AND NOT EXISTS(SELECT 1 FROM pg_type t CROSS JOIN LATERAL aclexplode(t.typacl) a WHERE a.grantee=$oid_anterior OR a.grantor=$oid_anterior))::text") == true ]]
}
exigir_connect() {
  [[ $(psql_valor "SELECT has_database_privilege('$rol',current_database(),'CONNECT')::text") == true ]]
}
huella_objetivo() {
  {
    psql_valor "SELECT rolname,rolcanlogin,rolsuper,rolcreatedb,rolcreaterole,rolinherit,rolreplication,rolbypassrls,rolconnlimit,COALESCE(rolpassword,'<NULL>'),COALESCE(rolvaliduntil::text,'<NULL>'),COALESCE(shobj_description(oid,'pg_authid'),'<NULL>') FROM pg_authid WHERE rolname='$rol'"
    psql_valor "SELECT setdatabase,setrole,setconfig FROM pg_db_role_setting WHERE setrole=(SELECT oid FROM pg_roles WHERE rolname='$rol') ORDER BY 1,2,3"
    psql_valor "SELECT roleid,member,grantor,admin_option,inherit_option,set_option FROM pg_auth_members WHERE roleid='$rol'::regrole OR member='$rol'::regrole OR grantor='$rol'::regrole ORDER BY 1,2,3"
    psql_valor "SELECT 'd',d.datname,a.grantor,a.grantee,a.privilege_type,a.is_grantable FROM pg_database d CROSS JOIN LATERAL aclexplode(d.datacl) a WHERE a.grantee='$rol'::regrole OR a.grantor='$rol'::regrole UNION ALL SELECT 'n',n.nspname,a.grantor,a.grantee,a.privilege_type,a.is_grantable FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a WHERE a.grantee='$rol'::regrole OR a.grantor='$rol'::regrole ORDER BY 1,2,3,4,5"
    psql_valor "SELECT 'c',c.oid::regclass::text,a.grantor,a.grantee,a.privilege_type,a.is_grantable FROM pg_class c CROSS JOIN LATERAL aclexplode(c.relacl) a WHERE a.grantee='$rol'::regrole OR a.grantor='$rol'::regrole UNION ALL SELECT 'p',p.oid::regprocedure::text,a.grantor,a.grantee,a.privilege_type,a.is_grantable FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE a.grantee='$rol'::regrole OR a.grantor='$rol'::regrole UNION ALL SELECT 't',t.oid::regtype::text,a.grantor,a.grantee,a.privilege_type,a.is_grantable FROM pg_type t CROSS JOIN LATERAL aclexplode(t.typacl) a WHERE a.grantee='$rol'::regrole OR a.grantor='$rol'::regrole ORDER BY 1,2,3,4,5"
  } | sha256sum | cut -d' ' -f1
}
huella_v2() {
  local filtro="ARRAY['vec_autorizacion_motivos_proyector'::regrole::oid,'vec_autorizacion_motivos_evaluador'::regrole::oid]"
  {
    psql_valor "SELECT rolname,rolcanlogin,rolsuper,rolcreatedb,rolcreaterole,rolinherit,rolreplication,rolbypassrls,rolconnlimit,COALESCE(rolpassword,'<NULL>'),COALESCE(rolvaliduntil::text,'<NULL>'),COALESCE(shobj_description(oid,'pg_authid'),'<NULL>') FROM pg_authid WHERE oid=ANY($filtro) ORDER BY rolname"
    psql_valor "SELECT setdatabase,setrole,setconfig FROM pg_db_role_setting WHERE setrole=ANY($filtro) ORDER BY 1,2,3"
    psql_valor "SELECT roleid,member,grantor,admin_option,inherit_option,set_option FROM pg_auth_members WHERE roleid=ANY($filtro) OR member=ANY($filtro) OR grantor=ANY($filtro) ORDER BY 1,2,3"
    psql_valor "SELECT 'd',d.datname,a.grantor,a.grantee,a.privilege_type,a.is_grantable FROM pg_database d CROSS JOIN LATERAL aclexplode(d.datacl) a WHERE a.grantee=ANY($filtro) OR a.grantor=ANY($filtro) UNION ALL SELECT 'n',n.nspname,a.grantor,a.grantee,a.privilege_type,a.is_grantable FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a WHERE a.grantee=ANY($filtro) OR a.grantor=ANY($filtro) ORDER BY 1,2,3,4,5"
    psql_valor "SELECT 'c',c.oid::regclass::text,a.grantor,a.grantee,a.privilege_type,a.is_grantable FROM pg_class c CROSS JOIN LATERAL aclexplode(c.relacl) a WHERE a.grantee=ANY($filtro) OR a.grantor=ANY($filtro) UNION ALL SELECT 'p',p.oid::regprocedure::text,a.grantor,a.grantee,a.privilege_type,a.is_grantable FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE a.grantee=ANY($filtro) OR a.grantor=ANY($filtro) UNION ALL SELECT 't',t.oid::regtype::text,a.grantor,a.grantee,a.privilege_type,a.is_grantable FROM pg_type t CROSS JOIN LATERAL aclexplode(t.typacl) a WHERE a.grantee=ANY($filtro) OR a.grantor=ANY($filtro) ORDER BY 1,2,3,4,5"
  } | sha256sum | cut -d' ' -f1
}
exigir_aislamiento() {
  [[ $(psql_valor "SELECT (count(*)=1 AND bool_and(a.grantor=d.datdba AND a.privilege_type='CONNECT' AND NOT a.is_grantable))::text FROM pg_database d CROSS JOIN LATERAL aclexplode(d.datacl) a WHERE d.datname=current_database() AND (a.grantee='$rol'::regrole OR a.grantor='$rol'::regrole)") == true ]]
  [[ $(psql_valor "SELECT (NOT has_database_privilege('$rol',current_database(),'CREATE') AND NOT has_database_privilege('$rol',current_database(),'TEMPORARY') AND NOT has_schema_privilege('$rol','vec_autorizacion','USAGE') AND NOT has_schema_privilege('$rol','vec_autorizacion','CREATE'))::text") == true ]]
  [[ $(psql_valor "SELECT (NOT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_autorizacion' AND (has_table_privilege('$rol',c.oid,'SELECT') OR has_table_privilege('$rol',c.oid,'INSERT') OR has_table_privilege('$rol',c.oid,'UPDATE') OR has_table_privilege('$rol',c.oid,'DELETE'))) AND NOT EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_autorizacion' AND has_function_privilege('$rol',p.oid,'EXECUTE')))::text") == true ]]
  [[ $(psql_valor "SELECT (NOT EXISTS(SELECT 1 FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname='vec_autorizacion' AND has_type_privilege('$rol',t.oid,'USAGE')) AND NOT EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a WHERE a.grantee='$rol'::regrole OR a.grantor='$rol'::regrole) AND NOT EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(c.relacl) a WHERE a.grantee='$rol'::regrole OR a.grantor='$rol'::regrole) AND NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE a.grantee='$rol'::regrole OR a.grantor='$rol'::regrole) AND NOT EXISTS(SELECT 1 FROM pg_type t CROSS JOIN LATERAL aclexplode(t.typacl) a WHERE a.grantee='$rol'::regrole OR a.grantor='$rol'::regrole))::text") == true ]]
}

psql_sql <<SQL
CREATE ROLE vec_m1r_dueno_base NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  NOINHERIT NOREPLICATION NOBYPASSRLS;
ALTER DATABASE $base OWNER TO vec_m1r_dueno_base;
DO \$base\$ BEGIN
  EXECUTE format('REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
                 current_database());
END \$base\$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
for archivo in \
  deploy/postgresql/contexto_actor_v1/roles_up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql \
  deploy/postgresql/autorizacion/roles_up.sql \
  deploy/postgresql/autorizacion/roles_v2_up.sql \
  deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql \
  deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
  deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql \
  deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql \
  deploy/postgresql/autorizacion/migraciones/000005_registro_decisiones_contexto_actor_v3.up.sql \
  deploy/postgresql/autorizacion/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.up.sql \
  deploy/postgresql/autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql \
  deploy/postgresql/autorizacion/migraciones/000008_vinculaciones_motivo_consultas_rrhh.up.sql \
  deploy/postgresql/autorizacion/migraciones/000009_publicacion_retirada_vinculaciones_motivo_consultas_rrhh.up.sql
do
  psql_archivo "$archivo" >/dev/null
done
psql_valor "CREATE ROLE vec_m1r_operador NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS" >/dev/null

huella_v2_inicial=$(huella_v2)

# Ejecutor no superusuario y homonimos se rechazan sin adopcion.
if psql_archivo_como vec_m1r_operador "$up" >/dev/null 2>&1; then
  echo 'un no-superusuario instaló el rol' >&2
  exit 1
fi
exigir_ausencia

psql_sql <<SQL
CREATE ROLE $rol NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT
  NOREPLICATION NOBYPASSRLS;
COMMENT ON ROLE $rol IS 'vec_autorizacion:rol-motivos-rrhh-resolutor:v1';
GRANT CONNECT ON DATABASE $base TO $rol;
SQL
huella_homonimo=$(huella_objetivo)
exigir_fallo_up 'homónimo exacto'
[[ $(huella_objetivo) == "$huella_homonimo" ]]
psql_valor "REVOKE CONNECT ON DATABASE $base FROM $rol; DROP ROLE $rol" >/dev/null

psql_valor "CREATE ROLE $rol LOGIN SUPERUSER CREATEDB CREATEROLE INHERIT REPLICATION BYPASSRLS" >/dev/null
exigir_fallo_up 'homónimo hostil'
[[ $(psql_valor "SELECT (rolcanlogin AND rolsuper AND rolcreatedb AND rolcreaterole AND rolreplication AND rolbypassrls)::text FROM pg_authid WHERE rolname='$rol'") == true ]]
psql_valor "DROP ROLE $rol" >/dev/null

# Dos altas concurrentes producen un único rol y un único éxito.
set +e
(psql_archivo "$up" >"$salidas/up_a" 2>&1) & pid_a=$!
(psql_archivo "$up" >"$salidas/up_b" 2>&1) & pid_b=$!
wait "$pid_a"; estado_a=$?
wait "$pid_b"; estado_b=$?
set -e
[[ $((estado_a + estado_b)) -ne 0 ]]
[[ $(grep -lF 'alta del resolutor RRHH rechazada' "$salidas/up_a" "$salidas/up_b" | wc -l) == 1 ]]
exigir_rol
exigir_connect
exigir_aislamiento
[[ $(huella_v2) == "$huella_v2_inicial" ]]

# Reentrada y retirada por no-superusuario no modifican la huella.
huella_instalada=$(huella_objetivo)
exigir_fallo_up 'reentrada'
[[ $(huella_objetivo) == "$huella_instalada" ]]
if psql_archivo_como vec_m1r_operador "$down" >/dev/null 2>&1; then
  echo 'un no-superusuario retiró el rol' >&2
  exit 1
fi
[[ $(huella_objetivo) == "$huella_instalada" ]]

# Cada atributo se altera y restaura como una familia aislada.
while IFS='|' read -r mutacion restauracion caso; do
  psql_valor "ALTER ROLE $rol $mutacion" >/dev/null
  exigir_fallo_down "$caso"
  exigir_connect
  psql_valor "ALTER ROLE $rol $restauracion" >/dev/null
done <<'ATRIBUTOS'
LOGIN|NOLOGIN|LOGIN
SUPERUSER|NOSUPERUSER|SUPERUSER
CREATEDB|NOCREATEDB|CREATEDB
CREATEROLE|NOCREATEROLE|CREATEROLE
INHERIT|NOINHERIT|INHERIT
REPLICATION|NOREPLICATION|REPLICATION
BYPASSRLS|NOBYPASSRLS|BYPASSRLS
CONNECTION LIMIT 2|CONNECTION LIMIT -1|límite de conexiones
ATRIBUTOS

psql_valor "COMMENT ON ROLE $rol IS 'marca hostil'" >/dev/null
exigir_fallo_down 'comentario'
exigir_connect
psql_valor "COMMENT ON ROLE $rol IS 'vec_autorizacion:rol-motivos-rrhh-resolutor:v1'" >/dev/null

docker exec --interactive --env CLAVE="$clave" "$contenedor" \
  psql -Xq --set ON_ERROR_STOP=1 -U postgres -d "$base" <<SQL
\getenv clave CLAVE
ALTER ROLE $rol PASSWORD :'clave';
SQL
exigir_fallo_down 'contraseña'
exigir_connect
psql_valor "ALTER ROLE $rol PASSWORD NULL" >/dev/null

psql_valor "ALTER ROLE $rol VALID UNTIL 'infinity'" >/dev/null
exigir_fallo_down 'caducidad'
exigir_connect
psql_valor "REVOKE CONNECT ON DATABASE $base FROM $rol; DROP ROLE $rol" >/dev/null
psql_archivo "$up" >/dev/null

psql_valor "ALTER ROLE $rol SET application_name='ajuste_hostil'" >/dev/null
exigir_fallo_down 'ajuste'
exigir_connect
psql_valor "ALTER ROLE $rol RESET application_name" >/dev/null

# ACL adicional en base, esquema, funcion y otra base siempre falla cerrado.
psql_valor "GRANT CREATE ON DATABASE $base TO $rol" >/dev/null
exigir_fallo_down 'ACL de base'
exigir_connect
psql_valor "REVOKE CREATE ON DATABASE $base FROM $rol" >/dev/null

psql_valor "GRANT USAGE ON SCHEMA vec_autorizacion TO $rol" >/dev/null
exigir_fallo_down 'ACL de esquema'
exigir_connect
[[ $(psql_valor "SELECT has_schema_privilege('$rol','vec_autorizacion','USAGE')::text") == true ]]
psql_valor "REVOKE USAGE ON SCHEMA vec_autorizacion FROM $rol" >/dev/null

psql_valor "GRANT EXECUTE ON FUNCTION vec_autorizacion.obtener_instantanea(text,text) TO $rol" >/dev/null
exigir_fallo_down 'ACL de función'
exigir_connect
[[ $(psql_valor "SELECT has_function_privilege('$rol','vec_autorizacion.obtener_instantanea(text,text)','EXECUTE')::text") == true ]]
psql_valor "REVOKE EXECUTE ON FUNCTION vec_autorizacion.obtener_instantanea(text,text) FROM $rol" >/dev/null

psql_valor "CREATE DATABASE vec_m1r_ajena" >/dev/null
psql_valor "GRANT CONNECT ON DATABASE vec_m1r_ajena TO $rol" >/dev/null
exigir_fallo_down 'ACL en otra base'
exigir_connect
[[ $(psql_valor "SELECT has_database_privilege('$rol','vec_m1r_ajena','CONNECT')::text") == true ]]
psql_valor "REVOKE CONNECT ON DATABASE vec_m1r_ajena FROM $rol" >/dev/null
psql_valor "DROP DATABASE vec_m1r_ajena" >/dev/null

# Las tres coordenadas de membresia se rechazan por separado.
psql_valor "CREATE ROLE vec_m1r_grupo NOLOGIN; CREATE ROLE vec_m1r_miembro NOLOGIN" >/dev/null
psql_valor "GRANT $rol TO vec_m1r_miembro" >/dev/null
exigir_fallo_down 'rol como grupo'
exigir_connect
psql_valor "REVOKE $rol FROM vec_m1r_miembro" >/dev/null

psql_valor "GRANT vec_m1r_grupo TO $rol" >/dev/null
exigir_fallo_down 'rol como miembro'
exigir_connect
psql_valor "REVOKE vec_m1r_grupo FROM $rol" >/dev/null

psql_sql <<SQL
GRANT vec_m1r_grupo TO $rol WITH ADMIN OPTION;
SET ROLE $rol;
GRANT vec_m1r_grupo TO vec_m1r_miembro;
RESET ROLE;
SQL
[[ $(psql_valor "SELECT (count(*)=1)::text FROM pg_auth_members WHERE grantor='$rol'::regrole") == true ]]
exigir_fallo_down 'rol como otorgante'
exigir_connect
psql_sql <<SQL
SET ROLE $rol;
REVOKE vec_m1r_grupo FROM vec_m1r_miembro;
RESET ROLE;
REVOKE vec_m1r_grupo FROM $rol;
SQL

# Las fachadas M1.3 exactas y cualquier sobrecarga bloquean la retirada.
psql_sql <<'SQL'
SET ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)
RETURNS text LANGUAGE sql AS 'SELECT NULL::text';
CREATE FUNCTION vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)
RETURNS text LANGUAGE sql AS 'SELECT NULL::text';
SQL
exigir_fallo_down 'fachadas M1.3'
exigir_connect
psql_valor "SET ROLE vec_autorizacion_propietario; DROP FUNCTION vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz),vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)" >/dev/null

psql_valor "SET ROLE vec_autorizacion_propietario; CREATE FUNCTION vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(integer) RETURNS text LANGUAGE sql AS 'SELECT NULL::text'" >/dev/null
exigir_fallo_down 'sobrecarga M1.3'
exigir_connect
psql_valor "SET ROLE vec_autorizacion_propietario; DROP FUNCTION vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(integer)" >/dev/null

# Propiedad y dependencia SQL real no se revocan ni eliminan.
psql_valor "CREATE TABLE public.propiedad_m1r(id integer); ALTER TABLE public.propiedad_m1r OWNER TO $rol" >/dev/null
exigir_fallo_down 'propiedad ajena'
exigir_connect
[[ $(psql_valor "SELECT (relowner='$rol'::regrole)::text FROM pg_class WHERE oid='public.propiedad_m1r'::regclass") == true ]]
psql_valor "ALTER TABLE public.propiedad_m1r OWNER TO postgres; DROP TABLE public.propiedad_m1r" >/dev/null

psql_valor "SET ROLE vec_autorizacion_propietario; CREATE POLICY dependencia_m1r ON vec_autorizacion.motivo_v2_evento_origen FOR SELECT TO $rol USING (false)" >/dev/null
exigir_fallo_down 'dependencia de política'
exigir_connect
[[ $(psql_valor "SELECT (count(*)=1)::text FROM pg_policy WHERE polname='dependencia_m1r'") == true ]]
psql_valor "SET ROLE vec_autorizacion_propietario; DROP POLICY dependencia_m1r ON vec_autorizacion.motivo_v2_evento_origen" >/dev/null

# Carrera causal down/GRANT: down posee los dos catalogos de roles mientras
# espera pg_database; el GRANT queda bloqueado por ese PID y termina 42704.
fifo_barrera="$salidas/fifo_barrera"
fifo_grant="$salidas/fifo_grant"
fifo_down="$salidas/fifo_down"
mkfifo "$fifo_barrera" "$fifo_grant" "$fifo_down"
docker exec --interactive "$contenedor" psql -XAtq --set ON_ERROR_STOP=1 \
  -U postgres -d "$base" <"$fifo_barrera" >"$salidas/sesion_barrera" 2>&1 &
pid_barrera=$!
exec 7>"$fifo_barrera"
cat >&7 <<'SQL'
SET application_name='ct60_barrera_base';
\! touch /tmp/ct60_barrera_conectada
SQL
docker exec --interactive "$contenedor" psql -XAtq --set ON_ERROR_STOP=1 \
  --set VERBOSITY=verbose \
  -U postgres -d "$base" <"$fifo_grant" >"$salidas/sesion_grant" 2>&1 &
pid_grant=$!
exec 9>"$fifo_grant"
cat >&9 <<'SQL'
SET application_name='ct60_grant_pendiente';
\! touch /tmp/ct60_grant_listo
SQL
docker exec --interactive "$contenedor" psql -Xq --set ON_ERROR_STOP=1 \
  -U postgres -d "$base" <"$fifo_down" >"$salidas/down_grant" 2>&1 &
pid_down_grant=$!
exec 6>"$fifo_down"
cat >&6 <<'SQL'
SET application_name='ct60_down_grant';
\! touch /tmp/ct60_down_conectado
SQL
for marca in /tmp/ct60_barrera_conectada /tmp/ct60_grant_listo \
  /tmp/ct60_down_conectado
do
  for _ in $(seq 1 100); do
    docker exec "$contenedor" test -f "$marca" && break
    sleep 0.05
  done
  docker exec "$contenedor" test -f "$marca"
done
oid_antes_carrera=$(psql_valor "SELECT '$rol'::regrole::oid")
cat >&7 <<'SQL'
BEGIN;
LOCK TABLE pg_catalog.pg_database IN ACCESS SHARE MODE;
\! touch /tmp/ct60_barrera_lista
SQL
for _ in $(seq 1 100); do
  docker exec "$contenedor" test -f /tmp/ct60_barrera_lista && break
  sleep 0.05
done
docker exec "$contenedor" test -f /tmp/ct60_barrera_lista
cat "$raiz/$down" >&6
exec 6>&-
down_preparado=false
for intento in $(seq 1 100); do
  ruta="/tmp/ct60_down_preparado_$intento"
  printf '\\o %s\n' "$ruta" >&7
  cat >&7 <<'SQL'
WITH actividad AS MATERIALIZED (
  SELECT * FROM pg_stat_get_activity(NULL::integer)
), bloqueos AS MATERIALIZED (
  SELECT * FROM pg_lock_status()
)
SELECT concat(
  (SELECT count(*) FROM actividad
    WHERE application_name='ct60_down_grant'),
  '|',
  COALESCE((SELECT wait_event_type FROM actividad
    WHERE application_name='ct60_down_grant'),'<nulo>'),
  '|',
  COALESCE((SELECT cardinality(pg_blocking_pids(pid)) FROM actividad
    WHERE application_name='ct60_down_grant'),0),
  '|',
  COALESCE((SELECT count(*) FROM bloqueos
    WHERE pid=(SELECT pid FROM actividad
      WHERE application_name='ct60_down_grant')
      AND granted AND mode='AccessExclusiveLock'
      AND relation=ANY(ARRAY[
        'pg_catalog.pg_authid'::regclass::oid,
        'pg_catalog.pg_auth_members'::regclass::oid])),0),
  '|',
  COALESCE((SELECT count(*) FROM bloqueos
    WHERE pid=(SELECT pid FROM actividad
      WHERE application_name='ct60_down_grant')
      AND NOT granted AND mode='AccessExclusiveLock'
      AND relation='pg_catalog.pg_database'::regclass::oid),0));
\o
SQL
  for _ in $(seq 1 20); do
    docker exec "$contenedor" test -s "$ruta" && break
    sleep 0.01
  done
  estado_down=$(docker exec "$contenedor" sh -c "cat '$ruta' 2>/dev/null || true")
  if [[ $estado_down == '1|Lock|1|2|1' ]]; then
    down_preparado=true
    break
  fi
  sleep 0.05
done
[[ $down_preparado == true ]] || {
  echo "estado causal inesperado del down: ${estado_down:-vacío}" >&2
  exit 1
}
printf 'GRANT %s TO vec_m1r_miembro;\n' "$rol" >&9
exec 9>&-
grant_bloqueado=false
for intento in $(seq 1 100); do
  ruta="/tmp/ct60_grant_bloqueado_$intento"
  printf '\\o %s\n' "$ruta" >&7
  cat >&7 <<'SQL'
WITH actividad AS MATERIALIZED (
  SELECT * FROM pg_stat_get_activity(NULL::integer)
)
SELECT (count(*)=1)::text
  FROM actividad AS concesion
  JOIN actividad AS retirada
    ON retirada.application_name='ct60_down_grant'
 WHERE concesion.application_name='ct60_grant_pendiente'
   AND concesion.wait_event_type='Lock'
   AND retirada.pid=ANY(pg_blocking_pids(concesion.pid));
\o
SQL
  for _ in $(seq 1 20); do
    docker exec "$contenedor" test -s "$ruta" && break
    sleep 0.01
  done
  if [[ $(docker exec "$contenedor" sh -c "cat '$ruta' 2>/dev/null || true") == true ]]; then
    grant_bloqueado=true
    break
  fi
  sleep 0.05
done
[[ $grant_bloqueado == true ]]
printf 'COMMIT;\n' >&7
exec 7>&-
wait "$pid_barrera"
wait "$pid_down_grant"
if wait "$pid_grant"; then
  echo 'el GRANT concurrente sobrevivió a la retirada' >&2
  exit 1
fi
grep -Fq '42704' "$salidas/sesion_grant"
exigir_ausencia_oid "$oid_antes_carrera"
psql_archivo "$up" >/dev/null
exigir_rol
exigir_connect

# Carrera causal 000010/down: advisory compartido y fachada real.
fifo_m13="$salidas/fifo_m13"
mkfifo "$fifo_m13"
docker exec --interactive "$contenedor" psql -Xq --set ON_ERROR_STOP=1 \
  -U postgres -d "$base" <"$fifo_m13" >"$salidas/sesion_m13" 2>&1 &
pid_m13=$!
exec 8>"$fifo_m13"
cat >&8 <<'SQL'
SET application_name='ct60_m13_abierto';
BEGIN;
SELECT pg_advisory_xact_lock_shared(
  hashtextextended('vec_autorizacion:rol-motivos-rrhh-resolutor:v1',0));
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)
RETURNS text LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog
AS 'SELECT NULL::text';
SQL
m13_abierto=false
for _ in $(seq 1 100); do
  if [[ $(psql_valor "SELECT count(*) FROM pg_stat_activity WHERE application_name='ct60_m13_abierto' AND state='idle in transaction'") == 1 ]]; then
    m13_abierto=true
    break
  fi
  sleep 0.05
done
[[ $m13_abierto == true ]]
(docker exec --interactive --env PGAPPNAME=ct60_down_m13 \
  "$contenedor" psql -Xq --set ON_ERROR_STOP=1 -U postgres -d "$base" \
  <"$raiz/$down" >"$salidas/down_m13" 2>&1) &
pid_down_m13=$!
advisory_observado=false
for _ in $(seq 1 100); do
  if [[ $(psql_valor "SELECT count(*) FROM pg_stat_activity WHERE application_name='ct60_down_m13' AND wait_event='advisory'") == 1 ]]; then
    advisory_observado=true
    break
  fi
  sleep 0.05
done
[[ $advisory_observado == true ]]
printf 'COMMIT;\n' >&8
exec 8>&-
wait "$pid_m13"
if wait "$pid_down_m13"; then
  echo 'la carrera 000010/down retiró el rol' >&2
  exit 1
fi
exigir_rol
exigir_connect
[[ $(psql_valor "SELECT (to_regprocedure('vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)') IS NOT NULL)::text") == true ]]
psql_valor "SET ROLE vec_autorizacion_propietario; DROP FUNCTION vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)" >/dev/null

# Ciclo limpio, segunda retirada y reinstalacion.
oid_ciclo=$(psql_valor "SELECT '$rol'::regrole::oid")
psql_archivo "$down" >/dev/null
exigir_ausencia_oid "$oid_ciclo"
if psql_archivo "$down" >/dev/null 2>&1; then
  echo 'la segunda retirada fue aceptada' >&2
  exit 1
fi
exigir_ausencia_oid "$oid_ciclo"
psql_archivo "$up" >/dev/null
exigir_rol
exigir_connect
exigir_aislamiento
[[ $(huella_v2) == "$huella_v2_inicial" ]]
oid_final=$(psql_valor "SELECT '$rol'::regrole::oid")
psql_archivo "$down" >/dev/null
exigir_ausencia_oid "$oid_final"

psql_valor "DROP ROLE vec_m1r_miembro; DROP ROLE vec_m1r_grupo; DROP ROLE vec_m1r_operador" >/dev/null
psql_valor "ALTER DATABASE $base OWNER TO postgres" >/dev/null
psql_valor "DROP ROLE vec_m1r_dueno_base" >/dev/null

echo 'OK: rol resolutor nominal de motivos RRHH en PostgreSQL 18.4'
