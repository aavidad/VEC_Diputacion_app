\set ON_ERROR_STOP on
-- Dobles AD3 para la prueba desechable de Bolsa 000039. No acreditan
-- criptografía ni la cadena AD3 real: el consumidor nominal devuelve el
-- efecto y la huella de la capacidad presentada y un consumo nuevo, y rechaza
-- la capacidad marcada para probar el fallo cerrado. El núcleo interno solo
-- declara (en su cuerpo) las acciones que exigen 000004-000006.
DO $$ BEGIN
    IF pg_catalog.current_setting('server_version_num')::integer / 100 <> 1800
       OR pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3') IS NOT NULL THEN
        RAISE EXCEPTION 'Bolsa 000039 requiere PG18 desechable sin AD3';
    END IF;
END $$;
CREATE ROLE vec_autorizacion_atestada_v3_propietario NOLOGIN NOINHERIT;
CREATE SCHEMA vec_autorizacion_atestada_v3 AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_bolsa_llamamiento_v3_atestada(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE (
    decision_ref text,efecto_ref text,huella_efecto_sha256 text,
    consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean
) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog
AS $f$
DECLARE c jsonb := convert_from(p_capacidad,'UTF8')::jsonb;
BEGIN
    IF c->>'denegar'='si' THEN
        RAISE EXCEPTION 'doble AD3: denegado' USING ERRCODE='42501';
    END IF;
    RETURN QUERY SELECT 'decision:'||gen_random_uuid()::text, c->>'efecto_ref', c->>'huella_efecto_sha256',
        encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'),
        'auditoria:'||gen_random_uuid()::text, clock_timestamp(), true;
END
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
    p_perfil text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog
AS $f$
BEGIN
    -- Operaciones declaradas: bolsa.llamamiento.aceptacion_rrhh.registrar,
    -- bolsa.llamamiento.renuncia_rrhh.registrar, bolsa.llamamiento.siguiente.abrir.
    RAISE EXCEPTION 'doble: sin uso directo' USING ERRCODE='42501';
END
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_bolsa_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_bolsa_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
RESET ROLE;
