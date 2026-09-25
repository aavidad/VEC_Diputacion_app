#!/usr/bin/env bash
# Ensayo en PostgreSQL 18.4 desechable de ContextoActor 000008 (revalidación
# por petición del vínculo corporativo RRHH) sobre la postimagen de 000007.
# Datos exclusivamente sintéticos. El contenedor se elimina al terminar.
# Con VEC_GO_INTEGRACION=1 ejecuta además la prueba de integración del
# adaptador Go contra la misma base antes de borrarla.
set -euo pipefail
base_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH='' cd -- "$base_dir/../../../.." && pwd)
motor=${VEC_CONTENEDOR_MOTOR:-docker}
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-ca-000008-${USER:-usuario}-$$"
base=vec_revalidacion_corporativa_prueba
ca="$repo_dir/deploy/postgresql/contexto_actor_v1"
up8="$ca/migraciones/000008_revalidacion_vinculo_corporativo_rrhh_v1.up.sql"
down8="$ca/migraciones/000008_revalidacion_vinculo_corporativo_rrhh_v1.down.sql"
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
  CREATE ROLE vec_p8_dueno_base NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
  EXECUTE format('ALTER DATABASE %I OWNER TO vec_p8_dueno_base', current_database());
  EXECUTE format('REVOKE ALL ON DATABASE %I FROM PUBLIC', current_database());
END $x$;
REVOKE ALL ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
for f in roles_up.sql migraciones/000001_contexto_actor_v1.up.sql \
  migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql \
  roles_contexto_corporativo_rrhh_selector_v1_up.sql \
  migraciones/000003_organizacion_corporativa_v1.up.sql \
  migraciones/000004a_vinculo_corporativo_rrhh_v1.up.sql \
  roles_historicos_up.sql migraciones/000005_lectura_contexto_historico_v2.up.sql; do
  archivo "$ca/$f"
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
archivo "$ca/migraciones/000006_vinculos_efectivos_temporales.up.sql"
archivo "$repo_dir/deploy/postgresql/personal/roles_up.sql"
archivo "$repo_dir/deploy/postgresql/personal/migraciones/000016_proyeccion_empleado_persona.up.sql"
archivo "$ca/migraciones/000007_alcance_proyecciones_empleado.up.sql" 2>/dev/null

