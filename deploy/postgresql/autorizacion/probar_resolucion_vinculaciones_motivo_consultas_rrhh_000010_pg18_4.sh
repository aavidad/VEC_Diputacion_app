#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-autorizacion-resolucion-rrhh-000010-${USER:-usuario}-$$"
base=vec_autorizacion_resolucion_rrhh_000010
clave=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
salidas=$(mktemp -d)
up=deploy/postgresql/autorizacion/migraciones/000010_resolucion_vinculaciones_motivo_consultas_rrhh.up.sql
down=deploy/postgresql/autorizacion/migraciones/000010_resolucion_vinculaciones_motivo_consultas_rrhh.down.sql
migrador=vec_m13_migrador
proyector=vec_m13_proyector
resolutor=vec_m13_resolutor
ajeno=vec_m13_ajeno

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
    --set ON_ERROR_STOP=1 -U postgres -d "$base" < "$raiz/$1"
}
psql_valor() {
  docker exec "$contenedor" psql -XAtq --set ON_ERROR_STOP=1 \
    -U postgres -d "$base" -c "$1"
}
psql_admin() {
  docker exec --interactive "$contenedor" psql -Xq \
    --set ON_ERROR_STOP=1 -U postgres -d "$base"
}
actor_valor() {
  local actor=$1 consulta=$2
  docker exec --env PGPASSWORD="$clave" "$contenedor" psql -XAtq \
    --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
    -h 127.0.0.1 -U "$actor" -d "$base" -c "$consulta"
}
migracion() {
  docker exec --interactive --env PGPASSWORD="$clave" "$contenedor" psql -Xq \
    --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
    -h 127.0.0.1 -U "$migrador" -d "$base"
}
numero_fachadas() {
  psql_valor "SELECT count(*) FROM pg_proc WHERE pronamespace='vec_autorizacion'::regnamespace AND proname=ANY(ARRAY['resolver_motivo_cuadro_rrhh_v1','resolver_motivo_detalle_rrhh_v1']::name[])"
}
estado_instalado() {
  [[ $(psql_valor "SELECT (count(*)=2 AND has_schema_privilege('vec_autorizacion_motivos_rrhh_resolutor','vec_autorizacion','USAGE') AND bool_and(has_function_privilege('vec_autorizacion_motivos_rrhh_resolutor',oid,'EXECUTE')))::text FROM pg_proc WHERE pronamespace='vec_autorizacion'::regnamespace AND proname=ANY(ARRAY['resolver_motivo_cuadro_rrhh_v1','resolver_motivo_detalle_rrhh_v1']::name[])") == true ]]
}
estado_retirado() {
  [[ $(numero_fachadas) == 0 ]]
  [[ $(psql_valor "SELECT (has_database_privilege('vec_autorizacion_motivos_rrhh_resolutor',current_database(),'CONNECT') AND NOT has_schema_privilege('vec_autorizacion_motivos_rrhh_resolutor','vec_autorizacion','USAGE'))::text") == true ]]
}
exigir_fallo_up() {
  local presentes=${2:-0}
  if migracion < "$raiz/$up" >"$salidas/fallo_up" 2>&1; then
    echo "000010 up aceptó estado hostil: $1" >&2
    exit 1
  fi
  [[ $(numero_fachadas) == "$presentes" ]]
}
exigir_fallo_down() {
  if migracion < "$raiz/$down" >"$salidas/fallo_down" 2>&1; then
    echo "000010 down aceptó estado hostil: $1" >&2
    exit 1
  fi
  estado_instalado
}
exigir_denegado() {
  local actor=$1 consulta=$2 caso=$3
  if actor_valor "$actor" "$consulta" >"$salidas/denegado" 2>&1; then
    echo "acceso indebido: $caso" >&2
    exit 1
  fi
  grep -Eq '42501|permission denied' "$salidas/denegado"
}
huella_fundamentos() {
  {
    docker exec "$contenedor" pg_dump -U postgres -d "$base" --schema-only \
      -t vec_autorizacion.motivo_v2_evento_origen \
      -t vec_autorizacion.motivo_v2_catalogo_publicado \
      -t vec_autorizacion.motivo_v2_entrada \
      -t vec_autorizacion.motivo_v2_retirada \
      -t vec_autorizacion.motivo_v2_checkpoint_origen \
      -t vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 \
      -t vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1 \
      -t vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1
    psql_valor "SELECT p.oid::regprocedure::text,pg_get_functiondef(p.oid),COALESCE(p.proacl::text,'') FROM pg_proc p WHERE p.pronamespace='vec_autorizacion'::regnamespace AND p.proname=ANY(ARRAY['bloquear_mutacion_vinculacion_motivo_rrhh_v1','validar_avance_vinculacion_motivo_rrhh_v1','bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1','validar_insercion_vinculacion_motivo_rrhh_evento_v1','registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1','registrar_retirada_vinculacion_motivo_consulta_rrhh_v1','publicar_vinculacion_motivo_cuadro_rrhh_v1','publicar_vinculacion_motivo_detalle_rrhh_v1','retirar_vinculacion_motivo_cuadro_rrhh_v1','retirar_vinculacion_motivo_detalle_rrhh_v1']::name[]) ORDER BY 1"
  } | sed '/^\\restrict /d;/^\\unrestrict /d' | sha256sum | cut -d' ' -f1
}
huella_fachadas() {
  psql_valor "SELECT encode(sha256(convert_to(string_agg(pg_get_functiondef(oid)||COALESCE(proacl::text,''),E'\\n' ORDER BY oid::regprocedure::text),'UTF8')),'hex') FROM pg_proc WHERE pronamespace='vec_autorizacion'::regnamespace AND proname=ANY(ARRAY['resolver_motivo_cuadro_rrhh_v1','resolver_motivo_detalle_rrhh_v1']::name[])"
}
huella_evidencia() {
  docker exec "$contenedor" pg_dump -U postgres -d "$base" --data-only \
    -t vec_autorizacion.motivo_v2_evento_origen \
    -t vec_autorizacion.motivo_v2_catalogo_publicado \
    -t vec_autorizacion.motivo_v2_entrada \
    -t vec_autorizacion.motivo_v2_retirada \
    -t vec_autorizacion.motivo_v2_checkpoint_origen \
    -t vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 \
    -t vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1 \
    -t vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 |
    sed '/^\\restrict /d;/^\\unrestrict /d' | sha256sum | cut -d' ' -f1
}

