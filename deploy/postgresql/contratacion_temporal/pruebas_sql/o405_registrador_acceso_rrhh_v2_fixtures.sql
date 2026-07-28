\set ON_ERROR_STOP on
\set QUIET on

SET search_path = pg_catalog;

DO $cuentas$
BEGIN
    PERFORM *
      FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
          'opr_aaaaaaaaaaaaaaaaaaaaaaaa',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa',
          'clave-hsm-prueba', 1,
          decode(repeat('11', 32), 'hex'),
          decode(repeat('22', 32), 'hex'),
          false, NULL
      );
    PERFORM *
      FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
          'opr_cccccccccccccccccccccccc',
          'vec.identidad.hmac-sha256.v1',
          'idh_cccccccccccccccccccccccc',
          'clave-hsm-prueba', 1,
          decode(repeat('55', 32), 'hex'),
          decode(repeat('66', 32), 'hex'),
          false, NULL
      );
END
$cuentas$;

CREATE FUNCTION vec_contratacion_temporal.c2d2_registrar_prueba(
    p_indice integer, p_tipo text, p_autenticacion text,
    p_sesion text, p_control text, p_revision numeric,
    p_control_huella text
)
RETURNS jsonb
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
BEGIN ATOMIC
SELECT vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(
    jsonb_build_object(
        'registro', jsonb_build_object(
            'accion', CASE p_tipo WHEN 'cuadro' THEN
                'contratacion_temporal.cuadro.consultar'
                ELSE 'contratacion_temporal.expediente.consultar' END,
            'actor_ref', 'actor:rrhh:' || p_indice::text,
            'ambito_ref', 'organizacion:rrhh:principal',
            'audiencia', CASE p_tipo WHEN 'cuadro' THEN
                'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'
                ELSE
                'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
                END,
            'auditoria_vec_huella_sha256',
                lpad(to_hex(p_indice + 80), 64, '8'),
            'auditoria_vec_ref', 'auditoria:vec:rrhh:' || p_indice::text,
            'capacidad_huella_sha256',
                lpad(to_hex(p_indice + 50), 64, '5'),
            'consulta_huella_sha256',
                lpad(to_hex(p_indice + 40), 64, '4'),
            'consumo_vec_huella_sha256',
                lpad(to_hex(p_indice + 60), 64, '6'),
            'correlacion_ref', 'correlacion:rrhh:' || p_indice::text,
            'decision_huella_sha256',
                lpad(to_hex(p_indice + 10), 64, '1'),
            'decision_ref', 'decision:rrhh:' || p_indice::text,
            'dominio_huella_consulta', CASE p_tipo WHEN 'cuadro' THEN
                'vec.contratacion_temporal.consulta_rrhh.cuadro.v1'
                ELSE
                'vec.contratacion_temporal.consulta_rrhh.detalle.v1' END,
            'expediente_ref', CASE p_tipo WHEN 'detalle'
                THEN 'expediente:rrhh:' || p_indice::text ELSE NULL END,
            'finalidad', CASE p_tipo WHEN 'cuadro' THEN
                'gestion_operativa_contratacion_temporal'
                ELSE 'tramitacion_expediente_contratacion_temporal' END,
            'modulo_id', 'contratacion_temporal',
            'organizacion_ref', 'organizacion:rrhh:principal',
            'perfil_id', 'perfil:rrhh:principal', 'perfil_version', 1,
            'recurso_ref', CASE p_tipo WHEN 'detalle'
                THEN 'expediente:rrhh:' || p_indice::text
                ELSE 'organizacion:rrhh:principal' END,
            'recurso_tipo', CASE p_tipo WHEN 'cuadro' THEN
                'cuadro_rrhh_contratacion_temporal'
                ELSE 'expediente_contratacion_temporal' END,
            'resultado_generico', 'entregado',
            'resultado_huella_sha256',
                lpad(to_hex(p_indice + 70), 64, '7'),
            'sesion_huella_sha256', p_control_huella,
            'sesion_id', p_sesion, 'tipo_consulta', p_tipo,
            'total', 1, 'version_expediente',
                CASE p_tipo WHEN 'detalle' THEN 3 ELSE NULL END
        ),
        'alcance', jsonb_build_object(
            'clase_ambito', 'organizacion', 'familia_ref', NULL
        ),
        'identidad', jsonb_build_object(
            'actor_ref', 'actor:rrhh:' || p_indice::text,
            'autenticacion_huella_sha256', repeat('a', 64),
            'autenticacion_ref', p_autenticacion,
            'control_sesion_huella_sha256', p_control_huella,
            'control_sesion_ref', p_control,
            'control_sesion_revision', p_revision,
            'organizacion_ref', 'organizacion:rrhh:principal',
            'perfil_ref', 'perfil:rrhh:principal', 'perfil_version', 1,
            'sesion_ref', p_sesion
        )
    )
);
END;

