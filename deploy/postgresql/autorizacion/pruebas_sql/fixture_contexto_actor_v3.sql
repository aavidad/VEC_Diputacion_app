-- Datos exclusivamente sinteticos para el contrato SQL V3.
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;

INSERT INTO vec_contexto_actor_v1.procedencias VALUES
 ('prc_maestra_sintetica_v3_00000001', 1, repeat('a',64),
  'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa', 2,
  'prc_maestra_sintetica_v3_00000001', 1, repeat('a',64),
  'autoridad_maestra_acreditada', 'activo',
  clock_timestamp()-interval '1 hour', clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.proyeccion_cuenta_actual SET version=2
 WHERE cuenta_ref='cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa';
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb', 2,
  'prc_maestra_sintetica_v3_00000001', 1, repeat('a',64),
  'autoridad_maestra_acreditada', 'activo',
  clock_timestamp()-interval '1 hour', clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.persona_actual SET version=2
 WHERE persona_ref='per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb';
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_sintetico_cccccccccccccccccccccccc', 2,
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
  'prc_maestra_sintetica_v3_00000001', 1, repeat('a',64),
  'autoridad_maestra_acreditada', 'activo',
  clock_timestamp()-interval '1 hour', clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.perfil_actual SET version=2
 WHERE perfil_ref='prf_sintetico_cccccccccccccccccccccccc';
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_sintetico_dddddddddddddddddddddddd', 2,
  'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
  'prf_sintetico_cccccccccccccccccccccccc',
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
  'prc_maestra_sintetica_v3_00000001', 1, repeat('a',64),
  'autoridad_maestra_acreditada', 'activo',
  clock_timestamp()-interval '1 hour', clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.vinculo_contexto_actual SET version=2
 WHERE vinculo_ref='vca_sintetico_dddddddddddddddddddddddd';
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones VALUES
 ('vin_sintetico_eeeeeeeeeeeeeeeeeeeeeeee', 2,
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb', 'candidato',
  'can_sintetico_ffffffffffffffffffffffff',
  'prc_maestra_sintetica_v3_00000001', 1, repeat('a',64),
  'autoridad_maestra_acreditada', 'activo',
  clock_timestamp()-interval '1 hour', clock_timestamp()+interval '1 hour'),
 ('vin_sintetico_gggggggggggggggggggggggg', 2,
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb', 'empleado',
  'emp_sintetico_hhhhhhhhhhhhhhhhhhhhhhhh',
  'prc_maestra_sintetica_v3_00000001', 1, repeat('a',64),
  'autoridad_maestra_acreditada', 'activo',
  clock_timestamp()-interval '1 hour', clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.vinculo_referencia_actual SET version=2
 WHERE vinculo_ref IN (
   'vin_sintetico_eeeeeeeeeeeeeeeeeeeeeeee',
   'vin_sintetico_gggggggggggggggggggggggg'
 );
COMMIT;

-- El runner crea el rca_ mediante un LOGIN runtime acreditado. El propietario
-- no puede suplantar esa identidad y este fixture no debilita dicho cierre.
