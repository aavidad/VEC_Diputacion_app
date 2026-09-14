\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000097',0));

-- Identidad funcional desde la primera reserva; no migra ni atribuye una
-- identidad retrospectiva a reservas anteriores. No contiene datos de RRHH.
CREATE TABLE vec_contratacion_temporal.identidad_reserva_operacion_analisis (
    ambito_raiz_hmac text NOT NULL REFERENCES vec_contratacion_temporal.reserva_operacion_analisis,
    generacion integer NOT NULL CHECK (generacion BETWEEN 1 AND 999999999),
    ambito_consulta_hmac text NOT NULL UNIQUE,
    huella_consulta_hmac text NOT NULL,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (ambito_raiz_hmac, generacion),
    CHECK (ambito_consulta_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'),
    CHECK (huella_consulta_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]huella-semantica/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'),
    CHECK (substring(ambito_consulta_hmac FROM '/v([1-9][0-9]{0,8}):')::integer = generacion),
    CHECK (substring(huella_consulta_hmac FROM '/v([1-9][0-9]{0,8}):')::integer = generacion)
);
CREATE TRIGGER identidad_reserva_operacion_analisis_inmutable
BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.identidad_reserva_operacion_analisis
FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
ALTER TABLE vec_contratacion_temporal.identidad_reserva_operacion_analisis ENABLE ROW LEVEL SECURITY;
CREATE POLICY identidad_reserva_operacion_analisis_propietario
ON vec_contratacion_temporal.identidad_reserva_operacion_analisis
TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true);
ALTER TABLE vec_contratacion_temporal.identidad_reserva_operacion_analisis FORCE ROW LEVEL SECURITY;
REVOKE ALL ON vec_contratacion_temporal.identidad_reserva_operacion_analisis
FROM PUBLIC, vec_contratacion_temporal_ejecutor;

