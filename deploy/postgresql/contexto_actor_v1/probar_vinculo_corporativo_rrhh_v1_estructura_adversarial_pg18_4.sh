#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C TZ=UTC

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-vinculo-corporativo-${USER:-usuario}-$$"
base=c22b_vinculo_corporativo
clave=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
directorio=deploy/postgresql/contexto_actor_v1
roles="$directorio/roles_up.sql"
roles_selector="$directorio/roles_contexto_corporativo_rrhh_selector_v1_up.sql"
up_1="$directorio/migraciones/000001_contexto_actor_v1.up.sql"
up_2="$directorio/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql"
down_2="$directorio/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.down.sql"
up_3="$directorio/migraciones/000003_organizacion_corporativa_v1.up.sql"
down_3="$directorio/migraciones/000003_organizacion_corporativa_v1.down.sql"
up="$directorio/migraciones/000004_vinculo_corporativo_rrhh_v1.up.sql"
down="$directorio/migraciones/000004_vinculo_corporativo_rrhh_v1.down.sql"
estructura="$directorio/pruebas_sql/vinculo_corporativo_rrhh_v1_estructura_catalogal.sql"
integridad="$directorio/pruebas_sql/vinculo_corporativo_rrhh_v1_integridad_relacional.sql"
fixtures_base="$directorio/pruebas_sql/fixtures_sinteticos.sql"
temporales=()

limpiar() {
  local archivo
  for archivo in "${temporales[@]:-}"; do
    [[ ! -e $archivo ]] || rm -- "$archivo"
  done
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

fallar() {
  local archivo
  echo "$1" >&2
  for archivo in "${temporales[@]:-}"; do
    [[ ! -s $archivo ]] || { echo "diagnostico ${archivo##*/}:" >&2; tail -n 50 "$archivo" >&2; }
  done
  docker logs --tail 200 "$contenedor" >&2 2>/dev/null || true
  exit 1
}

esperar_postgres() {
  local consecutivas=0 respuesta
  for _ in $(seq 1 240); do
    if respuesta=$(docker exec --env PGPASSWORD="$clave" "$contenedor" \
      psql -XAt -h 127.0.0.1 -U postgres -d "$base" \
      -c "SELECT current_setting('server_version_num')||'|'||pg_is_in_recovery()" \
      2>/dev/null) && [[ $respuesta == '180004|false' ]]; then
      consecutivas=$((consecutivas + 1))
      [[ $consecutivas -eq 3 ]] && return 0
    else
      consecutivas=0
    fi
    sleep 0.25
  done
  fallar 'PostgreSQL 18.4 primario no quedo disponible tres veces consecutivas'
}

psql_archivo() {
  docker exec --interactive --env PGPASSWORD="$clave" "$contenedor" \
    psql -Xq -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1 \
    < "$raiz/$1"
}

psql_temporal() {
  docker exec --interactive --env PGPASSWORD="$clave" "$contenedor" \
    psql -Xq -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1 < "$1"
}

psql_sql() {
  docker exec --interactive --env PGPASSWORD="$clave" "$contenedor" \
    psql -Xq -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1
}

consulta() {
  docker exec --env PGPASSWORD="$clave" "$contenedor" \
    psql -XAt -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1 -c "$1"
}

retirar() {
  docker exec --interactive --env PGPASSWORD="$clave" \
    --env PGOPTIONS='-c vec.confirmar_retirada_vinculo_corporativo_rrhh_v1=RETIRAR_VINCULO_CORPORATIVO_RRHH_V1' \
    "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
    -v ON_ERROR_STOP=1 < "$raiz/$down"
}

retirar_mal() {
  docker exec --interactive --env PGPASSWORD="$clave" \
    --env PGOPTIONS='-c vec.confirmar_retirada_vinculo_corporativo_rrhh_v1=NO' \
    "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
    -v ON_ERROR_STOP=1 < "$raiz/$down"
}

retirar_000002() {
  docker exec --interactive --env PGPASSWORD="$clave" \
    --env PGOPTIONS='-c vec.confirmar_retirada_acreditacion_contexto_actor_v2=RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2' \
    "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
    -v ON_ERROR_STOP=1 < "$raiz/$down_2"
}

retirar_000003() {
  docker exec --interactive --env PGPASSWORD="$clave" \
    --env PGOPTIONS='-c vec.confirmar_retirada_organizacion_corporativa_v1=RETIRAR_ORGANIZACION_CORPORATIVA_V1' \
    "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
    -v ON_ERROR_STOP=1 < "$raiz/$down_3"
}

huella() {
  docker exec --env PGPASSWORD="$clave" "$contenedor" pg_dump \
    -h 127.0.0.1 -U postgres -d "$base" --format=plain |
    sed -E '/^\\(un)?restrict /d' | sha256sum | cut -d' ' -f1
}

huella_filas_base() {
  docker exec --env PGPASSWORD="$clave" "$contenedor" pg_dump \
    -h 127.0.0.1 -U postgres -d "$base" --data-only \
    --schema=vec_contexto_actor_v1 \
    --exclude-table=vec_contexto_actor_v1.vinculo_corporativo_versiones \
    --exclude-table=vec_contexto_actor_v1.vinculo_corporativo_actual |
    sed -E '/^\\(un)?restrict /d' | sha256sum | cut -d' ' -f1
}

# Limita la comparación al esquema gobernado A; las funciones externas de
# public son deliberadamente ajenas y se acreditan por una huella separada.
huella_catalogal_a() {
  docker exec --env PGPASSWORD="$clave" "$contenedor" pg_dump \
    -h 127.0.0.1 -U postgres -d "$base" --schema-only \
    --schema=vec_contexto_actor_v1 |
    sed -E '/^\\(un)?restrict /d' | sha256sum | cut -d' ' -f1
}

huella_funciones_externas() {
  consulta "SELECT encode(sha256(convert_to(string_agg(format(
    '%s|%s|%s',p.oid::regprocedure::text,pg_get_userbyid(p.proowner),
    coalesce(p.proacl::text,'')||pg_get_functiondef(p.oid)),E'\\n'
    ORDER BY p.oid::regprocedure::text),'UTF8')),'hex')
   FROM pg_proc p WHERE p.oid IN (
    'public.c22b_externa_inocua(integer)'::regprocedure,
    'public.c22b_externa_posterior(text)'::regprocedure)"
}