psql_admin <<'SQL'
DO $b$ BEGIN
  EXECUTE format('REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
                 current_database());
END $b$;
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

psql_admin <<SQL
CREATE ROLE $migrador LOGIN PASSWORD '$clave' NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT CONNECT ON DATABASE $base TO $migrador;
GRANT vec_autorizacion_migrador TO $migrador
  WITH ADMIN FALSE,INHERIT TRUE,SET TRUE;
SQL

# Sin M1.R, el migrador mínimo no puede adoptar ni crear el rol nominal.
exigir_fallo_up 'M1.R ausente'
psql_archivo deploy/postgresql/autorizacion/roles_motivos_rrhh_resolutor_v1_up.sql \
  >/dev/null
psql_admin <<SQL
CREATE ROLE $proyector LOGIN PASSWORD '$clave' NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE $resolutor LOGIN PASSWORD '$clave' NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE $ajeno LOGIN PASSWORD '$clave' NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT CONNECT ON DATABASE $base TO $proyector,$resolutor,$ajeno;
GRANT vec_autorizacion_motivos_proyector TO $proyector
  WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
GRANT vec_autorizacion_motivos_rrhh_resolutor TO $resolutor
  WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL

huella_base=$(huella_fundamentos)

# Deriva previa: columnas, restricciones, triggers, RLS, funciones, ACL y rol.
psql_valor "SET ROLE vec_autorizacion_propietario; ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 ALTER COLUMN catalogo_id DROP NOT NULL" >/dev/null
exigir_fallo_up 'columna sin NOT NULL'
psql_valor "SET ROLE vec_autorizacion_propietario; ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 ALTER COLUMN catalogo_id SET NOT NULL" >/dev/null
psql_valor "SET ROLE vec_autorizacion_propietario; ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1 ADD CONSTRAINT m13_extra CHECK (true)" >/dev/null
exigir_fallo_up 'restricción adicional'
psql_valor "SET ROLE vec_autorizacion_propietario; ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1 DROP CONSTRAINT m13_extra" >/dev/null
psql_valor "SET ROLE vec_autorizacion_propietario; CREATE TRIGGER m13_extra BEFORE UPDATE ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1()" >/dev/null
exigir_fallo_up 'trigger adicional'
psql_valor "SET ROLE vec_autorizacion_propietario; DROP TRIGGER m13_extra ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1" >/dev/null
psql_valor "SET ROLE vec_autorizacion_propietario; ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 DISABLE ROW LEVEL SECURITY" >/dev/null
exigir_fallo_up 'RLS deshabilitado'
psql_valor "SET ROLE vec_autorizacion_propietario; ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 ENABLE ROW LEVEL SECURITY" >/dev/null
psql_valor "SET ROLE vec_autorizacion_propietario; ALTER POLICY acceso_propietario_exacto ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 USING (true) WITH CHECK (true)" >/dev/null
exigir_fallo_up 'política RLS alterada'
psql_valor "SET ROLE vec_autorizacion_propietario; ALTER POLICY acceso_propietario_exacto ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 TO vec_autorizacion_propietario USING (current_user='vec_autorizacion_propietario') WITH CHECK (current_user='vec_autorizacion_propietario')" >/dev/null
psql_valor "SET ROLE vec_autorizacion_propietario; ALTER FUNCTION vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamptz) SET search_path TO public" >/dev/null
exigir_fallo_up 'función fundamental alterada'
psql_valor "SET ROLE vec_autorizacion_propietario; ALTER FUNCTION vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamptz) SET search_path TO pg_catalog" >/dev/null
psql_valor "SET ROLE vec_autorizacion_propietario; GRANT SELECT ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 TO vec_autorizacion_motivos_rrhh_resolutor" >/dev/null
exigir_fallo_up 'ACL de tabla'
psql_valor "SET ROLE vec_autorizacion_propietario; REVOKE SELECT ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 FROM vec_autorizacion_motivos_rrhh_resolutor" >/dev/null
psql_valor "ALTER ROLE vec_autorizacion_motivos_rrhh_resolutor INHERIT" >/dev/null
exigir_fallo_up 'rol nominal alterado'
psql_valor "ALTER ROLE vec_autorizacion_motivos_rrhh_resolutor NOINHERIT" >/dev/null

