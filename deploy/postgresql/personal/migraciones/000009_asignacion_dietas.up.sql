\set ON_ERROR_STOP on
-- D7. Personal es la única autoridad de la asignación. Esta estructura no
-- concede a Dietas lectura de tablas ni permite corregir filas históricas.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000009:asignacion-dietas:v1',0));
DO $pre$
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR to_regclass('vec_personal.relacion_empleado_dietas') IS NULL
    OR to_regclass('vec_personal.asignacion_dietas') IS NOT NULL
    OR to_regprocedure('vec_personal.rechazar_mutacion_relacion_dietas_v1()') IS NULL
    OR EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='vec_personal.relacion_empleado_dietas'::regclass AND conname='relacion_dietas_identidad_unidad') THEN
   RAISE EXCEPTION 'Personal 000009: estructura previa incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;

ALTER TABLE vec_personal.relacion_empleado_dietas
 ADD CONSTRAINT relacion_dietas_identidad_unidad UNIQUE(relacion_ref,persona_ref,unidad_ref);
CREATE TABLE vec_personal.asignacion_dietas (
 asignacion_ref text PRIMARY KEY CHECK(asignacion_ref~'^ads_[A-Za-z0-9_-]{22,128}$'),
 relacion_ref text NOT NULL,
 persona_ref text NOT NULL CHECK(persona_ref~'^per_[A-Za-z0-9_-]{22,128}$'),
 unidad_ref text NOT NULL CHECK(length(unidad_ref) BETWEEN 1 AND 256),
 version bigint NOT NULL CHECK(version>0),
 centro_ref text NOT NULL CHECK(length(centro_ref) BETWEEN 1 AND 160 AND centro_ref !~ '[[:cntrl:]]'),
 administrativo_persona_ref text NOT NULL CHECK(administrativo_persona_ref~'^per_[A-Za-z0-9_-]{22,128}$'),
 responsable_persona_ref text NOT NULL CHECK(responsable_persona_ref~'^per_[A-Za-z0-9_-]{22,128}$'),
 grupo_dieta smallint NOT NULL CHECK(grupo_dieta BETWEEN 1 AND 3),
 vigente_desde date NOT NULL,
 motivo_revision text NOT NULL CHECK(length(motivo_revision) BETWEEN 3 AND 500 AND motivo_revision !~ '[[:cntrl:]]'),
 procedencia_acto_ref text NOT NULL CHECK(length(procedencia_acto_ref) BETWEEN 1 AND 256),
 registrada_por_ref text NOT NULL CHECK(length(registrada_por_ref) BETWEEN 1 AND 160),
 registrada_en timestamptz(6) NOT NULL,
 UNIQUE(relacion_ref,version),
 FOREIGN KEY(relacion_ref,persona_ref,unidad_ref)
  REFERENCES vec_personal.relacion_empleado_dietas(relacion_ref,persona_ref,unidad_ref),
 CHECK(administrativo_persona_ref<>responsable_persona_ref
   AND administrativo_persona_ref<>persona_ref AND responsable_persona_ref<>persona_ref)
);
ALTER TABLE vec_personal.asignacion_dietas ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.asignacion_dietas FORCE ROW LEVEL SECURITY;
CREATE POLICY persona_contextual ON vec_personal.asignacion_dietas FOR ALL TO vec_personal_propietario
 USING (persona_ref=current_setting('vec.dietas.persona_ref',true)
  AND current_setting('vec.dietas.persona_ref',true) IS NOT NULL)
 WITH CHECK (persona_ref=current_setting('vec.dietas.persona_ref',true)
  AND current_setting('vec.dietas.persona_ref',true) IS NOT NULL);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.asignacion_dietas
 FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.asignacion_dietas
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
REVOKE ALL ON vec_personal.asignacion_dietas FROM PUBLIC,vec_personal_ejecutor,vec_dietas_ejecutor,vec_dietas_propietario;
COMMENT ON TABLE vec_personal.asignacion_dietas IS 'D7: versiones inmutables de la asignación Personal para Dietas; la última versión se resuelve en una fachada nominal, no por SELECT de Dietas.';
COMMIT;