exigir_fallo_intacto() {
  local descripcion=$1 funcion=$2 antes despues estado
  antes=$(huella)
  set +e
  "$funcion" >/dev/null 2>&1
  estado=$?
  set -e
  [[ $estado -ne 0 ]] || fallar "se acepto: $descripcion"
  despues=$(huella)
  [[ $antes == "$despues" ]] || fallar "el rechazo altero estado: $descripcion"
}

up_reentrada() { psql_archivo "$up"; }
down_sin_confirmacion() { psql_archivo "$down"; }

docker run -d --rm --name "$contenedor" --publish 127.0.0.1::5432 \
  -e POSTGRES_DB="$base" -e POSTGRES_PASSWORD="$clave" "$imagen" \
  -c allow_system_table_mods=on >/dev/null
esperar_postgres
psql_sql <<'SQL'
DO $base$
BEGIN
  CREATE ROLE c22b_dueno_base NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOINHERIT NOREPLICATION NOBYPASSRLS;
  EXECUTE format('ALTER DATABASE %I OWNER TO c22b_dueno_base',current_database());
  EXECUTE format('REVOKE ALL ON DATABASE %I FROM PUBLIC',current_database());
END
$base$;
REVOKE ALL ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
CREATE ROLE c22b_desplazamiento_oid_01 NOLOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE c22b_desplazamiento_oid_02 NOLOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE c22b_desplazamiento_oid_03 NOLOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE c22b_desplazamiento_oid_04 NOLOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE c22b_desplazamiento_oid_05 NOLOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE c22b_desplazamiento_oid_06 NOLOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE c22b_desplazamiento_oid_07 NOLOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE c22b_desplazamiento_oid_08 NOLOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
SQL
psql_archivo "$roles"
[[ $(consulta "SELECT count(*)=8 AND bool_and(oid < (
  SELECT oid FROM pg_catalog.pg_authid
   WHERE rolname='vec_contexto_actor_v1_propietario'))
 FROM pg_catalog.pg_authid
 WHERE rolname=ANY(ARRAY[
  'c22b_desplazamiento_oid_01','c22b_desplazamiento_oid_02',
  'c22b_desplazamiento_oid_03','c22b_desplazamiento_oid_04',
  'c22b_desplazamiento_oid_05','c22b_desplazamiento_oid_06',
  'c22b_desplazamiento_oid_07','c22b_desplazamiento_oid_08'])") == t ]] ||
  fallar 'la regresion no desplazo los OID de los roles gobernados'
psql_archivo "$up_1"
psql_archivo "$up_2"
psql_archivo "$roles_selector"
psql_archivo "$up_3"
psql_sql <<'SQL'
CREATE ROLE c22b_consumidor NOLOGIN;
CREATE ROLE c22b_login LOGIN;
CREATE ROLE c22b_pdp NOLOGIN;
CREATE ROLE c22b_autorizacion NOLOGIN;
CREATE ROLE c22b_contratacion NOLOGIN;
CREATE ROLE c22b_bolsa NOLOGIN;
GRANT vec_contexto_actor_v1_runtime TO c22b_login
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE FUNCTION public.c22b_externa_inocua(integer) RETURNS integer
  LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS 'SELECT $1 + 1';
REVOKE ALL ON FUNCTION public.c22b_externa_inocua(integer)
  FROM PUBLIC, vec_contexto_actor_v1_runtime;
SQL
huella_a=$(huella)
huella_catalogal_a_inicial=$(huella_catalogal_a)

# B1, caso 1: atomicidad, instalacion literal y reentrada cerrada.
psql_sql <<'SQL'
INSERT INTO pg_catalog.pg_seclabel(objoid,classoid,objsubid,provider,label)
VALUES ('vec_contexto_actor_v1'::regnamespace,
        'pg_catalog.pg_namespace'::regclass,0,
        'c22b_sintetico','etiqueta_esquema_hostil');
SQL
exigir_fallo_intacto 'etiqueta de seguridad hostil sobre esquema' up_reentrada
psql_sql <<'SQL'
DELETE FROM pg_catalog.pg_seclabel
 WHERE objoid='vec_contexto_actor_v1'::regnamespace
   AND classoid='pg_catalog.pg_namespace'::regclass
   AND provider='c22b_sintetico';
SQL
[[ $(huella) == "$huella_a" ]] ||
  fallar 'la sonda pg_seclabel de alta no restauro A'
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
ALTER FUNCTION vec_contexto_actor_v1.organizacion_ref_valida(text) VOLATILE;
RESET ROLE;
SQL
exigir_fallo_intacto 'funcion predecesora hostil' up_reentrada
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
ALTER FUNCTION vec_contexto_actor_v1.organizacion_ref_valida(text) IMMUTABLE;
ALTER TABLE vec_contexto_actor_v1.organizacion_versiones
  SET (toast.autovacuum_enabled=false);
RESET ROLE;
SQL
exigir_fallo_intacto 'opcion TOAST predecesora hostil' up_reentrada
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
ALTER TABLE vec_contexto_actor_v1.organizacion_versiones
  RESET (toast.autovacuum_enabled);
ALTER TABLE vec_contexto_actor_v1.perfil_versiones
  ALTER COLUMN persona_ref SET STORAGE MAIN;
RESET ROLE;
SQL
exigir_fallo_intacto 'almacenamiento de columna predecesora hostil' up_reentrada
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
ALTER TABLE vec_contexto_actor_v1.perfil_versiones
  ALTER COLUMN persona_ref SET STORAGE EXTENDED;
ALTER TABLE vec_contexto_actor_v1.organizacion_versiones
  DISABLE ROW LEVEL SECURITY;
RESET ROLE;
SQL
exigir_fallo_intacto 'RLS predecesora hostil' up_reentrada
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
ALTER TABLE vec_contexto_actor_v1.organizacion_versiones
  ENABLE ROW LEVEL SECURITY;
ALTER POLICY acceso_propietario_exacto
  ON vec_contexto_actor_v1.organizacion_versiones TO PUBLIC;
RESET ROLE;
SQL
exigir_fallo_intacto 'politica predecesora hostil' up_reentrada
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
ALTER POLICY acceso_propietario_exacto
  ON vec_contexto_actor_v1.organizacion_versiones
  TO vec_contexto_actor_v1_propietario;
ALTER FUNCTION vec_contexto_actor_v1.organizacion_ref_valida(text)
  SECURITY DEFINER;
RESET ROLE;
SQL
exigir_fallo_intacto 'funcion predecesora SECURITY DEFINER' up_reentrada
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
ALTER FUNCTION vec_contexto_actor_v1.organizacion_ref_valida(text)
  SECURITY INVOKER;
RESET ROLE;
GRANT vec_contexto_actor_v1_propietario TO vec_contexto_actor_v1_runtime
  WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;
SET SESSION AUTHORIZATION vec_contexto_actor_v1_runtime;
SET ROLE vec_contexto_actor_v1_propietario;
RESET ROLE;
RESET SESSION AUTHORIZATION;
SQL
exigir_fallo_intacto 'membresia hostil runtime a propietario' up_reentrada
psql_sql <<'SQL'
REVOKE vec_contexto_actor_v1_propietario FROM vec_contexto_actor_v1_runtime;
GRANT vec_contexto_actor_corporativo_rrhh_selector TO c22b_login
  WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
SET SESSION AUTHORIZATION c22b_login;
SET ROLE vec_contexto_actor_corporativo_rrhh_selector;
RESET ROLE;
RESET SESSION AUTHORIZATION;
SQL
exigir_fallo_intacto 'membresia hostil LOGIN a selector' up_reentrada
psql_sql <<'SQL'
REVOKE vec_contexto_actor_corporativo_rrhh_selector FROM c22b_login;
ALTER ROLE c22b_login SET search_path=public;
SQL
exigir_fallo_intacto 'ajuste persistente hostil de LOGIN' up_reentrada
psql_sql <<'SQL'
ALTER ROLE c22b_login RESET ALL;
SQL
[[ $(huella) == "$huella_a" ]] || fallar 'las regresiones predecesoras no restauraron A'
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
CREATE TABLE vec_contexto_actor_v1.vinculo_corporativo_actual(id integer);
RESET ROLE;
SQL
exigir_fallo_intacto 'adopcion parcial de objeto nominal' up_reentrada
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
DROP TABLE vec_contexto_actor_v1.vinculo_corporativo_actual;
RESET ROLE;
CREATE FUNCTION public.c22b_fallo_tabla() RETURNS event_trigger LANGUAGE plpgsql AS $$
BEGIN
  IF tg_tag='CREATE TABLE' THEN RAISE EXCEPTION 'fallo atomico sintetico'; END IF;
END $$;
CREATE EVENT TRIGGER c22b_fallo_tabla ON ddl_command_start
  EXECUTE FUNCTION public.c22b_fallo_tabla();
SQL
exigir_fallo_intacto 'fallo atomico durante CREATE TABLE' up_reentrada
psql_sql <<'SQL'
DROP EVENT TRIGGER c22b_fallo_tabla;
DROP FUNCTION public.c22b_fallo_tabla();
SQL
[[ $(huella) == "$huella_a" ]] || fallar 'la prueba atomica no restauro A'
psql_sql <<'SQL'
CREATE FUNCTION public.c22b_fallo_politica() RETURNS event_trigger LANGUAGE plpgsql AS $$
BEGIN
  IF tg_tag='CREATE POLICY' THEN RAISE EXCEPTION 'fallo tardio sintetico'; END IF;
END $$;
CREATE EVENT TRIGGER c22b_fallo_politica ON ddl_command_start
  EXECUTE FUNCTION public.c22b_fallo_politica();
SQL
exigir_fallo_intacto 'fallo atomico tardio durante CREATE POLICY' up_reentrada
psql_sql <<'SQL'
DROP EVENT TRIGGER c22b_fallo_politica;
DROP FUNCTION public.c22b_fallo_politica();
SQL
[[ $(huella) == "$huella_a" ]] || fallar 'el fallo tardio no restauro A'
psql_sql <<'SQL'
CREATE FUNCTION public.c22b_mutar_postcondicion() RETURNS event_trigger
LANGUAGE plpgsql AS $$
BEGIN
  IF tg_tag='CREATE TABLE' AND
     to_regclass('vec_contexto_actor_v1.vinculo_corporativo_versiones') IS NOT NULL THEN
    ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
      REPLICA IDENTITY FULL;
  END IF;
END $$;
CREATE EVENT TRIGGER c22b_mutar_postcondicion ON ddl_command_end
  EXECUTE FUNCTION public.c22b_mutar_postcondicion();
SQL
exigir_fallo_intacto 'mutacion tardia de identidad de replica' up_reentrada
psql_sql <<'SQL'
DROP EVENT TRIGGER c22b_mutar_postcondicion;
DROP FUNCTION public.c22b_mutar_postcondicion();
SQL
[[ $(huella) == "$huella_a" ]] ||
  fallar 'la postcondicion hostil no produjo rollback integro'
psql_archivo "$up"
exigir_fallo_intacto 'reentrada de 000004 up' up_reentrada
psql_sql <<'SQL'
CREATE FUNCTION public.c22b_externa_posterior(text) RETURNS text
  LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS 'SELECT upper($1)';
REVOKE ALL ON FUNCTION public.c22b_externa_posterior(text)
  FROM PUBLIC, vec_contexto_actor_v1_runtime;
SQL
huella_externas=$(huella_funciones_externas)

# B1, casos 2-7: forma, relaciones, integridad, inmutabilidad y acceso.
psql_archivo "$estructura"
psql_archivo "$integridad"

# B1, casos 8-9: orden de migraciones y confirmacion de retirada.
exigir_fallo_intacto 'retirada 000002 con C2.2-B presente' retirar_000002
exigir_fallo_intacto 'retirada 000003 con C2.2-B presente' retirar_000003
exigir_fallo_intacto 'retirada B sin confirmacion' down_sin_confirmacion
exigir_fallo_intacto 'retirada B con confirmacion incorrecta' retirar_mal

# B1, caso 9: toda deriva o dependencia hostil impide la retirada.
psql_sql <<'SQL'
INSERT INTO pg_catalog.pg_seclabel(objoid,classoid,objsubid,provider,label)
VALUES ('vec_contexto_actor_v1.organizacion_ref_valida(text)'::regprocedure,
        'pg_catalog.pg_proc'::regclass,0,
        'c22b_sintetico','etiqueta_funcion_hostil');
SQL
exigir_fallo_intacto 'retirada con etiqueta de seguridad hostil en funcion' retirar
psql_sql <<'SQL'
DELETE FROM pg_catalog.pg_seclabel
 WHERE objoid='vec_contexto_actor_v1.organizacion_ref_valida(text)'::regprocedure
   AND classoid='pg_catalog.pg_proc'::regclass
   AND provider='c22b_sintetico';
GRANT vec_contexto_actor_corporativo_rrhh_selector TO c22b_login
  WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
SQL
exigir_fallo_intacto 'retirada con membresia hostil de LOGIN' retirar
psql_sql <<'SQL'
REVOKE vec_contexto_actor_corporativo_rrhh_selector FROM c22b_login;
ALTER ROLE c22b_login SET search_path=public;
SQL
exigir_fallo_intacto 'retirada con ajuste persistente hostil de LOGIN' retirar
psql_sql <<'SQL'
ALTER ROLE c22b_login RESET ALL;
SET ROLE vec_contexto_actor_v1_propietario;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_actual
  REPLICA IDENTITY USING INDEX vinculo_corporativo_actual_pk;
RESET ROLE;
SQL
exigir_fallo_intacto 'retirada con identidad de replica hostil' retirar
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_actual
  REPLICA IDENTITY DEFAULT;
ALTER FUNCTION vec_contexto_actor_v1.organizacion_ref_valida(text)
  SECURITY DEFINER;
RESET ROLE;
SQL
exigir_fallo_intacto 'retirada con funcion predecesora SECURITY DEFINER' retirar
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
ALTER FUNCTION vec_contexto_actor_v1.organizacion_ref_valida(text)
  SECURITY INVOKER;
RESET ROLE;
SET ROLE vec_contexto_actor_v1_propietario;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
  DISABLE ROW LEVEL SECURITY;
RESET ROLE;
SQL
exigir_fallo_intacto 'retirada con RLS hostil' retirar
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
  ENABLE ROW LEVEL SECURITY;
ALTER POLICY acceso_propietario_exacto
  ON vec_contexto_actor_v1.vinculo_corporativo_versiones TO PUBLIC;
RESET ROLE;
SQL
exigir_fallo_intacto 'retirada con politica hostil' retirar
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
ALTER POLICY acceso_propietario_exacto
  ON vec_contexto_actor_v1.vinculo_corporativo_versiones
  TO vec_contexto_actor_v1_propietario;
RESET ROLE;
COMMENT ON TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
  IS 'metadato hostil sintetico';
SQL
exigir_fallo_intacto 'retirada con comentario hostil' retirar
psql_sql <<'SQL'
COMMENT ON TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones IS NULL;
COMMENT ON POLICY acceso_propietario_exacto
  ON vec_contexto_actor_v1.vinculo_corporativo_versiones
  IS 'politica hostil sintetica';
SQL
exigir_fallo_intacto 'retirada con comentario hostil de politica' retirar
psql_sql <<'SQL'
COMMENT ON POLICY acceso_propietario_exacto
  ON vec_contexto_actor_v1.vinculo_corporativo_versiones IS NULL;
CREATE INDEX c22b_indice_hostil
 ON vec_contexto_actor_v1.vinculo_corporativo_versiones(estado);
SQL
exigir_fallo_intacto 'retirada con indice hostil' retirar
psql_sql <<'SQL'
DROP INDEX vec_contexto_actor_v1.c22b_indice_hostil;
GRANT USAGE ON TYPE vec_contexto_actor_v1.vinculo_corporativo_versiones
  TO c22b_consumidor;
SQL
exigir_fallo_intacto 'retirada con ACL hostil de tipo' retirar
psql_sql <<'SQL'
REVOKE USAGE ON TYPE vec_contexto_actor_v1.vinculo_corporativo_versiones
  FROM c22b_consumidor;
GRANT SELECT (estado) ON vec_contexto_actor_v1.vinculo_corporativo_versiones
  TO c22b_consumidor;
SQL
exigir_fallo_intacto 'retirada con ACL hostil de columna' retirar
psql_sql <<'SQL'
REVOKE SELECT (estado) ON vec_contexto_actor_v1.vinculo_corporativo_versiones
  FROM c22b_consumidor;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO c22b_consumidor;
SQL
exigir_fallo_intacto 'retirada con ACL hostil de namespace' retirar
psql_sql <<'SQL'
REVOKE USAGE ON SCHEMA vec_contexto_actor_v1 FROM c22b_consumidor;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
  SET (toast.autovacuum_enabled=false);
SQL
exigir_fallo_intacto 'retirada con opcion TOAST hostil' retirar
docker exec --user root "$contenedor" mkdir -p /tmp/c22b_espacio_hostil
docker exec --user root "$contenedor" chown postgres:postgres /tmp/c22b_espacio_hostil
psql_sql <<'SQL'
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
  RESET (toast.autovacuum_enabled);
CREATE TABLESPACE c22b_espacio_hostil LOCATION '/tmp/c22b_espacio_hostil';
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
  SET TABLESPACE c22b_espacio_hostil;
SQL
exigir_fallo_intacto 'retirada con tablespace hostil' retirar
psql_sql <<'SQL'
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
  SET TABLESPACE pg_default;
DROP TABLESPACE c22b_espacio_hostil;
CREATE STATISTICS vec_contexto_actor_v1.c22b_estadistica_hostil
  ON estado, superficie
  FROM vec_contexto_actor_v1.vinculo_corporativo_versiones;
SQL
exigir_fallo_intacto 'retirada con estadistica hostil' retirar
psql_sql <<'SQL'
DROP STATISTICS vec_contexto_actor_v1.c22b_estadistica_hostil;
CREATE PUBLICATION c22b_publicacion_hostil
  FOR TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones;
SQL
exigir_fallo_intacto 'retirada con publicacion hostil' retirar
psql_sql <<'SQL'
DROP PUBLICATION c22b_publicacion_hostil;
CREATE VIEW public.c22b_consumidor_hostil AS
 SELECT estado FROM vec_contexto_actor_v1.vinculo_corporativo_versiones;
SQL
exigir_fallo_intacto 'retirada con dependencia externa consumidora' retirar
psql_sql <<'SQL'
DROP VIEW public.c22b_consumidor_hostil;
SET ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_contexto_actor_v1.c22b_consumidor_dinamico()
RETURNS bigint LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE resultado bigint;
BEGIN
  EXECUTE 'SELECT count(*) FROM vec_contexto_actor_v1.'||
          'vinculo_'||'corporativo_'||'versiones'
    INTO resultado;
  RETURN resultado;
END $$;
RESET ROLE;
SQL
exigir_fallo_intacto 'retirada con consumidor dinamico gobernado ofuscado' retirar
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
DROP FUNCTION vec_contexto_actor_v1.c22b_consumidor_dinamico();
RESET ROLE;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario
 IN SCHEMA vec_contexto_actor_v1 GRANT SELECT ON TABLES TO c22b_consumidor;
SQL
exigir_fallo_intacto 'retirada con ACL predeterminada hostil' retirar
psql_sql <<'SQL'
ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario
 IN SCHEMA vec_contexto_actor_v1 REVOKE SELECT ON TABLES FROM c22b_consumidor;
SET ROLE vec_contexto_actor_v1_propietario;
CREATE TABLE vec_contexto_actor_v1.c22b_000005_sintetica(id integer);
RESET ROLE;
SQL
exigir_fallo_intacto 'retirada ante 000005 sintetica' retirar
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
DROP TABLE vec_contexto_actor_v1.c22b_000005_sintetica;
RESET ROLE;
SQL

# B1, caso 9: cualquier evidencia de B impide retirarla sin alterar estado.
restaurar_generacion=$(consulta "SELECT format(
 'UPDATE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 SET generacion=%s, actualizada_en=%L',
 generacion,actualizada_en) FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2")
evidencia=$(mktemp)
temporales+=("$evidencia")
sed 's/^ROLLBACK;$/COMMIT;/' "$integridad" > "$evidencia"
psql_temporal "$evidencia"
exigir_fallo_intacto 'retirada con historia y puntero persistentes' retirar
psql_sql <<SQL
SET session_replication_role=replica;
DELETE FROM vec_contexto_actor_v1.vinculo_corporativo_actual;
DELETE FROM vec_contexto_actor_v1.vinculo_corporativo_versiones;
DELETE FROM vec_contexto_actor_v1.organizacion_versiones
 WHERE organizacion_ref='org_diputaciondemo0001';
DELETE FROM vec_contexto_actor_v1.vinculo_contexto_versiones
 WHERE vinculo_ref='vca_corporativo_rrhh_000000000001';
DELETE FROM vec_contexto_actor_v1.perfil_versiones
 WHERE perfil_ref='prf_corporativo_rrhh_000000000001';
DELETE FROM vec_contexto_actor_v1.persona_versiones
 WHERE persona_ref='per_corporativa_rrhh_000000000001';
DELETE FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones
 WHERE cuenta_ref='cta_corporativa_rrhh_000000000001';
DELETE FROM vec_contexto_actor_v1.procedencias
 WHERE procedencia_ref IN ('prc_autoridad_corporativa_rrhh_0001',
                           'prc_vinculo_corporativo_rrhh_000001');
$restaurar_generacion;
SET session_replication_role=origin;
SQL
psql_archivo "$fixtures_base"
huella_base_poblada=$(huella_filas_base)
generacion_base_poblada=$(consulta 'SELECT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2')
retirar
[[ $(huella_catalogal_a) == "$huella_catalogal_a_inicial" ]] ||
  fallar 'la primera retirada altero el catalogo focal de A'
[[ $(huella_filas_base) == "$huella_base_poblada" ]] ||
  fallar 'la retirada altero filas de A pobladas'
[[ $(consulta 'SELECT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2') == "$generacion_base_poblada" ]] ||
  fallar 'la retirada altero la generacion comun'
[[ $(huella_funciones_externas) == "$huella_externas" ]] ||
  fallar 'la retirada altero funciones externas inocuas'

# B1, caso 10: reinstalacion y preservacion; caso 13: limpieza por trap.
psql_archivo "$up"
psql_archivo "$estructura"
retirar
[[ $(huella_catalogal_a) == "$huella_catalogal_a_inicial" ]] ||
  fallar 'el segundo ciclo altero el catalogo focal de A'
[[ $(huella_filas_base) == "$huella_base_poblada" ]] ||
  fallar 'el segundo ciclo altero filas de A pobladas'
[[ $(consulta 'SELECT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2') == "$generacion_base_poblada" ]] ||
  fallar 'el segundo ciclo altero la generacion comun'
[[ $(consulta 'SELECT public.c22b_externa_inocua(1)') == 2 ]] ||
  fallar 'los ciclos alteraron la funcion externa inocua'
[[ $(consulta "SELECT public.c22b_externa_posterior('demo')") == DEMO ]] ||
  fallar 'los ciclos alteraron la funcion externa posterior'
[[ $(huella_funciones_externas) == "$huella_externas" ]] ||
  fallar 'el segundo ciclo altero definicion, owner o ACL externas'

echo 'OK: vinculo corporativo RRHH V1 supera estructura, integridad y retirada PG 18.4'
