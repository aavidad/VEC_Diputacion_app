-- Historia original Contexto V2. Sin tablas, punteros vivos ni nuevas sesiones.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:base:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:acreditacion_uso:v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:lectura_historica:v2',0));
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL timezone='UTC';

DO $guardas$
DECLARE f oid:=to_regprocedure('vec_contexto_actor_v1.leer_contexto_original_v2(text,text,text)');
 r oid:='vec_contexto_actor_v1_lector_historico'::regrole;
 o oid:='vec_contexto_actor_v1_propietario'::regrole;
BEGIN
 IF f IS NULL OR (SELECT count(*) FROM pg_proc WHERE pronamespace='vec_contexto_actor_v1'::regnamespace
   AND proname='leer_contexto_original_v2')<>1
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner=o AND p.prolang=(SELECT oid FROM pg_language WHERE lanname='plpgsql')
   AND p.prokind='f' AND p.prosecdef AND NOT p.proisstrict AND NOT p.proleakproof
   AND p.provolatile='v' AND p.proparallel='u' AND p.proretset AND p.prorettype='record'::regtype
   AND p.pronargs=3 AND p.pronargdefaults=0 AND p.proargdefaults IS NULL
   AND p.provariadic=0 AND p.prosupport=0 AND p.probin IS NULL AND p.prosqlbody IS NULL
   AND p.procost=100 AND p.prorows=1000
   AND p.proargtypes='25 25 25'::oidvector
   AND p.proallargtypes=ARRAY[25,25,25,25,17,25,17,25,25,1184]::oid[]
   AND p.proargmodes=ARRAY['i','i','i','t','t','t','t','t','t','t']::"char"[]
   AND p.proargnames=ARRAY['p_registro_ref','p_contexto_sha256','p_procedencia_sha256',
    'registro_contexto_ref','representacion_canonica','huella_sha256','manifiesto_procedencia_canonico',
    'manifiesto_procedencia_huella_sha256','autoridad_efectiva','resuelto_en']
   AND p.proconfig=ARRAY['search_path=pg_catalog','statement_timeout=5s','lock_timeout=2s']
   AND octet_length(p.prosrc)=17293
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='b6a5a206b63a2c1df5924a6e9a16a0884dd320171585214adf28992e744bb713')
 OR EXISTS(SELECT 1 FROM pg_description WHERE classoid='pg_proc'::regclass AND objoid=f)
 OR EXISTS(SELECT 1 FROM pg_seclabel WHERE classoid='pg_proc'::regclass AND objoid=f)
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f
   AND (a.grantee NOT IN(o,r) OR a.grantor<>o OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR (SELECT count(*) FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a
   WHERE n.oid='vec_contexto_actor_v1'::regnamespace AND a.grantee=r)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a
   WHERE n.oid='vec_contexto_actor_v1'::regnamespace AND n.nspowner=o
     AND a.grantee=r AND a.grantor=o AND a.privilege_type='USAGE' AND NOT a.is_grantable)
 THEN RAISE EXCEPTION 'definicion o ACL historica alterada' USING ERRCODE='55000'; END IF;
 -- Dependencias catalogadas y consumidores PL/pgSQL de nombre literal.
 IF EXISTS(SELECT 1 FROM pg_depend d WHERE d.refclassid='pg_proc'::regclass AND d.refobjid=f)
 OR EXISTS(SELECT 1 FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f AND d.deptype='e')
 OR EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid<>f
   AND position('leer_contexto_original_v2' IN p.prosrc)>0)
 THEN RAISE EXCEPTION 'lector historico tiene consumidores' USING ERRCODE='55000'; END IF;
END $guardas$;
DROP FUNCTION vec_contexto_actor_v1.leer_contexto_original_v2(text,text,text) RESTRICT;
REVOKE USAGE ON SCHEMA vec_contexto_actor_v1 FROM vec_contexto_actor_v1_lector_historico RESTRICT;
-- No toca recibos, procedencias, versiones, resolutor ni rol de runtime.
-- Retirar el rol, una vez quitados sus LOGIN de ensayo, mediante roles_historicos_down.
COMMIT;
