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
admin -o /dev/null <<'SQL'
DO $f$ BEGIN
 BEGIN
  PERFORM vec_personal.validar_catalogo_acto_b2_interna('org:synthetic','regimen','{"ref":"reg:synthetic","version":1}'::jsonb,'2026-09-25');
  RAISE EXCEPTION '000019 admitió catálogo ausente';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $f$;
SQL
admin -o /dev/null <<'SQL'
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_catalogo_registro_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
 LANGUAGE sql AS $$ SELECT NULL::text,NULL::text,NULL::text,NULL::text,NULL::text,NULL::timestamptz,false WHERE false $$;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_catalogo_registro_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
SQL
archivo "$repo_dir/deploy/postgresql/personal/migraciones/000020_catalogos_registro_empleado.up.sql"
admin -o /dev/null <<'SQL'
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
INSERT INTO vec_personal.entrada_catalogo_registro_empleado_historia(
 organismo_ref,tipo,ref,version,revision,estado,vigente_desde,vigente_hasta,
 huella_sha256,denominacion,acto_ref,actor_ref,idempotencia_ref,decision_ref,
 consumo_huella_sha256,auditoria_ref,registrado_en)
SELECT o.organismo_ref,v.tipo,v.ref,1,1,'publicada','2020-01-01',NULL,
 repeat('a',64),v.denominacion,'acto:synthetic:catalogo','actor:synthetic:b2',gen_random_uuid(),
 'decision:synthetic:catalogo',encode(sha256(gen_random_uuid()::text::bytea),'hex'),
 'auditoria:synthetic:catalogo',clock_timestamp()
FROM (VALUES ('org:synthetic'),('org:synthetic:other')) o(organismo_ref)
CROSS JOIN (VALUES ('regimen','reg:synthetic','Régimen sintético'),
 ('modalidad','mod:synthetic','Modalidad sintética'),
 ('clase_servicio','antiguedad','Antigüedad sintética'),
 ('situacion','servicio_activo','Servicio activo sintético')) v(tipo,ref,denominacion);
COMMIT;
SQL
admin -o /dev/null <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL ROLE vec_personal_propietario;
DO $catalogo$
BEGIN
 BEGIN
  PERFORM vec_personal.validar_catalogo_acto_b2_interna(
   'org:synthetic','regimen','{"ref":"reg:synthetic","version":2}'::jsonb,'2026-09-25');
  RAISE EXCEPTION 'versión de catálogo inexistente admitida';
 EXCEPTION WHEN SQLSTATE '23514' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.validar_catalogo_acto_b2_interna(
   'org:ajeno','regimen','{"ref":"reg:synthetic","version":1}'::jsonb,'2026-09-25');
  RAISE EXCEPTION 'organismo de catálogo ajeno admitido';
 EXCEPTION WHEN SQLSTATE '23514' THEN NULL; END;
END $catalogo$;
INSERT INTO vec_personal.entrada_catalogo_registro_empleado_historia(
 organismo_ref,tipo,ref,version,revision,estado,vigente_desde,vigente_hasta,
 huella_sha256,denominacion,acto_ref,actor_ref,idempotencia_ref,decision_ref,
 consumo_huella_sha256,auditoria_ref,registrado_en)
SELECT organismo_ref,tipo,ref,version,2,'retirada',vigente_desde,vigente_hasta,
 huella_sha256,denominacion,acto_ref,actor_ref,gen_random_uuid(),decision_ref,
 encode(sha256(gen_random_uuid()::text::bytea),'hex'),auditoria_ref,clock_timestamp()
FROM vec_personal.entrada_catalogo_registro_empleado_historia
WHERE organismo_ref='org:synthetic' AND tipo='regimen' AND ref='reg:synthetic' AND revision=1;
DO $catalogo$
BEGIN
 BEGIN
  PERFORM vec_personal.validar_catalogo_acto_b2_interna(
   'org:synthetic','regimen','{"ref":"reg:synthetic","version":1}'::jsonb,'2026-09-25');
  RAISE EXCEPTION 'catálogo retirado admitido';
 EXCEPTION WHEN SQLSTATE '23514' THEN NULL; END;
