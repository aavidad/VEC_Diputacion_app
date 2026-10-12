\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $prueba$
DECLARE f oid;h text;
BEGIN
 f:=to_regprocedure('vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(bytea)');
 IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner='vec_contexto_actor_v1_propietario'::regrole AND p.prosecdef
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s'])
 OR has_function_privilege('public',f,'EXECUTE') IS TRUE
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE') IS TRUE
 OR has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE') IS NOT TRUE
 THEN RAISE EXCEPTION 'CA40: ACL o dueño de la fachada divergente'; END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') INTO h;
 IF h IS DISTINCT FROM 'cdaf5ff8bd35c92084d14e6747103c6859c2e438778c9dc4c7167a78f8eb77e0'
 OR vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(convert_to('{}','UTF8')) IS NOT FALSE
 THEN RAISE EXCEPTION 'CA40: contexto inválido admitido o función alterada'; END IF;
END $prueba$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $revocacion$
DECLARE
 cuenta text:='cta_ca40_aaaaaaaaaaaaaaaaaaaaaa';
 persona text:='per_ca40_aaaaaaaaaaaaaaaaaaaaaa';
 perfil text:='prf_ca40_aaaaaaaaaaaaaaaaaaaaaa';
 vinculo text:='vca_ca40_aaaaaaaaaaaaaaaaaaaaaa';
 organizacion text:='org_ca40aaaaaaaaaaaaaaaaaaaaaa';
 corporativo text:='vcr_ca40_aaaaaaaaaaaaaaaaaaaaaa';
 procedencia text:='prc_ca40_aaaaaaaaaaaaaaaaaaaaaa';
 contexto bytea;
BEGIN
 INSERT INTO vec_contexto_actor_v1.procedencias
 VALUES(procedencia,1,repeat('a',64),'autoridad_maestra_acreditada');
 INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones
 VALUES(cuenta,1,procedencia,1,repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
 INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES(cuenta,1);
 INSERT INTO vec_contexto_actor_v1.persona_versiones
 VALUES(persona,1,procedencia,1,repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
 INSERT INTO vec_contexto_actor_v1.persona_actual VALUES(persona,1);
 INSERT INTO vec_contexto_actor_v1.perfil_versiones
 VALUES(perfil,1,persona,procedencia,1,repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
 INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES(perfil,1);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
 VALUES(vinculo,1,cuenta,perfil,persona,procedencia,1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',
  clock_timestamp()+interval '3 hours');
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES(vinculo,1);
 INSERT INTO vec_contexto_actor_v1.organizacion_versiones
 VALUES(organizacion,1,procedencia,1,repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '3 hours');
 INSERT INTO vec_contexto_actor_v1.organizacion_actual VALUES(organizacion,1);
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones VALUES
 (corporativo,1,cuenta,1,persona,1,perfil,1,vinculo,1,organizacion,1,
  procedencia,1,repeat('a',64),'autoridad_maestra_acreditada',
  'interna_corporativa','consulta_rrhh',procedencia,1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',
  clock_timestamp()+interval '3 hours');
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual
 VALUES(cuenta,'interna_corporativa','consulta_rrhh',corporativo,1);
 contexto:=convert_to(jsonb_build_object('esquema','vec.contexto-actor.vinculado.v2',
  'estado','activo','cuenta_ref',cuenta,'persona_ref',persona,
  'perfil_activo_ref',perfil,'contexto_actor_ref',vinculo,
  'cuenta_version',1,'persona_version',1,'perfil_version',1,'contexto_version',1,
  'vigente_hasta',to_char(clock_timestamp()+interval '1 hour',
   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,'UTF8');
 IF vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(contexto) IS NOT TRUE
 THEN RAISE EXCEPTION 'CA40: contexto corporativo vivo denegado'; END IF;
 INSERT INTO vec_contexto_actor_v1.perfil_versiones
 VALUES(perfil,2,persona,procedencia,1,repeat('a',64),'autoridad_maestra_acreditada','revocado',
  clock_timestamp()-interval '1 minute',clock_timestamp()+interval '3 hours');
 UPDATE vec_contexto_actor_v1.perfil_actual SET version=2 WHERE perfil_ref=perfil;
 IF vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(contexto) IS NOT FALSE
 THEN RAISE EXCEPTION 'CA40: perfil revocado admitido'; END IF;
END $revocacion$;
RESET ROLE;
ROLLBACK;
