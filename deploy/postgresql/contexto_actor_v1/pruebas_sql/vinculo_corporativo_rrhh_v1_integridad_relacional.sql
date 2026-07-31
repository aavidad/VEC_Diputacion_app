-- Integridad relacional focal C2.2-B1. Todo se revierte al finalizar.
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

INSERT INTO vec_contexto_actor_v1.procedencias VALUES
 ('prc_autoridad_corporativa_rrhh_0001',1,repeat('a',64),
  'autoridad_maestra_acreditada'),
 ('prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_corporativa_rrhh_000000000001',1,
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_corporativa_rrhh_000000000001',1,
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_corporativo_rrhh_000000000001',1,
  'per_corporativa_rrhh_000000000001',
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_corporativo_rrhh_000000000001',1,
  'cta_corporativa_rrhh_000000000001','prf_corporativo_rrhh_000000000001',
  'per_corporativa_rrhh_000000000001',
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES
 ('org_diputaciondemo0001',1,'prc_autoridad_corporativa_rrhh_0001',1,
  repeat('a',64),'autoridad_maestra_acreditada','activo',
  '2026-01-01','2027-01-01');

INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones VALUES
 ('vcr_corporativo_rrhh_000000000001',1,
  'cta_corporativa_rrhh_000000000001',1,
  'per_corporativa_rrhh_000000000001',1,
  'prf_corporativo_rrhh_000000000001',1,
  'vca_corporativo_rrhh_000000000001',1,
  'org_diputaciondemo0001',1,
  'prc_autoridad_corporativa_rrhh_0001',1,repeat('a',64),
  'autoridad_maestra_acreditada','interna_corporativa','consulta_rrhh',
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01'),
 ('vcr_corporativo_rrhh_000000000001',2,
  'cta_corporativa_rrhh_000000000001',1,
  'per_corporativa_rrhh_000000000001',1,
  'prf_corporativo_rrhh_000000000001',1,
  'vca_corporativo_rrhh_000000000001',1,
  'org_diputaciondemo0001',1,
  'prc_autoridad_corporativa_rrhh_0001',1,repeat('a',64),
  'autoridad_maestra_acreditada','interna_corporativa','consulta_rrhh',
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','revocado','2026-01-01','2027-01-01');

DO $integridad$
DECLARE
    sentencia text;
    rechazadas integer := 0;
BEGIN
  FOREACH sentencia IN ARRAY ARRAY[
    $$INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
      SELECT 'vcr_corporativo_rrhh_000000000002',1,'cta_inexistente_00000000000001',
      cuenta_version,persona_ref,persona_version,perfil_ref,perfil_version,
      vinculo_contexto_ref,vinculo_contexto_version,organizacion_ref,
      organizacion_version,organizacion_procedencia_ref,
      organizacion_procedencia_version,organizacion_procedencia_huella_sha256,
      organizacion_procedencia_autoridad,superficie,uso,procedencia_ref,
      procedencia_version,procedencia_huella_sha256,procedencia_autoridad,
      estado,vigente_desde,vigente_hasta
      FROM vec_contexto_actor_v1.vinculo_corporativo_versiones LIMIT 1$$,
    $$INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
      SELECT 'vcr_corporativo_rrhh_000000000003',1,cuenta_ref,cuenta_version,
      'per_inexistente_000000000000001',persona_version,perfil_ref,
      perfil_version,vinculo_contexto_ref,vinculo_contexto_version,
      organizacion_ref,organizacion_version,organizacion_procedencia_ref,
      organizacion_procedencia_version,organizacion_procedencia_huella_sha256,
      organizacion_procedencia_autoridad,superficie,uso,procedencia_ref,
      procedencia_version,procedencia_huella_sha256,procedencia_autoridad,
      estado,vigente_desde,vigente_hasta
      FROM vec_contexto_actor_v1.vinculo_corporativo_versiones LIMIT 1$$,
    $$INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
      SELECT 'vcr_corporativo_rrhh_000000000004',1,cuenta_ref,cuenta_version,
      persona_ref,persona_version,perfil_ref,perfil_version,
      vinculo_contexto_ref,vinculo_contexto_version,organizacion_ref,
      organizacion_version,organizacion_procedencia_ref,
      organizacion_procedencia_version,organizacion_procedencia_huella_sha256,
      organizacion_procedencia_autoridad,'externa','consulta_rrhh',procedencia_ref,
      procedencia_version,procedencia_huella_sha256,procedencia_autoridad,
      estado,vigente_desde,vigente_hasta
      FROM vec_contexto_actor_v1.vinculo_corporativo_versiones LIMIT 1$$,
    $$INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
      SELECT 'vcr_corporativo_rrhh_000000000005',1,cuenta_ref,cuenta_version,
      persona_ref,persona_version,perfil_ref,perfil_version,
      vinculo_contexto_ref,vinculo_contexto_version,organizacion_ref,
      organizacion_version,organizacion_procedencia_ref,
      organizacion_procedencia_version,organizacion_procedencia_huella_sha256,
      organizacion_procedencia_autoridad,superficie,'administrar',procedencia_ref,
      procedencia_version,procedencia_huella_sha256,procedencia_autoridad,
      estado,vigente_desde,vigente_hasta
      FROM vec_contexto_actor_v1.vinculo_corporativo_versiones LIMIT 1$$,
    $$UPDATE vec_contexto_actor_v1.vinculo_corporativo_versiones SET estado='activo'$$,
    $$DELETE FROM vec_contexto_actor_v1.vinculo_corporativo_versiones$$,
    $$TRUNCATE vec_contexto_actor_v1.vinculo_corporativo_versiones$$,
    $$TRUNCATE vec_contexto_actor_v1.vinculo_corporativo_actual$$
  ] LOOP
    BEGIN
      EXECUTE sentencia;
    EXCEPTION WHEN OTHERS THEN
      rechazadas := rechazadas + 1;
    END;
  END LOOP;
  IF rechazadas <> 8
     OR (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_corporativo_versiones) <> 2 THEN
    RAISE EXCEPTION 'integridad adversarial incompleta: % rechazos', rechazadas;
  END IF;
END
$integridad$;

INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES
 ('cta_corporativa_rrhh_000000000001','interna_corporativa','consulta_rrhh',
  'vcr_corporativo_rrhh_000000000001',2);
DO $puntero$
BEGIN
  IF (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_corporativo_actual) <> 1 THEN
    RAISE EXCEPTION 'puntero corporativo no persistio dentro de la prueba';
  END IF;
END
$puntero$;
ROLLBACK;
