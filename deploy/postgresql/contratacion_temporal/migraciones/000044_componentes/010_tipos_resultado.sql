-- CT-000044: tipos privados del motor atómico de consultas RRHH.
--
-- Los tipos de materialización separan la lectura de la coordinación. El
-- motor conserva una sola instancia de cada colección y entrega esa misma
-- instancia al canon de CT-000043 y a la salida; no vuelve a consultar.

CREATE TYPE
vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3 AS (
    capacidad_canonica bytea,
    decision_canonica bytea,
    motivo_canonico bytea,
    contexto_actor_canonico bytea,
    persona_version numeric(20, 0),
    perfil_version numeric(20, 0),
    payload_vec_ad_3 bytea,
    sobre_cose_sign_1 bytea,
    evidencia_verificacion bytea,
    raiz_publica_spki bytea
);

CREATE TYPE
vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1 AS (
    -- false representa la página inicial: pagina_presentada=0, corte ya
    -- fijado y todos los demás campos NULL. true representa una continuación:
    -- pagina_presentada>=2 y ningún campo restante puede ser NULL.
    es_continuacion boolean,
    familia_ref text,
    corte_global numeric(20, 0),
    pagina_presentada numeric(20, 0),
    token_presentado_huella_sha256 text,
    acceso_emision_ref text,
    cursor_emitida_en timestamptz(6),
    familia_creada_en timestamptz(6),
    familia_valida_hasta timestamptz(6),
    ultimo_actualizado_en timestamptz(6),
    ultimo_expediente_ref text
);

CREATE TYPE
vec_contratacion_temporal.materializacion_cuadro_rrhh_v1 AS (
    resumenes
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1[],
    hay_mas boolean,
    ultimo_actualizado_en timestamptz(6),
    ultimo_expediente_ref text
);

CREATE TYPE
vec_contratacion_temporal.materializacion_detalle_rrhh_v1 AS (
    detalle
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1
);

-- El token solo vive en este valor transitorio y en la respuesta provisional.
-- Las tablas y los cánones conservan exclusivamente su SHA-256.
CREATE TYPE
vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1 AS (
    -- Sin página siguiente: cursor_siguiente='', cursor_huella vacío,
    -- pagina_nueva=0 y referencias, huellas y clave de avance NULL.
    -- Con página siguiente: todos los campos son obligatorios; la página 2
    -- carece de padre y las posteriores enlazan el token consumido.
    hay_mas boolean,
    cursor_siguiente text,
    cursor_huella bytea,
    familia_ref text,
    pagina_nueva numeric(20, 0),
    token_nuevo_huella_sha256 text,
    padre_token_huella_sha256 text,
    ultimo_actualizado_en timestamptz(6),
    ultimo_expediente_ref text
);

CREATE TYPE
vec_contratacion_temporal.resultado_motor_cuadro_rrhh_v1 AS (
    generada_en timestamptz(6),
    resumenes
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1[],
    hay_mas boolean,
    cursor_siguiente text,
    cierre
        vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
);

CREATE TYPE
vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1 AS (
    generada_en timestamptz(6),
    detalle
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1,
    cierre
        vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
);

-- Ninguno de estos tipos constituye una frontera pública. Las futuras
-- fachadas autorizadas traducirán sus resultados sin conceder USAGE directo.
REVOKE ALL ON TYPE
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3,
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
    vec_contratacion_temporal.materializacion_cuadro_rrhh_v1,
    vec_contratacion_temporal.materializacion_detalle_rrhh_v1,
    vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1,
    vec_contratacion_temporal.resultado_motor_cuadro_rrhh_v1,
    vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_consultor_rrhh,
    vec_contratacion_temporal_lector_resultado_cobertura;
