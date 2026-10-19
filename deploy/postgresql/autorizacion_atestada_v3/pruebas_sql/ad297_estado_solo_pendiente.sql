\set ON_ERROR_STOP on
-- Tras AD297 (F1-10) en copia desechable. Cuerpo, permisos y las dos
-- condiciones nuevas; no finge V3. El efecto con datos lo cubre b140_recotejo_reserva.sql.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
DO $guardas$ DECLARE f regprocedure;x record;cuerpo text;
BEGIN
 f:='vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text)'::regprocedure;
 SELECT p.prosecdef,p.provolatile,p.proacl,p.proconfig,pg_get_functiondef(p.oid) AS def
 INTO STRICT x FROM pg_proc p WHERE p.oid=f;
 cuerpo:=x.def;
 IF x.prosecdef IS NOT TRUE OR x.provolatile<>'v'
 OR x.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
   'TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']
 OR encode(sha256(convert_to(cuerpo,'UTF8')),'hex') IS DISTINCT FROM
   '662788f02fbaffe8c622da2216dac63354b49dc5db1b96002c0c4f20d003c9f5'
 OR x.proacl::text IS DISTINCT FROM
   '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_autorizacion_atestada_v3_registrador_intentos=X/vec_autorizacion_atestada_v3_propietario}'
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
 OR has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD297: ACL o cuerpo divergente'; END IF;
 -- La lectura actual exige estado solo en la persona pendiente; una reserva
 -- antigua que lo guardó en todas las entradas se sigue recuperando igual.
 IF position('WHERE ((v_estado_actual AND e->>''razon''=''pendiente_comprobacion_rrhh'') OR e ? ''comprobacion_ofrecimiento'') AND (' IN cuerpo)=0
 OR position('CASE WHEN (v_estado_actual AND x.valor->>''razon''=''pendiente_comprobacion_rrhh'') OR (v_estado AND reserva IS NOT NULL AND x.valor ? ''comprobacion_ofrecimiento'') THEN' IN cuerpo)=0
 OR position('WHERE (v_estado_actual OR e ? ''comprobacion_ofrecimiento'') AND (' IN cuerpo)>0
 THEN RAISE EXCEPTION 'AD297: condiciones de estado ausentes'; END IF;
END $guardas$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_registrador_intentos;
DO $denegada$ BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(
   '{}'::bytea,'{}'::bytea,'{"accion":"bolsa.origen_ct.consultar","comprobacion_contexto":{}}'::jsonb,
   '{}'::jsonb,'expediente:fixture','clave:fixture','categoria:fixture');
  RAISE EXCEPTION 'AD297: lectura sin autoridad admitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $denegada$;
SELECT 'ad297_estado_solo_pendiente: 0 fallos' AS resultado;
ROLLBACK;
