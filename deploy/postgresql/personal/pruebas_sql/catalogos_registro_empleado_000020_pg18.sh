#!/usr/bin/env bash
# Focal de Personal-20 en PostgreSQL 18. El consumidor AD3 se sustituye SOLO
# aquí para aislar catálogo, ACL, historia y transacciones; AD3-61 se prueba aparte.
set -euo pipefail
repo_dir=$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../../.." && pwd)
motor=${VEC_CONTENEDOR_MOTOR:-docker}
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-personal-cat-20-${BASHPID}"
limpiar() { "$motor" rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT
admin() { "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -o /dev/null; }
valor() { "$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
fallo() { printf 'Personal-20: %s\n' "$*" >&2; exit 1; }
"$motor" run -d --rm --name "$contenedor" -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in {1..80}; do if valor 'SELECT 1' >/dev/null 2>&1; then break; fi; sleep 0.25; done
[[ $(valor "SELECT current_setting('server_version_num')") == 180004 ]] || fallo 'se requiere PostgreSQL 18.4'
admin <<'SQL'
CREATE ROLE vec_personal_propietario NOLOGIN NOBYPASSRLS;
CREATE ROLE vec_personal_ejecutor NOLOGIN NOBYPASSRLS;
CREATE ROLE vec_personal_migrador NOLOGIN NOBYPASSRLS;
CREATE ROLE vec_cat_runtime LOGIN INHERIT NOBYPASSRLS;
CREATE ROLE vec_cat_ajeno LOGIN INHERIT NOBYPASSRLS;
GRANT vec_personal_ejecutor TO vec_cat_runtime WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT CONNECT ON DATABASE postgres TO vec_cat_runtime,vec_cat_ajeno;
CREATE SCHEMA vec_personal AUTHORIZATION vec_personal_propietario;
CREATE SCHEMA vec_autorizacion_atestada_v3;
GRANT USAGE ON SCHEMA vec_personal TO vec_personal_ejecutor;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_propietario;
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
CREATE TABLE vec_personal.registro_empleado_b2_recibo (id integer);
CREATE TABLE vec_personal.relacion_servicio_historia (id integer);
CREATE TABLE vec_personal.ocupacion_empleado_historia (id integer);
CREATE TABLE vec_personal.servicio_reconocido_historia (id integer);
CREATE TABLE vec_personal.situacion_empleado_historia (id integer);
CREATE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1() RETURNS trigger
 LANGUAGE plpgsql AS $f$ BEGIN RAISE EXCEPTION 'historia inmutable' USING ERRCODE='55000'; END $f$;
COMMIT;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_catalogo_registro_empleado_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
 LANGUAGE sql SECURITY DEFINER AS $f$
 SELECT d->>'decision_ref',d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',
 encode(sha256(gen_random_uuid()::text::bytea),'hex'),'auditoria:sintetica:catalogo',clock_timestamp(),true
 FROM (SELECT convert_from($2,'UTF8')::jsonb d) q
 $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_catalogo_registro_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_catalogo_registro_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
SQL
sql="$repo_dir/deploy/postgresql/personal/migraciones/000020_catalogos_registro_empleado.up.sql"
sed '$s/^COMMIT;/ROLLBACK;/' "$sql" | admin
[[ $(valor "SELECT to_regclass('vec_personal.entrada_catalogo_registro_empleado_historia') IS NULL") == t ]] || fallo 'ROLLBACK dejó historia'
admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
INSERT INTO vec_personal.relacion_servicio_historia VALUES (1);
COMMIT;
SQL
if admin < "$sql" >/dev/null 2>&1; then fallo 'historia previa admitida sin snapshot'; fi
[[ $(valor "SELECT to_regclass('vec_personal.entrada_catalogo_registro_empleado_historia') IS NULL") == t ]] || fallo 'rechazo dejó objetos nuevos'
admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
DELETE FROM vec_personal.relacion_servicio_historia;
COMMIT;
SQL
admin < "$sql"
[[ $(valor "SELECT has_function_privilege('vec_cat_runtime','vec_personal.registrar_entrada_catalogo_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') AND NOT has_table_privilege('vec_cat_runtime','vec_personal.entrada_catalogo_registro_empleado_historia','SELECT') AND NOT has_function_privilege('vec_cat_runtime','vec_personal.validar_entrada_registro_empleado_v1(text,text,text,integer,date)','EXECUTE') AND NOT has_function_privilege('vec_cat_ajeno','vec_personal.registrar_entrada_catalogo_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')") == t ]] || fallo 'ACL divergente'
"$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_cat_runtime -d postgres -o /dev/null < "$repo_dir/deploy/postgresql/personal/pruebas_sql/catalogos_registro_empleado_000020.sql"
[[ $(valor "SELECT bool_and(acto_ref='personal:catalogo:v3:'||encode(sha256(convert_to(decision_ref,'UTF8')),'hex') AND NOT eficacia_administrativa) AND count(*)=2 FROM vec_personal.entrada_catalogo_registro_empleado_historia") == t ]] || fallo 'procedencia interna o eficacia divergente'
"$motor" exec "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres \
  -c "BEGIN; SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:catalogo-b2:idempotencia:33333333-3333-4333-8333-333333333333',0)); SELECT pg_sleep(1.6); COMMIT;" >/dev/null &
locker_pid=$!
sleep 0.2
"$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_cat_runtime -d postgres -o /dev/null < "$repo_dir/deploy/postgresql/personal/pruebas_sql/catalogos_registro_empleado_000020_caducidad.sql"
wait "$locker_pid"
"$motor" exec "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres \
  -c "BEGIN; SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:catalogo-b2:org:sintetico:regimen:reg:carrera',0)); SELECT pg_sleep(1.6); COMMIT;" >/dev/null &
locker_pid=$!
sleep 0.2
"$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_cat_runtime -d postgres -o /dev/null < "$repo_dir/deploy/postgresql/personal/pruebas_sql/catalogos_registro_empleado_000020_caducidad.sql"
wait "$locker_pid"
[[ $(valor "SELECT count(*) FROM vec_personal.entrada_catalogo_registro_empleado_historia WHERE ref='reg:carrera'") == 0 ]] || fallo 'carrera caducada dejó efecto'
admin <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL ROLE vec_personal_propietario;
DO $neg$
BEGIN
 BEGIN
  PERFORM vec_personal.validar_entrada_registro_empleado_v1('org:sintetico','regimen','reg:sintetico',1,'2026-09-25');
  RAISE EXCEPTION 'retirada aceptada';
 EXCEPTION WHEN SQLSTATE '23514' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.validar_entrada_registro_empleado_v1('org:ajeno','regimen','reg:sintetico',1,'2026-09-25');
  RAISE EXCEPTION 'organismo ajeno aceptado';
 EXCEPTION WHEN SQLSTATE '23514' THEN NULL; END;
 BEGIN
  UPDATE vec_personal.entrada_catalogo_registro_empleado_historia SET estado='publicada';
  RAISE EXCEPTION 'historia modificada';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
END $neg$;
ROLLBACK;
SQL
"$motor" restart "$contenedor" >/dev/null
for _ in {1..80}; do if valor 'SELECT 1' >/dev/null 2>&1; then break; fi; sleep 0.25; done
[[ $(valor "SELECT count(*) FROM vec_personal.entrada_catalogo_registro_empleado_historia") == 2 ]] || fallo 'historia cambió tras reinicio'
[[ $(valor "SELECT estado FROM vec_personal.entrada_catalogo_registro_empleado_actual WHERE organismo_ref='org:sintetico' AND tipo='regimen' AND ref='reg:sintetico' AND version=1") == retirada ]] || fallo 'retirada perdida'
printf 'Personal-20 PG18: ROLLBACK/COMMIT, ACL, preimagen poblada denegada, publicación/replay/retirada/consulta, caducidad tras locks de clave y entrada, reinicio correctos (AD3 simulado).\n'
