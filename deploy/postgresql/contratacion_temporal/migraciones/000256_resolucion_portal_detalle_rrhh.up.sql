\set ON_ERROR_STOP on
-- CT256 / F1-127: proyecta la pareja CT249 de la resolución portal en
-- el lector RRHH existente. Fuente estática: pg_get_functiondef real post B2.
-- Solo cuatro datos de resolución; no altera V3, paginación, correo ni ACL.
-- Requiere CT249/CT250 finales y CT253. Una sola aplicación, sin DOWN.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000256',0));
DO $pre$
DECLARE f regprocedure:=to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)'); p pg_proc%ROWTYPE; meta jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 OR f IS NULL
 OR to_regclass('vec_contratacion_temporal.resolucion_manual_respuesta_rrhh') IS NULL
 OR to_regclass('vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1') IS NULL
 THEN RAISE EXCEPTION 'CT256: dependencias PG18/CT249 ausentes' USING ERRCODE='55000'; END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=f;
 IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
 OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v' OR p.proparallel<>'u'
 OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM '6c62a090c72f10ad4e898ef5594f3f8445fb2b2f37582a24823ca57c7e1946de'
 THEN RAISE EXCEPTION 'CT256: preimagen del lector divergente' USING ERRCODE='55000'; END IF;
SELECT jsonb_build_object('proc',to_jsonb(p)-'prosrc',
 'deps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=p.oid),
 'shared',(SELECT coalesce(jsonb_agg(to_jsonb(s) ORDER BY s.dbid,s.classid,s.objid,s.objsubid,s.refclassid,s.refobjid,s.deptype),'[]'::jsonb)
  FROM pg_shdepend s WHERE s.classid='pg_proc'::regclass AND s.objid=p.oid
   AND s.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())))
 INTO meta;
 PERFORM set_config('vec.ct256.meta',meta::text,true);
END $pre$;

-- Definición completa post B2; solo se añadió resolucion_ct en cada participación.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea, p_cursor text)
 RETURNS TABLE(contenido_canonico bytea, esquema text, acceso_ref text, secuencia numeric, anterior_sha256 text, huella_sha256 text, vinculo_identidad_huella_sha256 text, alcance_huella_sha256 text, registrada_en timestamp with time zone, auditoria_vec_ref text, auditoria_vec_huella_sha256 text, consumo_vec_huella_sha256 text, contenido_huella_sha256 text, resultado_huella_sha256 text, cursor_huella_sha256 text, generada_en timestamp with time zone, expediente_ref text, version_expediente numeric, total smallint, recibo_sello_sha256 text, resultado_bolsa jsonb)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '1s'
 SET statement_timeout TO '4s'
 SET idle_in_transaction_session_timeout TO '6s'
