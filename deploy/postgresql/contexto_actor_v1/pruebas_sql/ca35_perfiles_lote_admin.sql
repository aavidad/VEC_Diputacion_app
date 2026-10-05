\set ON_ERROR_STOP on
-- Vector CA35: estructura y frontera sin crear perfil ni tocar historia.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
DO $prueba$
DECLARE nombre text;f oid;p record;
BEGIN
 FOREACH nombre IN ARRAY ARRAY[
  'crear_perfil_vinculo_admin_lote_v1(text,text,numeric,numeric,text,text,text,numeric,text,timestamptz,timestamptz,timestamptz)',
  'revocar_perfil_vinculo_admin_lote_v1(text,text,text,text,numeric,numeric,numeric,numeric,text,numeric,text,timestamptz)',
  'registrar_procedencia_acto_admin_lote_v1(text,text)',
  'cuentas_titular_persona_admin_lote_v1(text)'
 ] LOOP
  f:=to_regprocedure('vec_contexto_actor_v1.'||nombre);
  SELECT proowner,prosecdef,proconfig,proacl INTO STRICT p FROM pg_proc WHERE oid=f;
  IF p.proowner IS DISTINCT FROM 'vec_contexto_actor_v1_propietario'::regrole OR NOT p.prosecdef
   OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on']
   OR NOT has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
   OR EXISTS(SELECT 1 FROM aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a
     WHERE a.grantee NOT IN(p.proowner,'vec_autorizacion_propietario'::regrole)
       OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
  THEN RAISE EXCEPTION 'CA35: ACL o owner divergente' USING ERRCODE='55000'; END IF;
 END LOOP;
 BEGIN
  PERFORM vec_contexto_actor_v1.crear_perfil_vinculo_admin_lote_v1('', '', 0, 0, '', '', '', 0, '', NULL, NULL, NULL);
  RAISE EXCEPTION 'CA35: alta invalida aceptada';
 EXCEPTION WHEN invalid_parameter_value THEN NULL;
 END;
 BEGIN
  PERFORM vec_contexto_actor_v1.cuentas_titular_persona_admin_lote_v1('no_es_persona');
  RAISE EXCEPTION 'CA35: persona invalida aceptada';
 EXCEPTION WHEN invalid_parameter_value THEN NULL;
 END;
 IF vec_contexto_actor_v1.cuentas_titular_persona_admin_lote_v1('per_'||repeat('z',30)) IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'CA35: persona sin cuentas no devuelve lista vacia'; END IF;
 BEGIN
  PERFORM vec_contexto_actor_v1.registrar_procedencia_acto_admin_lote_v1('prc_x', 'no');
  RAISE EXCEPTION 'CA35: procedencia invalida aceptada';
 EXCEPTION WHEN invalid_parameter_value THEN NULL;
 END;
 BEGIN
  PERFORM vec_contexto_actor_v1.revocar_perfil_vinculo_admin_lote_v1('', '', '', '', 0, 0, 0, 0, '', 0, '', NULL);
  RAISE EXCEPTION 'CA35: baja invalida aceptada';
 EXCEPTION WHEN invalid_parameter_value THEN NULL;
 END;
END $prueba$;
ROLLBACK;
