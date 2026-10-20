\set ON_ERROR_STOP on
-- CT243 / F1-97. «Rectificar análisis» no podía conservar el grupo ni la
-- entrada de retención de crédito elegidos porque el detalle RRHH no los
-- publica. Las cuatro lecturas del detalle de CT225 añaden, al final y fuera
-- del recibo, el grupo/subgrupo y la referencia de la entrada de retención de
-- crédito del análisis de la misma versión que devolvió el recibo.
--
-- - El recibo, su contenido canónico y sus huellas no cambian.
-- - Mismo permiso: solo vec_contratacion_temporal_consultor_rrhh ejecuta las
--   cuatro lecturas; nadie más lee los dos datos nuevos.
-- - Sin datos personales: son un código de grupo y una referencia opaca de
--   catálogo. La referencia identifica la entrada elegida al analizar; no
--   acredita crédito (la retención del centro sigue siendo declarada, F1-87).
-- - Las cuatro funciones se reconstruyen con definición explícita (DROP y
--   CREATE, porque cambian sus columnas de salida), tras comprobar su
--   preimagen exacta por sha256(pg_get_functiondef). Solo UP; ninguna otra
--   SQL las llama (lectura nominal de la aplicación).
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000243',0));

DO $pre$
DECLARE r record; n integer:=0;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.analisis_detalle_rrhh_ct243(text,text,numeric)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT243: preimagen CT225 incompatible' USING ERRCODE='55000'; END IF;
 FOR r IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.consultar_detalle_rrhh_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'e13eac702bb0ed958ec7697c66a1a4e84264aba7257db96156762e5f75073cb7'),
  ('vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)',
   'b3b7fdc0a8598d8353501094cb33969248d16f8638a8b47b6b2e81e1581f7b08'),
  ('vec_contratacion_temporal.consultar_detalle_rrhh_selectores_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)',
   '3b02aae33798f806172c13b88bd0551dab7da7c6fbc662066645ad847cdcec78'),
  ('vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_selectores_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,jsonb)',
   'af3401c1846c369e5a3698d4d1172b2212b8f3fed5e72ba3d0bf958ec604c0b8'),
  ('vec_contratacion_temporal.numero_personas_detalle_rrhh_ct225(text,text,numeric)',
   'a8608f16b59f789c09e194dc87c97ba8e7a5273221a99d4312b0499222438140')
 ) AS v(firma,huella) LOOP
  IF to_regprocedure(r.firma) IS NULL
     OR encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(r.firma)),'UTF8')),'hex')
        IS DISTINCT FROM r.huella
  THEN RAISE EXCEPTION 'CT243: definición instalada distinta de CT225: %', r.firma USING ERRCODE='55000'; END IF;
  n:=n+1;
 END LOOP;
 IF n<>5 THEN RAISE EXCEPTION 'CT243: preimagen incompleta' USING ERRCODE='55000'; END IF;
END $pre$;

-- Grupo y entrada de retención del análisis de la versión exacta del recibo.
-- Sin análisis, ambos nulos. Con análisis, ambos obligatorios (CT15 los exige
-- al persistir). Solo la ejecuta su propietario, desde las cuatro lecturas.
CREATE FUNCTION vec_contratacion_temporal.analisis_detalle_rrhh_ct243(
 p_organizacion text,p_expediente text,p_version numeric,
 OUT grupo_subgrupo text,OUT entrada_rc_ref text)
