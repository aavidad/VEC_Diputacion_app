\set ON_ERROR_STOP on
-- F1-11: modalidad y causa gobernadas de la oferta; sin reescribir historia.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000235',0));
DO $pre$ BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR to_regprocedure('vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint)') IS NULL
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint)'::regprocedure AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='777f09f2514dad4ac9a552de4a992d15ab2b6f1a8dad94cfd8401e1d5498fe23' AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef IS TRUE AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=5s']::text[] AND p.proacl::text='{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_contratacion_temporal_propietario,vec_bolsa_llamamientos_propietario=X/vec_contratacion_temporal_propietario}')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(jsonb,jsonb)'::regprocedure AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='e6c2b110a6536232368a200df631c04af3b67ab63d6a6260fdcd7480c0222132' AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef IS FALSE AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[] AND p.proacl::text='{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='f83d3f430f67ef60f886836cc7ac170984a3ff0e44c83eee59300d8c0f86c6d9' AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef IS TRUE AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[] AND p.proacl::text='{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}')
 THEN RAISE EXCEPTION 'CT235: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
-- Definición final obtenida de la copia ensayada; la preimagen anterior limita su aplicación.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(c jsonb, i jsonb)
 RETURNS boolean
 LANGUAGE sql
 IMMUTABLE
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
 SELECT coalesce(
  jsonb_typeof(c)='object' AND octet_length(c::text)<=1048576
  AND c ?& ARRAY['organizacion_ref','expediente_ref','version_expediente',
   'actor_ref','bolsa_ref','necesidad_ref','numero_plazas','numero_elegibles',
   'datos','origen_ct','clave_bolsa','inicio_plazo']
  AND c-ARRAY['organizacion_ref','expediente_ref','version_expediente',
   'actor_ref','bolsa_ref','necesidad_ref','numero_plazas','numero_elegibles',
   'datos','origen_ct','clave_bolsa','inicio_plazo']='{}'::jsonb
  AND c->>'organizacion_ref'=i->>'organizacion_ref'
  AND c->>'expediente_ref'=i->>'expediente_ref'
  AND c->'version_expediente'=i->'version_expediente'
  AND c->>'actor_ref'=i->>'actor_ref'
  AND jsonb_typeof(c->'version_expediente')='number'
  AND jsonb_typeof(c->'numero_plazas')='number'
  AND jsonb_typeof(c->'numero_elegibles')='number'
  AND c->>'numero_plazas' ~ '^([2-9]|[1-9][0-9]|100)$'
  AND c->>'numero_elegibles' ~ '^[1-9][0-9]{0,6}$'
  AND (c->>'numero_elegibles')::integer<=1000000
  AND (c->>'numero_elegibles')::integer >= (c->>'numero_plazas')::integer
  AND c->>'bolsa_ref' ~ '^bolsa:[A-Za-z0-9:_-]{1,250}$'
  AND c->>'necesidad_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  AND c->>'inicio_plazo'='publicacion'
  AND (c->>'clave_bolsa')=('ct-oferta:'||encode(sha256(convert_to(
   (i->>'organizacion_ref')||chr(31)||(i->>'expediente_ref')||chr(31)||
   (i->>'version_expediente')||chr(31)||(i->>'clave_idempotencia'),'UTF8')),'hex'))
  AND jsonb_typeof(c->'datos')='object'
  AND c->'datos' ?& ARRAY['categoria','centro','fecha_inicio','descripcion']
  AND (c->'datos')-ARRAY['categoria','centro','fecha_inicio','fecha_fin','descripcion','modalidad_clave','causa_clave']='{}'::jsonb
  AND ((NOT (c->'datos' ? 'modalidad_clave') AND NOT (c->'datos' ? 'causa_clave'))
       OR ((c->'datos' ?& ARRAY['modalidad_clave','causa_clave'])
         AND jsonb_typeof(c#>'{datos,modalidad_clave}')='string'
         AND jsonb_typeof(c#>'{datos,causa_clave}')='string'
         AND c#>>'{datos,modalidad_clave}' ~ '^[a-z][a-z0-9._-]{1,79}$'
         AND c#>>'{datos,causa_clave}' ~ '^[a-z][a-z0-9._-]{1,79}$'))
  AND NOT EXISTS(SELECT 1 FROM jsonb_each(c->'datos') d WHERE jsonb_typeof(d.value)<>'string')
  AND jsonb_typeof(c->'origen_ct')='object'
  AND c#>>'{origen_ct,organizacion_ref}'=i->>'organizacion_ref'
  AND c#>>'{origen_ct,expediente_ref}'=i->>'expediente_ref'
  AND c#>'{origen_ct,version_expediente}'=i->'version_expediente'
  AND c#>>'{origen_ct,clave_ct}'=i->>'clave_idempotencia'
  AND c#>>'{origen_ct,bolsa_ref}'=c->>'bolsa_ref'
  AND c#>>'{origen_ct,necesidad_ref}'=c->>'necesidad_ref'
  AND c#>>'{origen_ct,categoria_ref}'=i->>'categoria_ref'
  AND jsonb_typeof(c#>'{origen_ct,lectura}')='object'
  AND jsonb_typeof(c#>'{origen_ct,lectura,entradas}')='array'
  AND c#>>'{origen_ct,lectura,corte}' IS NOT NULL,false)
$function$;

-- Definición final obtenida de la copia ensayada; la preimagen anterior limita su aplicación.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.reservar_seleccion_oferta_ct220(p_intencion jsonb, p_comando jsonb, p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE e vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
 v_cabecera record;v_agregado jsonb;v_ahora timestamptz(6);v_auth record;v_insertadas integer;
BEGIN
 IF vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(p_comando,p_intencion) IS NOT TRUE THEN
  RAISE EXCEPTION 'CT220: comando de oferta inválido' USING ERRCODE='22023'; END IF;
 -- Misma fila que bloquea CT122 antes de cancelar. El corte 3 añadirá la
 -- guarda de reservas pendientes después de adquirir exactamente este lock.
 SELECT a.version,v.agregado_json INTO v_cabecera
 FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_alta alta ON alta.expediente_ref=a.expediente_ref
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE a.expediente_ref=p_intencion->>'expediente_ref'
   AND alta.organizacion_ref=p_intencion->>'organizacion_ref'
 FOR UPDATE OF a;
 SELECT * INTO STRICT v_auth FROM vec_contratacion_temporal.consumir_etapa_seleccion_oferta_ct220(
  'reservar',p_intencion,p_comando,NULL,p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_cabecera.version IS NULL THEN
  RAISE EXCEPTION 'CT220: expediente inexistente' USING ERRCODE='23503'; END IF;
 v_agregado:=v_cabecera.agregado_json;
 IF v_cabecera.version IS DISTINCT FROM (p_intencion->>'version_expediente')::numeric
    OR v_agregado->>'fase_actual' IS DISTINCT FROM 'fiscalizacion'
    OR v_agregado->>'estado_actual' IS DISTINCT FROM 'en_curso'
    OR v_agregado#>>'{via_cobertura,via_clave}' IS DISTINCT FROM 'bolsa_vigente'
    OR v_agregado#>>'{via_cobertura,decision_gobernada,via_elegida}' IS DISTINCT FROM 'bolsa_vigente'
    OR v_agregado#>>'{solicitud,numero_personas}' IS DISTINCT FROM p_comando->>'numero_plazas'
    OR v_agregado#>>'{solicitud,centro_ref}' IS DISTINCT FROM p_intencion->>'centro_ref'
    OR v_agregado#>>'{analisis,categoria_ref}' IS DISTINCT FROM p_intencion->>'categoria_ref'
    OR v_agregado#>>'{asignacion,unidad_ref}' IS DISTINCT FROM p_comando#>>'{origen_ct,unidad_ref}'
 THEN RAISE EXCEPTION 'CT220: expediente ya no seleccionable' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
   WHERE x.expediente_ref=p_intencion->>'expediente_ref') THEN
  RAISE EXCEPTION 'CT220: expediente ya vinculado a llamamiento' USING ERRCODE='23505'; END IF;
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 x
   WHERE x.tipo_seleccion='llamamiento'
     AND x.solicitud_json->>'expediente_ref'=p_intencion->>'expediente_ref') THEN
  RAISE EXCEPTION 'CT220: expediente ya tiene selección singular' USING ERRCODE='23505'; END IF;
 -- Las guardas de CT220 valen también para el reintento. Solo la reserva nueva
 -- exige el par y lo coteja; la anterior a F1-11 se recupera sin él.
 IF NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 previa
   WHERE previa.clave_idempotencia=(p_intencion->>'clave_idempotencia')::uuid) THEN
 IF p_comando->'datos' ?& ARRAY['modalidad_clave','causa_clave'] IS NOT TRUE THEN
  RAISE EXCEPTION 'CT235: la nueva reserva exige modalidad y causa' USING ERRCODE='22023'; END IF;
 PERFORM 1 FROM vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(
   p_intencion->>'expediente_ref',p_intencion->>'categoria_ref',
   p_intencion->>'organizacion_ref',(p_intencion->>'version_expediente')::bigint) x
  WHERE x.modalidad_clave=p_comando#>>'{datos,modalidad_clave}'
    AND x.causa_clave=p_comando#>>'{datos,causa_clave}'
    AND x.version_expediente=(p_intencion->>'version_expediente')::bigint;
 IF NOT FOUND THEN RAISE EXCEPTION 'CT235: modalidad o causa distinta del análisis vigente' USING ERRCODE='42501'; END IF;
 END IF;
 v_ahora:=date_trunc('microseconds',clock_timestamp());
 INSERT INTO vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6(
  clave_idempotencia,huella_semantica,solicitud_json,reserva_ref,tipo_seleccion,comando_oferta_json)
 VALUES((p_intencion->>'clave_idempotencia')::uuid,p_intencion->>'huella_intencion',p_intencion,
  'reserva:seleccion-oferta:v1:'||encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'),
  'oferta',p_comando)
 ON CONFLICT (clave_idempotencia) DO NOTHING;
 GET DIAGNOSTICS v_insertadas=ROW_COUNT;
 SELECT * INTO STRICT e FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 WHERE clave_idempotencia=(p_intencion->>'clave_idempotencia')::uuid FOR UPDATE;
 IF e.tipo_seleccion<>'oferta' OR e.solicitud_json IS DISTINCT FROM p_intencion
    OR e.huella_semantica IS DISTINCT FROM p_intencion->>'huella_intencion'
    OR e.comando_oferta_json IS DISTINCT FROM p_comando THEN
  RETURN jsonb_build_object('situacion','colision'); END IF;
 IF v_insertadas=1 THEN RETURN vec_contratacion_temporal.estado_seleccion_oferta_ct220(e); END IF;
 IF e.situacion='propietaria' AND e.lease_hasta>v_ahora THEN
  RETURN jsonb_build_object('situacion','ocupada'); END IF;
 IF e.situacion='propietaria' AND e.ventana_orden_abierta THEN
  UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
  SET situacion='indeterminada',actualizada_en=v_ahora
  WHERE clave_idempotencia=e.clave_idempotencia RETURNING * INTO e;
 ELSIF e.situacion='propietaria' THEN
  UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
  SET reserva_ref='reserva:seleccion-oferta:v1:'||encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'),
      fencing_version=fencing_version+1,
      actualizada_en=v_ahora,lease_hasta=v_ahora+interval '30 seconds'
  WHERE clave_idempotencia=e.clave_idempotencia RETURNING * INTO e;
 END IF;
 RETURN vec_contratacion_temporal.estado_seleccion_oferta_ct220(e);
END $function$;
DO $post_funciones$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(jsonb,jsonb)'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='0d7ad798497eaa6661e92873632edb34d6b24d633e88b6501b45126836ce059c'
  AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef IS FALSE
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[] AND p.proacl::text='{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}')
 THEN RAISE EXCEPTION 'F1-11: postimagen incompatible: comando_seleccion_oferta_valido_ct220' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='b190f4d197cba3a33556e58e414b370c547aa4b2070524efaca3e67d1e63e8e7'
  AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef IS TRUE
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[] AND p.proacl::text='{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}')
 THEN RAISE EXCEPTION 'F1-11: postimagen incompatible: reservar_seleccion_oferta_ct220' USING ERRCODE='55000'; END IF;
END $post_funciones$;
COMMIT;
