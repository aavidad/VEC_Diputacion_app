\set ON_ERROR_STOP on
-- Fixture sintética y transaccional: no deja cuenta, perfil ni candidato.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path=pg_catalog;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES
 ('prc_ctx13_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_ctx13_sintetica_abcdefghijklmnopqrstuv',1,'prc_ctx13_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES('cta_ctx13_sintetica_abcdefghijklmnopqrstuv',1);
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_ctx13_sintetica_abcdefghijklmnopqrstuv',1,'prc_ctx13_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.persona_actual VALUES('per_ctx13_sintetica_abcdefghijklmnopqrstuv',1);
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_ctx13_sintetico_abcdefghijklmnopqrstuv',1,'per_ctx13_sintetica_abcdefghijklmnopqrstuv','prc_ctx13_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES('prf_ctx13_sintetico_abcdefghijklmnopqrstuv',1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_ctx13_sintetico_abcdefghijklmnopqrstuv',1,'cta_ctx13_sintetica_abcdefghijklmnopqrstuv','prf_ctx13_sintetico_abcdefghijklmnopqrstuv','per_ctx13_sintetica_abcdefghijklmnopqrstuv','prc_ctx13_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES('vca_ctx13_sintetico_abcdefghijklmnopqrstuv',1);
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones VALUES
 ('vin_ctx13_sintetico_abcdefghijklmnopqrstuv',1,'per_ctx13_sintetica_abcdefghijklmnopqrstuv','candidato','can_ctx13_sintetico_abcdefghijklmnopqrstuv','prc_ctx13_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_actual VALUES('vin_ctx13_sintetico_abcdefghijklmnopqrstuv',1);
DO $provision$
DECLARE desde timestamptz:=clock_timestamp()-interval '1 minute';
        hasta timestamptz:=clock_timestamp()+interval '1 day'; aprobada text;
BEGIN
 aprobada:=encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.jsonb_build_array(
  'pce_ctx13_sintetica_abcdefghijklmnopqrstuv',1::numeric,
  'cta_ctx13_sintetica_abcdefghijklmnopqrstuv',1::numeric,
  'prf_ctx13_sintetico_abcdefghijklmnopqrstuv',1::numeric,
  'per_ctx13_sintetica_abcdefghijklmnopqrstuv',1::numeric,
  'vca_ctx13_sintetico_abcdefghijklmnopqrstuv',1::numeric,
  'vin_ctx13_sintetico_abcdefghijklmnopqrstuv',1::numeric,
  'can_ctx13_sintetico_abcdefghijklmnopqrstuv','activo',desde,hasta,repeat('b',64))::text,'UTF8')),'hex');
 PERFORM vec_contexto_actor_v1.publicar_provision_candidato_externo_v1(
  'pce_ctx13_sintetica_abcdefghijklmnopqrstuv',0,NULL,aprobada,
  'cta_ctx13_sintetica_abcdefghijklmnopqrstuv',1,
  'prf_ctx13_sintetico_abcdefghijklmnopqrstuv',1,
  'per_ctx13_sintetica_abcdefghijklmnopqrstuv',1,
  'vca_ctx13_sintetico_abcdefghijklmnopqrstuv',1,
  'vin_ctx13_sintetico_abcdefghijklmnopqrstuv',1,
  'can_ctx13_sintetico_abcdefghijklmnopqrstuv','activo',desde,hasta,repeat('b',64));
END $provision$;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $comprobar$
BEGIN
 IF vec_contexto_actor_v1.acreditar_candidato_externo_v1(
    'per_ctx13_sintetica_abcdefghijklmnopqrstuv','prf_ctx13_sintetico_abcdefghijklmnopqrstuv',
    'can_ctx13_sintetico_abcdefghijklmnopqrstuv') IS NOT TRUE
 OR vec_contexto_actor_v1.acreditar_candidato_externo_v1(
    'per_ctx13_sintetica_abcdefghijklmnopqrstuv','prf_ctx13_sintetico_abcdefghijklmnopqrstuv',
    'can_ctx13_ajeno_abcdefghijklmnopqrstuv') IS NOT FALSE
 OR vec_contexto_actor_v1.acreditar_candidato_externo_v1(
    'per_ctx13_sintetica_abcdefghijklmnopqrstuv','prf_ctx13_ajeno_abcdefghijklmnopqrstuv',
    'can_ctx13_sintetico_abcdefghijklmnopqrstuv') IS NOT FALSE
 OR vec_contexto_actor_v1.acreditar_candidato_externo_v1(
    'per_ctx13_ajena_abcdefghijklmnopqrstuv','prf_ctx13_sintetico_abcdefghijklmnopqrstuv',
    'can_ctx13_sintetico_abcdefghijklmnopqrstuv') IS NOT FALSE THEN
  RAISE EXCEPTION 'ContextoActor 000013: identidad, perfil o candidato cruzado aceptado' USING ERRCODE='55000';
 END IF;
END $comprobar$;
RESET ROLE;
DO $acl$ BEGIN
 IF pg_catalog.has_function_privilege('vec_contexto_actor_v1_candidato_externo',
    'vec_contexto_actor_v1.acreditar_candidato_externo_v1(text,text,text)','EXECUTE')
 OR pg_catalog.has_function_privilege('vec_contexto_actor_v1_runtime',
    'vec_contexto_actor_v1.acreditar_candidato_externo_v1(text,text,text)','EXECUTE') THEN
  RAISE EXCEPTION 'ContextoActor 000013: ACL abierta' USING ERRCODE='55000';
 END IF;
END $acl$;
ROLLBACK;
