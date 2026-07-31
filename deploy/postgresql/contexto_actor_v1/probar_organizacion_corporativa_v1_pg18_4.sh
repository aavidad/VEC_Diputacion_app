#!/usr/bin/env bash
set -euo pipefail

export LC_ALL=C
export TZ=UTC

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"

imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-contexto-actor-organizacion-pg-${USER:-usuario}-$$"
base=ct140_organizacion
clave_admin=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
directorio=deploy/postgresql/contexto_actor_v1
roles="$directorio/roles_up.sql"
roles_selector="$directorio/roles_contexto_corporativo_rrhh_selector_v1_up.sql"
up_1="$directorio/migraciones/000001_contexto_actor_v1.up.sql"
up_2="$directorio/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql"
down_2="$directorio/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.down.sql"
up="$directorio/migraciones/000003_organizacion_corporativa_v1.up.sql"
down="$directorio/migraciones/000003_organizacion_corporativa_v1.down.sql"
temporales=()

limpiar() {
  local proceso archivo
  while IFS= read -r proceso; do
    kill "$proceso" >/dev/null 2>&1 || true
    wait "$proceso" >/dev/null 2>&1 || true
  done < <(jobs -pr)
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
    [[ ! -s $archivo ]] || {
      echo "diagnostico ${archivo##*/}:" >&2
      tail -n 30 "$archivo" >&2
    }
  done
  exit 1
}