# ---------------------------------------------------------------------------
# Datos sintéticos: persona R con cuenta, perfil, vínculo de contexto F1,
# organización corporativa y vínculo corporativo interna_corporativa/consulta_rrhh.
# ---------------------------------------------------------------------------
CTA=cta_sintetica_corporativa_r_000000001
PER=per_sintetica_corporativa_r_000000001
PRF=prf_sintetico_corporativo_r_000000001
VCA=vca_sintetico_corporativo_r_000000001
ORG=org_sinteticacorporativar0001
VCR=vcr_sintetico_corporativo_r_000000001
PRC=prc_maestra_sintetica_p8_000000000001
admin -o /dev/null <<SQL
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES ('$PRC',1,repeat('a',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('$CTA',1,'$PRC',1,repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES ('$CTA',1);
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('$PER',1,'$PRC',1,repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
INSERT INTO vec_contexto_actor_v1.persona_actual VALUES ('$PER',1);
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('$PRF',1,'$PER','$PRC',1,repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES ('$PRF',1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('$VCA',1,'$CTA','$PRF','$PER','$PRC',1,repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES ('$VCA',1);
INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES
 ('$ORG',1,'$PRC',1,repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
INSERT INTO vec_contexto_actor_v1.organizacion_actual VALUES ('$ORG',1);
COMMIT;
CREATE ROLE vec_ca_runtime_p8 LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contexto_actor_v1_runtime TO vec_ca_runtime_p8 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_ca_ajeno_p8 LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
DO \$c\$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_ca_ajeno_p8', current_database()); END \$c\$;
SQL

# vinculo_corporativo version estado org_version [hasta_sql]: publica una
# versión y mueve el puntero actual, como haría la fuente corporativa.
vinculo_corporativo() {
  local hasta=${4:-"clock_timestamp()+interval '3 hours'"}
  admin -o /dev/null <<SQL
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones VALUES
 ('$VCR',$1,'$CTA',1,'$PER',1,'$PRF',1,'$VCA',1,'$ORG',$3,'$PRC',1,repeat('a',64),
  'autoridad_maestra_acreditada','interna_corporativa','consulta_rrhh',
  '$PRC',1,repeat('a',64),'autoridad_maestra_acreditada','$2',
  clock_timestamp()-interval '1 hour',$hasta);
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES
 ('$CTA','interna_corporativa','consulta_rrhh','$VCR',$1)
 ON CONFLICT (cuenta_ref,superficie,uso) DO UPDATE SET vinculo_corporativo_ref=EXCLUDED.vinculo_corporativo_ref,
   version=EXCLUDED.version;
COMMIT;
SQL
}
organizacion() { # version estado
  admin -o /dev/null <<SQL
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES
 ('$ORG',$1,'$PRC',1,repeat('a',64),'autoridad_maestra_acreditada','$2',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
UPDATE vec_contexto_actor_v1.organizacion_actual SET version=$1 WHERE organizacion_ref='$ORG';
COMMIT;
SQL
}
runtime() { "$motor" exec -i "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_ca_runtime_p8 -d "$base" "$@"; }
# revalidar [persona] [perfil] [vca_version]: t/f o el error SQL.
revalidar() {
  runtime -c "SELECT vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(
    '$CTA','${2:-$PRF}','${1:-$PER}','$VCA',${3:-1})" 2>&1 || true
}

echo 'ContextoActor 000008:'
sed '$s/^COMMIT;/ROLLBACK;/' "$up8" | admin -o /dev/null 2>/dev/null
[[ $(admin_valor "SELECT to_regprocedure('vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric)') IS NULL") == t ]] \
  || fallo 'ROLLBACK de 000008 dejó objetos'
[[ $(runtime -c 'SELECT acreditada FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()') == t ]] \
  || fallo 'runtime de 000007 no acreditado tras el ROLLBACK'
ok 'ROLLBACK limpio; runtime de cinco funciones intacto'
if archivo "$down8" >/dev/null 2>&1; then fallo 'DOWN aceptado sin 000008 instalada'; fi
archivo "$up8"
if archivo "$up8" >/dev/null 2>&1; then fallo '000008 admitió una segunda aplicación'; fi
ok 'instalación única; DOWN rechazado antes de instalar'

[[ $(runtime -c 'SELECT acreditada FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()') == t ]] \
  || fallo 'runtime de seis funciones no acreditado'
salida=$(runtime -c "SELECT count(*) FROM vec_contexto_actor_v1.vinculo_corporativo_actual" 2>&1 || true)
[[ $salida == *'permission denied'* ]] || fallo "runtime leyó la tabla directamente: $salida"
salida=$("$motor" exec "$contenedor" psql -X -qAt -U vec_ca_ajeno_p8 -d "$base" -c \
  "SELECT vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1('$CTA','$PRF','$PER','$VCA',1)" 2>&1 || true)
[[ $salida == *'permission denied'* ]] || fallo "un LOGIN ajeno ejecutó la revalidación: $salida"
ok 'ACL: runtime acreditado con seis funciones, sin lectura de tablas; ajenos sin EXECUTE'

[[ $(revalidar) == f ]] || fallo 'sin vínculo corporativo debía denegar'
ok 'ausencia deniega'
vinculo_corporativo 1 activo 1
[[ $(revalidar) == t ]] || fallo "vínculo activo no autorizó: $(revalidar)"
ok 'vínculo corporativo activo y vigente autoriza'
[[ $(revalidar per_sintetica_corporativa_x_000000001) == f ]] || fallo 'otra persona autorizó'
[[ $(revalidar "$PER" prf_sintetico_corporativo_x_000000001) == f ]] || fallo 'otro perfil autorizó'
[[ $(revalidar "$PER" "$PRF" 2) == f ]] || fallo 'otra versión del vínculo de contexto autorizó'
[[ $(revalidar 'no-valida') == f ]] || fallo 'entrada inválida no denegó'
ok 'persona, perfil, versión de contexto o entrada distintos deniegan'

vinculo_corporativo 2 revocado 1
[[ $(revalidar) == f ]] || fallo 'vínculo revocado siguió autorizando'
ok 'revocación: la siguiente consulta deniega'
vinculo_corporativo 3 activo 1
[[ $(revalidar) == t ]] || fallo 'restitución con versión nueva no autorizó'
ok 'restitución con versión nueva vuelve a autorizar'

organizacion 2 revocado
[[ $(revalidar) == f ]] || fallo 'organización revocada siguió autorizando'
organizacion 3 activo
[[ $(revalidar) == f ]] || fallo 'vínculo ligado a una versión anterior de la organización autorizó'
vinculo_corporativo 4 activo 3
[[ $(revalidar) == t ]] || fallo 'vínculo republicado con la organización actual no autorizó'
ok 'organización revocada deniega; el vínculo queda ligado a la versión actual'

vinculo_corporativo 5 activo 3 "clock_timestamp()+interval '3 seconds'"
[[ $(revalidar) == t ]] || fallo 'vínculo de vigencia breve no autorizó dentro de su vigencia'
sleep 4
[[ $(revalidar) == f ]] || fallo 'vínculo caducado siguió autorizando'
ok 'caducidad: deniega al terminar la vigencia'
vinculo_corporativo 6 activo 3

if [[ ${VEC_GO_INTEGRACION:-0} == 1 ]]; then
  puerto=$("$motor" port "$contenedor" 5432/tcp | head -n1 | sed 's/.*://')
  admin_valor "ALTER ROLE vec_ca_runtime_p8 PASSWORD 'ensayo-desechable'" >/dev/null
  (cd "$repo_dir" && \
    VEC_CONTEXTO_ACTOR_CORPORATIVO_POSTGRES_DSN="postgres://vec_ca_runtime_p8:ensayo-desechable@127.0.0.1:$puerto/$base?sslmode=disable" \
    VEC_CONTEXTO_ACTOR_CORPORATIVO_ADMIN_DSN="postgres://postgres@127.0.0.1:$puerto/$base?sslmode=disable" \
    go test -count=1 -v -run 'TestIntegracionPostgreSQLRevalidacionVinculoCorporativo' ./internal/vec/adapters/contextoactor/postgres/) \
    > "${TMPDIR:-/tmp}/vec_p8_go_$$" 2>&1 || { cat "${TMPDIR:-/tmp}/vec_p8_go_$$" >&2; fallo 'integración Go del adaptador'; }
  grep -q -- '--- PASS: TestIntegracionPostgreSQLRevalidacionVinculoCorporativo' "${TMPDIR:-/tmp}/vec_p8_go_$$" \
    || { rm -f "${TMPDIR:-/tmp}/vec_p8_go_$$"; fallo 'la integración Go no se ejecutó (omitida)'; }
  rm -f "${TMPDIR:-/tmp}/vec_p8_go_$$"
  ok 'adaptador Go contra PostgreSQL 18.4'
fi

archivo "$down8"
[[ $(admin_valor "SELECT to_regprocedure('vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric)') IS NULL") == t ]] \
  || fallo 'DOWN no retiró la función'
[[ $(runtime -c 'SELECT acreditada FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()') == t ]] \
  || fallo 'runtime de cinco funciones no acreditado tras DOWN'
archivo "$up8"
[[ $(revalidar) == t ]] || fallo 'reinstalación tras DOWN'
ok 'DOWN restaura el runtime de 000007 y admite reinstalar'
echo 'ContextoActor 000008: todo verde'
