-- Reversion de contrato de lectura. No destruye el paquete KMS de 000004;
-- solo retira su exposicion por el wrapper autorizado.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DROP FUNCTION vec_bolsa_convocatorias.listar_borradores_v1(
    jsonb,jsonb,jsonb,bytea,bytea
);

CREATE FUNCTION vec_bolsa_convocatorias.listar_borradores_v1(
    p_selector jsonb, p_lectura jsonb, p_prueba jsonb,
    p_decision_canonica bytea, p_contexto_recurso_canonico bytea
)
RETURNS jsonb
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
BEGIN
    IF vec_bolsa_convocatorias.identidad_runtime_borrador_valida(
           'vec_bolsa_convocatorias_ejecutor_consulta', true
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.contexto_lectura_borrador_valido(
           p_contexto_recurso_canonico, p_lectura,
           'borradores:' || (p_lectura ->> 'organizacion_ref'), true
       ) IS NOT TRUE
       OR vec_autorizacion.revalidar_decision_borrador_convocatorias_v2(
           p_prueba, p_decision_canonica, p_contexto_recurso_canonico,
           'bolsa.convocatoria.borrador.listar',
           'coleccion_versiones_convocatoria_gobernada',
           p_lectura ->> 'recurso_ref',
           'consulta_interna_convocatorias',
           '["version_convocatoria"]'::jsonb, clock_timestamp()
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'listado de borradores no revalidado';
    END IF;
    RETURN vec_bolsa_convocatorias.listar_borradores_interna_v1(
        p_selector, p_lectura
    );
END
$funcion$;

REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.listar_borradores_v1(
    jsonb,jsonb,jsonb,bytea,bytea
) FROM PUBLIC,
       vec_bolsa_convocatorias_proyector_gobierno,
       vec_bolsa_convocatorias_registrador_atestacion,
       vec_bolsa_convocatorias_verificador_recibo;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.listar_borradores_v1(
    jsonb,jsonb,jsonb,bytea,bytea
) TO vec_bolsa_convocatorias_ejecutor_consulta;

DROP FUNCTION vec_bolsa_convocatorias.obtener_borrador_v1(
    text,jsonb,jsonb,bytea,bytea
);

CREATE FUNCTION vec_bolsa_convocatorias.obtener_borrador_v1(
    p_referencia text, p_lectura jsonb, p_prueba jsonb,
    p_decision_canonica bytea, p_contexto_recurso_canonico bytea
)
RETURNS TABLE (
    metadatos jsonb, sobre_cifrado bytea,
    huella_sobre_cifrado_sha256 text, algoritmo_cifrado text,
    clave_cifrado_ref text, generacion_clave_cifrado bigint,
    nonce bytea, etiqueta_autenticacion bytea,
    atestacion_cifrado_ref text,
    huella_atestacion_cifrado_sha256 text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
BEGIN
    IF vec_bolsa_convocatorias.identidad_runtime_borrador_valida(
           'vec_bolsa_convocatorias_ejecutor_consulta', true
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.contexto_lectura_borrador_valido(
           p_contexto_recurso_canonico, p_lectura, p_referencia, false
       ) IS NOT TRUE
       OR vec_autorizacion.revalidar_decision_borrador_convocatorias_v2(
           p_prueba, p_decision_canonica, p_contexto_recurso_canonico,
           'bolsa.convocatoria.borrador.consultar',
           'version_convocatoria_gobernada', p_referencia,
           'consulta_interna_convocatorias',
           '["version_convocatoria"]'::jsonb, clock_timestamp()
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'detalle de borrador no revalidado';
    END IF;
    RETURN QUERY SELECT *
      FROM vec_bolsa_convocatorias.obtener_borrador_interna_v1(
          p_referencia, p_lectura
      );
END
$funcion$;

REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.obtener_borrador_v1(
    text,jsonb,jsonb,bytea,bytea
) FROM PUBLIC,
       vec_bolsa_convocatorias_proyector_gobierno,
       vec_bolsa_convocatorias_registrador_atestacion,
       vec_bolsa_convocatorias_verificador_recibo;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.obtener_borrador_v1(
    text,jsonb,jsonb,bytea,bytea
) TO vec_bolsa_convocatorias_ejecutor_consulta;
DROP FUNCTION
    vec_bolsa_convocatorias.selector_lista_borradores_valido_v1(jsonb);
DROP FUNCTION
    vec_bolsa_convocatorias.texto_selector_borradores_valido_v1(text);
COMMIT;
