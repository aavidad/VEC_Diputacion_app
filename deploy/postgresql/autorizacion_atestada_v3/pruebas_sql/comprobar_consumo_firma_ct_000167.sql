\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL timezone='UTC';
DO $test$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(jsonb)'::regprocedure;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
  AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
  AND strpos(p.prosrc,'vec_contratacion_temporal.firma_vec.v2')>0
  AND strpos(p.prosrc,'vec_contratacion_temporal.firma_externa.v2')>0
  AND strpos(p.prosrc,'vec_contratacion_temporal.firma_vec.v1')=0
  AND strpos(p.prosrc,'vec_contratacion_temporal.firma_externa.v1')=0)
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
   AND a.grantee NOT IN ('vec_autorizacion_atestada_v3_propietario'::regrole,
      'vec_autorizacion_propietario'::regrole))
  OR NOT has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE') THEN
  RAISE EXCEPTION 'AD167 ACL/propietario divergente' USING ERRCODE='55000'; END IF;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1('{}'::jsonb);
  RAISE EXCEPTION 'AD167 aceptó consumo vacío' USING ERRCODE='55000';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
 -- Un recibo comprometido en una transacción anterior nunca pasa la prueba
 -- de xmin de consumo y auditoría, aunque su JSON declare consumo_nuevo=true.
END $test$;
ROLLBACK;