psql_admin <<'SQL'
SET ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)
RETURNS TABLE(catalogo_id text,catalogo_version integer,
  catalogo_huella_sha256 text,entrada_clave text)
LANGUAGE sql AS 'SELECT NULL::text,NULL::integer,NULL::text,NULL::text';
SQL
exigir_fallo_up 'homónimo exacto' 1
psql_valor "SET ROLE vec_autorizacion_propietario; DROP FUNCTION vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)" >/dev/null
psql_valor "SET ROLE vec_autorizacion_propietario; CREATE FUNCTION vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(integer) RETURNS boolean LANGUAGE sql AS 'SELECT false'" >/dev/null
exigir_fallo_up 'sobrecarga' 1
psql_valor "SET ROLE vec_autorizacion_propietario; DROP FUNCTION vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(integer)" >/dev/null

# Instalación, reentrada estable y ciclo reversible por LOGIN migrador.
migracion < "$raiz/$up" >/dev/null
estado_instalado
[[ $(psql_valor "SELECT (count(*)=2 AND bool_and(proowner='vec_autorizacion_propietario'::regrole AND prosecdef AND proretset AND proconfig=ARRAY['search_path=pg_catalog']))::text FROM pg_proc WHERE pronamespace='vec_autorizacion'::regnamespace AND proname=ANY(ARRAY['resolver_motivo_cuadro_rrhh_v1','resolver_motivo_detalle_rrhh_v1']::name[])") == true ]]
huella_instalada=$(huella_fachadas)
if migracion < "$raiz/$up" >/dev/null 2>&1; then
  echo '000010 aceptó reentrada' >&2
  exit 1
