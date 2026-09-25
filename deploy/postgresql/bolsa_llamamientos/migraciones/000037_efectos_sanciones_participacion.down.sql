\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000037:down', 0));
-- Solo para ensayo aislado: con penalizaciones, reversiones o suspensiones
-- con fin registradas no se deshace.
DO $f$ BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.penalizacion_orden_sancion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.reversion_sancion_participacion') IS NULL THEN
  RAISE EXCEPTION 'estado incompatible para retirar los efectos de las sanciones' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.penalizacion_orden_sancion)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.reversion_sancion_participacion)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.sancion_participacion s
                 JOIN vec_bolsa_llamamientos.situacion_participacion sp ON sp.participacion_ref = s.participacion_ref AND sp.desde = s.situacion_desde
                WHERE sp.situacion = 'disponible_desde') THEN
  RAISE EXCEPTION 'hay historia de efectos de sanciones; no se deshace' USING ERRCODE='55000';
 END IF;
END $f$;
DROP FUNCTION vec_bolsa_llamamientos.listar_sanciones_participacion_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v2(text,text,text,date,text,text,text,text,timestamptz,boolean,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.registrar_sancion_participacion_v2(text,text,text,text,text,text,text,date,text,text,text,text,text,date,date,text,text,timestamptz,text,text,text,timestamptz,boolean,boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.registrar_suspension_con_fin_v1(text,text,text,text,text,text,date,text,text,text,text,text,date,date,text,text,timestamptz,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
-- Cuerpo exacto de 000018.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(p_bolsa_ref text,p_en timestamptz)
RETURNS TABLE(politica_ref text,version_politica bigint,criterio text,tipo_lista text,reposicion text,provisional boolean,rotulo text,actor text,vigente_desde timestamptz,participacion_ref text,orden_acta bigint,orden_vigente bigint,situacion text,razon text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 WITH politica AS (
  SELECT p.* FROM vec_bolsa_llamamientos.politica_orden_bolsa p
   WHERE p.bolsa_ref=p_bolsa_ref AND p.vigente_desde<=p_en AND (p.vigente_hasta IS NULL OR p.vigente_hasta>p_en)
   ORDER BY p.version DESC LIMIT 1
 ), base AS (
  SELECT e.participacion_ref,e.orden AS orden_acta,s.situacion,s.fecha_disponible,
         (s.situacion='disponible' OR (s.situacion='disponible_desde' AND s.fecha_disponible<=p_en)) AS ocupa_turno,
         r.aplicada_en AS repuesta_en
    FROM vec_bolsa_llamamientos.constitucion c
    JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
    JOIN LATERAL (SELECT sp.situacion,sp.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion sp WHERE sp.participacion_ref=e.participacion_ref AND sp.desde<=p_en ORDER BY sp.desde DESC LIMIT 1) s ON true
    LEFT JOIN LATERAL (SELECT ro.aplicada_en FROM vec_bolsa_llamamientos.reposicion_orden_bolsa ro WHERE ro.bolsa_ref=c.bolsa_ref AND ro.participacion_ref=e.participacion_ref AND ro.aplicada_en<=p_en ORDER BY ro.aplicada_en DESC LIMIT 1) r ON true
   WHERE c.bolsa_ref=p_bolsa_ref
 ), elegibles AS (
  SELECT b.participacion_ref,row_number() OVER(ORDER BY
    CASE WHEN p.reposicion='fin_lista' AND b.repuesta_en IS NOT NULL THEN 1 ELSE 0 END,
    CASE WHEN p.reposicion='fin_lista' AND b.repuesta_en IS NOT NULL THEN NULL ELSE b.orden_acta END,
    b.repuesta_en,b.orden_acta,b.participacion_ref)::bigint AS orden_vigente
   FROM base b CROSS JOIN politica p WHERE b.ocupa_turno
 )
 SELECT p.politica_ref,p.version,p.criterio,p.tipo_lista,p.reposicion,p.provisional,p.rotulo,p.actor,p.vigente_desde,
        b.participacion_ref,b.orden_acta,e.orden_vigente,b.situacion,
        CASE WHEN NOT b.ocupa_turno AND b.situacion IN ('no_disponible','disponible_desde') THEN 'pausa'
             WHEN NOT b.ocupa_turno AND b.situacion='trabajando' THEN 'trabajando'
             WHEN NOT b.ocupa_turno THEN 'sin_turno'
             WHEN b.repuesta_en IS NOT NULL AND e.orden_vigente IS DISTINCT FROM b.orden_acta THEN 'reposicion_tras_contrato'
             WHEN e.orden_vigente IS DISTINCT FROM b.orden_acta THEN 'pausa'
             ELSE 'orden_acta' END
   FROM base b CROSS JOIN politica p LEFT JOIN elegibles e USING(participacion_ref)
  ORDER BY e.orden_vigente NULLS LAST,b.orden_acta,b.participacion_ref
 $f$;
DO $comprobacion$ BEGIN
 IF (SELECT md5(p.prosrc) FROM pg_catalog.pg_proc p WHERE p.oid = to_regprocedure('vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)')) <> '99bc67526e88fef0febade892b3846d2' THEN
  RAISE EXCEPTION 'no se restauro el orden vigente de 000018' USING ERRCODE='55000';
 END IF;
END $comprobacion$;
DROP TABLE vec_bolsa_llamamientos.reversion_sancion_participacion RESTRICT;
DROP TABLE vec_bolsa_llamamientos.penalizacion_orden_sancion RESTRICT;
COMMIT;
