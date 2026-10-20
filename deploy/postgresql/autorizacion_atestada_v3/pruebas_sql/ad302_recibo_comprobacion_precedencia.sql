\set ON_ERROR_STOP on
-- Tras AD302 (F1-103), en copia desechable y transacción revertida. Fija el
-- cuerpo y la ACL y comprueba la precedencia que corrige: sin paréntesis,
-- 'recibo:'||e#>>'{...}' convierte el literal a jsonb y falla con 22P02. El
-- recorrido con datos («Llamar a la siguiente persona» desde la ficha) se hizo
-- en la copia local. Nunca escribe.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';
DO $guardas$ DECLARE f regprocedure;x record;cuerpo text;
BEGIN
 f:='vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text)'::regprocedure;
 SELECT p.prosecdef,p.provolatile,p.proacl,p.proconfig,pg_get_userbyid(p.proowner) AS dueno,
  pg_get_functiondef(p.oid) AS def
 INTO STRICT x FROM pg_proc p WHERE p.oid=f;
 cuerpo:=x.def;
 IF x.dueno<>'vec_autorizacion_atestada_v3_propietario' OR x.prosecdef IS NOT TRUE OR x.provolatile<>'v'
 OR x.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
   'TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']
 OR encode(sha256(convert_to(cuerpo,'UTF8')),'hex') IS DISTINCT FROM
   '9a27312d4f65b129c20b40a8a6297b7cd88234a15eb50345dc19292743552f63'
 OR x.proacl::text IS DISTINCT FROM
   '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_autorizacion_atestada_v3_registrador_intentos=X/vec_autorizacion_atestada_v3_propietario}'
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
 OR has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD302: ACL o cuerpo divergente'; END IF;
 IF position('''recibo:''||(e#>>''{comprobacion_ofrecimiento,comprobacion_ref}'')' IN cuerpo)=0
 OR position('''recibo:''||e#>>' IN cuerpo)>0
 THEN RAISE EXCEPTION 'AD302: comparación del recibo sin paréntesis'; END IF;
END $guardas$;
-- La misma comparación sobre una entrada sintética con comprobación registrada.
DO $precedencia$ DECLARE e jsonb:=jsonb_build_object('comprobacion_ofrecimiento',jsonb_build_object(
  'comprobacion_ref','comprobacion:'||repeat('a',64),'recibo_ref','recibo:comprobacion:'||repeat('a',64)));
 r boolean;
BEGIN
 r:=e#>>'{comprobacion_ofrecimiento,recibo_ref}' =
   'recibo:'||(e#>>'{comprobacion_ofrecimiento,comprobacion_ref}');
 IF r IS NOT TRUE THEN RAISE EXCEPTION 'AD302: recibo coherente rechazado'; END IF;
 r:=e#>>'{comprobacion_ofrecimiento,recibo_ref}' =
   'recibo:'||(e#>>'{comprobacion_ofrecimiento,comprobacion_ref}')||'x';
 IF r IS NOT FALSE THEN RAISE EXCEPTION 'AD302: recibo ajeno admitido'; END IF;
 BEGIN
  r:=e#>>'{comprobacion_ofrecimiento,recibo_ref}' =
    'recibo:'||e#>>'{comprobacion_ofrecimiento,comprobacion_ref}';
  RAISE EXCEPTION 'AD302: la forma sin paréntesis ya no falla; revisar la prueba';
 EXCEPTION WHEN invalid_text_representation THEN NULL; END;
END $precedencia$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_registrador_intentos;
DO $denegada$ BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(
   '{}'::bytea,'{}'::bytea,'{"accion":"bolsa.origen_ct.consultar","comprobacion_contexto":{}}'::jsonb,
   '{}'::jsonb,'expediente:fixture','clave:fixture','categoria:fixture');
  RAISE EXCEPTION 'AD302: lectura sin autoridad admitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $denegada$;
SELECT 'ad302_recibo_comprobacion_precedencia: 0 fallos' AS resultado;
ROLLBACK;
