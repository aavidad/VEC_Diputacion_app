\set ON_ERROR_STOP on
-- RRHH 02/10/2026: la aceptación telemática precede a la adjudicación.
-- B71 (activación con inicio configurable) precede a esta migración.
-- La política de cada oferta inmoviliza esta decisión. Las anteriores
-- conservan su segundo plazo y sus actos; no se modifica ninguna fila.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000070',0));
DO $pre$
BEGIN
 IF to_regprocedure('vec_bolsa_llamamientos.proyectar_oferta_v2(text,timestamp with time zone)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.publicar_oferta_v4(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_ofertas_version') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.acto_plaza_oferta') IS NULL THEN
  RAISE EXCEPTION 'B70: clave=dependencias_B58 esperado=presentes actual=ausentes' USING ERRCODE='55000';
 END IF;
 IF strpos(pg_get_functiondef('vec_bolsa_llamamientos.proyectar_oferta_v2(text,timestamp with time zone)'::regprocedure),'confirmacion_adjudicacion')>0 THEN
  RAISE EXCEPTION 'B70: clave=adjudicacion_previa esperado=no_instalada actual=instalada' USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta text, p_corte timestamptz)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE
 base jsonb; o record; v_n integer; v_config jsonb; v_gestion boolean;
 v_estados text[]:='{}'; v_ultimos jsonb[]:='{}'; ultimo record; v_estado text;
 v_candidatos text[]:='{}'; v_ordenes bigint[]:='{}'; v_siguiente integer:=1;
 v_pendiente boolean; v_propuesta jsonb; v_plazas jsonb:='[]'::jsonb; v_historial jsonb;
 v_cubiertas integer:=0; v_directas integer:=0; v_actos integer:=0; v_primera boolean:=true;
 v_puede boolean; i integer; v_confirmacion text;
