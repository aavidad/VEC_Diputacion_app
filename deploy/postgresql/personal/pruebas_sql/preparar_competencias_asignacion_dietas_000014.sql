\set ON_ERROR_STOP on
-- TEST-ONLY: AD3 D7b todavía no tiene número ni instalación. Este stub
-- permite probar únicamente el contrato SQL de Personal en PG18 efímero.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_competencias_asignacion_dietas_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $$
 SELECT * FROM vec_autorizacion_atestada_v3.d7_consumir_prueba('competencias',$1,$2)
$$;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_competencias_asignacion_dietas_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 OWNER TO vec_personal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_competencias_asignacion_dietas_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_personal_propietario;
