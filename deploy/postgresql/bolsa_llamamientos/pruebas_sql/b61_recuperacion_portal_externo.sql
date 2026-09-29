\set ON_ERROR_STOP on
-- Ejecutar tras reiniciar PostgreSQL del clon, sin repetir B60 ni B61.
SET search_path=pg_catalog;
DO $recuperacion$ BEGIN
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.denegacion_frontera_portal_externo)<>6
    OR (SELECT count(DISTINCT ruta) FROM vec_bolsa_llamamientos.denegacion_frontera_portal_externo)<>6
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.denegacion_frontera_portal_externo
      WHERE registrada_en IS NULL OR superficie<>'api.bolsa.mi_bolsa.ruta_exacta')
 THEN RAISE EXCEPTION 'B61: historia alterada tras reinicio'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid='vec_bolsa_llamamientos.denegacion_frontera_portal_externo'::regclass
   AND c.relrowsecurity AND c.relforcerowsecurity)
 THEN RAISE EXCEPTION 'B61: RLS no conservada'; END IF;
END $recuperacion$;
