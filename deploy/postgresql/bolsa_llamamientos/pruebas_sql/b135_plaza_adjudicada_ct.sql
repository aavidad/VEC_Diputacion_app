\set ON_ERROR_STOP on
-- Ejecutar tras B135 en una copia local. No crea ofertas, firmas ni autoridad.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='10s';
DO $test$
DECLARE
 f oid:=to_regprocedure(
  'vec_bolsa_llamamientos.verificar_plaza_adjudicada_ct_v1(text,text,text,integer,integer,text,text)');
 estado text; mensaje text;
BEGIN
 IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
    OR has_table_privilege(current_user,
       'vec_bolsa_llamamientos.acto_plaza_oferta','SELECT')
 THEN RAISE EXCEPTION 'B135: ACL nominal divergente'; END IF;
 IF EXISTS(SELECT 1 FROM pg_proc p
    CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND a.privilege_type='EXECUTE'
      AND a.grantee NOT IN (p.proowner,current_user::regrole))
 THEN RAISE EXCEPTION 'B135: EXECUTE concedido fuera del propietario CT'; END IF;

 -- Un propietario que llama directamente no es el LOGIN ejecutor de CT.
 -- La denegación ocurre antes de consultar cualquier dato personal.
 BEGIN
  PERFORM vec_bolsa_llamamientos.verificar_plaza_adjudicada_ct_v1(
   'organizacion:ensayo','expediente:ensayo',
   'oferta:'||repeat('a',64),1,1,'participacion:ensayo',
   'recibo:plaza-oferta:'||repeat('b',64));
  RAISE EXCEPTION 'B135: llamada directa aceptada';
 EXCEPTION WHEN insufficient_privilege THEN
  GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE,mensaje=MESSAGE_TEXT;
  IF estado<>'42501' OR mensaje<>'B135: sesión denegada' THEN
   RAISE EXCEPTION 'B135: denegación inesperada'; END IF;
 END;
END $test$;
ROLLBACK;