BEGIN
 base:=vec_bolsa_llamamientos.proyectar_oferta_v1(p_oferta,p_corte);
 IF base IS NULL THEN RETURN NULL; END IF;
 SELECT * INTO STRICT o FROM vec_bolsa_llamamientos.oferta_publicada WHERE oferta_ref=p_oferta;
 SELECT coalesce((SELECT numero_plazas FROM vec_bolsa_llamamientos.plazas_oferta WHERE oferta_ref=p_oferta),1) INTO v_n;
 SELECT pv.politica->'plazas', pv.politica#>>'{adjudicacion,confirmacion}' INTO v_config,v_confirmacion FROM vec_bolsa_llamamientos.politica_ofertas_version pv
  WHERE pv.bolsa_ref=o.bolsa_ref AND pv.version::text=o.plazo->>'politica_version';
 v_gestion:=jsonb_typeof(v_config)='object';
 IF NOT v_gestion THEN v_config:=NULL; END IF;
 -- Resolución única anterior (B28): se muestra como su única plaza, sin actos.
 IF jsonb_typeof(base->'resolucion')='object' THEN
  RETURN base || jsonb_build_object('propuesta',NULL,'numero_plazas',1,'politica_plazas',v_config,'confirmacion_adjudicacion',v_confirmacion,
   'plazas',jsonb_build_array(jsonb_build_object('numero_de_plaza',1,
    'estado',CASE WHEN base#>>'{resolucion,tipo}'='adjudicada' THEN 'cubierta' ELSE 'llamamiento_directo' END,
    'secuencia',0,'participacion_ref',base#>'{resolucion,participacion_ref}','orden_vigente',base#>'{resolucion,orden_vigente}',
    'responder_antes_de',NULL,'puede_sin_respuesta',false,'propuesta',NULL,'historial','[]'::jsonb)));
 END IF;
 FOR i IN 1..v_n LOOP
  SELECT a.* INTO ultimo FROM vec_bolsa_llamamientos.acto_plaza_oferta a
   WHERE a.oferta_ref=p_oferta AND a.numero_de_plaza=i AND a.registrado_en<=p_corte
   ORDER BY a.secuencia DESC LIMIT 1;
  IF ultimo.tipo IS NULL THEN v_estado:='vacante';
  ELSIF ultimo.tipo='adjudicada' THEN v_estado:=CASE WHEN v_gestion AND v_confirmacion IS DISTINCT FROM 'aceptacion_previa' THEN 'pendiente_respuesta' ELSE 'cubierta' END;
  ELSIF ultimo.tipo='aceptada' THEN v_estado:='cubierta';
  ELSIF ultimo.tipo='llamamiento_directo' THEN v_estado:='llamamiento_directo';
  ELSE v_estado:='vacante';
  END IF;
  v_estados:=v_estados||v_estado;
  v_ultimos:=v_ultimos||CASE WHEN ultimo.tipo IS NULL THEN 'null'::jsonb ELSE jsonb_build_object(
   'secuencia',ultimo.secuencia,'tipo',ultimo.tipo,'participacion_ref',ultimo.participacion_ref,
   'orden_vigente',ultimo.orden_vigente,
   'responder_antes_de',to_char(ultimo.responder_antes_de AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END;
  IF v_estado='cubierta' THEN v_cubiertas:=v_cubiertas+1; END IF;
  IF v_estado='llamamiento_directo' THEN v_directas:=v_directas+1; END IF;
  IF ultimo.tipo IS NOT NULL THEN v_actos:=v_actos+1; END IF;
 END LOOP;
 v_pendiente:='pendiente_respuesta'=ANY(v_estados);
 IF p_corte>=o.vence_antes_de THEN
  SELECT coalesce(array_agg(x.participacion_ref ORDER BY x.orden_vigente,x.participacion_ref),'{}'),
         coalesce(array_agg(x.orden_vigente ORDER BY x.orden_vigente,x.participacion_ref),'{}')
    INTO v_candidatos,v_ordenes
    FROM (SELECT d.participacion_ref,ov.orden_vigente
            FROM vec_bolsa_llamamientos.disposicion_oferta d
            JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(o.bolsa_ref,p_corte) ov ON ov.participacion_ref=d.participacion_ref
           WHERE d.oferta_ref=p_oferta AND d.manifestada_en<o.vence_antes_de AND ov.orden_vigente IS NOT NULL
             AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.acto_plaza_oferta a
                              WHERE a.oferta_ref=p_oferta AND a.participacion_ref=d.participacion_ref AND a.registrado_en<=p_corte)) x;
 END IF;
 FOR i IN 1..v_n LOOP
  v_propuesta:=NULL;
  IF p_corte>=o.vence_antes_de AND v_estados[i]='vacante' THEN
   -- Sin apartado de plazas o llamada sucesiva: una propuesta cada vez, para
   -- la primera plaza libre y sin nadie pendiente de responder.
   IF (v_gestion AND v_config->>'llamada'='simultanea') OR (v_primera AND NOT v_pendiente) THEN
    IF v_gestion AND v_ultimos[i]->>'tipo' IN ('renuncia','sin_respuesta') AND v_config->>'tras_renuncia'='llamamiento_directo' THEN
     v_propuesta:=jsonb_build_object('tipo','llamamiento_directo');
    ELSIF v_siguiente<=coalesce(array_length(v_candidatos,1),0) THEN
     v_propuesta:=jsonb_build_object('tipo','adjudicar','participacion_ref',v_candidatos[v_siguiente],'orden_vigente',v_ordenes[v_siguiente]);
     v_siguiente:=v_siguiente+1;
    ELSE
     v_propuesta:=jsonb_build_object('tipo','llamamiento_directo');
    END IF;
   END IF;
   v_primera:=false;
  END IF;
  SELECT coalesce(jsonb_agg(jsonb_build_object('secuencia',a.secuencia,'tipo',a.tipo,'orden_vigente',a.orden_vigente,
          'registrado_en',to_char(a.registrado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'recibo_ref',a.recibo_ref)
          ORDER BY a.secuencia),'[]'::jsonb)
    INTO v_historial FROM vec_bolsa_llamamientos.acto_plaza_oferta a
   WHERE a.oferta_ref=p_oferta AND a.numero_de_plaza=i AND a.registrado_en<=p_corte;
  v_puede:=v_estados[i]='pendiente_respuesta'
   AND (v_ultimos[i]->>'responder_antes_de' IS NULL OR p_corte>=(v_ultimos[i]->>'responder_antes_de')::timestamptz);
  v_plazas:=v_plazas||jsonb_build_array(jsonb_build_object(
   'numero_de_plaza',i,'estado',v_estados[i],
   'secuencia',coalesce((v_ultimos[i]->>'secuencia')::int,0),
   'participacion_ref',CASE WHEN v_estados[i] IN ('pendiente_respuesta','cubierta') THEN v_ultimos[i]->'participacion_ref' ELSE NULL END,
   'orden_vigente',CASE WHEN v_estados[i] IN ('pendiente_respuesta','cubierta') THEN v_ultimos[i]->'orden_vigente' ELSE NULL END,
   'responder_antes_de',CASE WHEN v_estados[i]='pendiente_respuesta' THEN v_ultimos[i]->'responder_antes_de' ELSE NULL END,
   'puede_sin_respuesta',v_puede,'propuesta',v_propuesta,'historial',v_historial));
 END LOOP;
 RETURN base || jsonb_build_object(
  'estado',CASE WHEN p_corte<o.vence_antes_de THEN 'abierta'
                WHEN v_cubiertas=v_n THEN 'adjudicada'
                WHEN v_directas=v_n THEN 'llamamiento_directo'
                WHEN v_cubiertas+v_directas=v_n THEN 'cerrada'
                WHEN v_actos>0 THEN 'en_curso'
                ELSE 'pendiente_resolucion' END,
  'propuesta',NULL,'numero_plazas',v_n,'politica_plazas',v_config,'confirmacion_adjudicacion',v_confirmacion,'plazas',v_plazas);
END $f$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_acto_plaza_oferta_v1(
 p_oferta text,p_recibo text,p_bolsa text,p_numero_de_plaza integer,p_tipo text,p_participacion text,
 p_secuencia_esperada integer,p_actor text,p_clave text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(oferta jsonb,reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE o record; previo record; consumo record; d jsonb; v_ahora timestamptz;
 v_proy jsonb; v_plaza jsonb; v_huella text; v_responder timestamptz; v_orden bigint;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_oferta IS NULL OR p_oferta !~ '^oferta:[0-9a-f]{64}$'
    OR p_recibo IS NULL OR p_recibo !~ '^recibo:plaza-oferta:[0-9a-f]{64}$'
    OR p_bolsa IS NULL OR octet_length(p_bolsa) NOT BETWEEN 1 AND 256 OR p_bolsa<>btrim(p_bolsa)
    OR p_numero_de_plaza IS NULL OR p_numero_de_plaza NOT BETWEEN 1 AND 100
    OR p_tipo IS NULL OR p_tipo NOT IN ('adjudicada','aceptada','renuncia','sin_respuesta','llamamiento_directo')
    OR (p_tipo='llamamiento_directo')<>(p_participacion IS NULL)
    OR (p_participacion IS NOT NULL AND (p_participacion<>btrim(p_participacion) OR octet_length(p_participacion) NOT BETWEEN 1 AND 256))
    OR p_secuencia_esperada IS NULL OR p_secuencia_esperada NOT BETWEEN 0 AND 9999
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave<>btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256 THEN
  RAISE EXCEPTION 'B58: acto de plaza inválido' USING ERRCODE='22023';
 END IF;
 -- El mismo cerrojo que la disposición y la resolución B28 de esta oferta.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:resolucion-oferta:'||p_oferta,0));
 -- El instante se toma ya con el cerrojo: así ve el acto que otra sesión
 -- acaba de confirmar sobre la misma plaza y responde «la plaza ha cambiado».
 v_ahora:=clock_timestamp();
 SELECT * INTO o FROM vec_bolsa_llamamientos.oferta_publicada WHERE oferta_ref=p_oferta;
 IF NOT FOUND OR o.bolsa_ref<>p_bolsa THEN
  RAISE EXCEPTION 'B58: oferta inexistente en la bolsa' USING ERRCODE='23503';
 END IF;
 -- La decisión viva se consume antes de resolver el replay: un reintento no
 -- devuelve el recibo sin una autorización nueva y verificada.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B58: acto de plaza no autorizado' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR consumo.consumo_nuevo IS NOT TRUE
    OR consumo.decision_ref IS NULL OR consumo.auditoria_ref IS NULL
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_llamamientos_bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM p_bolsa
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' THEN
  RAISE EXCEPTION 'B58: acto de plaza no autorizado' USING ERRCODE='42501';
 END IF;
 v_huella:=encode(sha256(convert_to(array_to_string(ARRAY[p_oferta,p_numero_de_plaza::text,p_tipo,
   coalesce(p_participacion,''),p_secuencia_esperada::text],chr(31)),'UTF8')),'hex');
 SELECT * INTO previo FROM vec_bolsa_llamamientos.acto_plaza_oferta WHERE oferta_ref=p_oferta AND clave_idempotencia=p_clave;
 IF FOUND THEN
  IF previo.actor_ref<>p_actor OR previo.huella_comando_sha256<>v_huella OR previo.recibo_ref<>p_recibo THEN
   RAISE EXCEPTION 'B58: clave reutilizada con otro acto' USING ERRCODE='VBO01';
  END IF;
  RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,v_ahora),true;
  RETURN;
 END IF;
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.resolucion_oferta r WHERE r.oferta_ref=p_oferta) THEN
  RAISE EXCEPTION 'B58: oferta ya resuelta' USING ERRCODE='VBO02';
 END IF;
 IF v_ahora<o.vence_antes_de THEN
  RAISE EXCEPTION 'B58: plazo de disposición abierto' USING ERRCODE='VBO03';
 END IF;
 v_proy:=vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,v_ahora);
 v_plaza:=v_proy->'plazas'->(p_numero_de_plaza-1);
 IF v_plaza IS NULL OR (v_plaza->>'numero_de_plaza')::int IS DISTINCT FROM p_numero_de_plaza THEN
  RAISE EXCEPTION 'B58: plaza inexistente' USING ERRCODE='22023';
 END IF;
 IF (v_plaza->>'secuencia')::int IS DISTINCT FROM p_secuencia_esperada THEN
  RAISE EXCEPTION 'B58: la plaza ha cambiado' USING ERRCODE='VBO04';
 END IF;
 IF p_tipo IN ('adjudicada','llamamiento_directo') THEN
  IF v_plaza->>'estado' IS DISTINCT FROM 'vacante' OR jsonb_typeof(v_plaza->'propuesta') IS DISTINCT FROM 'object'
     OR v_plaza#>>'{propuesta,tipo}' IS DISTINCT FROM (CASE p_tipo WHEN 'adjudicada' THEN 'adjudicar' ELSE 'llamamiento_directo' END)
     OR v_plaza#>>'{propuesta,participacion_ref}' IS DISTINCT FROM p_participacion THEN
   RAISE EXCEPTION 'B58: la propuesta ha cambiado' USING ERRCODE='VBO04';
  END IF;
  v_orden:=(v_plaza#>>'{propuesta,orden_vigente}')::bigint;
  IF p_tipo='adjudicada' AND jsonb_typeof(v_proy->'politica_plazas')='object'
     AND v_proy->>'confirmacion_adjudicacion' IS DISTINCT FROM 'aceptacion_previa' THEN
   v_responder:=v_ahora+((v_proy#>>'{politica_plazas,respuesta_horas}')::int*interval '1 hour');
  END IF;
 ELSE
  IF v_proy->>'confirmacion_adjudicacion'='aceptacion_previa'
     OR jsonb_typeof(v_proy->'politica_plazas') IS DISTINCT FROM 'object'
     OR v_plaza->>'estado' IS DISTINCT FROM 'pendiente_respuesta'
     OR v_plaza->>'participacion_ref' IS DISTINCT FROM p_participacion THEN
   RAISE EXCEPTION 'B58: la plaza ha cambiado' USING ERRCODE='VBO04';
  END IF;
  -- «aceptada» y «renuncia» se admiten también tras el plazo: RRHH puede
  -- registrar tarde una respuesta que llegó a tiempo. Solo la falta de
  -- respuesta exige que el plazo haya vencido.
  IF p_tipo='sin_respuesta' AND (v_plaza->>'puede_sin_respuesta')::boolean IS NOT TRUE THEN
   RAISE EXCEPTION 'B58: plazo de respuesta abierto' USING ERRCODE='VBO07';
  END IF;
  v_orden:=(v_plaza->>'orden_vigente')::bigint;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.acto_plaza_oferta(oferta_ref,numero_de_plaza,secuencia,tipo,participacion_ref,orden_vigente,
  responder_antes_de,recibo_ref,actor_ref,clave_idempotencia,huella_comando_sha256,decision_ref,auditoria_ref,registrado_en)
 VALUES(p_oferta,p_numero_de_plaza,p_secuencia_esperada+1,p_tipo,p_participacion,v_orden,
  v_responder,p_recibo,p_actor,p_clave,v_huella,consumo.decision_ref,consumo.auditoria_ref,v_ahora);
 INSERT INTO vec_bolsa_llamamientos.acto_plaza_oferta_outbox(recibo_ref,oferta_ref,bolsa_ref,numero_de_plaza,secuencia,tipo,creada_en)
 VALUES(p_recibo,p_oferta,p_bolsa,p_numero_de_plaza,p_secuencia_esperada+1,p_tipo,v_ahora);
 RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,v_ahora),false;
END $f$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.publicar_politica_ofertas_v1(
 p_bolsa text,p_version_esperada bigint,p_politica jsonb,p_actor text,p_clave text,p_recibo text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(politica jsonb,reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE consumo record; d jsonb; previa record; v_version bigint; v_huella text; v_ahora timestamptz:=clock_timestamp();
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_bolsa IS NULL OR p_bolsa !~ '^bolsa:[A-Za-z0-9:_-]{1,250}$'
    OR p_version_esperada IS NULL OR p_version_esperada<0 OR p_version_esperada>2147483646
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave !~ '^[A-Za-z0-9:_-]+$' OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_recibo IS NULL OR p_recibo !~ '^recibo:politica-ofertas:[0-9a-f]{64}$'
    OR jsonb_typeof(p_politica) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica)) NOT IN (3,4)
    OR NOT (p_politica ?& ARRAY['plazo','adjudicacion','no_cubierta'])
    OR ((SELECT count(*) FROM jsonb_object_keys(p_politica))=4 AND NOT (p_politica ? 'plazas'))
    -- B58: apartado opcional de plazas. Solo valores que ejecuta la proyección.
    OR (p_politica ? 'plazas' AND (
        jsonb_typeof(p_politica->'plazas') IS DISTINCT FROM 'object'
        OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'plazas'))<>3
        OR NOT (p_politica->'plazas' ?& ARRAY['llamada','respuesta_horas','tras_renuncia'])
        OR coalesce(p_politica#>>'{plazas,llamada}','') NOT IN ('simultanea','sucesiva')
        OR jsonb_typeof(p_politica#>'{plazas,respuesta_horas}') IS DISTINCT FROM 'number'
        OR coalesce(p_politica#>>'{plazas,respuesta_horas}','') !~ '^([1-9][0-9]?|[1-6][0-9]{2}|7[01][0-9]|720)$'
        OR coalesce(p_politica#>>'{plazas,tras_renuncia}','') NOT IN ('siguiente_en_orden','llamamiento_directo')))
    OR jsonb_typeof(p_politica->'plazo') IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'plazo')) NOT IN (4,5)
    OR ((SELECT count(*) FROM jsonb_object_keys(p_politica->'plazo'))=5
        AND p_politica#>>'{plazo,inicio}' IS DISTINCT FROM 'notificacion')
    OR NOT (p_politica->'plazo' ?& ARRAY['unidad','cantidad','computo','municipio_sede'])
    OR coalesce(p_politica#>>'{plazo,unidad}','') NOT IN ('dias_habiles','dias_naturales','horas_naturales')
    OR jsonb_typeof(p_politica#>'{plazo,cantidad}') IS DISTINCT FROM 'number'
    OR coalesce(p_politica#>>'{plazo,cantidad}','') !~ '^[0-9]{1,3}$'
    OR ((p_politica#>>'{plazo,unidad}' IN ('dias_habiles','dias_naturales')
             AND (p_politica#>>'{plazo,cantidad}')::int BETWEEN 1 AND 30
             AND p_politica#>>'{plazo,computo}'='administrativo')
         OR (p_politica#>>'{plazo,unidad}'='horas_naturales'
             AND (p_politica#>>'{plazo,cantidad}')::int BETWEEN 1 AND 720
             AND p_politica#>>'{plazo,computo}'='continuo_utc')) IS NOT TRUE
    OR coalesce(p_politica#>>'{plazo,municipio_sede}','') !~ '^[0-9]{5}$'
    OR jsonb_typeof(p_politica->'adjudicacion') IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'adjudicacion')) NOT IN (2,3)
    OR ((SELECT count(*) FROM jsonb_object_keys(p_politica->'adjudicacion'))=3
        AND p_politica#>>'{adjudicacion,confirmacion}' IS DISTINCT FROM 'aceptacion_previa')
    OR p_politica#>>'{adjudicacion,criterio}' IS DISTINCT FROM 'orden_vigente'
    OR p_politica#>>'{adjudicacion,elegibilidad}' IS DISTINCT FROM 'disposicion_en_plazo'
    OR jsonb_typeof(p_politica->'no_cubierta') IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'no_cubierta'))<>2
    OR p_politica#>>'{no_cubierta,accion}' IS DISTINCT FROM 'llamamiento_directo'
    OR p_politica#>>'{no_cubierta,condicion}' IS DISTINCT FROM 'sin_disposiciones_elegibles'
 THEN RAISE EXCEPTION 'B47: política de ejemplo inválida' USING ERRCODE='22023'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_politica_ofertas_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B47: autorización inválida' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR consumo.consumo_nuevo IS NOT TRUE
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'bolsa.politica_ofertas.publicar'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida'
    OR d->>'finalidad' IS DISTINCT FROM 'gobierno_politica_ofertas_bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM p_bolsa
 THEN RAISE EXCEPTION 'B47: publicación no autorizada' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:politica-ofertas:'||p_bolsa,0));
 v_huella:=encode(sha256(convert_to(p_politica::text,'UTF8')),'hex');
 SELECT * INTO previa FROM vec_bolsa_llamamientos.politica_ofertas_version
  WHERE bolsa_ref=p_bolsa AND clave_idempotencia=p_clave;
 IF FOUND THEN
  IF previa.actor_ref<>p_actor OR previa.version_esperada<>p_version_esperada
     OR previa.huella_sha256<>v_huella OR previa.recibo_ref<>p_recibo
  THEN RAISE EXCEPTION 'B47: clave reutilizada' USING ERRCODE='VBP01'; END IF;
  RETURN QUERY SELECT jsonb_build_object('bolsa_ref',previa.bolsa_ref,'version',previa.version,
   'huella_sha256',previa.huella_sha256,'ejemplo',true,'configurada',true,'politica',previa.politica,
   'recibo_ref',previa.recibo_ref,
   'publicada_en',to_char(previa.publicada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),true;
  RETURN;
 END IF;
 -- Las versiones anteriores pueden repetirse, pero toda política nueva declara
 -- en su propia versión que el plazo empieza en la notificación.
 IF p_politica#>>'{plazo,inicio}' IS DISTINCT FROM 'notificacion' THEN
  RAISE EXCEPTION 'B71: inicio de plazo no configurado' USING ERRCODE='22023'; END IF;
 -- El replay histórico ya devolvió arriba el contenido y recibo originales.
 -- Toda versión nueva requiere la aceptación telemática previa.
 IF p_politica#>>'{adjudicacion,confirmacion}' IS DISTINCT FROM 'aceptacion_previa' THEN
  RAISE EXCEPTION 'B70: confirmación telemática requerida para la versión nueva' USING ERRCODE='22023';
 END IF;
 SELECT coalesce(max(version),0) INTO v_version FROM vec_bolsa_llamamientos.politica_ofertas_version WHERE bolsa_ref=p_bolsa;
 IF v_version<>p_version_esperada THEN
  RAISE EXCEPTION 'B47: versión esperada obsoleta' USING ERRCODE='VBP01';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion WHERE bolsa_ref=p_bolsa) THEN
  RAISE EXCEPTION 'B47: bolsa no constituida' USING ERRCODE='23503'; END IF;
 INSERT INTO vec_bolsa_llamamientos.politica_ofertas_version(
  bolsa_ref,version,politica,huella_sha256,ejemplo,actor_ref,clave_idempotencia,version_esperada,
  recibo_ref,publicada_en,decision_ref,auditoria_ref)
 VALUES(p_bolsa,v_version+1,p_politica,v_huella,true,p_actor,p_clave,p_version_esperada,
  p_recibo,v_ahora,consumo.decision_ref,consumo.auditoria_ref);
 INSERT INTO vec_bolsa_llamamientos.politica_ofertas_outbox(recibo_ref,bolsa_ref,version,huella_sha256,creada_en)
 VALUES(p_recibo,p_bolsa,v_version+1,v_huella,v_ahora);
 RETURN QUERY SELECT vec_bolsa_llamamientos.leer_politica_ofertas_v1(p_bolsa),false;
END $f$;

-- Las funciones sustituidas conservan propietario, firmas y ACL nominales.
COMMIT;
