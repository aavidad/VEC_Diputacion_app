\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000098',0));

DO $pre$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.llamamiento_emitido') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.respuesta_portal_llamamiento') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.contacto_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NULL
    OR to_regrole('vec_contratacion_temporal_propietario') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.verificar_emision_ct_v1(text,text,text,text)') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.leer_resultado_emision_ct_v1(text,text,text)') IS NOT NULL THEN
   RAISE EXCEPTION 'B98: PARO clave=preimagen_emision_ct esperado=tablas_y_rol_sin_funciones actual=incompatible'
     USING ERRCODE='55000';
 END IF;
END $pre$;

-- La referencia de negocio sólo ayuda a localizar. El vínculo lo confirma
-- RRHH y se valida con las tres claves inmutables de la emisión real.
CREATE FUNCTION vec_bolsa_llamamientos.verificar_emision_ct_v1(
 p_bolsa text,p_llamamiento text,p_recibo text,p_referencia text)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET statement_timeout='3s' AS $f$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR session_user=current_user
    OR (pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
        AND pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE)
    OR p_bolsa IS NULL OR p_llamamiento IS NULL OR p_recibo IS NULL
    OR p_referencia IS NULL OR p_referencia='' THEN
   RAISE EXCEPTION 'B98: consulta de emisión no autorizada' USING ERRCODE='42501';
 END IF;
 RETURN EXISTS (
  SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_emitido e
  WHERE e.llamamiento_ref=p_llamamiento AND e.bolsa_ref=p_bolsa
    AND e.recibo_ref=p_recibo AND e.configuracion->>'referencia'=p_referencia
 );
END $f$;

CREATE INDEX llamamiento_emitido_ct_necesidad_pagina
 ON vec_bolsa_llamamientos.llamamiento_emitido
 (bolsa_ref,(configuracion->>'referencia'),emitido_en DESC,llamamiento_ref DESC);
CREATE INDEX contacto_participacion_ct_resultado_exacta
 ON vec_bolsa_llamamientos.contacto_participacion
 (llamamiento_ref,participacion_ref,instante DESC,contacto_ref DESC)
 WHERE llamamiento_ref IS NOT NULL;

-- Sugerencias acotadas: la coincidencia de referencia nunca asocia por sí
-- sola una emisión. Un llamamiento se selecciona y confirma expresamente.
CREATE FUNCTION vec_bolsa_llamamientos.listar_emisiones_ct_v1(
 p_bolsa text,p_referencia text,p_antes timestamptz,p_antes_ref text,p_limite integer)
RETURNS TABLE(bolsa_ref text,llamamiento_ref text,recibo_emision_ref text,
 referencia_visible text,emitido_en timestamptz)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET statement_timeout='3s' AS $f$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR p_bolsa IS NULL OR p_bolsa='' OR p_referencia IS NULL OR p_referencia=''
    OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 21
    OR (p_antes IS NULL) IS DISTINCT FROM (p_antes_ref IS NULL) THEN
   RAISE EXCEPTION 'B98: lista de emisiones no autorizada' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT e.bolsa_ref,e.llamamiento_ref,e.recibo_ref,
   e.configuracion->>'referencia',e.emitido_en
 FROM vec_bolsa_llamamientos.llamamiento_emitido e
 WHERE e.bolsa_ref=p_bolsa AND e.configuracion->>'referencia'=p_referencia
   AND (p_antes IS NULL OR (e.emitido_en,e.llamamiento_ref)<(p_antes,p_antes_ref))
 ORDER BY e.emitido_en DESC,e.llamamiento_ref DESC LIMIT p_limite;
END $f$;

-- Sólo el propietario CT puede invocar esta lectura. Debe llamarla desde
-- la transacción de detalle CT, después de la autorización de sesión y el
-- asiento de lectura común. Los resultados se ligan al llamamiento exacto.
CREATE FUNCTION vec_bolsa_llamamientos.leer_resultado_emision_ct_v1(
 p_bolsa text,p_llamamiento text,p_recibo text)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET statement_timeout='3s' AS $f$