fi
[[ $(huella_fachadas) == "$huella_instalada" ]]
migracion < "$raiz/$down" >/dev/null
estado_retirado
[[ $(huella_fundamentos) == "$huella_base" ]]
migracion < "$raiz/$up" >/dev/null

# Dos motivos distintos y una entrada de vigencia breve para probar su caducidad.
[[ $(actor_valor "$resolutor" "SELECT (SELECT count(*) FROM vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(clock_timestamp()))+(SELECT count(*) FROM vec_autorizacion.resolver_motivo_detalle_rrhh_v1(clock_timestamp()))") == 0 ]]
[[ $(actor_valor "$proyector" "SELECT vec_autorizacion.publicar_motivos_autorizacion_v2('evento_11111111111111111111111111111111',1,repeat('1',64),'motivos_rrhh_m13',1,repeat('2',64),clock_timestamp()-interval '31 days',jsonb_build_array(jsonb_build_object('clave','motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','vigente_desde',to_char((clock_timestamp()-interval '30 days') AT TIME ZONE 'UTC','YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"'),'vigente_hasta',NULL),jsonb_build_object('clave','motivo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb','vigente_desde',to_char((clock_timestamp()-interval '30 days') AT TIME ZONE 'UTC','YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"'),'vigente_hasta',NULL)))") == t ]]
publicar_cuadro="SELECT vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1('evento_vinculacion_motivo_rrhh_11111111111111111111111111111111',repeat('3',64),1,'publicacion_motivo_rrhh_11111111111111111111111111111111',repeat('4',64),'motivos_rrhh_m13',1,repeat('2',64),'motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',clock_timestamp()-interval '20 days')"
publicar_detalle="SELECT vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1('evento_vinculacion_motivo_rrhh_22222222222222222222222222222222',repeat('5',64),1,'publicacion_motivo_rrhh_22222222222222222222222222222222',repeat('6',64),'motivos_rrhh_m13',1,repeat('2',64),'motivo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',clock_timestamp()-interval '19 days')"
[[ $(actor_valor "$proyector" "$publicar_cuadro") == t ]]
[[ $(actor_valor "$proyector" "$publicar_detalle") == t ]]
esperado_c="motivos_rrhh_m13|1|$(printf '2%.0s' {1..64})|motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
esperado_d="motivos_rrhh_m13|1|$(printf '2%.0s' {1..64})|motivo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
resolver_c="SELECT catalogo_id,catalogo_version,catalogo_huella_sha256,entrada_clave FROM vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(clock_timestamp()-interval '1 minute')"
resolver_d="SELECT catalogo_id,catalogo_version,catalogo_huella_sha256,entrada_clave FROM vec_autorizacion.resolver_motivo_detalle_rrhh_v1(clock_timestamp()-interval '1 minute')"
huella_antes_resolver=$(huella_evidencia)
[[ $(actor_valor "$resolutor" "$resolver_c") == "$esperado_c" ]]
[[ $(actor_valor "$resolutor" "$resolver_d") == "$esperado_d" ]]
[[ $(huella_evidencia) == "$huella_antes_resolver" ]]