ALTER FUNCTION vec_contratacion_temporal.c2d2_registrar_prueba(
    integer, text, text, text, text, numeric, text
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.c2d2_registrar_prueba(
        integer, text, text, text, text, numeric, text
    ) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contratacion_temporal
    TO vec_contratacion_temporal_consultor_rrhh;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.c2d2_registrar_prueba(
        integer, text, text, text, text, numeric, text
    ) TO vec_contratacion_temporal_consultor_rrhh;

CREATE FUNCTION vec_contratacion_temporal.c2d2_verificar_prueba(
    p_cuenta_ref text,
    p_cuenta_ordinaria_ref text
)
RETURNS void
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
AS $verificar$
DECLARE
    base record;
BEGIN
    SELECT * INTO STRICT base
      FROM vec_contratacion_temporal.control_registrador_acceso_rrhh_v2
     WHERE control;
    IF (
        SELECT count(*)
          FROM vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
    ) <> 4 OR (
        SELECT count(*)
          FROM vec_contratacion_temporal.alcance_acceso_rrhh
         WHERE acceso_ref IN (
             SELECT acceso_ref
               FROM vec_contratacion_temporal
                    .vinculo_identidad_acceso_rrhh_v2
         )
    ) <> 3 OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .vinculo_identidad_acceso_rrhh_v2 vinculo
         WHERE vinculo.cuenta_ref <> p_cuenta_ref
            OR vinculo.cuenta_ordinaria_ref <> p_cuenta_ordinaria_ref
            OR vinculo.cuenta_ref !~ '^cta_[A-Za-z0-9_-]{22,128}$'
            OR vinculo.prueba_huella_sha256 <> encode(
                sha256(vinculo.prueba_canonica), 'hex'
            )
            OR convert_from(vinculo.prueba_canonica, 'UTF8')
               LIKE '%vec_c2d2_registro_runtime%'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_attribute atributo
         WHERE atributo.attrelid =
               'vec_contratacion_temporal.'
               'vinculo_identidad_acceso_rrhh_v2'::regclass
           AND atributo.attnum > 0 AND NOT atributo.attisdropped
           AND atributo.attname = 'login_tecnico'
    ) OR pg_catalog.pg_get_functiondef(pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.'
        'registrar_acceso_rrhh_interno_v2(jsonb)'
    )) LIKE '%login_tecnico%'
    OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.registro_acceso_rrhh acceso
         WHERE acceso.secuencia > base.secuencia_base
           AND convert_from(acceso.prueba_canonica, 'UTF8')
               LIKE '%vec_c2d2_registro_runtime%'
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.alcance_acceso_rrhh alcance
         WHERE alcance.acceso_ref IN (
             SELECT acceso_ref
               FROM vec_contratacion_temporal
                    .vinculo_identidad_acceso_rrhh_v2
         )
           AND convert_from(alcance.prueba_canonica, 'UTF8')
               LIKE '%vec_c2d2_registro_runtime%'
    ) OR EXISTS (
        SELECT 1
          FROM (
              SELECT acceso.*,
                     lag(
                         huella_sha256, 1, base.cabeza_base_sha256
                     ) OVER (ORDER BY secuencia) anterior_esperado
                FROM vec_contratacion_temporal.registro_acceso_rrhh acceso
               WHERE acceso.secuencia > base.secuencia_base
          ) cadena
         WHERE cadena.anterior_sha256 <> cadena.anterior_esperado
    ) THEN
        RAISE EXCEPTION 'cadena/vínculo v2 incorrectos';
    END IF;
    BEGIN
        UPDATE vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
           SET cuenta_ref = cuenta_ref;
        RAISE EXCEPTION 'vínculo mutable';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        TRUNCATE vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2;
        RAISE EXCEPTION 'vínculo truncable';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
