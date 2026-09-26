\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000041', 0));

-- Parámetros de los avisos y marcas de Bolsa, publicados desde el catálogo de
-- reglas (vec.bolsa.reglas) al arrancar, como la separación de funciones
-- (000033). Hasta ahora el aviso de tres años tenía el plazo y la antelación
-- fijos en SQL (000020), y las reglas b16 (ya presta servicios) y b17
-- (encadenamiento) eran solo texto. Esta migración:
--  * guarda los parámetros en una política versionada de solo adición; la
--    versión 1 reproduce exactamente la conducta anterior (tres años, treinta
--    días de antelación, sin encadenamiento ni marca de servicios);
--  * añade la bandeja v2, que aplica la versión vigente y el aviso de
--    encadenamiento a partir del histórico de contratos (000024);
--  * añade las marcas por participación que el cuadro y la ficha muestran
--    (en revisión, ya presta servicios, encadenamiento);
--  * si la política dice «excluir», rechaza en la base un llamamiento que
--    incluya a quien ya presta servicios.
-- 000020 queda intacta: la v1 sigue disponible para una aplicación anterior.
-- El histórico de contratos es informativo (ver 000024): el aviso y la marca
-- no cambian situación, orden ni llamamiento de nadie.
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_avisos_rrhh_v1(timestamptz)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.contrato_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.respuesta_portal_llamamiento') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(text)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.llamamiento_emitido') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_avisos_bolsa') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_avisos_rrhh_v2(timestamptz)') IS NOT NULL THEN
  RAISE EXCEPTION 'estado incompatible para los parametros de avisos 000041' USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE TABLE vec_bolsa_llamamientos.politica_avisos_bolsa (
    version bigint PRIMARY KEY CHECK (version >= 1),
    catalogo_ref text NOT NULL CHECK (octet_length(catalogo_ref) BETWEEN 1 AND 512 AND catalogo_ref = btrim(catalogo_ref)),
    catalogo_sha256 text NOT NULL CHECK (catalogo_sha256 ~ '^[a-f0-9]{64}$'),
    -- Aviso de trabajo continuado (b19): plazo y antelación del aviso.
    continuado_meses integer NOT NULL CHECK (continuado_meses BETWEEN 1 AND 600),
    continuado_antelacion_dias integer NOT NULL CHECK (continuado_antelacion_dias BETWEEN 0 AND 3650),
    -- Encadenamiento (b17): más de umbral meses de contrato en la ventana.
    -- Nulos: sin aviso de encadenamiento.
    encadenamiento_umbral_meses integer CHECK (encadenamiento_umbral_meses BETWEEN 1 AND 600),
    encadenamiento_ventana_meses integer CHECK (encadenamiento_ventana_meses BETWEEN 1 AND 1200),
    -- Ya presta servicios (b16): situaciones de otra participación de la
    -- misma persona que cuentan como «trabajando» y qué se hace. Nulos: nada.
    presta_servicios_modo text CHECK (presta_servicios_modo IN ('aviso','excluir')),
    presta_servicios_situaciones text[],
    publicada_en timestamptz(6) NOT NULL,
    -- Constancia de quién publicó: la cuenta de conexión (session_user).
    publicada_por text NOT NULL DEFAULT session_user CHECK (octet_length(publicada_por) BETWEEN 1 AND 128),
    CHECK ((encadenamiento_umbral_meses IS NULL) = (encadenamiento_ventana_meses IS NULL)),
    CHECK (encadenamiento_umbral_meses IS NULL OR encadenamiento_umbral_meses <= encadenamiento_ventana_meses),
    CHECK ((presta_servicios_modo IS NULL) = (presta_servicios_situaciones IS NULL)),
    CHECK (presta_servicios_situaciones IS NULL OR (
        array_ndims(presta_servicios_situaciones) = 1
        AND cardinality(presta_servicios_situaciones) BETWEEN 1 AND 6
        AND array_position(presta_servicios_situaciones, NULL) IS NULL
        AND presta_servicios_situaciones <@ ARRAY['no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde']))
);
COMMENT ON TABLE vec_bolsa_llamamientos.politica_avisos_bolsa IS
    'Versiones de solo adición de los parámetros de avisos y marcas de Bolsa (catálogo vec.bolsa.reglas: b16, b17 y b19). La vigente es la de mayor versión.';
