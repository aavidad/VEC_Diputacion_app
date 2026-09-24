#!/usr/bin/env bash
# Personal 000019 en PostgreSQL 18.4 desechable. B1 000008 es real.
# El consumidor AD3 se simula exclusivamente para probar la lógica SQL B2.
# Datos sintéticos; la transacción positiva se revierte y el contenedor se retira.
set -euo pipefail
base_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH='' cd -- "$base_dir/../../../.." && pwd)
motor=${VEC_CONTENEDOR_MOTOR:-docker}
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-personal-b2-019-${USER:-usuario}-$$"
base=vec_registro_empleado_b2_prueba
up7="$repo_dir/deploy/postgresql/contexto_actor_v1/migraciones/000007_alcance_proyecciones_empleado.up.sql"
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
archivo "$up7" 2>/dev/null
archivo "$up8" 2>/dev/null
archivo "$repo_dir/deploy/postgresql/personal/pruebas_sql/organizacion_historica_000010_stub.sql"
admin -o /dev/null -c "GRANT CONNECT ON DATABASE $base TO vec_prueba_personal"
archivo "$repo_dir/deploy/postgresql/personal/migraciones/000010_organizacion_historica.up.sql"
archivo "$repo_dir/deploy/postgresql/personal/migraciones/000017_registro_empleado_bitemporal.up.sql"
admin -o /dev/null <<'SQL'
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
 LANGUAGE sql AS $$ SELECT d->>'decision_ref',d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',encode(sha256($1||$2||gen_random_uuid()::text::bytea),'hex'),'auditoria:synthetic:b2',clock_timestamp(),true FROM (SELECT convert_from($2,'UTF8')::jsonb d) q $$;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