END $catalogo$;
ROLLBACK;
SQL
casos_tmp=$(mktemp)
cat > "$casos_tmp" <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TimeZone='UTC';
DO $t$
DECLARE material text; actor jsonb; contexto jsonb; cap jsonb; decision jsonb; clave text; mh text; rh text; res jsonb; replay jsonb; hecho jsonb; hecho2 jsonb; relacion text; emp text; material_error text; cap_error jsonb; decision_error jsonb;
BEGIN
 actor:=jsonb_build_object('actor_ref','actor:synthetic:b2','contexto_actor_ref','ctx:synthetic:b2','contexto_version',1,'cuenta_ref','cuenta:synthetic:b2','cuenta_version',1,'perfil_ref','perfil:synthetic:b2','perfil_version',1,'persona_ref','per_sintetica_alcance_p_00000000000001','persona_version',1);
 contexto:=jsonb_build_object('esquema','vec.contexto-actor.vinculado.v2','principal_ref','actor:synthetic:b2','contexto_actor_ref','ctx:synthetic:b2','contexto_version',1,'cuenta_ref','cuenta:synthetic:b2','cuenta_version',1,'perfil_activo_ref','perfil:synthetic:b2','persona_ref','per_sintetica_alcance_p_00000000000001','persona_version',1,'perfil_version',1);
 material:=jsonb_build_object('esquema','vec.personal.registro-empleado-b2.alta.v1','operacion','alta','persona_ref','per_sintetica_alcance_p_00000000000001','organismo_ref','org:synthetic','unidad_ref','uni:synthetic','regimen',jsonb_build_object('ref','reg:synthetic','version',1),'modalidad',jsonb_build_object('ref','mod:synthetic','version',1),'vigente_desde','2026-09-25','vigente_hasta','','version_esperada',0,'procedencia',jsonb_build_object('acto_ref','acto:synthetic:b2','fuente_ref','fuente:synthetic:b2','fuente_version',1,'fuente_huella_sha256',repeat('b',64),'idempotencia_ref','11111111-1111-4111-8111-111111111111'),'actor',actor)::text;
 mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"objetivo_ref":"per_sintetica_alcance_p_00000000000001","organismo_ref":"org:synthetic"},"atributos":{"material_sha256":"'||mh||'","operacion":"alta"}}','UTF8')),'hex');
 cap:=jsonb_build_object('operacion','personal.registro_empleado.alta.registrar','audiencia_consumo','vec_personal.registro_empleado.alta.v1','efecto_ref','per_sintetica_alcance_p_00000000000001','huella_efecto_sha256',rh,'emitida_en','2020-01-01T00:00:00Z','expira_en','2100-01-01T00:00:00Z','decision_valida_hasta','2100-01-01T00:00:00Z');
 decision:=jsonb_build_object('principal_id','actor:synthetic:b2','perfil_activo_ref','perfil:synthetic:b2','concedida',true,'modulo_id','personal','obligaciones',jsonb_build_array(),'tipo_recurso','alta_empleado_rrhh','finalidad','registrar_empleado','campos_permitidos','["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","persona_ref","proyeccion_ref","recibo","relacion_ref","version"]'::jsonb,'accion','personal.registro_empleado.alta.registrar','recurso_ref','per_sintetica_alcance_p_00000000000001','contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2','valida_hasta','2100-01-01T00:00:00Z');
 res:=vec_personal.registrar_empleado_rrhh_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to(contexto::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 IF res->'recibo'->>'empleado_ref' !~ '^emp_' OR res->'acceso_actual'->>'estado_replay'<>'registrado' THEN RAISE EXCEPTION 'alta no registrada %',res; END IF;
 replay:=vec_personal.registrar_empleado_rrhh_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to(contexto::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 IF replay->'recibo' IS DISTINCT FROM res->'recibo' OR replay->'acceso_actual'->>'estado_replay'<>'replay' OR replay->'acceso_actual'->>'consumo_huella_sha256'=res->'acceso_actual'->>'consumo_huella_sha256' THEN RAISE EXCEPTION 'replay mutable o sin V3 nuevo'; END IF;
 -- Otra clave para la misma persona (aprovisionamiento repetido con plan
 -- distinto): conflicto sin segundo empleado ni segunda proyección.
 material_error:=replace(material,'11111111-1111-4111-8111-111111111111','77777777-7777-4777-8777-777777777777');
 mh:=encode(sha256(convert_to(material_error,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"objetivo_ref":"per_sintetica_alcance_p_00000000000001","organismo_ref":"org:synthetic"},"atributos":{"material_sha256":"'||mh||'","operacion":"alta"}}','UTF8')),'hex');
 BEGIN
  PERFORM vec_personal.registrar_empleado_rrhh_v1(material_error,convert_to((cap||jsonb_build_object('huella_efecto_sha256',rh))::text,'UTF8'),convert_to((decision||jsonb_build_object('contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2:segunda'))::text,'UTF8'),convert_to('{}','UTF8'),convert_to(contexto::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  RAISE EXCEPTION 'segunda alta de la misma persona admitida';
 EXCEPTION WHEN SQLSTATE '23505' THEN NULL; END;
 emp:=res->'recibo'->>'empleado_ref';
 material:=jsonb_build_object('esquema','vec.personal.registro-empleado-b2.hecho.v1','operacion','hecho','tipo','relacion','empleado_ref',emp,'organismo_ref','org:synthetic:other','relacion_ref','','revision_esperada',1,'relacion_version_esperada',0,'unidad_ref','uni:synthetic','regimen',jsonb_build_object('ref','reg:synthetic','version',1),'modalidad',jsonb_build_object('ref','mod:synthetic','version',1),'situacion',jsonb_build_object('ref','','version',0),'clase_servicio',jsonb_build_object('ref','','version',0),'clase_ocupacion','','estado','vigente','plaza_ref','','puesto_ref','','version_plaza_ref','','version_puesto_ref','','periodo_desde','','periodo_hasta','','dias_reconocidos',0,'vigente_desde','2026-09-26','vigente_hasta','','procedencia',jsonb_build_object('acto_ref','acto:synthetic:b2','fuente_ref','fuente:synthetic:b2','fuente_version',1,'fuente_huella_sha256',repeat('b',64),'idempotencia_ref','22222222-2222-4222-8222-222222222222'),'actor',actor)::text;
 mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"objetivo_ref":"'||emp||'","organismo_ref":"'||(material::jsonb->>'organismo_ref')||'"},"atributos":{"material_sha256":"'||mh||'","operacion":"hecho"}}','UTF8')),'hex');
 cap:=jsonb_build_object('operacion','personal.registro_empleado.hecho.registrar','audiencia_consumo','vec_personal.registro_empleado.hecho.v1','efecto_ref',emp,'huella_efecto_sha256',rh,'emitida_en','2020-01-01T00:00:00Z','expira_en','2100-01-01T00:00:00Z','decision_valida_hasta','2100-01-01T00:00:00Z');
 decision:=decision||jsonb_build_object('tipo_recurso','hecho_empleado_rrhh','finalidad','registrar_hecho_empleado','campos_permitidos','["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","hecho_ref","recibo","relacion_ref","tipo","version"]'::jsonb,'accion','personal.registro_empleado.hecho.registrar','recurso_ref',emp,'contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2:relation');
 hecho:=vec_personal.registrar_hecho_empleado_rrhh_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to(contexto::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 IF hecho->'recibo'->>'tipo'<>'relacion' THEN RAISE EXCEPTION 'relación nueva falló'; END IF;
 relacion:=hecho->'recibo'->>'relacion_ref';
 material:=(material::jsonb || jsonb_build_object('relacion_ref',relacion,'revision_esperada',2,'relacion_version_esperada',1,'estado','suspendida','procedencia',jsonb_build_object('acto_ref','acto:synthetic:b2','fuente_ref','fuente:synthetic:b2','fuente_version',2,'fuente_huella_sha256',repeat('b',64),'idempotencia_ref','33333333-3333-4333-8333-333333333333')))::text;
 mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"objetivo_ref":"'||emp||'","organismo_ref":"'||(material::jsonb->>'organismo_ref')||'"},"atributos":{"material_sha256":"'||mh||'","operacion":"hecho"}}','UTF8')),'hex');
 cap:=cap||jsonb_build_object('huella_efecto_sha256',rh);
 decision:=decision||jsonb_build_object('contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2:revision');
 hecho2:=vec_personal.registrar_hecho_empleado_rrhh_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to(contexto::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 IF hecho2->'recibo'->>'version'<>'2' OR hecho2->'recibo'->>'relacion_ref'<>relacion THEN RAISE EXCEPTION 'revisión relación falló'; END IF;
 relacion:=res->'recibo'->>'relacion_ref';
 material:=(material::jsonb || jsonb_build_object('organismo_ref','org:synthetic','tipo','servicio','regimen',jsonb_build_object('ref','','version',0),'modalidad',jsonb_build_object('ref','','version',0),'clase_servicio',jsonb_build_object('ref','antiguedad','version',1),'relacion_ref',relacion,'revision_esperada',1,'relacion_version_esperada',1,'estado','reconocido','periodo_desde','2026-01-01','periodo_hasta','2026-02-01','dias_reconocidos',31,'procedencia',jsonb_build_object('acto_ref','acto:synthetic:b2','fuente_ref','fuente:synthetic:b2','fuente_version',3,'fuente_huella_sha256',repeat('b',64),'idempotencia_ref','44444444-4444-4444-8444-444444444444')))::text;
 mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"objetivo_ref":"'||emp||'","organismo_ref":"'||(material::jsonb->>'organismo_ref')||'"},"atributos":{"material_sha256":"'||mh||'","operacion":"hecho"}}','UTF8')),'hex');
 cap:=cap||jsonb_build_object('huella_efecto_sha256',rh);
 decision:=decision||jsonb_build_object('contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2:service');
 hecho:=vec_personal.registrar_hecho_empleado_rrhh_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to(contexto::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 IF hecho->'recibo'->>'tipo'<>'servicio' THEN RAISE EXCEPTION 'servicio falló'; END IF;
 material_error:=(material::jsonb || jsonb_build_object('organismo_ref','org:synthetic:other','procedencia',jsonb_build_object('acto_ref','acto:synthetic:b2','fuente_ref','fuente:synthetic:b2','fuente_version',9,'fuente_huella_sha256',repeat('b',64),'idempotencia_ref','99999999-9999-4999-8999-999999999999')))::text;
 mh:=encode(sha256(convert_to(material_error,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"objetivo_ref":"'||emp||'","organismo_ref":"org:synthetic:other"},"atributos":{"material_sha256":"'||mh||'","operacion":"hecho"}}','UTF8')),'hex');
 cap_error:=cap||jsonb_build_object('huella_efecto_sha256',rh);
 decision_error:=decision||jsonb_build_object('contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2:cross-org');
 BEGIN
  PERFORM vec_personal.registrar_hecho_empleado_rrhh_v1(material_error,convert_to(cap_error::text,'UTF8'),convert_to(decision_error::text,'UTF8'),convert_to('{}','UTF8'),convert_to(contexto::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  RAISE EXCEPTION 'servicio admitió organismo ajeno';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;

 material:=(material::jsonb || jsonb_build_object('tipo','situacion','estado','vigente','clase_servicio',jsonb_build_object('ref','','version',0),'situacion',jsonb_build_object('ref','servicio_activo','version',1),'procedencia',jsonb_build_object('acto_ref','acto:synthetic:b2','fuente_ref','fuente:synthetic:b2','fuente_version',4,'fuente_huella_sha256',repeat('b',64),'idempotencia_ref','55555555-5555-4555-8555-555555555555')))::text;
 mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"objetivo_ref":"'||emp||'","organismo_ref":"'||(material::jsonb->>'organismo_ref')||'"},"atributos":{"material_sha256":"'||mh||'","operacion":"hecho"}}','UTF8')),'hex');
 cap:=cap||jsonb_build_object('huella_efecto_sha256',rh);
 decision:=decision||jsonb_build_object('contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2:situation');
 hecho:=vec_personal.registrar_hecho_empleado_rrhh_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to(contexto::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 IF hecho->'recibo'->>'tipo'<>'situacion' THEN RAISE EXCEPTION 'situación falló'; END IF;


END $t$;
ROLLBACK;
SQL
"$motor" exec -i "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_prueba_personal -d "$base" < "$casos_tmp"
# Carrera: AD3 sintético entrega una decisión válida; otra transacción retiene
# el lock de idempotencia hasta que expire. La función debe denegar sin historia.
"$motor" exec "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d "$base" \
 -c "BEGIN; SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:registro-b2:idempotencia:66666666-6666-4666-8666-666666666666',0)); SELECT pg_sleep(1.8); COMMIT;" >/dev/null &
locker_pid=$!
sleep 0.2
"$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_prueba_personal -d "$base" <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TimeZone='UTC';
DO $race$
DECLARE material text; actor jsonb; contexto jsonb; cap jsonb; decision jsonb;
 mh text; rh text; vence text; empezado timestamptz;
BEGIN
 actor:=jsonb_build_object('actor_ref','actor:synthetic:b2','contexto_actor_ref','ctx:synthetic:b2','contexto_version',1,'cuenta_ref','cuenta:synthetic:b2','cuenta_version',1,'perfil_ref','perfil:synthetic:b2','perfil_version',1,'persona_ref','per_sintetica_alcance_p_00000000000001','persona_version',1);
 contexto:=jsonb_build_object('esquema','vec.contexto-actor.vinculado.v2','principal_ref','actor:synthetic:b2','contexto_actor_ref','ctx:synthetic:b2','contexto_version',1,'cuenta_ref','cuenta:synthetic:b2','cuenta_version',1,'perfil_activo_ref','perfil:synthetic:b2','persona_ref','per_sintetica_alcance_p_00000000000001','persona_version',1,'perfil_version',1);
 material:=jsonb_build_object('esquema','vec.personal.registro-empleado-b2.alta.v1','operacion','alta','persona_ref','per_sintetica_alcance_p_00000000000001','organismo_ref','org:synthetic','unidad_ref','uni:synthetic','regimen',jsonb_build_object('ref','reg:synthetic','version',1),'modalidad',jsonb_build_object('ref','mod:synthetic','version',1),'vigente_desde','2026-09-25','vigente_hasta','','version_esperada',0,'procedencia',jsonb_build_object('acto_ref','acto:synthetic:b2','fuente_ref','fuente:synthetic:b2','fuente_version',1,'fuente_huella_sha256',repeat('b',64),'idempotencia_ref','66666666-6666-4666-8666-666666666666'),'actor',actor)::text;
 mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"objetivo_ref":"per_sintetica_alcance_p_00000000000001","organismo_ref":"org:synthetic"},"atributos":{"material_sha256":"'||mh||'","operacion":"alta"}}','UTF8')),'hex');
 vence:=to_char((clock_timestamp()+interval '0.75 seconds') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 cap:=jsonb_build_object('operacion','personal.registro_empleado.alta.registrar','audiencia_consumo','vec_personal.registro_empleado.alta.v1','efecto_ref','per_sintetica_alcance_p_00000000000001','huella_efecto_sha256',rh,'emitida_en','2020-01-01T00:00:00Z','expira_en',vence,'decision_valida_hasta',vence);
 decision:=jsonb_build_object('principal_id','actor:synthetic:b2','perfil_activo_ref','perfil:synthetic:b2','concedida',true,'modulo_id','personal','obligaciones',jsonb_build_array(),'tipo_recurso','alta_empleado_rrhh','finalidad','registrar_empleado','campos_permitidos','["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","persona_ref","proyeccion_ref","recibo","relacion_ref","version"]'::jsonb,'accion','personal.registro_empleado.alta.registrar','recurso_ref','per_sintetica_alcance_p_00000000000001','contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2:expired','valida_hasta',vence);
 empezado:=clock_timestamp();
 BEGIN
  PERFORM vec_personal.registrar_empleado_rrhh_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to(contexto::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  RAISE EXCEPTION 'caducidad con lock admitió alta';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 IF clock_timestamp()-empezado<interval '0.75 seconds' THEN
  RAISE EXCEPTION 'caducidad no atravesó el bloqueo idempotente'; END IF;
END $race$;
ROLLBACK;
SQL
wait "$locker_pid"
[[ $(admin_valor "SELECT (SELECT count(*)=0 FROM vec_personal.registro_empleado_b2_recibo) AND (SELECT count(*)=0 FROM vec_personal.relacion_servicio_historia) AND (SELECT count(*)=0 FROM vec_personal.proyeccion_empleado_persona_historia)") == t ]] || fallo 'caducidad dejó historia'
sed '$s/^ROLLBACK;/COMMIT;/' "$casos_tmp" | "$motor" exec -i "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_prueba_personal -d "$base"
rm -f "$casos_tmp"
[[ $(admin_valor "SELECT (SELECT count(*)=5 FROM vec_personal.registro_empleado_b2_recibo) AND (SELECT count(*)=3 FROM vec_personal.relacion_servicio_historia) AND (SELECT count(*)=1 FROM vec_personal.servicio_reconocido_historia) AND (SELECT count(*)=1 FROM vec_personal.situacion_empleado_historia) AND (SELECT count(*)=1 FROM vec_personal.proyeccion_empleado_persona_historia) AND (SELECT bool_and(catalogo_snapshot ? 'modalidad') FROM vec_personal.relacion_servicio_historia)") == t ]] || fallo 'historia o snapshots no confirmados'
"$motor" restart "$contenedor" >/dev/null
esperar
[[ $(admin_valor "SELECT to_regclass('vec_personal.registro_empleado_b2_recibo') IS NOT NULL AND count(*)=5 FROM vec_personal.registro_empleado_b2_recibo") == t ]] || fallo 'recibos no recuperados tras reinicio'
# La proyección que publica el alta es la que ContextoActor 000007 consume:
# la persona resuelve exactamente al empleado del recibo de alta.
[[ $(admin_valor "SELECT p.resultado='empleado' AND p.empleado_ref=r.empleado_ref AND p.proyeccion_ref=r.proyeccion_ref FROM vec_personal.registro_empleado_b2_recibo r CROSS JOIN LATERAL vec_contexto_actor_v1.proyeccion_empleado_personal_v2(r.efecto_ref, clock_timestamp()) p WHERE r.operacion='alta'") == t ]] || fallo 'ContextoActor no resuelve el empleado publicado por el alta'

# ---------------------------------------------------------------------------
# Personal 000021: lista RRHH de empleados del organismo sobre la historia ya
# confirmada. AD3-56 se simula igual que AD3-54; se prueba en su propio ensayo.
# ---------------------------------------------------------------------------
archivo "$repo_dir/deploy/postgresql/personal/migraciones/000018_lectura_registro_empleado_b2.up.sql"
admin -o /dev/null <<'SQL'
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_empleados_registro_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
 LANGUAGE sql AS $$ SELECT d->>'decision_ref',d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',encode(sha256($1||$2||gen_random_uuid()::text::bytea),'hex'),'auditoria:synthetic:b2',clock_timestamp(),true FROM (SELECT convert_from($2,'UTF8')::jsonb d) q $$;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_empleados_registro_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
SQL
up21="$repo_dir/deploy/postgresql/personal/migraciones/000021_lista_empleados_organismo_b2.up.sql"
sed '$s/^COMMIT;/ROLLBACK;/' "$up21" | admin -o /dev/null
[[ $(admin_valor "SELECT to_regclass('vec_personal.recibo_lista_empleados_b2') IS NULL AND to_regprocedure('vec_personal.consultar_empleados_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL") == t ]] || fallo 'ROLLBACK 000021 dejó objetos'
archivo "$up21"
if archivo "$up21" 2>/dev/null; then fallo 'segunda aplicación de 000021 aceptada'; fi
ok '000021 ROLLBACK sin rastro, COMMIT y segunda aplicación rechazada'
[[ $(admin_valor "SELECT has_function_privilege('vec_personal_ejecutor','vec_personal.consultar_empleados_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') AND NOT has_function_privilege('public','vec_personal.consultar_empleados_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') AND NOT has_table_privilege('vec_personal_ejecutor','vec_personal.recibo_lista_empleados_b2','SELECT') AND (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid='vec_personal.recibo_lista_empleados_b2'::regclass)") == t ]] || fallo 'ACL o RLS de 000021 divergente'
ok '000021 ACL: solo el ejecutor consulta; recibo sin lectura directa y con RLS forzada'
lista_tmp=$(mktemp)
cat > "$lista_tmp" <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TimeZone='UTC';
DO $l$
DECLARE conocido text; material text; mh text; rh text; cap jsonb; decision jsonb; res jsonb; fila jsonb;
 llamar_material text; err text;
BEGIN
 conocido:=to_char(transaction_timestamp() AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 material:='{"esquema":"vec.personal.registro-empleado-b2.consulta.v1","operacion":"empleados","empleado_ref":"","organismo_ref":"org:synthetic","vigente_en":"2026-09-26","conocido_en":"'||conocido||'","limite":1,"cursor":"","actor_ref":"per_sintetica_alcance_p_00000000000001","contexto_actor_ref":"ctx:synthetic:b2","contexto_version":1,"cuenta_ref":"cta_sintetica_alcance_p_0000000000001","cuenta_version":1,"perfil_ref":"prf_sintetico_alcance_p_0000000000001","perfil_version":1,"persona_ref":"per_sintetica_alcance_p_00000000000001","persona_version":1}';
 mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"organismo_ref":"org:synthetic"},"atributos":{"conocido_en":"'||conocido||'","material_sha256":"'||mh||'","operacion":"empleados","vigente_en":"2026-09-26"}}','UTF8')),'hex');
 cap:=jsonb_build_object('operacion','personal.registro_empleado.empleados.consultar','audiencia_consumo','vec_personal.registro_empleado.empleados.v1','efecto_ref','org:synthetic','huella_efecto_sha256',rh);
 decision:=jsonb_build_object('principal_id','per_sintetica_alcance_p_00000000000001','perfil_activo_ref','prf_sintetico_alcance_p_0000000000001','concedida',true,'modulo_id','personal','obligaciones',jsonb_build_array(),'tipo_recurso','empleados_rrhh','finalidad','consultar_empleados','campos_permitidos','["corte","cursor","cursor_siguiente","empleados","evidencia","limite","organismo_ref"]'::jsonb,'accion','personal.registro_empleado.empleados.consultar','recurso_ref','org:synthetic','contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2:lista','valida_hasta','2100-01-01T00:00:00Z');
 res:=vec_personal.consultar_empleados_rrhh_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 fila:=res->'pagina'->'empleados'->0;
 IF jsonb_array_length(res->'pagina'->'empleados')<>1 OR fila->>'empleado_ref' !~ '^emp_'
    OR jsonb_array_length(fila->'relaciones')<>1
    OR fila->'relaciones'->0->>'regimen_denominacion'<>'Régimen sintético'
    OR fila->'relaciones'->0->>'modalidad_denominacion'<>'Modalidad sintética'
    OR fila->'relaciones'->0->>'estado'<>'vigente'
    OR res->'pagina'->>'cursor_siguiente'<>'' OR res->'evidencia'->>'decision_ref'<>'decision:synthetic:b2:lista'
    OR res::text ~ 'per_sintetica' THEN
  RAISE EXCEPTION 'lista de empleados inesperada %',res; END IF;
 -- La referencia elegida en la lista abre la ficha (Personal 000018).
 llamar_material:='{"esquema":"vec.personal.registro-empleado-b2.consulta.v1","operacion":"ficha","empleado_ref":"'||(fila->>'empleado_ref')||'","organismo_ref":"org:synthetic","vigente_en":"2026-09-26","conocido_en":"'||conocido||'","limite":0,"cursor":"","actor_ref":"per_sintetica_alcance_p_00000000000001","contexto_actor_ref":"ctx:synthetic:b2","contexto_version":1,"cuenta_ref":"cta_sintetica_alcance_p_0000000000001","cuenta_version":1,"perfil_ref":"prf_sintetico_alcance_p_0000000000001","perfil_version":1,"persona_ref":"per_sintetica_alcance_p_00000000000001","persona_version":1}';
 mh:=encode(sha256(convert_to(llamar_material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"empleado_ref":"'||(fila->>'empleado_ref')||'","organismo_ref":"org:synthetic"},"atributos":{"conocido_en":"'||conocido||'","material_sha256":"'||mh||'","operacion":"ficha","vigente_en":"2026-09-26"}}','UTF8')),'hex');
 res:=vec_personal.consultar_registro_empleado_rrhh_v1(llamar_material,
  convert_to(jsonb_build_object('operacion','personal.registro_empleado.ficha.consultar','audiencia_consumo','vec_personal.registro_empleado.ficha.v1','efecto_ref',fila->>'empleado_ref','huella_efecto_sha256',rh)::text,'UTF8'),
  convert_to((decision||jsonb_build_object('tipo_recurso','registro_empleado_rrhh','finalidad','consultar_ficha_empleado','accion','personal.registro_empleado.ficha.consultar','recurso_ref',fila->>'empleado_ref','contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2:ficha','campos_permitidos','["corte","eficacia_administrativa","empleado_ref","evidencia","firma_oficial","ocupaciones","organismo_ref","persona_ref","relaciones","servicios","situaciones","version"]'::jsonb))::text,'UTF8'),
  convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 IF res->'ficha'->>'empleado_ref'<>fila->>'empleado_ref' OR jsonb_array_length(res->'ficha'->'relaciones')<>1
    OR jsonb_array_length(res->'ficha'->'servicios')<>1 OR jsonb_array_length(res->'ficha'->'situaciones')<>1 THEN
  RAISE EXCEPTION 'ficha desde la lista inesperada %',res; END IF;
 -- Fecha anterior a toda relación: el empleado se lista, sin relaciones.
 llamar_material:=replace(material,'"vigente_en":"2026-09-26"','"vigente_en":"2026-01-01"');
 mh:=encode(sha256(convert_to(llamar_material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"organismo_ref":"org:synthetic"},"atributos":{"conocido_en":"'||conocido||'","material_sha256":"'||mh||'","operacion":"empleados","vigente_en":"2026-01-01"}}','UTF8')),'hex');
 res:=vec_personal.consultar_empleados_rrhh_v1(llamar_material,convert_to((cap||jsonb_build_object('huella_efecto_sha256',rh))::text,'UTF8'),convert_to((decision||jsonb_build_object('contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:b2:lista:antes'))::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 IF jsonb_array_length(res->'pagina'->'empleados')<>1 OR jsonb_array_length(res->'pagina'->'empleados'->0->'relaciones')<>0 THEN
  RAISE EXCEPTION 'fecha anterior mostró relaciones %',res; END IF;
 -- Negativos: campos ampliados, operación ajena y material no canónico.
 BEGIN
  PERFORM vec_personal.consultar_empleados_rrhh_v1(material,convert_to(cap::text,'UTF8'),convert_to((decision||jsonb_build_object('campos_permitidos','["empleados","persona_ref"]'::jsonb))::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  RAISE EXCEPTION 'campos ampliados admitidos';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.consultar_empleados_rrhh_v1(material,convert_to((cap||jsonb_build_object('operacion','personal.registro_empleado.ficha.consultar'))::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  RAISE EXCEPTION 'operación ajena admitida';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.consultar_empleados_rrhh_v1(replace(material,'"limite":1,','"limite": 1,'),convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  RAISE EXCEPTION 'material no canónico admitido';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.consultar_empleados_rrhh_v1(replace(material,'"cursor":""','"cursor":"p_5_'||repeat('0',64)||'"'),convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  RAISE EXCEPTION 'cursor ajeno admitido';
 EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '42501' THEN NULL; END;
END $l$;
COMMIT;
SQL
"$motor" exec -i "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_prueba_personal -d "$base" < "$lista_tmp" >/dev/null
rm -f "$lista_tmp"
ok '000021 lista paginada sin datos civiles, ficha 000018 abierta desde la lista, relaciones vigentes por fecha y negativos 42501/22023'
if "$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d "$base" -c "SET ROLE vec_personal_ejecutor; SELECT count(*) FROM vec_personal.recibo_lista_empleados_b2" >/dev/null 2>&1; then fallo 'recibo de lista legible por el ejecutor'; fi
"$motor" restart "$contenedor" >/dev/null
esperar
[[ $(admin_valor "SELECT count(*) FROM vec_personal.recibo_lista_empleados_b2") == 2 && $(admin_valor "SELECT count(*) FROM vec_personal.recibo_lectura_registro_empleado_b2 WHERE operacion='ficha'") == 1 ]] || fallo 'recibos de lista no conservados tras reinicio'
ok '000021 dos recibos de lista conservados tras reinicio'

# ---------------------------------------------------------------------------
# Personal 000022: ficha propia de la persona empleada («mis datos») sobre la
# misma historia. AD3-74 se simula como AD3-54/56 (su ensayo estructural es
# aparte). El rol de registro de frontera lo crea el DBA con Personal 000013;
# aquí se crea igual, sin instalar 000012/000013.
# ---------------------------------------------------------------------------
admin -o /dev/null <<'SQL'
CREATE ROLE vec_personal_registrador_frontera NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_prueba_frontera_personal LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_personal_registrador_frontera TO vec_prueba_frontera_personal WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
admin_valor "GRANT CONNECT ON DATABASE $base TO vec_prueba_frontera_personal" >/dev/null
up22="$repo_dir/deploy/postgresql/personal/migraciones/000022_ficha_propia_empleado.up.sql"
if archivo "$up22" 2>/dev/null; then fallo '000022 aceptada sin el consumidor AD3-74'; fi
admin -o /dev/null <<'SQL'
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_ficha_propia_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
 LANGUAGE sql AS $$ SELECT d->>'decision_ref',d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',encode(sha256($1||$2||gen_random_uuid()::text::bytea),'hex'),'auditoria:synthetic:ficha-propia',clock_timestamp(),true FROM (SELECT convert_from($2,'UTF8')::jsonb d) q $$;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_ficha_propia_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
SQL
sed '$s/^COMMIT;/ROLLBACK;/' "$up22" | admin -o /dev/null
[[ $(admin_valor "SELECT to_regclass('vec_personal.recibo_ficha_propia_empleado') IS NULL AND to_regclass('vec_personal.denegacion_frontera_ficha_propia') IS NULL AND to_regprocedure('vec_personal.consultar_ficha_propia_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL") == t ]] || fallo 'ROLLBACK 000022 dejó objetos'
archivo "$up22"
if archivo "$up22" 2>/dev/null; then fallo 'segunda aplicación de 000022 aceptada'; fi
ok '000022 rechazada sin AD3-74; ROLLBACK sin rastro, COMMIT y segunda aplicación rechazada'
f22='vec_personal.consultar_ficha_propia_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
d22='vec_personal.registrar_denegacion_ficha_propia_v1(text,text,smallint,text)'
[[ $(admin_valor "SELECT has_function_privilege('vec_personal_ejecutor','$f22','EXECUTE') AND NOT has_function_privilege('public','$f22','EXECUTE') AND NOT has_function_privilege('vec_personal_registrador_frontera','$f22','EXECUTE') AND has_function_privilege('vec_personal_registrador_frontera','$d22','EXECUTE') AND NOT has_function_privilege('vec_personal_ejecutor','$d22','EXECUTE') AND NOT has_function_privilege('public','$d22','EXECUTE') AND NOT has_table_privilege('vec_personal_ejecutor','vec_personal.recibo_ficha_propia_empleado','SELECT') AND NOT has_table_privilege('vec_personal_registrador_frontera','vec_personal.denegacion_frontera_ficha_propia','SELECT') AND (SELECT bool_and(relrowsecurity AND relforcerowsecurity) FROM pg_class WHERE oid IN ('vec_personal.recibo_ficha_propia_empleado'::regclass,'vec_personal.denegacion_frontera_ficha_propia'::regclass))") == t ]] || fallo 'ACL o RLS de 000022 divergente'
ok '000022 ACL: el ejecutor consulta, el registrador solo registra denegaciones; tablas sin lectura directa y con RLS forzada'
propia_tmp=$(mktemp)
cat > "$propia_tmp" <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TimeZone='UTC';
DO $p$
DECLARE conocido text; emp text; material text; mh text; rh text; cap jsonb; decision jsonb; res jsonb; r0 jsonb; r1 jsonb; s0 jsonb;
 otro text; ajeno text;
BEGIN
 conocido:=to_char(transaction_timestamp() AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 emp:=current_setting('vec_prueba.empleado');
 material:='{"esquema":"vec.personal.ficha-propia.consulta.v1","empleado_ref":"'||emp||'","vigente_en":"2026-09-26","conocido_en":"'||conocido||'","actor_ref":"per_sintetica_alcance_p_00000000000001","contexto_actor_ref":"ctx:synthetic:b2","contexto_version":1,"cuenta_ref":"cta_sintetica_alcance_p_0000000000001","cuenta_version":1,"perfil_ref":"prf_sintetico_alcance_p_0000000000001","perfil_version":1,"persona_ref":"per_sintetica_alcance_p_00000000000001","persona_version":1}';
 mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"empleado_ref":"'||emp||'"},"atributos":{"conocido_en":"'||conocido||'","material_sha256":"'||mh||'","operacion":"ficha_propia","vigente_en":"2026-09-26"}}','UTF8')),'hex');
 cap:=jsonb_build_object('operacion','personal.registro_empleado.ficha_propia.consultar','audiencia_consumo','vec_personal.registro_empleado.ficha_propia.v1','efecto_ref',emp,'huella_efecto_sha256',rh);
 decision:=jsonb_build_object('principal_id','per_sintetica_alcance_p_00000000000001','perfil_activo_ref','prf_sintetico_alcance_p_0000000000001','concedida',true,'modulo_id','personal','obligaciones',jsonb_build_array(),'tipo_recurso','ficha_propia_empleado','finalidad','consultar_ficha_propia','campos_permitidos','["corte","evidencia","relaciones","servicios"]'::jsonb,'accion','personal.registro_empleado.ficha_propia.consultar','recurso_ref',emp,'contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:ficha-propia','valida_hasta','2100-01-01T00:00:00Z');
 res:=vec_personal.consultar_ficha_propia_empleado_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 r0:=res->'ficha'->'relaciones'->0; r1:=res->'ficha'->'relaciones'->1; s0:=res->'ficha'->'servicios'->0;
 -- Dos relaciones (la más reciente primero, en su última revisión), un
 -- servicio y la situación vigente con sus denominaciones publicadas.
 IF jsonb_array_length(res->'ficha'->'relaciones')<>2 OR jsonb_array_length(res->'ficha'->'servicios')<>1
    OR r0->>'inicio'<>'2026-09-26' OR r0->>'estado'<>'suspendida' OR r0->>'fin'<>''
    OR r0->>'regimen'<>'Régimen sintético' OR r0->>'modalidad'<>'Modalidad sintética' OR r0->>'situacion'<>''
    OR r1->>'inicio'<>'2026-09-25' OR r1->>'estado'<>'vigente' OR r1->>'situacion'<>'Servicio activo sintético'
    OR r1->>'unidad'<>'' OR r1->>'puesto'<>''
    OR s0->>'inicio'<>'2026-01-01' OR s0->>'fin'<>'2026-02-01' OR (s0->>'dias')::int<>31
    OR s0->>'estado'<>'reconocido' OR s0->>'clase'<>'Antigüedad sintética'
    OR res->'ficha'->'corte'->>'vigente_en'<>'2026-09-26' OR res->'ficha'->'corte'->>'conocido_en'<>conocido
    OR res->'evidencia'->>'decision_ref'<>'decision:synthetic:ficha-propia' OR res->'evidencia'->>'recibo_ref' !~ '^fichapropia:'
    OR (res->'ficha')::text ~ '(per|emp|rel|srv|sit|ocu)_' THEN
  RAISE EXCEPTION 'ficha propia inesperada %',res; END IF;
 -- Antes de toda relación: la situación (vigente desde el 26) no se muestra.
 otro:=replace(material,'"vigente_en":"2026-09-26"','"vigente_en":"2026-01-01"');
 mh:=encode(sha256(convert_to(otro,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"empleado_ref":"'||emp||'"},"atributos":{"conocido_en":"'||conocido||'","material_sha256":"'||mh||'","operacion":"ficha_propia","vigente_en":"2026-01-01"}}','UTF8')),'hex');
 res:=vec_personal.consultar_ficha_propia_empleado_v1(otro,convert_to((cap||jsonb_build_object('huella_efecto_sha256',rh))::text,'UTF8'),convert_to((decision||jsonb_build_object('contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:ficha-propia:antes'))::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
 IF jsonb_array_length(res->'ficha'->'relaciones')<>2 OR res->'ficha'->'relaciones'->1->>'situacion'<>'' THEN
  RAISE EXCEPTION 'ficha propia anterior inesperada %',res; END IF;
 -- Negativos: empleado ajeno a la persona, persona sin empleado, campos
 -- ampliados, operación ajena, material no canónico y concesión caducada.
 ajeno:='emp_sintetico_ajeno_0000000000000001';
 otro:=replace(material,emp,ajeno);
 mh:=encode(sha256(convert_to(otro,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"empleado_ref":"'||ajeno||'"},"atributos":{"conocido_en":"'||conocido||'","material_sha256":"'||mh||'","operacion":"ficha_propia","vigente_en":"2026-09-26"}}','UTF8')),'hex');
 BEGIN
  PERFORM vec_personal.consultar_ficha_propia_empleado_v1(otro,convert_to((cap||jsonb_build_object('efecto_ref',ajeno,'huella_efecto_sha256',rh))::text,'UTF8'),convert_to((decision||jsonb_build_object('recurso_ref',ajeno,'contexto_recurso_huella_sha256',rh))::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  RAISE EXCEPTION 'empleado ajeno admitido';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 otro:=replace(replace(replace(material,'per_sintetica_alcance_p_','per_sintetica_alcance_l_'),'cta_sintetica_alcance_p_','cta_sintetica_alcance_l_'),'prf_sintetico_alcance_p_','prf_sintetico_alcance_l_');
 mh:=encode(sha256(convert_to(otro,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"empleado_ref":"'||emp||'"},"atributos":{"conocido_en":"'||conocido||'","material_sha256":"'||mh||'","operacion":"ficha_propia","vigente_en":"2026-09-26"}}','UTF8')),'hex');
 BEGIN
  PERFORM vec_personal.consultar_ficha_propia_empleado_v1(otro,convert_to((cap||jsonb_build_object('huella_efecto_sha256',rh))::text,'UTF8'),convert_to((decision||jsonb_build_object('principal_id','per_sintetica_alcance_l_00000000000001','perfil_activo_ref','prf_sintetico_alcance_l_0000000000001','contexto_recurso_huella_sha256',rh))::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  RAISE EXCEPTION 'ficha de otra persona admitida';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.consultar_ficha_propia_empleado_v1(material,convert_to(cap::text,'UTF8'),convert_to((decision||jsonb_build_object('campos_permitidos','["corte","evidencia","relaciones","servicios","persona_ref"]'::jsonb))::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  RAISE EXCEPTION 'campos ampliados admitidos';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.consultar_ficha_propia_empleado_v1(material,convert_to((cap||jsonb_build_object('operacion','personal.registro_empleado.ficha.consultar'))::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  RAISE EXCEPTION 'operación ajena admitida';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.consultar_ficha_propia_empleado_v1(replace(material,'"cuenta_version":1,','"cuenta_version": 1,'),convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  RAISE EXCEPTION 'material no canónico admitido';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.consultar_ficha_propia_empleado_v1(material,convert_to(cap::text,'UTF8'),convert_to((decision||jsonb_build_object('valida_hasta','2020-01-01T00:00:00Z'))::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  RAISE EXCEPTION 'concesión caducada admitida';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $p$;
COMMIT;
SQL
# El ejecutor no puede resolver la proyección: el empleado canónico lo aporta
# el DBA del ensayo (en producción lo aporta ContextoActor).
emp22=$(admin_valor "SELECT empleado_ref FROM vec_personal.resolver_empleado_canonico_persona_v1('per_sintetica_alcance_p_00000000000001',clock_timestamp())")
[[ $emp22 =~ ^emp_ ]] || fallo 'la persona de ensayo no tiene empleado canónico'
{ printf "SET vec_prueba.empleado='%s';\n" "$emp22"; cat "$propia_tmp"; } |
  "$motor" exec -i "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_prueba_personal -d "$base" >/dev/null
rm -f "$propia_tmp"
[[ $(admin_valor "SELECT count(*) FROM vec_personal.recibo_ficha_propia_empleado") == 2 ]] || fallo 'recibos de ficha propia inesperados'
ok '000022 ficha propia: relaciones, situación y servicio con denominaciones, sin referencias internas; empleado ajeno, otra persona, campos, operación, material y caducidad denegados'
# Frontera: solo el LOGIN del registrador inscribe; motivos cerrados.
"$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_prueba_frontera_personal -d "$base" \
  -c "SELECT vec_personal.registrar_denegacion_ficha_propia_v1('corr_0123456789abcdef0123456789abcdef','sin_empleado',403::smallint,'per_sintetica_alcance_l_00000000000001')" >/dev/null ||
  fallo 'el registrador no pudo inscribir la denegación'
if "$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_prueba_frontera_personal -d "$base" \
  -c "SELECT vec_personal.registrar_denegacion_ficha_propia_v1('corr_0123456789abcdef0123456789abcdef','autenticacion_requerida',401::smallint,'per_sintetica_alcance_l_00000000000001')" >/dev/null 2>&1; then
  fallo 'autenticación requerida con actor admitida'; fi
# Una ficha que excede el límite de filas se registra con su estado propio (422).
"$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_prueba_frontera_personal -d "$base" \
  -c "SELECT vec_personal.registrar_denegacion_ficha_propia_v1('corr_0123456789abcdef0123456789abcdef','excede_limite',422::smallint,'per_sintetica_alcance_l_00000000000001')" >/dev/null ||
  fallo 'el registrador no pudo inscribir el exceso de filas'
if "$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_prueba_frontera_personal -d "$base" \
  -c "SELECT vec_personal.registrar_denegacion_ficha_propia_v1('corr_0123456789abcdef0123456789abcdef','excede_limite',503::smallint,NULL)" >/dev/null 2>&1; then
  fallo 'exceso de filas con estado ajeno admitido'; fi
if "$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_prueba_personal -d "$base" \
  -c "SELECT vec_personal.registrar_denegacion_ficha_propia_v1('corr_no_disponible','acceso_denegado',403::smallint,NULL)" >/dev/null 2>&1; then
  fallo 'el ejecutor registró una denegación'; fi
if "$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_prueba_frontera_personal -d "$base" \
  -c "SELECT count(*) FROM vec_personal.denegacion_frontera_ficha_propia" >/dev/null 2>&1; then
  fallo 'el registrador lee las denegaciones'; fi
ok '000022 denegaciones de frontera: solo el registrador inscribe, motivo/estado cerrados, sin lectura'
# ---------------------------------------------------------------------------
# 000022, casos adicionales. Una llamada del lector se parametriza con
# variables de transacción (vec_prueba.*) y compara SQLSTATE:mensaje con lo
# esperado; el resultado queda en vec_prueba.res para las comprobaciones.
# ---------------------------------------------------------------------------
Q=per_sintetica_alcance_q_00000000000001
R=per_sintetica_alcance_r_00000000000001
S=per_sintetica_alcance_s_00000000000001
propia_do() {
  cat <<'SQL'
DO $p$
DECLARE conocido text; emp text:=current_setting('vec_prueba.empleado'); per text:=current_setting('vec_prueba.persona');
 pri text:=current_setting('vec_prueba.principal'); vig text:=current_setting('vec_prueba.vigente');
 espera text:=current_setting('vec_prueba.espera'); material text; mh text; rh text; cap jsonb; decision jsonb;
 res jsonb; obtenido text; mensaje text;
BEGIN
 conocido:=to_char(transaction_timestamp() AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 material:='{"esquema":"vec.personal.ficha-propia.consulta.v1","empleado_ref":"'||emp||'","vigente_en":"'||vig||'","conocido_en":"'||conocido||'","actor_ref":"'||per||'","contexto_actor_ref":"ctx:synthetic:b2","contexto_version":1,"cuenta_ref":"cta_sintetica_alcance_p_0000000000001","cuenta_version":1,"perfil_ref":"prf_sintetico_alcance_p_0000000000001","perfil_version":1,"persona_ref":"'||per||'","persona_version":1}';
 mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
 rh:=encode(sha256(convert_to('{"ambitos":{"empleado_ref":"'||emp||'"},"atributos":{"conocido_en":"'||conocido||'","material_sha256":"'||mh||'","operacion":"ficha_propia","vigente_en":"'||vig||'"}}','UTF8')),'hex');
 cap:=jsonb_build_object('operacion','personal.registro_empleado.ficha_propia.consultar','audiencia_consumo','vec_personal.registro_empleado.ficha_propia.v1','efecto_ref',emp,'huella_efecto_sha256',rh);
 decision:=jsonb_build_object('principal_id',pri,'perfil_activo_ref','prf_sintetico_alcance_p_0000000000001','concedida',true,'modulo_id','personal','obligaciones',jsonb_build_array(),'tipo_recurso','ficha_propia_empleado','finalidad','consultar_ficha_propia','campos_permitidos','["corte","evidencia","relaciones","servicios"]'::jsonb,'accion','personal.registro_empleado.ficha_propia.consultar','recurso_ref',emp,'contexto_recurso_huella_sha256',rh,'decision_ref','decision:synthetic:ficha-propia:'||current_setting('vec_prueba.caso'),'valida_hasta','2100-01-01T00:00:00Z');
 IF espera='propagar' THEN
  -- Sin captura: el error aborta la transacción del lector tal cual.
  res:=vec_personal.consultar_ficha_propia_empleado_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  RAISE EXCEPTION 'lector con instantánea obsoleta admitido: %',res;
 END IF;
 BEGIN
  res:=vec_personal.consultar_ficha_propia_empleado_v1(material,convert_to(cap::text,'UTF8'),convert_to(decision::text,'UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  obtenido:='ok';
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS obtenido=RETURNED_SQLSTATE,mensaje=MESSAGE_TEXT;
  obtenido:=obtenido||':'||mensaje;
 END;
 IF obtenido NOT LIKE espera THEN
  RAISE EXCEPTION 'ficha propia %: esperado «%», obtenido «%»',current_setting('vec_prueba.caso'),espera,obtenido;
 END IF;
 PERFORM set_config('vec_prueba.res',coalesce(res::text,''),true);
END $p$;
SQL
}
# caso nivel empleado persona principal vigente espera
lector_propia() {
  printf "BEGIN ISOLATION LEVEL %s;\nSET LOCAL TimeZone='UTC';\n" "$2"
  printf "SELECT set_config('vec_prueba.caso','%s',true),set_config('vec_prueba.empleado','%s',true),set_config('vec_prueba.persona','%s',true),set_config('vec_prueba.principal','%s',true),set_config('vec_prueba.vigente','%s',true),set_config('vec_prueba.espera','%s',true);\n" \
    "$1" "$3" "$4" "$5" "$6" "$7"
  propia_do
}
como_lector() { "$motor" exec -i "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -v VERBOSITY=verbose -U vec_prueba_personal -d "$base"; }

# Siembra: unidad y puesto de la relación vigente de P en org:synthetic, y dos
# nodos que no deben aparecer (otra unidad del organismo y la misma unidad en
# otro organismo). Q: empleado propio con una relación inscrita a otra persona.
# R: dos proyecciones activas (ambigua). S: 201 servicios (más de los 200 que
# admite la ficha).
admin -o /dev/null <<SQL
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
INSERT INTO vec_personal.org_nodo_historia(nodo_ref,revision,organismo_ref,unidad_ref,clase,catalogo_ref,catalogo_version,catalogo_revision,catalogo_entrada_clave,denominacion,vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
VALUES (gen_random_uuid(),1,'org:synthetic','uni:synthetic','centro','estructura-organizativa-dipgra',1,1,'nodo:synthetic:destino','Unidad sintética de destino','2020-01-01',clock_timestamp(),'fuente:synthetic:org','acto:synthetic:org',repeat('c',64)),
 (gen_random_uuid(),1,'org:synthetic','uni:ajena','centro','estructura-organizativa-dipgra',1,1,'nodo:synthetic:ajena','Unidad ajena sintética','2020-01-01',clock_timestamp(),'fuente:synthetic:org','acto:synthetic:org',repeat('c',64)),
 (gen_random_uuid(),1,'org:synthetic:ajeno','uni:synthetic','centro','estructura-organizativa-dipgra',1,1,'nodo:synthetic:otro-organismo','Unidad de otro organismo','2020-01-01',clock_timestamp(),'fuente:synthetic:org','acto:synthetic:org',repeat('c',64));
INSERT INTO vec_personal.version_rpt_historia(version_ref,revision,organismo_ref,codigo_version_fuente,estado,aprobada_en,publicada_en,vigente_desde,conocido_desde,fuente_ref,documento_ref,acto_ref,huella_fuente_sha256)
VALUES ('00000000-0000-4000-8000-0000000000a1',1,'org:synthetic','RPT-SINT-1','publicada','2020-01-01','2020-01-01','2020-01-01',clock_timestamp(),'fuente:synthetic:rpt','doc:synthetic:rpt','acto:synthetic:rpt',repeat('c',64));
INSERT INTO vec_personal.puesto_tipo_historia(tipo_ref,revision,organismo_ref,unidad_ref,rpt_version_ref,rpt_revision,codigo_fila_fuente,denominacion,clasificacion_ref,nivel_destino,vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
VALUES ('00000000-0000-4000-8000-0000000000a2',1,'org:synthetic','uni:synthetic','00000000-0000-4000-8000-0000000000a1',1,'F-1','Técnico sintético de personal','cla:synthetic',20,'2020-01-01',clock_timestamp(),'fuente:synthetic:rpt','acto:synthetic:rpt',repeat('c',64));
INSERT INTO vec_personal.puesto_rpt_historia(puesto_ref,revision,organismo_ref,unidad_ref,tipo_ref,tipo_revision,codigo_puesto_fuente,estado_estructural,vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
VALUES ('00000000-0000-4000-8000-0000000000a3',1,'org:synthetic','uni:synthetic','00000000-0000-4000-8000-0000000000a2',1,'P-1','vigente','2020-01-01',clock_timestamp(),'fuente:synthetic:rpt','acto:synthetic:rpt',repeat('c',64));
INSERT INTO vec_personal.version_plantilla_historia(version_ref,revision,organismo_ref,ejercicio,codigo_version_fuente,estado,aprobada_en,publicada_en,vigente_desde,conocido_desde,fuente_ref,documento_ref,acto_ref,huella_fuente_sha256)
VALUES ('00000000-0000-4000-8000-0000000000a4',1,'org:synthetic',2026,'PL-SINT-2026','publicada','2020-01-01','2020-01-01','2020-01-01',clock_timestamp(),'fuente:synthetic:plantilla','doc:synthetic:plantilla','acto:synthetic:plantilla',repeat('c',64));
INSERT INTO vec_personal.plaza_plantilla_historia(plaza_ref,revision,organismo_ref,unidad_ref,plantilla_version_ref,plantilla_revision,codigo_plaza_fuente,clasificacion_ref,estado_estructural,dotacion_presupuestaria,vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
VALUES ('00000000-0000-4000-8000-0000000000a5',1,'org:synthetic','uni:synthetic','00000000-0000-4000-8000-0000000000a4',1,'PZ-1','cla:synthetic','vigente','acreditada','2020-01-01',clock_timestamp(),'fuente:synthetic:plantilla','acto:synthetic:plantilla',repeat('c',64));
INSERT INTO vec_personal.ocupacion_empleado_historia(ocupacion_ref,revision,relacion_ref,relacion_revision,empleado_ref,organismo_ref,unidad_ref,plaza_ref,plaza_revision,puesto_ref,puesto_revision,clase,modalidad_ref,estado,vigente_desde,conocido_desde,acto_ref,fuente_ref,fuente_version,fuente_huella_sha256,decision_ref,auditoria_ref,catalogo_snapshot)
SELECT 'ocu_sintetica_alcance_p_00000000000001',1,r.relacion_ref,r.revision,r.empleado_ref,r.organismo_ref,'uni:synthetic','00000000-0000-4000-8000-0000000000a5',1,'00000000-0000-4000-8000-0000000000a3',1,'titular','mod:synthetic','vigente','2026-09-25',clock_timestamp(),'acto:synthetic:b2','fuente:synthetic:b2',1,repeat('b',64),'decision:synthetic:ocupacion','auditoria:synthetic:ocupacion','{}'::jsonb
FROM (SELECT DISTINCT ON (relacion_ref) * FROM vec_personal.relacion_servicio_historia
      WHERE empleado_ref='$emp22' AND organismo_ref='org:synthetic' ORDER BY relacion_ref,revision DESC) r;
SELECT vec_personal.publicar_proyeccion_empleado_persona_v1(p.pep,1,p.per,p.emp,'activa',clock_timestamp()-interval '1 hour','2100-01-01',NULL,'prc_maestra_sintetica_p7_000000000001',1,repeat('a',64))
FROM (VALUES ('pep_sintetica_alcance_q_0000000000001','$Q','emp_sintetico_alcance_q_00000000000001'),
             ('pep_sintetica_alcance_r_0000000000001','$R','emp_sintetico_alcance_r_00000000000001'),
             ('pep_sintetica_alcance_r_0000000000002','$R','emp_sintetico_alcance_r_00000000000002'),
             ('pep_sintetica_alcance_s_0000000000001','$S','emp_sintetico_alcance_s_00000000000001')) p(pep,per,emp);
INSERT INTO vec_personal.relacion_servicio_historia(relacion_ref,revision,persona_ref,empleado_ref,organismo_ref,unidad_ref,regimen_ref,modalidad_ref,estado,vigente_desde,conocido_desde,acto_ref,fuente_ref,fuente_version,fuente_huella_sha256,decision_ref,auditoria_ref,catalogo_snapshot)
VALUES ('rel_sintetica_alcance_q_00000000000001',1,'$L','emp_sintetico_alcance_q_00000000000001','org:synthetic','uni:synthetic','reg:synthetic','mod:synthetic','vigente','2026-09-01',clock_timestamp(),'acto:synthetic:b2','fuente:synthetic:b2',1,repeat('b',64),'decision:synthetic:q','auditoria:synthetic:q','{}'::jsonb),
 ('rel_sintetica_alcance_s_00000000000001',1,'$S','emp_sintetico_alcance_s_00000000000001','org:synthetic','uni:synthetic','reg:synthetic','mod:synthetic','vigente','2026-09-01',clock_timestamp(),'acto:synthetic:b2','fuente:synthetic:b2',1,repeat('b',64),'decision:synthetic:s','auditoria:synthetic:s','{}'::jsonb);
INSERT INTO vec_personal.servicio_reconocido_historia(servicio_ref,revision,relacion_ref,relacion_revision,empleado_ref,organismo_ref,clase_ref,dias_reconocidos,periodo_desde,periodo_hasta,estado,vigente_desde,conocido_desde,acto_ref,fuente_ref,fuente_version,fuente_huella_sha256,decision_ref,auditoria_ref,catalogo_snapshot)
SELECT 'srv_sintetico_alcance_s_'||lpad(i::text,14,'0'),1,'rel_sintetica_alcance_s_00000000000001',1,'emp_sintetico_alcance_s_00000000000001','org:synthetic','antiguedad',1,
 date '2000-01-01'+i,date '2000-01-02'+i,'reconocido','2020-01-01',clock_timestamp(),'acto:synthetic:b2','fuente:synthetic:b2',1,repeat('b',64),'decision:synthetic:s','auditoria:synthetic:s','{}'::jsonb
FROM generate_series(1,201) i;
COMMIT;
SQL
[[ $(admin_valor "SELECT count(*) FROM vec_personal.ocupacion_empleado_historia WHERE empleado_ref='$emp22'") == 1 ]] || fallo 'ocupación sintética no sembrada'

# Positivo con unidad y puesto: solo la unidad y el puesto de la relación de
# org:synthetic; ni la otra unidad ni la misma unidad de otro organismo.
{ lector_propia unidad_puesto 'SERIALIZABLE READ WRITE' "$emp22" "$P" "$P" 2026-09-26 ok
  cat <<'SQL'
DO $a$
DECLARE res jsonb:=current_setting('vec_prueba.res')::jsonb; r0 jsonb; r1 jsonb;
BEGIN
 r0:=res->'ficha'->'relaciones'->0; r1:=res->'ficha'->'relaciones'->1;
 IF jsonb_array_length(res->'ficha'->'relaciones')<>2
    OR r1->>'inicio'<>'2026-09-25' OR r1->>'unidad'<>'Unidad sintética de destino' OR r1->>'puesto'<>'Técnico sintético de personal'
    OR r0->>'inicio'<>'2026-09-26' OR r0->>'unidad'<>'' OR r0->>'puesto'<>''
    OR (res->'ficha')::text ~ '(Unidad ajena|otro organismo)'
    OR (res->'ficha')::text ~ '(per|emp|rel|srv|sit|ocu)_' THEN
  RAISE EXCEPTION 'unidad o puesto inesperados %',res; END IF;
END $a$;
COMMIT;
SQL
} | como_lector >/dev/null
ok '000022 unidad y puesto sembrados de la relación propia; otra unidad y otro organismo ausentes'

# Negativos del lector: aislamiento y solo lectura, principal ajeno con la
# misma proyección, relación inscrita a otra persona y empleado ambiguo.
{ lector_propia read_committed 'READ COMMITTED' "$emp22" "$P" "$P" 2026-09-26 '42501:ficha propia denegada'; echo 'COMMIT;'
  lector_propia repeatable_read 'REPEATABLE READ' "$emp22" "$P" "$P" 2026-09-26 '42501:ficha propia denegada'; echo 'COMMIT;'
  lector_propia solo_lectura 'SERIALIZABLE READ ONLY' "$emp22" "$P" "$P" 2026-09-26 '42501:ficha propia denegada'; echo 'COMMIT;'
  lector_propia principal_ajeno 'SERIALIZABLE READ WRITE' "$emp22" "$P" "$L" 2026-09-26 '42501:material de ficha propia incompatible'; echo 'COMMIT;'
  lector_propia relacion_ajena 'SERIALIZABLE READ WRITE' emp_sintetico_alcance_q_00000000000001 "$Q" "$Q" 2026-09-26 '55000:ficha propia incoherente'; echo 'COMMIT;'
  lector_propia ambiguo 'SERIALIZABLE READ WRITE' emp_sintetico_alcance_r_00000000000001 "$R" "$R" 2026-09-26 '42501:empleado ajeno a la persona'; echo 'COMMIT;'
  lector_propia excede 'SERIALIZABLE READ WRITE' emp_sintetico_alcance_s_00000000000001 "$S" "$S" 2026-09-26 '54000:ficha propia excede límite'; echo 'COMMIT;'
} | como_lector >/dev/null
[[ $(admin_valor "SELECT count(*) FROM vec_personal.recibo_ficha_propia_empleado") == 3 ]] || fallo 'un negativo de ficha propia dejó recibo'
ok '000022 READ COMMITTED, REPEATABLE READ y READ ONLY, principal ajeno, relación de otra persona, empleado ambiguo y 201 servicios (54000) denegados sin recibo'

# Carrera: el lector toma su instantánea, otra sesión revoca y confirma la
# proyección de P, y solo entonces el lector consulta. La barrera de 000016
# debe devolver 40001 y no quedar recibo; sin ella se leería la proyección
# revocada como vigente.
carrera_err=$(mktemp)
revocar="SELECT vec_personal.publicar_proyeccion_empleado_persona_v1(h.proyeccion_ref,h.version+1,h.persona_ref,h.empleado_ref,'revocada',h.vigente_desde,h.vigente_hasta,'baja',h.procedencia_ref,h.procedencia_version,h.procedencia_huella_sha256) FROM (SELECT DISTINCT ON (proyeccion_ref) * FROM vec_personal.proyeccion_empleado_persona_historia WHERE persona_ref='$P' ORDER BY proyeccion_ref,version DESC) h"
if { printf "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;\nSET LOCAL TimeZone='UTC';\n"
     printf "SELECT set_config('vec_prueba.caso','carrera',true),set_config('vec_prueba.empleado','%s',true),set_config('vec_prueba.persona','%s',true),set_config('vec_prueba.principal','%s',true),set_config('vec_prueba.vigente','2026-09-26',true),set_config('vec_prueba.espera','propagar',true);\n" "$emp22" "$P" "$P"
     printf '\\! psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d %s -o /dev/null -c "%s"\n' "$base" "$revocar"
     propia_do; echo 'COMMIT;'
   } | como_lector >/dev/null 2>"$carrera_err"; then
  cat "$carrera_err" >&2; rm -f "$carrera_err"; fallo 'lector con instantánea anterior a la revocación admitido'
fi
grep -q '40001' "$carrera_err" || { cat "$carrera_err" >&2; rm -f "$carrera_err"; fallo 'la carrera no falló con 40001'; }
rm -f "$carrera_err"
[[ $(admin_valor "SELECT resultado FROM vec_personal.resolver_empleado_canonico_persona_v1('$P',clock_timestamp())") == sin_empleado ]] || fallo 'la revocación de la carrera no se confirmó'
[[ $(admin_valor "SELECT count(*) FROM vec_personal.recibo_ficha_propia_empleado") == 3 ]] || fallo 'la carrera dejó recibo'
{ lector_propia revocada 'SERIALIZABLE READ WRITE' "$emp22" "$P" "$P" 2026-09-26 '42501:empleado ajeno a la persona'; echo 'COMMIT;'; } | como_lector >/dev/null
ok '000022 carrera: revocación confirmada tras la instantánea da 40001 sin recibo; después, la persona revocada se deniega'
"$motor" restart "$contenedor" >/dev/null
esperar
[[ $(admin_valor "SELECT count(*) FROM vec_personal.recibo_ficha_propia_empleado") == 3 && $(admin_valor "SELECT count(*) FROM vec_personal.denegacion_frontera_ficha_propia") == 2 ]] || fallo 'recibos o denegaciones de ficha propia perdidos tras reinicio'
ok '000022 recibos y denegación conservados tras reinicio'
printf 'PG18 000019: B1 real; ROLLBACK/COMMIT, ACL, alta/replay/hechos, rechazo multi-org ajeno, caducidad con lock, reinicio y proyección consumida por ContextoActor 000007, lista 000021 y ficha propia 000022 (unidad/puesto, aislamiento, principal ajeno, relación de otra persona, ambigüedad y carrera 40001) correctas (AD3 simulado).\n'