# Valores temporales tipables: vacío uniforme; timestamptz ya canoniza a microsegundos.
for instante in \
  "NULL::timestamptz" "'infinity'::timestamptz" "'-infinity'::timestamptz" \
  "'0001-01-01 BC'::timestamptz" "'10000-01-01'::timestamptz" \
  "clock_timestamp()+interval '1 day'" "clock_timestamp()-interval '25 days'"
do
  [[ $(actor_valor "$resolutor" "SELECT count(*) FROM vec_autorizacion.resolver_motivo_cuadro_rrhh_v1($instante)") == 0 ]]
done
[[ $(psql_valor "SELECT ('2026-01-01T00:00:00.1234567Z'::timestamptz=date_trunc('microseconds','2026-01-01T00:00:00.1234567Z'::timestamptz))::text") == true ]]

# ACL efectiva: solo el pool resolutor; ningún acceso directo ni SET ROLE.
[[ $(psql_valor "SELECT (NOT has_function_privilege('vec_autorizacion_fuente','vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)','EXECUTE') AND NOT has_function_privilege('vec_autorizacion_registro','vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)','EXECUTE') AND NOT has_function_privilege('vec_autorizacion_motivos_proyector','vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)','EXECUTE') AND NOT has_function_privilege('vec_autorizacion_motivos_evaluador','vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)','EXECUTE'))::text") == true ]]
[[ $(psql_valor "SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.pronamespace='vec_autorizacion'::regnamespace AND p.proname=ANY(ARRAY['resolver_motivo_cuadro_rrhh_v1','resolver_motivo_detalle_rrhh_v1']::name[]) AND a.grantee=0") == 0 ]]
exigir_denegado "$proyector" "SELECT * FROM vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(clock_timestamp())" 'proyector'
exigir_denegado "$ajeno" "SELECT * FROM vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(clock_timestamp())" 'LOGIN ajeno'
for orden in \
  "SELECT * FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1" \
  "INSERT INTO vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1(clase_consulta,publicacion_version) VALUES('cuadro',99)" \
  "UPDATE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 SET catalogo_id=catalogo_id" \
  "DELETE FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1" \
  "TRUNCATE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1"
do
  exigir_denegado "$resolutor" "$orden" 'lectura/DML directo'
done
exigir_denegado "$resolutor" "SET ROLE vec_autorizacion_propietario" 'SET ROLE'

# search_path y homónimos pg_temp no alteran la resolución cualificada.
resultado_hostil=$(docker exec --interactive "$contenedor" psql -XAtq \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<SQL
CREATE TEMP TABLE activar_pg_temp(i integer);
CREATE FUNCTION pg_temp.resolver_motivo_cuadro_rrhh_v1(timestamptz)
RETURNS TABLE(catalogo_id text,catalogo_version integer,
  catalogo_huella_sha256 text,entrada_clave text)
LANGUAGE sql AS 'SELECT ''hostil'',0,''hostil'',''hostil''';
SET search_path=pg_temp,public;
SET SESSION AUTHORIZATION $resolutor;
$resolver_c;
SQL
)
[[ $resultado_hostil == "$esperado_c" ]]

# Pool directo exacto: grantor OID 10 positivo; grantor distinto y puente fallan.
[[ $(psql_valor "SELECT (count(*)=1 AND bool_and(m.grantor=10 AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option))::text FROM pg_auth_members m JOIN pg_roles r ON r.oid=m.member WHERE r.rolname='$resolutor'") == true ]]
psql_valor "GRANT vec_autorizacion_motivos_evaluador TO $resolutor WITH ADMIN FALSE,INHERIT TRUE,SET FALSE" >/dev/null
[[ $(actor_valor "$resolutor" "SELECT count(*) FROM vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(clock_timestamp()-interval '1 minute')") == 0 ]]
psql_valor "REVOKE vec_autorizacion_motivos_evaluador FROM $resolutor" >/dev/null
psql_admin <<SQL
CREATE ROLE vec_m13_grantor LOGIN PASSWORD '$clave' NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_m13_grantor_distinto LOGIN PASSWORD '$clave' NOSUPERUSER
  NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT CONNECT ON DATABASE $base TO vec_m13_grantor,vec_m13_grantor_distinto;
