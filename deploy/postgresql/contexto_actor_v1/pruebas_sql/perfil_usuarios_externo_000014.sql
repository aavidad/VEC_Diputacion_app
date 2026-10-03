\set ON_ERROR_STOP on
-- Fixture sintética y transaccional: no deja cuenta, perfil ni identidad personal.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path=pg_catalog;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES
 ('prc_ctx14_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_ctx14_sintetica_abcdefghijklmnopqrstuv',1,'prc_ctx14_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES('cta_ctx14_sintetica_abcdefghijklmnopqrstuv',1);
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_ctx14_sintetica_abcdefghijklmnopqrstuv',1,'prc_ctx14_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.persona_actual VALUES('per_ctx14_sintetica_abcdefghijklmnopqrstuv',1);
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_ctx14_sintetico_abcdefghijklmnopqrstuv',1,'per_ctx14_sintetica_abcdefghijklmnopqrstuv','prc_ctx14_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES('prf_ctx14_sintetico_abcdefghijklmnopqrstuv',1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_ctx14_sintetico_abcdefghijklmnopqrstuv',1,'cta_ctx14_sintetica_abcdefghijklmnopqrstuv','prf_ctx14_sintetico_abcdefghijklmnopqrstuv','per_ctx14_sintetica_abcdefghijklmnopqrstuv','prc_ctx14_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES('vca_ctx14_sintetico_abcdefghijklmnopqrstuv',1);
DO $provision$
DECLARE desde timestamptz:=clock_timestamp()-interval '1 minute';
        hasta timestamptz:=clock_timestamp()+interval '1 day'; aprobada text;
BEGIN
 aprobada:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.jsonb_build_array(
  'pue_ctx14_sintetica_abcdefghijklmnopqrstuv',1::numeric,
  'cta_ctx14_sintetica_abcdefghijklmnopqrstuv',1::numeric,
  'per_ctx14_sintetica_abcdefghijklmnopqrstuv',1::numeric,
  'prf_ctx14_sintetico_abcdefghijklmnopqrstuv',1::numeric,
  'vca_ctx14_sintetico_abcdefghijklmnopqrstuv',1::numeric,
  'activo',desde,hasta,repeat('b',64))::text,'UTF8')),'hex');
 PERFORM vec_contexto_actor_v1.publicar_perfil_usuarios_externo_v1(
  'pue_ctx14_sintetica_abcdefghijklmnopqrstuv',0,NULL,aprobada,
  'cta_ctx14_sintetica_abcdefghijklmnopqrstuv',1,
  'per_ctx14_sintetica_abcdefghijklmnopqrstuv',1,
  'prf_ctx14_sintetico_abcdefghijklmnopqrstuv',1,
  'vca_ctx14_sintetico_abcdefghijklmnopqrstuv',1,
  'activo',desde,hasta,repeat('b',64));
END $provision$;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $consultas$
BEGIN
 IF vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
    'cta_ctx14_sintetica_abcdefghijklmnopqrstuv','per_ctx14_sintetica_abcdefghijklmnopqrstuv',
    'prf_ctx14_sintetico_abcdefghijklmnopqrstuv') IS NOT TRUE
 OR vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(
    'per_ctx14_sintetica_abcdefghijklmnopqrstuv','prf_ctx14_sintetico_abcdefghijklmnopqrstuv') IS NOT TRUE
 OR vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
    'cta_ctx14_ajena_abcdefghijklmnopqrstuv','per_ctx14_sintetica_abcdefghijklmnopqrstuv',
    'prf_ctx14_sintetico_abcdefghijklmnopqrstuv') IS NOT FALSE
 OR vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
    'cta_ctx14_sintetica_abcdefghijklmnopqrstuv','per_ctx14_ajena_abcdefghijklmnopqrstuv',
    'prf_ctx14_sintetico_abcdefghijklmnopqrstuv') IS NOT FALSE
 OR vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
    'cta_ctx14_sintetica_abcdefghijklmnopqrstuv','per_ctx14_sintetica_abcdefghijklmnopqrstuv',
    'prf_ctx14_ajeno_abcdefghijklmnopqrstuv') IS NOT FALSE
 OR vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(
    'per_ctx14_ajena_abcdefghijklmnopqrstuv','prf_ctx14_sintetico_abcdefghijklmnopqrstuv') IS NOT FALSE THEN
  RAISE EXCEPTION 'ContextoActor 000014: cuenta, persona o perfil cruzado' USING ERRCODE='55000';
 END IF;
END $consultas$;
RESET ROLE;
DO $identidad$
BEGIN
 BEGIN
  INSERT INTO vec_contexto_actor_v1.perfil_usuarios_externo_identidad
   (perfil_ref,provision_ref,cuenta_ref,persona_ref,contexto_ref)
  VALUES('prf_ctx14_sintetico_abcdefghijklmnopqrstuv',
   'pue_ctx14_otro_abcdefghijklmnopqrstuv',
   'cta_ctx14_ajena_abcdefghijklmnopqrstuv',
   'per_ctx14_sintetica_abcdefghijklmnopqrstuv',
   'vca_ctx14_sintetico_abcdefghijklmnopqrstuv');
  RAISE EXCEPTION 'perfil histórico reutilizado' USING ERRCODE='55000';
 EXCEPTION WHEN unique_violation THEN NULL;
 END;