esperar_postgres() {
  local consecutivas=0 respuesta
  for _ in $(seq 1 240); do
    if respuesta=$(docker exec --env PGPASSWORD="$clave_admin" "$contenedor" \
      psql -XAt --set ON_ERROR_STOP=1 --host 127.0.0.1 \
      --username postgres --dbname "$base" \
      --command \
      "SELECT current_setting('server_version_num') || '|' ||
              pg_catalog.pg_is_in_recovery()" 2>/dev/null) &&
      [[ $respuesta == '180004|false' ]]; then
      consecutivas=$((consecutivas + 1))
      [[ $consecutivas -eq 3 ]] && return 0
    else
      consecutivas=0
    fi
    sleep 0.25
  done
  docker logs --tail 200 "$contenedor" >&2 || true
  fallar 'PostgreSQL 18.4 primario no quedó disponible por TCP dentro del plazo'
}

psql_archivo() {
  local archivo=$1
  docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/$archivo"
}

psql_admin() {
  docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base"
}

consulta() {
  local sql=$1
  docker exec "$contenedor" psql -XAt --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command "$sql"
}

retirar() {
  docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_retirada_organizacion_corporativa_v1=RETIRAR_ORGANIZACION_CORPORATIVA_V1" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/$down"
}

retirar_000002() {
  docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_retirada_acreditacion_contexto_actor_v2=RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/$down_2"
}

huella_estado() {
  docker exec "$contenedor" pg_dump --format=plain \
    --username postgres --dbname "$base" |
    sed -E '/^\\(un)?restrict /d' |
    sha256sum | cut -d' ' -f1
}

exigir_fallo_intacto() {
  local descripcion=$1 funcion=$2 antes despues estado
  antes=$(huella_estado)
  set +e
  "$funcion" >/dev/null 2>&1
  estado=$?
  set -e
  [[ $estado -ne 0 ]] || fallar "se acepto: $descripcion"
  despues=$(huella_estado)
  [[ $antes == "$despues" ]] ||
    fallar "el rechazo altero estado: $descripcion"
}

retirar_sin_confirmacion() {
  psql_archivo "$down"
}

retirar_confirmacion_incorrecta() {
  docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_retirada_organizacion_corporativa_v1=NO" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/$down"
}

psql_archivo_reentrada() {
  psql_archivo "$up"
}

esperar_sesion() {
  local aplicacion=$1 predicado=$2 descripcion=$3
  for _ in $(seq 1 300); do
    if [[ $(consulta "
SELECT pg_catalog.count(*) FROM pg_catalog.pg_stat_activity
 WHERE application_name='$aplicacion' AND $predicado") == 1 ]]; then
      return
    fi
    sleep 0.02
  done
  fallar "no se observo $descripcion"
}

docker run --detach --rm --name "$contenedor" --publish 127.0.0.1::5432 \
  --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave_admin" \
  "$imagen" >/dev/null
esperar_postgres

psql_admin <<'SQL'
DO $base$
BEGIN
  CREATE ROLE ct140_dueno_base NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOINHERIT NOREPLICATION NOBYPASSRLS;
  EXECUTE pg_catalog.format(
    'ALTER DATABASE %I OWNER TO ct140_dueno_base',
    pg_catalog.current_database()
  );
  EXECUTE pg_catalog.format(
    'REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
    pg_catalog.current_database()
  );
END
$base$;
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
psql_archivo "$roles"
psql_archivo "$up_1"
psql_archivo "$up_2"
psql_archivo "$roles_selector"
psql_admin <<'SQL'
CREATE ROLE ct140_consumidor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE ct140_login LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contexto_actor_v1_runtime TO ct140_login
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL

# 1. Instalacion atomica y rechazo de reentrada sin deriva.
huella_000002=$(huella_estado)
psql_admin <<'SQL'
GRANT vec_contexto_actor_v1_propietario
  TO vec_contexto_actor_v1_runtime
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL
exigir_fallo_intacto 'runtime heredando al propietario antes del up' \
  psql_archivo_reentrada
psql_admin <<'SQL'
REVOKE vec_contexto_actor_v1_propietario
  FROM vec_contexto_actor_v1_runtime;
SQL
[[ $(huella_estado) == "$huella_000002" ]] ||
  fallar 'la prueba de topologia no restauro 000002'
psql_admin <<'SQL'
CREATE FUNCTION public.ct140_fallar_creacion_tabla()
RETURNS event_trigger LANGUAGE plpgsql AS $funcion$
BEGIN
  IF tg_tag='CREATE TABLE' THEN
    RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='fallo sintetico atomico';
  END IF;
END
$funcion$;
CREATE EVENT TRIGGER ct140_fallar_creacion_tabla
  ON ddl_command_start EXECUTE FUNCTION public.ct140_fallar_creacion_tabla();
SQL
exigir_fallo_intacto 'fallo atomico durante CREATE TABLE' \
  psql_archivo_reentrada
[[ $(consulta "SELECT
  pg_catalog.to_regprocedure(
    'vec_contexto_actor_v1.organizacion_ref_valida(text)') IS NULL
  AND pg_catalog.to_regclass(
    'vec_contexto_actor_v1.organizacion_versiones') IS NULL
  AND pg_catalog.to_regclass(
    'vec_contexto_actor_v1.organizacion_actual') IS NULL") == t ]] ||
  fallar 'el alta fallida dejo objetos parciales'
psql_admin <<'SQL'
DROP EVENT TRIGGER ct140_fallar_creacion_tabla;
DROP FUNCTION public.ct140_fallar_creacion_tabla();
SQL
[[ $(huella_estado) == "$huella_000002" ]] ||
  fallar 'la prueba atomica no restauro 000002'
psql_admin <<'SQL'
CREATE FUNCTION public.ct140_inyectar_membresia()
RETURNS event_trigger LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog AS $funcion$
BEGIN
  GRANT vec_contexto_actor_v1_propietario
    TO vec_contexto_actor_v1_runtime
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
END
$funcion$;
CREATE EVENT TRIGGER ct140_inyectar_membresia
  ON ddl_command_end WHEN TAG IN ('CREATE TABLE')
  EXECUTE FUNCTION public.ct140_inyectar_membresia();
SQL
exigir_fallo_intacto 'acceso efectivo inyectado antes de postcondicion' \
  psql_archivo_reentrada
[[ $(consulta "SELECT NOT pg_catalog.pg_has_role(
  'vec_contexto_actor_v1_runtime',
  'vec_contexto_actor_v1_propietario','MEMBER')") == t ]] ||
  fallar 'el rechazo postcondicional conservo la membresia hostil'
psql_admin <<'SQL'
DROP EVENT TRIGGER ct140_inyectar_membresia;
DROP FUNCTION public.ct140_inyectar_membresia();
SQL
[[ $(huella_estado) == "$huella_000002" ]] ||
  fallar 'la prueba postcondicional no restauro 000002'
psql_archivo "$up"
exigir_fallo_intacto 'reentrada 000003' psql_archivo_reentrada

# 2. Forma, ACL, RLS, indices, politicas, triggers y TOAST exactos.
psql_admin <<'SQL'
DO $estructura$
DECLARE
  propietario oid := 'vec_contexto_actor_v1_propietario'::regrole;
  versiones oid := 'vec_contexto_actor_v1.organizacion_versiones'::regclass;
  actual oid := 'vec_contexto_actor_v1.organizacion_actual'::regclass;
BEGIN
  IF (
    SELECT pg_catalog.count(*) FROM pg_catalog.pg_class
     WHERE oid IN (versiones, actual) AND relkind='r' AND relpersistence='p'
       AND relowner=propietario AND relrowsecurity AND relforcerowsecurity
  ) <> 2 OR (
    SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
     WHERE polrelid IN (versiones, actual)
       AND polname='acceso_propietario_exacto' AND polpermissive
       AND polcmd='*' AND polroles=ARRAY[propietario]::oid[]
       AND pg_catalog.pg_get_expr(polqual,polrelid)=
           '(CURRENT_USER = ''vec_contexto_actor_v1_propietario''::name)'
       AND pg_catalog.pg_get_expr(polwithcheck,polrelid)=
           '(CURRENT_USER = ''vec_contexto_actor_v1_propietario''::name)'
  ) <> 2 OR (
    SELECT pg_catalog.count(*) FROM pg_catalog.pg_index
     WHERE indrelid IN (versiones, actual)
  ) <> 3 OR (
    SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger
     WHERE tgrelid IN (versiones, actual) AND NOT tgisinternal
  ) <> 5 OR (
    SELECT pg_catalog.count(*) FROM pg_catalog.pg_class AS c
     JOIN pg_catalog.pg_class AS t ON t.oid=c.reltoastrelid
     WHERE c.oid IN (versiones, actual) AND t.relkind='t'
       AND t.relowner=propietario AND t.relacl IS NULL
  ) <> 2 OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_attribute
     WHERE attrelid IN (versiones,actual) AND attnum>0
       AND (attisdropped OR attacl IS NOT NULL)
  ) OR (
    SELECT pg_catalog.string_agg(pg_catalog.format(
             '%s|%s|%s|%s|%s',attnum,attname,
             pg_catalog.format_type(atttypid,atttypmod),
             attnotnull,atttypmod
           ),';' ORDER BY attnum)
      FROM pg_catalog.pg_attribute
     WHERE attrelid=versiones AND attnum>0 AND NOT attisdropped
  ) IS DISTINCT FROM
    '1|organizacion_ref|text|t|-1;2|version|numeric(20,0)|t|1310724;3|procedencia_ref|text|t|-1;4|procedencia_version|numeric(20,0)|t|1310724;5|procedencia_huella_sha256|text|t|-1;6|procedencia_autoridad|text|t|-1;7|estado|text|t|-1;8|vigente_desde|timestamp(6) with time zone|t|6;9|vigente_hasta|timestamp(6) with time zone|t|6'
  OR (
    SELECT pg_catalog.string_agg(pg_catalog.format(
             '%s|%s|%s|%s|%s',attnum,attname,
             pg_catalog.format_type(atttypid,atttypmod),
             attnotnull,atttypmod
           ),';' ORDER BY attnum)
      FROM pg_catalog.pg_attribute
     WHERE attrelid=actual AND attnum>0 AND NOT attisdropped
  ) IS DISTINCT FROM
    '1|organizacion_ref|text|t|-1;2|version|numeric(20,0)|t|1310724'
  OR (
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
             pg_catalog.string_agg(pg_catalog.format(
               '%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
               conname,contype::text,conkey::text,
               CASE WHEN confrelid=0 THEN '' ELSE confrelid::regclass::text END,
               confkey::text,confmatchtype::text,confupdtype::text,
               confdeltype::text,condeferrable,condeferred,convalidated,
               pg_catalog.pg_get_constraintdef(oid,false)
             ),E'\n' ORDER BY conname),'UTF8')),'hex')
      FROM pg_catalog.pg_constraint
     WHERE conrelid IN (versiones,actual)
  ) IS DISTINCT FROM
    'e4a9482197ea72dbeba0b8cc8a4c47790b8cca8e52283bd2de82920481073ec4'
  OR (
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
             pg_catalog.string_agg(pg_catalog.format(
               '%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
               c.relname,i.relname,x.indkey::text,x.indclass::text,
               x.indcollation::text,x.indoption::text,x.indnkeyatts,
               x.indnatts,x.indisprimary,x.indisunique,
               pg_catalog.pg_get_indexdef(i.oid)
             ),E'\n' ORDER BY c.relname,i.relname),'UTF8')),'hex')
      FROM pg_catalog.pg_index AS x
      JOIN pg_catalog.pg_class AS c ON c.oid=x.indrelid
      JOIN pg_catalog.pg_class AS i ON i.oid=x.indexrelid
     WHERE x.indrelid IN (versiones,actual)
  ) IS DISTINCT FROM
    'd7dd73ef70b6c43829e458329820a136942808d8ad4969fa1ce6edfd309d29a5'
  THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='estructura de organizacion corporativa V1 no exacta';
  END IF;
END
$estructura$;
SQL

# 3. Fronteras byte a byte del identificador org_.
[[ $(consulta "
SELECT pg_catalog.bool_and(resultado) FROM (VALUES
 (vec_contexto_actor_v1.organizacion_ref_valida('org_'||repeat('a',16)) IS TRUE),
 (vec_contexto_actor_v1.organizacion_ref_valida('org_'||repeat('z9',40)) IS TRUE),
 (vec_contexto_actor_v1.organizacion_ref_valida(NULL) IS FALSE),
 (vec_contexto_actor_v1.organizacion_ref_valida('') IS FALSE),
 (vec_contexto_actor_v1.organizacion_ref_valida('org_'||repeat('a',15)) IS FALSE),
 (vec_contexto_actor_v1.organizacion_ref_valida('org_'||repeat('a',81)) IS FALSE),
 (vec_contexto_actor_v1.organizacion_ref_valida('org_'||repeat('A',16)) IS FALSE),
 (vec_contexto_actor_v1.organizacion_ref_valida('org_'||repeat('-',16)) IS FALSE),
 (vec_contexto_actor_v1.organizacion_ref_valida('org_'||repeat('_',16)) IS FALSE),
 (vec_contexto_actor_v1.organizacion_ref_valida('org_'||repeat('ñ',16)) IS FALSE),
 (vec_contexto_actor_v1.organizacion_ref_valida('org_'||repeat(' ',16)) IS FALSE),
 (vec_contexto_actor_v1.organizacion_ref_valida('org_'||repeat(chr(1),16)) IS FALSE)
) AS casos(resultado)") == t ]] || fallar 'limites org_ no acreditados'

# 4-6. Valores, avance comun e inmutabilidad, todo dentro de ROLLBACK.
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES (
  'prc_aaaaaaaaaaaaaaaaaaaaaa',1,repeat('a',64),
  'autoridad_maestra_acreditada'
);
INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES
 ('org_aaaaaaaaaaaaaaaa',1,'prc_aaaaaaaaaaaaaaaaaaaaaa',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',
  '2026-01-01 00:00:00+00','2027-01-01 00:00:00+00'),
 ('org_aaaaaaaaaaaaaaaa',18446744073709551615,
  'prc_aaaaaaaaaaaaaaaaaaaaaa',1,repeat('a',64),
  'autoridad_maestra_acreditada','revocado',
  '2027-01-01 00:00:00+00','2028-01-01 00:00:00+00');
INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES
 ('org_dddddddddddddddd',1,'prc_aaaaaaaaaaaaaaaaaaaaaa',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',
  '2026-01-01 00:00:00.1234567+00','2027-01-01 00:00:00.7654321+00');
DO $precision$
BEGIN
  IF (SELECT pg_catalog.to_char(vigente_desde AT TIME ZONE 'UTC','US')
        FROM vec_contexto_actor_v1.organizacion_versiones
       WHERE organizacion_ref='org_dddddddddddddddd') <> '123457'
     OR (SELECT pg_catalog.to_char(vigente_hasta AT TIME ZONE 'UTC','US')
        FROM vec_contexto_actor_v1.organizacion_versiones
       WHERE organizacion_ref='org_dddddddddddddddd') <> '765432' THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='precision timestamptz(6) no acreditada';
  END IF;
END
$precision$;

DO $invalidos$
DECLARE
  sentencia text;
  rechazado boolean;
BEGIN
  FOREACH sentencia IN ARRAY ARRAY[
    $$INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES ('org_bbbbbbbbbbbbbbbb',0,'prc_aaaaaaaaaaaaaaaaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada','activo','2026-01-01+00','2027-01-01+00')$$,
    $$INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES ('org_bbbbbbbbbbbbbbbb',18446744073709551616,'prc_aaaaaaaaaaaaaaaaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada','activo','2026-01-01+00','2027-01-01+00')$$,
    $$INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES ('org_bbbbbbbbbbbbbbbb',1,'prc_aaaaaaaaaaaaaaaaaaaaaa',1,repeat('A',64),'autoridad_maestra_acreditada','activo','2026-01-01+00','2027-01-01+00')$$,
    $$INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES ('org_bbbbbbbbbbbbbbbb',1,'prc_aaaaaaaaaaaaaaaaaaaaaa',1,repeat('a',64),'no_autoritativa','activo','2026-01-01+00','2027-01-01+00')$$,
    $$INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES ('org_bbbbbbbbbbbbbbbb',1,'prc_aaaaaaaaaaaaaaaaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada','otro','2026-01-01+00','2027-01-01+00')$$,
    $$INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES ('org_bbbbbbbbbbbbbbbb',1,'prc_aaaaaaaaaaaaaaaaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada','activo','infinity','2027-01-01+00')$$,
    $$INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES ('org_bbbbbbbbbbbbbbbb',1,'prc_aaaaaaaaaaaaaaaaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada','activo','-infinity','2027-01-01+00')$$,
    $$INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES ('org_bbbbbbbbbbbbbbbb',1,'prc_aaaaaaaaaaaaaaaaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada','activo','2026-01-01+00','infinity')$$,
    $$INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES ('org_bbbbbbbbbbbbbbbb',1,'prc_aaaaaaaaaaaaaaaaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada','activo','2026-01-01+00','-infinity')$$,
    $$INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES ('org_bbbbbbbbbbbbbbbb',1,'prc_aaaaaaaaaaaaaaaaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada','activo','2027-01-01+00','2027-01-01+00')$$,
    $$INSERT INTO vec_contexto_actor_v1.organizacion_actual VALUES ('org_aaaaaaaaaaaaaaaa',2)$$
  ] LOOP
    rechazado := false;
    BEGIN
      EXECUTE sentencia;
    EXCEPTION WHEN OTHERS THEN
      rechazado := true;
    END;
    IF NOT rechazado THEN
      RAISE EXCEPTION USING ERRCODE='55000',
        MESSAGE='valor organizativo invalido aceptado';
    END IF;
  END LOOP;
END
$invalidos$;

DO $puntero_e_inmutabilidad$
DECLARE
  inicial numeric;
  final numeric;
  sentencia text;
  rechazado boolean;
BEGIN
  SELECT generacion INTO STRICT inicial
    FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2;
  INSERT INTO vec_contexto_actor_v1.organizacion_actual
    VALUES ('org_aaaaaaaaaaaaaaaa',1);
  UPDATE vec_contexto_actor_v1.organizacion_actual
     SET version=18446744073709551615
   WHERE organizacion_ref='org_aaaaaaaaaaaaaaaa';
  SELECT generacion INTO STRICT final
    FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2;
  IF final <> inicial+2 THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='generacion comun no avanzo exactamente';
  END IF;
  FOREACH sentencia IN ARRAY ARRAY[
    $$UPDATE vec_contexto_actor_v1.organizacion_versiones SET estado='activo'$$,
    $$DELETE FROM vec_contexto_actor_v1.organizacion_versiones$$,
    $$TRUNCATE vec_contexto_actor_v1.organizacion_versiones$$,
    $$TRUNCATE vec_contexto_actor_v1.organizacion_actual$$
  ] LOOP
    rechazado := false;
    BEGIN EXECUTE sentencia;
    EXCEPTION WHEN OTHERS THEN rechazado := true;
    END;
    IF NOT rechazado THEN
      RAISE EXCEPTION USING ERRCODE='55000',
        MESSAGE='mutacion de historia o truncado aceptado';
    END IF;
  END LOOP;
  IF (SELECT pg_catalog.count(*) FROM
        vec_contexto_actor_v1.organizacion_versiones) <> 3
     OR (SELECT pg_catalog.count(*) FROM
        vec_contexto_actor_v1.organizacion_actual) <> 1
     OR (SELECT generacion FROM
        vec_contexto_actor_v1.control_generacion_punteros_actuales_v2) <> final THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='un rechazo altero datos o generacion';
  END IF;
END
$puntero_e_inmutabilidad$;
ROLLBACK;
SQL

# 7. Denegacion directa y efectiva para runtime, selector y LOGIN.
psql_admin <<'SQL'
DO $acl$
DECLARE
  rol name;
  relacion regclass;
  tipo oid;
BEGIN
  FOREACH rol IN ARRAY ARRAY[
    'vec_contexto_actor_v1_runtime',
    'vec_contexto_actor_corporativo_rrhh_selector',
    'ct140_login', 'ct140_consumidor'
  ] LOOP
    FOREACH relacion IN ARRAY ARRAY[
      'vec_contexto_actor_v1.organizacion_versiones'::regclass,
      'vec_contexto_actor_v1.organizacion_actual'::regclass
    ] LOOP
      IF pg_catalog.has_table_privilege(rol,relacion,'SELECT')
         OR pg_catalog.has_table_privilege(rol,relacion,'INSERT')
         OR pg_catalog.has_table_privilege(rol,relacion,'UPDATE')
         OR pg_catalog.has_table_privilege(rol,relacion,'DELETE')
         OR pg_catalog.has_any_column_privilege(rol,relacion,'SELECT')
         OR pg_catalog.has_any_column_privilege(rol,relacion,'UPDATE') THEN
        RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='ACL de tabla abierta';
      END IF;
      FOR tipo IN
        SELECT t.oid FROM pg_catalog.pg_type AS t
         WHERE t.typrelid=relacion OR t.typelem=(
           SELECT fila.oid FROM pg_catalog.pg_type AS fila
            WHERE fila.typrelid=relacion
         )
      LOOP
        IF pg_catalog.has_type_privilege(rol,tipo,'USAGE') THEN
          RAISE EXCEPTION USING ERRCODE='55000',
            MESSAGE='ACL efectiva de tipo fila o array abierta';
        END IF;
      END LOOP;
    END LOOP;
    IF pg_catalog.has_function_privilege(
         rol,'vec_contexto_actor_v1.organizacion_ref_valida(text)'::regprocedure,
         'EXECUTE') THEN
      RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='ACL de funcion abierta';
    END IF;
  END LOOP;
END
$acl$;
SQL

# 8. La retirada 000002 no atraviesa la migracion posterior.
exigir_fallo_intacto '000002 con 000003 instalada' retirar_000002

# 9. Opt-in, evidencia, ACL, metadatos y consumidores hostiles.
exigir_fallo_intacto 'down sin confirmacion' retirar_sin_confirmacion
exigir_fallo_intacto 'down con confirmacion incorrecta' \
  retirar_confirmacion_incorrecta

psql_admin <<'SQL'
ALTER FUNCTION vec_contexto_actor_v1.organizacion_ref_valida(text) COST 101;
SQL
exigir_fallo_intacto 'down con COST hostil' retirar
psql_admin <<'SQL'
ALTER FUNCTION vec_contexto_actor_v1.organizacion_ref_valida(text) COST 100;
COMMENT ON POLICY acceso_propietario_exacto
  ON vec_contexto_actor_v1.organizacion_actual IS 'hostil';
SQL
exigir_fallo_intacto 'down con comentario de politica hostil' retirar
psql_admin <<'SQL'
COMMENT ON POLICY acceso_propietario_exacto
  ON vec_contexto_actor_v1.organizacion_actual IS NULL;
SQL

generacion_previa=$(consulta "
SELECT generacion||'|'||actualizada_en::text
  FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2")
psql_admin <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES (
  'prc_bbbbbbbbbbbbbbbbbbbbbb',1,repeat('b',64),
  'autoridad_maestra_acreditada'
);
INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES (
  'org_bbbbbbbbbbbbbbbb',1,'prc_bbbbbbbbbbbbbbbbbbbbbb',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo',
  '2026-01-01 00:00:00+00','2027-01-01 00:00:00+00'
);
INSERT INTO vec_contexto_actor_v1.organizacion_actual
  VALUES ('org_bbbbbbbbbbbbbbbb',1);
SQL
exigir_fallo_intacto 'down con evidencia organizativa' retirar
IFS='|' read -r generacion_valor actualizada_valor <<< "$generacion_previa"
psql_admin <<SQL
SET ROLE vec_contexto_actor_v1_propietario;
ALTER TABLE vec_contexto_actor_v1.organizacion_actual DISABLE TRIGGER USER;
ALTER TABLE vec_contexto_actor_v1.organizacion_versiones DISABLE TRIGGER USER;
ALTER TABLE vec_contexto_actor_v1.procedencias DISABLE TRIGGER USER;
DELETE FROM vec_contexto_actor_v1.organizacion_actual;
DELETE FROM vec_contexto_actor_v1.organizacion_versiones;
DELETE FROM vec_contexto_actor_v1.procedencias
 WHERE procedencia_ref='prc_bbbbbbbbbbbbbbbbbbbbbb';
UPDATE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
   SET generacion=$generacion_valor,
       actualizada_en='$actualizada_valor'::timestamptz;
ALTER TABLE vec_contexto_actor_v1.organizacion_versiones ENABLE TRIGGER USER;
ALTER TABLE vec_contexto_actor_v1.organizacion_actual ENABLE TRIGGER USER;
ALTER TABLE vec_contexto_actor_v1.procedencias ENABLE TRIGGER USER;
SQL

psql_admin <<'SQL'
GRANT SELECT ON vec_contexto_actor_v1.organizacion_actual TO ct140_consumidor;
SQL
exigir_fallo_intacto 'down con ACL hostil' retirar
psql_admin <<'SQL'
REVOKE SELECT ON vec_contexto_actor_v1.organizacion_actual FROM ct140_consumidor;
CREATE INDEX ct140_indice_hostil
  ON vec_contexto_actor_v1.organizacion_actual(version);
SQL
exigir_fallo_intacto 'down con indice hostil' retirar
psql_admin <<'SQL'
DROP INDEX vec_contexto_actor_v1.ct140_indice_hostil;
COMMENT ON TABLE vec_contexto_actor_v1.organizacion_actual IS 'hostil';
SQL
exigir_fallo_intacto 'down con comentario hostil' retirar
psql_admin <<'SQL'
COMMENT ON TABLE vec_contexto_actor_v1.organizacion_actual IS NULL;
CREATE VIEW vec_contexto_actor_v1.ct140_consumidor_000004 AS
  SELECT organizacion_ref,version
    FROM vec_contexto_actor_v1.organizacion_actual;
SQL
exigir_fallo_intacto 'down ante 000004 sintetica' retirar
psql_admin <<'SQL'
DROP VIEW vec_contexto_actor_v1.ct140_consumidor_000004;
SQL

# La gramatica ASCII no depende del nombre global de la colacion C.
retirar
psql_admin <<'SQL'
ALTER COLLATION pg_catalog."C" RENAME TO "C_roto";
SQL
psql_archivo "$up"
[[ $(consulta "SELECT
  vec_contexto_actor_v1.organizacion_ref_valida('org_abcdefghijklmnop')
  AND NOT vec_contexto_actor_v1.organizacion_ref_valida(
    'org_abcdefghijklmnoñ')") == t ]] ||
  fallar 'el validador dependio de la colacion C renombrada'
retirar
psql_admin <<'SQL'
ALTER COLLATION pg_catalog."C_roto" RENAME TO "C";
SQL
psql_archivo "$up"

# 10. Retirada vacia exacta y reinstalacion, sin deriva de 000002.
retirar
[[ $(huella_estado) == "$huella_000002" ]] ||
  fallar '000003 down altero la instalacion 000002'
psql_archivo "$up"

# 11. Dos altas se serializan: exactamente una vence y no hay interbloqueo.
retirar
salida_alta_1=$(mktemp "$raiz/.ct140-alta-1.XXXXXX")
salida_alta_2=$(mktemp "$raiz/.ct140-alta-2.XXXXXX")
temporales+=("$salida_alta_1" "$salida_alta_2")
set +e
docker exec --interactive --env PGAPPNAME=ct140_alta_1 "$contenedor" \
  psql -Xq --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
  < "$raiz/$up" >"$salida_alta_1" 2>&1 &
pid_1=$!
docker exec --interactive --env PGAPPNAME=ct140_alta_2 "$contenedor" \
  psql -Xq --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
  < "$raiz/$up" >"$salida_alta_2" 2>&1 &
pid_2=$!
wait "$pid_1"; estado_1=$?
wait "$pid_2"; estado_2=$?
set -e
exitos=$(((estado_1 == 0 ? 1 : 0) + (estado_2 == 0 ? 1 : 0)))
[[ $exitos -eq 1 ]] ||
  fallar 'las altas concurrentes no produjeron un unico vencedor'
[[ $(consulta "SELECT pg_catalog.count(*) FROM pg_catalog.pg_class
 WHERE oid IN ('vec_contexto_actor_v1.organizacion_versiones'::regclass,
               'vec_contexto_actor_v1.organizacion_actual'::regclass)") == 2 ]] ||
  fallar 'alta concurrente dejo una instalacion parcial'

# Un DDL que retrocede queda delante de la fotografia; el down espera y vence.
salida_ddl=$(mktemp "$raiz/.ct140-ddl.XXXXXX")
salida_down=$(mktemp "$raiz/.ct140-down.XXXXXX")
temporales+=("$salida_ddl" "$salida_down")
docker exec --interactive --env PGAPPNAME=ct140_ddl "$contenedor" \
  psql -Xq --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
  >"$salida_ddl" 2>&1 <<'SQL' &
BEGIN;
ALTER FUNCTION vec_contexto_actor_v1.organizacion_ref_valida(text) COST 101;
SELECT pg_catalog.pg_sleep(1);
ROLLBACK;
SQL
pid_ddl=$!
esperar_sesion ct140_ddl "wait_event_type='Timeout'" 'DDL concurrente'
docker exec --interactive --env PGAPPNAME=ct140_down_ddl \
  --env PGOPTIONS="-c vec.confirmar_retirada_organizacion_corporativa_v1=RETIRAR_ORGANIZACION_CORPORATIVA_V1" \
  "$contenedor" psql -Xq --set ON_ERROR_STOP=1 \
  --username postgres --dbname "$base" < "$raiz/$down" \
  >"$salida_down" 2>&1 &
pid_down=$!
esperar_sesion ct140_down_ddl "wait_event_type='Lock'" 'down esperando DDL'
wait "$pid_ddl"
wait "$pid_down" || fallar 'down no vencio tras rollback DDL'
psql_archivo "$up"

# Insercion y avance de puntero que retroceden bloquean el down, sin ciclo.
psql_admin <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES (
  'prc_cccccccccccccccccccccc',1,repeat('c',64),
  'autoridad_maestra_acreditada'
);
SQL
salida_dml=$(mktemp "$raiz/.ct140-dml.XXXXXX")
salida_down_dml=$(mktemp "$raiz/.ct140-down-dml.XXXXXX")
temporales+=("$salida_dml" "$salida_down_dml")
docker exec --interactive --env PGAPPNAME=ct140_dml "$contenedor" \
  psql -Xq --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
  >"$salida_dml" 2>&1 <<'SQL' &
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES (
  'org_cccccccccccccccc',1,'prc_cccccccccccccccccccccc',1,repeat('c',64),
  'autoridad_maestra_acreditada','activo',
  '2026-01-01 00:00:00+00','2027-01-01 00:00:00+00'
);
INSERT INTO vec_contexto_actor_v1.organizacion_actual
  VALUES ('org_cccccccccccccccc',1);
SELECT pg_catalog.pg_sleep(1);
ROLLBACK;
SQL
pid_dml=$!
esperar_sesion ct140_dml "wait_event_type='Timeout'" 'DML concurrente'
docker exec --interactive --env PGAPPNAME=ct140_down_dml \
  --env PGOPTIONS="-c vec.confirmar_retirada_organizacion_corporativa_v1=RETIRAR_ORGANIZACION_CORPORATIVA_V1" \
  "$contenedor" psql -Xq --set ON_ERROR_STOP=1 \
  --username postgres --dbname "$base" < "$raiz/$down" \
  >"$salida_down_dml" 2>&1 &
pid_down_dml=$!
esperar_sesion ct140_down_dml "wait_event_type='Lock'" 'down esperando DML'
wait "$pid_dml"
wait "$pid_down_dml" || fallar 'down no vencio tras rollback DML'

# 12. El adaptador ejecuta los bytes completos mediante pgx y sanea siempre
# la conexion dedicada; la puerta puede omitirse en validaciones solo SQL.
psql_archivo "$up"
if [[ ${VEC_CONTEXTO_ACTOR_OMITIR_GO:-0} != 1 ]]; then
  puerto=$(docker port "$contenedor" 5432/tcp | head -n1)
  puerto=${puerto##*:}
  dsn="postgres://postgres:${clave_admin}@127.0.0.1:${puerto}/${base}?sslmode=disable"
  VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN="$dsn" \
    go test ./internal/vec/adapters/contextoactor/postgres \
      -run '^TestRetiradaOrganizacionCorporativaV1' -count=1
  VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN="$dsn" \
    go test -race ./internal/vec/adapters/contextoactor/postgres \
      -run '^TestRetiradaOrganizacionCorporativaV1' -count=1
fi

# 13. El trap acredita la limpieza incluso ante INT/TERM; aqui se comprueba
# ademas que no quedan sesiones de trabajo antes de retirar el contenedor.
[[ $(consulta "SELECT pg_catalog.count(*) FROM pg_catalog.pg_stat_activity
 WHERE datname=current_database() AND pid<>pg_catalog.pg_backend_pid()") == 0 ]] ||
  fallar 'quedaron sesiones PostgreSQL residuales'

echo 'ContextoActor: organizacion corporativa V1 superada en PostgreSQL 18.4'
