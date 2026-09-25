-- Dobles de ensayo de los consumidores AD3 v3 para probar Bolsa 000031:
-- solo firmas y retorno; la autorización real se prueba en su esquema.
DO $r$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_autorizacion_atestada_v3_propietario') THEN CREATE ROLE vec_autorizacion_atestada_v3_propietario NOLOGIN; END IF;
END $r$;
CREATE SCHEMA IF NOT EXISTS vec_autorizacion_atestada_v3 AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
DO $f$ DECLARE n text; BEGIN
 FOREACH n IN ARRAY ARRAY['registrar_y_consumir_bolsa_llamamiento_v3_atestada','registrar_y_consumir_mi_bolsa_v3_atestada','registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada','registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada','registrar_y_consumir_situacion_participacion_v3_atestada','registrar_y_consumir_consulta_contacto_v3_atestada','registrar_y_consumir_contacto_participacion_v3_atestada','registrar_y_consumir_datos_contacto_participacion_v3_atestada','registrar_y_consumir_emision_llamamiento_v3_atestada','registrar_y_consumir_despacho_correo_llamamiento_ct_v3_atestada','registrar_y_consumir_resultado_correo_llamamiento_ct_v3_atestada'] LOOP
  EXECUTE format($s$CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.%I(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
   RETURNS TABLE(efecto_ref text,consumo_nuevo boolean) LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS 'SELECT convert_from(p_decision,''UTF8'')::jsonb->>''recurso_ref'', convert_from(p_capacidad,''UTF8'')<>''repetida'''$s$, n);
  EXECUTE format('ALTER FUNCTION vec_autorizacion_atestada_v3.%I(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario', n);
 END LOOP;
END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(p_perfil_mutacion text,p_capacidad_canonica bytea,p_decision_canonica bytea,p_motivo_canonico bytea,p_contexto_actor_canonico bytea,p_persona_version numeric,p_perfil_version numeric,p_payload_vec_ad_3 bytea,p_sobre_cose_sign1 bytea,p_evidencia_verificacion bytea,p_raiz_publica_spki bytea)
 RETURNS TABLE(efecto_ref text,consumo_nuevo boolean) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
BEGIN
 -- bolsa.llamamiento.aceptacion_rrhh.registrar bolsa.llamamiento.renuncia_rrhh.registrar bolsa.llamamiento.siguiente.abrir
 RETURN QUERY SELECT convert_from(p_decision_canonica,'UTF8')::jsonb->>'recurso_ref', true;
END $f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario;
