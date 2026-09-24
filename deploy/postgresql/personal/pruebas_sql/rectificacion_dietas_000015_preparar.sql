\set ON_ERROR_STOP on
-- TEST-ONLY. El stub AD3 no acredita autorización real.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_rectificacion_dietas_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT 'dec_prueba'::text,'efecto_prueba'::text,repeat('a',64),
  encode(sha256(convert_to(clock_timestamp()::text||random()::text,'UTF8')),'hex'),
  'aud_prueba'::text,clock_timestamp(),true
$f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_rectificacion_dietas_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_personal_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_rectificacion_dietas_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_rectificacion_dietas_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_competencias_asignacion_dietas_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT 'dec_comp_prueba'::text,'efecto_comp_prueba'::text,repeat('a',64),
  encode(sha256(convert_to(clock_timestamp()::text||random()::text,'UTF8')),'hex'),
  'aud_comp_prueba'::text,clock_timestamp(),true
$f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_competencias_asignacion_dietas_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_personal_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_competencias_asignacion_dietas_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_competencias_asignacion_dietas_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
