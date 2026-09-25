#!/usr/bin/env bash
# Ensayo PostgreSQL 18.4 desechable de vec-preparar-material-interno (T4).
#
# Base: la misma cadena canónica que sonda_consumidor_ad3_000069_pg18.sh
# (ContextoActor 1/2 + Autorización 1–7 + AD3 1/2). Como en ese ensayo, el
# DBA amplía la restricción de audiencias de AD3-002 a la clave base CT y a
# las ocho audiencias B2 que dejan las migraciones posteriores, sin
# ejecutarlas. Gobierno, raíz y claves son sintéticos; no hay datos reales.
#
# LOGIN de lectura: solo puede asumir el rol propietario AD3 (SET, sin
# INHERIT), como el publicador de vec-server; la herramienta lo usa dentro
# de una transacción READ ONLY. Un segundo LOGIN sin ese rol debe fallar.
set -Eeuo pipefail
trap 'printf "T4: orden fallida en la línea %s\n" "$LINENO" >&2' ERR
repo_dir=$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
motor=${VEC_CONTENEDOR_MOTOR:-docker}
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-t4-preparar-${BASHPID}"
limpiar() { "$motor" rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT
sql() { "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres; }
sql_como() { "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U "$1" -d postgres; }
valor() { "$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
archivo() { sql < "$repo_dir/$1" >/dev/null; }
fallo() { printf 'T4: %s\n' "$*" >&2; exit 1; }

"$motor" run -d --rm --name "$contenedor" -p 127.0.0.1::5432 \
  -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in {1..120}; do valor 'SELECT 1' >/dev/null 2>&1 && break; sleep 0.25; done
[[ $(valor "SELECT current_setting('server_version_num')") == 180004 ]] || fallo 'se requiere PostgreSQL 18.4'
puerto=$("$motor" port "$contenedor" 5432/tcp | head -n1 | sed 's/.*://')

sql >/dev/null <<'SQL'
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
ca=deploy/postgresql/contexto_actor_v1
archivo "$ca/roles_up.sql"
archivo "$ca/migraciones/000001_contexto_actor_v1.up.sql"
archivo "$ca/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql"
archivo "$ca/pruebas_sql/fixtures_sinteticos.sql"
archivo deploy/postgresql/autorizacion/pruebas_sql/fixture_contexto_actor_v3.sql
sql >/dev/null <<'SQL'
CREATE EXTENSION pgcrypto WITH SCHEMA public;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;
SQL
au=deploy/postgresql/autorizacion
archivo "$au/roles_up.sql"
archivo "$au/roles_v2_up.sql"
archivo "$au/migraciones/000001_autorizacion.up.sql"
archivo deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
archivo "$au/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql"
archivo "$au/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql"
archivo "$au/migraciones/000005_registro_decisiones_contexto_actor_v3.up.sql"
archivo "$au/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.up.sql"
archivo "$au/pruebas_sql/fixture_autorizacion_contexto_actor_v3.sql"
archivo "$au/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql"
archivo deploy/postgresql/contratacion_temporal/roles_up.sql
archivo deploy/postgresql/autorizacion_atestada_v3/roles_up.sql
sql >/dev/null <<'SQL'
CREATE ROLE vec_t4_migrador LOGIN NOINHERIT;
GRANT CONNECT ON DATABASE postgres TO vec_t4_migrador;
GRANT vec_autorizacion_atestada_v3_migrador TO vec_t4_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
SQL
a3=deploy/postgresql/autorizacion_atestada_v3/migraciones
sql_como vec_t4_migrador < "$repo_dir/$a3/000001_gobierno_y_registro_v3.up.sql" >/dev/null
sql_como vec_t4_migrador < "$repo_dir/$a3/000002_consumidor_capacidad_v3.up.sql" >/dev/null
printf 'OK cadena canónica: AD3 1/2 instaladas\n'

sql >/dev/null <<'SQL'
DO $aud$
DECLARE d text;
BEGIN
  SELECT c.conname INTO STRICT d FROM pg_constraint c
   WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
     AND c.contype='c' AND pg_get_constraintdef(c.oid) LIKE '%audiencia_consumo%';
  EXECUTE format('ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT %I', d);
END $aud$;
ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  ADD CONSTRAINT prueba_t4_audiencias CHECK (audiencia_consumo = ANY (ARRAY[
   'vec_contratacion_temporal.confirmar_alta_atestada.v1',
   'vec_personal.registro_empleado.ficha.v1','vec_personal.registro_empleado.vacantes.v1',
   'vec_personal.registro_empleado.alta.v1','vec_personal.registro_empleado.hecho.v1',
   'vec_personal.registro_empleado.catalogo.consultar.v1',
   'vec_personal.registro_empleado.catalogo.publicar.v1',
   'vec_personal.registro_empleado.catalogo.retirar.v1',
   'vec_personal.registro_empleado.empleados.v1']));
CREATE ROLE vec_t4_lector LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_atestada_v3_propietario TO vec_t4_lector WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
CREATE ROLE vec_t4_sin_rol LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT CONNECT ON DATABASE postgres TO vec_t4_lector, vec_t4_sin_rol;
SQL
[[ $(valor "SELECT has_table_privilege('vec_t4_lector','vec_autorizacion_atestada_v3.clave_capacidad_version','SELECT')") == f ]] \
  || fallo 'el LOGIN lector hereda lectura sin SET ROLE'
printf 'OK LOGIN lector (solo SET ROLE propietario) y LOGIN sin rol\n'

dsn() { printf 'host=127.0.0.1 port=%s user=%s dbname=postgres sslmode=disable' "$puerto" "$1"; }
cd "$repo_dir"
VEC_T4_PG_DESECHABLE=si \
VEC_T4_PG_ADMIN_DSN="$(dsn postgres)" \
VEC_T4_PG_LECTOR_DSN="$(dsn vec_t4_lector)" \
VEC_T4_PG_SIN_ROL_DSN="$(dsn vec_t4_sin_rol)" \
TMPDIR=${TMPDIR:-/dev/shm} GOMAXPROCS=${GOMAXPROCS:-4} \
  nice go test -count=1 -run 'TestCotejoContraGobiernoPostgreSQL18' -v ./cmd/vec-preparar-material-interno/
printf 'OK T4 contra PostgreSQL 18.4 desechable\n'
