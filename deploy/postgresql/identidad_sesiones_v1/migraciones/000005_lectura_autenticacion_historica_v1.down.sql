-- Lector de autenticacion ORIGINAL: no renovacion, no asercion ni permiso actual.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_identidad_sesiones_v1:migracion:base:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_identidad_sesiones_v1:migracion:lectura_historica:v1',0));
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL timezone='UTC';

DO $guardas$
DECLARE f oid:=to_regprocedure('vec_identidad_sesiones_v1.leer_autenticacion_original_v1(text,text,text)');
 r oid:='vec_identidad_sesiones_v1_lector_historico'::regrole;
 o oid:='vec_identidad_sesiones_v1_propietario'::regrole;
BEGIN
 IF f IS NULL OR (SELECT count(*) FROM pg_proc WHERE pronamespace='vec_identidad_sesiones_v1'::regnamespace
   AND proname='leer_autenticacion_original_v1')<>1
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner=o AND p.prolang=(SELECT oid FROM pg_language WHERE lanname='plpgsql')
   AND p.prokind='f' AND p.prosecdef AND NOT p.proisstrict AND NOT p.proleakproof
   AND p.provolatile='v' AND p.proparallel='u' AND p.proretset AND p.prorettype='record'::regtype
   AND p.pronargs=3 AND p.pronargdefaults=0 AND p.proargdefaults IS NULL
   AND p.provariadic=0 AND p.prosupport=0 AND p.probin IS NULL AND p.prosqlbody IS NULL
   AND p.procost=100 AND p.prorows=1000
   AND p.proargtypes='25 25 25'::oidvector
   AND p.proallargtypes=ARRAY[25,25,25,25,25,25,25,25,25,25,25,25,16,25,25,25,25,25,1184,1184,1184,1184]::oid[]
   AND p.proargmodes=ARRAY['i','i','i','t','t','t','t','t','t','t','t','t','t','t','t','t','t','t','t','t','t','t']::"char"[]
   AND p.proargnames=ARRAY['p_autenticacion_ref','p_sesion_ref','p_autenticacion_sha256',
    'autenticacion_ref','autenticacion_huella_sha256','asercion_ref','sesion_ref','control_sesion_ref','control_sesion_revision','control_sesion_huella_sha256','cuenta_ref','cuenta_ordinaria_ref','cuenta_privilegiada','superficie','metodo_observado','garantia_observada','politica_garantia_ref','politica_garantia_huella_sha256','autenticacion_verificada_en','sesion_emitida_en','sesion_valida_hasta','sesion_revalidada_en']
   AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','statement_timeout=5s','lock_timeout=2s']
   AND octet_length(p.prosrc)=19458
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='de5d267d2cbf354962429e943686d00cc2e57e6e223b2d4dda525259698869eb')
 OR EXISTS(SELECT 1 FROM pg_description WHERE classoid='pg_proc'::regclass AND objoid=f)
 OR EXISTS(SELECT 1 FROM pg_seclabel WHERE classoid='pg_proc'::regclass AND objoid=f)
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f
   AND (a.grantee NOT IN(o,r) OR a.grantor<>o OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR (SELECT count(*) FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a
   WHERE n.oid='vec_identidad_sesiones_v1'::regnamespace AND a.grantee=r)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a
   WHERE n.oid='vec_identidad_sesiones_v1'::regnamespace AND n.nspowner=o
     AND a.grantee=r AND a.grantor=o AND a.privilege_type='USAGE' AND NOT a.is_grantable)
 THEN RAISE EXCEPTION 'definicion o ACL historica alterada' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM pg_depend d WHERE d.refclassid='pg_proc'::regclass AND d.refobjid=f)
 OR EXISTS(SELECT 1 FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f AND d.deptype='e')
 OR EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid<>f
   AND position('leer_autenticacion_original_v1' IN p.prosrc)>0)
 THEN RAISE EXCEPTION 'lector historico tiene consumidores' USING ERRCODE='55000'; END IF;
END $guardas$;
DROP FUNCTION vec_identidad_sesiones_v1.leer_autenticacion_original_v1(text,text,text) RESTRICT;
REVOKE USAGE ON SCHEMA vec_identidad_sesiones_v1 FROM vec_identidad_sesiones_v1_lector_historico RESTRICT;
-- Conservar sesion/control, consumo, cuentas, revalidadores y capacidades interpropietarias.
-- Tras retirar LOGIN/membresias de ensayo, usar roles_historicos_down.sql.
COMMIT;
