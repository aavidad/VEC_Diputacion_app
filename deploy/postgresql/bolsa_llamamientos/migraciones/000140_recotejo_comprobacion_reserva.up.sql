\set ON_ERROR_STOP on
-- B140 / F1-10: recotejo de contexto y acto en la reserva; ninguna excepción habilitada.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000140',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_bolsa_llamamientos_propietario'
 OR to_regclass('vec_bolsa_llamamientos.version_comprobacion_reserva_v1') IS NOT NULL
 OR to_regprocedure('vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(jsonb)') IS NOT NULL
 OR encode(sha256(convert_to(pg_get_functiondef(
  'vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text,jsonb)'::regprocedure),'UTF8')),'hex')
 IS DISTINCT FROM '401b1e9430d2d7c5be462ba733abb6378277b45d2224d1a554768fde93b9afac'
 OR encode(sha256(convert_to(pg_get_functiondef(
  'vec_bolsa_llamamientos.registrar_comprobacion_ofrecimiento_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
 IS DISTINCT FROM '155125748bb761af85c1b872e755079bf5df5944e6e6e448c20da05399575042'
 OR encode(sha256(convert_to(pg_get_functiondef(
  'vec_bolsa_llamamientos.reservar_llamamiento_ct_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)'::regprocedure),'UTF8')),'hex')
 IS DISTINCT FROM '4b250b48f08751fdb3ea56f5b747beb00b9deef68f95c96628c02d95bb3c2990'
 THEN RAISE EXCEPTION 'B140: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE TABLE vec_bolsa_llamamientos.version_comprobacion_reserva_v1(
 organizacion_ref text NOT NULL,expediente_ref text NOT NULL,necesidad_ref text NOT NULL,
 participacion_ref text NOT NULL,version bigint NOT NULL CHECK(version>=0),
 PRIMARY KEY(organizacion_ref,expediente_ref,necesidad_ref,participacion_ref));
ALTER TABLE vec_bolsa_llamamientos.version_comprobacion_reserva_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.version_comprobacion_reserva_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY version_comprobacion_propietario ON vec_bolsa_llamamientos.version_comprobacion_reserva_v1
 TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.version_comprobacion_reserva_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.version_comprobacion_reserva_v1 FROM PUBLIC;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(p_categoria text, p_modalidad text, p_causa text, p_contexto jsonb DEFAULT NULL::jsonb)
 RETURNS TABLE(acta_ref text, categoria_ref text, bolsa_ref text, version_bolsa bigint, huella_bolsa_sha256 text, bolsa_canonica bytea, instantanea_ref text, version_instantanea bigint, huella_instantanea_sha256 text, instantanea_canonica bytea, politica_ref text, version_politica bigint, politica_canonica bytea, politica_provisional boolean, corte timestamp with time zone, bolsa_vigente_hasta timestamp with time zone, politica_vigente_hasta timestamp with time zone, entradas jsonb, criterios_ofrecimiento jsonb)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE
 v_corte timestamptz(6):=date_trunc('microseconds',statement_timestamp());
 v_const record;
 v_politica vec_bolsa_llamamientos.politica_orden_bolsa%ROWTYPE;
 v_entradas jsonb;
 v_total bigint;
 v_orden bigint;
 v_sin_vinculo bigint;
 v_total_posiciones bigint;
 v_politica_divergente boolean;
 v_oferta record;
 v_criterios jsonb;
BEGIN
 IF p_categoria IS NULL OR octet_length(p_categoria) NOT BETWEEN 3 AND 512
    OR p_categoria<>btrim(p_categoria)
    OR p_modalidad IS NULL OR p_modalidad !~ '^[a-z][a-z0-9._-]{1,79}$'
    OR p_causa IS NULL OR (p_causa<>'' AND p_causa !~ '^[a-z][a-z0-9._-]{1,79}$') THEN
  RAISE EXCEPTION 'B110: categoría inválida' USING ERRCODE='22023'; END IF;
 IF p_contexto IS NOT NULL THEN
  IF jsonb_typeof(p_contexto) IS DISTINCT FROM 'object'
   OR octet_length(p_contexto::text)>4096
  THEN RAISE EXCEPTION 'B137: contexto inválido' USING ERRCODE='42501'; END IF;
  IF (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(p_contexto) k)
    IS DISTINCT FROM ARRAY['ambito_ref','categoria_ref','causa_ofrecida','expediente_ref',
      'modalidad_ofrecida','necesidad_ref','organizacion_ref','unidad_ref','version_expediente']
   OR EXISTS(SELECT 1 FROM jsonb_each(p_contexto) x WHERE
      CASE WHEN x.key='version_expediente' THEN
        jsonb_typeof(x.value)<>'number' OR x.value#>>'{}' !~ '^[1-9][0-9]{0,8}$'
      WHEN x.key='causa_ofrecida' THEN
        jsonb_typeof(x.value)<>'string' OR octet_length(x.value#>>'{}')>256
      ELSE jsonb_typeof(x.value)<>'string'
        OR octet_length(x.value#>>'{}') NOT BETWEEN 2 AND 256
        OR x.value#>>'{}' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]+$' END)
   OR p_contexto->>'categoria_ref' IS DISTINCT FROM p_categoria
   OR p_contexto->>'modalidad_ofrecida' IS DISTINCT FROM p_modalidad
   OR p_contexto->>'causa_ofrecida' IS DISTINCT FROM p_causa
  THEN RAISE EXCEPTION 'B137: contexto inválido' USING ERRCODE='42501'; END IF;
  IF p_contexto->>'ambito_ref' IS DISTINCT FROM
    (vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1())->>'ambito_ref'
  THEN RAISE EXCEPTION 'B137: ámbito ajeno' USING ERRCODE='42501'; END IF;
  IF vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(p_contexto,true) IS NOT TRUE
  THEN RAISE EXCEPTION 'B137: contexto CT ajeno' USING ERRCODE='42501'; END IF;
 END IF;
 SELECT count(*) INTO v_total
 FROM vec_bolsa_llamamientos.constitucion c
 JOIN vec_bolsa_llamamientos.bolsa_constituida b
   ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa
  AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
 WHERE c.categoria_ref=p_categoria AND b.estado='vigente'
   AND b.vigente_desde<=v_corte AND (b.vigente_hasta IS NULL OR v_corte<b.vigente_hasta)
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa s
     WHERE s.bolsa_ref_sustituida=c.bolsa_ref AND s.version_sustituida=c.version_bolsa
       AND s.huella_sustituida_sha256=c.huella_bolsa_sha256);
 IF v_total<>1 THEN
  RAISE EXCEPTION 'B110: constitución vigente ausente o ambigua' USING ERRCODE='P1101'; END IF;
 SELECT c.acta_ref,c.categoria_ref,c.bolsa_ref,c.version_bolsa,c.huella_bolsa_sha256,
        b.bolsa_canonica,c.instantanea_ref,c.version_instantanea,
        c.huella_instantanea_sha256,i.instantanea_canonica,i.total_participaciones,
        b.vigente_hasta AS bolsa_vigente_hasta
 INTO STRICT v_const
 FROM vec_bolsa_llamamientos.constitucion c
 JOIN vec_bolsa_llamamientos.bolsa_constituida b
   ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa
  AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
 JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i
   ON i.instantanea_ref=c.instantanea_ref AND i.version=c.version_instantanea
  AND i.huella_instantanea_sha256=c.huella_instantanea_sha256
 WHERE c.categoria_ref=p_categoria AND b.estado='vigente'
   AND b.vigente_desde<=v_corte AND (b.vigente_hasta IS NULL OR v_corte<b.vigente_hasta)
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa s
     WHERE s.bolsa_ref_sustituida=c.bolsa_ref AND s.version_sustituida=c.version_bolsa
       AND s.huella_sustituida_sha256=c.huella_bolsa_sha256);
 IF encode(sha256(v_const.bolsa_canonica),'hex')<>v_const.huella_bolsa_sha256
    OR encode(sha256(v_const.instantanea_canonica),'hex')<>v_const.huella_instantanea_sha256
 THEN RAISE EXCEPTION 'B110: canónico divergente' USING ERRCODE='P1101'; END IF;
 SELECT * INTO v_politica FROM vec_bolsa_llamamientos.politica_orden_bolsa p
 WHERE p.bolsa_ref=v_const.bolsa_ref AND p.vigente_desde<=v_corte
   AND (p.vigente_hasta IS NULL OR v_corte<p.vigente_hasta)
 ORDER BY p.version DESC LIMIT 1;
 IF NOT FOUND THEN
  RAISE EXCEPTION 'B110: política de orden publicada ausente' USING ERRCODE='P1101'; END IF;
 SELECT po.version,po.huella_sha256,po.politica->'criterios_ofrecimiento' AS criterios
 INTO v_oferta FROM vec_bolsa_llamamientos.politica_ofertas_version po
 WHERE po.bolsa_ref=v_const.bolsa_ref AND po.publicada_en<=v_corte
 ORDER BY po.version DESC LIMIT 1;
 IF FOUND AND v_oferta.criterios IS NOT NULL AND v_oferta.criterios<>'{}'::jsonb THEN
  IF vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(v_oferta.criterios) IS NOT TRUE
  THEN RAISE EXCEPTION 'B118: criterios publicados inválidos' USING ERRCODE='55000'; END IF;
  v_criterios:=jsonb_build_object('version_politica_ofertas',v_oferta.version,
   'huella_politica_ofertas_sha256',v_oferta.huella_sha256,
   'modalidad_ofrecida',p_modalidad,'causa_ofrecida',p_causa,
   'configuracion',v_oferta.criterios);
 END IF;
 WITH posiciones AS MATERIALIZED (
   SELECT o.* FROM vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(v_const.bolsa_ref,p_modalidad,p_causa,v_corte) o
 ), control_posiciones AS (
   SELECT count(*) AS total,
     coalesce(bool_or(o.politica_ref IS DISTINCT FROM v_politica.politica_ref
       OR o.version_politica IS DISTINCT FROM v_politica.version
       OR o.provisional IS DISTINCT FROM v_politica.provisional),false) AS divergente
   FROM posiciones o
 )
 SELECT count(*),count(p.participacion_ref),
        count(*) FILTER(WHERE p.orden_vigente IS NOT NULL AND v.candidato_ref IS NULL),
        jsonb_agg(jsonb_build_object('orden_acta',e.orden,'orden_vigente',p.orden_vigente,
          'participacion_ref',e.participacion_ref,'candidato_ref',v.candidato_ref,
          'situacion',p.situacion,'razon',p.razon,
          'eventos_relacion',coalesce(p.eventos_relacion,'[]'::jsonb)) ORDER BY e.orden),
        max(control_posiciones.total),bool_or(control_posiciones.divergente)
 INTO v_total,v_orden,v_sin_vinculo,v_entradas,
      v_total_posiciones,v_politica_divergente
 FROM vec_bolsa_llamamientos.constitucion_entrada e
 CROSS JOIN control_posiciones
 LEFT JOIN posiciones p ON p.participacion_ref=e.participacion_ref AND p.orden_acta=e.orden
 LEFT JOIN vec_bolsa_llamamientos.vinculo_candidato v
   ON v.participacion_ref=e.participacion_ref AND v.acta_ref=v_const.acta_ref
 WHERE e.instantanea_ref=v_const.instantanea_ref
   AND e.version_instantanea=v_const.version_instantanea;
 IF v_total<>v_const.total_participaciones OR v_orden<>v_total OR v_sin_vinculo<>0
    OR v_entradas IS NULL
    OR v_total_posiciones IS DISTINCT FROM v_total
    OR v_politica_divergente IS NOT FALSE
 THEN RAISE EXCEPTION 'B110: orden o vínculo incompleto' USING ERRCODE='P1101'; END IF;
 IF p_contexto IS NOT NULL AND v_criterios IS NOT NULL THEN
  SELECT jsonb_agg(e.valor||CASE WHEN e.valor->>'razon'='pendiente_comprobacion_rrhh' THEN jsonb_build_object('comprobacion_ofrecimiento',
    CASE WHEN a.comprobacion_ref IS NULL THEN jsonb_build_object('version_esperada',0)
    ELSE jsonb_build_object(
     'comprobacion_ref',a.comprobacion_ref,'resultado',a.resultado->>'resultado',
     'motivo_clave',a.resultado->>'motivo_clave',
     'version_politica_ofertas',a.contexto->'version_politica_ofertas',
     'version_esperada',a.version,'registrada_en',a.registrada_en,'recibo_ref',a.recibo_ref,
     'vigente',a.contexto=jsonb_build_object(
       'ambito_ref',p_contexto->>'ambito_ref','bolsa_ref',v_const.bolsa_ref,
       'categoria_ref',p_categoria,'causa_ofrecida',p_causa,
       'expediente_ref',p_contexto->>'expediente_ref',
       'huella_politica_ofertas_sha256',v_oferta.huella_sha256,
       'modalidad_ofrecida',p_modalidad,'necesidad_ref',p_contexto->>'necesidad_ref',
       'orden_acta',(e.valor->>'orden_acta')::bigint,
       'organizacion_ref',p_contexto->>'organizacion_ref',
       'participacion_ref',e.valor->>'participacion_ref','unidad_ref',p_contexto->>'unidad_ref',
       'version_bolsa',v_const.version_bolsa,
       'version_expediente',(p_contexto->>'version_expediente')::bigint,
       'version_instantanea',v_const.version_instantanea,
       'version_politica_ofertas',v_oferta.version) AND a.bolsa_ref=v_const.bolsa_ref) END) ELSE '{}'::jsonb END ORDER BY e.n)
  INTO v_entradas
  FROM jsonb_array_elements(v_entradas) WITH ORDINALITY e(valor,n)
  LEFT JOIN LATERAL (
   SELECT x.* FROM vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 x
   WHERE x.organizacion_ref=p_contexto->>'organizacion_ref'
     AND x.expediente_ref=p_contexto->>'expediente_ref'
     AND x.necesidad_ref=p_contexto->>'necesidad_ref'
     AND x.participacion_ref=e.valor->>'participacion_ref'
     AND e.valor->>'razon'='pendiente_comprobacion_rrhh'
   ORDER BY x.version DESC LIMIT 1) a ON true;
  IF v_entradas IS NULL THEN RAISE EXCEPTION 'B137: proyección incompleta' USING ERRCODE='55000'; END IF;
 END IF;
 RETURN QUERY SELECT v_const.acta_ref,v_const.categoria_ref,v_const.bolsa_ref,v_const.version_bolsa,
   v_const.huella_bolsa_sha256,v_const.bolsa_canonica,v_const.instantanea_ref,
   v_const.version_instantanea,v_const.huella_instantanea_sha256,v_const.instantanea_canonica,
   v_politica.politica_ref,v_politica.version,convert_to(to_jsonb(v_politica)::text,'UTF8'),
   v_politica.provisional,v_corte,v_const.bolsa_vigente_hasta,v_politica.vigente_hasta,v_entradas,v_criterios;
END $function$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_comprobacion_ofrecimiento_v1(p jsonb, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE d jsonb;c jsonb;consumo record;previa record;politica record;entrada record;
 version_actual bigint;huella text;contexto_sha text;referencia text;recibo text;evento text;
 instante timestamptz(6);resultado jsonb;hechos jsonb;conjunto jsonb;es_replay boolean;origen record;
BEGIN
 IF session_user=current_user
 OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
 OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
 OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
 THEN RAISE EXCEPTION 'B133: registro denegado' USING ERRCODE='42501'; END IF;
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR octet_length(p::text)>49152
 OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(p) k)
    IS DISTINCT FROM ARRAY['clave_idempotencia','contexto','hechos','version_esperada']
 OR jsonb_typeof(p->'contexto') IS DISTINCT FROM 'object'
 OR jsonb_typeof(p->'version_esperada') IS DISTINCT FROM 'number'
 OR p->>'version_esperada' !~ '^(0|[1-9][0-9]{0,8})$'
 OR jsonb_typeof(p->'clave_idempotencia') IS DISTINCT FROM 'string'
 OR octet_length(p->>'clave_idempotencia') NOT BETWEEN 8 AND 256
 OR p->>'clave_idempotencia' !~ '^[A-Za-z0-9:_-]+$'
 THEN RAISE EXCEPTION 'B133: solicitud inválida' USING ERRCODE='22023'; END IF;
 c:=p->'contexto';
 IF (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(c) k) IS DISTINCT FROM
 ARRAY['ambito_ref','bolsa_ref','categoria_ref','causa_ofrecida','expediente_ref',
  'huella_politica_ofertas_sha256','modalidad_ofrecida','necesidad_ref','orden_acta',
  'organizacion_ref','participacion_ref','unidad_ref','version_bolsa','version_expediente',
  'version_instantanea','version_politica_ofertas']
 OR EXISTS(SELECT 1 FROM jsonb_each(c) x WHERE
  CASE WHEN x.key IN ('orden_acta','version_bolsa','version_expediente','version_instantanea','version_politica_ofertas')
   THEN jsonb_typeof(x.value)<>'number' OR x.value#>>'{}' !~ '^[1-9][0-9]{0,8}$'
   ELSE jsonb_typeof(x.value)<>'string' OR octet_length(x.value#>>'{}')>256
    OR (x.key<>'causa_ofrecida' AND x.value#>>'{}' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]+$') END)
 OR c->>'huella_politica_ofertas_sha256' !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'B133: contexto inválido' USING ERRCODE='22023'; END IF;
 huella:=encode(sha256(convert_to(p::text,'UTF8')),'hex');
 -- Canon del recurso V3, igual que B115 y RecursoAutorizable en Go.
 contexto_sha:=encode(sha256(convert_to('{"ambitos":{"ambito_ref":'||to_json(c->>'ambito_ref')::text||
  ',"unidad_ref":'||to_json(c->>'unidad_ref')::text||
  '},"atributos":{"material_sha256":"'||huella||'"}}','UTF8')),'hex');
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_comprobacion_ofrecimiento_v1(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 d:=convert_from(p_decision,'UTF8')::jsonb;
 IF consumo.efecto_ref IS DISTINCT FROM c->>'participacion_ref'
 OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha
 OR d#>>'{ambitos,unidad_ref}' IS DISTINCT FROM c->>'unidad_ref'
 OR d#>>'{ambitos,ambito_ref}' IS DISTINCT FROM c->>'ambito_ref'
 THEN RAISE EXCEPTION 'B133: decisión ajena al acto' USING ERRCODE='42501'; END IF;
 conjunto:=vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1();
 IF c->>'ambito_ref' IS DISTINCT FROM conjunto->>'ambito_ref' OR conjunto->>'ambito_ref' IS NULL
 THEN RAISE EXCEPTION 'B133: ámbito ajeno' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:comprobacion:'||
  jsonb_build_array(c->>'organizacion_ref',c->>'expediente_ref',c->>'necesidad_ref',c->>'participacion_ref')::text,0));
 INSERT INTO vec_bolsa_llamamientos.version_comprobacion_reserva_v1(
  organizacion_ref,expediente_ref,necesidad_ref,participacion_ref,version)
 SELECT c->>'organizacion_ref',c->>'expediente_ref',c->>'necesidad_ref',c->>'participacion_ref',
  coalesce(max(x.version),0) FROM vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 x
 WHERE x.organizacion_ref=c->>'organizacion_ref' AND x.expediente_ref=c->>'expediente_ref'
  AND x.necesidad_ref=c->>'necesidad_ref' AND x.participacion_ref=c->>'participacion_ref'
 ON CONFLICT (organizacion_ref,expediente_ref,necesidad_ref,participacion_ref)
 DO UPDATE SET version=vec_bolsa_llamamientos.version_comprobacion_reserva_v1.version;
 SELECT * INTO previa FROM vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 x
 WHERE x.organizacion_ref=c->>'organizacion_ref' AND x.expediente_ref=c->>'expediente_ref'
  AND x.necesidad_ref=c->>'necesidad_ref' AND x.participacion_ref=c->>'participacion_ref'
  AND x.clave_idempotencia=p->>'clave_idempotencia';
 es_replay:=FOUND;
 IF vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(c,NOT es_replay) IS NOT TRUE
 THEN RAISE EXCEPTION 'B133: contexto CT divergente' USING ERRCODE='VBC01'; END IF;
 IF es_replay THEN
  IF previa.material_sha256 IS DISTINCT FROM huella
  OR previa.actor_ref IS DISTINCT FROM d->>'principal_id'
  OR previa.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref'
  THEN RAISE EXCEPTION 'B133: clave reutilizada' USING ERRCODE='VBC01'; END IF;
 ELSE
  IF consumo.consumo_nuevo IS NOT TRUE THEN
   RAISE EXCEPTION 'B133: falta efecto del consumo' USING ERRCODE='42501'; END IF;
  PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:politica-ofertas:'||(c->>'bolsa_ref'),0));
  SELECT * INTO STRICT politica FROM vec_bolsa_llamamientos.politica_ofertas_version x
   WHERE x.bolsa_ref=c->>'bolsa_ref' ORDER BY x.version DESC LIMIT 1;
  SELECT * INTO STRICT origen FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
   c->>'categoria_ref',c->>'modalidad_ofrecida',c->>'causa_ofrecida');
  SELECT * INTO entrada FROM jsonb_to_recordset(origen.entradas)
   x(participacion_ref text,orden_acta bigint,orden_vigente bigint,razon text)
   WHERE x.orden_vigente=1 OR x.razon='pendiente_comprobacion_rrhh' ORDER BY x.orden_acta LIMIT 1;
  IF NOT FOUND OR origen.bolsa_ref IS DISTINCT FROM c->>'bolsa_ref'
  OR origen.version_bolsa IS DISTINCT FROM (c->>'version_bolsa')::bigint
  OR origen.version_instantanea IS DISTINCT FROM (c->>'version_instantanea')::bigint
  OR entrada.participacion_ref IS DISTINCT FROM c->>'participacion_ref'
  OR entrada.razon IS DISTINCT FROM 'pendiente_comprobacion_rrhh'
  OR entrada.orden_acta IS DISTINCT FROM (c->>'orden_acta')::bigint
  OR politica.version IS DISTINCT FROM (c->>'version_politica_ofertas')::bigint
  OR politica.huella_sha256 IS DISTINCT FROM c->>'huella_politica_ofertas_sha256'
  THEN RAISE EXCEPTION 'B133: origen o política divergentes' USING ERRCODE='VBC01'; END IF;
  SELECT coalesce(max(x.version),0) INTO version_actual FROM vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 x
   WHERE x.organizacion_ref=c->>'organizacion_ref' AND x.expediente_ref=c->>'expediente_ref'
   AND x.necesidad_ref=c->>'necesidad_ref' AND x.participacion_ref=c->>'participacion_ref';
  IF version_actual<>(p->>'version_esperada')::bigint THEN
   RAISE EXCEPTION 'B133: versión obsoleta' USING ERRCODE='VBC01'; END IF;
  instante:=date_trunc('microseconds',clock_timestamp());
  resultado:=vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(
   p->'hechos',politica.politica->'criterios_ofrecimiento',instante);
  IF p#>>'{hechos,prelacion,valor}' IS NOT NULL
   AND p#>>'{hechos,prelacion,valor}' IS DISTINCT FROM c->>'orden_acta'
  THEN RAISE EXCEPTION 'B133: prelación divergente' USING ERRCODE='VBC01'; END IF;
  SELECT coalesce(jsonb_object_agg(key,value||jsonb_build_object('verificada_por',d->>'principal_id',
   'verificada_en',instante)),'{}') INTO hechos FROM jsonb_each(p->'hechos');
  referencia:='comprobacion:'||encode(sha256(convert_to(
   jsonb_build_array(c,p->>'clave_idempotencia',version_actual+1)::text,'UTF8')),'hex');
  recibo:='recibo:'||referencia;evento:='evento:'||referencia;
  INSERT INTO vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 VALUES(
   referencia,c->>'organizacion_ref',c->>'expediente_ref',c->>'necesidad_ref',c->>'bolsa_ref',c->>'participacion_ref',
   version_actual+1,version_actual,c,politica.politica,hechos,resultado,huella,d->>'principal_id',d->>'perfil_activo_ref',
   p->>'clave_idempotencia',recibo,evento,consumo.decision_ref,consumo.auditoria_ref,instante)
  RETURNING * INTO previa;
  UPDATE vec_bolsa_llamamientos.version_comprobacion_reserva_v1 g
   SET version=version_actual+1
   WHERE g.organizacion_ref=c->>'organizacion_ref' AND g.expediente_ref=c->>'expediente_ref'
    AND g.necesidad_ref=c->>'necesidad_ref' AND g.participacion_ref=c->>'participacion_ref'
    AND g.version=version_actual;
  IF NOT FOUND THEN RAISE EXCEPTION 'B140: versión de control divergente' USING ERRCODE='40001'; END IF;
 END IF;
 -- Recibo mínimo: nunca publica hechos, evidencias ni datos de la relación.
 RETURN jsonb_build_object('comprobacion_ref',previa.comprobacion_ref,'version',previa.version,
  'version_esperada',previa.version_esperada,'resultado',previa.resultado->>'resultado',
  'motivo_clave',previa.resultado->>'motivo_clave','version_politica_ofertas',previa.contexto->'version_politica_ofertas',
  'recibo_ref',previa.recibo_ref,'evento_ref',previa.evento_ref,'auditoria_ref',previa.auditoria_ref,
  'registrada_en',previa.registrada_en,'reutilizada',es_replay);
END $function$;

CREATE FUNCTION vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(p_origen jsonb)
 RETURNS boolean
 LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE e jsonb; capturada jsonb; actual jsonb; contexto jsonb; bolsa record;
        n bigint; version_actual bigint; version_control bigint; seleccionado bigint;
        acto_capturado jsonb; acto_actual jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR jsonb_typeof(p_origen) IS DISTINCT FROM 'object'
 OR jsonb_typeof(p_origen->'lectura') IS DISTINCT FROM 'object'
 OR jsonb_typeof(p_origen#>'{lectura,entradas}') IS DISTINCT FROM 'array'
 OR jsonb_array_length(p_origen#>'{lectura,entradas}') NOT BETWEEN 1 AND 100000
 OR jsonb_typeof(p_origen#>'{lectura,criterios_ofrecimiento}') IS DISTINCT FROM 'object'
 OR p_origen->'criterios_ofrecimiento' IS DISTINCT FROM p_origen#>'{lectura,criterios_ofrecimiento}'
 -- La reserva singular B110 trae el orden del acta; la oferta CT220 no
 -- selecciona persona y solo recoteja contexto y actos.
 OR (p_origen ? 'orden_seleccionado' AND (
   jsonb_typeof(p_origen->'orden_seleccionado') IS DISTINCT FROM 'number'
   OR p_origen->>'orden_seleccionado' !~ '^[1-9][0-9]{0,8}$'))
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(p_origen#>'{lectura,entradas}') WITH ORDINALITY x(valor,n)
   WHERE jsonb_typeof(x.valor) IS DISTINCT FROM 'object'
     OR x.valor->>'orden_acta' IS DISTINCT FROM x.n::text)
 THEN RAISE EXCEPTION 'B140: origen contextual inválido' USING ERRCODE='22023'; END IF;
 seleccionado:=(p_origen->>'orden_seleccionado')::bigint;
 -- B133 usa esta misma llave antes de leer CT y la política. La actualización
 -- fuerza 40001 si un acto se confirmó después del snapshot SERIALIZABLE.
 FOR e IN SELECT x.valor FROM jsonb_array_elements(p_origen#>'{lectura,entradas}') WITH ORDINALITY x(valor,n)
  WHERE x.valor->>'razon'='pendiente_comprobacion_rrhh'
  ORDER BY x.n LOOP
  IF e->>'participacion_ref' IS NULL OR e->>'orden_acta' !~ '^[1-9][0-9]{0,8}$'
  THEN RAISE EXCEPTION 'B140: entrada inválida' USING ERRCODE='22023'; END IF;
  PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:comprobacion:'||
   jsonb_build_array(p_origen->>'organizacion_ref',p_origen->>'expediente_ref',
    p_origen->>'necesidad_ref',e->>'participacion_ref')::text,0));
  INSERT INTO vec_bolsa_llamamientos.version_comprobacion_reserva_v1(
   organizacion_ref,expediente_ref,necesidad_ref,participacion_ref,version)
  SELECT p_origen->>'organizacion_ref',p_origen->>'expediente_ref',
   p_origen->>'necesidad_ref',e->>'participacion_ref',coalesce(max(x.version),0)
  FROM vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 x
  WHERE x.organizacion_ref=p_origen->>'organizacion_ref'
   AND x.expediente_ref=p_origen->>'expediente_ref'
   AND x.necesidad_ref=p_origen->>'necesidad_ref'
   AND x.participacion_ref=e->>'participacion_ref'
  ON CONFLICT (organizacion_ref,expediente_ref,necesidad_ref,participacion_ref)
  DO UPDATE SET version=vec_bolsa_llamamientos.version_comprobacion_reserva_v1.version;
  SELECT g.version INTO STRICT version_control
  FROM vec_bolsa_llamamientos.version_comprobacion_reserva_v1 g
  WHERE g.organizacion_ref=p_origen->>'organizacion_ref'
   AND g.expediente_ref=p_origen->>'expediente_ref'
   AND g.necesidad_ref=p_origen->>'necesidad_ref'
   AND g.participacion_ref=e->>'participacion_ref';
  SELECT coalesce(max(x.version),0) INTO version_actual
  FROM vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 x
  WHERE x.organizacion_ref=p_origen->>'organizacion_ref'
   AND x.expediente_ref=p_origen->>'expediente_ref'
   AND x.necesidad_ref=p_origen->>'necesidad_ref'
   AND x.participacion_ref=e->>'participacion_ref';
  IF version_control IS DISTINCT FROM version_actual
  THEN RAISE EXCEPTION 'B140: acto concurrente' USING ERRCODE='40001'; END IF;
 END LOOP;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended(
  'vec_bolsa_llamamientos:politica-ofertas:'||(p_origen->>'bolsa_ref'),0));
 contexto:=jsonb_build_object(
  'ambito_ref',(vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1())->>'ambito_ref',
  'organizacion_ref',p_origen->>'organizacion_ref','expediente_ref',p_origen->>'expediente_ref',
  'necesidad_ref',p_origen->>'necesidad_ref','unidad_ref',p_origen->>'unidad_ref',
  'categoria_ref',p_origen->>'categoria_ref','version_expediente',p_origen->'version_expediente',
  'modalidad_ofrecida',p_origen#>>'{criterios_ofrecimiento,modalidad_ofrecida}',
  'causa_ofrecida',p_origen#>>'{criterios_ofrecimiento,causa_ofrecida}');
 SELECT * INTO STRICT bolsa FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  contexto->>'categoria_ref',contexto->>'modalidad_ofrecida',contexto->>'causa_ofrecida',contexto);
 IF bolsa.acta_ref IS DISTINCT FROM p_origen->>'acta_ref'
 OR bolsa.bolsa_ref IS DISTINCT FROM p_origen->>'bolsa_ref'
 OR bolsa.version_bolsa IS DISTINCT FROM (p_origen->>'version_bolsa')::bigint
 OR bolsa.version_instantanea IS DISTINCT FROM (p_origen->>'version_instantanea')::bigint
 OR bolsa.version_politica IS DISTINCT FROM (p_origen->>'version_politica')::bigint
 OR bolsa.criterios_ofrecimiento IS DISTINCT FROM p_origen->'criterios_ofrecimiento'
 OR jsonb_array_length(bolsa.entradas) IS DISTINCT FROM jsonb_array_length(p_origen#>'{lectura,entradas}')
 THEN RAISE EXCEPTION 'B140: contexto cambiado' USING ERRCODE='55000'; END IF;
 FOR capturada,actual IN SELECT c.value,a.value
  FROM jsonb_array_elements(p_origen#>'{lectura,entradas}') WITH ORDINALITY c(value,n)
  JOIN jsonb_array_elements(bolsa.entradas) WITH ORDINALITY a(value,n) USING(n) LOOP
  IF capturada->>'orden_acta' IS DISTINCT FROM actual->>'orden_acta'
  OR capturada->>'participacion_ref' IS DISTINCT FROM actual->>'participacion_ref'
  OR capturada->>'candidato_ref' IS DISTINCT FROM coalesce(actual->>'candidato_ref','')
  OR capturada->>'orden_vigente' IS DISTINCT FROM actual->>'orden_vigente'
  OR capturada->>'situacion' IS DISTINCT FROM actual->>'situacion'
  OR capturada->>'razon' IS DISTINCT FROM actual->>'razon'
  OR capturada->'eventos_relacion' IS DISTINCT FROM actual->'eventos_relacion'
  OR (capturada ? 'comprobacion_ofrecimiento') IS DISTINCT FROM
     (actual ? 'comprobacion_ofrecimiento')
  THEN RAISE EXCEPTION 'B140: acto o selección cambiados' USING ERRCODE='55000'; END IF;
  -- CT conserva la lectura tras pasar por su adaptador: el instante del acto
  -- se compara como instante, no por su forma textual.
  acto_capturado:=capturada->'comprobacion_ofrecimiento';
  acto_actual:=actual->'comprobacion_ofrecimiento';
  IF acto_capturado ? 'registrada_en' THEN
   IF jsonb_typeof(acto_capturado->'registrada_en') IS DISTINCT FROM 'string'
   OR acto_capturado->>'registrada_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:.]+(Z|[+-][0-9]{2}:[0-9]{2})$'
   THEN RAISE EXCEPTION 'B140: acto o selección cambiados' USING ERRCODE='55000'; END IF;
   acto_capturado:=jsonb_set(acto_capturado,'{registrada_en}',
    to_jsonb((acto_capturado->>'registrada_en')::timestamptz));
  END IF;
  IF acto_actual ? 'registrada_en' THEN
   acto_actual:=jsonb_set(acto_actual,'{registrada_en}',
    to_jsonb((acto_actual->>'registrada_en')::timestamptz));
  END IF;
  IF acto_capturado IS DISTINCT FROM acto_actual
  THEN RAISE EXCEPTION 'B140: acto o selección cambiados' USING ERRCODE='55000'; END IF;
  IF seleccionado IS NOT NULL AND actual->>'razon'='pendiente_comprobacion_rrhh'
     AND (actual->>'orden_acta')::bigint<=seleccionado
  THEN RAISE EXCEPTION 'B140: comprobación pendiente' USING ERRCODE='55000'; END IF;
 END LOOP;
 RETURN true;
END $function$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(jsonb) TO vec_contratacion_temporal_propietario;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.reservar_llamamiento_ct_v1(p_llamamiento text, p_recibo text, p_bolsa text, p_actor text, p_clave text, p_participaciones jsonb, p_configuracion jsonb, p_emitido timestamp with time zone, p_huella_finalizacion bytea, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea, p_origen jsonb)
 RETURNS TABLE(emision jsonb, reutilizada boolean)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE
 previo vec_bolsa_llamamientos.llamamiento_emitido%ROWTYPE;
 origen_previo vec_bolsa_llamamientos.origen_llamamiento_ct_v1%ROWTYPE;
 consumo record;
 d jsonb;
 v_huella text;
 v_origen_huella text;
 v_evento text;
 v_canon_politica bytea;
 v_confirmacion jsonb;
 v_cuantos bigint;
 v_lectura jsonb;
 v_entrada jsonb;
 v_ct record;
 v_politica_oferta record;
 v_criterios jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_llamamiento !~ '^llamamiento:[0-9a-f]{64}$'
    OR p_recibo !~ '^recibo:llamamiento:[0-9a-f]{64}$'
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave<>btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_emitido IS NULL OR octet_length(p_huella_finalizacion) IS DISTINCT FROM 32
    OR jsonb_typeof(p_participaciones)<>'array' OR jsonb_array_length(p_participaciones)<>1
    OR jsonb_typeof(p_participaciones->0)<>'string'
    OR jsonb_typeof(p_configuracion)<>'object' OR
       p_configuracion ?& ARRAY['referencia','descripcion','categoria','centro','modalidad',
       'fecha_inicio','plazo','plantilla_version','asunto','cuerpo'] IS NOT TRUE
    OR (SELECT count(*)<>10 OR bool_or(jsonb_typeof(value)<>'string'
      OR octet_length(value#>>'{}') NOT BETWEEN 2 AND 4000
      OR value#>>'{}'<>btrim(value#>>'{}')) FROM jsonb_each(p_configuracion))
    OR p_configuracion->>'plantilla_version' !~ '^bolsa-llamamiento-v[1-9][0-9]{0,3}$'
    OR jsonb_typeof(p_origen)<>'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_origen)) NOT IN (22,23)
    OR ((SELECT count(*) FROM jsonb_object_keys(p_origen))=23
        AND NOT (p_origen ? 'criterios_ofrecimiento'))
    OR p_origen ?& ARRAY['organizacion_ref','unidad_ref','expediente_ref','version_expediente','clave_ct',
      'necesidad_ref','acta_ref','categoria_ref','bolsa_ref','version_bolsa','huella_bolsa_sha256',
      'instantanea_ref','version_instantanea','huella_instantanea_sha256','politica_ref',
      'version_politica','huella_politica_sha256','participacion_ref','candidato_ref','orden_seleccionado','lectura','version_token'] IS NOT TRUE
    OR p_origen->>'bolsa_ref' IS DISTINCT FROM p_bolsa
    OR p_origen->>'participacion_ref' IS DISTINCT FROM p_participaciones->>0
    OR p_origen->>'candidato_ref' !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_origen->>'clave_ct' IS NULL OR p_origen->>'expediente_ref' IS NULL
    OR p_origen->>'organizacion_ref' IS NULL OR p_origen->>'necesidad_ref' IS NULL
    OR p_origen->>'unidad_ref' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    OR p_origen->>'version_token' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 THEN RAISE EXCEPTION 'B110: reserva CT inválida' USING ERRCODE='22023'; END IF;
 v_lectura:=p_origen->'lectura';
 v_criterios:=p_origen->'criterios_ofrecimiento';
 IF jsonb_typeof(v_lectura)<>'object'
    OR v_lectura->>'acta_ref' IS DISTINCT FROM p_origen->>'acta_ref'
    OR v_lectura->>'categoria_ref' IS DISTINCT FROM p_origen->>'categoria_ref'
    OR v_lectura->>'bolsa_ref' IS DISTINCT FROM p_bolsa
    OR v_lectura->>'version_bolsa' IS DISTINCT FROM p_origen->>'version_bolsa'
    OR v_lectura->>'huella_bolsa_sha256' IS DISTINCT FROM p_origen->>'huella_bolsa_sha256'
    OR v_lectura->>'instantanea_ref' IS DISTINCT FROM p_origen->>'instantanea_ref'
    OR v_lectura->>'version_instantanea' IS DISTINCT FROM p_origen->>'version_instantanea'
    OR v_lectura->>'huella_instantanea_sha256' IS DISTINCT FROM p_origen->>'huella_instantanea_sha256'
    OR v_lectura->>'politica_ref' IS DISTINCT FROM p_origen->>'politica_ref'
    OR v_lectura->>'version_politica' IS DISTINCT FROM p_origen->>'version_politica'
    OR v_lectura->>'huella_politica_sha256' IS DISTINCT FROM p_origen->>'huella_politica_sha256'
    OR jsonb_typeof(v_lectura->'politica_provisional') IS DISTINCT FROM 'boolean'
    OR jsonb_typeof(v_lectura->'entradas')<>'array'
    OR v_lectura->'criterios_ofrecimiento' IS DISTINCT FROM v_criterios
    OR (v_criterios IS NOT NULL AND (jsonb_typeof(v_criterios) IS DISTINCT FROM 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(v_criterios))<>5
       OR v_criterios ?& ARRAY['version_politica_ofertas',
          'huella_politica_ofertas_sha256','modalidad_ofrecida',
          'causa_ofrecida','configuracion'] IS NOT TRUE
       OR v_criterios->>'huella_politica_ofertas_sha256' !~ '^[0-9a-f]{64}$'
       OR jsonb_typeof(v_criterios->'version_politica_ofertas') IS DISTINCT FROM 'number'
       OR v_criterios->>'version_politica_ofertas' !~ '^[1-9][0-9]{0,17}$'
       OR jsonb_typeof(v_criterios->'causa_ofrecida') IS DISTINCT FROM 'string'
       OR (v_criterios->>'causa_ofrecida'<>''
         AND v_criterios->>'causa_ofrecida' !~ '^[a-z][a-z0-9._-]{1,79}$')
       OR vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(
          v_criterios->'configuracion') IS NOT TRUE))
    OR (v_lectura->>'corte')::timestamptz>p_emitido
    OR encode(sha256(decode(v_lectura->>'bolsa_canonica','base64')),'hex')
       IS DISTINCT FROM p_origen->>'huella_bolsa_sha256'
    OR encode(sha256(decode(v_lectura->>'instantanea_canonica','base64')),'hex')
       IS DISTINCT FROM p_origen->>'huella_instantanea_sha256'
 THEN RAISE EXCEPTION 'B110: lectura CT incompatible' USING ERRCODE='22023'; END IF;
 SELECT x.value INTO v_entrada FROM jsonb_array_elements(v_lectura->'entradas') x(value)
 WHERE x.value->>'participacion_ref'=p_origen->>'participacion_ref'
   AND x.value->>'candidato_ref'=p_origen->>'candidato_ref'
   AND x.value->>'orden_acta'=p_origen->>'orden_seleccionado'
   AND x.value->>'elegible'='true' AND x.value->>'orden_vigente'='1';
 IF NOT FOUND THEN RAISE EXCEPTION 'B110: selección no pertenece a lectura' USING ERRCODE='22023'; END IF;
 -- La validación de la lista y del orden se hace con la política efectiva B6.
 v_huella:=encode(sha256(convert_to(p_bolsa||chr(31)||p_participaciones::text||chr(31)||p_configuracion::text,'UTF8')),'hex');
 v_origen_huella:=encode(sha256(convert_to(p_origen::text,'UTF8')),'hex');
 v_evento:='evento:bolsa:ct:'||encode(sha256(convert_to((p_origen->>'expediente_ref')||chr(31)||(p_origen->>'clave_ct'),'UTF8')),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:emision:'||p_bolsa||':'||p_clave,0));
 SELECT * INTO previo FROM vec_bolsa_llamamientos.llamamiento_emitido
  WHERE bolsa_ref=p_bolsa AND clave_idempotencia=p_clave FOR UPDATE;
 IF FOUND THEN
  SELECT * INTO origen_previo FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1
   WHERE expediente_ref=p_origen->>'expediente_ref' AND clave_ct=p_origen->>'clave_ct';
  IF NOT FOUND OR previo.actor_ref<>p_actor OR previo.huella_comando_sha256<>v_huella
     OR previo.llamamiento_ref<>p_llamamiento OR previo.recibo_ref<>p_recibo
     OR origen_previo.origen_huella_sha256<>v_origen_huella
     OR origen_previo.llamamiento_ref<>previo.llamamiento_ref THEN
   RAISE EXCEPTION 'B110: clave CT divergente' USING ERRCODE='VBE01'; END IF;
  RETURN QUERY SELECT vec_bolsa_llamamientos.recuperar_llamamiento_emitido_v1(p_bolsa,p_clave)
    || jsonb_build_object('origen_ct',vec_bolsa_llamamientos.recuperar_origen_llamamiento_ct_v1(
      p_origen->>'expediente_ref',p_origen->>'clave_ct')),true;
  RETURN;
 END IF;
 IF v_criterios IS NOT NULL THEN
  PERFORM vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(p_origen);
 END IF;
 -- Solo una reserva nueva revalida modalidad y versión del expediente. El
 -- replay anterior ya conservó su origen y vuelve sin reinterpretar hechos.
 SELECT * INTO STRICT v_ct FROM vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(
  p_origen->>'expediente_ref',p_origen->>'categoria_ref',
  p_origen->>'organizacion_ref',(p_origen->>'version_expediente')::bigint);
 IF v_ct.modalidad_clave IS DISTINCT FROM p_configuracion->>'modalidad'
 THEN RAISE EXCEPTION 'B118: modalidad del expediente divergente' USING ERRCODE='23503'; END IF;
 IF v_criterios IS NOT NULL AND v_criterios->>'causa_ofrecida' IS DISTINCT FROM v_ct.causa_clave
 THEN RAISE EXCEPTION 'B118: causa del expediente divergente' USING ERRCODE='23503'; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended(
  'vec_bolsa_llamamientos:politica-ofertas:'||p_bolsa,0));
 SELECT po.version,po.huella_sha256,po.politica->'criterios_ofrecimiento' AS criterios
 INTO v_politica_oferta FROM vec_bolsa_llamamientos.politica_ofertas_version po
 WHERE po.bolsa_ref=p_bolsa ORDER BY po.version DESC LIMIT 1;
 IF FOUND AND v_politica_oferta.criterios IS NOT NULL
    AND v_politica_oferta.criterios<>'{}'::jsonb THEN
  IF v_criterios IS NULL
   OR v_criterios->>'version_politica_ofertas' IS DISTINCT FROM v_politica_oferta.version::text
   OR v_criterios->>'huella_politica_ofertas_sha256' IS DISTINCT FROM v_politica_oferta.huella_sha256
   OR v_criterios->'configuracion' IS DISTINCT FROM v_politica_oferta.criterios
   OR v_criterios->>'modalidad_ofrecida' IS DISTINCT FROM v_ct.modalidad_clave
   OR v_criterios->>'causa_ofrecida' IS DISTINCT FROM v_ct.causa_clave
  THEN RAISE EXCEPTION 'B118: criterio de oferta divergente' USING ERRCODE='23503'; END IF;
 ELSIF v_criterios IS NOT NULL THEN
  RAISE EXCEPTION 'B118: criterio de oferta ausente' USING ERRCODE='23503';
 END IF;
 SELECT count(*) INTO v_cuantos
 FROM vec_bolsa_llamamientos.constitucion c
 JOIN vec_bolsa_llamamientos.bolsa_constituida b ON b.bolsa_ref=c.bolsa_ref
   AND b.version=c.version_bolsa AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
 JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i ON i.instantanea_ref=c.instantanea_ref
   AND i.version=c.version_instantanea AND i.huella_instantanea_sha256=c.huella_instantanea_sha256
 JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=c.instantanea_ref
   AND e.version_instantanea=c.version_instantanea
 JOIN vec_bolsa_llamamientos.vinculo_candidato vc ON vc.participacion_ref=e.participacion_ref
   AND vc.acta_ref=c.acta_ref AND vc.instantanea_ref=c.instantanea_ref
   AND vc.version_instantanea=c.version_instantanea
 JOIN vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(p_bolsa,v_ct.modalidad_clave,v_ct.causa_clave,p_emitido) o
   ON o.participacion_ref=e.participacion_ref AND o.orden_acta=e.orden
 JOIN vec_bolsa_llamamientos.politica_orden_bolsa pol
   ON pol.politica_ref=o.politica_ref AND pol.version=o.version_politica
 WHERE c.acta_ref=p_origen->>'acta_ref' AND c.categoria_ref=p_origen->>'categoria_ref'
   AND c.bolsa_ref=p_bolsa AND c.version_bolsa=(p_origen->>'version_bolsa')::bigint
   AND c.huella_bolsa_sha256=p_origen->>'huella_bolsa_sha256'
   AND c.instantanea_ref=p_origen->>'instantanea_ref'
   AND c.version_instantanea=(p_origen->>'version_instantanea')::bigint
   AND c.huella_instantanea_sha256=p_origen->>'huella_instantanea_sha256'
   AND e.participacion_ref=p_origen->>'participacion_ref'
   AND e.orden=(p_origen->>'orden_seleccionado')::bigint
   AND vc.candidato_ref=p_origen->>'candidato_ref'
   AND o.orden_vigente=1
   AND o.politica_ref=p_origen->>'politica_ref'
   AND o.version_politica=(p_origen->>'version_politica')::bigint
   AND pol.provisional=(v_lectura->>'politica_provisional')::boolean
   AND encode(sha256(convert_to(to_jsonb(pol)::text,'UTF8')),'hex')=p_origen->>'huella_politica_sha256'
   AND b.estado='vigente' AND b.vigente_desde<=p_emitido
   AND (b.vigente_hasta IS NULL OR p_emitido<b.vigente_hasta)
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa s
     WHERE s.bolsa_ref_sustituida=c.bolsa_ref AND s.version_sustituida=c.version_bolsa
       AND s.huella_sustituida_sha256=c.huella_bolsa_sha256);
 IF v_cuantos<>1 THEN RAISE EXCEPTION 'B110: origen CT no vigente o divergente' USING ERRCODE='23503'; END IF;
 -- La decisión completa que se conservará para replay se coteja con el
 -- orden B6 al corte original; ninguna entrada puede sustituirse en tránsito.
 SELECT count(*) INTO v_cuantos
 FROM jsonb_array_elements(v_lectura->'entradas') WITH ORDINALITY x(valor,n)
 JOIN vec_bolsa_llamamientos.constitucion_entrada e
   ON e.instantanea_ref=p_origen->>'instantanea_ref'
  AND e.version_instantanea=(p_origen->>'version_instantanea')::bigint
  AND e.orden=x.n AND e.participacion_ref=x.valor->>'participacion_ref'
 JOIN vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(
   p_bolsa,v_ct.modalidad_clave,v_ct.causa_clave,(v_lectura->>'corte')::timestamptz) o
   ON o.participacion_ref=e.participacion_ref AND o.orden_acta=e.orden
 LEFT JOIN vec_bolsa_llamamientos.vinculo_candidato vc
   ON vc.participacion_ref=e.participacion_ref AND vc.acta_ref=p_origen->>'acta_ref'
 WHERE x.valor->>'orden_acta'=e.orden::text
   AND x.valor->>'orden_vigente' IS NOT DISTINCT FROM o.orden_vigente::text
   AND x.valor->>'situacion' IS NOT DISTINCT FROM o.situacion
   AND x.valor->>'razon' IS NOT DISTINCT FROM o.razon
   -- Una lectura sin criterios (B110 general) no trae eventos; con criterios
   -- publicados la lectura contextual los trae siempre y se cotejan.
   AND CASE WHEN x.valor ? 'eventos_relacion'
        THEN x.valor->'eventos_relacion' IS NOT DISTINCT FROM o.eventos_relacion
        ELSE v_criterios IS NULL END
   AND x.valor->>'candidato_ref' IS NOT DISTINCT FROM coalesce(vc.candidato_ref,'')
   AND x.valor->>'elegible'=(o.orden_vigente IS NOT NULL)::text
   AND o.politica_ref=p_origen->>'politica_ref'
   AND o.version_politica=(p_origen->>'version_politica')::bigint
   AND o.provisional=(v_lectura->>'politica_provisional')::boolean;
 IF v_cuantos<>jsonb_array_length(v_lectura->'entradas') THEN
  RAISE EXCEPTION 'B110: orden completo divergente' USING ERRCODE='23503'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B110: decisión inválida' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR consumo.consumo_nuevo IS NOT TRUE
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' THEN
  RAISE EXCEPTION 'B110: emisión no autorizada' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(
  llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,
  participaciones,configuracion,huella_comando_sha256,huella_finalizacion,
  estado,emitido_en,decision_ref)
 VALUES(p_llamamiento,p_recibo,p_bolsa,p_actor,p_clave,p_participaciones,
  p_configuracion,v_huella,p_huella_finalizacion,'emision_reservada',p_emitido,consumo.decision_ref);
 INSERT INTO vec_bolsa_llamamientos.origen_llamamiento_ct_v1(
  organizacion_ref,unidad_ref,expediente_ref,version_expediente,clave_ct,necesidad_ref,
  acta_ref,categoria_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,
  version_instantanea,huella_instantanea_sha256,politica_ref,version_politica,
  huella_politica_sha256,participacion_ref,candidato_ref,orden_seleccionado,version_token,
  llamamiento_ref,recibo_ref,decision_ref,auditoria_ref,evento_ref,emitido_en,
  origen_canonico,origen_huella_sha256)
 VALUES(p_origen->>'organizacion_ref',p_origen->>'unidad_ref',p_origen->>'expediente_ref',
  (p_origen->>'version_expediente')::bigint,p_origen->>'clave_ct',p_origen->>'necesidad_ref',
  p_origen->>'acta_ref',p_origen->>'categoria_ref',p_bolsa,(p_origen->>'version_bolsa')::bigint,
  p_origen->>'huella_bolsa_sha256',p_origen->>'instantanea_ref',
  (p_origen->>'version_instantanea')::bigint,p_origen->>'huella_instantanea_sha256',
  p_origen->>'politica_ref',(p_origen->>'version_politica')::bigint,
  p_origen->>'huella_politica_sha256',p_origen->>'participacion_ref',
  p_origen->>'candidato_ref',(p_origen->>'orden_seleccionado')::bigint,p_origen->>'version_token',
  p_llamamiento,p_recibo,consumo.decision_ref,consumo.auditoria_ref,v_evento,
  p_emitido,p_origen,v_origen_huella);
 INSERT INTO vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1
   (evento_ref,llamamiento_ref,tipo,estado,creada_en)
 VALUES(v_evento,p_llamamiento,'bolsa.llamamiento.ct.origen_confirmado','pendiente',p_emitido);
 v_confirmacion:=vec_bolsa_llamamientos.recuperar_origen_llamamiento_ct_v1(
  p_origen->>'expediente_ref',p_origen->>'clave_ct') || jsonb_build_object('reutilizada',false);
 RETURN QUERY SELECT jsonb_build_object('llamamiento_ref',p_llamamiento,'recibo_ref',p_recibo,
  'bolsa_ref',p_bolsa,'estado','emision_reservada_resultado_pendiente',
  'participaciones',p_participaciones,'configuracion',p_configuracion,
  'emitido_en',p_emitido,'contactos','[]'::jsonb,'origen_ct',v_confirmacion),false;
END $function$;

DO $post$ DECLARE p record; acl_esperada text; huella_esperada text; BEGIN
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.version_comprobacion_reserva_v1)<>0
 THEN RAISE EXCEPTION 'B140: control inicial inesperado' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=
   'vec_bolsa_llamamientos.version_comprobacion_reserva_v1'::regclass
   AND c.relowner='vec_bolsa_llamamientos_propietario'::regrole
   AND c.relrowsecurity AND c.relforcerowsecurity)
 OR has_table_privilege('vec_bolsa_llamamientos_ejecutor',
   'vec_bolsa_llamamientos.version_comprobacion_reserva_v1','SELECT,INSERT,UPDATE,DELETE')
 OR has_table_privilege('vec_contratacion_temporal_propietario',
   'vec_bolsa_llamamientos.version_comprobacion_reserva_v1','SELECT,INSERT,UPDATE,DELETE')
 OR (SELECT count(*) FROM pg_policy pol WHERE pol.polrelid=
   'vec_bolsa_llamamientos.version_comprobacion_reserva_v1'::regclass
   AND pol.polname='version_comprobacion_propietario' AND pol.polcmd='*'
   AND pol.polpermissive AND pol.polroles=ARRAY['vec_bolsa_llamamientos_propietario'::regrole]::oid[]
   AND pol.polqual IS NOT NULL AND pol.polwithcheck IS NOT NULL)<>1
 OR (SELECT count(*) FROM pg_policy pol WHERE pol.polrelid=
   'vec_bolsa_llamamientos.version_comprobacion_reserva_v1'::regclass)<>1
 THEN RAISE EXCEPTION 'B140: tabla o RLS incompatible' USING ERRCODE='55000'; END IF;
 FOR p IN SELECT q.oid,q.proowner,q.prosecdef,q.proacl::text AS acl,q.proconfig,
  encode(sha256(convert_to(pg_get_functiondef(q.oid),'UTF8')),'hex') AS huella
  FROM pg_proc q WHERE q.oid IN (
   'vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text,jsonb)'::regprocedure,
   'vec_bolsa_llamamientos.registrar_comprobacion_ofrecimiento_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
   'vec_bolsa_llamamientos.reservar_llamamiento_ct_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)'::regprocedure,
   'vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(jsonb)'::regprocedure)
 LOOP
  acl_esperada:=CASE p.oid
    WHEN 'vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text,jsonb)'::regprocedure
     THEN '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_bolsa_llamamientos_propietario}'
    WHEN 'vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(jsonb)'::regprocedure
     THEN '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_contratacion_temporal_propietario=X/vec_bolsa_llamamientos_propietario}'
    ELSE '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}' END;
  huella_esperada:=CASE p.oid
    WHEN 'vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text,jsonb)'::regprocedure
     THEN 'cc6c1fef7cff07fdcca765328557ddc44908af04de869c739fc98691f521bd4a'
    WHEN 'vec_bolsa_llamamientos.registrar_comprobacion_ofrecimiento_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
     THEN '4abd1757b2465812fb7f08c06f5c8e8702d70ef7b16c6bebcfadc94fdd3e433c'
    WHEN 'vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(jsonb)'::regprocedure
     THEN '8eb02751df9746a5f687b58a2dfe2d58b5fb6f6c2bc02cf5aebd65873d6c2677'
    ELSE 'f5035c9e1f83ae60a2cdc8d7b9a38232dfad207a1c0eff11a90cbfb0ffa56bc4' END;
  IF pg_get_userbyid(p.proowner)<>'vec_bolsa_llamamientos_propietario'
   OR p.prosecdef IS NOT TRUE
   OR (SELECT array_agg(x ORDER BY x COLLATE "C") FROM unnest(p.proconfig) x) IS DISTINCT FROM
     ARRAY['TimeZone=UTC','lock_timeout=2s','row_security=on','search_path=pg_catalog, pg_temp',
      'statement_timeout=15s']
   OR p.acl IS DISTINCT FROM acl_esperada
   OR p.huella IS DISTINCT FROM huella_esperada
  THEN RAISE EXCEPTION 'B140: postimagen incompatible %',p.oid USING ERRCODE='55000'; END IF;
 END LOOP;
END $post$;
COMMIT;
