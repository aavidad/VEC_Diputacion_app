\set ON_ERROR_STOP on
-- B58: reversión estructural solo sin historia de plazas (ni ofertas con
-- número de plazas, ni actos, ni políticas con el apartado «plazas»).
-- Nunca se ejecuta DOWN sobre historia conservada.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000058',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.plazas_oferta') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.acto_plaza_oferta') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.acto_plaza_oferta_outbox') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_acto_plaza_oferta_v1(text,text,text,integer,text,text,integer,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    -- Solo revierte los cuerpos que instaló B58, nunca los de una migración posterior.
    OR position('proyectar_oferta_v2' in pg_get_functiondef('vec_bolsa_llamamientos.listar_ofertas_bolsa_v1(text,timestamptz,integer)'::regprocedure))=0
    OR position('acto_plaza_oferta' in pg_get_functiondef('vec_bolsa_llamamientos.listar_ofertas_candidato_v1(text,timestamptz)'::regprocedure))=0
    OR position('plazas' in pg_get_functiondef('vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))=0
 THEN RAISE EXCEPTION 'B58: preimagen incompatible para DOWN' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.plazas_oferta)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.acto_plaza_oferta)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.politica_ofertas_version WHERE politica ? 'plazas')
 THEN RAISE EXCEPTION 'B58: historia de plazas impide DOWN' USING ERRCODE='55000'; END IF;
END $pre$;

DROP FUNCTION vec_bolsa_llamamientos.registrar_acto_plaza_oferta_v1(text,text,text,integer,text,text,integer,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.publicar_oferta_v3(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer);

-- Consultas y publicador de política tal como los dejaron B28, B29 y B54.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.listar_ofertas_bolsa_v1(p_bolsa text, p_corte timestamptz, p_limite integer)
RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
 SELECT coalesce(jsonb_agg(vec_bolsa_llamamientos.proyectar_oferta_v1(x.oferta_ref, p_corte) ORDER BY x.publicada_en DESC, x.oferta_ref), '[]'::jsonb)
   FROM (SELECT o.oferta_ref, o.publicada_en FROM vec_bolsa_llamamientos.oferta_publicada o
          WHERE o.bolsa_ref = p_bolsa AND o.publicada_en <= p_corte AND p_limite BETWEEN 1 AND 100
          ORDER BY o.publicada_en DESC, o.oferta_ref LIMIT greatest(least(p_limite, 100), 1)) x
$f$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.listar_ofertas_candidato_v1(p_candidato_ref text, p_corte timestamptz)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_consumo_candidato_v1(ARRAY['consulta'], p_candidato_ref);
 RETURN (
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'oferta_ref', o.oferta_ref, 'bolsa', o.bolsa_ref, 'datos', o.datos,
   'publicada_en', to_char(o.publicada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'vence_antes_de', to_char(o.vence_antes_de,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'estado', CASE WHEN r.oferta_ref IS NOT NULL AND r.participacion_ref = p.participacion_ref THEN 'adjudicada_propia'
                  WHEN r.oferta_ref IS NOT NULL THEN 'resuelta'
                  WHEN p_corte < o.vence_antes_de THEN 'abierta'
                  ELSE 'pendiente_resolucion' END,
   'disposicion', CASE WHEN d.oferta_ref IS NULL THEN NULL ELSE jsonb_build_object('recibo', d.recibo_ref,
      'manifestada_en', to_char(d.manifestada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END)
   ORDER BY o.vence_antes_de, o.oferta_ref), '[]'::jsonb)
 FROM vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(p_candidato_ref) p
 JOIN vec_bolsa_llamamientos.oferta_publicada o ON o.bolsa_ref = p.bolsa_ref AND o.publicada_en <= p_corte
 LEFT JOIN vec_bolsa_llamamientos.disposicion_oferta d
   ON d.oferta_ref = o.oferta_ref AND d.participacion_ref = p.participacion_ref AND d.manifestada_en <= p_corte
 LEFT JOIN vec_bolsa_llamamientos.resolucion_oferta r ON r.oferta_ref = o.oferta_ref AND r.resuelta_en <= p_corte
 WHERE p_corte IS NOT NULL
   AND ((r.oferta_ref IS NULL AND p_corte < o.vence_antes_de) OR d.oferta_ref IS NOT NULL));
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
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica))<>3
    OR NOT (p_politica ?& ARRAY['plazo','adjudicacion','no_cubierta'])
    OR jsonb_typeof(p_politica->'plazo') IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'plazo'))<>4
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
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'adjudicacion'))<>2
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

DROP FUNCTION vec_bolsa_llamamientos.proyectar_oferta_v2(text,timestamptz);
DROP TABLE vec_bolsa_llamamientos.acto_plaza_oferta_outbox;
DROP TABLE vec_bolsa_llamamientos.acto_plaza_oferta;
DROP TABLE vec_bolsa_llamamientos.plazas_oferta;

GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_oferta_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)
 TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.resolver_oferta_v1(text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_ejecutor;
DO $post$
BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.publicar_oferta_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.resolver_oferta_v1(text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR to_regclass('vec_bolsa_llamamientos.plazas_oferta') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.acto_plaza_oferta') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.acto_plaza_oferta_outbox') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.proyectar_oferta_v2(text,timestamptz)') IS NOT NULL
 THEN RAISE EXCEPTION 'B58: DOWN incompleto' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
