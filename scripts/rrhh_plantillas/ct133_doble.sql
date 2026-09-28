\set ON_ERROR_STOP on
-- CT133 no existe en la base de este candidato. Esta firma es un doble de
-- preimagen exclusivamente para CT135; nunca acreditar lectura documental.
SET ROLE vec_contratacion_temporal_propietario;
CREATE FUNCTION vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS jsonb LANGUAGE sql SECURITY DEFINER AS $f$ SELECT '{}'::jsonb $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
RESET ROLE;