AS $function$
DECLARE d record; actual record; vinculos jsonb; sugeridas jsonb; origenes jsonb; opacas jsonb; nativa record; n integer;
 entradas jsonb; todos jsonb; lote jsonb; pagina_n integer; antes_en timestamptz; antes_ref text;
 ultimo_en timestamptz; ultimo_ref text; mas boolean:=false; siguiente text; bolsa_vigente text;
 personas_text text; personas numeric; aceptaciones integer;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_consultor_rrhh','MEMBER') IS NOT TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR (p_cursor IS NOT NULL AND p_cursor !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z#llamamiento:[0-9a-f]{64}$') THEN
  RAISE EXCEPTION 'CT201: lectura de resultado denegada' USING ERRCODE='42501';
 END IF;
 IF p_cursor IS NOT NULL THEN
  antes_en:=split_part(p_cursor,'#',1)::timestamptz;
  antes_ref:=split_part(p_cursor,'#',2);
 END IF;
 SELECT * INTO STRICT d FROM vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
  p_alcance,p_consulta,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT coalesce(jsonb_agg(jsonb_build_object('bolsa_ref',x.bolsa_ref,
  'llamamiento_ref',x.llamamiento_ref,'recibo_emision_ref',x.recibo_emision_ref)),
  '[]'::jsonb) INTO todos FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
  WHERE x.expediente_ref=d.expediente_ref;
 n:=jsonb_array_length(todos);
 aceptaciones:=vec_bolsa_llamamientos.contar_aceptaciones_firmes_ct_v1(todos);
 SELECT coalesce(jsonb_agg(jsonb_build_object('bolsa_ref',x.bolsa_ref,
   'llamamiento_ref',x.llamamiento_ref,'recibo_emision_ref',x.recibo_emision_ref)
   ORDER BY x.vinculado_en DESC,x.llamamiento_ref DESC),'[]'::jsonb) INTO entradas
 FROM (SELECT v.bolsa_ref,v.llamamiento_ref,v.recibo_emision_ref,v.vinculado_en
   FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 v
   WHERE v.expediente_ref=d.expediente_ref
     AND (p_cursor IS NULL OR (v.vinculado_en,v.llamamiento_ref)<(antes_en,antes_ref))
   ORDER BY v.vinculado_en DESC,v.llamamiento_ref DESC LIMIT 20) x;
 pagina_n:=jsonb_array_length(entradas);
 IF pagina_n=20 THEN
  SELECT v.vinculado_en,v.llamamiento_ref INTO ultimo_en,ultimo_ref
   FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 v
   WHERE v.expediente_ref=d.expediente_ref
     AND (p_cursor IS NULL OR (v.vinculado_en,v.llamamiento_ref)<(antes_en,antes_ref))
   ORDER BY v.vinculado_en DESC,v.llamamiento_ref DESC OFFSET 19 LIMIT 1;
  SELECT EXISTS(SELECT 1 FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 v
    WHERE v.expediente_ref=d.expediente_ref
      AND (v.vinculado_en,v.llamamiento_ref)<(ultimo_en,ultimo_ref)) INTO mas;
  IF mas THEN
   siguiente:=to_char(ultimo_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')||'#'||ultimo_ref;
  END IF;
 END IF;
 lote:=vec_bolsa_llamamientos.leer_resultados_emisiones_ct_v1(entradas);
 -- O6 se recorre una vez para esta página; el mapa solo contiene emisiones
 -- con una selección exacta y snapshot fiscalizado original.
 WITH pagina AS MATERIALIZED (
  SELECT x.organizacion_ref,x.expediente_ref,x.bolsa_ref,x.llamamiento_ref,
   x.recibo_emision_ref
  FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
  JOIN LATERAL jsonb_array_elements(entradas) item(valor)
   ON item.valor->>'llamamiento_ref'=x.llamamiento_ref
  WHERE x.expediente_ref=d.expediente_ref
 ), o6 AS MATERIALIZED (
  SELECT e.solicitud_json->>'organizacion_ref' AS organizacion_ref,
   e.solicitud_json->>'expediente_ref' AS expediente_ref,
   e.recibo_json#>>'{bolsa,referencia}' AS bolsa_ref,
   e.recibo_json->>'llamamiento_ref' AS llamamiento_ref,
   e.recibo_json->>'recibo_ref' AS recibo_emision_ref,
   e.solicitud_json->'version_expediente' AS version_solicitud,
   e.recibo_json->'version_expediente' AS version_recibo,
   e.recibo_json->>'organizacion_ref' AS organizacion_recibo,
   e.recibo_json->>'expediente_ref' AS expediente_recibo,
   (e.tipo_seleccion='llamamiento' AND
    vec_contratacion_temporal.origen_seleccion_o6_ct249_v1(
     e.solicitud_json,e.recibo_json,e.artefacto_canonico) IS TRUE) AS financiera
  FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
  WHERE e.situacion='confirmada'
   AND e.solicitud_json->>'expediente_ref'=d.expediente_ref
   AND e.recibo_json->'propuesta_generada'='true'::jsonb
 ), exactas AS (
  SELECT x.llamamiento_ref,count(*) AS cantidad,
   count(*) FILTER (WHERE e.financiera) AS cantidad_financiera FROM pagina x
  JOIN o6 e ON e.organizacion_ref=x.organizacion_ref
   AND e.expediente_ref=x.expediente_ref
   AND e.organizacion_recibo=x.organizacion_ref
   AND e.expediente_recibo=x.expediente_ref
   AND e.bolsa_ref=x.bolsa_ref AND e.llamamiento_ref=x.llamamiento_ref
   AND e.recibo_emision_ref=x.recibo_emision_ref
   AND e.version_recibo=e.version_solicitud
   AND jsonb_typeof(e.version_recibo)='number'
  JOIN vec_contratacion_temporal.expediente_version_integral h
   ON h.expediente_ref=x.expediente_ref AND h.version=CASE
      WHEN e.version_recibo#>>'{}' ~ '^[1-9][0-9]{0,15}$'
      THEN (e.version_recibo#>>'{}')::numeric END
   AND h.agregado_json#>>'{via_cobertura,via_clave}'='bolsa_vigente'
   AND h.agregado_json#>>'{via_cobertura,decision_gobernada,via_elegida}'='bolsa_vigente'
   AND h.agregado_json#>>'{analisis,categoria_ref}' IS NOT NULL
   AND h.agregado_json#>>'{asignacion,unidad_ref}' IS NOT NULL
  GROUP BY x.llamamiento_ref
 ) SELECT coalesce(jsonb_object_agg(llamamiento_ref,jsonb_build_object(
    'confirmado',cantidad=1,'admitido',cantidad=1 AND cantidad_financiera=1)),'{}'::jsonb)
 INTO origenes FROM exactas;
 -- CT201 fija la emisión original; B98 entrega sus participantes. La O6
 -- exacta y única acredita al participante solo cuando B98 devuelve uno.
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'bolsa_ref',x.bolsa_ref,'llamamiento_ref',x.llamamiento_ref,
   'recibo_emision_ref',x.recibo_emision_ref,
   'recibo_vinculo_ref',x.recibo_ref,'vinculado_en',x.vinculado_en,
   'emitido_en',b.valor->'emitido_en',
   'participaciones',(
    SELECT coalesce(jsonb_agg(part.valor || jsonb_build_object(
     'origen_ct_confirmado',coalesce(
       jsonb_array_length(b.valor->'participaciones')=1
       AND part.valor->>'respuesta'='acepta' AND part.valor->>'modo'='firme'
       AND part.valor->>'recibo_respuesta_ref' IS NOT NULL
       AND part.valor->>'respondida_en' IS NOT NULL
       AND origenes#>>ARRAY[x.llamamiento_ref,'confirmado']='true',false),
     'resolucion_portal_disponible',coalesce(
       jsonb_array_length(b.valor->'participaciones')=1
       AND part.valor->>'respuesta'='acepta' AND part.valor->>'modo'='firme'
       AND part.valor->>'recibo_respuesta_ref' IS NOT NULL
       AND part.valor->>'respondida_en' IS NOT NULL
       AND origenes#>>ARRAY[x.llamamiento_ref,'admitido']='true'
       AND r.resolucion_ref IS NULL,false),
     'resolucion_ct',CASE WHEN
   jsonb_array_length(b.valor->'participaciones')=1
   AND origenes#>>ARRAY[x.llamamiento_ref,'confirmado']='true'
   AND part.valor->>'respuesta'='acepta'
   AND part.valor->>'modo'='firme'
   AND part.valor->>'recibo_respuesta_ref' IS NOT NULL
   AND part.valor->>'respondida_en' IS NOT NULL
   AND r.fuente_respuesta='portal'
   AND r.organizacion_ref=x.organizacion_ref
   AND r.expediente_ref=x.expediente_ref
   AND r.llamamiento_ref=x.llamamiento_ref
   AND r.participacion_ref=part.valor->>'participacion_ref'
   AND r.recibo_respuesta_portal_ref=part.valor->>'recibo_respuesta_ref'
   AND r.estado='confirmado'
   AND r.version_resultante=1
   AND r.estado_plazo='revision_manual_portal'
   AND r.revision_respuesta_rrhh AND r.revision_plazo_rrhh
   AND r.comunicacion_ref IS NULL AND r.justificante_ref IS NULL
   AND r.solicitud_json->>'Respuesta'='aceptacion'
   AND r.solicitud_json->>'OrganizacionRef'=x.organizacion_ref
   AND r.solicitud_json->>'ExpedienteRef'=x.expediente_ref
   AND r.solicitud_json->>'LlamamientoRef'=x.llamamiento_ref
   AND r.solicitud_json->>'BolsaRef'=x.bolsa_ref
   AND r.solicitud_json->>'ReciboEmisionRef'=x.recibo_emision_ref
   AND r.solicitud_json->>'ReciboRespuestaRef'=r.recibo_respuesta_portal_ref
   AND r.solicitud_json->>'ParticipacionRef'=r.participacion_ref
   AND r.recibo_json->>'Estado'='confirmado'
   AND r.recibo_json->'Solicitud'=r.solicitud_json
   AND r.recibo_json->>'ResolucionRef'=r.resolucion_ref
   AND r.recibo_json->>'ReciboLocalRef'=r.recibo_ref
   AND r.recibo_json#>>'{RespuestaPortal,RespuestaRef}'=r.respuesta_portal_ref
   AND r.recibo_json#>>'{RespuestaPortal,ReciboRespuestaRef}'=r.recibo_respuesta_portal_ref
   AND r.recibo_json#>>'{RespuestaPortal,ParticipacionRef}'=r.participacion_ref
   AND (r.recibo_json#>>'{RespuestaPortal,RespondidaEn}')::timestamptz=
       (part.valor->>'respondida_en')::timestamptz
   AND r.recibo_json->>'ResueltaEn'=to_char(r.resuelta_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
  THEN jsonb_build_object('resolucion_ref',r.resolucion_ref,
   'recibo_ref',r.recibo_ref,'resuelta_en',r.resuelta_en,'respuesta','aceptacion')
  END
    ) ORDER BY part.ordinal),'[]'::jsonb)
    FROM jsonb_array_elements(b.valor->'participaciones') WITH ORDINALITY AS part(valor,ordinal)
    LEFT JOIN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r
      ON r.organizacion_ref=x.organizacion_ref AND r.expediente_ref=x.expediente_ref
     AND r.llamamiento_ref=x.llamamiento_ref
   ))
   ORDER BY x.vinculado_en DESC,x.llamamiento_ref DESC),'[]'::jsonb) INTO vinculos
 FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
 JOIN LATERAL jsonb_array_elements(lote) b(valor)
   ON b.valor->>'llamamiento_ref'=x.llamamiento_ref
  AND b.valor->>'bolsa_ref'=x.bolsa_ref
  AND b.valor->>'recibo_emision_ref'=x.recibo_emision_ref
 WHERE x.expediente_ref=d.expediente_ref;
 IF jsonb_array_length(vinculos)<>pagina_n THEN
  RAISE EXCEPTION 'CT201: resultado Bolsa no recuperable' USING ERRCODE='55000';
 END IF;
 SELECT e.numero_visible,a.version,v.agregado_json INTO actual
 FROM vec_contratacion_temporal.expediente_alta e
 JOIN vec_contratacion_temporal.expediente_integral_actual a ON a.expediente_ref=e.expediente_ref
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE e.expediente_ref=d.expediente_ref;
 IF NOT FOUND OR actual.version IS DISTINCT FROM d.version_expediente THEN
  RAISE EXCEPTION 'CT201: agregado de lectura divergente' USING ERRCODE='55000';
 END IF;
 personas_text:=actual.agregado_json#>>'{solicitud,numero_personas}';
 IF personas_text IS NULL THEN
  personas_text:=actual.agregado_json#>>'{solicitud,necesidad,campos,numero_personas}';
 END IF;
 IF personas_text IS NOT NULL THEN
  IF personas_text !~ '^[1-9][0-9]{0,9}$' THEN
   RAISE EXCEPTION 'CT201: número de personas inválido' USING ERRCODE='55000';
  END IF;
  IF personas_text::numeric>4294967295 THEN
   RAISE EXCEPTION 'CT201: número de personas inválido' USING ERRCODE='55000';
  END IF;
  personas:=personas_text::numeric;
 END IF;
 sugeridas:='[]'::jsonb;
 IF actual.agregado_json#>>'{via_cobertura,via_clave}'='bolsa_vigente'
    AND actual.agregado_json#>>'{via_cobertura,decision_gobernada,via_elegida}'='bolsa_vigente'
    AND actual.agregado_json#>'{via_cobertura,bolsa_ref}' IS NULL
    AND actual.agregado_json#>>'{analisis,categoria_ref}' IS NOT NULL THEN
  bolsa_vigente:=vec_bolsa_llamamientos.resolver_bolsa_vigente_ct_v1(
   actual.agregado_json#>>'{analisis,categoria_ref}');
 END IF;
 IF bolsa_vigente IS NOT NULL THEN
  SELECT coalesce(jsonb_agg(to_jsonb(b) ORDER BY b.emitido_en DESC,b.llamamiento_ref DESC),'[]'::jsonb)
   INTO sugeridas FROM vec_bolsa_llamamientos.listar_emisiones_ct_v1(
    bolsa_vigente,
    vec_contratacion_temporal.numero_visible_vigente_v1(d.expediente_ref,actual.numero_visible),
    NULL,NULL,20,ARRAY(SELECT x.llamamiento_ref
     FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
     WHERE x.expediente_ref=d.expediente_ref)) b;
  -- CT238 permite una O6 confirmada por expediente; B98 devuelve su
  -- referencia de configuración real, aunque sea la opaca del expediente.
  SELECT e.recibo_json->>'llamamiento_ref' AS llamamiento_ref,
   e.recibo_json->>'recibo_ref' AS recibo_emision_ref INTO nativa
  FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
  JOIN vec_contratacion_temporal.expediente_version_integral h
   ON h.expediente_ref=d.expediente_ref AND h.version=CASE
    WHEN e.recibo_json->>'version_expediente' ~ '^[1-9][0-9]{0,15}$'
    THEN (e.recibo_json->>'version_expediente')::numeric END
  WHERE e.tipo_seleccion='llamamiento' AND e.situacion='confirmada'
    AND e.artefacto_canonico IS NOT NULL
    AND jsonb_typeof(e.recibo_json->'version_expediente')='number'
    AND e.solicitud_json->>'organizacion_ref'=p_alcance.organizacion_ref
    AND e.solicitud_json->>'expediente_ref'=d.expediente_ref
    AND e.recibo_json->>'organizacion_ref'=p_alcance.organizacion_ref
    AND e.recibo_json->>'expediente_ref'=d.expediente_ref
    AND e.recibo_json#>>'{bolsa,referencia}'=bolsa_vigente
    AND e.recibo_json->'version_expediente'=e.solicitud_json->'version_expediente'
    AND e.recibo_json->'propuesta_generada'='true'::jsonb
    AND e.recibo_json->>'llamamiento_ref' ~ '^llamamiento:[0-9a-f]{64}$'
    AND e.recibo_json->>'recibo_ref'='recibo:'||(e.recibo_json->>'llamamiento_ref')
    AND h.agregado_json#>>'{via_cobertura,via_clave}'='bolsa_vigente'
    AND h.agregado_json#>>'{via_cobertura,decision_gobernada,via_elegida}'='bolsa_vigente'
    AND h.agregado_json#>>'{analisis,categoria_ref}'=actual.agregado_json#>>'{analisis,categoria_ref}'
    AND h.agregado_json#>>'{asignacion,unidad_ref}' IS NOT NULL
    AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
      WHERE x.llamamiento_ref=e.recibo_json->>'llamamiento_ref');
  IF FOUND THEN
   SELECT coalesce(jsonb_agg(to_jsonb(b)),'[]'::jsonb) INTO opacas
   FROM vec_bolsa_llamamientos.listar_emisiones_ct_v1(
    bolsa_vigente,d.expediente_ref,NULL,NULL,20,
    ARRAY(SELECT x.llamamiento_ref FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
      WHERE x.expediente_ref=d.expediente_ref)) b
   WHERE b.llamamiento_ref=nativa.llamamiento_ref
     AND b.recibo_emision_ref=nativa.recibo_emision_ref;
   IF jsonb_array_length(opacas)=1 THEN
    SELECT coalesce(jsonb_agg(z.valor ORDER BY z.orden),'[]'::jsonb) INTO sugeridas
    FROM (SELECT opacas->0 AS valor,0::bigint AS orden
      UNION ALL
      SELECT m.valor,m.pos FROM jsonb_array_elements(sugeridas) WITH ORDINALITY m(valor,pos)
      WHERE m.valor->>'llamamiento_ref' IS DISTINCT FROM nativa.llamamiento_ref
      ORDER BY orden LIMIT 20) z;
   END IF;
  END IF;
 END IF;
 RETURN QUERY SELECT d.contenido_canonico,d.esquema,d.acceso_ref,d.secuencia,
  d.anterior_sha256,d.huella_sha256,d.vinculo_identidad_huella_sha256,
  d.alcance_huella_sha256,d.registrada_en,d.auditoria_vec_ref,
  d.auditoria_vec_huella_sha256,d.consumo_vec_huella_sha256,
  d.contenido_huella_sha256,d.resultado_huella_sha256,d.cursor_huella_sha256,
  d.generada_en,d.expediente_ref,d.version_expediente,d.total,d.recibo_sello_sha256,
  jsonb_build_object('vinculos',vinculos,'total_vinculos',n,
   'personas_solicitadas',personas,'aceptaciones_firmes',aceptaciones,
   'emisiones_vinculables',sugeridas,'siguiente_cursor',siguiente);
END $function$
;

DO $post$
DECLARE f regprocedure:=to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)'); p pg_proc%ROWTYPE; meta jsonb;
BEGIN
 SELECT * INTO p FROM pg_proc WHERE oid=f;
SELECT jsonb_build_object('proc',to_jsonb(p)-'prosrc',
 'deps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=p.oid),
 'shared',(SELECT coalesce(jsonb_agg(to_jsonb(s) ORDER BY s.dbid,s.classid,s.objid,s.objsubid,s.refclassid,s.refobjid,s.deptype),'[]'::jsonb)
  FROM pg_shdepend s WHERE s.classid='pg_proc'::regclass AND s.objid=p.oid
   AND s.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())))
 INTO meta;
 IF NOT FOUND OR meta IS DISTINCT FROM current_setting('vec.ct256.meta')::jsonb
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM '7748e4d63deb1766850aeb8596c5eca996705176e2de0cb90312acbd2a14cf0d'
 THEN RAISE EXCEPTION 'CT256: postimagen, metadatos o ACL divergentes' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