ALTER TABLE vec_bolsa_llamamientos.politica_avisos_bolsa ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.politica_avisos_bolsa FORCE ROW LEVEL SECURITY;
CREATE POLICY politica_avisos_bolsa_solo_propietario ON vec_bolsa_llamamientos.politica_avisos_bolsa TO vec_bolsa_llamamientos_propietario USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.politica_avisos_bolsa FROM PUBLIC;
CREATE TRIGGER politica_avisos_bolsa_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.politica_avisos_bolsa FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Versión 1: exactamente la conducta de 000020 y sin las reglas nuevas.
INSERT INTO vec_bolsa_llamamientos.politica_avisos_bolsa(version, catalogo_ref, catalogo_sha256, continuado_meses, continuado_antelacion_dias, publicada_en)
VALUES (1, 'migracion:bolsa_llamamientos:000041:defecto', encode(sha256(convert_to('36|30', 'UTF8')), 'hex'), 36, 30, clock_timestamp());

-- Versión vigente. Interna: sin EXECUTE para la aplicación.
CREATE FUNCTION vec_bolsa_llamamientos.politica_avisos_bolsa_vigente()
RETURNS vec_bolsa_llamamientos.politica_avisos_bolsa
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT p FROM vec_bolsa_llamamientos.politica_avisos_bolsa p ORDER BY p.version DESC LIMIT 1
$f$;

-- Publica los parámetros del catálogo. Solo crea versión si difieren de la
-- vigente. Un plazo de trabajo continuado nulo conserva el de la versión 1
-- (la conducta anterior); encadenamiento y servicios nulos los desactivan.
-- Quién publica: la aplicación, al arrancar, con la cuenta de ejecución, como
-- en 000033; cada versión guarda la cuenta de conexión que la publicó.
CREATE FUNCTION vec_bolsa_llamamientos.publicar_politica_avisos_bolsa_v1(
 p_catalogo_ref text, p_catalogo_sha256 text, p_continuado_meses integer, p_continuado_antelacion_dias integer,
 p_encadenamiento_umbral_meses integer, p_encadenamiento_ventana_meses integer,
 p_presta_servicios_modo text, p_presta_servicios_situaciones text[])