DECLARE v_resultado jsonb;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR p_bolsa IS NULL OR p_llamamiento IS NULL OR p_recibo IS NULL THEN
   RAISE EXCEPTION 'B98: lectura de resultado no autorizada' USING ERRCODE='42501';
 END IF;
 SELECT jsonb_build_object(
    'bolsa_ref',e.bolsa_ref,'llamamiento_ref',e.llamamiento_ref,
    'recibo_emision_ref',e.recibo_ref,'emitido_en',e.emitido_en,
    'participaciones',coalesce((
      SELECT jsonb_agg(jsonb_build_object(
        'participacion_ref',p.ref,
        'respuesta',r.respuesta,'modo',r.modo,
        'recibo_respuesta_ref',r.recibo_ref,'respondida_en',r.respondida_en,
        'justificante_ref',r.justificante_ref,
        'contacto_resultado',c.resultado,'recibo_contacto_ref',c.recibo_ref,
        'contacto_en',c.instante,
        'situacion_actual',s.situacion,'recibo_situacion_ref',s.recibo_ref,
        'situacion_desde',s.desde
      ) ORDER BY p.ordinal)
      FROM jsonb_array_elements_text(e.participaciones) WITH ORDINALITY AS p(ref,ordinal)
      LEFT JOIN LATERAL (
        SELECT x.respuesta,x.modo,x.recibo_ref,x.respondida_en,x.justificante_ref
        FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento x
        WHERE x.llamamiento_ref=e.llamamiento_ref AND x.participacion_ref=p.ref
        LIMIT 1
      ) r ON true
      LEFT JOIN LATERAL (
        SELECT x.resultado,x.recibo_ref,x.instante
        FROM vec_bolsa_llamamientos.contacto_participacion x
        WHERE x.llamamiento_ref=e.llamamiento_ref AND x.participacion_ref=p.ref
        ORDER BY x.instante DESC,x.contacto_ref DESC LIMIT 1
      ) c ON true
      LEFT JOIN LATERAL (
        SELECT x.situacion,x.recibo_ref,x.desde
        FROM vec_bolsa_llamamientos.situacion_participacion x
        WHERE x.participacion_ref=p.ref
        ORDER BY x.desde DESC LIMIT 1
      ) s ON true
    ),'[]'::jsonb)
  ) INTO v_resultado
 FROM vec_bolsa_llamamientos.llamamiento_emitido e
 WHERE e.llamamiento_ref=p_llamamiento AND e.bolsa_ref=p_bolsa AND e.recibo_ref=p_recibo;
 RETURN v_resultado;
END $f$;

