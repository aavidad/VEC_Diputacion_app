\set ON_ERROR_STOP on
-- Personal39: lectura de la instantánea RPT publicada desde un fichero fijado
-- por huella. El fichero se valida en Go y sólo se entrega tras consumir la
-- fachada nominal RPT V3 propia de AD217, sobre la postimagen AD216.
-- Consumo, auditoría común y recibo se confirman en la misma transacción.
-- No publica categorías para CT/Bolsa ni acredita vigencia administrativa.
-- Orden causal: AD217 -> Personal39. No aplicar sin su postimagen exacta.
-- DBA prepara fuera de Git el LOGIN vec_personal_rpt_v2_app y el grupo
-- vec_personal_rpt_v2_lector; Personal39 no crea identidades ni credenciales.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000039:rpt-publica-v2',0));
SET LOCAL ROLE vec_personal_propietario;

DO $pre$
DECLARE
 consumidor oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_rpt_publica_v2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 guardia oid:=to_regprocedure('vec_personal.rechazar_mutacion_registro_empleado_v1()');
 grupo_valido boolean; login_valido boolean;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_personal_propietario' THEN
  RAISE EXCEPTION 'PARO clave=P39.entorno, esperado=PG18/vec_personal_propietario, obtenido=%/%',
   current_setting('server_version_num'),current_user USING ERRCODE='55000'; END IF;
 IF consumidor IS NULL THEN
  RAISE EXCEPTION 'PARO clave=P39.fachada_rpt, esperado=presente, obtenido=ausente' USING ERRCODE='55000'; END IF;
 IF (SELECT proowner FROM pg_proc WHERE oid=consumidor) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR (SELECT prosecdef FROM pg_proc WHERE oid=consumidor) IS NOT TRUE THEN
  RAISE EXCEPTION 'PARO clave=P39.fachada_owner_definer, esperado=AD_propietario/true, obtenido=%/%',
   (SELECT proowner::regrole::text FROM pg_proc WHERE oid=consumidor),
   (SELECT prosecdef FROM pg_proc WHERE oid=consumidor) USING ERRCODE='55000'; END IF;
 IF NOT has_schema_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3','USAGE')
 OR NOT has_function_privilege('vec_personal_propietario',consumidor,'EXECUTE') THEN
  RAISE EXCEPTION 'PARO clave=P39.fachada_acl, esperado=USAGE/EXECUTE, obtenido=%/%',
   has_schema_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3','USAGE'),
   has_function_privilege('vec_personal_propietario',consumidor,'EXECUTE') USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM pg_proc p,
   LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=consumidor AND a.privilege_type='EXECUTE'
     AND (a.grantee NOT IN (p.proowner,'vec_personal_propietario'::regrole)
       OR (a.grantee='vec_personal_propietario'::regrole AND a.is_grantable))) THEN
  RAISE EXCEPTION 'PARO clave=P39.fachada_acl_exclusiva, esperado=EXECUTE_solo_AD_owner_y_Personal_owner_sin_grant_option, obtenido=otra_acl' USING ERRCODE='55000'; END IF;
 IF (SELECT nspowner FROM pg_namespace WHERE nspname='vec_personal')
    IS DISTINCT FROM 'vec_personal_propietario'::regrole THEN
  RAISE EXCEPTION 'PARO clave=P39.esquema_owner, esperado=vec_personal_propietario, obtenido=%',
   (SELECT nspowner::regrole::text FROM pg_namespace WHERE nspname='vec_personal') USING ERRCODE='55000'; END IF;
 IF guardia IS NULL OR (SELECT proowner FROM pg_proc WHERE oid=guardia) IS DISTINCT FROM 'vec_personal_propietario'::regrole
 OR (SELECT prorettype FROM pg_proc WHERE oid=guardia) IS DISTINCT FROM 'trigger'::regtype THEN
  RAISE EXCEPTION 'PARO clave=P39.historia_inmutable, esperado=guardia Personal propietaria/trigger, obtenido=%/%',
   guardia IS NOT NULL,(SELECT prorettype::regtype::text FROM pg_proc WHERE oid=guardia) USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_personal.consumir_consulta_rpt_publica_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regclass('vec_personal.recibo_consulta_rpt_publica_v2') IS NOT NULL THEN
  RAISE EXCEPTION 'PARO clave=P39.historia_preexistente, esperado=funcion_ausente/tabla_ausente, obtenido=%/%',
   to_regprocedure('vec_personal.consumir_consulta_rpt_publica_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL,
   to_regclass('vec_personal.recibo_consulta_rpt_publica_v2') IS NOT NULL USING ERRCODE='55000'; END IF;
 SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_personal_rpt_v2_lector'
   AND NOT(rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
   AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=to_regrole('vec_personal_rpt_v2_lector'))
 INTO grupo_valido;
 IF NOT grupo_valido THEN
  RAISE EXCEPTION 'PARO clave=P39.grupo_rpt, esperado=NOLOGIN_no_privilegiado_sin_membresias, obtenido=%',grupo_valido USING ERRCODE='55000'; END IF;
 IF to_regrole('vec_personal_rpt_v2_app') IS NULL THEN
  RAISE EXCEPTION 'PARO clave=P39.login_rpt, esperado=LOGIN_presente, obtenido=ausente' USING ERRCODE='55000'; END IF;
 SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_personal_rpt_v2_app'
   AND rolcanlogin AND rolinherit
   AND NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
   AND EXISTS(SELECT 1 FROM pg_auth_members WHERE member=to_regrole('vec_personal_rpt_v2_app')
     AND roleid=to_regrole('vec_personal_rpt_v2_lector')
     AND inherit_option AND NOT set_option AND NOT admin_option)
   AND (SELECT count(*) FROM pg_auth_members WHERE member=to_regrole('vec_personal_rpt_v2_app'))=1
   AND (SELECT count(*) FROM pg_auth_members WHERE roleid=to_regrole('vec_personal_rpt_v2_lector'))=1
   AND NOT has_database_privilege('vec_personal_rpt_v2_app',current_database(),'TEMPORARY')
   AND NOT has_schema_privilege('vec_personal_rpt_v2_app','vec_personal','CREATE')
 INTO login_valido;
 IF NOT login_valido THEN
  RAISE EXCEPTION 'PARO clave=P39.login_rpt, esperado=LOGIN_INHERIT_una_membresia_RPT_sin_TEMP_CREATE, obtenido=%',login_valido USING ERRCODE='55000'; END IF;
END $pre$;

REVOKE ALL ON SCHEMA vec_personal FROM vec_personal_rpt_v2_lector,vec_personal_rpt_v2_app;
GRANT USAGE ON SCHEMA vec_personal TO vec_personal_rpt_v2_lector;
REVOKE GRANT OPTION FOR USAGE ON SCHEMA vec_personal FROM vec_personal_rpt_v2_lector;

CREATE TABLE vec_personal.recibo_consulta_rpt_publica_v2 (
 recibo_ref text PRIMARY KEY CHECK(recibo_ref ~ '^rptpublica:[0-9a-f-]{36}$'),
 publicacion_ref text NOT NULL CHECK(publicacion_ref ~ '^rpt-publicada:[a-z0-9][a-z0-9:-]{2,127}$'),
 corte date NOT NULL CHECK(isfinite(corte)),
 fuente_sha256 text NOT NULL CHECK(fuente_sha256 ~ '^[a-f0-9]{64}$'),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[a-f0-9]{64}$'),
 decision_ref text NOT NULL CHECK(length(decision_ref) BETWEEN 1 AND 512),
 auditoria_ref text NOT NULL CHECK(length(auditoria_ref) BETWEEN 1 AND 512),
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[a-f0-9]{64}$'),
 consultada_en timestamptz(6) NOT NULL CHECK(isfinite(consultada_en))
);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.recibo_consulta_rpt_publica_v2
 FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.recibo_consulta_rpt_publica_v2
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
ALTER TABLE vec_personal.recibo_consulta_rpt_publica_v2 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.recibo_consulta_rpt_publica_v2 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_interno ON vec_personal.recibo_consulta_rpt_publica_v2
 FOR ALL TO vec_personal_propietario USING(true) WITH CHECK(true);
REVOKE ALL ON TABLE vec_personal.recibo_consulta_rpt_publica_v2 FROM PUBLIC,vec_personal_ejecutor,vec_personal_rpt_v2_lector,vec_personal_rpt_v2_app;
REVOKE ALL ON TYPE vec_personal.recibo_consulta_rpt_publica_v2 FROM PUBLIC,vec_personal_ejecutor,vec_personal_rpt_v2_lector,vec_personal_rpt_v2_app;

CREATE FUNCTION vec_personal.consumir_consulta_rpt_publica_v2(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog, pg_temp SET row_security=on SET timezone='UTC'
 SET lock_timeout='2s' SET statement_timeout='5s' AS $f$
DECLARE
 m jsonb; f jsonb; d jsonb; c jsonb; x jsonb; consumo record;
 pub text; corte date; fuente_sha text; q text; material_sha text;
 contexto_canon text; contexto_sha text; recibo text; ahora timestamptz(6);
 campos constant jsonb:='["categorias_pendientes_grupo","corte","esquema","estado","evidencia","fuente","huella_sha256","items","limit","offset","publicacion_ref","resumen","total","vista"]';
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
 OR session_user<>'vec_personal_rpt_v2_app'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolcanlogin
   AND NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=session_user::regrole
   AND roleid='vec_personal_rpt_v2_lector'::regrole AND inherit_option AND NOT set_option AND NOT admin_option)
 OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_personal_rpt_v2_lector'::regrole)
 OR has_schema_privilege(session_user,'vec_personal','CREATE')
 OR has_database_privilege(session_user,current_database(),'TEMPORARY')
 OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 4096
 OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
 OR p_motivo IS NULL OR p_contexto IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL
 OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
  RAISE EXCEPTION 'Personal39: lectura nominal denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Personal39: material inválido' USING ERRCODE='22023'; END;
 f:=m->'filtro';
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(f) IS DISTINCT FROM 'object'
 OR jsonb_typeof(d) IS DISTINCT FROM 'object' OR jsonb_typeof(c) IS DISTINCT FROM 'object'
 OR jsonb_typeof(x) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
   'actor_ref','contexto_actor_ref','contexto_version','corte','esquema','filtro',
   'huella_sha256','perfil_ref','perfil_version','persona_version','publicacion_ref']
 OR ARRAY(SELECT jsonb_object_keys(f) ORDER BY 1) IS DISTINCT FROM ARRAY['categoria_clave','centro_codigo','limite','offset','q','vista']
 OR m->>'esquema' IS DISTINCT FROM 'vec.personal.rpt-publica.consulta.v2'
 OR m->>'publicacion_ref' !~ '^rpt-publicada:[a-z0-9][a-z0-9:-]{2,127}$'
 OR m->>'corte' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
 OR m->>'huella_sha256' !~ '^[a-f0-9]{64}$'
 OR m->>'actor_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR m->>'contexto_actor_ref' !~ '^vca_[A-Za-z0-9_-]{22,128}$'
 OR m->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR f->>'vista' NOT IN ('categorias','puestos')
 OR (f->>'categoria_clave'<>'' AND (char_length(f->>'categoria_clave')>60
   OR f->>'categoria_clave' !~ '^[a-z][a-z0-9]*(-[a-z0-9]+)*$'))
 OR (f->>'centro_codigo'<>'' AND f->>'centro_codigo' !~ '^[A-Za-z0-9-]{1,64}$')
 OR (f->>'vista'<>'puestos' AND (f->>'categoria_clave'<>'' OR f->>'centro_codigo'<>''))
 OR f->>'limite' !~ '^([1-9]|[1-9][0-9]|100)$'
 OR f->>'offset' !~ '^(0|[1-9][0-9]{0,5})$'
 OR f->>'q' IS NULL OR char_length(f->>'q')>100
 OR f->>'q' IS DISTINCT FROM btrim(f->>'q')
 OR position(E'\n' IN f->>'q')>0 OR position(E'\r' IN f->>'q')>0 OR position(E'\t' IN f->>'q')>0
 OR EXISTS(SELECT 1 FROM jsonb_each(m) e WHERE
   (e.key IN ('contexto_version','perfil_version','persona_version') AND jsonb_typeof(e.value)<>'number')
   OR (e.key='filtro' AND jsonb_typeof(e.value)<>'object')
   OR (e.key NOT IN ('contexto_version','perfil_version','persona_version','filtro') AND jsonb_typeof(e.value)<>'string'))
 OR jsonb_typeof(f->'limite')<>'number' OR jsonb_typeof(f->'offset')<>'number'
 OR jsonb_typeof(f->'q')<>'string' OR jsonb_typeof(f->'vista')<>'string'
 OR jsonb_typeof(f->'categoria_clave')<>'string' OR jsonb_typeof(f->'centro_codigo')<>'string'
 OR EXISTS(SELECT 1 FROM jsonb_each_text(m) e WHERE e.key IN ('contexto_version','perfil_version','persona_version')
   AND e.value !~ '^[1-9][0-9]{0,19}$') THEN
  RAISE EXCEPTION 'Personal39: material incompatible' USING ERRCODE='22023'; END IF;
 BEGIN corte:=(m->>'corte')::date;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Personal39: corte inválido' USING ERRCODE='22023'; END;
 IF corte IS NULL OR NOT isfinite(corte) OR to_char(corte,'YYYY-MM-DD') IS DISTINCT FROM m->>'corte'
 OR (f->>'offset')::integer>100000
 OR m->>'actor_ref' IS DISTINCT FROM x->>'principal_ref'
 OR m->>'contexto_actor_ref' IS DISTINCT FROM x->>'contexto_actor_ref'
 OR m->>'contexto_version' IS DISTINCT FROM x->>'contexto_version'
 OR m->>'perfil_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
 OR m->>'perfil_version' IS DISTINCT FROM x->>'perfil_version'
 OR m->>'persona_version' IS DISTINCT FROM x->>'persona_version'
 OR m->>'persona_version' IS DISTINCT FROM p_persona_version::text
 OR m->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
 OR x->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR x->>'estado' IS DISTINCT FROM 'activo' THEN
  RAISE EXCEPTION 'Personal39: actor o corte divergente' USING ERRCODE='42501'; END IF;
 pub:=m->>'publicacion_ref'; fuente_sha:=m->>'huella_sha256'; q:=f->>'q';
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 -- JSON de map[string]string del recurso Go: claves alfabéticas y ámbito
 -- de publicación obtenido del servidor, nunca de una ruta o cabecera libre.
 contexto_canon:='{"ambitos":{"publicacion_ref":'||to_jsonb(pub)::text||'},"atributos":{"categoria_clave":'||to_jsonb(CASE WHEN f->>'categoria_clave'='' THEN 'sin_filtro' ELSE f->>'categoria_clave' END)::text||
  ',"centro_codigo":'||to_jsonb(CASE WHEN f->>'centro_codigo'='' THEN 'sin_filtro' ELSE f->>'centro_codigo' END)::text||
  ',"corte":'||to_jsonb(m->>'corte')::text||
  ',"huella_sha256":'||to_jsonb(fuente_sha)::text||',"limite":'||to_jsonb(f->>'limite')::text||
  ',"material_sha256":"'||material_sha||'","offset":'||to_jsonb(f->>'offset')::text||
  ',"publicacion_ref":'||to_jsonb(pub)::text||
  ',"q_sha256":"'||encode(sha256(convert_to(q,'UTF8')),'hex')||'","vista":'||to_jsonb(f->>'vista')::text||'}}';
 contexto_sha:=encode(sha256(convert_to(contexto_canon,'UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM 'personal.rpt_publica.consultar'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'rpt_publica_publicacion'
 OR d->>'finalidad' IS DISTINCT FROM 'consultar_organizacion_publicada'
 OR d->'campos_permitidos' IS DISTINCT FROM campos
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR d->>'recurso_ref' IS DISTINCT FROM pub
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
 OR c->>'operacion' IS DISTINCT FROM 'personal.rpt_publica.consultar'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.rpt_publica.consultar.v2'
 OR c->>'efecto_ref' IS DISTINCT FROM pub
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'Personal39: concesión divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_rpt_publica_v2_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR consumo.efecto_ref IS DISTINCT FROM pub OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha
 OR consumo.auditoria_ref IS NULL OR consumo.auditoria_ref=''
 OR consumo.consumo_huella_sha256 IS NULL OR consumo.consumo_huella_sha256 !~ '^[a-f0-9]{64}$' THEN
  RAISE EXCEPTION 'Personal39: consumo divergente' USING ERRCODE='42501'; END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 recibo:='rptpublica:'||gen_random_uuid()::text;
 INSERT INTO vec_personal.recibo_consulta_rpt_publica_v2(recibo_ref,publicacion_ref,corte,fuente_sha256,
  material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(recibo,pub,corte,fuente_sha,material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);
 RETURN jsonb_build_object('evidencia',jsonb_build_object(
  'recibo_ref',recibo,'decision_ref',consumo.decision_ref,'efecto_ref',consumo.efecto_ref,
  'consumo_huella_sha256',consumo.consumo_huella_sha256,'auditoria_ref',consumo.auditoria_ref,
  'consultada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $f$;
REVOKE ALL ON FUNCTION vec_personal.consumir_consulta_rpt_publica_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_personal_ejecutor,vec_personal_rpt_v2_lector,vec_personal_rpt_v2_app;
GRANT EXECUTE ON FUNCTION vec_personal.consumir_consulta_rpt_publica_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_rpt_v2_lector;
REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_personal.consumir_consulta_rpt_publica_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_personal_rpt_v2_lector;
DO $acl$
DECLARE
 funcion oid:='vec_personal.consumir_consulta_rpt_publica_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 grupo oid:='vec_personal_rpt_v2_lector'::regrole;
BEGIN
 IF NOT has_schema_privilege('vec_personal_rpt_v2_app','vec_personal','USAGE')
 OR NOT has_function_privilege('vec_personal_rpt_v2_app',funcion,'EXECUTE')
 OR has_function_privilege('vec_personal_ejecutor',funcion,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc p,
   LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=funcion AND a.privilege_type='EXECUTE'
     AND (a.grantee NOT IN (p.proowner,grupo)
       OR (a.grantee=grupo AND a.is_grantable)))
 OR NOT EXISTS(SELECT 1 FROM pg_proc p,
   LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=funcion AND a.grantee=grupo AND a.privilege_type='EXECUTE') THEN
  RAISE EXCEPTION 'PARO clave=P39.acl_funcion, esperado=EXECUTE_solo_propietario_y_grupo_RPT, obtenido=divergente' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM pg_class t,
   LATERAL aclexplode(coalesce(t.relacl,acldefault('r',t.relowner))) a
   WHERE t.oid='vec_personal.recibo_consulta_rpt_publica_v2'::regclass
     AND a.grantee IN (0,grupo,'vec_personal_rpt_v2_app'::regrole))
 OR EXISTS(SELECT 1 FROM pg_type t,
   LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
   WHERE t.oid='vec_personal.recibo_consulta_rpt_publica_v2'::regtype
     AND a.grantee IN (0,grupo,'vec_personal_rpt_v2_app'::regrole)) THEN
  RAISE EXCEPTION 'PARO clave=P39.acl_recibo, esperado=sin_acceso_PUBLIC_grupo_RPT_LOGIN, obtenido=otra_acl' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
