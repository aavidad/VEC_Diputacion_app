#!/usr/bin/env bash
# Ensayo en PostgreSQL 18.4 desechable de la fachada ContextoActor 000008.
# Datos exclusivamente sintéticos. El contenedor se elimina al terminar.
set -euo pipefail
base_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH='' cd -- "$base_dir/../../../.." && pwd)
motor=${VEC_CONTENEDOR_MOTOR:-docker}
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-ca-b2-persona-${USER:-usuario}-$$"
base=vec_persona_tercero_b2_prueba
up7="$repo_dir/deploy/postgresql/contexto_actor_v1/migraciones/000007_alcance_proyecciones_empleado.up.sql"
down7="$repo_dir/deploy/postgresql/contexto_actor_v1/migraciones/000007_alcance_proyecciones_empleado.down.sql"
limpiar() { "$motor" rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT INT TERM

esperar() {
  for _ in $(seq 1 150); do
    if "$motor" exec "$contenedor" psql -X -qAt -U postgres -d "$base" -c 'SELECT 1' >/dev/null 2>&1; then
      sleep 0.5
      "$motor" exec "$contenedor" psql -X -qAt -U postgres -d "$base" -c 'SELECT 1' >/dev/null 2>&1 && return 0
    fi
    sleep 0.3
  done
  echo 'PostgreSQL no quedó disponible' >&2; return 1
}
admin() { "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d "$base" "$@"; }
admin_valor() { "$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -v VERBOSITY=verbose -U postgres -d "$base" -c "$1"; }
archivo() { admin -o /dev/null < "$1"; }
fallo() { echo "FALLO: $*" >&2; exit 1; }
# 000004 selló la huella del manifiesto de su predecesor antes de que
# 203bf293 ampliara privilegios_efectivos_runtime_minimos en 000001; una base
# nueva ya no la reproduce. Deuda previa, ajena a este corte: el ensayo aplica
# 000004 sustituyendo solo esa huella esperada por la de la base actual.
aplicar_000004() {
  sed 's/bddc55742ae4d509cb884bbf464ac4f90c23c6b680d338943160ea1ee3b1742c/613d1837116a903bc49accb0f63d25a3ce2094421888daa915674a1546e4cbe3/' \
    "$repo_dir/deploy/postgresql/contexto_actor_v1/migraciones/000004_vinculo_corporativo_rrhh_v1.up.sql" | admin -o /dev/null
}
ok() { echo "  ok: $*"; }

"$motor" run -d --rm --name "$contenedor" -e POSTGRES_DB="$base" -p 127.0.0.1::5432 \
  -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
esperar
[[ $(admin_valor "SELECT current_setting('server_version_num')") == 180004 ]] || fallo 'no es PostgreSQL 18.4'

admin -o /dev/null <<'SQL'
DO $x$ BEGIN
  CREATE ROLE vec_p7_dueno_base NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
  EXECUTE format('ALTER DATABASE %I OWNER TO vec_p7_dueno_base', current_database());
  EXECUTE format('REVOKE ALL ON DATABASE %I FROM PUBLIC', current_database());
END $x$;
REVOKE ALL ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL

for f in roles_up.sql migraciones/000001_contexto_actor_v1.up.sql \
  migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql \
  roles_contexto_corporativo_rrhh_selector_v1_up.sql \
  migraciones/000003_organizacion_corporativa_v1.up.sql; do
  archivo "$repo_dir/deploy/postgresql/contexto_actor_v1/$f"
done
aplicar_000004
for f in roles_historicos_up.sql migraciones/000005_lectura_contexto_historico_v2.up.sql; do
  archivo "$repo_dir/deploy/postgresql/contexto_actor_v1/$f"
done
admin -o /dev/null <<'SQL'
CREATE ROLE vec_autorizacion_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
  text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,
  text,text,timestamptz,timestamptz) TO vec_autorizacion_propietario;
COMMIT;
SQL
archivo "$repo_dir/deploy/postgresql/contexto_actor_v1/migraciones/000006_vinculos_efectivos_temporales.up.sql"

# Sin Personal 000016, 000007 se rechaza por contrato ausente.
if archivo "$up7" >/dev/null 2>&1; then fallo '000007 aceptó una base sin Personal 000016'; fi
archivo "$repo_dir/deploy/postgresql/personal/roles_up.sql"
archivo "$repo_dir/deploy/postgresql/personal/migraciones/000016_proyeccion_empleado_persona.up.sql"

# ---------------------------------------------------------------------------
# Datos sintéticos, todos con procedencia autoritativa de ensayo.
#  P: RRHH/CT con candidato; recibirá proyecciones de Personal.
#  L: persona con puntero de empleado del núcleo anterior a 000007 (heredado).
# ---------------------------------------------------------------------------
P=per_sintetica_alcance_p_00000000000001
L=per_sintetica_alcance_l_00000000000001
alta_actor() { # letra persona
  cat <<SQL
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_sintetica_alcance_$1_0000000000001',1,'prc_maestra_sintetica_p7_000000000001',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES ('cta_sintetica_alcance_$1_0000000000001',1);
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('$2',1,'prc_maestra_sintetica_p7_000000000001',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
INSERT INTO vec_contexto_actor_v1.persona_actual VALUES ('$2',1);
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_sintetico_alcance_$1_0000000000001',1,'$2','prc_maestra_sintetica_p7_000000000001',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES ('prf_sintetico_alcance_$1_0000000000001',1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_sintetico_alcance_$1_0000000000001',1,'cta_sintetica_alcance_$1_0000000000001',
  'prf_sintetico_alcance_$1_0000000000001','$2','prc_maestra_sintetica_p7_000000000001',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES ('vca_sintetico_alcance_$1_0000000000001',1);
SQL
}
admin -o /dev/null <<SQL
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES
 ('prc_maestra_sintetica_p7_000000000001',1,repeat('a',64),'autoridad_maestra_acreditada');
$(alta_actor p "$P")
$(alta_actor l "$L")
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones VALUES
 ('vin_sintetico_alcance_candidato_p_0001',1,'$P','candidato','can_sintetico_alcance_p_00000000000001',
  'prc_maestra_sintetica_p7_000000000001',1,repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours'),
 ('vin_sintetico_alcance_empleado_l_00001',1,'$L','empleado','emp_sintetico_alcance_l_00000000000001',
  'prc_maestra_sintetica_p7_000000000001',1,repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_actual VALUES
 ('vin_sintetico_alcance_candidato_p_0001',1),('vin_sintetico_alcance_empleado_l_00001',1);
COMMIT;
CREATE ROLE vec_ca_runtime_p7 LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contexto_actor_v1_runtime TO vec_ca_runtime_p7 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL

up8="$repo_dir/deploy/postgresql/contexto_actor_v1/migraciones/000008_acreditacion_persona_tercero.up.sql"
persona_valor() { admin_valor "BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL ROLE vec_personal_propietario;
    SELECT persona_ref||'|'||persona_version||'|'||procedencia_ref||'|'||procedencia_version||'|'||procedencia_huella_sha256
      FROM vec_contexto_actor_v1.acreditar_persona_tercero_v1('$1'); COMMIT;"; }
fallo_sql() {
  local etiqueta=$1 patron=$2 sentencia=$3 salida
  salida=$(admin_valor "$sentencia" 2>&1) && fallo "$etiqueta aceptado"
  [[ $salida == *"$patron"* ]] || fallo "$etiqueta: error inesperado: $salida"
}
esperar_sueno() {
  local _
  for _ in $(seq 1 100); do
    [[ $(admin_valor "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database()
      AND state='active' AND query LIKE '%SELECT pg_sleep(2)%' AND query NOT LIKE '%pg_stat_activity%')") == t ]] && return 0
    sleep 0.05
  done
  fallo 'no se observo la transaccion de carrera'
}

echo 'ContextoActor 000008: persona tercero B2'
archivo "$up7" 2>/dev/null
# Una aplicacion abortada no deja ACL ni funcion.
sed '$s/^COMMIT;/ROLLBACK;/' "$up8" | admin -o /dev/null
[[ $(admin_valor "SELECT to_regprocedure('vec_contexto_actor_v1.acreditar_persona_tercero_v1(text)') IS NULL") == t ]] ||
  fallo 'ROLLBACK dejo funcion'
archivo "$up8" 2>/dev/null
if archivo "$up8" >/dev/null 2>&1; then fallo 'segunda aplicacion 000008 admitida'; fi
ok 'preimagen, ROLLBACK, instalacion unica'

func='vec_contexto_actor_v1.acreditar_persona_tercero_v1(text)'
[[ $(admin_valor "SELECT has_function_privilege('vec_personal_propietario','$func','EXECUTE')
  AND NOT has_function_privilege('vec_personal_ejecutor','$func','EXECUTE')
  AND NOT has_function_privilege('vec_contexto_actor_v1_runtime','$func','EXECUTE')
  AND NOT has_function_privilege('vec_autorizacion_propietario','$func','EXECUTE')
  AND has_schema_privilege('vec_personal_propietario','vec_contexto_actor_v1','USAGE')") == t ]] ||
  fallo 'ACL nominal divergente'
[[ $(admin_valor "SELECT p.prosecdef AND p.provolatile='v'
    AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on']::text[]
    FROM pg_proc p WHERE p.oid='$func'::regprocedure") == t ]] ||
  fallo 'funcion sin frontera SECURITY DEFINER esperada'
fallo_sql 'ejecutor Personal' 'permission denied' "BEGIN ISOLATION LEVEL SERIALIZABLE;
  SET LOCAL ROLE vec_personal_ejecutor;
  SELECT * FROM vec_contexto_actor_v1.acreditar_persona_tercero_v1('$P'); COMMIT;"
fallo_sql 'runtime' 'permission denied' "BEGIN ISOLATION LEVEL SERIALIZABLE;
  SET LOCAL ROLE vec_contexto_actor_v1_runtime;
  SELECT * FROM vec_contexto_actor_v1.acreditar_persona_tercero_v1('$P'); COMMIT;"
ok 'ACL y SECURITY DEFINER'

esperado="$P|1|prc_maestra_sintetica_p7_000000000001|1|$(printf 'a%.0s' {1..64})"
[[ $(persona_valor "$P") == "$esperado" ]] || fallo 'persona vigente no acreditada'
fallo_sql 'READ COMMITTED' '25000' "BEGIN; SET LOCAL ROLE vec_personal_propietario;
  SELECT * FROM vec_contexto_actor_v1.acreditar_persona_tercero_v1('$P'); COMMIT;"
fallo_sql 'READ ONLY' '25000' "BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
  SET LOCAL ROLE vec_personal_propietario;
  SELECT * FROM vec_contexto_actor_v1.acreditar_persona_tercero_v1('$P'); COMMIT;"
fallo_sql 'persona ausente' 'P0002' "BEGIN ISOLATION LEVEL SERIALIZABLE;
  SET LOCAL ROLE vec_personal_propietario;
  SELECT * FROM vec_contexto_actor_v1.acreditar_persona_tercero_v1('per_sintetica_ausente_b2_00000000001'); COMMIT;"
fallo_sql 'referencia invalida' '22023' "BEGIN ISOLATION LEVEL SERIALIZABLE;
  SET LOCAL ROLE vec_personal_propietario;
  SELECT * FROM vec_contexto_actor_v1.acreditar_persona_tercero_v1('DNI123'); COMMIT;"
ok 'vigencia, negativos de aislamiento y formato'

# Snapshot anterior al COMMIT de revocacion: el control MVCC debe forzar 40001.
salida_vieja=$(mktemp)
(admin_valor "BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL ROLE vec_personal_propietario;
  SELECT count(*) FROM pg_catalog.pg_class;
  SELECT pg_sleep(2);
  SELECT * FROM vec_contexto_actor_v1.acreditar_persona_tercero_v1('$P'); COMMIT;" >"$salida_vieja" 2>&1) &
vieja_pid=$!
esperar_sueno
admin_valor "BEGIN; SET LOCAL ROLE vec_contexto_actor_v1_propietario;
  INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
  ('$P',2,'prc_maestra_sintetica_p7_000000000001',1,repeat('a',64),
   'autoridad_maestra_acreditada','revocado',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
  UPDATE vec_contexto_actor_v1.persona_actual SET version=2 WHERE persona_ref='$P'; COMMIT;" >/dev/null
wait "$vieja_pid" || true
rg -q '40001' "$salida_vieja" || fallo "snapshot obsoleto no dio 40001: $(cat "$salida_vieja")"
rm -f "$salida_vieja"
fallo_sql 'revocado' 'P0002' "BEGIN ISOLATION LEVEL SERIALIZABLE;
  SET LOCAL ROLE vec_personal_propietario;
  SELECT * FROM vec_contexto_actor_v1.acreditar_persona_tercero_v1('$P'); COMMIT;"
ok 'revocacion y snapshot obsoleto'

# Un mutador que llega tras acreditar espera el advisory compartido hasta COMMIT.
retencion=$(mktemp)
(admin_valor "BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL ROLE vec_personal_propietario;
  SELECT persona_version FROM vec_contexto_actor_v1.acreditar_persona_tercero_v1('$L');
  SELECT pg_sleep(2); COMMIT;" >"$retencion" 2>&1) &
retenida_pid=$!
esperar_sueno
revocacion=$(mktemp)
(admin_valor "BEGIN; SET LOCAL ROLE vec_contexto_actor_v1_propietario;
  INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
  ('$L',2,'prc_maestra_sintetica_p7_000000000001',1,repeat('a',64),
   'autoridad_maestra_acreditada','revocado',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
  UPDATE vec_contexto_actor_v1.persona_actual SET version=2 WHERE persona_ref='$L'; COMMIT;" >"$revocacion" 2>&1) &
revocacion_pid=$!
sleep 0.4
[[ $(admin_valor "SELECT version FROM vec_contexto_actor_v1.persona_actual WHERE persona_ref='$L'") == 1 ]] ||
  fallo 'revocacion paso antes de COMMIT de acreditacion'
wait "$retenida_pid"
wait "$revocacion_pid"
rm -f "$retencion" "$revocacion"
[[ $(admin_valor "SELECT version FROM vec_contexto_actor_v1.persona_actual WHERE persona_ref='$L'") == 2 ]] ||
  fallo 'revocacion no se publico despues de COMMIT'
ok 'serializacion de revocacion posterior'

"$motor" restart "$contenedor" >/dev/null
esperar
[[ $(admin_valor "SELECT count(*) FROM vec_contexto_actor_v1.persona_actual WHERE version=2") == 2 ]] ||
  fallo 'historia de revocacion no persistio tras reinicio'
[[ $(admin_valor "SELECT has_function_privilege('vec_personal_propietario','$func','EXECUTE')") == t ]] ||
  fallo 'ACL no persistio tras reinicio'
ok 'reinicio conserva historia y ACL'
