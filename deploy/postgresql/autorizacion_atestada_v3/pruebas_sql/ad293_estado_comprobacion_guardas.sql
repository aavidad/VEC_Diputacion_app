\set ON_ERROR_STOP on
-- Tras AD293 en copia desechable. Fuente y permisos; no finge V3 ni toca datos.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
DO $guardas$ DECLARE f regprocedure;x record;
BEGIN
 f:='vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text)'::regprocedure;
 SELECT p.prosecdef,p.provolatile,p.proacl,p.proconfig,
  encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') AS huella
 INTO STRICT x FROM pg_proc p WHERE p.oid=f;
 IF x.prosecdef IS NOT TRUE OR x.provolatile<>'v'
 OR x.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
   'TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']
 OR x.huella IS DISTINCT FROM '81549c426b997d792e0c3e5d56adf9720f25633ceb518a4d989ba120e210de32'
 OR x.proacl::text IS DISTINCT FROM
   '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_autorizacion_atestada_v3_registrador_intentos=X/vec_autorizacion_atestada_v3_propietario}'
 OR EXISTS(SELECT 1 FROM aclexplode(coalesce(x.proacl,acldefault('f',(SELECT proowner FROM pg_proc WHERE oid=f)))) a
    WHERE a.grantee=0 AND a.privilege_type='EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD293: ACL o cuerpo divergente'; END IF;
 IF (SELECT proacl::text FROM pg_proc WHERE oid=
   'vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text,jsonb)'::regprocedure)
   IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_bolsa_llamamientos_propietario}'
 THEN RAISE EXCEPTION 'AD293: lectura Bolsa abierta'; END IF;
END $guardas$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_registrador_intentos;
DO $denegada$ BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(
   '{}'::bytea,'{}'::bytea,'{"accion":"bolsa.origen_ct.consultar","comprobacion_contexto":{}}'::jsonb,
   '{}'::jsonb,'expediente:fixture','clave:fixture','categoria:fixture');
  RAISE EXCEPTION 'AD293: lectura sin autoridad admitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $denegada$;
ROLLBACK;
