-- CT-000043: relaciones compuestas exactas y prueba de solo anexado.
-- Los sentinelas son columnas generadas: ninguna entrada puede aportarlos.
ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh
ADD COLUMN expediente_ref_prueba_v2 text
    GENERATED ALWAYS AS (
        COALESCE(expediente_ref, '')
    ) STORED,
ADD COLUMN version_expediente_prueba_v2 numeric(20, 0)
    GENERATED ALWAYS AS (
        COALESCE(version_expediente, 0::numeric)
    ) STORED;

-- La relación de resultado permanece activa también para cuadro. No usa
-- MATCH SIMPLE sobre los nulos originales, que omitiría toda la comprobación.
ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh
ADD CONSTRAINT registro_acceso_rrhh_prueba_resultado_v2_unica
UNIQUE (
    acceso_ref, tipo_consulta, expediente_ref_prueba_v2,
    version_expediente_prueba_v2, total,
    resultado_huella_sha256, registrada_en
),
ADD CONSTRAINT registro_acceso_rrhh_prueba_cadena_v2_unica
UNIQUE (
    acceso_ref, secuencia, anterior_sha256, huella_sha256
),
ADD CONSTRAINT registro_acceso_rrhh_prueba_vec_v2_unica
UNIQUE (
    acceso_ref, auditoria_vec_ref, auditoria_vec_huella_sha256,
    consumo_vec_huella_sha256, decision_ref,
    decision_huella_sha256, capacidad_huella_sha256,
    consulta_huella_sha256, correlacion_ref, accion, finalidad
);

ALTER TABLE
vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
ADD CONSTRAINT vinculo_identidad_acceso_rrhh_prueba_v2_unica
UNIQUE (
    acceso_ref, prueba_huella_sha256, autenticacion_ref,
    autenticacion_huella_sha256, sesion_ref, control_sesion_ref,
    control_sesion_revision, control_sesion_huella_sha256,
    actor_ref, perfil_ref, perfil_version, organizacion_ref,
    clase_ambito, ambito_ref, sesion_huella_sha256,
    acceso_registrado_en
);

ALTER TABLE vec_contratacion_temporal.alcance_acceso_rrhh
ADD CONSTRAINT alcance_acceso_rrhh_prueba_v2_unica
UNIQUE (acceso_ref, prueba_huella_sha256);

