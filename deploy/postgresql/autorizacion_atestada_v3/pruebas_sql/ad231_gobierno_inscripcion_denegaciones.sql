\set ON_ERROR_STOP on
-- Ejecutar sólo en PG18 aislado después de AUT66 y AD227→AD230→AD228→AD229→AD231.
-- La prueba es reversible y no publica perfiles ni ejecuta actos de inscripción.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $test$
DECLARE fn pg_catalog.regprocedure := 'vec_autorizacion_atestada_v3.consumir_version_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::pg_catalog.regprocedure;
DECLARE antes bigint;
DECLARE despues bigint;
BEGIN
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
   CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE p.oid=fn::pg_catalog.oid AND a.grantee=0 AND a.privilege_type='EXECUTE')
 OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',fn,'EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario',fn,'EXECUTE')
 THEN RAISE EXCEPTION 'AD231: ACL del consumidor ampliada' USING ERRCODE='P0001'; END IF;
 IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_presentacion_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_revision_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_incorporacion_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD231: consumidor Bolsa ajeno ausente' USING ERRCODE='P0001'; END IF;
 SELECT pg_catalog.count(*) INTO antes FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.consumir_version_inscripcion_v3_atestada(
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AD231: sesión no habilitada fue aceptada' USING ERRCODE='P0001';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 SELECT pg_catalog.count(*) INTO despues FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 IF despues IS DISTINCT FROM antes THEN
  RAISE EXCEPTION 'AD231: denegación alteró auditoría' USING ERRCODE='P0001';
 END IF;
END $test$;
ROLLBACK;
