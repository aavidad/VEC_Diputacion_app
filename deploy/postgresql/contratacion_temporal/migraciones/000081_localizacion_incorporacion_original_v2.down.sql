\set ON_ERROR_STOP on
-- Retirar solo fachada; conservar registro, raiz, estados, auditoria y outbox.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000081:localizador:v1',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL timezone='UTC';

DO $guardas$
DECLARE f oid:=to_regprocedure('vec_contratacion_temporal.localizar_incorporacion_original_v2(text,text,text)');
 r oid:='vec_contratacion_temporal_localizador_incorporacion'::regrole;
 o oid:='vec_contratacion_temporal_propietario'::regrole;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE oid=o AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
   AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE oid=r AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
   AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
   AND rolconfig IS NULL AND rolconnlimit=-1 AND rolvaliduntil IS NULL)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=r)
 OR f IS NULL OR (SELECT count(*) FROM pg_proc WHERE pronamespace='vec_contratacion_temporal'::regnamespace
   AND proname='localizar_incorporacion_original_v2')<>1
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner=o AND p.prolang=(SELECT oid FROM pg_language WHERE lanname='plpgsql')
   AND p.prokind='f' AND p.prosecdef AND NOT p.proisstrict AND NOT p.proleakproof
   AND p.provolatile='v' AND p.proparallel='u' AND p.proretset AND p.prorettype='record'::regtype
   AND p.pronargs=3 AND p.pronargdefaults=0 AND p.proargdefaults IS NULL
   AND p.provariadic=0 AND p.prosupport=0 AND p.probin IS NULL AND p.prosqlbody IS NULL
   AND p.procost=100 AND p.prorows=1000
   AND p.proargtypes='25 25 25'::oidvector
   AND p.proallargtypes=ARRAY[25,25,25,25,25,25]::oid[]
   AND p.proargmodes=ARRAY['i','i','i','t','t','t']::"char"[]
   AND p.proargnames=ARRAY['p_organizacion_ref','p_expediente_ref','p_solicitud_personal_ref','recibo_ref','material_sha256','intencion_sha256']
   AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','statement_timeout=5s','lock_timeout=2s']
   AND octet_length(p.prosrc)=27241
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='c6444520d67ab4871f0a921e1ea282a1a2d0e7c89b3eea2f3d4689480e3bd05b')
 OR EXISTS(SELECT 1 FROM pg_description WHERE classoid='pg_proc'::regclass AND objoid=f)
 OR EXISTS(SELECT 1 FROM pg_seclabel WHERE classoid='pg_proc'::regclass AND objoid=f)
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f
   AND (a.grantee NOT IN(o,r) OR a.grantor<>o OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR (SELECT count(*) FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a
   WHERE n.oid='vec_contratacion_temporal'::regnamespace AND a.grantee=r)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a
   WHERE n.oid='vec_contratacion_temporal'::regnamespace AND n.nspowner=o
     AND a.grantee=r AND a.grantor=o AND a.privilege_type='USAGE' AND NOT a.is_grantable)
 OR (SELECT count(*) FROM pg_shdepend WHERE refclassid='pg_authid'::regclass AND refobjid=r)<>3
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_authid'::regclass AND refobjid=r
   AND (deptype<>'a' OR objsubid<>0 OR NOT (
     (classid='pg_database'::regclass AND objid=(SELECT oid FROM pg_database WHERE datname=current_database())) OR
     (classid='pg_namespace'::regclass AND objid='vec_contratacion_temporal'::regnamespace) OR
     (classid='pg_proc'::regclass AND objid=f))))
 THEN RAISE EXCEPTION 'definicion o ACL historica alterada' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM pg_depend d WHERE d.refclassid='pg_proc'::regclass AND d.refobjid=f)
 OR EXISTS(SELECT 1 FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f AND d.deptype='e')
 OR EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid<>f
   AND position('localizar_incorporacion_original_v2' IN p.prosrc)>0)
 THEN RAISE EXCEPTION 'lector historico tiene consumidores' USING ERRCODE='55000'; END IF;
END $guardas$;
DROP FUNCTION vec_contratacion_temporal.localizar_incorporacion_original_v2(text,text,text) RESTRICT;
REVOKE USAGE ON SCHEMA vec_contratacion_temporal FROM vec_contratacion_temporal_localizador_incorporacion RESTRICT;
-- Sin borrar historia, alterar codecs ni ampliar roles ejecutores.
-- Tras retirar LOGIN/membresias de ensayo, usar roles_localizador_incorporacion_down.sql.
COMMIT;
