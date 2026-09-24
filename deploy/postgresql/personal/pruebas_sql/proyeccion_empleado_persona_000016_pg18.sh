#!/usr/bin/env bash
# Ensayo en PostgreSQL 18.4 desechable de Personal 000016 y regresión del
# incidente del 23/09/2026: publicar un empleado para la persona que usa
# Contratación temporal no puede alterar ni anular su contexto de actor.
# Datos exclusivamente sintéticos. El contenedor se elimina al terminar.
set -euo pipefail
base_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH='' cd -- "$base_dir/../../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-personal-000016-${USER:-usuario}-$$"
base=vec_proyeccion_empleado_prueba
clave_runtime=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
limpiar() { docker rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT INT TERM

esperar() {
  for _ in $(seq 1 120); do
    if docker exec "$contenedor" psql -X -qAt -U postgres -d "$base" -c 'SELECT 1' >/dev/null 2>&1; then
      sleep 0.3
      docker exec "$contenedor" psql -X -qAt -U postgres -d "$base" -c 'SELECT 1' >/dev/null 2>&1 && return 0
    fi
    sleep 0.3
  done
  echo 'PostgreSQL no quedó disponible' >&2; return 1
}
admin() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d "$base" "$@"; }
admin_valor() { docker exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d "$base" -c "$1"; }
archivo() { admin -o /dev/null < "$1"; }
fallo() { echo "FALLO: $*" >&2; exit 1; }

docker run -d --rm --name "$contenedor" -e POSTGRES_DB="$base" \
  -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
esperar
[[ $(admin_valor "SELECT current_setting('server_version_num')") == 180004 ]] || fallo 'no es PostgreSQL 18.4'

# Base dedicada preendurecida, como exige ContextoActor.
admin <<'SQL'
DO $b$ BEGIN EXECUTE format('REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',current_database()); END $b$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
archivo "$repo_dir/deploy/postgresql/personal/roles_up.sql"
archivo "$repo_dir/deploy/postgresql/contexto_actor_v1/roles_up.sql"
archivo "$repo_dir/deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql"
archivo "$repo_dir/deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql"

# 000016: ensayo de reversión por ROLLBACK, instalación y no reaplicación.
sed '$s/^COMMIT;/ROLLBACK;/' "$repo_dir/deploy/postgresql/personal/migraciones/000016_proyeccion_empleado_persona.up.sql" | admin -o /dev/null
[[ $(admin_valor "SELECT to_regclass('vec_personal.proyeccion_empleado_persona_historia') IS NULL") == t ]] || fallo 'ROLLBACK dejó objetos'
archivo "$repo_dir/deploy/postgresql/personal/migraciones/000016_proyeccion_empleado_persona.up.sql"
if archivo "$repo_dir/deploy/postgresql/personal/migraciones/000016_proyeccion_empleado_persona.up.sql" >/dev/null 2>&1; then
  fallo '000016 admitió una segunda aplicación'
fi
# DOWN sin historia retira; se reinstala para los casos.
archivo "$repo_dir/deploy/postgresql/personal/migraciones/000016_proyeccion_empleado_persona.down.sql"
[[ $(admin_valor "SELECT to_regclass('vec_personal.proyeccion_empleado_persona_historia') IS NULL") == t ]] || fallo 'DOWN vacío no retiró'
archivo "$repo_dir/deploy/postgresql/personal/migraciones/000016_proyeccion_empleado_persona.up.sql"
admin -o /dev/null < "$base_dir/proyeccion_empleado_persona_000016_casos.sql"

