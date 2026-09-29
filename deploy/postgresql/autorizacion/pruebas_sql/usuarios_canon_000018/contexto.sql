\set ON_ERROR_STOP on
-- Fixture sintética para AUT18: exige CTX15 instalada, además de AUT17/AUT18.
-- Continúa con el SQL del generador Go en la misma conexión; termina en ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $precondicion_ctx15$
BEGIN
 IF pg_catalog.to_regprocedure('vec_contexto_actor_v1.publicar_snapshot_contexto_externo_v1(jsonb,numeric,text,text)') IS NULL
    OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(jsonb,numeric)') IS NULL
    OR pg_catalog.to_regclass('vec_contexto_actor_v1.contexto_externo_versiones') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc
       WHERE oid=pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(text,text,text)')
         AND pg_catalog.strpos(prosrc,'vec_contexto_actor_v1.snapshot_contexto_externo_vigente_v1(')>0)
 THEN RAISE EXCEPTION 'AUT18: fixture requiere CTX15 efectiva' USING ERRCODE='55000'; END IF;
END $precondicion_ctx15$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $provision_snapshot_externo$
DECLARE snapshot jsonb; componente jsonb; k text; referencia text; aprobada text;
        publicada record; antes jsonb; despues jsonb;
BEGIN
 SELECT pg_catalog.jsonb_build_array(
  (SELECT count(*) FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones),
  (SELECT count(*) FROM vec_contexto_actor_v1.persona_versiones),
  (SELECT count(*) FROM vec_contexto_actor_v1.perfil_versiones),
  (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_contexto_versiones),
  (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_referencia_versiones),
  (SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto)) INTO antes;
 snapshot:=pg_catalog.jsonb_build_object(
  'provision_ref','pue_aut18_sintetica_abcdefghijklmnopqrstuv',
  'poblacion','usuarios','estado','activo','vinculo_candidato',null);
 FOREACH k IN ARRAY ARRAY['cuenta','persona','perfil','contexto'] LOOP
  referencia:=CASE k
   WHEN 'cuenta' THEN 'cta_aut18_sintetica_abcdefghijklmnopqrstuv'
   WHEN 'persona' THEN 'per_aut18_sintetica_abcdefghijklmnopqrstuv'
   WHEN 'perfil' THEN 'prf_aut18_sintetico_abcdefghijklmnopqrstuv'
   ELSE 'vca_aut18_sintetico_abcdefghijklmnopqrstuv' END;
  componente:=pg_catalog.jsonb_build_object(
   'referencia',referencia,'version',1,
   'procedencia_ref','prc_aut18_sintetica_abcdefghijklmnopqrstuv','procedencia_version',1,
   'procedencia_huella_sha256',repeat('a',64),
   'procedencia_autoridad','autoridad_maestra_acreditada','estado','activo',
   'vigente_desde','2020-01-01T00:00:00.000000Z',
   'vigente_hasta','2099-01-01T00:00:00.000000Z');
  snapshot:=snapshot||pg_catalog.jsonb_build_object(k,componente);
 END LOOP;
 aprobada:=vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(snapshot,1);
 SELECT * INTO STRICT publicada FROM vec_contexto_actor_v1.publicar_snapshot_contexto_externo_v1(
  snapshot,0,NULL,aprobada);
 IF publicada.version IS DISTINCT FROM 1::numeric
    OR publicada.huella_sha256 IS DISTINCT FROM aprobada
    OR vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
      'cta_aut18_sintetica_abcdefghijklmnopqrstuv','per_aut18_sintetica_abcdefghijklmnopqrstuv',
      'prf_aut18_sintetico_abcdefghijklmnopqrstuv') IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT18: snapshot externo no acreditado' USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.jsonb_build_array(
  (SELECT count(*) FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones),
  (SELECT count(*) FROM vec_contexto_actor_v1.persona_versiones),
  (SELECT count(*) FROM vec_contexto_actor_v1.perfil_versiones),
  (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_contexto_versiones),
  (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_referencia_versiones),
  (SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto)) INTO despues;
 IF antes IS DISTINCT FROM despues
 THEN RAISE EXCEPTION 'AUT18: fixture alteró historia compartida' USING ERRCODE='55000'; END IF;
END $provision_snapshot_externo$;
RESET ROLE;
