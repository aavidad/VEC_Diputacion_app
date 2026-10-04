\set ON_ERROR_STOP on
-- Vector estructural tras UP44. El escenario funcional requiere V3 real.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
DO $prueba$
DECLARE f oid;check_evento text;
BEGIN
 IF to_regrole('vec_admin_perfiles_lote_ejecutor') IS NULL
 OR to_regclass('vec_autorizacion.material_fuentes_lote_admin_v1') IS NULL
 THEN RAISE EXCEPTION 'AUT44: tabla de fuentes ausente' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion.aplicar_lote_ordinario_admin_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef)
 OR has_function_privilege('vec_admin_perfiles_lote_ejecutor',f,'EXECUTE') IS NOT TRUE
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND(a.grantee NOT IN(p.proowner,'vec_admin_perfiles_lote_ejecutor'::regrole)
   OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AUT44: ACL de fachada ampliada' USING ERRCODE='55000'; END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT check_evento FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion.outbox_acto_admin_v1'::regclass
 AND c.conname='outbox_acto_admin_v1_evento_check' AND c.contype='c' AND c.convalidated;
 IF strpos(check_evento,'lote_perfiles_aplicado')=0
 THEN RAISE EXCEPTION 'AUT44: evento de outbox ausente' USING ERRCODE='55000'; END IF;
 BEGIN
  PERFORM vec_autorizacion.aplicar_lote_ordinario_admin_v1('{}',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AUT44: superusuario admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $prueba$;
ROLLBACK;