SQL
sed '$s/^COMMIT;/ROLLBACK;/' "$repo_dir/deploy/postgresql/personal/migraciones/000019_escritura_registro_empleado_b2.up.sql" | admin -o /dev/null
[[ $(admin_valor "SELECT to_regclass('vec_personal.registro_empleado_b2_recibo') IS NULL") == t ]] || fallo 'ROLLBACK 000019 dejó tabla'
archivo "$repo_dir/deploy/postgresql/personal/migraciones/000019_escritura_registro_empleado_b2.up.sql"
archivo "$base_dir/escritura_registro_empleado_b2_000019.sql"
"$motor" exec -i "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_prueba_personal -d "$base" <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TimeZone='UTC';
DO $t$
DECLARE material text; actor jsonb; contexto jsonb; cap jsonb; decision jsonb; clave text; mh text; rh text; res jsonb; replay jsonb; hecho jsonb; hecho2 jsonb; relacion text; emp text;
BEGIN
 actor:=jsonb_build_object('actor_ref','actor:synthetic:b2','contexto_actor_ref','ctx:synthetic:b2','contexto_version',1,'cuenta_ref','cuenta:synthetic:b2','cuenta_version',1,'perfil_ref','perfil:synthetic:b2','perfil_version',1,'persona_ref','per_sintetica_alcance_p_00000000000001','persona_version',1);
 contexto:=jsonb_build_object('esquema','vec.contexto-actor.vinculado.v2','principal_ref','actor:synthetic:b2','contexto_actor_ref','ctx:synthetic:b2','contexto_version',1,'cuenta_ref','cuenta:synthetic:b2','cuenta_version',1,'perfil_activo_ref','perfil:synthetic:b2','persona_ref','per_sintetica_alcance_p_00000000000001','persona_version',1,'perfil_version',1);
 material:=jsonb_build_object('esquema','vec.personal.registro-empleado-b2.alta.v1','operacion','alta','persona_ref','per_sintetica_alcance_p_00000000000001','organismo_ref','org:synthetic','unidad_ref','uni:synthetic','regimen_ref','reg:synthetic','modalidad_ref','mod:synthetic','vigente_desde','2026-09-25','vigente_hasta','','version_esperada',0,'procedencia',jsonb_build_object('acto_ref','acto:synthetic:b2','fuente_ref','fuente:synthetic:b2','fuente_version',1,'fuente_huella_sha256',repeat('b',64),'idempotencia_ref','11111111-1111-4111-8111-111111111111'),'actor',actor)::text;
 mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"objetivo_ref":"per_sintetica_alcance_p_00000000000001"},"atributos":{"material_sha256":"'||mh||'","operacion":"alta"}}','UTF8')),'hex');
 cap:=jsonb_build_object('operacion','personal.registro_empleado.alta.registrar','audiencia_consumo','vec_personal.registro_empleado.alta.v1','efecto_ref','per_sintetica_alcance_p_00000000000001','huella_efecto_sha256',rh);
 decision:=jsonb_build_object('principal_id','actor:synthetic:b2','perfil_activo_ref','perfil:synthetic:b2','concedida',true,'modulo_id','personal','obligaciones',jsonb_build_array(),'tipo_recurso','alta_empleado_rrhh','finalidad','registrar_empleado','campos_permitidos','["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","persona_ref","proyeccion_ref","recibo","relacion_ref","version"]'::jsonb,'accion','personal.registro_empleado.alta.registrar','recurso_ref','per_sintetica_alcance_p_00000000000001','contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2');
 res:=vec_personal.registrar_empleado_rrhh_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to(contexto::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 IF res->'recibo'->>'empleado_ref' !~ '^emp_' OR res->'acceso_actual'->>'estado_replay'<>'registrado' THEN RAISE EXCEPTION 'alta no registrada %',res; END IF;
 replay:=vec_personal.registrar_empleado_rrhh_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to(contexto::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 IF replay->'recibo' IS DISTINCT FROM res->'recibo' OR replay->'acceso_actual'->>'estado_replay'<>'replay' OR replay->'acceso_actual'->>'consumo_huella_sha256'=res->'acceso_actual'->>'consumo_huella_sha256' THEN RAISE EXCEPTION 'replay mutable o sin V3 nuevo'; END IF;
 emp:=res->'recibo'->>'empleado_ref';
 material:=jsonb_build_object('esquema','vec.personal.registro-empleado-b2.hecho.v1','operacion','hecho','tipo','relacion','empleado_ref',emp,'relacion_ref','','revision_esperada',1,'relacion_version_esperada',0,'unidad_ref','uni:synthetic','regimen_ref','reg:synthetic','modalidad_ref','mod:synthetic','estado','vigente','plaza_ref','','puesto_ref','','clase_ref','','version_plaza_ref','','version_puesto_ref','','periodo_desde','','periodo_hasta','','dias_reconocidos',0,'vigente_desde','2026-09-26','vigente_hasta','','procedencia',jsonb_build_object('acto_ref','acto:synthetic:b2','fuente_ref','fuente:synthetic:b2','fuente_version',1,'fuente_huella_sha256',repeat('b',64),'idempotencia_ref','22222222-2222-4222-8222-222222222222'),'actor',actor)::text;
 mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"objetivo_ref":"'||emp||'"},"atributos":{"material_sha256":"'||mh||'","operacion":"hecho"}}','UTF8')),'hex');
 cap:=jsonb_build_object('operacion','personal.registro_empleado.hecho.registrar','audiencia_consumo','vec_personal.registro_empleado.hecho.v1','efecto_ref',emp,'huella_efecto_sha256',rh);
 decision:=decision||jsonb_build_object('tipo_recurso','hecho_empleado_rrhh','finalidad','registrar_hecho_empleado','campos_permitidos','["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","hecho_ref","recibo","relacion_ref","tipo","version"]'::jsonb,'accion','personal.registro_empleado.hecho.registrar','recurso_ref',emp,'contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2:relation');
 hecho:=vec_personal.registrar_hecho_empleado_rrhh_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to(contexto::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 IF hecho->'recibo'->>'tipo'<>'relacion' THEN RAISE EXCEPTION 'relación nueva falló'; END IF;
 relacion:=hecho->'recibo'->>'relacion_ref';
 material:=(material::jsonb || jsonb_build_object('relacion_ref',relacion,'revision_esperada',2,'relacion_version_esperada',1,'estado','suspendida','procedencia',jsonb_build_object('acto_ref','acto:synthetic:b2','fuente_ref','fuente:synthetic:b2','fuente_version',2,'fuente_huella_sha256',repeat('b',64),'idempotencia_ref','33333333-3333-4333-8333-333333333333')))::text;
 mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"objetivo_ref":"'||emp||'"},"atributos":{"material_sha256":"'||mh||'","operacion":"hecho"}}','UTF8')),'hex');
 cap:=cap||jsonb_build_object('huella_efecto_sha256',rh);
 decision:=decision||jsonb_build_object('contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2:revision');
 hecho2:=vec_personal.registrar_hecho_empleado_rrhh_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to(contexto::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 IF hecho2->'recibo'->>'version'<>'2' OR hecho2->'recibo'->>'relacion_ref'<>relacion THEN RAISE EXCEPTION 'revisión relación falló'; END IF;
 relacion:=res->'recibo'->>'relacion_ref';
 material:=(material::jsonb || jsonb_build_object('tipo','servicio','relacion_ref',relacion,'revision_esperada',1,'relacion_version_esperada',1,'estado','reconocido','clase_ref','antiguedad','periodo_desde','2026-01-01','periodo_hasta','2026-02-01','dias_reconocidos',31,'procedencia',jsonb_build_object('acto_ref','acto:synthetic:b2','fuente_ref','fuente:synthetic:b2','fuente_version',3,'fuente_huella_sha256',repeat('b',64),'idempotencia_ref','44444444-4444-4444-8444-444444444444')))::text;
 mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"objetivo_ref":"'||emp||'"},"atributos":{"material_sha256":"'||mh||'","operacion":"hecho"}}','UTF8')),'hex');
 cap:=cap||jsonb_build_object('huella_efecto_sha256',rh);
 decision:=decision||jsonb_build_object('contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2:service');
 hecho:=vec_personal.registrar_hecho_empleado_rrhh_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to(contexto::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 IF hecho->'recibo'->>'tipo'<>'servicio' THEN RAISE EXCEPTION 'servicio falló'; END IF;
 material:=(material::jsonb || jsonb_build_object('tipo','situacion','estado','vigente','clase_ref','servicio_activo','procedencia',jsonb_build_object('acto_ref','acto:synthetic:b2','fuente_ref','fuente:synthetic:b2','fuente_version',4,'fuente_huella_sha256',repeat('b',64),'idempotencia_ref','55555555-5555-4555-8555-555555555555')))::text;
 mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"objetivo_ref":"'||emp||'"},"atributos":{"material_sha256":"'||mh||'","operacion":"hecho"}}','UTF8')),'hex');
 cap:=cap||jsonb_build_object('huella_efecto_sha256',rh);
 decision:=decision||jsonb_build_object('contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2:situation');
 hecho:=vec_personal.registrar_hecho_empleado_rrhh_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to(contexto::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 IF hecho->'recibo'->>'tipo'<>'situacion' THEN RAISE EXCEPTION 'situación falló'; END IF;


END $t$;
ROLLBACK;
SQL
"$motor" restart "$contenedor" >/dev/null
esperar
[[ $(admin_valor "SELECT to_regclass('vec_personal.registro_empleado_b2_recibo') IS NOT NULL AND count(*)=0 FROM vec_personal.registro_empleado_b2_recibo") == t ]] || fallo 'reinicio o ROLLBACK positivo divergente'
printf 'PG18 000019: B1 real; ROLLBACK/COMMIT, ACL, alta/replay/relación/servicio/situación sintéticas y reinicio correctos (AD3 simulado).\n'
