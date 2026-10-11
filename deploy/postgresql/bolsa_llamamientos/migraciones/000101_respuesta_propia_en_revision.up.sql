\set ON_ERROR_STOP on
-- B101. Estado de revisión de la última respuesta propia. Literal extraído de
-- la copia HZ20 + inscripción; mismo criterio temporal que B100. No modifica
-- situaciones, permisos ni auditoría. La lectura conserva su consumo nominal.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000101',0));
DO $pre$
DECLARE actual text;
BEGIN
 SELECT md5(prosrc) INTO actual FROM pg_proc
  WHERE oid='vec_bolsa_llamamientos.leer_portal_candidato_v1(text,timestamptz,text[])'::regprocedure;
 IF actual IS DISTINCT FROM 'ab73f65d0dbc862026d4424304e20680' THEN
  RAISE EXCEPTION 'B101 clave=lector_preimagen actual=% esperado=ab73f65d0dbc862026d4424304e20680',actual USING ERRCODE='55000';
 END IF;
 IF to_regprocedure('vec_bolsa_llamamientos.respuestas_portal_sin_reflejar_v1(text,timestamptz)') IS NULL THEN
  RAISE EXCEPTION 'B101 clave=B100 actual=ausente esperado=instalada' USING ERRCODE='55000';
 END IF;
 PERFORM set_config('vec_b101.preimagen',(SELECT jsonb_build_object('oid',oid,'owner',proowner,'acl',proacl) FROM pg_proc
  WHERE oid='vec_bolsa_llamamientos.leer_portal_candidato_v1(text,timestamptz,text[])'::regprocedure)::text,true);
END $pre$;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.leer_portal_candidato_v1(p_candidato_ref text, p_corte timestamp with time zone, p_resultados_efectivos text[])
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_consumo_candidato_v1(ARRAY['consulta','responder'],p_candidato_ref);
 RETURN (
 SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
  'bolsa',p.bolsa_ref,
  'llamamiento_abierto',CASE WHEN a.contacto_en IS NULL THEN NULL ELSE pg_catalog.jsonb_build_object(
    'contacto_en',pg_catalog.to_char(a.contacto_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END,
  'solicitud_pendiente',(SELECT pg_catalog.jsonb_build_object('tipo',s.tipo,'recibo',s.recibo_ref,
    'registrada_en',pg_catalog.to_char(s.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'pausa_hasta',CASE WHEN s.pausa_hasta IS NULL THEN NULL ELSE pg_catalog.to_char(s.pausa_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END)
   FROM vec_bolsa_llamamientos.solicitud_portal_candidato s
   WHERE s.participacion_ref=p.participacion_ref AND s.registrada_en<=p_corte
     AND vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(s.solicitud_ref)
   ORDER BY s.registrada_en DESC LIMIT 1),
  'ultima_solicitud_documental',(SELECT pg_catalog.jsonb_build_object(
    'tipo','documental_rrhh','recibo',s.recibo_ref,
    'registrada_en',pg_catalog.to_char(s.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'estado',coalesce(r.resultado,'pendiente_rrhh'),'recibo_resolucion_ref',r.recibo_ref,
    'recibo_b8_ref',r.recibo_b8_ref)
   FROM vec_bolsa_llamamientos.solicitud_documental_rrhh s
   LEFT JOIN vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh r ON r.solicitud_ref=s.solicitud_ref
   WHERE s.participacion_ref=p.participacion_ref AND s.candidato_ref=p_candidato_ref
     AND s.registrada_en<=p_corte ORDER BY s.registrada_en DESC,s.solicitud_ref DESC LIMIT 1),
  'solicitud_documental_pendiente',EXISTS(
   SELECT 1 FROM vec_bolsa_llamamientos.solicitud_documental_rrhh s
   WHERE s.participacion_ref=p.participacion_ref AND s.candidato_ref=p_candidato_ref
     AND s.registrada_en<=p_corte AND NOT EXISTS
      (SELECT 1 FROM vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh r WHERE r.solicitud_ref=s.solicitud_ref)),
  'ultima_respuesta',(SELECT pg_catalog.jsonb_build_object('respuesta',r.respuesta,'modo',r.modo,
    'recibo',r.recibo_ref,'respondida_en',pg_catalog.to_char(r.respondida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'en_revision',NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s
      WHERE s.participacion_ref=r.participacion_ref
        AND s.registrada_en>=r.respondida_en AND s.registrada_en<=p_corte AND s.desde<=p_corte))
   FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento r
   WHERE r.participacion_ref=p.participacion_ref AND r.respondida_en<=p_corte
   ORDER BY r.respondida_en DESC,r.respuesta_ref DESC LIMIT 1)
 ) ORDER BY p.bolsa_ref),'[]'::jsonb)
 FROM vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(p_candidato_ref) p
 LEFT JOIN LATERAL vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(
  p.participacion_ref,p_corte,p_resultados_efectivos) a ON true);
END $function$;


DO $post$
DECLARE p record; anterior jsonb;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc
  WHERE oid='vec_bolsa_llamamientos.leer_portal_candidato_v1(text,timestamptz,text[])'::regprocedure;
 anterior:=current_setting('vec_b101.preimagen')::jsonb;
 IF md5(p.prosrc) IS DISTINCT FROM '97b529be23bad1bb5e97d46d773a4429' THEN
  RAISE EXCEPTION 'B101 clave=lector_postimagen actual=% esperado=97b529be23bad1bb5e97d46d773a4429',md5(p.prosrc) USING ERRCODE='55000';
 END IF;
 IF p.oid IS DISTINCT FROM (anterior->>'oid')::oid OR p.proowner IS DISTINCT FROM (anterior->>'owner')::oid
    OR to_jsonb(p.proacl) IS DISTINCT FROM anterior->'acl' OR NOT p.prosecdef
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[] THEN
  RAISE EXCEPTION 'B101 clave=lector_contrato actual=%/%/%/%/% esperado=%/%/%/true/search_path=pg_catalog,pg_temp',
   p.oid,p.proowner,p.proacl,p.prosecdef,p.proconfig,anterior->>'oid',anterior->>'owner',anterior->'acl' USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