-- La ficha consulta todos sus vínculos en una sola sentencia: ni autorización
-- ni auditoría por participación, y las tres fuentes se reúnen por clave.
CREATE FUNCTION vec_bolsa_llamamientos.leer_resultados_emisiones_ct_v1(p_vinculos jsonb)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET statement_timeout='5s' AS $f$
DECLARE v_resultado jsonb; v_esperados integer;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR p_vinculos IS NULL OR jsonb_typeof(p_vinculos)<>'array'
    OR jsonb_array_length(p_vinculos)>100 THEN
  RAISE EXCEPTION 'B98: lectura por lote no autorizada' USING ERRCODE='42501';
 END IF;
 v_esperados:=jsonb_array_length(p_vinculos);
 WITH vinculos AS MATERIALIZED (
   SELECT v.bolsa_ref,v.llamamiento_ref,v.recibo_emision_ref
   FROM jsonb_to_recordset(p_vinculos) AS v(bolsa_ref text,llamamiento_ref text,recibo_emision_ref text)
 ), emisiones AS MATERIALIZED (
   SELECT e.bolsa_ref,e.llamamiento_ref,e.recibo_ref,e.emitido_en,e.participaciones
   FROM vinculos v JOIN vec_bolsa_llamamientos.llamamiento_emitido e
     ON e.bolsa_ref=v.bolsa_ref AND e.llamamiento_ref=v.llamamiento_ref
    AND e.recibo_ref=v.recibo_emision_ref
 ), partes AS MATERIALIZED (
   SELECT e.llamamiento_ref,e.bolsa_ref,p.ref,p.orden
   FROM emisiones e CROSS JOIN LATERAL
     jsonb_array_elements_text(e.participaciones) WITH ORDINALITY AS p(ref,orden)
 ), contactos AS (
   SELECT DISTINCT ON (c.llamamiento_ref,c.participacion_ref)
     c.llamamiento_ref,c.participacion_ref,c.resultado,c.recibo_ref,c.instante
   FROM vec_bolsa_llamamientos.contacto_participacion c
   JOIN partes p ON p.llamamiento_ref=c.llamamiento_ref AND p.ref=c.participacion_ref
   ORDER BY c.llamamiento_ref,c.participacion_ref,c.instante DESC,c.contacto_ref DESC
 ), situaciones AS (
   SELECT DISTINCT ON (s.participacion_ref)
     s.participacion_ref,s.situacion,s.recibo_ref,s.desde
   FROM vec_bolsa_llamamientos.situacion_participacion s
   JOIN partes p ON p.ref=s.participacion_ref
   ORDER BY s.participacion_ref,s.desde DESC
 ), filas AS (
   SELECT p.llamamiento_ref,p.orden,
     jsonb_build_object('participacion_ref',p.ref,
       'respuesta',r.respuesta,'modo',r.modo,
       'recibo_respuesta_ref',r.recibo_ref,'respondida_en',r.respondida_en,
       'justificante_ref',r.justificante_ref,
       'contacto_resultado',c.resultado,'recibo_contacto_ref',c.recibo_ref,
       'contacto_en',c.instante,'situacion_actual',s.situacion,
       'recibo_situacion_ref',s.recibo_ref,'situacion_desde',s.desde) AS dato
   FROM partes p
   LEFT JOIN vec_bolsa_llamamientos.respuesta_portal_llamamiento r
     ON r.llamamiento_ref=p.llamamiento_ref AND r.participacion_ref=p.ref
   LEFT JOIN contactos c ON c.llamamiento_ref=p.llamamiento_ref AND c.participacion_ref=p.ref
   LEFT JOIN situaciones s ON s.participacion_ref=p.ref
 ), grupos AS (
   SELECT e.bolsa_ref,e.llamamiento_ref,e.recibo_ref,e.emitido_en,
     jsonb_agg(f.dato ORDER BY f.orden) AS participaciones
   FROM emisiones e JOIN filas f ON f.llamamiento_ref=e.llamamiento_ref
   GROUP BY e.bolsa_ref,e.llamamiento_ref,e.recibo_ref,e.emitido_en
 )
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'bolsa_ref',g.bolsa_ref,'llamamiento_ref',g.llamamiento_ref,
   'recibo_emision_ref',g.recibo_ref,'emitido_en',g.emitido_en,
   'participaciones',g.participaciones) ORDER BY g.emitido_en,g.llamamiento_ref),'[]'::jsonb)
 INTO v_resultado FROM grupos g;
 IF jsonb_array_length(v_resultado)<>v_esperados THEN
  RAISE EXCEPTION 'B98: lote de emisiones divergente' USING ERRCODE='55000';
 END IF;
 RETURN v_resultado;
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.verificar_emision_ct_v1(text,text,text,text),
 vec_bolsa_llamamientos.leer_resultado_emision_ct_v1(text,text,text),
 vec_bolsa_llamamientos.leer_resultados_emisiones_ct_v1(jsonb),
 vec_bolsa_llamamientos.listar_emisiones_ct_v1(text,text,timestamptz,text,integer) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.verificar_emision_ct_v1(text,text,text,text),
 vec_bolsa_llamamientos.leer_resultado_emision_ct_v1(text,text,text),
 vec_bolsa_llamamientos.leer_resultados_emisiones_ct_v1(jsonb),
 vec_bolsa_llamamientos.listar_emisiones_ct_v1(text,text,timestamptz,text,integer)
 TO vec_contratacion_temporal_propietario;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_contratacion_temporal_propietario;
COMMIT;
