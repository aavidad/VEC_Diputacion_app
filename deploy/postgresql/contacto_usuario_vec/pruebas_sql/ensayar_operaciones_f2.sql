\set ON_ERROR_STOP on
-- Prepara exclusivamente la persona sintética de ContextoActor que consume
-- el ensayo Go de contacto en un contenedor PG18 sin red. La primera versión
-- no autoritativa procede del fixture oficial de ContextoActor; esta revisión
-- no representa una fuente corporativa ni contiene datos reales.
-- pg_dump --schema-only omite los dos singleton iniciales de migración. Se
-- restauran únicamente en esta base desechable y se comprueba que los
-- disparadores originales quedan activos antes de emitir material V3.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
ALTER TABLE vec_autorizacion.motivo_v2_checkpoint_origen DISABLE TRIGGER bloquear_insercion_checkpoint;
INSERT INTO vec_autorizacion.motivo_v2_checkpoint_origen(control_id,ultima_secuencia,actualizado_en)
VALUES(true,0,clock_timestamp());
ALTER TABLE vec_autorizacion.motivo_v2_checkpoint_origen ENABLE TRIGGER bloquear_insercion_checkpoint;
COMMIT;
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.checkpoint_gobierno VALUES(true,0,0,0,clock_timestamp());
COMMIT;
DO $control$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid='vec_autorizacion.motivo_v2_checkpoint_origen'::regclass
   AND tgname='bloquear_insercion_checkpoint' AND tgenabled='O')
   OR (SELECT count(*) FROM vec_autorizacion.motivo_v2_checkpoint_origen)<>1
   OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.checkpoint_gobierno)<>1 THEN
   RAISE EXCEPTION 'F2: singletons de gobierno no restaurados';
 END IF;
END $control$;
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path=pg_catalog;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES
 ('prc_maestra_sintetica_contacto_f2_01',1,repeat('a',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',2,
  'prc_maestra_sintetica_contacto_f2_01',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.proyeccion_cuenta_actual SET version=2
 WHERE cuenta_ref='cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa';
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',2,
  'prc_maestra_sintetica_contacto_f2_01',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.persona_actual SET version=2
 WHERE persona_ref='per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb';
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_sintetico_cccccccccccccccccccccccc',2,'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
  'prc_maestra_sintetica_contacto_f2_01',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.perfil_actual SET version=2
 WHERE perfil_ref='prf_sintetico_cccccccccccccccccccccccc';
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_sintetico_dddddddddddddddddddddddd',2,
  'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa','prf_sintetico_cccccccccccccccccccccccc',
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
  'prc_maestra_sintetica_contacto_f2_01',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.vinculo_contexto_actual SET version=2
 WHERE vinculo_ref='vca_sintetico_dddddddddddddddddddddddd';
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones VALUES
 ('vin_sintetico_eeeeeeeeeeeeeeeeeeeeeeee',2,
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb','candidato','can_sintetico_ffffffffffffffffffffffff',
  'prc_maestra_sintetica_contacto_f2_01',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour'),
 ('vin_sintetico_gggggggggggggggggggggggg',2,
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb','empleado','emp_sintetico_hhhhhhhhhhhhhhhhhhhhhhhh',
  'prc_maestra_sintetica_contacto_f2_01',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.vinculo_referencia_actual SET version=2
 WHERE vinculo_ref IN ('vin_sintetico_eeeeeeeeeeeeeeeeeeeeeeee','vin_sintetico_gggggggggggggggggggggggg');
COMMIT;