GRANT vec_autorizacion_motivos_rrhh_resolutor TO vec_m13_grantor
  WITH ADMIN TRUE,INHERIT TRUE,SET FALSE;
SQL
actor_valor vec_m13_grantor \
  "GRANT vec_autorizacion_motivos_rrhh_resolutor TO vec_m13_grantor_distinto WITH ADMIN FALSE,INHERIT TRUE,SET FALSE" \
  >/dev/null
[[ $(actor_valor vec_m13_grantor_distinto "SELECT count(*) FROM vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(clock_timestamp()-interval '1 minute')") == 0 ]]
actor_valor vec_m13_grantor \
  "REVOKE vec_autorizacion_motivos_rrhh_resolutor FROM vec_m13_grantor_distinto" \
  >/dev/null
psql_admin <<SQL
REVOKE vec_autorizacion_motivos_rrhh_resolutor FROM vec_m13_grantor CASCADE;
REVOKE CONNECT ON DATABASE $base FROM vec_m13_grantor,vec_m13_grantor_distinto;
DROP ROLE vec_m13_grantor_distinto,vec_m13_grantor;
SQL

carrera_retirada() {
  local nombre=$1 consulta_resolucion=$2 consulta_retirada=$3
  local app_resuelve="m13_resuelve_$nombre" app_retira="m13_retira_$nombre"
  (actor_valor "$resolutor" \
    "BEGIN; SET application_name='$app_resuelve'; $consulta_resolucion; SELECT pg_sleep(5); COMMIT" \
    >"$salidas/resuelve_$nombre" 2>&1) &
  local proceso_resuelve=$! pid_resuelve='' listo=false
  for _ in $(seq 1 100); do
    pid_resuelve=$(psql_valor "SELECT COALESCE((SELECT pid::text FROM pg_stat_activity WHERE application_name='$app_resuelve' AND wait_event='PgSleep'),'')")
    if [[ -n $pid_resuelve ]]; then listo=true; break; fi
    sleep 0.05
  done
  [[ $listo == true ]]
  (actor_valor "$proyector" \
    "SET application_name='$app_retira'; $consulta_retirada" \
    >"$salidas/retira_$nombre" 2>&1) &
  local proceso_retira=$! bloqueado=false
  for _ in $(seq 1 100); do
    if [[ $(psql_valor "SELECT COALESCE((SELECT x.wait_event_type='Lock' AND r.pid=ANY(pg_blocking_pids(x.pid)) FROM pg_stat_activity r,pg_stat_activity x WHERE r.application_name='$app_resuelve' AND x.application_name='$app_retira'),false)::text") == true ]]; then
      bloqueado=true
      break
    fi
    sleep 0.05
  done
  [[ $bloqueado == true ]]
  wait "$proceso_resuelve"
  wait "$proceso_retira"
  grep -Fxq 1 "$salidas/resuelve_$nombre"
  grep -Fxq t "$salidas/retira_$nombre"
}

# La retirada nominal espera causalmente al FOR SHARE de la resolución.
retirar_detalle="SELECT vec_autorizacion.retirar_vinculacion_motivo_detalle_rrhh_v1('evento_vinculacion_motivo_rrhh_22222222222222222222222222222223',repeat('7',64),1,'publicacion_motivo_rrhh_22222222222222222222222222222222',repeat('6',64),clock_timestamp()-interval '8 days')"
carrera_retirada nominal \
  "SELECT count(*) FROM vec_autorizacion.resolver_motivo_detalle_rrhh_v1(clock_timestamp()-interval '1 minute')" \
  "$retirar_detalle"
