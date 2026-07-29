\set ON_ERROR_STOP 1

CREATE FUNCTION
vec_contratacion_temporal.prueba_evidencia_consumo_consulta_rrhh_v3(
    p_caso text,
    p_fachada text,
    p_pieza text DEFAULT ''
)
RETURNS TABLE (
    decision_ref text,
    efecto_ref text,
    huella_efecto_sha256 text,
    consumo_huella_sha256 text,
    auditoria_ref text,
    auditoria_huella_sha256 text,
    consumida_en timestamptz,
    revalidada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v public.vectores_consulta_rrhh_v3%ROWTYPE;
BEGIN
    SELECT * INTO STRICT v
      FROM public.vectores_consulta_rrhh_v3
     WHERE caso = p_caso;
    CASE p_pieza
        WHEN '' THEN NULL;
        WHEN 'capacidad' THEN
            v.capacidad := pg_catalog.set_byte(
                v.capacidad, pg_catalog.octet_length(v.capacidad) - 1, 93
            );
        WHEN 'decision' THEN
            v.decision := pg_catalog.set_byte(
                v.decision, pg_catalog.octet_length(v.decision) - 1, 93
            );
        WHEN 'motivo' THEN
            v.motivo := pg_catalog.set_byte(v.motivo, 0, 91);
        WHEN 'contexto' THEN
            v.contexto := pg_catalog.set_byte(v.contexto, 0, 91);
        WHEN 'persona_version' THEN
            v.persona_version := v.persona_version + 1;
        WHEN 'perfil_version' THEN
            v.perfil_version := v.perfil_version + 1;
        WHEN 'payload' THEN
            v.payload := pg_catalog.set_byte(v.payload, 0, 91);
        WHEN 'cose' THEN
            v.cose := pg_catalog.set_byte(v.cose, 0, 91);
        WHEN 'evidencia' THEN
            v.evidencia := pg_catalog.set_byte(v.evidencia, 0, 91);
        WHEN 'spki' THEN
            v.spki := pg_catalog.set_byte(v.spki, 43, 255);
        ELSE
            RAISE EXCEPTION 'pieza de prueba desconocida';
    END CASE;

    IF p_fachada = 'cuadro' THEN
        RETURN QUERY
        SELECT *
          FROM vec_autorizacion_atestada_v3
               .revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada(
              v.capacidad, v.decision, v.motivo, v.contexto,
              v.persona_version, v.perfil_version,
              v.payload, v.cose, v.evidencia, v.spki
          );
    ELSIF p_fachada = 'detalle' THEN
        RETURN QUERY
        SELECT *
          FROM vec_autorizacion_atestada_v3
               .revalidar_evidencia_consumo_consulta_detalle_rrhh_v3_atestada(
              v.capacidad, v.decision, v.motivo, v.contexto,
              v.persona_version, v.perfil_version,
              v.payload, v.cose, v.evidencia, v.spki
          );
    ELSE
        RAISE EXCEPTION 'fachada de prueba desconocida';
    END IF;
END
$funcion$;

ALTER FUNCTION
    vec_contratacion_temporal.prueba_evidencia_consumo_consulta_rrhh_v3(
        text, text, text
    )
OWNER TO vec_contratacion_temporal_propietario;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.prueba_evidencia_consumo_consulta_rrhh_v3(
        text, text, text
    )
FROM PUBLIC;

GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.prueba_evidencia_consumo_consulta_rrhh_v3(
        text, text, text
    )
TO vec_contratacion_temporal_consultor_rrhh;
