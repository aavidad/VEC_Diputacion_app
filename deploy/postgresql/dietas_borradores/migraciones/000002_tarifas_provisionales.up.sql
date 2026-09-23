\set ON_ERROR_STOP on
-- D3/D4. Referencias de cuantías solo para ensayo sintético; RRHH debe
-- confirmar grupos, vigencia y reglas de devengo antes de su uso real.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000002:tarifas:v1',0));
DO $pre$
BEGIN
 IF current_user<>'vec_dietas_propietario'
    OR to_regclass('vec_dietas.borrador_comision') IS NULL
    OR to_regclass('vec_dietas.version_tarifa_provisional') IS NOT NULL
    OR to_regclass('vec_dietas.importe_dieta_provisional') IS NOT NULL
    OR to_regclass('vec_dietas.importe_km_provisional') IS NOT NULL
    OR to_regprocedure('vec_dietas.rechazar_mutacion_borrador_v1()') IS NULL THEN
   RAISE EXCEPTION 'Dietas 000002: estructura previa incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE TABLE vec_dietas.version_tarifa_provisional (
 version_ref text PRIMARY KEY CHECK(version_ref~'^provisional:[a-z0-9:-]{8,120}$'),
 rotulo text NOT NULL CHECK(rotulo='PROVISIONAL · pendiente de confirmación por RRHH'),
 referencia_dietas text NOT NULL CHECK(referencia_dietas='BOE-A-2005-19988 / RD 462/2002'),
 referencia_km text NOT NULL CHECK(referencia_km='BOE-A-2023-16462 / RD 462/2002'),
 vigente_desde date NOT NULL, vigente_hasta date,
 publicada_en timestamptz(6) NOT NULL,
 CHECK(vigente_hasta IS NULL OR vigente_desde<vigente_hasta)
);
CREATE TABLE vec_dietas.importe_dieta_provisional (
 version_ref text NOT NULL REFERENCES vec_dietas.version_tarifa_provisional(version_ref),
 pais_iso2 text NOT NULL CHECK(pais_iso2~'^[A-Z]{2}$'),
 grupo smallint NOT NULL CHECK(grupo BETWEEN 1 AND 3),
 alojamiento_eur numeric(9,2) NOT NULL CHECK(alojamiento_eur>0),
 manutencion_eur numeric(9,2) NOT NULL CHECK(manutencion_eur>0),
 PRIMARY KEY(version_ref,pais_iso2,grupo)
);
CREATE TABLE vec_dietas.importe_km_provisional (
 version_ref text NOT NULL REFERENCES vec_dietas.version_tarifa_provisional(version_ref),
 vehiculo text NOT NULL CHECK(vehiculo IN ('automovil','motocicleta')),
 eur_por_km numeric(8,4) NOT NULL CHECK(eur_por_km>0),
 PRIMARY KEY(version_ref,vehiculo)
);

INSERT INTO vec_dietas.version_tarifa_provisional VALUES
 ('provisional:rd462:20260923','PROVISIONAL · pendiente de confirmación por RRHH',
  'BOE-A-2005-19988 / RD 462/2002','BOE-A-2023-16462 / RD 462/2002',
  DATE '2026-09-23',NULL,clock_timestamp());
INSERT INTO vec_dietas.importe_dieta_provisional VALUES
 ('provisional:rd462:20260923','ES',1,102.56,53.34),
 ('provisional:rd462:20260923','ES',2,65.97,37.40),
 ('provisional:rd462:20260923','ES',3,48.92,28.21);
INSERT INTO vec_dietas.importe_km_provisional VALUES
 ('provisional:rd462:20260923','automovil',0.2600),
 ('provisional:rd462:20260923','motocicleta',0.1060);

DO $politicas$ DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['version_tarifa_provisional','importe_dieta_provisional','importe_km_provisional'] LOOP
  EXECUTE format('ALTER TABLE vec_dietas.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_dietas.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE POLICY propietario_catalogo ON vec_dietas.%I FOR ALL TO vec_dietas_propietario USING (true) WITH CHECK (true)',tabla);
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_dietas.%I FOR EACH ROW EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1()',tabla);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_dietas.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1()',tabla);
  EXECUTE format('REVOKE ALL ON vec_dietas.%I FROM PUBLIC,vec_dietas_ejecutor',tabla);
 END LOOP;
END $politicas$;
COMMENT ON TABLE vec_dietas.version_tarifa_provisional IS 'D3/D4: versiones sintéticas provisionales; no determinan derecho ni liquidación. Pendientes de RRHH.';
COMMENT ON TABLE vec_dietas.importe_dieta_provisional IS 'D3: importes de referencia nacionales por grupo, sin regla de tramo aprobada.';
COMMENT ON TABLE vec_dietas.importe_km_provisional IS 'D4: importe por km de referencia, sin ruta acreditada ni aprobación de RRHH.';
COMMIT;
