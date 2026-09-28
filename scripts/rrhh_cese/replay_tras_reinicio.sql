\set ON_ERROR_STOP on
-- Misma fila y triple CT; no se emite una segunda restricción ni otro recibo.
SELECT r.origen_ref AS origen,r.origen_huella_sha256 AS huella,
       r.origen_posicion AS posicion,r.recibo_ref AS recibo,
       r.disponible_desde::text AS disponible,r.politica_version AS politica
  FROM vec_bolsa_llamamientos.restriccion_cese_bolsa r
 ORDER BY r.recibida_en DESC LIMIT 1 \gset
SET SESSION AUTHORIZATION vec_rrhh_cese_relevo_pg18;
SELECT reutilizada AS replay,recibo_ref=:'recibo' AS mismo_recibo,
       disponible_desde::text=:'disponible' AS misma_fecha,
       politica_version::text=:'politica' AS misma_politica
  FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1(
    :'origen',:'huella',:'posicion'::bigint) \gset
RESET SESSION AUTHORIZATION;
SELECT 1/CASE WHEN :'replay'::boolean AND :'mismo_recibo'::boolean
    AND :'misma_fecha'::boolean AND :'misma_politica'::boolean THEN 1 ELSE 0 END;
