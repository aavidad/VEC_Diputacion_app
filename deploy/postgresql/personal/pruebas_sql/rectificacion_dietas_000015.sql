\set ON_ERROR_STOP on
-- Ensayo estructural PG18. El consumidor AD3 de esta base es un stub TEST-ONLY.
DO $acl$
DECLARE solicitud regprocedure:='vec_personal.solicitar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 consulta regprocedure:='vec_personal.consultar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 resuelve regprocedure:='vec_personal.resolver_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 confirma regprocedure:='vec_personal.confirmar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 competente regprocedure:='vec_personal.consultar_rectificaciones_competentes_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 frontera regprocedure:='vec_personal.registrar_auditoria_frontera_rectificacion_dietas_v1(text,text,text,text,text,text,text,integer)'::regprocedure;
 interno regprocedure:='vec_personal.ejecutar_rectificacion_dietas_interna_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF has_table_privilege('vec_personal_ejecutor','vec_personal.solicitud_rectificacion_dietas','SELECT')
    OR has_table_privilege('vec_personal_registrador_frontera','vec_personal.auditoria_frontera_rectificacion_dietas','INSERT')
    OR has_table_privilege('vec_personal_d7_ejecutor','vec_personal.recibo_consulta_competentes_rectificacion_dietas','SELECT')
    OR has_function_privilege('vec_dietas_ejecutor',solicitud,'EXECUTE')
    OR has_function_privilege('vec_dietas_propietario',solicitud,'EXECUTE')
    OR has_function_privilege('vec_personal_ejecutor',solicitud,'EXECUTE')
    OR has_function_privilege('vec_personal_d7_ejecutor',interno,'EXECUTE')
    OR has_function_privilege('vec_personal_ejecutor',frontera,'EXECUTE')
    OR NOT has_function_privilege('vec_personal_d7_ejecutor',solicitud,'EXECUTE')
    OR NOT has_function_privilege('vec_personal_d7_ejecutor',consulta,'EXECUTE')
    OR NOT has_function_privilege('vec_personal_d7_ejecutor',resuelve,'EXECUTE')
    OR NOT has_function_privilege('vec_personal_d7_ejecutor',confirma,'EXECUTE')
    OR NOT has_function_privilege('vec_personal_d7_ejecutor',competente,'EXECUTE')
    OR NOT has_function_privilege('vec_personal_registrador_frontera',frontera,'EXECUTE')
    OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
      WHERE p.oid IN (solicitud,consulta,resuelve,confirma,competente,frontera,interno) AND x.grantee=0)
 THEN RAISE EXCEPTION 'ACL rectificación 000015 abierta' USING ERRCODE='55000'; END IF;
END $acl$;

CREATE ROLE vec_prueba_d7c_frontera NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_personal_registrador_frontera TO vec_prueba_d7c_frontera WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SET SESSION AUTHORIZATION vec_prueba_d7c_frontera;
DO $frontera$
BEGIN
 IF NOT vec_personal.registrar_auditoria_frontera_rectificacion_dietas_v1(
  'corr_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','dependencia_no_disponible',
  'api.personal.solicitudes_rectificacion_dietas',
  '/api/vec/personal/solicitudes-rectificacion-dietas',
  'solicitar','actor_sintetico',NULL,503)
 THEN RAISE EXCEPTION 'auditoría 503 sin recurso'; END IF;
 BEGIN
  PERFORM vec_personal.registrar_auditoria_frontera_rectificacion_dietas_v1(
   'corr_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','acceso_denegado',
   'api.personal.solicitudes_rectificacion_dietas',
   '/api/vec/personal/solicitudes-rectificacion-dietas',
   'solicitar','actor_sintetico',NULL,503);
  RAISE EXCEPTION 'auditoría estado/motivo discordantes';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
END $frontera$;
RESET SESSION AUTHORIZATION;
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
DO $auditoria$
BEGIN
 IF (SELECT count(*) FROM vec_personal.auditoria_frontera_rectificacion_dietas
     WHERE actor_ref='actor_sintetico' AND recurso_ref IS NULL AND estado_http=503)<>1
 THEN RAISE EXCEPTION 'auditoría de rechazo identificada ausente'; END IF;
END $auditoria$;
COMMIT;

-- Confirmación requiere corrección D7 y cambio de estado en una única tx.
CREATE ROLE vec_prueba_d7c_personal NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_personal_d7_ejecutor TO vec_prueba_d7c_personal WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SET SESSION AUTHORIZATION vec_prueba_d7c_personal;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $confirmar$
BEGIN
 BEGIN
  PERFORM vec_personal.confirmar_rectificacion_dietas_v1('{}','p','d','m','c',1,1,'p','s','e','r',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'confirmación sintética aceptada';
 EXCEPTION WHEN SQLSTATE '55000' THEN
  IF SQLERRM<>'confirmación requiere corrección D7 completa'
  THEN RAISE; END IF;
 END;
END $confirmar$;
COMMIT;
RESET SESSION AUTHORIZATION;

-- Una fila sembrada solo por la prueba acredita restricciones e inmutabilidad,
-- nunca el recorrido de solicitud autorizada.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SELECT set_config('vec.dietas.persona_ref','per_abcdefghijklmnopqrstuv',true);
INSERT INTO vec_personal.asignacion_dietas VALUES(
 'ads_abcdefghijklmnopqrstuv','rel_abcdefghijklmnopqrstuv',
 'per_abcdefghijklmnopqrstuv','unidad-sintetica',1,'centro-sintetico',
 'per_bbbbbbbbbbbbbbbbbbbbbb','per_cccccccccccccccccccccc',1,
 DATE '2026-01-01','alta sintética','acto-sintetico','actor-sintetico',clock_timestamp());
INSERT INTO vec_personal.solicitud_rectificacion_dietas VALUES(
 'srd_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','rel_abcdefghijklmnopqrstuv',
 'per_abcdefghijklmnopqrstuv','unidad-sintetica','ads_abcdefghijklmnopqrstuv',1,DATE '2026-02-01',
 '["centro_ref"]'::jsonb,'Revisar centro','Centro incorrecto',
 '018f1d1e-1234-4abc-8def-0123456789ab',repeat('a',64),'actor-sintetico',clock_timestamp());
DO $historia$
BEGIN
 BEGIN
  UPDATE vec_personal.solicitud_rectificacion_dietas SET detalle_solicitado='alterado';
  RAISE EXCEPTION 'historia modificable';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 IF (SELECT count(*) FROM vec_personal.solicitud_rectificacion_dietas)<>1
 THEN RAISE EXCEPTION 'historia perdida'; END IF;
END $historia$;
ROLLBACK;