RETURNS TABLE(version bigint, reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET lock_timeout = '2s' SET statement_timeout = '5s' AS $f$
DECLARE v_vigente vec_bolsa_llamamientos.politica_avisos_bolsa; v_defecto vec_bolsa_llamamientos.politica_avisos_bolsa;
 v_meses integer; v_antelacion integer; v_situaciones text[];
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='operacion no autorizada'; END IF;
 IF p_catalogo_ref IS NULL OR p_catalogo_sha256 IS NULL OR (p_continuado_meses IS NULL) <> (p_continuado_antelacion_dias IS NULL)
    OR (p_presta_servicios_situaciones IS NOT NULL AND (array_ndims(p_presta_servicios_situaciones) IS DISTINCT FROM 1
        OR array_position(p_presta_servicios_situaciones, NULL) IS NOT NULL)) THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='parametros de avisos invalidos';
 END IF;
 -- Situaciones en orden canónico y sin repetidos.
 IF p_presta_servicios_situaciones IS NOT NULL THEN
  v_situaciones := ARRAY(SELECT s FROM unnest(ARRAY['no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde']) WITH ORDINALITY AS c(s, n)
                          WHERE s = ANY (p_presta_servicios_situaciones) ORDER BY n);
  IF cardinality(v_situaciones) <> cardinality(p_presta_servicios_situaciones) THEN
   RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='parametros de avisos invalidos';
  END IF;
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:politica_avisos_bolsa', 0));
 v_vigente := vec_bolsa_llamamientos.politica_avisos_bolsa_vigente();
 SELECT p.* INTO STRICT v_defecto FROM vec_bolsa_llamamientos.politica_avisos_bolsa p WHERE p.version = 1;
 v_meses := coalesce(p_continuado_meses, v_defecto.continuado_meses);
 v_antelacion := coalesce(p_continuado_antelacion_dias, v_defecto.continuado_antelacion_dias);
 IF v_vigente.catalogo_ref = p_catalogo_ref AND v_vigente.catalogo_sha256 = p_catalogo_sha256
    AND v_vigente.continuado_meses = v_meses AND v_vigente.continuado_antelacion_dias = v_antelacion
    AND v_vigente.encadenamiento_umbral_meses IS NOT DISTINCT FROM p_encadenamiento_umbral_meses
    AND v_vigente.encadenamiento_ventana_meses IS NOT DISTINCT FROM p_encadenamiento_ventana_meses
    AND v_vigente.presta_servicios_modo IS NOT DISTINCT FROM p_presta_servicios_modo
    AND v_vigente.presta_servicios_situaciones IS NOT DISTINCT FROM v_situaciones THEN
  RETURN QUERY SELECT v_vigente.version, true;
  RETURN;
 END IF;
 -- Los CHECK de la tabla validan rangos y combinaciones (22023/23514).
 INSERT INTO vec_bolsa_llamamientos.politica_avisos_bolsa(version, catalogo_ref, catalogo_sha256, continuado_meses,
   continuado_antelacion_dias, encadenamiento_umbral_meses, encadenamiento_ventana_meses, presta_servicios_modo,
   presta_servicios_situaciones, publicada_en)
 VALUES (v_vigente.version + 1, p_catalogo_ref, p_catalogo_sha256, v_meses, v_antelacion, p_encadenamiento_umbral_meses,
   p_encadenamiento_ventana_meses, p_presta_servicios_modo, v_situaciones, clock_timestamp());
 RETURN QUERY SELECT v_vigente.version + 1, false;
EXCEPTION WHEN check_violation THEN
 RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='parametros de avisos invalidos';
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_politica_avisos_bolsa_v1()
RETURNS TABLE(version bigint, catalogo_ref text, continuado_meses integer, continuado_antelacion_dias integer,
 encadenamiento_umbral_meses integer, encadenamiento_ventana_meses integer, presta_servicios_modo text, presta_servicios_situaciones text[])
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT p.version, p.catalogo_ref, p.continuado_meses, p.continuado_antelacion_dias, p.encadenamiento_umbral_meses,
        p.encadenamiento_ventana_meses, p.presta_servicios_modo, p.presta_servicios_situaciones
   FROM vec_bolsa_llamamientos.politica_avisos_bolsa_vigente() p
$f$;

-- Situación vigente de cada participación constituida en un instante. Una
-- participación sin fila de situación se considera disponible, como en B2.
CREATE FUNCTION vec_bolsa_llamamientos.situaciones_en_v1(p_corte timestamptz)
RETURNS TABLE(participacion_ref text, bolsa_ref text, clave_persona text, situacion text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
 WITH participaciones AS (
  SELECT DISTINCT ON (e.participacion_ref) e.participacion_ref, c.bolsa_ref
    FROM vec_bolsa_llamamientos.constitucion_entrada e
    JOIN vec_bolsa_llamamientos.constitucion c USING (instantanea_ref, version_instantanea)
   WHERE c.confirmada_en <= p_corte
   ORDER BY e.participacion_ref, c.confirmada_en DESC
 )
 SELECT p.participacion_ref, p.bolsa_ref, coalesce(v.candidato_ref, 'participacion:' || p.participacion_ref),
        coalesce((SELECT s.situacion FROM vec_bolsa_llamamientos.situacion_participacion s
                   WHERE s.participacion_ref = p.participacion_ref AND s.desde <= p_corte
                   ORDER BY s.desde DESC, s.clave_idempotencia DESC LIMIT 1), 'disponible')
   FROM participaciones p
   LEFT JOIN vec_bolsa_llamamientos.vinculo_candidato v ON v.participacion_ref = p.participacion_ref
$f$;

-- Participaciones cuya persona presta ya servicios por OTRA participación
-- (de esta bolsa en otra constitución o de otra bolsa), según la política.
CREATE FUNCTION vec_bolsa_llamamientos.presta_servicios_en_v1(p_corte timestamptz)
RETURNS TABLE(participacion_ref text, modo text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
 WITH p AS (SELECT * FROM vec_bolsa_llamamientos.politica_avisos_bolsa_vigente()),
 s AS (SELECT * FROM vec_bolsa_llamamientos.situaciones_en_v1(p_corte))
 SELECT a.participacion_ref, p.presta_servicios_modo
   FROM s a CROSS JOIN p
  WHERE p.presta_servicios_modo IS NOT NULL
    AND EXISTS (SELECT 1 FROM s b WHERE b.clave_persona = a.clave_persona AND b.participacion_ref <> a.participacion_ref
                  AND b.situacion = ANY (p.presta_servicios_situaciones))
$f$;

-- Encadenamiento por persona (vínculo de candidato; sin vínculo, la propia
-- participación): tiempo de contrato dentro de la ventana que termina en el
-- corte, sin contar dos veces los solapes. Un contrato es un llamamiento del
-- histórico: empieza en su inicio y termina en el cese, en su fin previsto o,
-- si sigue abierto, en el corte.
CREATE FUNCTION vec_bolsa_llamamientos.encadenamiento_en_v1(p_corte timestamptz)
RETURNS TABLE(clave_persona text, participacion_ref text, bolsa_ref text, segundos numeric, contratos bigint,
 ultimo_inicio timestamptz, umbral_meses integer, ventana_meses integer)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
 WITH p AS (SELECT * FROM vec_bolsa_llamamientos.politica_avisos_bolsa_vigente()),
 contratos AS (
  SELECT coalesce(v.candidato_ref, 'participacion:' || c.participacion_ref) AS clave_persona,
         c.llamamiento_ref,
         (array_agg(c.participacion_ref ORDER BY c.ocurrido_en DESC))[1] AS participacion_ref,
         (array_agg(c.bolsa_ref ORDER BY c.ocurrido_en DESC))[1] AS bolsa_ref,
         min(c.inicio) AS inicio,
         coalesce(max(c.fin_previsto) FILTER (WHERE c.tipo = 'cese'), max(c.fin_previsto) FILTER (WHERE c.tipo = 'incorporacion'), p_corte) AS fin
    FROM vec_bolsa_llamamientos.contrato_participacion c
    LEFT JOIN vec_bolsa_llamamientos.vinculo_candidato v ON v.participacion_ref = c.participacion_ref
   WHERE c.participacion_ref IS NOT NULL AND c.recibido_en <= p_corte
   GROUP BY 1, 2
 ), recortes AS (
  SELECT c.*, tstzrange(greatest(c.inicio, p_corte - make_interval(months => p.encadenamiento_ventana_meses)), least(c.fin, p_corte), '[)') AS tramo
    FROM contratos c CROSS JOIN p
   WHERE p.encadenamiento_umbral_meses IS NOT NULL AND c.inicio IS NOT NULL AND c.inicio < p_corte
 ), personas AS (
  SELECT r.clave_persona, range_agg(r.tramo) AS tramos, count(*) AS contratos,
         (array_agg(r.participacion_ref ORDER BY r.inicio DESC, r.llamamiento_ref DESC))[1] AS participacion_ref,
         (array_agg(r.bolsa_ref ORDER BY r.inicio DESC, r.llamamiento_ref DESC))[1] AS bolsa_ref,
         max(r.inicio) AS ultimo_inicio
    FROM recortes r WHERE NOT isempty(r.tramo)
   GROUP BY r.clave_persona
 )
 SELECT x.clave_persona, x.participacion_ref, x.bolsa_ref,
        (SELECT coalesce(sum(extract(epoch FROM upper(t) - lower(t))), 0) FROM unnest(x.tramos) t),
        x.contratos, x.ultimo_inicio, p.encadenamiento_umbral_meses, p.encadenamiento_ventana_meses
   FROM personas x CROSS JOIN p
$f$;

-- Bandeja v2: v1 para los saltos de orden, el trabajo continuado con los
-- parámetros vigentes y el encadenamiento. Mismo contrato de columnas.
CREATE FUNCTION vec_bolsa_llamamientos.consultar_avisos_rrhh_v2(p_corte timestamptz)
RETURNS TABLE(tipo text, bolsa_ref text, referencia text, detalle jsonb, fecha timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
 WITH p AS (SELECT * FROM vec_bolsa_llamamientos.politica_avisos_bolsa_vigente()),
 trabajando AS (
  SELECT s.bolsa_ref, s.participacion_ref, t.desde
    FROM vec_bolsa_llamamientos.situaciones_en_v1(p_corte) s
    CROSS JOIN LATERAL (
      SELECT sp.desde FROM vec_bolsa_llamamientos.situacion_participacion sp
       WHERE sp.participacion_ref = s.participacion_ref AND sp.desde <= p_corte
       ORDER BY sp.desde DESC, sp.clave_idempotencia DESC LIMIT 1) t
   WHERE s.situacion = 'trabajando'
 )
 SELECT v.tipo, v.bolsa_ref, v.referencia, v.detalle, v.fecha
   FROM vec_bolsa_llamamientos.consultar_avisos_rrhh_v1(p_corte) v WHERE v.tipo = 'salto_orden'
 UNION ALL
 SELECT 'tres_anos'::text, t.bolsa_ref,
        'aviso:tres-anos:' || encode(sha256(convert_to(t.participacion_ref || chr(31) || t.desde::text, 'UTF8')), 'hex'),
        jsonb_build_object(
          'participacion_ref', t.participacion_ref,
          'inicio_periodo_continuo', t.desde,
          'alcanza_tres_anos_en', t.desde + make_interval(months => p.continuado_meses),
          'plazo_meses', p.continuado_meses,
          'antelacion_dias', p.continuado_antelacion_dias,
          'estado_calculo', CASE WHEN t.desde + make_interval(months => p.continuado_meses) <= p_corte THEN 'cumplido' ELSE 'proximo' END),
        t.desde + make_interval(months => p.continuado_meses)
   FROM trabajando t CROSS JOIN p
  WHERE t.desde + make_interval(months => p.continuado_meses) <= p_corte + make_interval(days => p.continuado_antelacion_dias)
 UNION ALL
 SELECT 'encadenamiento'::text, e.bolsa_ref,
        'aviso:encadenamiento:' || encode(sha256(convert_to(e.clave_persona || chr(31) || e.umbral_meses || chr(31) || e.ventana_meses, 'UTF8')), 'hex'),
        jsonb_build_object(
          'participacion_ref', e.participacion_ref,
          'dias_acumulados', floor(e.segundos / 86400)::bigint,
          'contratos', e.contratos,
          'umbral_meses', e.umbral_meses,
          'ventana_meses', e.ventana_meses),
        e.ultimo_inicio
   FROM vec_bolsa_llamamientos.encadenamiento_en_v1(p_corte) e
  WHERE e.segundos > extract(epoch FROM p_corte - (p_corte - make_interval(months => e.umbral_meses)))
$f$;

-- Marcas de las participaciones de una bolsa para el cuadro y la ficha. Solo
-- devuelve las que tienen alguna marca. en_revision: renuncia comunicada desde
-- el portal que RRHH aún no ha reflejado en la situación, o solicitud del
-- portal pendiente de validar.
CREATE FUNCTION vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(p_bolsa_ref text, p_corte timestamptz)
RETURNS TABLE(participacion_ref text, presta_servicios text, en_revision text, encadenamiento_dias bigint,
 encadenamiento_umbral_meses integer, encadenamiento_ventana_meses integer)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
 WITH propias AS (
  SELECT s.participacion_ref, s.clave_persona FROM vec_bolsa_llamamientos.situaciones_en_v1(p_corte) s WHERE s.bolsa_ref = p_bolsa_ref
 ), presta AS (SELECT * FROM vec_bolsa_llamamientos.presta_servicios_en_v1(p_corte)),
 encadenadas AS (
  SELECT e.* FROM vec_bolsa_llamamientos.encadenamiento_en_v1(p_corte) e
   WHERE e.segundos > extract(epoch FROM p_corte - (p_corte - make_interval(months => e.umbral_meses)))
 ), revision AS (
  SELECT r.participacion_ref, 'renuncia_pendiente'::text AS motivo
    FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento r
   WHERE r.bolsa_ref = p_bolsa_ref AND r.respondida_en <= p_corte AND r.respuesta IN ('renuncia','renuncia_justificada')
     AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s
                      WHERE s.participacion_ref = r.participacion_ref AND s.desde >= r.respondida_en AND s.desde <= p_corte)
  UNION ALL
  SELECT s.participacion_ref, 'solicitud_pendiente'::text
    FROM vec_bolsa_llamamientos.solicitud_portal_candidato s
   WHERE s.bolsa_ref = p_bolsa_ref AND s.registrada_en <= p_corte AND vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(s.solicitud_ref)
 )
 SELECT o.participacion_ref, ps.modo,
        (SELECT min(r.motivo) FROM revision r WHERE r.participacion_ref = o.participacion_ref),
        floor(e.segundos / 86400)::bigint, e.umbral_meses, e.ventana_meses
   FROM propias o
   LEFT JOIN presta ps ON ps.participacion_ref = o.participacion_ref
   LEFT JOIN encadenadas e ON e.clave_persona = o.clave_persona
  WHERE ps.modo IS NOT NULL OR e.clave_persona IS NOT NULL
     OR EXISTS (SELECT 1 FROM revision r WHERE r.participacion_ref = o.participacion_ref)
  ORDER BY o.participacion_ref
  LIMIT 20000
$f$;

-- Con la política en «excluir», la base rechaza el llamamiento que incluya a
-- quien ya presta servicios; en «aviso» solo se marca.
CREATE FUNCTION vec_bolsa_llamamientos.exigir_no_presta_servicios()
RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
BEGIN
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_bolsa_llamamientos:politica_avisos_bolsa', 0));
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.presta_servicios_en_v1(NEW.emitido_en) x
             WHERE x.modo = 'excluir' AND NEW.participaciones ? x.participacion_ref) THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='participacion que ya presta servicios';
 END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.exigir_no_presta_servicios() FROM PUBLIC;
CREATE TRIGGER llamamiento_emitido_presta_servicios BEFORE INSERT ON vec_bolsa_llamamientos.llamamiento_emitido
    FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.exigir_no_presta_servicios();

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.politica_avisos_bolsa_vigente() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.publicar_politica_avisos_bolsa_v1(text,text,integer,integer,integer,integer,text,text[]) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_politica_avisos_bolsa_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.situaciones_en_v1(timestamptz) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.presta_servicios_en_v1(timestamptz) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.encadenamiento_en_v1(timestamptz) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_avisos_rrhh_v2(timestamptz) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(text,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_politica_avisos_bolsa_v1(text,text,integer,integer,integer,integer,text,text[]) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_politica_avisos_bolsa_v1() TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_avisos_rrhh_v2(timestamptz) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(text,timestamptz) TO vec_bolsa_llamamientos_ejecutor;
REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_avisos_rrhh_v2(timestamptz) FROM vec_bolsa_llamamientos_ejecutor;
COMMIT;
