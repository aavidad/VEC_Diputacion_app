\set ON_ERROR_STOP on
-- D3/D4: lectura de importes propios y versionados para un cálculo sintético.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000003:tarifas-consulta:v1',0));
DO $pre$ DECLARE nombre text;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR to_regclass('vec_dietas.borrador_comision') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_ejecutor' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'Dietas 000003: preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY['version_tarifa_provisional','importe_dieta_provisional','importe_km_provisional'] LOOP
  IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_dietas' AND c.relname=nombre AND c.relkind='r' AND c.relowner=current_user::regrole AND c.relrowsecurity AND c.relforcerowsecurity)
     OR EXISTS (SELECT 1 FROM pg_policy p WHERE p.polrelid=format('vec_dietas.%I',nombre)::regclass AND p.polname='lectura_ejecutor_tarifas')
  THEN RAISE EXCEPTION 'Dietas 000003: catálogo incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;
DO $lectura$ DECLARE nombre text;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['version_tarifa_provisional','importe_dieta_provisional','importe_km_provisional'] LOOP
  EXECUTE format('CREATE POLICY lectura_ejecutor_tarifas ON vec_dietas.%I FOR SELECT TO vec_dietas_ejecutor USING (true)',nombre);
  EXECUTE format('GRANT SELECT ON vec_dietas.%I TO vec_dietas_ejecutor',nombre);
 END LOOP;
END $lectura$;
COMMIT;
