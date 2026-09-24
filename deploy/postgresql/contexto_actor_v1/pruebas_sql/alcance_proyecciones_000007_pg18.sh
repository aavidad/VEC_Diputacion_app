#!/usr/bin/env bash
# Ensayo en PostgreSQL 18.4 desechable de ContextoActor 000007 (alcance cerrado
# de proyecciones) sobre la postimagen de 000006 y Personal 000016.
# Datos exclusivamente sintéticos. El contenedor se elimina al terminar.
set -euo pipefail
base_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH='' cd -- "$base_dir/../../../.." && pwd)
motor=${VEC_CONTENEDOR_MOTOR:-docker}
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-ca-000007-${USER:-usuario}-$$"
base=vec_alcance_proyecciones_prueba
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
admin_valor() { "$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d "$base" -c "$1"; }
archivo() { admin -o /dev/null < "$1"; }
fallo() { echo "FALLO: $*" >&2; exit 1; }
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
  migraciones/000003_organizacion_corporativa_v1.up.sql roles_historicos_up.sql \
  migraciones/000005_lectura_contexto_historico_v2.up.sql; do
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

runtime() { "$motor" exec -i "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_ca_runtime_p7 -d "$base" "$@"; }
nueva_ref() { od -An -N15 -tx1 /dev/urandom | tr -d '[:space:]'; }
# resolver letra alcance [oca] [solicitado_en]: "manifiesto_hex|canon_sin_resuelto_en" o error SQL.
resolver() {
  local letra=$1 alcance=$2 sufijo=${3:-$(nueva_ref)} instante=${4:-clock_timestamp()} firma
  if [[ $alcance == heredada ]]; then firma=''; else firma=",'$alcance'::text[]"; fi
  runtime 2>&1 <<SQL
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SELECT encode(manifiesto_procedencia_canonico,'hex') || '|' ||
       regexp_replace(convert_from(representacion_canonica,'UTF8'),'"resuelto_en":"[^"]*"','')
  FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
       'oca_$sufijo','rca_$sufijo','cta_sintetica_alcance_${letra}_0000000000001',
       'prf_sintetico_alcance_${letra}_0000000000001','certificado','alto',$instante$firma);
COMMIT;
SQL
}
exigir_error() { # descripcion sqlstate_o_texto salida
  [[ $3 == *"$2"* ]] || fallo "$1: esperado '$2', obtenido: $3"
}
# acreditar oca [segundos]: imprime el instante o vacío si deniega.
acreditar() {
  local segundos=${2:-60}
  admin_valor "BEGIN ISOLATION LEVEL SERIALIZABLE;
    SELECT coalesce(vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
      r.registro_contexto_ref,'vec.contexto-actor.vinculado.v2',r.huella_sha256,
      r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,r.cuenta_ref,
      (d->>'cuenta_version')::numeric,d->>'persona_ref',(d->>'persona_version')::numeric,
      r.perfil_ref,(d->>'perfil_version')::numeric,d->>'contexto_actor_ref',
      (d->>'contexto_version')::numeric,r.metodo,r.garantia,
      clock_timestamp(),clock_timestamp()+make_interval(secs=>$segundos))::text,'')
      FROM vec_contexto_actor_v1.registros_contexto r,
           LATERAL (SELECT convert_from(r.representacion_canonica,'UTF8')::jsonb) AS x(d)
     WHERE r.operacion_ref='oca_$1';
    COMMIT;" | sed '/^$/d'
}
publicar() { # pep version emp estado [motivo] [hasta_sql]
  local motivo=${5:-NULL} hasta=${6:-"date_trunc('day',clock_timestamp())+interval '400 days'"}
  [[ $motivo == NULL ]] || motivo="'$motivo'"
  admin_valor "BEGIN; SET LOCAL ROLE vec_personal_propietario;
    SELECT version FROM vec_personal.publicar_proyeccion_empleado_persona_v1('$1',$2,'$P','$3','$4',
      date_trunc('day',clock_timestamp())-interval '1 day',$hasta,
      $motivo,'prc_personal_sintetica_p7_0000000000001',1,repeat('c',64)); COMMIT;" >/dev/null
}

