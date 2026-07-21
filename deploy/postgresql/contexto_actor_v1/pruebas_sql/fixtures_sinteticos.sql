-- FIXTURES EXCLUSIVAMENTE SINTETICOS. No representan ni importan una fuente
-- corporativa, DNI, certificado, rol, correo ni dato declarado por usuarios.
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

INSERT INTO vec_contexto_actor_v1.procedencias VALUES
 ('prc_fixture_sintetico_no_corporativo_01',1,repeat('1',64),'no_autoritativa');

INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',1,
  'prc_fixture_sintetico_no_corporativo_01',1,
  repeat('1',64),'no_autoritativa','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES
 ('cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',1);

INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',1,
  'prc_fixture_sintetico_no_corporativo_01',1,repeat('1',64),'no_autoritativa',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.persona_actual VALUES
 ('per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',1);

INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_sintetico_cccccccccccccccccccccccc',1,'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
  'prc_fixture_sintetico_no_corporativo_01',1,repeat('1',64),'no_autoritativa',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES
 ('prf_sintetico_cccccccccccccccccccccccc',1);

INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_sintetico_dddddddddddddddddddddddd',1,
  'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa','prf_sintetico_cccccccccccccccccccccccc',
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
  'prc_fixture_sintetico_no_corporativo_01',1,repeat('1',64),'no_autoritativa',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES
 ('vca_sintetico_dddddddddddddddddddddddd',1);

INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones VALUES
 ('vin_sintetico_eeeeeeeeeeeeeeeeeeeeeeee',1,
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb','candidato',
  'can_sintetico_ffffffffffffffffffffffff',
  'prc_fixture_sintetico_no_corporativo_01',1,repeat('1',64),'no_autoritativa',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour'),
 ('vin_sintetico_gggggggggggggggggggggggg',1,
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb','empleado',
  'emp_sintetico_hhhhhhhhhhhhhhhhhhhhhhhh',
  'prc_fixture_sintetico_no_corporativo_01',1,repeat('1',64),'no_autoritativa',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_actual VALUES
 ('vin_sintetico_eeeeeeeeeeeeeeeeeeeeeeee',1),
 ('vin_sintetico_gggggggggggggggggggggggg',1);
COMMIT;
