\set ON_ERROR_STOP on
-- Fixture sintética y transaccional: no deja cuenta, perfil ni identidad personal.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path=pg_catalog;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES
 ('prc_aut18_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_aut18_sintetica_abcdefghijklmnopqrstuv',1,'prc_aut18_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES('cta_aut18_sintetica_abcdefghijklmnopqrstuv',1);
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_aut18_sintetica_abcdefghijklmnopqrstuv',1,'prc_aut18_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.persona_actual VALUES('per_aut18_sintetica_abcdefghijklmnopqrstuv',1);
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_aut18_sintetico_abcdefghijklmnopqrstuv',1,'per_aut18_sintetica_abcdefghijklmnopqrstuv','prc_aut18_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES('prf_aut18_sintetico_abcdefghijklmnopqrstuv',1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_aut18_sintetico_abcdefghijklmnopqrstuv',1,'cta_aut18_sintetica_abcdefghijklmnopqrstuv','prf_aut18_sintetico_abcdefghijklmnopqrstuv','per_aut18_sintetica_abcdefghijklmnopqrstuv','prc_aut18_sintetica_abcdefghijklmnopqrstuv',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES('vca_aut18_sintetico_abcdefghijklmnopqrstuv',1);
DO $provision$
DECLARE desde timestamptz:=clock_timestamp()-interval '1 minute';
        hasta timestamptz:=clock_timestamp()+interval '1 day'; aprobada text;
BEGIN
 aprobada:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.jsonb_build_array(
  'pue_aut18_sintetica_abcdefghijklmnopqrstuv',1::numeric,
  'cta_aut18_sintetica_abcdefghijklmnopqrstuv',1::numeric,
  'per_aut18_sintetica_abcdefghijklmnopqrstuv',1::numeric,
  'prf_aut18_sintetico_abcdefghijklmnopqrstuv',1::numeric,
  'vca_aut18_sintetico_abcdefghijklmnopqrstuv',1::numeric,
  'activo',desde,hasta,repeat('b',64))::text,'UTF8')),'hex');
 PERFORM vec_contexto_actor_v1.publicar_perfil_usuarios_externo_v1(
  'pue_aut18_sintetica_abcdefghijklmnopqrstuv',0,NULL,aprobada,
  'cta_aut18_sintetica_abcdefghijklmnopqrstuv',1,
  'per_aut18_sintetica_abcdefghijklmnopqrstuv',1,
  'prf_aut18_sintetico_abcdefghijklmnopqrstuv',1,
  'vca_aut18_sintetico_abcdefghijklmnopqrstuv',1,
  'activo',desde,hasta,repeat('b',64));
END $provision$;
RESET ROLE;