# Referencias previas a 000007 (firma heredada, representación de 000006).
ref_p=$(resolver p heredada) || fallo "base CT P: $ref_p"
ref_l=$(resolver l heredada) || fallo "base L: $ref_l"
[[ $ref_l == *'"tipo":"empleado"'* ]] || fallo 'L debía incluir su puntero heredado'

echo 'ContextoActor 000007:'
# Ensayo transaccional: ROLLBACK no deja objetos ni concesiones.
sed '$s/^COMMIT;/ROLLBACK;/' "$up7" | admin -o /dev/null 2>/dev/null
[[ $(admin_valor "SELECT to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])') IS NULL
  AND NOT has_function_privilege('vec_contexto_actor_v1_propietario','vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz)','EXECUTE')") == t ]] \
  || fallo 'ROLLBACK de 000007 dejó objetos'
ok 'ROLLBACK limpio'
archivo "$up7" 2>/dev/null
if archivo "$up7" >/dev/null 2>&1; then fallo '000007 admitió una segunda aplicación'; fi
if archivo "$down7" >/dev/null 2>&1; then fallo 'DOWN de 000007 aceptado'; fi
ok 'instalación única y DOWN prohibido'

# ACL: runtime acreditado con exactamente cinco funciones; Personal solo para
# el propietario de ContextoActor; auxiliares sin EXECUTE ajeno.
[[ $(runtime -c 'SELECT acreditada FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()') == t ]] || fallo 'runtime no acreditado'
[[ $(admin_valor "SELECT has_function_privilege('vec_contexto_actor_v1_runtime','vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz)','EXECUTE')
   OR has_function_privilege('vec_ca_runtime_p7','vec_contexto_actor_v1.proyeccion_empleado_personal_v2(text,timestamptz)','EXECUTE')
   OR has_function_privilege('vec_ca_runtime_p7','vec_contexto_actor_v1.alcance_registro_contexto_v2(bytea)','EXECUTE')
   OR has_function_privilege('vec_autorizacion_propietario','vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz)','EXECUTE')
   OR has_schema_privilege('vec_ca_runtime_p7','vec_personal','USAGE')") == f ]] || fallo 'ACL de 000007 excesiva'
salida=$(runtime -c "SELECT 1 FROM vec_personal.resolver_empleado_canonico_persona_v1('$P',clock_timestamp())" 2>&1 || true)
exigir_error 'runtime leyendo Personal directamente' 'permission denied' "$salida"
ok 'ACL mínima nominal'

exigir_ct() { # caso
  local a b c
  a=$(resolver p heredada) || fallo "CT heredada ($1): $a"
  b=$(resolver p '{}') || fallo "CT alcance vacío ($1): $b"
  [[ $a == "$ref_p" && $b == "$ref_p" ]] || fallo "el contexto CT cambió: $1"
  c=$(resolver l '{}') || fallo "L alcance vacío ($1): $c"
  [[ $c == "$ref_l" ]] || fallo "la representación heredada de L cambió: $1"
}
exigir_ct 'tras instalar 000007'
ok 'alcance vacío: canon y manifiesto idénticos a 000006 (P y heredado L)'

salida=$(resolver p '{empleado,candidato}' || true); exigir_error 'alcance abierto' 'invalida' "$salida"
salida=$(resolver p '{candidato}' || true); exigir_error 'alcance desconocido' 'invalida' "$salida"
ok 'alcance cerrado: solo {} o {empleado}'

# Ausencia: pedir empleado sin proyección deniega con motivo, también con puntero heredado.
salida=$(resolver p '{empleado}' || true); exigir_error 'ausencia P' 'sin empleado canonico' "$salida"
salida=$(resolver l '{empleado}' || true); exigir_error 'ausencia L (puntero del núcleo ignorado)' 'sin empleado canonico' "$salida"
exigir_ct 'persona sin empleado'
ok 'ausencia deniega (PCA01); el puntero heredado no sustituye a Personal'

# Candidato + empleado.
publicar pep_sintetica_alcance_e1_00000000000001 1 emp_sintetico_alcance_e1_00000000000001 activa
oca_e1=$(nueva_ref)
con_emp=$(resolver p '{empleado}' "$oca_e1") || fallo "candidato+empleado: $con_emp"
[[ $con_emp == *'"tipo":"candidato","referencia":"can_sintetico_alcance_p_00000000000001"'*'"vinculo_ref":"pep_sintetica_alcance_e1_00000000000001","version":1,"tipo":"empleado","referencia":"emp_sintetico_alcance_e1_00000000000001","estado":"activo"'* ]] \
  || fallo "canon candidato+empleado inesperado: $con_emp"
manifiesto=$(printf '%s' "${con_emp%%|*}" | xxd -r -p)
[[ $manifiesto == *'"vinculo_ref":"pep_sintetica_alcance_e1_00000000000001","version":1,"tipo":"empleado","referencia":"emp_sintetico_alcance_e1_00000000000001","procedencia_ref":"prc_personal_sintetica_p7_0000000000001","procedencia_version":1,"procedencia_huella_sha256":"'"$(printf 'c%.0s' $(seq 64))"'"'* ]] \
  || fallo "manifiesto sin procedencia de Personal: $manifiesto"
[[ -n $(acreditar "$oca_e1") ]] || fallo 'acreditación del contexto con empleado'
exigir_ct 'empleado activo'
ok 'candidato+empleado: canon, manifiesto con procedencia/versión de Personal y acreditación'

# Idempotencia y cambio de alcance en replay y reconciliación.
solicitado=$(admin_valor "SELECT solicitado_en FROM vec_contexto_actor_v1.registros_contexto WHERE operacion_ref='oca_$oca_e1'")
replay=$(resolver p '{empleado}' "$oca_e1" "'$solicitado'::timestamptz") || fallo "replay: $replay"
[[ $replay == "$con_emp" ]] || fallo 'replay con el mismo alcance cambió'
salida=$(resolver p '{}' "$oca_e1" "'$solicitado'::timestamptz" || true); exigir_error 'replay con alcance vacío' 'colision' "$salida"
salida=$(resolver p heredada "$oca_e1" "'$solicitado'::timestamptz" || true); exigir_error 'replay por firma heredada' 'colision' "$salida"
reconciliar() { # oca alcance
  runtime -c "SELECT count(*) FROM vec_contexto_actor_v1.reconciliar_contexto_actor_v2(
    'oca_$1','rca_$1','cta_sintetica_alcance_p_0000000000001','prf_sintetico_alcance_p_0000000000001',
    'certificado','alto','$solicitado'::timestamptz,'$2'::text[])"
}
[[ $(reconciliar "$oca_e1" '{empleado}') == 1 && $(reconciliar "$oca_e1" '{}') == 0 ]] || fallo 'reconciliación ignora el alcance'
ok 'replay y reconciliación exigen el mismo alcance'

# Ambigüedad.
publicar pep_sintetica_alcance_e2_00000000000001 1 emp_sintetico_alcance_e2_00000000000001 activa
salida=$(resolver p '{empleado}' || true); exigir_error 'ambigüedad' 'ambigua' "$salida"
[[ -z $(acreditar "$oca_e1") ]] || fallo 'acreditación admitida con Personal ambiguo'
exigir_ct 'dos empleados canónicos'
ok 'ambigüedad deniega (PCA02) y anula el uso posterior'

# Revocación de uno de ellos: vuelve el único efectivo sin cambiar bytes.
publicar pep_sintetica_alcance_e2_00000000000001 2 emp_sintetico_alcance_e2_00000000000001 revocada error_material
[[ -n $(acreditar "$oca_e1") ]] || fallo 'acreditación tras revocar el segundo'
exigir_ct 'un empleado revocado'
# Revocación del empleado del registro: el uso posterior se deniega.
publicar pep_sintetica_alcance_e1_00000000000001 2 emp_sintetico_alcance_e1_00000000000001 revocada error_material
[[ -z $(acreditar "$oca_e1") ]] || fallo 'acreditación tras revocar el empleado registrado'
salida=$(resolver p '{empleado}' || true); exigir_error 'tras revocar' 'sin empleado canonico' "$salida"
exigir_ct 'empleado revocado'
ok 'revocación: el registro firmado deja de acreditarse y la resolución nueva deniega'

# Caducidad: vigencia que termina en segundos.
publicar pep_sintetica_alcance_e3_00000000000001 1 emp_sintetico_alcance_e3_00000000000001 activa NULL \
  "clock_timestamp()+interval '4 seconds'"
oca_e3=$(nueva_ref)
r3=$(resolver p '{empleado}' "$oca_e3") || fallo "empleado con vigencia breve: $r3"
[[ -n $(acreditar "$oca_e3" 1) ]] || fallo 'acreditación dentro de la vigencia de Personal'
[[ -z $(acreditar "$oca_e3" 60) ]] || fallo 'acreditación más allá de la vigencia de Personal'
sleep 5
[[ -z $(acreditar "$oca_e3" 1) ]] || fallo 'acreditación tras caducar'
salida=$(resolver p '{empleado}' || true); exigir_error 'caducado' 'sin empleado canonico' "$salida"
exigir_ct 'empleado caducado'
ok 'caducidad: ni uso más allá de la vigencia ni resolución tras caducar'

# Concurrencia 1: misma operación en paralelo, un único registro idéntico.
publicar pep_sintetica_alcance_e4_00000000000001 1 emp_sintetico_alcance_e4_00000000000001 activa
oca_c=$(nueva_ref); instante=$(admin_valor "SELECT clock_timestamp()")
for i in 1 2 3 4; do (resolver p '{empleado}' "$oca_c" "'$instante'::timestamptz" > "/tmp/vec_p7_$$_$i" || true) & done
wait
[[ $(admin_valor "SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto WHERE operacion_ref='oca_$oca_c'") == 1 ]] || fallo 'concurrencia duplicó el registro'
exitos=$(grep -l '^[0-9a-f]*|' /tmp/vec_p7_$$_* | while read -r f; do sed -n 1p "$f"; done | sort -u | wc -l)
rm -f /tmp/vec_p7_$$_*
[[ $exitos == 1 ]] || fallo "concurrencia devolvió respuestas distintas ($exitos)"
ok 'misma operación concurrente: un registro, una respuesta'

# Concurrencia 2: revocación en curso mientras se resuelve. La resolución
# espera el bloqueo de Personal antes del reloj; si firma con la versión
# anterior, la acreditación posterior en su instante autoritativo la anula.
oca_r=$(nueva_ref)
( admin_valor "BEGIN; SET LOCAL ROLE vec_personal_propietario;
    SELECT version FROM vec_personal.publicar_proyeccion_empleado_persona_v1('pep_sintetica_alcance_e4_00000000000001',2,'$P',
      'emp_sintetico_alcance_e4_00000000000001','revocada',date_trunc('day',clock_timestamp())-interval '1 day',
      date_trunc('day',clock_timestamp())+interval '400 days','error_material','prc_personal_sintetica_p7_0000000000001',1,repeat('c',64));
    SELECT pg_sleep(2); COMMIT;" >/dev/null ) &
sleep 0.7
rc=$(resolver p '{empleado}' "$oca_r" || true)
wait
if [[ $rc == *'|'* ]]; then
  [[ -z $(acreditar "$oca_r") ]] || fallo 'firma concurrente con versión revocada siguió acreditándose'
  ok 'revocación concurrente: firmado antes, uso posterior anulado'
else
  exigir_error 'revocación concurrente' 'sin empleado canonico' "$rc"
  ok 'revocación concurrente: resolución posterior deniega'
fi
exigir_ct 'tras concurrencia'

# Subdecisión: no hay altas nuevas de punteros de empleado; los existentes
# conservan historia y admiten versiones nuevas.
salida=$(admin -c "BEGIN; SET LOCAL ROLE vec_contexto_actor_v1_propietario;
  INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones VALUES
  ('vin_sintetico_alcance_empleado_nuevo1',1,'$P','empleado','emp_sintetico_alcance_nuevo_0000000001',
   'prc_maestra_sintetica_p7_000000000001',1,repeat('a',64),'autoridad_maestra_acreditada','activo',
   clock_timestamp()-interval '1 hour',clock_timestamp()+interval '2 hours'); COMMIT;" 2>&1 || true)
exigir_error 'alta nueva de puntero empleado' 'alta de puntero empleado cerrada' "$salida"
admin -o /dev/null <<SQL
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones
SELECT vinculo_ref,2,persona_ref,tipo,referencia,procedencia_ref,procedencia_version,procedencia_huella_sha256,
       procedencia_autoridad,'revocado',vigente_desde,vigente_hasta
  FROM vec_contexto_actor_v1.vinculo_referencia_versiones WHERE vinculo_ref='vin_sintetico_alcance_empleado_l_00001';
UPDATE vec_contexto_actor_v1.vinculo_referencia_actual SET version=2 WHERE vinculo_ref='vin_sintetico_alcance_empleado_l_00001';
COMMIT;
SQL
[[ $(admin_valor "SELECT count(*) FROM vec_contexto_actor_v1.vinculo_referencia_versiones WHERE vinculo_ref='vin_sintetico_alcance_empleado_l_00001'") == 2 ]] || fallo 'historia del puntero heredado'
ref_l=$(resolver l '{}') || fallo "L tras revocar su puntero: $ref_l"
[[ $ref_l != *'"tipo":"empleado"'* ]] || fallo 'puntero heredado revocado sigue en el contexto'
ok 'alta de puntero empleado cerrada; el heredado conserva y amplía su historia'

# Recuperación tras reinicio: registros, replay e historia intactos.
publicar pep_sintetica_alcance_e5_00000000000001 1 emp_sintetico_alcance_e5_00000000000001 activa
oca_f=$(nueva_ref)
antes=$(resolver p '{empleado}' "$oca_f") || fallo "previo al reinicio: $antes"
registros=$(admin_valor "SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto")
"$motor" restart "$contenedor" >/dev/null
esperar
[[ $(admin_valor "SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto") == "$registros" ]] || fallo 'registros tras reinicio'
solicitado=$(admin_valor "SELECT solicitado_en FROM vec_contexto_actor_v1.registros_contexto WHERE operacion_ref='oca_$oca_f'")
despues=$(resolver p '{empleado}' "$oca_f" "'$solicitado'::timestamptz") || fallo "replay tras reinicio: $despues"
[[ $despues == "$antes" ]] || fallo 'replay tras reinicio distinto'
[[ -n $(acreditar "$oca_f") ]] || fallo 'acreditación tras reinicio'
[[ $(reconciliar "$oca_f" '{empleado}') == 1 ]] || fallo 'reconciliación tras reinicio'
exigir_ct 'tras reinicio'
ok 'recuperación tras reinicio: replay, reconciliación y acreditación'

# Adaptador Go real contra la misma base (P con empleado e5, L sin proyección).
if command -v go >/dev/null 2>&1; then
  puerto=$("$motor" port "$contenedor" 5432/tcp | sed -n 's/.*:\([0-9]*\)$/\1/p' | head -1)
  salida_go=$(cd "$repo_dir" && VEC_CONTEXTO_ACTOR_ALCANCE_POSTGRES_DSN="postgres://vec_ca_runtime_p7@127.0.0.1:$puerto/$base?sslmode=disable" \
    go test -count=1 -v -run TestIntegracionPostgreSQLAlcanceProyeccionesContextoActorV2 ./internal/vec/adapters/contextoactor/postgres/ 2>&1) \
    || fallo "integración Go del adaptador con alcance: $salida_go"
  [[ $salida_go == *'--- PASS: TestIntegracionPostgreSQLAlcanceProyeccionesContextoActorV2'* ]] || fallo "integración Go omitida: $salida_go"
  exigir_ct 'tras la integración Go'
  ok 'adaptador Go: {empleado}, replay, colisión de alcance, {} y ausencia con motivo'
fi
printf 'PG18.4: ContextoActor 000007 alcance {}/{empleado}, ACL, ausencia, ambigüedad, candidato+empleado, revocación, caducidad, replay, concurrencia, subdecisión y reinicio OK; CT sin cambios.\n'
