ALTER TABLE vec_contratacion_temporal.candidatura_alta_tecnica
    OWNER TO vec_contratacion_temporal_propietario;
ALTER TABLE vec_contratacion_temporal.candidatura_alta_alias
    OWNER TO vec_contratacion_temporal_propietario;
ALTER TABLE vec_contratacion_temporal.candidatura_alta_tecnica
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.candidatura_alta_tecnica
    FORCE ROW LEVEL SECURITY;
CREATE POLICY candidatura_alta_tecnica_propietario
    ON vec_contratacion_temporal.candidatura_alta_tecnica
    TO vec_contratacion_temporal_propietario
    USING (true)
    WITH CHECK (true);
ALTER TABLE vec_contratacion_temporal.candidatura_alta_alias
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.candidatura_alta_alias
    FORCE ROW LEVEL SECURITY;
CREATE POLICY candidatura_alta_alias_propietario
    ON vec_contratacion_temporal.candidatura_alta_alias
    TO vec_contratacion_temporal_propietario
    USING (true)
    WITH CHECK (true);
ALTER FUNCTION vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(
    text[], text[], text, text, text, text, text, text, text, timestamptz
) OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION vec_contratacion_temporal.confirmar_alta_atestada_v2(
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea, bytea, bytea
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON TABLE
    vec_contratacion_temporal.candidatura_alta_tecnica,
    vec_contratacion_temporal.candidatura_alta_alias
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.preparar_alta_v2(jsonb),
    vec_contratacion_temporal.confirmar_alta_atestada_v1(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    ),
    vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(
        text[], text[], text, text, text, text, text, text, text, timestamptz
    ),
    vec_contratacion_temporal.confirmar_alta_atestada_v2(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    ) FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(
        text[], text[], text, text, text, text, text, text, text, timestamptz
    ),
    vec_contratacion_temporal.confirmar_alta_atestada_v2(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    ) TO vec_contratacion_temporal_ejecutor;
