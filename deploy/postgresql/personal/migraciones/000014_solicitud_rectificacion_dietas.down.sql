\set ON_ERROR_STOP on
-- Sólo para ensayo vacío aislado. No revertir solicitudes, recibos ni auditoría.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL row_security=off;
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000014:rectificacion-dietas:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
 THEN RAISE EXCEPTION 'Personal 000014 DOWN requiere DBA' USING ERRCODE='42501'; END IF;
END $pre$;
LOCK TABLE vec_personal.solicitud_rectificacion_dietas,
 vec_personal.evento_rectificacion_dietas,
 vec_personal.auditoria_frontera_rectificacion_dietas,
 vec_personal.recibo_consulta_competentes_rectificacion_dietas IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_personal.solicitud_rectificacion_dietas)
    OR EXISTS(SELECT 1 FROM vec_personal.evento_rectificacion_dietas)
    OR EXISTS(SELECT 1 FROM vec_personal.auditoria_frontera_rectificacion_dietas)
    OR EXISTS(SELECT 1 FROM vec_personal.recibo_consulta_competentes_rectificacion_dietas)
 THEN RAISE EXCEPTION 'Personal 000014 DOWN protege historia' USING ERRCODE='55000'; END IF;
END $historia$;
SET LOCAL ROLE vec_personal_propietario;
DROP FUNCTION vec_personal.solicitar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_personal.consultar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_personal.resolver_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_personal.confirmar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_personal.consultar_rectificaciones_competentes_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_personal.ejecutar_rectificacion_dietas_interna_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_personal.registrar_auditoria_frontera_rectificacion_dietas_v1(text,text,text,text,text,text,text,integer) RESTRICT;
DROP TABLE vec_personal.evento_rectificacion_dietas RESTRICT;
DROP TABLE vec_personal.solicitud_rectificacion_dietas RESTRICT;
DROP TABLE vec_personal.auditoria_frontera_rectificacion_dietas RESTRICT;
DROP TABLE vec_personal.recibo_consulta_competentes_rectificacion_dietas RESTRICT;
COMMIT;
