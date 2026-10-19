\set ON_ERROR_STOP on
-- CT240 / F1-10: la reserva de oferta recoteja Bolsa antes de bloquear CT.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000240',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_contratacion_temporal_propietario'
 OR to_regprocedure('vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(jsonb)') IS NULL
 OR has_function_privilege('vec_contratacion_temporal_propietario',
  'vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(jsonb)','EXECUTE') IS NOT TRUE
 OR encode(sha256(convert_to(pg_get_functiondef(
  'vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
 IS DISTINCT FROM 'f83d3f430f67ef60f886836cc7ac170984a3ff0e44c83eee59300d8c0f86c6d9'
 THEN RAISE EXCEPTION 'CT240: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
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
 -- Misma fila que bloquea CT122 antes de cancelar. Orden fijo desde CT240:
 -- consumo de la autorización, cerrojo del expediente, cotejo con Bolsa y
 -- FOR UPDATE de la cabecera.
 SELECT * INTO STRICT v_auth FROM vec_contratacion_temporal.consumir_etapa_seleccion_oferta_ct220(
  'reservar',p_intencion,p_comando,NULL,p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 x
  WHERE x.clave_idempotencia=(p_intencion->>'clave_idempotencia')::uuid)
  AND p_comando#>'{origen_ct,lectura,criterios_ofrecimiento}' IS NOT NULL THEN
  -- El cotejo toma el expediente compartido antes del FOR UPDATE: dos reservas
  -- del mismo expediente se ordenan aquí para no bloquearse entre sí.
  PERFORM pg_advisory_xact_lock(hashtextextended(
   'vec_contratacion_temporal:reserva-oferta:'||(p_intencion->>'expediente_ref'),0));
  -- La oferta guarda los criterios dentro de su lectura; Bolsa los coteja en raíz.
  PERFORM vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(
   (p_comando->'origen_ct')||jsonb_build_object('criterios_ofrecimiento',
    p_comando#>'{origen_ct,lectura,criterios_ofrecimiento}'));
 END IF;
 SELECT a.version,v.agregado_json INTO v_cabecera
 FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_alta alta ON alta.expediente_ref=a.expediente_ref
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE a.expediente_ref=p_intencion->>'expediente_ref'
   AND alta.organizacion_ref=p_intencion->>'organizacion_ref'
 FOR UPDATE OF a;
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

DO $post$ DECLARE p record; BEGIN
 SELECT pg_get_userbyid(q.proowner) AS dueno,q.prosecdef,q.proacl::text AS acl,q.proconfig,
  encode(sha256(convert_to(pg_get_functiondef(q.oid),'UTF8')),'hex') AS huella
 INTO STRICT p FROM pg_proc q WHERE q.oid=
  'vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF p.dueno<>'vec_contratacion_temporal_propietario' OR p.prosecdef IS NOT TRUE
 OR p.acl IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']
 OR p.huella IS DISTINCT FROM '27a91dbe9d3c194edd39f7accad0fbf8b891968fbe8fbccef2605814ef0ff534'
 THEN RAISE EXCEPTION 'CT240: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
