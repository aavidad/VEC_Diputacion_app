\set ON_ERROR_STOP on
-- Sólo sobre PostgreSQL 18 desechable tras AUT76, AUT77 y AD248. ROLLBACK.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $prueba$
DECLARE nuevo text:='vec_admin_version_rol_bolsa_externa_ejecutor';
 anterior text:='vec_admin_version_rol_bolsa_b103_ejecutor';
 propuesta text:='vec_autorizacion.proponer_version_rol_bolsa_externa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
 cierre text:='vec_autorizacion.cerrar_version_rol_bolsa_externa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
 base text:='vec_autorizacion.resolver_base_version_rol_bolsa_externa_v1()';
 resolver_admin text:='vec_autorizacion.resolver_rol_administrable_v1(text)';
 consumidor text:='vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_externa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
 tabla text;v record;p record;
BEGIN
 IF to_regrole(nuevo) IS NULL OR to_regrole(anterior) IS NULL
 OR to_regprocedure(propuesta) IS NULL OR to_regprocedure(cierre) IS NULL
 OR to_regprocedure(base) IS NULL OR to_regprocedure(resolver_admin) IS NULL
 OR to_regprocedure(consumidor) IS NULL
 THEN RAISE EXCEPTION 'AUT77: falta pieza nominal' USING ERRCODE='55000';END IF;
 IF has_function_privilege(nuevo,propuesta,'EXECUTE') IS NOT TRUE
 OR has_function_privilege(nuevo,cierre,'EXECUTE') IS NOT TRUE
 OR has_function_privilege(nuevo,base,'EXECUTE') IS NOT TRUE
 OR has_function_privilege(nuevo,resolver_admin,'EXECUTE') IS NOT TRUE
 OR has_function_privilege(anterior,propuesta,'EXECUTE') IS TRUE
 OR has_function_privilege(anterior,cierre,'EXECUTE') IS TRUE
 OR has_function_privilege(anterior,base,'EXECUTE') IS TRUE
 OR has_function_privilege('public',propuesta,'EXECUTE') IS TRUE
 OR has_function_privilege('public',cierre,'EXECUTE') IS TRUE
 OR has_function_privilege('public',base,'EXECUTE') IS TRUE
 THEN RAISE EXCEPTION 'AUT77: ACL cruzada o pública' USING ERRCODE='42501';END IF;
 IF EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE left(n.nspname,4)='vec_' AND has_function_privilege(nuevo,p.oid,'EXECUTE')
   AND p.oid NOT IN(to_regprocedure(propuesta),to_regprocedure(cierre),
    to_regprocedure(base),to_regprocedure(resolver_admin)))
 THEN RAISE EXCEPTION 'AUT77: ejecutor con funciones adicionales' USING ERRCODE='42501';END IF;
 FOREACH tabla IN ARRAY ARRAY['propuesta_version_rol_bolsa_externa_v1',
  'cierre_version_rol_bolsa_externa_v1','outbox_version_rol_bolsa_externa_v1',
  'sello_efecto_admin_externo_tx_v1'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class c
   WHERE c.oid=to_regclass('vec_autorizacion.'||tabla)
   AND c.relowner=to_regrole('vec_autorizacion_propietario')
   AND c.relrowsecurity AND c.relforcerowsecurity)
  OR has_table_privilege(nuevo,'vec_autorizacion.'||tabla,'SELECT,INSERT,UPDATE,DELETE') IS TRUE
  OR has_table_privilege('public','vec_autorizacion.'||tabla,'SELECT,INSERT,UPDATE,DELETE') IS TRUE
  THEN RAISE EXCEPTION 'AUT77: tabla expuesta %',tabla USING ERRCODE='42501';END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint
  WHERE conrelid='vec_autorizacion.sello_efecto_admin_externo_tx_v1'::regclass
   AND confrelid='vec_autorizacion.asignacion_perfil_externa'::regclass
   AND contype='f')
 THEN RAISE EXCEPTION 'AUT77: sello externo sin FK nominal' USING ERRCODE='55000';END IF;
 IF (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p
  WHERE p.oid=to_regprocedure('vec_autorizacion.rol_candidato_externo_acotado_v1(jsonb)'))
  IS DISTINCT FROM '4212e0388ee883840011ea7874079e24eac5c947b2e8cc3c2f71921e613965da'
 THEN RAISE EXCEPTION 'AUT77: validador técnico anterior alterado' USING ERRCODE='55000';END IF;
 FOR v IN SELECT * FROM (VALUES
  ('vec_autorizacion.publicar_rol_candidato_externo_v1(bytea,text,bytea,text,numeric,text,text,text)',
   '958adcfd570ec730964e5bc39c170ac6a0013a1a49a1d12e437c47191bdadd8b'),
  ('vec_autorizacion.publicar_asignacion_candidato_externo_v1(bytea,text,bigint,text,text,text)',
   '197a1fd1227eb9d0f82b5aaa31bc108cd71aeb05251dc405acbd323775b91f9b')
 ) AS x(firma,fuente_sha) LOOP
  SELECT prosrc,proowner,prosecdef,proacl INTO p FROM pg_proc WHERE oid=to_regprocedure(v.firma);
  IF NOT FOUND OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM v.fuente_sha
  OR p.proowner<>to_regrole('vec_autorizacion_propietario') OR p.prosecdef IS NOT TRUE
  OR has_function_privilege('vec_autorizacion_publicador_candidato_externo',v.firma,'EXECUTE') IS NOT TRUE
  THEN RAISE EXCEPTION 'AUT77: fachada técnica sin cierre prospectivo %',v.firma USING ERRCODE='42501';END IF;
 END LOOP;
END $prueba$;
ROLLBACK;
