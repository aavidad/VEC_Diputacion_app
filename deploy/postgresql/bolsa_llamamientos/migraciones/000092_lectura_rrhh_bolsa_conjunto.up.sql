\set ON_ERROR_STOP on
-- B92: una lectura de conjunto acotada a una bolsa constituida vigente.
-- El corte se aplica a situaciones y ceses; la constitución vigente coincide
-- con la última por categoría de listar_constituciones_v1 (B14).
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000092',0));

DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.constitucion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.bolsa_constituida') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.instantanea_orden_bolsa') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.constitucion_entrada') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.listar_constituciones_v1()') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.estado_cese_bolsa_lote_interno_v2(text[],timestamptz)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(text,timestamptz)') IS NOT NULL THEN
  RAISE EXCEPTION 'B92: clave=preimagen_lectura_bolsa actual=incompatible_o_ya_instalada'
   USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(
 p_bolsa_ref text,p_corte timestamptz)
RETURNS TABLE(bolsa_ref text,categoria_ref text,confirmada_en timestamptz,
 instantanea_ref text,version_instantanea bigint,orden bigint,participacion_ref text,
 situacion text,desde timestamptz,fecha_disponible timestamptz,
 cese_fecha_efecto date,cese_disponible_desde date,cese_en_restriccion boolean,
 cese_trabajo_cesado boolean,cese_pendiente boolean,pendiente_desde timestamptz,
 fila_numero integer)
