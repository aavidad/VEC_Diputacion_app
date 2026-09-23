\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000020', 0));

DO $precondiciones$
BEGIN
 IF to_regclass('vec_bolsa_llamamientos.llamamiento_emitido') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)') IS NULL THEN
  RAISE EXCEPTION 'dependencias D3-AV ausentes' USING ERRCODE='55000';
 END IF;
 IF to_regprocedure('vec_bolsa_llamamientos.consultar_avisos_rrhh_v1(timestamptz)') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion 000020 ya aplicada' USING ERRCODE='55000';
 END IF;
END $precondiciones$;

-- Proyección reproducible y sin escritura. El salto usa el orden B6 y la
-- situación que existían exactamente al emitir B7. El aviso de tres años se
-- limita al periodo B2 vigente en «trabajando»; no interpreta encadenamientos.
CREATE FUNCTION vec_bolsa_llamamientos.consultar_avisos_rrhh_v1(p_corte timestamptz)
RETURNS TABLE(tipo text,bolsa_ref text,referencia text,detalle jsonb,fecha timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER
SET search_path=pg_catalog
SET timezone='UTC'
AS $f$
 WITH emisiones AS (
  SELECT l.*
    FROM vec_bolsa_llamamientos.llamamiento_emitido l
   WHERE l.emitido_en<=p_corte
 ), ordenes_emision AS (
  SELECT l.bolsa_ref,l.llamamiento_ref,l.emitido_en,l.participaciones,
         o.participacion_ref,o.orden_vigente,
         min(o.orden_vigente) FILTER (WHERE l.participaciones ? o.participacion_ref) OVER (PARTITION BY l.llamamiento_ref) AS primer_orden,
         max(o.orden_vigente) FILTER (WHERE l.participaciones ? o.participacion_ref) OVER (PARTITION BY l.llamamiento_ref) AS ultimo_orden_llamado
    FROM emisiones l
    CROSS JOIN LATERAL vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(l.bolsa_ref,l.emitido_en) o
 ), saltos AS (
  SELECT 'salto_orden'::text AS tipo,o.bolsa_ref,
         'aviso:salto:'||encode(sha256(convert_to(o.llamamiento_ref||chr(31)||o.participacion_ref,'UTF8')),'hex') AS referencia,
         jsonb_build_object(
           'llamamiento_ref',o.llamamiento_ref,
           'participacion_ref',o.participacion_ref,
           'orden',o.orden_vigente,
           'orden_primero_llamado',o.primer_orden
         ) AS detalle,
         o.emitido_en AS fecha
    FROM ordenes_emision o
   WHERE o.orden_vigente IS NOT NULL AND o.primer_orden IS NOT NULL AND o.ultimo_orden_llamado IS NOT NULL
     -- También detecta huecos internos: si se llama a 1 y 3, la 2 fue
     -- saltada aunque el primer orden llamado sea correcto.
     AND o.orden_vigente<o.ultimo_orden_llamado
     AND NOT (o.participaciones ? o.participacion_ref)
 ), participaciones AS (
  SELECT DISTINCT ON (e.participacion_ref)
         e.participacion_ref,c.bolsa_ref
    FROM vec_bolsa_llamamientos.constitucion_entrada e
    JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea)
   WHERE c.confirmada_en<=p_corte
   ORDER BY e.participacion_ref,c.confirmada_en DESC
 ), trabajando AS (
  SELECT p.bolsa_ref,p.participacion_ref,s.desde
    FROM participaciones p
    CROSS JOIN LATERAL (
      SELECT sp.situacion,sp.desde
        FROM vec_bolsa_llamamientos.situacion_participacion sp
       WHERE sp.participacion_ref=p.participacion_ref AND sp.desde<=p_corte
       ORDER BY sp.desde DESC,sp.clave_idempotencia DESC LIMIT 1
    ) s
   WHERE s.situacion='trabajando'
     AND s.desde+interval '3 years'<=p_corte+interval '30 days'
 ), tres_anos AS (
  SELECT 'tres_anos'::text AS tipo,t.bolsa_ref,
         'aviso:tres-anos:'||encode(sha256(convert_to(t.participacion_ref||chr(31)||t.desde::text,'UTF8')),'hex') AS referencia,
         jsonb_build_object(
           'participacion_ref',t.participacion_ref,
           'inicio_periodo_continuo',t.desde,
           'alcanza_tres_anos_en',t.desde+interval '3 years',
           'estado_calculo',CASE WHEN t.desde+interval '3 years'<=p_corte THEN 'cumplido' ELSE 'proximo_30_dias' END,
           'provisionalidad','Cómputo legal de encadenamiento pendiente de RRHH (duda 13; art. 15.5 ET tras la Ley 20/2021)'
         ) AS detalle,
         t.desde+interval '3 years' AS fecha
    FROM trabajando t
 )
 SELECT * FROM saltos
 UNION ALL
 SELECT * FROM tres_anos
$f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_avisos_rrhh_v1(timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_avisos_rrhh_v1(timestamptz) TO vec_bolsa_llamamientos_ejecutor;
REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_avisos_rrhh_v1(timestamptz) FROM vec_bolsa_llamamientos_ejecutor;
COMMIT;