CREATE TABLE
vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 (
    acceso_ref text PRIMARY KEY,
    tipo_consulta text NOT NULL,
    expediente_ref text,
    version_expediente numeric(20, 0),
    expediente_ref_prueba_v2 text
        GENERATED ALWAYS AS (
            COALESCE(expediente_ref, '')
        ) STORED,
    version_expediente_prueba_v2 numeric(20, 0)
        GENERATED ALWAYS AS (
            COALESCE(version_expediente, 0::numeric)
        ) STORED,
    total smallint NOT NULL,
    generada_en timestamptz(6) NOT NULL,
    resumenes
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1[]
        NOT NULL,
    hay_mas boolean NOT NULL,
    cursor_material_huella_sha256 bytea NOT NULL,
    detalle
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1,
    contenido_canonico bytea NOT NULL,
    contenido_huella_sha256 text NOT NULL,
    cursor_huella_sha256 text,
    resultado_canonico bytea NOT NULL,
    resultado_huella_sha256 text NOT NULL,
    material_huella_sha256 text NOT NULL,
    revalidada_en timestamptz(6) NOT NULL,
    recibo_canonico bytea NOT NULL,
    recibo_sello_sha256 text NOT NULL UNIQUE,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    anterior_sha256 text NOT NULL,
    huella_sha256 text NOT NULL UNIQUE,
    vinculo_identidad_huella_sha256 text NOT NULL,
    alcance_acceso_ref text,
    alcance_huella_sha256 text,
    registrada_en timestamptz(6) NOT NULL,
    auditoria_vec_ref text NOT NULL,
    auditoria_vec_huella_sha256 text NOT NULL,
    consumo_vec_huella_sha256 text NOT NULL,
    decision_ref text NOT NULL,
    decision_huella_sha256 text NOT NULL,
    capacidad_huella_sha256 text NOT NULL,
    consulta_huella_sha256 text NOT NULL,
    correlacion_ref text NOT NULL,
    autenticacion_ref text NOT NULL,
    autenticacion_huella_sha256 text NOT NULL,
    sesion_ref text NOT NULL,
    sesion_huella_sha256 text NOT NULL,
    control_sesion_ref text NOT NULL,
    control_sesion_revision numeric(20, 0) NOT NULL,
    control_sesion_huella_sha256 text NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    perfil_version numeric(20, 0) NOT NULL,
    organizacion_ref text NOT NULL,
    clase_ambito text NOT NULL,
    ambito_ref text NOT NULL,
    accion text NOT NULL,
    finalidad text NOT NULL,
    FOREIGN KEY (
        acceso_ref, tipo_consulta, expediente_ref_prueba_v2,
        version_expediente_prueba_v2, total,
        resultado_huella_sha256, registrada_en
    ) REFERENCES vec_contratacion_temporal.registro_acceso_rrhh (
        acceso_ref, tipo_consulta, expediente_ref_prueba_v2,
        version_expediente_prueba_v2, total,
        resultado_huella_sha256, registrada_en
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        acceso_ref, secuencia, anterior_sha256, huella_sha256
    ) REFERENCES vec_contratacion_temporal.registro_acceso_rrhh (
        acceso_ref, secuencia, anterior_sha256, huella_sha256
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        acceso_ref, auditoria_vec_ref, auditoria_vec_huella_sha256,
        consumo_vec_huella_sha256, decision_ref,
        decision_huella_sha256, capacidad_huella_sha256,
        consulta_huella_sha256, correlacion_ref, accion, finalidad
    ) REFERENCES vec_contratacion_temporal.registro_acceso_rrhh (
        acceso_ref, auditoria_vec_ref, auditoria_vec_huella_sha256,
        consumo_vec_huella_sha256, decision_ref,
        decision_huella_sha256, capacidad_huella_sha256,
        consulta_huella_sha256, correlacion_ref, accion, finalidad
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        acceso_ref, vinculo_identidad_huella_sha256,
        autenticacion_ref, autenticacion_huella_sha256, sesion_ref,
        control_sesion_ref, control_sesion_revision,
        control_sesion_huella_sha256, actor_ref, perfil_ref,
        perfil_version, organizacion_ref, clase_ambito, ambito_ref,
        sesion_huella_sha256, registrada_en
    ) REFERENCES
      vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2 (
        acceso_ref, prueba_huella_sha256,
        autenticacion_ref, autenticacion_huella_sha256, sesion_ref,
        control_sesion_ref, control_sesion_revision,
        control_sesion_huella_sha256, actor_ref, perfil_ref,
        perfil_version, organizacion_ref, clase_ambito, ambito_ref,
        sesion_huella_sha256, acceso_registrado_en
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    -- Para detalle ambos valores son nulos; para cuadro ambos son no nulos.
    FOREIGN KEY (alcance_acceso_ref, alcance_huella_sha256)
    REFERENCES vec_contratacion_temporal.alcance_acceso_rrhh (
        acceso_ref, prueba_huella_sha256
    ) MATCH FULL ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        pg_catalog.octet_length(contenido_canonico)
            BETWEEN 1 AND 262144
        AND pg_catalog.octet_length(recibo_canonico)
            BETWEEN 1 AND 262144
        AND contenido_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND contenido_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND contenido_huella_sha256 = pg_catalog.encode(
            pg_catalog.sha256(contenido_canonico), 'hex'
        )
        AND resultado_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND resultado_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND resultado_huella_sha256 = pg_catalog.encode(
            pg_catalog.sha256(resultado_canonico), 'hex'
        )
        AND material_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND material_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND recibo_sello_sha256 ~ '^[0-9a-f]{64}$'
        AND recibo_sello_sha256 <> pg_catalog.repeat('0', 64)
        AND recibo_sello_sha256 = pg_catalog.encode(
            pg_catalog.sha256(recibo_canonico), 'hex'
        )
    ),
    CHECK (
        (
            tipo_consulta = 'cuadro'
            AND total = pg_catalog.cardinality(resumenes)
            AND detalle IS NOT DISTINCT FROM
                NULL::vec_contratacion_temporal
                    .entrada_detalle_expediente_rrhh_v1
            AND contenido_canonico =
                vec_contratacion_temporal
                .canon_contenido_cuadro_rrhh_v1(
                    generada_en, resumenes, hay_mas,
                    cursor_material_huella_sha256
                )
            AND cursor_huella_sha256 IS NOT DISTINCT FROM
                CASE WHEN hay_mas THEN pg_catalog.encode(
                    cursor_material_huella_sha256, 'hex'
                ) ELSE NULL END
        )
        OR (
            tipo_consulta = 'detalle'
            AND pg_catalog.array_ndims(resumenes) IS NULL
            AND NOT hay_mas
            AND pg_catalog.octet_length(
                cursor_material_huella_sha256
            ) = 0
            AND detalle IS DISTINCT FROM
                NULL::vec_contratacion_temporal
                    .entrada_detalle_expediente_rrhh_v1
            AND contenido_canonico =
                vec_contratacion_temporal
                .canon_contenido_detalle_rrhh_v1(
                    generada_en, detalle
                )
            AND cursor_huella_sha256 IS NULL
        )
    ),
    CHECK (
        resultado_canonico =
            vec_contratacion_temporal
            .canon_resultado_consulta_rrhh_puro_v1(ROW(
                tipo_consulta, generada_en, total,
                contenido_huella_sha256,
                COALESCE(cursor_huella_sha256, '')
            )::vec_contratacion_temporal.evidencia_resultado_rrhh_v1)
    ),
    CHECK (
        recibo_canonico =
            vec_contratacion_temporal.canon_recibo_lectura_rrhh_v2(
                ROW(
                    'vec.contratacion-temporal.'
                    || 'recibo-acceso-rrhh.o4-05.v2',
                    acceso_ref, secuencia, anterior_sha256,
                    huella_sha256,
                    vinculo_identidad_huella_sha256,
                    COALESCE(alcance_huella_sha256, ''),
                    registrada_en, auditoria_vec_ref,
                    auditoria_vec_huella_sha256,
                    consumo_vec_huella_sha256, decision_ref,
                    decision_huella_sha256, capacidad_huella_sha256,
                    material_huella_sha256, consulta_huella_sha256,
                    correlacion_ref, autenticacion_ref,
                    autenticacion_huella_sha256, sesion_ref,
                    control_sesion_ref, control_sesion_revision,
                    control_sesion_huella_sha256, actor_ref,
                    perfil_ref, perfil_version, organizacion_ref,
                    clase_ambito, ambito_ref, accion, finalidad,
                    expediente_ref_prueba_v2,
                    version_expediente_prueba_v2, total,
                    contenido_huella_sha256,
                    resultado_huella_sha256,
                    COALESCE(cursor_huella_sha256, ''),
                    generada_en
                )::vec_contratacion_temporal
                  .evidencia_recibo_lectura_rrhh_v2
            )
    ),
    CHECK (
        generada_en =
            pg_catalog.date_trunc('microseconds', generada_en)
        AND revalidada_en =
            pg_catalog.date_trunc('microseconds', revalidada_en)
        AND registrada_en =
            pg_catalog.date_trunc('microseconds', registrada_en)
        AND generada_en <= revalidada_en
        AND revalidada_en <= registrada_en
    ),
    CHECK (
        (
            tipo_consulta = 'cuadro'
            AND expediente_ref IS NULL
            AND version_expediente IS NULL
            AND total BETWEEN 0 AND 100
            AND alcance_acceso_ref = acceso_ref
            AND alcance_huella_sha256 IS NOT NULL
            AND accion =
                'contratacion_temporal.cuadro.consultar'
            AND finalidad =
                'gestion_operativa_contratacion_temporal'
            AND (
                cursor_huella_sha256 IS NULL
                OR total > 0
                   AND cursor_huella_sha256 ~ '^[0-9a-f]{64}$'
                   AND cursor_huella_sha256 <>
                       pg_catalog.repeat('0', 64)
            )
        )
        OR (
            tipo_consulta = 'detalle'
            AND expediente_ref ~
                '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
            AND version_expediente BETWEEN
                1 AND 9007199254740991::numeric
            AND total = 1
            AND expediente_ref =
                (detalle).resumen.expediente_ref
            AND version_expediente =
                (detalle).resumen.version
            AND alcance_acceso_ref IS NULL
            AND alcance_huella_sha256 IS NULL
            AND cursor_huella_sha256 IS NULL
            AND accion =
                'contratacion_temporal.expediente.consultar'
            AND finalidad =
                'tramitacion_expediente_contratacion_temporal'
        )
    )
);

CREATE TRIGGER prueba_resultado_recibo_rrhh_v2_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
FOR EACH ROW EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE TRIGGER prueba_resultado_recibo_rrhh_v2_no_truncar
BEFORE TRUNCATE
ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
FOR EACH STATEMENT EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();

ALTER TABLE vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total
ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
FOR ALL TO vec_contratacion_temporal_propietario
USING (true) WITH CHECK (true);

COMMENT ON COLUMN
vec_contratacion_temporal.registro_acceso_rrhh.expediente_ref_prueba_v2 IS
'CT-000043: sentinela generado para FK exacta de cuadro/detalle';
COMMENT ON COLUMN
vec_contratacion_temporal.registro_acceso_rrhh
    .version_expediente_prueba_v2 IS
'CT-000043: sentinela generado para FK exacta de cuadro/detalle';
COMMENT ON TABLE
vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 IS
'CT-000043: prueba minimizada de solo anexado de resultado y Recibo RRHH V2';