LANGUAGE plpgsql STABLE SECURITY DEFINER ROWS 20000
SET search_path=pg_catalog,pg_temp SET statement_timeout='5s' AS $f$
DECLARE
 v_constitucion record;
 v_coincidencias integer:=0;
 v_entradas bigint;
 v_referencias text[];
 v_empates bigint;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_relevo_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR p_bolsa_ref IS NULL OR octet_length(p_bolsa_ref) NOT BETWEEN 1 AND 512
    OR p_corte IS NULL OR NOT isfinite(p_corte) THEN
  RAISE EXCEPTION 'B92: lectura de bolsa denegada' USING ERRCODE='42501';
 END IF;

 -- B14 ordena por confirmada_en y registrada_en, sin filtrar antes de
 -- elegir la última categoría. Un empate no obtiene una bolsa arbitraria.
 FOR v_constitucion IN
  SELECT c.bolsa_ref,c.categoria_ref,c.confirmada_en,c.registrada_en,
         c.instantanea_ref,c.version_instantanea,i.total_participaciones,
         b.estado,b.vigente_desde,
         coalesce(s.sustituida_en,b.vigente_hasta) AS vigente_hasta,
         s.bolsa_ref_sustituida IS NOT NULL AS sustituida
    FROM vec_bolsa_llamamientos.constitucion c
    JOIN vec_bolsa_llamamientos.bolsa_constituida b
      ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa
     AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
    JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i
      ON i.instantanea_ref=c.instantanea_ref AND i.version=c.version_instantanea
     AND i.huella_instantanea_sha256=c.huella_instantanea_sha256
    LEFT JOIN vec_bolsa_llamamientos.sustitucion_bolsa s
      ON s.bolsa_ref_sustituida=b.bolsa_ref AND s.version_sustituida=b.version
     AND s.huella_sustituida_sha256=b.huella_bolsa_sha256
   WHERE c.bolsa_ref=p_bolsa_ref
     AND NOT EXISTS (
      SELECT 1 FROM vec_bolsa_llamamientos.constitucion posterior
       WHERE posterior.categoria_ref=c.categoria_ref
         AND (posterior.confirmada_en,posterior.registrada_en)>
             (c.confirmada_en,c.registrada_en))
 LOOP
  v_coincidencias:=v_coincidencias+1;
  IF v_coincidencias>1 THEN
   RAISE EXCEPTION 'B92: clave=constitucion_ambigua bolsa=% coincidencias=%',p_bolsa_ref,v_coincidencias
    USING ERRCODE='55000';
  END IF;
 END LOOP;
 IF v_coincidencias=0 THEN RETURN; END IF;

 SELECT count(*) INTO v_empates
   FROM vec_bolsa_llamamientos.constitucion c
  WHERE c.categoria_ref=v_constitucion.categoria_ref
    AND c.confirmada_en=v_constitucion.confirmada_en
    AND c.registrada_en=v_constitucion.registrada_en;
 IF v_empates<>1 THEN
  RAISE EXCEPTION 'B92: clave=constitucion_empate bolsa=% categoria=% coincidencias=%',
   p_bolsa_ref,v_constitucion.categoria_ref,v_empates USING ERRCODE='55000';
 END IF;
 IF v_constitucion.sustituida OR v_constitucion.estado<>'vigente'
    OR v_constitucion.vigente_desde>p_corte
    OR (v_constitucion.vigente_hasta IS NOT NULL AND v_constitucion.vigente_hasta<=p_corte) THEN
  RETURN;
 END IF;

 SELECT count(*),array_agg(e.participacion_ref ORDER BY e.orden)
   INTO v_entradas,v_referencias
   FROM vec_bolsa_llamamientos.constitucion_entrada e
  WHERE e.instantanea_ref=v_constitucion.instantanea_ref
    AND e.version_instantanea=v_constitucion.version_instantanea;
 IF v_entradas=0 OR v_entradas>20000
    OR v_entradas<>v_constitucion.total_participaciones THEN
  RAISE EXCEPTION 'B92: clave=entradas_bolsa bolsa=% actual=% esperado=% maximo=20000',
   p_bolsa_ref,v_entradas,v_constitucion.total_participaciones USING ERRCODE='55000';
 END IF;

 RETURN QUERY
 WITH ceses AS MATERIALIZED (
  SELECT x.* FROM vec_bolsa_llamamientos.estado_cese_bolsa_lote_interno_v2(v_referencias,p_corte) x
 )
 SELECT v_constitucion.bolsa_ref,v_constitucion.categoria_ref,v_constitucion.confirmada_en,
        v_constitucion.instantanea_ref,v_constitucion.version_instantanea,
        e.orden,e.participacion_ref,sp.situacion,sp.desde,sp.fecha_disponible,
        CASE WHEN x.cese_pendiente THEN NULL ELSE x.fecha_efecto END,
        CASE WHEN x.cese_pendiente THEN NULL ELSE x.disponible_desde END,
        CASE WHEN x.cese_pendiente THEN true ELSE x.en_restriccion END,
        CASE WHEN x.cese_pendiente THEN false ELSE x.trabajo_cesado END,
        coalesce(x.cese_pendiente,false),
        CASE WHEN x.cese_pendiente THEN x.pendiente_desde ELSE NULL END,
        e.fila_numero
   FROM vec_bolsa_llamamientos.constitucion_entrada e
   LEFT JOIN LATERAL (
    SELECT s.situacion,s.desde,s.fecha_disponible
      FROM vec_bolsa_llamamientos.situacion_participacion s
     WHERE s.participacion_ref=e.participacion_ref AND s.desde<=p_corte
     ORDER BY s.desde DESC LIMIT 1
   ) sp ON true
   LEFT JOIN ceses x ON x.participacion_ref=e.participacion_ref
  WHERE e.instantanea_ref=v_constitucion.instantanea_ref
    AND e.version_instantanea=v_constitucion.version_instantanea
  ORDER BY e.orden;
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(text,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(text,timestamptz)
 TO vec_bolsa_llamamientos_ejecutor;

DO $post$
DECLARE f regprocedure:='vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(text,timestamptz)'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT 'search_path=pg_catalog, pg_temp'=ANY(proconfig) FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT array_agg(a.grantee::regrole::text ORDER BY a.grantee::regrole::text)
          FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
         WHERE p.oid=f) IS DISTINCT FROM
       ARRAY['vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos_propietario']
    OR has_function_privilege('vec_bolsa_llamamientos_relevo_cese',f,'EXECUTE') THEN
  RAISE EXCEPTION 'B92: clave=postimagen_lectura_bolsa actual=ACL_o_definicion_divergente'
   USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
