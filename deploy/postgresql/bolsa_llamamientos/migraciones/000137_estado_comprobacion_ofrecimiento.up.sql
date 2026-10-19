\set ON_ERROR_STOP on
-- B137 / F1-09: misma lectura contextual, estado mínimo del acto comprobado.
-- Secuencia: B133 y CT233 instaladas; AD293 después, detrás de AD288.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000137',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_bolsa_llamamientos_propietario'
 OR to_regclass('vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1') IS NULL
 OR to_regprocedure('vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(jsonb,boolean)') IS NULL
 OR to_regprocedure('vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1()') IS NULL
 OR has_function_privilege('vec_bolsa_llamamientos_propietario',
   'vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1()','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_bolsa_llamamientos_propietario',
   'vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(jsonb,boolean)','EXECUTE') IS NOT TRUE
 OR encode(sha256(convert_to(pg_get_functiondef(
   'vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(jsonb,boolean)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM 'fd7f4251e849feffe45c0ad3638c2e71c1524c9a6523f17858d70c16beb5cdad'
 OR to_regprocedure('vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text,jsonb)') IS NOT NULL
 OR encode(sha256(convert_to(pg_get_functiondef(
   'vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM 'e6dc233b8302fd622d7f2960431bdab88d59f7b0668b06348ff577e4e8558633'
 OR (SELECT p.proacl::text FROM pg_proc p WHERE p.oid=
   'vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text)'::regprocedure)
   IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_bolsa_llamamientos_propietario}'
 THEN RAISE EXCEPTION 'B137: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
DROP FUNCTION vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text) RESTRICT;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(p_categoria text, p_modalidad text, p_causa text, p_contexto jsonb DEFAULT NULL::jsonb)
 RETURNS TABLE(acta_ref text, categoria_ref text, bolsa_ref text, version_bolsa bigint, huella_bolsa_sha256 text, bolsa_canonica bytea, instantanea_ref text, version_instantanea bigint, huella_instantanea_sha256 text, instantanea_canonica bytea, politica_ref text, version_politica bigint, politica_canonica bytea, politica_provisional boolean, corte timestamp with time zone, bolsa_vigente_hasta timestamp with time zone, politica_vigente_hasta timestamp with time zone, entradas jsonb, criterios_ofrecimiento jsonb)
 LANGUAGE plpgsql
 VOLATILE SECURITY DEFINER
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
  SELECT jsonb_agg(e.valor||jsonb_build_object('comprobacion_ofrecimiento',
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
       'version_politica_ofertas',v_oferta.version)) END) ORDER BY e.n)
  INTO v_entradas
  FROM jsonb_array_elements(v_entradas) WITH ORDINALITY e(valor,n)
  LEFT JOIN LATERAL (
   SELECT x.* FROM vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 x
   WHERE x.organizacion_ref=p_contexto->>'organizacion_ref'
     AND x.expediente_ref=p_contexto->>'expediente_ref'
     AND x.necesidad_ref=p_contexto->>'necesidad_ref'
     AND x.bolsa_ref=v_const.bolsa_ref
     AND x.participacion_ref=e.valor->>'participacion_ref'
   ORDER BY x.version DESC LIMIT 1) a ON true;
  IF v_entradas IS NULL THEN RAISE EXCEPTION 'B137: proyección incompleta' USING ERRCODE='55000'; END IF;
 END IF;
 RETURN QUERY SELECT v_const.acta_ref,v_const.categoria_ref,v_const.bolsa_ref,v_const.version_bolsa,
   v_const.huella_bolsa_sha256,v_const.bolsa_canonica,v_const.instantanea_ref,
   v_const.version_instantanea,v_const.huella_instantanea_sha256,v_const.instantanea_canonica,
   v_politica.politica_ref,v_politica.version,convert_to(to_jsonb(v_politica)::text,'UTF8'),
   v_politica.provisional,v_corte,v_const.bolsa_vigente_hasta,v_politica.vigente_hasta,v_entradas,v_criterios;
END $function$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text,jsonb)
 TO vec_autorizacion_atestada_v3_propietario;
DO $post$ DECLARE p record; BEGIN
 SELECT pg_get_userbyid(x.proowner) AS dueno,x.prosecdef,x.provolatile,x.proacl::text AS acl,
   x.proconfig,encode(sha256(convert_to(pg_get_functiondef(x.oid),'UTF8')),'hex') AS huella
 INTO STRICT p FROM pg_proc x WHERE x.oid=
  'vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text,jsonb)'::regprocedure;
 IF p.dueno<>'vec_bolsa_llamamientos_propietario' OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v'
 OR p.acl IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_bolsa_llamamientos_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']
 OR p.huella IS DISTINCT FROM '401b1e9430d2d7c5be462ba733abb6378277b45d2224d1a554768fde93b9afac'
 THEN RAISE EXCEPTION 'B137: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
