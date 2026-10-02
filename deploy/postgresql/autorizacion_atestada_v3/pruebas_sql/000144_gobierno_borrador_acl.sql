\set ON_ERROR_STOP on
-- Sólo para ensayo PostgreSQL aislado, después de extender el núcleo GobiernoG
-- e instalar AD144. Los materiales negativos se rechazan antes del consumo.
BEGIN ISOLATION LEVEL SERIALIZABLE;
RESET ROLE;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $contrato$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
        nucleo oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
        propietario oid:='vec_bolsa_reglas_baremo_propietario'::regrole;
        ejecutor oid:='vec_bolsa_reglas_baremo_ejecutor_gobierno'::regrole;
BEGIN
 IF f IS NULL OR nucleo IS NULL
 OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR NOT has_schema_privilege(propietario,'vec_autorizacion_atestada_v3','USAGE')
 OR NOT has_function_privilege(propietario,f,'EXECUTE')
 OR has_function_privilege(ejecutor,f,'EXECUTE')
 OR has_function_privilege(ejecutor,nucleo,'EXECUTE')
 OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantee NOT IN (p.proowner,propietario)
    OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
 THEN RAISE EXCEPTION 'AD3-144: fachada o ACL incompatible'; END IF;
END $contrato$;
DO $negativos$
DECLARE rechazados integer:=0;
BEGIN
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada(
   convert_to('{}','UTF8'),convert_to('{}','UTF8'),''::bytea,''::bytea,
   1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea);
  RAISE EXCEPTION 'AD3-144: material vacío autorizado';
 EXCEPTION WHEN insufficient_privilege THEN rechazados:=rechazados+1; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada(
   decode('ffff','hex'),convert_to('{}','UTF8'),''::bytea,''::bytea,
   1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea);
  RAISE EXCEPTION 'AD3-144: UTF8 inválido autorizado';
 EXCEPTION WHEN invalid_parameter_value THEN rechazados:=rechazados+1; END;
 IF rechazados<>2 THEN RAISE EXCEPTION 'AD3-144: negativos incompletos'; END IF;
END $negativos$;
ROLLBACK;
