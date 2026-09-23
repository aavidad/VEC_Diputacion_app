\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000003:tarifas-consulta:v1',0));
DO $lectura$ DECLARE nombre text;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['version_tarifa_provisional','importe_dieta_provisional','importe_km_provisional'] LOOP
  IF NOT EXISTS (SELECT 1 FROM pg_policy p WHERE p.polrelid=format('vec_dietas.%I',nombre)::regclass AND p.polname='lectura_ejecutor_tarifas' AND p.polcmd='r')
  THEN RAISE EXCEPTION 'Dietas 000003: postimagen incompatible' USING ERRCODE='55000'; END IF;
  EXECUTE format('REVOKE SELECT ON vec_dietas.%I FROM vec_dietas_ejecutor',nombre);
  EXECUTE format('DROP POLICY lectura_ejecutor_tarifas ON vec_dietas.%I',nombre);
 END LOOP;
END $lectura$;
COMMIT;