-- Conserva CT7 y su contrato v1 sin reinterpretar su huella del artefacto.
-- La nueva identidad se guarda en la MISMA transacción que la reserva.
CREATE FUNCTION vec_contratacion_temporal.preparar_operacion_analisis_v2(p_operacion jsonb)
RETURNS TABLE (
    resultado text, expediente_json text, recibo_json text,
    reserva_ref text, recibo_ref text, operacion text,
    organizacion_ref text, expediente_ref text,
    version_expediente bigint, actor_ref text, perfil_ref text,
    artefacto_ref text, artefacto_huella_sha256 text,
    ambito_hmac text, huella_semantica_hmac text, estado text
)
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_fila record;
    v_consulta jsonb;
    v_pares jsonb;
    v_artefactos jsonb;
    v_par jsonb;
    v_base jsonb;
    v_raiz text;
    v_i integer;
    v_generacion integer;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR session_user = current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
        RAISE EXCEPTION 'identidad de ejecución no autorizada' USING ERRCODE='42501';
    END IF;
    v_consulta := p_operacion -> 'sellos_consulta';
    IF jsonb_typeof(v_consulta) IS DISTINCT FROM 'object'
       OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(v_consulta) AS x(k))
          IS DISTINCT FROM ARRAY['activo','retenidos']::text[]
       OR jsonb_typeof(v_consulta -> 'activo') IS DISTINCT FROM 'object'
       OR jsonb_typeof(v_consulta -> 'retenidos') IS DISTINCT FROM 'array'
       OR jsonb_array_length(v_consulta -> 'retenidos') > 3 THEN
        RAISE EXCEPTION 'identidad funcional de reserva inválida' USING ERRCODE='22023';
    END IF;
    -- CT7 valida el resto del contrato, actor, versiones, límites y sellos.
    SELECT * INTO STRICT v_fila
    FROM vec_contratacion_temporal.preparar_operacion_analisis_v1(p_operacion - 'sellos_consulta');
    v_pares := jsonb_build_array(v_consulta -> 'activo') || (v_consulta -> 'retenidos');
    v_artefactos := jsonb_build_array(p_operacion #> '{sellos_hmac,activo}')
                   || (p_operacion #> '{sellos_hmac,retenidos}');
    IF jsonb_array_length(v_pares) <> jsonb_array_length(v_artefactos) THEN
        RAISE EXCEPTION 'generaciones de reserva no alineadas' USING ERRCODE='22023';
    END IF;
    FOR v_i IN 0..jsonb_array_length(v_pares)-1 LOOP
        v_par := v_pares -> v_i;
        v_base := v_artefactos -> v_i;
        IF jsonb_typeof(v_par) IS DISTINCT FROM 'object'
           OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(v_par) AS x(k))
              IS DISTINCT FROM ARRAY['ambito_hmac','generacion','huella_peticion_hmac']::text[]
           OR (v_par -> 'generacion') IS DISTINCT FROM (v_base -> 'generacion')
           OR (v_par -> 'ambito_hmac') IS DISTINCT FROM (v_base -> 'ambito_hmac')
           OR coalesce(v_par ->> 'huella_peticion_hmac','') !~
              '^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]huella-semantica/v[1-9][0-9]{0,8}:[a-f0-9]{64}$' THEN
            RAISE EXCEPTION 'sello funcional de reserva inválido' USING ERRCODE='22023';
        END IF;
        v_generacion := (v_par ->> 'generacion')::integer;
        IF substring(v_par ->> 'huella_peticion_hmac' FROM '/v([1-9][0-9]{0,8}):')::integer <> v_generacion THEN
            RAISE EXCEPTION 'generación funcional no alineada' USING ERRCODE='22023';
        END IF;
    END LOOP;
    IF v_fila.resultado IN ('reservada','reutilizada') THEN
        SELECT a.ambito_raiz_hmac INTO STRICT v_raiz
        FROM vec_contratacion_temporal.alias_operacion_analisis a
        WHERE a.alias_ambito_hmac = v_fila.ambito_hmac;
        FOR v_i IN 0..jsonb_array_length(v_pares)-1 LOOP
            v_par := v_pares -> v_i;
            v_generacion := (v_par ->> 'generacion')::integer;
            IF v_fila.resultado = 'reservada' THEN
                INSERT INTO vec_contratacion_temporal.identidad_reserva_operacion_analisis
                    (ambito_raiz_hmac,generacion,ambito_consulta_hmac,huella_consulta_hmac)
                VALUES (v_raiz,v_generacion,v_par ->> 'ambito_hmac',v_par ->> 'huella_peticion_hmac');
            END IF;
            IF NOT EXISTS (
                SELECT 1 FROM vec_contratacion_temporal.identidad_reserva_operacion_analisis i
                WHERE i.ambito_raiz_hmac=v_raiz AND i.generacion=v_generacion
                  AND i.ambito_consulta_hmac=v_par ->> 'ambito_hmac'
                  AND i.huella_consulta_hmac=v_par ->> 'huella_peticion_hmac'
            ) THEN
                RAISE EXCEPTION 'reserva sin identidad funcional coincidente' USING ERRCODE='23505';
            END IF;
        END LOOP;
    END IF;
    RETURN QUERY SELECT v_fila.resultado,v_fila.expediente_json,v_fila.recibo_json,
        v_fila.reserva_ref,v_fila.recibo_ref,v_fila.operacion,v_fila.organizacion_ref,
        v_fila.expediente_ref,v_fila.version_expediente,v_fila.actor_ref,v_fila.perfil_ref,
        v_fila.artefacto_ref,v_fila.artefacto_huella_sha256,v_fila.ambito_hmac,
        v_fila.huella_semantica_hmac,v_fila.estado;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.preparar_operacion_analisis_v2(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.preparar_operacion_analisis_v2(jsonb)
TO vec_contratacion_temporal_ejecutor;
COMMIT;
