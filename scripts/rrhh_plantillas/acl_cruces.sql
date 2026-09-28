\set ON_ERROR_STOP on
-- Firmas sintéticas de CT108 y CT132 para probar el preflight real.
-- Sus cuerpos no modelan auditoría ni se invocan; solo aíslan las ACL.
SET ROLE vec_contratacion_temporal_propietario;
CREATE FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)
RETURNS void LANGUAGE plpgsql AS $f$ BEGIN RETURN; END $f$;
CREATE FUNCTION vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(
 text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS void LANGUAGE plpgsql AS $f$ BEGIN RETURN; END $f$;
REVOKE ALL ON FUNCTION
 vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text),
 vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC;
RESET ROLE;