LANGUAGE plpgsql STABLE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' AS $f$
DECLARE v_analisis jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' THEN
  RAISE EXCEPTION 'CT243: análisis no autorizable' USING ERRCODE='42501';
 END IF;
 SELECT v.agregado_json->'analisis' INTO STRICT v_analisis
 FROM vec_contratacion_temporal.expediente_version_integral v
 WHERE v.expediente_ref=p_expediente AND v.version=p_version
   AND v.agregado_json->>'organizacion_ref'=p_organizacion;
 IF v_analisis IS NULL OR jsonb_typeof(v_analisis)='null' THEN
  RETURN;
 END IF;
 IF jsonb_typeof(v_analisis)<>'object'
    OR jsonb_typeof(v_analisis->'grupo_subgrupo') IS DISTINCT FROM 'string'
    OR jsonb_typeof(v_analisis#>'{entrada_rc_esperada,referencia}') IS DISTINCT FROM 'string'
 THEN RAISE EXCEPTION 'CT243: análisis histórico inválido' USING ERRCODE='55000'; END IF;
 grupo_subgrupo:=v_analisis->>'grupo_subgrupo';
 entrada_rc_ref:=v_analisis#>>'{entrada_rc_esperada,referencia}';
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.analisis_detalle_rrhh_ct243(text,text,numeric) FROM PUBLIC;

DROP FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_numero_ct225(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_con_numero_ct225(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text);
DROP FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_selectores_con_numero_ct225(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb);
DROP FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_selectores_con_numero_ct225(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,jsonb);

-- Las cuatro lecturas devuelven el recibo original sin tocar sus bytes ni su
-- hash y añaden la cifra (CT225) y el grupo y la entrada de retención del
-- análisis (CT243), en la misma transacción. Cada una exige una fila.
CREATE FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_numero_ct225(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(contenido_canonico bytea,esquema text,acceso_ref text,secuencia numeric,
 anterior_sha256 text,huella_sha256 text,vinculo_identidad_huella_sha256 text,
 alcance_huella_sha256 text,registrada_en timestamptz,auditoria_vec_ref text,
 auditoria_vec_huella_sha256 text,consumo_vec_huella_sha256 text,
 contenido_huella_sha256 text,resultado_huella_sha256 text,cursor_huella_sha256 text,
 generada_en timestamptz,expediente_ref text,version_expediente numeric,total smallint,
 recibo_sello_sha256 text,
 numero_personas smallint,analisis_grupo_subgrupo text,analisis_entrada_rc_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s' AS $f$
DECLARE n bigint;
BEGIN
 RETURN QUERY SELECT d.*,vec_contratacion_temporal.numero_personas_detalle_rrhh_ct225(
  p_alcance.organizacion_ref,d.expediente_ref,d.version_expediente),
  a.grupo_subgrupo,a.entrada_rc_ref
 FROM vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(p_alcance,p_consulta,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz) d
 CROSS JOIN LATERAL vec_contratacion_temporal.analisis_detalle_rrhh_ct243(
  p_alcance.organizacion_ref,d.expediente_ref,d.version_expediente) a;
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION 'CT225: detalle no único' USING ERRCODE='55000'; END IF;
END $f$;

CREATE FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_con_numero_ct225(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea,p_cursor text)
RETURNS TABLE(contenido_canonico bytea,esquema text,acceso_ref text,secuencia numeric,
 anterior_sha256 text,huella_sha256 text,vinculo_identidad_huella_sha256 text,
 alcance_huella_sha256 text,registrada_en timestamptz,auditoria_vec_ref text,
 auditoria_vec_huella_sha256 text,consumo_vec_huella_sha256 text,
 contenido_huella_sha256 text,resultado_huella_sha256 text,cursor_huella_sha256 text,
 generada_en timestamptz,expediente_ref text,version_expediente numeric,total smallint,
 recibo_sello_sha256 text,resultado_bolsa jsonb,
 numero_personas smallint,analisis_grupo_subgrupo text,analisis_entrada_rc_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s' AS $f$
DECLARE n bigint;
BEGIN
 RETURN QUERY SELECT d.*,vec_contratacion_temporal.numero_personas_detalle_rrhh_ct225(
  p_alcance.organizacion_ref,d.expediente_ref,d.version_expediente),
  a.grupo_subgrupo,a.entrada_rc_ref
 FROM vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(p_alcance,p_consulta,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz,p_cursor) d
 CROSS JOIN LATERAL vec_contratacion_temporal.analisis_detalle_rrhh_ct243(
  p_alcance.organizacion_ref,d.expediente_ref,d.version_expediente) a;
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION 'CT225: detalle no único' USING ERRCODE='55000'; END IF;
END $f$;

CREATE FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_selectores_con_numero_ct225(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea,p_filtro jsonb)
RETURNS TABLE(contenido_canonico bytea,esquema text,acceso_ref text,secuencia numeric,
 anterior_sha256 text,huella_sha256 text,vinculo_identidad_huella_sha256 text,
 alcance_huella_sha256 text,registrada_en timestamptz,auditoria_vec_ref text,
 auditoria_vec_huella_sha256 text,consumo_vec_huella_sha256 text,
 contenido_huella_sha256 text,resultado_huella_sha256 text,cursor_huella_sha256 text,
 generada_en timestamptz,expediente_ref text,version_expediente numeric,total smallint,
 recibo_sello_sha256 text,selectores_canon bytea,selectores_huella_sha256 text,
 sello_selectores_sha256 text,recibo_selectores_canon bytea,
 numero_personas smallint,analisis_grupo_subgrupo text,analisis_entrada_rc_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s' AS $f$
DECLARE n bigint;
BEGIN
 RETURN QUERY SELECT d.*,vec_contratacion_temporal.numero_personas_detalle_rrhh_ct225(
  p_alcance.organizacion_ref,d.expediente_ref,d.version_expediente),
  a.grupo_subgrupo,a.entrada_rc_ref
 FROM vec_contratacion_temporal.consultar_detalle_rrhh_con_selectores_v1(p_alcance,p_consulta,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz,p_filtro) d
 CROSS JOIN LATERAL vec_contratacion_temporal.analisis_detalle_rrhh_ct243(
  p_alcance.organizacion_ref,d.expediente_ref,d.version_expediente) a;
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION 'CT225: detalle no único' USING ERRCODE='55000'; END IF;
END $f$;

CREATE FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_selectores_con_numero_ct225(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea,p_cursor text,p_filtro jsonb)
RETURNS TABLE(contenido_canonico bytea,esquema text,acceso_ref text,secuencia numeric,
 anterior_sha256 text,huella_sha256 text,vinculo_identidad_huella_sha256 text,
 alcance_huella_sha256 text,registrada_en timestamptz,auditoria_vec_ref text,
 auditoria_vec_huella_sha256 text,consumo_vec_huella_sha256 text,
 contenido_huella_sha256 text,resultado_huella_sha256 text,cursor_huella_sha256 text,
 generada_en timestamptz,expediente_ref text,version_expediente numeric,total smallint,
 recibo_sello_sha256 text,resultado_bolsa jsonb,selectores_canon bytea,
 selectores_huella_sha256 text,sello_selectores_sha256 text,recibo_selectores_canon bytea,
 numero_personas smallint,analisis_grupo_subgrupo text,analisis_entrada_rc_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s' AS $f$
DECLARE n bigint;
BEGIN
 RETURN QUERY SELECT d.*,vec_contratacion_temporal.numero_personas_detalle_rrhh_ct225(
  p_alcance.organizacion_ref,d.expediente_ref,d.version_expediente),
  a.grupo_subgrupo,a.entrada_rc_ref
 FROM vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_y_selectores_v1(p_alcance,p_consulta,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz,p_cursor,p_filtro) d
 CROSS JOIN LATERAL vec_contratacion_temporal.analisis_detalle_rrhh_ct243(
  p_alcance.organizacion_ref,d.expediente_ref,d.version_expediente) a;
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION 'CT225: detalle no único' USING ERRCODE='55000'; END IF;
END $f$;

-- Mismas concesiones que en CT225: solo el consultor RRHH, nunca PUBLIC.
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_numero_ct225(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_con_numero_ct225(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_selectores_con_numero_ct225(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_selectores_con_numero_ct225(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_numero_ct225(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_consultor_rrhh;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_con_numero_ct225(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)
 TO vec_contratacion_temporal_consultor_rrhh;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_selectores_con_numero_ct225(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)
 TO vec_contratacion_temporal_consultor_rrhh;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_selectores_con_numero_ct225(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,jsonb)
 TO vec_contratacion_temporal_consultor_rrhh;

DO $post$
DECLARE r record; v_huella text; v_fallos text:=''; n integer:=0;
BEGIN
 FOR r IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.consultar_detalle_rrhh_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   '2a6a9d19a8209c7159abc19e3746e449021cc9ea07f8603579884eee0333df53'),
  ('vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)',
   'd523263a679c192a898af7352773bd5191ddf6c49d9d6f7cd0b3fcad38dc76e4'),
  ('vec_contratacion_temporal.consultar_detalle_rrhh_selectores_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)',
   '7b311de07038dfcc525286a124d3dfba660cfbc4c59a9b01d50b487407c22542'),
  ('vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_selectores_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,jsonb)',
   '2ee210d06ebe7ef20b4993d06175aeb8828149eebe6c3a1545a8d5a597b3080e'),
  ('vec_contratacion_temporal.analisis_detalle_rrhh_ct243(text,text,numeric)',
   '0097c2563bb84c7a56251c113c8cff332d2dc5ff3f16a478a41b9bd4db9f1fc9')
 ) AS v(firma,huella) LOOP
  v_huella:=encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(r.firma)),'UTF8')),'hex');
  IF v_huella IS DISTINCT FROM r.huella THEN
   v_fallos:=v_fallos||E'\n'||r.firma||' '||coalesce(v_huella,'-');
  ELSE n:=n+1;
  END IF;
 END LOOP;
 IF v_fallos<>'' THEN
  RAISE EXCEPTION 'CT243: postimagen divergente:%', v_fallos USING ERRCODE='55000';
 END IF;
 SELECT n+count(*) INTO n FROM pg_proc p WHERE p.oid IN (
   to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
   to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)'),
   to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_selectores_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)'),
   to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_selectores_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,jsonb)'))
  AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef
  AND p.provolatile='v'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s']::text[]
  AND p.proacl::text='{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario}';
 IF n<>9
    OR NOT EXISTS (SELECT 1 FROM pg_proc p
      WHERE p.oid=to_regprocedure('vec_contratacion_temporal.analisis_detalle_rrhh_ct243(text,text,numeric)')
      AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND NOT p.prosecdef
      AND p.provolatile='s'
      AND p.proacl::text='{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}')
 THEN RAISE EXCEPTION 'CT243: postimagen o ACL divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