END $identidad$;
-- El mismo perfil con otra cuenta corporativa histórica nunca es externo.
SAVEPOINT comprobacion_corporativa;
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_ctx14_corporativa_abcdefghijklmnopqrstuv',1,'prc_ctx14_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES('cta_ctx14_corporativa_abcdefghijklmnopqrstuv',1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_ctx14_corporativo_abcdefghijklmnopqrstuv',1,
  'cta_ctx14_corporativa_abcdefghijklmnopqrstuv','prf_ctx14_sintetico_abcdefghijklmnopqrstuv',
  'per_ctx14_sintetica_abcdefghijklmnopqrstuv','prc_ctx14_sintetica_abcdefghijklmnopqrstuv',1,
  repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES('vca_ctx14_corporativo_abcdefghijklmnopqrstuv',1);
INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES
 ('org_ctx14sinteticaabcdefghijklmnopqrstuv',1,'prc_ctx14_sintetica_abcdefghijklmnopqrstuv',1,
  repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones VALUES
 ('vcr_ctx14_sintetico_abcdefghijklmnopqrstuv',1,
  'cta_ctx14_corporativa_abcdefghijklmnopqrstuv',1,
  'per_ctx14_sintetica_abcdefghijklmnopqrstuv',1,
  'prf_ctx14_sintetico_abcdefghijklmnopqrstuv',1,
  'vca_ctx14_corporativo_abcdefghijklmnopqrstuv',1,
  'org_ctx14sinteticaabcdefghijklmnopqrstuv',1,
  'prc_ctx14_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada',
  'interna_corporativa','consulta_rrhh',
  'prc_ctx14_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES
 ('cta_ctx14_corporativa_abcdefghijklmnopqrstuv','interna_corporativa','consulta_rrhh',
  'vcr_ctx14_sintetico_abcdefghijklmnopqrstuv',1);
SET LOCAL ROLE vec_autorizacion_propietario;
DO $corporativo$
BEGIN
 IF vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
    'cta_ctx14_sintetica_abcdefghijklmnopqrstuv','per_ctx14_sintetica_abcdefghijklmnopqrstuv',
    'prf_ctx14_sintetico_abcdefghijklmnopqrstuv') IS NOT FALSE
 OR vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(
    'per_ctx14_sintetica_abcdefghijklmnopqrstuv','prf_ctx14_sintetico_abcdefghijklmnopqrstuv') IS NOT FALSE THEN
  RAISE EXCEPTION 'ContextoActor 000014: perfil corporativo cruzado aceptado' USING ERRCODE='55000';
 END IF;
END $corporativo$;
RESET ROLE;
ROLLBACK TO SAVEPOINT comprobacion_corporativa;
RELEASE SAVEPOINT comprobacion_corporativa;
DO $revocacion$
DECLARE p record; desde timestamptz:=clock_timestamp()-interval '1 minute';
        hasta timestamptz:=clock_timestamp()+interval '1 day'; aprobada text;
BEGIN
 SELECT v.* INTO STRICT p FROM vec_contexto_actor_v1.perfil_usuarios_externo_actual a
 JOIN vec_contexto_actor_v1.perfil_usuarios_externo_versiones v USING(provision_ref,version)
 WHERE a.provision_ref='pue_ctx14_sintetica_abcdefghijklmnopqrstuv';
 aprobada:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.jsonb_build_array(
  p.provision_ref,2::numeric,p.cuenta_ref,p.cuenta_version,p.persona_ref,p.persona_version,
  p.perfil_ref,p.perfil_version,p.contexto_ref,p.contexto_version,'revocado',desde,hasta,
  p.fuente_huella_sha256)::text,'UTF8')),'hex');
 PERFORM vec_contexto_actor_v1.publicar_perfil_usuarios_externo_v1(
  p.provision_ref,p.version,p.huella_sha256,aprobada,
  p.cuenta_ref,p.cuenta_version,p.persona_ref,p.persona_version,
  p.perfil_ref,p.perfil_version,p.contexto_ref,p.contexto_version,
  'revocado',desde,hasta,p.fuente_huella_sha256);
END $revocacion$;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $tras_revocar$
BEGIN
 IF vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
    'cta_ctx14_sintetica_abcdefghijklmnopqrstuv','per_ctx14_sintetica_abcdefghijklmnopqrstuv',
    'prf_ctx14_sintetico_abcdefghijklmnopqrstuv') IS NOT FALSE
 OR vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(
    'per_ctx14_sintetica_abcdefghijklmnopqrstuv','prf_ctx14_sintetico_abcdefghijklmnopqrstuv') IS NOT FALSE THEN
  RAISE EXCEPTION 'ContextoActor 000014: perfil revocado aceptado' USING ERRCODE='55000';
 END IF;
END $tras_revocar$;
RESET ROLE;
DO $acl$
BEGIN
 IF pg_catalog.has_function_privilege('vec_contexto_actor_v1_runtime',
    'vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(text,text,text)','EXECUTE')
 OR pg_catalog.has_function_privilege('vec_contexto_actor_v1_candidato_externo',
    'vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(text,text)','EXECUTE')
 OR pg_catalog.has_table_privilege('vec_autorizacion_propietario',
    'vec_contexto_actor_v1.perfil_usuarios_externo_actual','SELECT') THEN
  RAISE EXCEPTION 'ContextoActor 000014: ACL abierta' USING ERRCODE='55000';
 END IF;
END $acl$;
ROLLBACK;
