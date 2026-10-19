\set ON_ERROR_STOP on
-- CT241 / F1-19, segundo corte (guardas). Un nombramiento derivado (CT234)
-- tiene ya su plaza adjudicada: no puede entrar en otra selección de Bolsa ni
-- vincularse a otro llamamiento. Se añade la misma guarda (23505) a las tres
-- entradas reales: reserva singular O6, reserva de oferta (postimagen CT240) y
-- vínculo de emisión (CT201); en las dos últimas, antes de consumir la
-- autorización. Cuerpos instalados en main sin otro cambio; propietario y
-- ACL se conservan. La reserva O6 recibe además pg_temp al final de su
-- search_path, como exige la puerta de funciones SECURITY DEFINER.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000241',0));
DO $pre$
DECLARE f record; p record;
BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_contratacion_temporal_propietario'
    OR to_regclass('vec_contratacion_temporal.nombramiento_derivado_v1') IS NULL
    OR NOT EXISTS(SELECT 1 FROM pg_attribute a
       WHERE a.attrelid='vec_contratacion_temporal.expediente_alta'::regclass
         AND a.attname='derivacion_ref' AND NOT a.attisdropped) THEN
  RAISE EXCEPTION 'CT241: dependencias incompatibles' USING ERRCODE='55000'; END IF;
 FOR f IN SELECT * FROM (VALUES
 ('vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(uuid,text,text)','5edef187208e21d903410b156c124c7b70eca8164dec39c813aa901e455043e3',ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s','idle_in_transaction_session_timeout=20s']),
 ('vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','45f84d4328a28c909966560149e08fd51ad13c73e737657a889e07dc3d3dd735',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']),
 ('vec_contratacion_temporal.registrar_vinculo_emision_bolsa_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','6911e480b80031a664a4c89368b64e763cfaf2d5062fa5ce5c69ca6e7d4eae4b',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s'])
 ) AS x(firma,huella,config) LOOP
  SELECT pg_get_userbyid(q.proowner) AS dueno,q.prosecdef,q.proacl::text AS acl,q.proconfig,
   encode(sha256(convert_to(pg_get_functiondef(q.oid),'UTF8')),'hex') AS huella
  INTO p FROM pg_proc q WHERE q.oid=to_regprocedure(f.firma);
  IF NOT FOUND OR p.dueno<>'vec_contratacion_temporal_propietario' OR p.prosecdef IS NOT TRUE
     OR p.acl IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
     OR p.proconfig IS DISTINCT FROM f.config OR p.huella IS DISTINCT FROM f.huella THEN
   RAISE EXCEPTION 'CT241: preimagen incompatible: %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

-- Reserva singular O6.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(p_clave uuid, p_huella text, p_solicitud_texto text)
 RETURNS TABLE(situacion text, solicitud_json text, reserva_ref text, efecto text, recibo_json text, artefacto_json text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
 SET idle_in_transaction_session_timeout TO '20s'
AS $function$
DECLARE
    v_insertadas integer := 0;
    v_ahora timestamptz;
    p_solicitud jsonb;
    v_ejecucion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
BEGIN
    IF NOT pg_catalog.pg_has_role(
        session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
    ) OR p_clave IS NULL OR p_huella IS NULL
       OR p_huella !~ '^[0-9a-f]{64}$' OR p_solicitud_texto IS NULL
       OR pg_catalog.octet_length(p_solicitud_texto) > 1048576 THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'reserva O6 invalida';
    END IF;
    p_solicitud := vec_contratacion_temporal.solicitud_desde_texto_seleccion_llamamiento_o6_v1(
        p_solicitud_texto
    );
    IF p_solicitud IS NULL
       OR p_solicitud->>'clave_idempotencia' IS DISTINCT FROM p_clave::text
       OR p_solicitud->>'huella_semantica' IS DISTINCT FROM p_huella
       OR vec_contratacion_temporal.huella_solicitud_seleccion_llamamiento_o6_v1(
           p_solicitud
       ) IS DISTINCT FROM p_huella THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'reserva O6 invalida';
    END IF;
    -- CT241 / F1-19: un nombramiento derivado no entra en otra selección.
    IF EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.expediente_alta alta
         WHERE alta.expediente_ref = p_solicitud->>'expediente_ref'
           AND alta.derivacion_ref IS NOT NULL
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'CT241: expediente derivado sin selección nueva';
    END IF;
    INSERT INTO vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 (
        clave_idempotencia, huella_semantica, solicitud_json
    ) VALUES (p_clave, p_huella, p_solicitud)
    ON CONFLICT (clave_idempotencia) DO NOTHING;
    GET DIAGNOSTICS v_insertadas = ROW_COUNT;
    SELECT ejecucion.* INTO STRICT v_ejecucion
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 ejecucion
     WHERE ejecucion.clave_idempotencia = p_clave
     FOR UPDATE;
    v_ahora := pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp());
    IF v_insertadas = 1 THEN
        RETURN QUERY SELECT 'propietaria', v_ejecucion.solicitud_json::text,
            v_ejecucion.reserva_ref, '', '', '';
    ELSIF v_ejecucion.huella_semantica IS DISTINCT FROM p_huella
       OR v_ejecucion.solicitud_json IS DISTINCT FROM p_solicitud THEN
        RETURN QUERY SELECT 'colision', p_solicitud::text, '', '', '', '';
    ELSIF v_ejecucion.situacion = 'propietaria' AND v_ejecucion.lease_hasta > v_ahora THEN
        RETURN QUERY SELECT 'ocupada', v_ejecucion.solicitud_json::text, '',
            COALESCE(v_ejecucion.efecto, ''), '', '';
    ELSIF v_ejecucion.situacion = 'propietaria' AND
          (v_ejecucion.ventana_orden_abierta OR v_ejecucion.ventana_llamamiento_abierta) THEN
        UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
           SET situacion = 'indeterminada',
               efecto = CASE WHEN ventana_llamamiento_abierta
                   THEN 'solicitar_llamamiento' ELSE 'preparar_orden' END,
               actualizada_en = v_ahora
         WHERE clave_idempotencia = p_clave RETURNING * INTO v_ejecucion;
        RETURN QUERY SELECT 'indeterminada', v_ejecucion.solicitud_json::text, '',
            v_ejecucion.efecto, '', '';
    ELSIF v_ejecucion.situacion = 'propietaria' THEN
        UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
           SET reserva_ref = vec_contratacion_temporal.nuevo_token_fencing_seleccion_llamamiento_o6_v2(),
               fencing_version = fencing_version + 1,
               lease_hasta = v_ahora + interval '30 seconds', actualizada_en = v_ahora
         WHERE clave_idempotencia = p_clave RETURNING * INTO v_ejecucion;
        RETURN QUERY SELECT 'propietaria', v_ejecucion.solicitud_json::text,
            v_ejecucion.reserva_ref, '', '', '';
    ELSE
        RETURN QUERY SELECT v_ejecucion.situacion, v_ejecucion.solicitud_json::text, '',
            COALESCE(v_ejecucion.efecto, ''),
            COALESCE(v_ejecucion.recibo_json::text, ''),
            COALESCE(v_ejecucion.artefacto_canonico, '');
    END IF;
END
$function$;

-- Reserva de oferta (postimagen CT240).
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
 -- CT241 / F1-19: un nombramiento derivado no entra en otra selección.
 -- Antes de consumir la autorización: el hijo no tiene selección propia.
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta alta
   WHERE alta.expediente_ref=p_intencion->>'expediente_ref' AND alta.derivacion_ref IS NOT NULL) THEN
  RAISE EXCEPTION 'CT241: expediente derivado sin selección nueva' USING ERRCODE='23505'; END IF;
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

-- Vínculo de emisión de Bolsa (CT201).
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.registrar_vinculo_emision_bolsa_ct_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE s jsonb; d jsonb; c record; actual_ct record; amb record; anterior record;
 clave uuid; version_esperada numeric; material_h text; contexto_h text;
 numero text; recibo text; evento text; instante timestamptz(6); salida jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'CT201: sesión de vínculo denegada' USING ERRCODE='42501';
 END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 128 AND 4096 THEN
  RAISE EXCEPTION 'CT201: material inválido' USING ERRCODE='22023';
 END IF;
 BEGIN
  s:=p_material::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  clave:=(s->>'clave_idempotencia')::uuid;
  version_esperada:=(s->>'version_esperada')::numeric;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'CT201: material ilegible' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(s)<>'object' OR (SELECT count(*) FROM json_each(p_material::json))<>8
    OR (SELECT count(*) FROM jsonb_object_keys(s))<>8
    OR (SELECT array_agg(key ORDER BY ord) FROM json_each(p_material::json) WITH ORDINALITY AS j(key,val,ord))
      IS DISTINCT FROM ARRAY['esquema','organizacion_ref','expediente_ref','version_esperada',
        'bolsa_ref','llamamiento_ref','recibo_emision_ref','clave_idempotencia']
    OR s->>'esquema'<>'vec.ct.vinculo-emision-bolsa.v1'
    OR version_esperada NOT BETWEEN 1 AND 9007199254740991::numeric
    OR version_esperada<>trunc(version_esperada)
    OR s->>'organizacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'expediente_ref' !~ '^expediente:[A-Za-z0-9._:/#-]{2,149}$'
    OR s->>'bolsa_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'llamamiento_ref' !~ '^llamamiento:[0-9a-f]{64}$'
    OR s->>'recibo_emision_ref' !~ '^recibo:llamamiento:[0-9a-f]{64}$'
    OR s->>'clave_idempotencia' IS DISTINCT FROM clave::text THEN
  RAISE EXCEPTION 'CT201: contrato de vínculo inválido' USING ERRCODE='22023';
 END IF;
 -- CT241 / F1-19: un nombramiento derivado no se vincula a otro llamamiento.
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta alta
   WHERE alta.expediente_ref=s->>'expediente_ref' AND alta.derivacion_ref IS NOT NULL) THEN
  RAISE EXCEPTION 'CT241: expediente derivado sin selección nueva' USING ERRCODE='23505';
 END IF;
	material_h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
	SELECT * INTO amb FROM vec_contratacion_temporal.leer_ambitos_vinculo_emision_bolsa_ct_v1(
	 s->>'organizacion_ref',s->>'expediente_ref',version_esperada,s->>'bolsa_ref',
	 s->>'llamamiento_ref',s->>'recibo_emision_ref',s->>'clave_idempotencia');
	IF NOT FOUND THEN
	 RAISE EXCEPTION 'CT201: ámbitos del expediente divergentes' USING ERRCODE='23505';
	END IF;
	contexto_h:=encode(sha256(convert_to(
	   '{"ambitos":{"categoria_ref":'||to_json(amb.categoria_ref)::text||
	   ',"centro_ref":'||to_json(amb.centro_ref)::text||
	   ',"organizacion_ref":'||(s->'organizacion_ref')::text||
   '},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.bolsa.vincular'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'vinculo_bolsa_expediente'
    OR d->>'finalidad' IS DISTINCT FROM 'tramitacion_expediente_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM s->>'expediente_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'CT201: decisión no ligada al material' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT c FROM vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_emision_bolsa_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF c.efecto_ref IS DISTINCT FROM s->>'expediente_ref' OR c.huella_efecto_sha256 IS DISTINCT FROM contexto_h
    OR c.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'CT201: consumo de efecto divergente' USING ERRCODE='42501';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:vinculo_bolsa:'||(s->>'expediente_ref'),0));
 SELECT * INTO anterior FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
  WHERE x.organizacion_ref=s->>'organizacion_ref' AND x.clave_idempotencia=clave;
 IF FOUND THEN
  IF anterior.material_sha256 IS DISTINCT FROM material_h
     OR anterior.actor_ref IS DISTINCT FROM d->>'principal_id'
     OR anterior.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref' THEN
   RAISE EXCEPTION 'CT201: clave usada con otro contenido' USING ERRCODE='23505';
  END IF;
  RETURN anterior.recibo_json||jsonb_build_object('reutilizado',true);
 END IF;
 SELECT e.organizacion_ref,e.numero_visible,actual.version,v.agregado_json
  INTO actual_ct FROM vec_contratacion_temporal.expediente_alta e
  JOIN vec_contratacion_temporal.expediente_integral_actual actual ON actual.expediente_ref=e.expediente_ref
  JOIN vec_contratacion_temporal.expediente_version_integral v
    ON v.expediente_ref=actual.expediente_ref AND v.version=actual.version
  WHERE e.expediente_ref=s->>'expediente_ref' AND e.organizacion_ref=s->>'organizacion_ref'
  FOR SHARE OF actual;
 IF NOT FOUND OR actual_ct.version IS DISTINCT FROM version_esperada
    OR actual_ct.agregado_json#>>'{via_cobertura,via_clave}' IS DISTINCT FROM 'bolsa_vigente'
    OR actual_ct.agregado_json#>>'{via_cobertura,decision_gobernada,via_elegida}' IS DISTINCT FROM 'bolsa_vigente'
    OR actual_ct.agregado_json#>'{via_cobertura,bolsa_ref}' IS NOT NULL
    OR actual_ct.agregado_json#>>'{solicitud,centro_ref}' IS DISTINCT FROM amb.centro_ref
    OR actual_ct.agregado_json#>>'{analisis,categoria_ref}' IS DISTINCT FROM amb.categoria_ref THEN
  RAISE EXCEPTION 'CT201: versión o bolsa de cobertura divergente' USING ERRCODE='23505';
 END IF;
 numero:=vec_contratacion_temporal.numero_visible_vigente_v1(s->>'expediente_ref',actual_ct.numero_visible);
 IF vec_bolsa_llamamientos.verificar_emision_ct_v1(
    s->>'bolsa_ref',s->>'llamamiento_ref',s->>'recibo_emision_ref',numero,
    actual_ct.agregado_json#>>'{analisis,categoria_ref}') IS NOT TRUE THEN
  RAISE EXCEPTION 'CT201: emisión real ajena al expediente' USING ERRCODE='42501';
 END IF;
 instante:=date_trunc('microseconds',clock_timestamp());
 recibo:='recibo:ct:bolsa:'||gen_random_uuid()::text;
 evento:='evento:ct:bolsa:'||gen_random_uuid()::text;
 salida:=jsonb_build_object('expediente_ref',s->>'expediente_ref',
  'bolsa_ref',s->>'bolsa_ref','llamamiento_ref',s->>'llamamiento_ref',
  'recibo_emision_ref',s->>'recibo_emision_ref','recibo_vinculo_ref',recibo,
  'vinculado_en',instante,'auditoria_ref',c.auditoria_ref,'evento_ref',evento,'reutilizado',false);
 INSERT INTO vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1(
  organizacion_ref,expediente_ref,bolsa_ref,llamamiento_ref,recibo_emision_ref,
  version_expediente_esperada,clave_idempotencia,material_sha256,actor_ref,perfil_ref,
  decision_ref,auditoria_ref,consumo_sha256,recibo_ref,vinculado_en,recibo_json)
 VALUES(s->>'organizacion_ref',s->>'expediente_ref',s->>'bolsa_ref',s->>'llamamiento_ref',
  s->>'recibo_emision_ref',version_esperada,clave,material_h,d->>'principal_id',d->>'perfil_activo_ref',
  c.decision_ref,c.auditoria_ref,c.consumo_huella_sha256,recibo,instante,salida);
 INSERT INTO vec_contratacion_temporal.outbox_vinculo_emision_bolsa_ct_v1(
  evento_ref,expediente_ref,llamamiento_ref,recibo_ref,tipo,payload,payload_sha256,registrada_en)
 VALUES(evento,s->>'expediente_ref',s->>'llamamiento_ref',recibo,
  'contratacion_temporal.bolsa.emision_vinculada.v1',salida,
  encode(sha256(convert_to(salida::text,'UTF8')),'hex'),instante);
 RETURN salida;
END $function$;

DO $post$
DECLARE f record; p record;
BEGIN
 FOR f IN SELECT * FROM (VALUES
 ('vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(uuid,text,text)','af2d9ff4aa9a26c12c2445833473b3427b9f6be4731c2c7f12121c5eaa249529',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s','idle_in_transaction_session_timeout=20s']),
 ('vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','6a8e5f1feff9051e960a8db3866669569c5c551eca009d7f6889c87f88e383e8',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']),
 ('vec_contratacion_temporal.registrar_vinculo_emision_bolsa_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','fd7421e9d5dba11dda82368873a8ea41a30fc5002f921e0f226b649ba13ac344',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s'])
 ) AS x(firma,huella,config) LOOP
  SELECT pg_get_userbyid(q.proowner) AS dueno,q.prosecdef,q.proacl::text AS acl,q.proconfig,
   encode(sha256(convert_to(pg_get_functiondef(q.oid),'UTF8')),'hex') AS huella
  INTO STRICT p FROM pg_proc q WHERE q.oid=f.firma::regprocedure;
  IF p.dueno<>'vec_contratacion_temporal_propietario' OR p.prosecdef IS NOT TRUE
     OR p.acl IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
     OR p.proconfig IS DISTINCT FROM f.config OR p.huella IS DISTINCT FROM f.huella THEN
   RAISE EXCEPTION 'CT241: postimagen incompatible: %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $post$;
COMMIT;
