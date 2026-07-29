\set ON_ERROR_STOP on

-- CT-000044C: única ayuda del fixture para producir una revocación canónica.
-- El token claro solo vive durante la llamada y se reduce inmediatamente a
-- su SHA-256. La función no concede permisos ni crea una autoridad paralela.

CREATE FUNCTION
vec_contratacion_temporal.prueba_revocar_cursor_cuadro_ct44c(
    p_cursor text,
    p_etiqueta text
)
RETURNS text
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
DECLARE
    v_cursor_huella text;
    v_familia record;
    v_decision_ref text;
    v_decision_huella text;
    v_auditoria_ref text;
    v_auditoria_huella text;
    v_motivo_ref text;
    v_motivo_huella text;
    v_revocada_en timestamptz(6);
    v_prueba bytea;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_isolation')
          <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR p_cursor IS NULL
       OR p_cursor !~ '^[A-Za-z0-9_-]{43}$'
       OR p_etiqueta IS NULL
       OR p_etiqueta !~ '^[a-z0-9_]{3,40}$' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'revocación sintética CT44C rechazada';
    END IF;

    v_cursor_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(p_cursor, 'UTF8')
    ), 'hex');
    SELECT familia.familia_ref, familia.creada_en
      INTO STRICT v_familia
      FROM vec_contratacion_temporal.cursor_cuadro_rrhh cursor
      JOIN vec_contratacion_temporal
           .familia_cursor_cuadro_rrhh familia
        USING (familia_ref)
     WHERE cursor.token_huella_sha256 = v_cursor_huella
     FOR SHARE OF familia, cursor;

    v_decision_ref := 'decision:revocacion:ct44c:' || p_etiqueta;
    v_auditoria_ref := 'auditoria:revocacion:ct44c:' || p_etiqueta;
    v_motivo_ref := 'motivo:revocacion:ct44c:' || p_etiqueta;
    v_decision_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(v_decision_ref, 'UTF8')
    ), 'hex');
    v_auditoria_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(v_auditoria_ref, 'UTF8')
    ), 'hex');
    v_motivo_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(v_motivo_ref, 'UTF8')
    ), 'hex');
    v_revocada_en := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    IF v_revocada_en < v_familia.creada_en THEN
        v_revocada_en := v_familia.creada_en;
    END IF;
    v_prueba := pg_catalog.convert_to(
        'VEC-CT-REVOCACION-FAMILIA-CURSOR-RRHH-V1'
            || pg_catalog.chr(10), 'UTF8'
    )
    || vec_contratacion_temporal.encuadrar_texto_v1(
        v_familia.familia_ref
    )
    || vec_contratacion_temporal.encuadrar_texto_v1(
        vec_contratacion_temporal.instante_utc_v1(
            v_familia.creada_en
        )
    )
    || vec_contratacion_temporal.encuadrar_texto_v1(v_decision_ref)
    || vec_contratacion_temporal.encuadrar_texto_v1(v_decision_huella)
    || vec_contratacion_temporal.encuadrar_texto_v1(v_auditoria_ref)
    || vec_contratacion_temporal.encuadrar_texto_v1(v_auditoria_huella)
    || vec_contratacion_temporal.encuadrar_texto_v1(v_motivo_ref)
    || vec_contratacion_temporal.encuadrar_texto_v1('1')
    || vec_contratacion_temporal.encuadrar_texto_v1(v_motivo_huella)
    || vec_contratacion_temporal.encuadrar_texto_v1(
        vec_contratacion_temporal.instante_utc_v1(v_revocada_en)
    );

    INSERT INTO
    vec_contratacion_temporal.revocacion_familia_cursor_rrhh (
        familia_ref, familia_creada_en,
        decision_ref, decision_huella_sha256,
        auditoria_vec_ref, auditoria_vec_huella_sha256,
        motivo_ref, motivo_version, motivo_huella_sha256,
        revocada_en, prueba_canonica, prueba_huella_sha256
    ) VALUES (
        v_familia.familia_ref, v_familia.creada_en,
        v_decision_ref, v_decision_huella,
        v_auditoria_ref, v_auditoria_huella,
        v_motivo_ref, 1, v_motivo_huella,
        v_revocada_en, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
    );
    RETURN v_familia.familia_ref;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'revocación sintética CT44C rechazada';
END
$funcion$;

ALTER FUNCTION
vec_contratacion_temporal.prueba_revocar_cursor_cuadro_ct44c(text, text)
OWNER TO vec_contratacion_temporal_propietario;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.prueba_revocar_cursor_cuadro_ct44c(text, text)
FROM PUBLIC;