[[ $(actor_valor "$resolutor" "SELECT count(*) FROM vec_autorizacion.resolver_motivo_detalle_rrhh_v1(clock_timestamp()-interval '1 minute')") == 0 ]]

# La entrada se publica y enlaza justo antes de probar su caducidad efectiva.
publicar_catalogo_caducable="SELECT vec_autorizacion.publicar_motivos_autorizacion_v2('evento_33333333333333333333333333333333',2,repeat('a',64),'motivos_rrhh_m13',2,repeat('c',64),clock_timestamp()-interval '1 minute',jsonb_build_array(jsonb_build_object('clave','motivo_cccccccccccccccccccccccccccccccc','vigente_desde',to_char((clock_timestamp()-interval '1 minute') AT TIME ZONE 'UTC','YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"'),'vigente_hasta',to_char((clock_timestamp()+interval '5 seconds') AT TIME ZONE 'UTC','YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"'))))"
[[ $(actor_valor "$proyector" "$publicar_catalogo_caducable") == t ]]
publicar_detalle_expirado="SELECT vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1('evento_vinculacion_motivo_rrhh_22222222222222222222222222222224',repeat('8',64),2,'publicacion_motivo_rrhh_22222222222222222222222222222224',repeat('9',64),'motivos_rrhh_m13',2,repeat('c',64),'motivo_cccccccccccccccccccccccccccccccc',clock_timestamp()-interval '30 seconds')"
[[ $(actor_valor "$proyector" "$publicar_detalle_expirado") == t ]]
instante_caducado=$(psql_valor "SELECT to_char(clock_timestamp() AT TIME ZONE 'UTC','YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"')")
resolver_caducado="SELECT catalogo_id,catalogo_version,catalogo_huella_sha256,entrada_clave FROM vec_autorizacion.resolver_motivo_detalle_rrhh_v1('$instante_caducado'::timestamptz)"
esperado_caducado="motivos_rrhh_m13|2|$(printf 'c%.0s' {1..64})|motivo_cccccccccccccccccccccccccccccccc"
[[ $(actor_valor "$resolutor" "$resolver_caducado") == "$esperado_caducado" ]]
caducada=false
for _ in $(seq 1 200); do
  if [[ $(psql_valor "SELECT (clock_timestamp()>=vigente_hasta)::text FROM vec_autorizacion.motivo_v2_entrada WHERE catalogo_id='motivos_rrhh_m13' AND catalogo_version=2 AND entrada_clave=('motivo_'||repeat('c',32))") == true ]]; then
    caducada=true
    break
  fi
  sleep 0.05
done
[[ $caducada == true ]]
[[ $(actor_valor "$resolutor" "SELECT count(*) FROM vec_autorizacion.resolver_motivo_detalle_rrhh_v1('$instante_caducado'::timestamptz)") == 0 ]]
[[ $(actor_valor "$resolutor" "$resolver_c") == "$esperado_c" ]]

# La retirada V2 espera al FOR SHARE global y deja también la resolución vacía.
retirar_v2="SELECT vec_autorizacion.retirar_motivos_autorizacion_v2('evento_22222222222222222222222222222222',3,repeat('d',64),'motivos_rrhh_m13',1,repeat('2',64),repeat('b',64),clock_timestamp()-interval '1 day')"
carrera_retirada v2 \
  "SELECT count(*) FROM vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(clock_timestamp()-interval '1 minute')" \
  "$retirar_v2"
[[ $(actor_valor "$resolutor" "SELECT count(*) FROM vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(clock_timestamp()-interval '1 minute')") == 0 ]]

# M1.3 puede retirarse sin borrar evidencia; reentrada down falla cerrada.
huella_datos=$(huella_evidencia)
migracion < "$raiz/$down" >/dev/null
estado_retirado
[[ $(huella_evidencia) == "$huella_datos" ]]
[[ $(huella_fundamentos) == "$huella_base" ]]
if migracion < "$raiz/$down" >/dev/null 2>&1; then
  echo '000010 down aceptó reentrada' >&2
  exit 1
fi
estado_retirado
migracion < "$raiz/$up" >/dev/null
estado_instalado

# Safe-down niega deriva de cuerpo, propietario, configuración, ACL y fundamento.
definicion_cuadro=$(psql_valor "SELECT pg_get_functiondef('vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)'::regprocedure)")
psql_admin <<'SQL'
SET ROLE vec_autorizacion_propietario;
CREATE OR REPLACE FUNCTION
  vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(p_instante timestamptz)
RETURNS TABLE(catalogo_id text,catalogo_version integer,
  catalogo_huella_sha256 text,entrada_clave text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
AS $f$ BEGIN RETURN; END $f$;
SQL
exigir_fallo_down 'cuerpo alterado'
printf '%s\n' "$definicion_cuadro" | psql_admin >/dev/null
psql_valor "ALTER FUNCTION vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz) OWNER TO vec_autorizacion_migrador" >/dev/null
exigir_fallo_down 'propietario alterado'
psql_admin <<'SQL'
ALTER FUNCTION vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)
  OWNER TO vec_autorizacion_propietario;
SET ROLE vec_autorizacion_propietario;
REVOKE ALL ON FUNCTION
  vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)
  FROM PUBLIC,vec_autorizacion_fuente,vec_autorizacion_registro,
       vec_autorizacion_migrador,vec_autorizacion_motivos_proyector,
       vec_autorizacion_motivos_evaluador,
       vec_autorizacion_motivos_rrhh_resolutor;
