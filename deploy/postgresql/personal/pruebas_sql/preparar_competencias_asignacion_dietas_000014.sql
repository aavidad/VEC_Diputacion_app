\set ON_ERROR_STOP on
-- TEST-ONLY: sustituye a la fachada real AD3-61 solo en este PG18 efímero sin
-- núcleo AD3, para probar la lógica de Personal con consumos simulados. La
-- instalación de 000014 sobre la fachada real se ensaya en
-- autorizacion_atestada_v3/pruebas_sql/dietas_d7bc_ad3_000061_pg18.sh.
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
