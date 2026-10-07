\set ON_ERROR_STOP on
-- Ejecutar únicamente después de AD158 sobre un clon PostgreSQL 18 causal.
-- La prueba inspecciona instalación y ACL; un recorrido positivo requiere
-- decisiones V3 publicadas para las dos acciones y Documentos13.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $t$
DECLARE n oid; w oid; fn text; fw text; a record;
BEGIN
 n:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 w:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF n IS NULL OR w IS NULL THEN RAISE EXCEPTION 'AD3-158 prueba: funciones ausentes'; END IF;
 SELECT pg_get_functiondef(n),pg_get_functiondef(w) INTO fn,fw;
 IF strpos(fn,'documentos.generado.alta')=0 OR strpos(fn,'documentos.firmado.custodiar')=0
    OR strpos(fn,'documentos.original_firmable.reservar')=0
    OR strpos(fn,'documentos.original_firmable.confirmar')=0
    OR strpos(fn,'firma_externa_documento_ct')=0
    OR strpos(fw,'documentos.generado.alta')=0 OR strpos(fw,'documentos.firmado.custodiar')=0
    OR strpos(fw,'documentos.original_firmable.reservar')=0
    OR strpos(fw,'documentos.original_firmable.confirmar')=0
    OR strpos(fw,'revalidar_decision_contexto_actor_v3_viva')=0 THEN
  RAISE EXCEPTION 'AD3-158 prueba: acciones o revalidación histórica ausentes'; END IF;
 IF (SELECT proowner FROM pg_proc WHERE oid=n) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT proowner FROM pg_proc WHERE oid=w) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT proconfig FROM pg_proc WHERE oid=n) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR (SELECT proconfig FROM pg_proc WHERE oid=w) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR (SELECT prosecdef FROM pg_proc WHERE oid=n) IS NOT TRUE
    OR (SELECT prosecdef FROM pg_proc WHERE oid=w) IS NOT TRUE THEN
  RAISE EXCEPTION 'AD3-158 prueba: propietario o entorno alterados'; END IF;
 FOR a IN SELECT p.oid,p.proowner,x.grantee,x.grantor,x.privilege_type,x.is_grantable
  FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
  WHERE p.oid IN (n,w) LOOP
  IF a.grantee=0 OR a.grantor<>a.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable
     OR (a.oid=n AND a.grantee<>a.proowner)
     OR (a.oid=w AND a.grantee NOT IN (a.proowner,'vec_documentos_propietario'::regrole)) THEN
   RAISE EXCEPTION 'AD3-158 prueba: ACL de consumo ampliada'; END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
    WHERE p.oid=w AND x.grantee='vec_documentos_propietario'::regrole
      AND x.privilege_type='EXECUTE' AND NOT x.is_grantable) THEN
  RAISE EXCEPTION 'AD3-158 prueba: falta ejecución nominal Documentos'; END IF;
END $t$;
ROLLBACK;