GRANT EXECUTE ON FUNCTION
  vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)
  TO vec_autorizacion_motivos_rrhh_resolutor;
ALTER FUNCTION vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)
  SET search_path TO public;
SQL
exigir_fallo_down 'search_path alterado'
psql_admin <<'SQL'
SET ROLE vec_autorizacion_propietario;
ALTER FUNCTION vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)
  SET search_path TO pg_catalog;
GRANT EXECUTE ON FUNCTION
  vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)
  TO vec_autorizacion_motivos_evaluador;
SQL
exigir_fallo_down 'ACL alterada'
psql_admin <<'SQL'
SET ROLE vec_autorizacion_propietario;
REVOKE EXECUTE ON FUNCTION
  vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)
  FROM vec_autorizacion_motivos_evaluador;
ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
  ALTER COLUMN entrada_clave DROP NOT NULL;
SQL
exigir_fallo_down 'fundamento alterado'
psql_admin <<'SQL'
SET ROLE vec_autorizacion_propietario;
ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
  ALTER COLUMN entrada_clave SET NOT NULL;
CREATE VIEW vec_autorizacion.dependencia_resolucion_m13 AS
SELECT * FROM vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(
  '2026-01-01T00:00:00Z'::timestamptz);
SQL
huella_dependencia=$(huella_fachadas)
if migracion < "$raiz/$down" >"$salidas/dependencia" 2>&1; then
  echo 'safe-down ignoró una dependencia SQL real' >&2
  exit 1
fi
grep -Fq '2BP01' "$salidas/dependencia"
estado_instalado
[[ $(huella_fachadas) == "$huella_dependencia" ]]
[[ $(psql_valor "SELECT (to_regclass('vec_autorizacion.dependencia_resolucion_m13') IS NOT NULL)::text") == true ]]
psql_valor "SET ROLE vec_autorizacion_propietario; DROP VIEW vec_autorizacion.dependencia_resolucion_m13" >/dev/null
migracion < "$raiz/$down" >/dev/null
estado_retirado
[[ $(huella_fundamentos) == "$huella_base" ]]
migracion < "$raiz/$up" >/dev/null

echo 'OK: resolución nominal RRHH 000010 en PostgreSQL 18.4'
