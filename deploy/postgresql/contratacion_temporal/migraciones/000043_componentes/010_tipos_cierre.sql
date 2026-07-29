-- CT-000043: entradas y salida nominales de la primitiva privada.
CREATE TYPE
vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2 AS (
    organizacion_ref text,
    clase_ambito text,
    ambito_ref text,
    consulta_cuadro
        vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    consulta_detalle
        vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    familia_ref text
);

CREATE TYPE
vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2 AS (
    tipo_consulta text,
    generada_en timestamptz(6),
    resumenes vec_contratacion_temporal.resumen_publicacion_rrhh_v1[],
    hay_mas boolean,
    cursor_huella bytea,
    detalle vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1
);

-- Es la salida nominal del consumidor VEC ejecutado inmediatamente antes por
-- el futuro motor. La primitiva exige consumo_nuevo y vuelve a revalidar las
-- diez piezas originales; nunca acepta huellas de contenido, resultado,
-- material o recibo.
CREATE TYPE
vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3 AS (
    decision_ref text,
    efecto_ref text,
    huella_efecto_sha256 text,
    consumo_huella_sha256 text,
    auditoria_ref text,
    auditoria_huella_sha256 text,
    consumida_en timestamptz(6),
    consumo_nuevo boolean
);

CREATE TYPE
vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2 AS (
    esquema text,
    acceso_ref text,
    secuencia numeric(20, 0),
    anterior_sha256 text,
    huella_sha256 text,
    vinculo_identidad_huella_sha256 text,
    alcance_huella_sha256 text,
    registrada_en timestamptz(6),
    auditoria_vec_ref text,
    auditoria_vec_huella_sha256 text,
    consumo_vec_huella_sha256 text,
    contenido_huella_sha256 text,
    resultado_huella_sha256 text,
    cursor_huella_sha256 text,
    generada_en timestamptz(6),
    expediente_ref text,
    version_expediente numeric(20, 0),
    total smallint,
    recibo_sello_sha256 text
);
