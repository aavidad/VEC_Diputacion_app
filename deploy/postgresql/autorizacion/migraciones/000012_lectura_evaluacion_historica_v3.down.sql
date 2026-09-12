-- Evaluación histórica: documentos originales, nunca autoridad vigente.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_autorizacion:migracion:registro_contexto_actor_v3:000005',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:lectura_evaluacion_historica_v3:000012',0));
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL timezone='UTC';

DO $guardas$
DECLARE f oid:=to_regprocedure('vec_autorizacion.leer_evaluacion_original_contexto_actor_v3(text,text,text)');
 r oid:='vec_autorizacion_evaluacion_historica_lector'::regrole;
 o oid:='vec_autorizacion_propietario'::regrole;
BEGIN
 IF f IS NULL OR (SELECT count(*) FROM pg_proc WHERE pronamespace='vec_autorizacion'::regnamespace
   AND proname='leer_evaluacion_original_contexto_actor_v3')<>1
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner=o AND p.prolang=(SELECT oid FROM pg_language WHERE lanname='plpgsql')
   AND p.prokind='f' AND p.prosecdef AND NOT p.proisstrict AND NOT p.proleakproof
   AND p.provolatile='v' AND p.proparallel='u' AND p.proretset AND p.prorettype='record'::regtype
   AND p.pronargs=3 AND p.pronargdefaults=0 AND p.proargdefaults IS NULL
   AND p.provariadic=0 AND p.prosupport=0 AND p.probin IS NULL AND p.prosqlbody IS NULL
   AND p.procost=100 AND p.prorows=1000
   AND p.proargtypes='25 25 25'::oidvector
   AND p.proallargtypes=ARRAY[25,25,25,3802,3802,3802,25,25,3802]::oid[]
   AND p.proargmodes=ARRAY['i','i','i','t','t','t','t','t','t']::"char"[]
   AND p.proargnames=ARRAY['p_decision_ref','p_decision_sha256','p_solicitud_sha256',
    'documento_asignacion','documento_rol','documento_control_rol','revision_catalogo','huella_catalogo','documentos_politicas']
   AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','statement_timeout=5s','lock_timeout=2s']
   AND octet_length(p.prosrc)=21075
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='72932131b8bb7bc17cef6cc701870c68023d0f41428da07eca7a9cb89ccdc0ff')
 OR EXISTS(SELECT 1 FROM pg_description WHERE classoid='pg_proc'::regclass AND objoid=f)
 OR EXISTS(SELECT 1 FROM pg_seclabel WHERE classoid='pg_proc'::regclass AND objoid=f)
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f
   AND (a.grantee NOT IN(o,r) OR a.grantor<>o OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR (SELECT count(*) FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a
   WHERE n.oid='vec_autorizacion'::regnamespace AND a.grantee=r)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a
   WHERE n.oid='vec_autorizacion'::regnamespace AND n.nspowner=o
     AND a.grantee=r AND a.grantor=o AND a.privilege_type='USAGE' AND NOT a.is_grantable)
 THEN RAISE EXCEPTION 'definicion o ACL historica alterada' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM pg_depend d WHERE d.refclassid='pg_proc'::regclass AND d.refobjid=f)
 OR EXISTS(SELECT 1 FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f AND d.deptype='e')
 OR EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid<>f
   AND position('leer_evaluacion_original_contexto_actor_v3' IN p.prosrc)>0)
 THEN RAISE EXCEPTION 'lector historico tiene consumidores' USING ERRCODE='55000'; END IF;
END $guardas$;
DROP FUNCTION vec_autorizacion.leer_evaluacion_original_contexto_actor_v3(text,text,text) RESTRICT;
REVOKE USAGE ON SCHEMA vec_autorizacion FROM vec_autorizacion_evaluacion_historica_lector RESTRICT;
-- Preserva toda historia y APIs anteriores. Retirar LOGIN de ensayo antes del rol.
COMMIT;
