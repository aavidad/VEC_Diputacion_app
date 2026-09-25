\set ON_ERROR_STOP on
-- Preimagen desechable PG18.4 para CT111. Ejecutar solo en una base vacía de
-- contenedor propio. Crea roles, esquemas, auxiliares y dobles explícitos de
-- los consumidores AD3; después el ejecutor aplica las migraciones CT reales
-- 000054, 000056-000060, 000062-000064 para reconstruir el cuerpo exacto de
-- la resolución manual. No acredita criptografía ni la cadena AD3 real.
DO $$ BEGIN
    IF pg_catalog.current_setting('server_version_num')::integer / 100 <> 1800
       OR pg_catalog.to_regnamespace('vec_contratacion_temporal') IS NOT NULL THEN
        RAISE EXCEPTION 'CT111 requiere PG18 desechable y base vacía';
    END IF;
END $$;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE ROLE vec_contratacion_temporal_propietario NOLOGIN NOINHERIT;
CREATE ROLE vec_contratacion_temporal_ejecutor NOLOGIN NOINHERIT;
CREATE ROLE vec_contratacion_temporal_migrador NOLOGIN NOINHERIT;
CREATE ROLE vec_autorizacion_atestada_v3_propietario NOLOGIN NOINHERIT;
-- Identidad de ejecución: miembro del ejecutor, nunca del propietario.
CREATE ROLE vec_ct111_runtime LOGIN INHERIT;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct111_runtime WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
-- Identidad ajena: sin pertenencia al ejecutor.
CREATE ROLE vec_ct111_ajeno LOGIN INHERIT;
GRANT CREATE ON DATABASE postgres TO vec_contratacion_temporal_propietario, vec_autorizacion_atestada_v3_propietario;

CREATE SCHEMA vec_contratacion_temporal AUTHORIZATION vec_contratacion_temporal_propietario;
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_contratacion_temporal_ejecutor;
CREATE SCHEMA vec_autorizacion_atestada_v3 AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_contratacion_temporal_propietario;

SET ROLE vec_contratacion_temporal_propietario;
-- Copias literales de 000001 y 000052.
CREATE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'la historia de contratación temporal es inmutable';
END
$funcion$;
CREATE FUNCTION vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
    p_documento jsonb,
    p_claves text[]
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.jsonb_typeof(p_documento) = 'object'
       AND ARRAY(
           SELECT clave
             FROM pg_catalog.jsonb_object_keys(p_documento) AS k(clave)
            ORDER BY clave
       ) = ARRAY(
           SELECT clave
             FROM pg_catalog.unnest(p_claves) clave
            ORDER BY clave
       )
$funcion$;
-- Doble mínimo de 000046: solo las columnas que leen CT54-CT64.
CREATE TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 (
    clave_idempotencia uuid PRIMARY KEY,
    situacion text NOT NULL,
    solicitud_json jsonb NOT NULL,
    recibo_json jsonb NOT NULL
);
ALTER TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
    TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 FROM PUBLIC;
RESET ROLE;

-- Doble de consumidor AD3: devuelve el efecto y la huella que declara la
-- decisión y un consumo nuevo con referencias aleatorias. Rechaza la
-- decisión marcada como denegada para probar el fallo cerrado.
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.doble_consumo_ct111(p_decision bytea)
RETURNS TABLE (
    decision_ref text,efecto_ref text,huella_efecto_sha256 text,
    consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog
AS $funcion$
DECLARE d jsonb := convert_from(p_decision,'UTF8')::jsonb;
BEGIN
    IF d->>'denegar'='si' THEN
        RAISE EXCEPTION 'doble AD3: denegado' USING ERRCODE='P0583';
    END IF;
    RETURN QUERY SELECT 'decision:'||gen_random_uuid()::text, d->>'recurso_ref',
        d->>'contexto_recurso_huella_sha256', encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'),
        'auditoria:'||gen_random_uuid()::text, clock_timestamp(), true;
END
$funcion$;
DO $dobles$
DECLARE v_nombre text;
BEGIN
    FOREACH v_nombre IN ARRAY ARRAY[
        'registrar_y_consumir_comunicacion_llamamiento_v3_atestada',
        'registrar_y_consumir_respuesta_recibida_rrhh_v3_atestada',
        'registrar_y_consumir_justificante_respuesta_ct_v3_atestada',
        'registrar_y_consumir_resolucion_manual_ct_v3_atestada',
        'registrar_y_consumir_continuacion_ct_v3_atestada'
    ] LOOP
        EXECUTE format($f$CREATE FUNCTION vec_autorizacion_atestada_v3.%I(
            p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
            p_persona_version numeric,p_perfil_version numeric,
            p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
        ) RETURNS TABLE (
            decision_ref text,efecto_ref text,huella_efecto_sha256 text,
            consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean
        ) LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = pg_catalog
        AS 'SELECT * FROM vec_autorizacion_atestada_v3.doble_consumo_ct111(p_decision)'$f$, v_nombre);
        EXECUTE format('REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.%I(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC', v_nombre);
        EXECUTE format('GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.%I(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_propietario', v_nombre);
    END LOOP;
END
$dobles$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.doble_consumo_ct111(bytea) FROM PUBLIC;
RESET ROLE;
