-- CT-000042: tipos nominales privados; ningún JSON abierto cruza el canon.
CREATE TYPE vec_contratacion_temporal.resumen_publicacion_rrhh_v1 AS (
    expediente_ref text,
    organizacion_ref text,
    numero_visible text,
    version numeric(20, 0),
    flujo_ref text,
    flujo_version numeric(20, 0),
    flujo_huella_sha256 text,
    fase_clave text,
    estado_clave text,
    centro_ref text,
    categoria_ref text,
    modalidad_clave text,
    unidad_ref text,
    creado_en timestamptz(6),
    actualizado_en timestamptz(6)
);
CREATE TYPE vec_contratacion_temporal.solicitud_operativa_rrhh_v1 AS (
    grupo_subgrupo text,
    motivo_clave text,
    periodo_inicio timestamptz(6),
    periodo_fin timestamptz(6)
);
CREATE TYPE vec_contratacion_temporal.analisis_operativo_rrhh_v1 AS (
    modalidad_clave text,
    categoria_ref text,
    causa_clave text,
    periodo_inicio timestamptz(6),
    periodo_fin timestamptz(6),
    porcentaje_jornada smallint,
    resultado_rc text,
    coste_presente boolean,
    coste_centimos bigint,
    coste_moneda text,
    fuente_coste_ref text
);
CREATE TYPE
vec_contratacion_temporal.comprobacion_operativa_rrhh_v1 AS (
    clave text,
    resultado text
);
CREATE TYPE vec_contratacion_temporal.cobertura_operativa_rrhh_v1 AS (
    via_clave text,
    decision_gobernada boolean,
    procedimiento_ref text,
    bolsa_ref text,
    comprobaciones
        vec_contratacion_temporal.comprobacion_operativa_rrhh_v1[]
);
CREATE TYPE vec_contratacion_temporal.asignacion_operativa_rrhh_v1 AS (
    unidad_ref text,
    asignada_en timestamptz(6),
    motivo_clave text
);
CREATE TYPE vec_contratacion_temporal.hito_expediente_rrhh_v1 AS (
    secuencia numeric(20, 0),
    version_expediente numeric(20, 0),
    accion_clave text,
    realizada_en timestamptz(6),
    fase_origen text,
    fase_destino text,
    estado_origen text,
    estado_destino text
);
CREATE TYPE
vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1 AS (
    resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1,
    solicitud vec_contratacion_temporal.solicitud_operativa_rrhh_v1,
    analisis_presente boolean,
    analisis vec_contratacion_temporal.analisis_operativo_rrhh_v1,
    referencia_analisis numeric(20, 0),
    cobertura_presente boolean,
    cobertura vec_contratacion_temporal.cobertura_operativa_rrhh_v1,
    referencia_cobertura numeric(20, 0),
    asignacion_presente boolean,
    asignacion vec_contratacion_temporal.asignacion_operativa_rrhh_v1,
    referencia_asignacion numeric(20, 0),
    hitos vec_contratacion_temporal.hito_expediente_rrhh_v1[]
);
CREATE TYPE
vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2 AS (
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
    decision_ref text,
    decision_huella_sha256 text,
    capacidad_huella_sha256 text,
    material_huella_sha256 text,
    consulta_huella_sha256 text,
    correlacion_ref text,
    autenticacion_ref text,
    autenticacion_huella_sha256 text,
    sesion_ref text,
    control_sesion_ref text,
    control_sesion_revision numeric(20, 0),
    control_sesion_huella_sha256 text,
    actor_ref text,
    perfil_ref text,
    perfil_version numeric(20, 0),
    organizacion_ref text,
    clase_ambito text,
    ambito_ref text,
    accion text,
    finalidad text,
    expediente_ref text,
    version_expediente numeric(20, 0),
    total smallint,
    contenido_huella_sha256 text,
    resultado_huella_sha256 text,
    cursor_huella_sha256 text,
    generada_en timestamptz(6)
);
