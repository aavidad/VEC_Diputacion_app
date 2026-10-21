\set ON_ERROR_STOP on
-- La consulta privada no se convierte en una lectura directa del ejecutor.
-- El positivo portal se comprueba por HTTP con el nombramiento nativo.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $contrato$
DECLARE f oid:='vec_contratacion_temporal.antecedentes_plan_personal_ct155(text,text)'::regprocedure;
BEGIN
 IF NOT has_function_privilege('vec_contratacion_temporal_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
 OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
            WHERE p.oid=f AND a.grantee=0 AND a.privilege_type='EXECUTE')
 OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.pronargs=2
     AND p.prorettype='jsonb'::regtype AND NOT p.prosecdef AND p.provolatile='s'
     AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[]) THEN
  RAISE EXCEPTION 'CT255: lectura privada o contrato divergente';
 END IF;
END $contrato$;
SET LOCAL ROLE vec_contratacion_temporal_ejecutor;
DO $ejecutor$
BEGIN
 BEGIN
  PERFORM vec_contratacion_temporal.antecedentes_plan_personal_ct155(
    'organizacion:f134:ausente','expediente:f134:ausente');
  RAISE EXCEPTION 'CT255: el ejecutor accedió a los antecedentes privados';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $ejecutor$;
ROLLBACK;
