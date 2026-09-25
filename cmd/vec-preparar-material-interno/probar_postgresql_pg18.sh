#!/usr/bin/env bash
# Ensayo PostgreSQL 18.4 desechable de vec-preparar-material-interno (T4).
#
# Base: la misma cadena canónica que sonda_consumidor_ad3_000069_pg18.sh
# (ContextoActor 1/2 + Autorización 1–7 + AD3 1/2/50a/53a/69). Como en ese
# ensayo, el DBA amplía la restricción de audiencias de AD3-002 a la clave
# base CT, las cinco de CT interno, despacho de correo y las ocho B2 que dejan
# las migraciones posteriores, sin ejecutarlas. Material de idempotencia,
# gobierno, raíz y claves son sintéticos; no hay datos reales.
#
# Nadie toca checkpoint_gobierno.revision: solo lo mueven los triggers reales.
# El publicador de vec-server numera las claves por encima de ese contador y
# AD3-69 (v2) y la herramienta deben aceptarlas de inmediato (consenso B2
# R3–R5); v1, intacta, sigue rechazando CT.
#
# No se crea ningún LOGIN nuevo para la herramienta: se reutiliza el LOGIN de
# gobierno de vec-server, miembro de vec_autorizacion_atestada_v3_migrador
# (que puede asumir el propietario AD3 con SET, sin INHERIT). El ensayo crea
# un LOGIN así (vec_t4_gobierno) y con él:
#   1. publica con el publicador real de vec-server la clave base CT, las
#      cinco claves que lee CT interno y las ocho B2: primero con la
#      configuración de ayer y después con la de hoy (renovación), repetida,
#      concurrente, con un rollback y un salto de versión; v2('ct') y
#      v2('personal_b2') aceptan de inmediato con el LOGIN nominal de
#      preflight, y la consulta exacta de vec-interno para CT también;
#   2. repite la publicación B2 como un arranque de vec-server;
#   3. reinicia PostgreSQL, republica (idempotente), vuelve a aceptar y
#      comprueba los negativos que alteran el gobierno (mínimos de
#      configuración y raíz retrocedidos, clave sustituida y revocada), solo
#      sobre CT; y
#   4. ejecuta la herramienta sobre ese mismo material y gobierno, que debe
#      aceptarlo sin ajustar el checkpoint, más los casos negativos de
#      identidad, material divergente, revocación programada y puntero rotado.
set -Eeuo pipefail
trap 'printf "T4: orden fallida en la línea %s\n" "$LINENO" >&2' ERR
repo_dir=$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
motor=${VEC_CONTENEDOR_MOTOR:-docker}
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-t4-preparar-${BASHPID}"
material=$(mktemp -d "${TMPDIR:-/dev/shm}/vec-t4-material.XXXXXX")
limpiar() {
  "$motor" rm -f "$contenedor" >/dev/null 2>&1 || true
  rm -rf -- "$material"
}
trap limpiar EXIT
sql() { "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres; }
sql_como() { "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U "$1" -d postgres; }
valor() { "$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
archivo() { sql < "$repo_dir/$1" >/dev/null; }
fallo() { printf 'T4: %s\n' "$*" >&2; exit 1; }

# Material de idempotencia sintético con el formato de
# scripts/generar_credenciales_desarrollo.sh (generaciones 2 y 1), fuera de
# cualquier repositorio: <material>/idempotencia, 0700, ficheros 0600.
chmod 0700 "$material"
install -d -m 0700 "$material/idempotencia"
umask 077
printf '%s' '{"version":1,"esquema":"vec.bolsa.convocatoria.idempotencia-hmac.desarrollo.v1","autoridad":"no_autoritativo","version_esquema_hmac":2,"generaciones":[{"generacion":2,"referencia_localizador":"clave:hmac:convocatorias:localizador:desarrollo:v2","referencia_huella_solicitud":"clave:hmac:convocatorias:huella:desarrollo:v2"},{"generacion":1,"referencia_localizador":"clave:hmac:convocatorias:localizador:desarrollo:v1","referencia_huella_solicitud":"clave:hmac:convocatorias:huella:desarrollo:v1"}]}' \
  >"$material/idempotencia/configuracion.json"
for g in 2 1; do
  for dominio in localizador huella-solicitud; do
    head -c 32 /dev/urandom >"$material/idempotencia/g$g-$dominio.bin"
  done
done
chmod 0600 "$material"/idempotencia/*

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
archivo "$a3/000050a_preflight_material_interno.up.sql"
archivo "$a3/000053a_lectura_configuracion_interna.up.sql"
archivo "$a3/000069_sonda_lectura_gobierno_por_consumidor.up.sql"
printf 'OK cadena canónica: AD3 1/2/50a/53a/69 instaladas\n'

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
   'vec_personal.alta_ejercicio.v1','vec_personal.lectura_incorporacion.v2',
   'vec_contratacion_temporal.incorporacion_ejercicio.v2',
   'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
   'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
   'vec_contratacion_temporal.despacho_correo_llamamiento.v1',
   'vec_personal.registro_empleado.ficha.v1','vec_personal.registro_empleado.vacantes.v1',
   'vec_personal.registro_empleado.alta.v1','vec_personal.registro_empleado.hecho.v1',
   'vec_personal.registro_empleado.catalogo.consultar.v1',
   'vec_personal.registro_empleado.catalogo.publicar.v1',
   'vec_personal.registro_empleado.catalogo.retirar.v1',
   'vec_personal.registro_empleado.empleados.v1']));
-- LOGIN de gobierno como el de vec-server: atributos que exige su pool y
-- una sola membresía, el grupo migrador (SET sin INHERIT).
CREATE ROLE vec_t4_gobierno LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_atestada_v3_migrador TO vec_t4_gobierno WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
-- LOGIN que la herramienta debe rechazar por identidad.
CREATE ROLE vec_t4_sin_rol LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_t4_bypass LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION BYPASSRLS;
CREATE ROLE vec_t4_createrole LOGIN INHERIT NOSUPERUSER NOCREATEDB CREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_t4_extra LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_t4_noinherit LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_atestada_v3_migrador TO vec_t4_bypass, vec_t4_createrole, vec_t4_extra, vec_t4_noinherit
  WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
GRANT vec_autorizacion_atestada_v3_consumidor TO vec_t4_extra WITH ADMIN FALSE, INHERIT FALSE, SET FALSE;
GRANT CONNECT ON DATABASE postgres TO vec_t4_gobierno, vec_t4_sin_rol, vec_t4_bypass, vec_t4_createrole, vec_t4_extra, vec_t4_noinherit;
-- LOGIN nominal de preflight de vec-interno (AD3-50a/53a/69).
CREATE ROLE vec_interno_preflight_v3_desarrollo LOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_atestada_v3_preflight_interno TO vec_interno_preflight_v3_desarrollo
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT CONNECT ON DATABASE postgres TO vec_interno_preflight_v3_desarrollo;
SQL
[[ $(valor "SELECT has_table_privilege('vec_t4_gobierno','vec_autorizacion_atestada_v3.clave_capacidad_version','SELECT')") == f ]] \
  || fallo 'el LOGIN de gobierno hereda lectura sin SET ROLE'
printf 'OK LOGIN de gobierno como vec-server y LOGIN no admitidos\n'

dsn() { printf 'host=127.0.0.1 port=%s user=%s dbname=postgres sslmode=disable' "$puerto" "$1"; }
exportar_dsn() {
  export VEC_T4_PG_GOBIERNO_DSN="$(dsn vec_t4_gobierno)"
  export VEC_T4_PG_ADMIN_DSN="$(dsn postgres)"
  export VEC_T4_PG_PREFLIGHT_DSN="$(dsn vec_interno_preflight_v3_desarrollo)"
}
checkpoint() { valor "SELECT revision||'/'||configuracion_secuencia_minima||'/'||raiz_version_minima FROM vec_autorizacion_atestada_v3.checkpoint_gobierno"; }
cd "$repo_dir"
export VEC_T4_PG_DESECHABLE=si
export VEC_T4_MATERIAL_IDEMPOTENCIA="$material/idempotencia"
export VEC_AD369_MATERIAL_CT="$material/material_ct_publico.json"
export TMPDIR=${TMPDIR:-/dev/shm} GOMAXPROCS=${GOMAXPROCS:-4}
exportar_dsn
printf 'checkpoint inicial (revisión/secuencia mínima/raíz mínima): %s\n' "$(checkpoint)"

VEC_AD369_FASE=publicar nice go test -count=1 -run 'TestGobiernoV2AceptaClavesRecienPublicadasSinCheckpointPostgreSQL18' -v ./internal/app/bootstrap/
printf 'OK CT5 + B2-8 publicadas por vec-server (ayer→hoy, repetición, concurrencia, rollback, salto): v2 acepta sin tocar el checkpoint; v1 rechaza CT\n'
nice go test -count=1 -run 'TestLecturaCTV2ContraGobiernoPublicadoPostgreSQL18' -v ./internal/app/composicion/internactproveedores/
printf 'OK consulta exacta de vec-interno CT (v2 ct) sobre el gobierno publicado\n'

nice go test -count=1 -run 'TestPublicaGobiernoB2ComoVecServerParaEnsayoPostgreSQL18' -v ./internal/app/bootstrap/
[[ $(valor "SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE audiencia_consumo LIKE 'vec_personal.registro_empleado.%'") == 8 ]] \
  || fallo 'vec-server no publicó las ocho claves B2'
printf 'OK gobierno publicado con el publicador real de vec-server; checkpoint: %s\n' "$(checkpoint)"

"$motor" restart "$contenedor" >/dev/null
for _ in {1..120}; do valor 'SELECT 1' >/dev/null 2>&1 && break; sleep 0.25; done
puerto=$("$motor" port "$contenedor" 5432/tcp | head -n1 | sed 's/.*://')
exportar_dsn
VEC_AD369_FASE=reinicio nice go test -count=1 -run 'TestGobiernoV2AceptaClavesRecienPublicadasSinCheckpointPostgreSQL18' -v ./internal/app/bootstrap/
printf 'OK tras reiniciar PostgreSQL: republicación idempotente, v2 acepta; mínimos retrocedidos, clave sustituida y revocada rechazados\n'

VEC_T4_PG_SIN_ROL_DSN="$(dsn vec_t4_sin_rol)" \
VEC_T4_PG_SUPERUSUARIO_DSN="$(dsn postgres)" \
VEC_T4_PG_BYPASSRLS_DSN="$(dsn vec_t4_bypass)" \
VEC_T4_PG_CREATEROLE_DSN="$(dsn vec_t4_createrole)" \
VEC_T4_PG_MEMBRESIA_EXTRA_DSN="$(dsn vec_t4_extra)" \
VEC_T4_PG_NOINHERIT_DSN="$(dsn vec_t4_noinherit)" \
  nice go test -count=1 -run 'TestCotejoContraGobiernoPostgreSQL18' -v ./cmd/vec-preparar-material-interno/
printf 'OK T4 contra PostgreSQL 18.4 desechable; checkpoint final: %s\n' "$(checkpoint)"