# ---------------------------------------------------------------------------
# Regresión del incidente sobre el resolutor ContextoActor V2 real.
# Persona de RRHH sintética con contexto CT y un candidato de Bolsa activo.
# ---------------------------------------------------------------------------
per=per_sintetica_rrhh_ct_000000000000000001
admin <<SQL
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES
 ('prc_maestra_sintetica_ct_0000000000001',1,repeat('a',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_sintetica_rrhh_ct_00000000000000001',1,'prc_maestra_sintetica_ct_0000000000001',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '2 hours');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES ('cta_sintetica_rrhh_ct_00000000000000001',1);
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('$per',1,'prc_maestra_sintetica_ct_0000000000001',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '2 hours');
INSERT INTO vec_contexto_actor_v1.persona_actual VALUES ('$per',1);
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_sintetico_rrhh_ct_000000000000000001',1,'$per','prc_maestra_sintetica_ct_0000000000001',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '2 hours');
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES ('prf_sintetico_rrhh_ct_000000000000000001',1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_sintetico_rrhh_ct_000000000000000001',1,'cta_sintetica_rrhh_ct_00000000000000001',
  'prf_sintetico_rrhh_ct_000000000000000001','$per','prc_maestra_sintetica_ct_0000000000001',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '2 hours');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES ('vca_sintetico_rrhh_ct_000000000000000001',1);
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones VALUES
 ('vin_sintetico_candidato_ct_0000000000001',1,'$per','candidato','can_sintetico_candidato_ct_0000000000001',
  'prc_maestra_sintetica_ct_0000000000001',1,repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '2 hours');
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_actual VALUES ('vin_sintetico_candidato_ct_0000000000001',1);
COMMIT;
SQL
docker exec -i -e CLAVE="$clave_runtime" "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
\getenv clave CLAVE
CREATE ROLE vec_contexto_actor_runtime_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS PASSWORD :'clave';
GRANT vec_contexto_actor_v1_runtime TO vec_contexto_actor_runtime_prueba WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL
[[ $(docker exec "$contenedor" psql -X -qAt -U vec_contexto_actor_runtime_prueba -d "$base" \
  -c 'SELECT acreditada FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()') == t ]] || fallo 'runtime no acreditado'

# Devuelve "manifiesto_hex|representacion_sin_resuelto_en" o falla con el error SQL.
resolver_ct() {
  local sufijo; sufijo=$(od -An -N15 -tx1 /dev/urandom | tr -d '[:space:]')
  docker exec -i "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_contexto_actor_runtime_prueba -d "$base" <<SQL
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SELECT encode(manifiesto_procedencia_canonico,'hex') || '|' ||
       regexp_replace(convert_from(representacion_canonica,'UTF8'),'"resuelto_en":"[^"]*"','')
  FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
       'oca_$sufijo','rca_$sufijo','cta_sintetica_rrhh_ct_00000000000000001',
       'prf_sintetico_rrhh_ct_000000000000000001','certificado','alto',clock_timestamp());
COMMIT;
SQL
}
publicar() { # pep version emp estado motivo
  local motivo=${5:-NULL}; [[ $motivo == NULL ]] || motivo="'$motivo'"
  admin_valor "BEGIN; SET LOCAL ROLE vec_personal_propietario;
    SELECT version FROM vec_personal.publicar_proyeccion_empleado_persona_v1('$1',$2,'$per','$3','$4',
      date_trunc('day',clock_timestamp())-interval '1 day',date_trunc('day',clock_timestamp())+interval '400 days',
      $motivo,'prc_personal_sintetica_ct_0000000000001',1,repeat('c',64)); COMMIT;" >/dev/null
}
clase_personal() {
  admin_valor "SELECT resultado FROM vec_personal.resolver_empleado_canonico_persona_v1('$per',clock_timestamp())"
}
exigir_ct_igual() { # caso clase_personal_esperada
  local actual
  actual=$(resolver_ct) || fallo "CT dejó de resolver: $1"
  [[ $actual == "$referencia" ]] || fallo "el contexto CT cambió: $1"
  [[ $(clase_personal) == "$2" ]] || fallo "Personal no está en $2: $1"
  echo "  CT idéntico con Personal en '$2': $1"
}

referencia=$(resolver_ct) || fallo 'contexto CT base no resuelve'
[[ $referencia == *'"tipo":"candidato"'* && $referencia != *'"tipo":"empleado"'* ]] || fallo 'contexto CT base inesperado'
echo 'Regresión incidente 23/09 (resolutor ContextoActor V2 real):'
exigir_ct_igual 'persona sin empleado (solo candidato)' sin_empleado
publicar pep_sintetica_rrhh_ct_e1_00000000000001 1 emp_sintetico_rrhh_ct_e1_00000000000001 activa
exigir_ct_igual 'empleado activo + candidato' empleado
publicar pep_sintetica_rrhh_ct_e2_00000000000001 1 emp_sintetico_rrhh_ct_e2_00000000000001 activa
exigir_ct_igual 'dos empleados canónicos (ambiguo)' ambiguo
publicar pep_sintetica_rrhh_ct_e2_00000000000001 2 emp_sintetico_rrhh_ct_e2_00000000000001 revocada error_material
exigir_ct_igual 'un empleado revocado' empleado
publicar pep_sintetica_rrhh_ct_e1_00000000000001 2 emp_sintetico_rrhh_ct_e1_00000000000001 no_activa suspension
exigir_ct_igual 'empleado no activo' sin_empleado
[[ $(admin_valor "SELECT count(*) FROM vec_contexto_actor_v1.vinculo_referencia_versiones WHERE persona_ref='$per'") == 1 ]] \
  || fallo 'Personal escribió punteros del núcleo'

# Contraste: el camino del incidente (puntero de empleado del núcleo) sí rompe
# CT. Se documenta para que nadie vuelva a activar Dietas por esa vía.
admin <<SQL
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones VALUES
 ('vin_sintetico_empleado_incidente_000001',1,'$per','empleado','emp_sintetico_rrhh_ct_e1_00000000000001',
  'prc_maestra_sintetica_ct_0000000000001',1,repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '2 hours');
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_actual VALUES ('vin_sintetico_empleado_incidente_000001',1);
COMMIT;
SQL
incidente=$(resolver_ct) || fallo 'contraste: el puntero activo debía resolver'
[[ $incidente != "$referencia" ]] || fallo 'contraste: el puntero del núcleo no alteró el contexto'
echo '  contraste: puntero de empleado del núcleo altera el contexto CT (incidente reproducido)'
admin <<SQL
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones VALUES
 ('vin_sintetico_empleado_incidente_000001',2,'$per','empleado','emp_sintetico_rrhh_ct_e1_00000000000001',
  'prc_maestra_sintetica_ct_0000000000001',1,repeat('a',64),'autoridad_maestra_acreditada','revocado',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '2 hours');
UPDATE vec_contexto_actor_v1.vinculo_referencia_actual SET version=2 WHERE vinculo_ref='vin_sintetico_empleado_incidente_000001';
COMMIT;
SQL
if resolver_ct >/dev/null 2>&1; then fallo 'contraste: puntero revocado debía anular el contexto'; fi
echo '  contraste: puntero de empleado revocado anula el contexto CT (P0002, incidente reproducido)'

# Persistencia tras reinicio y DOWN no destructivo con historia.
docker restart "$contenedor" >/dev/null
esperar
[[ $(admin_valor "SELECT count(*) FROM vec_personal.proyeccion_empleado_persona_historia") == 15 ]] || fallo 'historia tras reinicio'
[[ $(clase_personal) == sin_empleado ]] || fallo 'estado tras reinicio'
if archivo "$repo_dir/deploy/postgresql/personal/migraciones/000016_proyeccion_empleado_persona.down.sql" >/dev/null 2>&1; then
  fallo 'DOWN borró una proyección con historia'
fi
[[ $(admin_valor "SELECT count(*) FROM vec_personal.proyeccion_empleado_persona_historia") == 15 ]] || fallo 'DOWN alteró historia'
printf 'PG18.4: Personal 000016 ROLLBACK limpio, COMMIT, casos, ACL/RLS, regresión CT del incidente, reinicio y DOWN cerrado con historia OK.\n'