END
$verificar$;
ALTER FUNCTION vec_contratacion_temporal.c2d2_verificar_prueba(text, text)
    OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.c2d2_verificar_prueba(text, text) FROM PUBLIC;

CREATE TABLE vec_contratacion_temporal.c2d2_marcador_privacidad_prueba (
    marcador_ref text PRIMARY KEY,
    valor_huella_sha256 text NOT NULL
        CHECK (valor_huella_sha256 ~ '^[0-9a-f]{64}$')
);
ALTER TABLE vec_contratacion_temporal.c2d2_marcador_privacidad_prueba
    OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON TABLE
    vec_contratacion_temporal.c2d2_marcador_privacidad_prueba FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.c2d2_centinela_prueba(
    p_indice integer, p_valor text, p_autenticacion text,
    p_sesion text, p_control text, p_revision numeric,
    p_control_huella text
)
RETURNS text
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
AS $centinela$
DECLARE
    marcador text := 'marcador:privacidad:' || p_indice::text;
BEGIN
    BEGIN
        PERFORM vec_contratacion_temporal.c2d2_registrar_prueba(
            p_indice, p_valor, p_autenticacion, p_sesion, p_control,
            p_revision, p_control_huella
        );
        RAISE EXCEPTION 'la entrada adversa fue aceptada';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        IF SQLERRM <> 'registro de acceso RRHH v2 inválido' THEN
            RAISE;
        END IF;
    END;
    INSERT INTO
        vec_contratacion_temporal.c2d2_marcador_privacidad_prueba
        (marcador_ref, valor_huella_sha256)
    VALUES (
        marcador,
        encode(sha256(convert_to(p_valor, 'UTF8')), 'hex')
    );
    RETURN marcador;
END
$centinela$;
ALTER FUNCTION vec_contratacion_temporal.c2d2_centinela_prueba(
    integer, text, text, text, text, numeric, text
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.c2d2_centinela_prueba(
        integer, text, text, text, text, numeric, text
    ) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.c2d2_centinela_prueba(
        integer, text, text, text, text, numeric, text
    ) TO vec_contratacion_temporal_consultor_rrhh;

SELECT 'a', autenticacion_ref, sesion_ref, control_sesion_ref,
       control_sesion_revision_texto, control_sesion_huella_sha256,
       cuenta_ref, cuenta_ordinaria_ref
FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
    'opr_bbbbbbbbbbbbbbbbbbbbbbbb',
    'vec.identidad.hmac-sha256.v1',
    'idh_aaaaaaaaaaaaaaaaaaaaaaaa',
    'clave-hsm-prueba', 1,
    decode(repeat('33', 32), 'hex'),
    decode(repeat('44', 32), 'hex'),
    decode(repeat('22', 32), 'hex'),
    decode(repeat('11', 32), 'hex'),
    NULL, false, 'interna_corporativa', 'kerberos_ad', 'alto',
    repeat('a', 64),
    date_trunc('microseconds', clock_timestamp() - interval '2 seconds'),
    date_trunc('microseconds', clock_timestamp() - interval '1 second'),
    date_trunc('microseconds', clock_timestamp() + interval '4 minutes'),
    'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
)
UNION ALL
SELECT 'b', autenticacion_ref, sesion_ref, control_sesion_ref,
       control_sesion_revision_texto, control_sesion_huella_sha256,
       cuenta_ref, cuenta_ordinaria_ref
FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
    'opr_dddddddddddddddddddddddd',
    'vec.identidad.hmac-sha256.v1',
    'idh_cccccccccccccccccccccccc',
    'clave-hsm-prueba', 1,
    decode(repeat('77', 32), 'hex'),
    decode(repeat('88', 32), 'hex'),
    decode(repeat('66', 32), 'hex'),
    decode(repeat('55', 32), 'hex'),
    NULL, false, 'interna_corporativa', 'kerberos_ad', 'alto',
    repeat('a', 64),
    date_trunc('microseconds', clock_timestamp() - interval '2 seconds'),
    date_trunc('microseconds', clock_timestamp() - interval '1 second'),
    date_trunc('microseconds', clock_timestamp() + interval '4 minutes'),
    'pga_cccccccccccccccccccccccc', repeat('d', 64)
)
ORDER BY 1;
