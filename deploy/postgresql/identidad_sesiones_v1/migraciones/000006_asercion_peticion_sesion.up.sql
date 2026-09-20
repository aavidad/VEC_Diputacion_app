-- Consumo antirrepetición de una aserción ya verificada por la frontera.
-- No crea sesiones, cuentas, emisores ni autorización: solo liga una petición
-- al registro V1 existente mediante las coordenadas HMAC de su alta.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SELECT pg_advisory_xact_lock(
    hashtextextended('vec_identidad_sesiones_v1:migracion:asercion-peticion:v1', 0)
);
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL timezone = 'UTC';

DO $pre$
DECLARE propietario oid := 'vec_identidad_sesiones_v1_propietario'::regrole;
BEGIN
    IF to_regclass('vec_identidad_sesiones_v1.consumo_asercion') IS NULL
       OR to_regclass('vec_identidad_sesiones_v1.estado_cuenta_actual') IS NULL
       OR to_regclass('vec_autorizacion.sesion_autenticacion_v1') IS NULL
       OR to_regprocedure('vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)') IS NULL
       OR to_regprocedure('vec_identidad_sesiones_v1.revocar_sesion_v1(text,text,text,text)') IS NULL
       OR EXISTS (
           SELECT 1 FROM pg_class
            WHERE relnamespace = 'vec_identidad_sesiones_v1'::regnamespace
              AND relname = 'consumo_asercion_peticion_sesion_v1'
       )
       OR NOT has_schema_privilege(propietario, 'vec_autorizacion', 'USAGE')
       OR NOT has_table_privilege(propietario, 'vec_autorizacion.sesion_autenticacion_v1', 'SELECT')
       OR NOT has_table_privilege(propietario, 'vec_autorizacion.control_sesion_actual_v1', 'SELECT')
    THEN
        RAISE EXCEPTION 'precondiciones de asercion por peticion V1 no satisfechas'
            USING ERRCODE = '55000';
    END IF;
END
$pre$;

CREATE TABLE vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1 (
    nonce_sha256 text PRIMARY KEY,
    esquema_hmac text NOT NULL,
    dominio_hmac_ref text NOT NULL,
    clave_hmac_id text NOT NULL,
    clave_hmac_version bigint NOT NULL,
    sesion_id_hmac bytea NOT NULL,
    sesion_ref text NOT NULL,
    autenticacion_ref text NOT NULL,
    asercion_ref text NOT NULL,
    cuenta_ref text NOT NULL,
    control_sesion_ref text NOT NULL,
    control_sesion_revision numeric(20, 0) NOT NULL,
    superficie text NOT NULL,
    canal_vinculado_ref text NOT NULL,
    metodo text NOT NULL,
    destino text NOT NULL,
    cuerpo_sha256 text NOT NULL,
    emitida_en timestamptz(6) NOT NULL,
    expira_en timestamptz(6) NOT NULL,
    consumida_en timestamptz(6) NOT NULL,
    CONSTRAINT consumo_peticion_nonce CHECK (
        nonce_sha256 ~ '^[0-9a-f]{64}$'
        AND nonce_sha256 <> repeat('0', 64)
    ),
    CONSTRAINT consumo_peticion_coordenadas CHECK (
        vec_identidad_sesiones_v1.coordenadas_hmac_validas(
            esquema_hmac, dominio_hmac_ref, clave_hmac_id, clave_hmac_version
        ) IS TRUE
        AND vec_identidad_sesiones_v1.huella_hmac_valida(sesion_id_hmac) IS TRUE
    ),
    CONSTRAINT consumo_peticion_referencias CHECK (
        vec_identidad_sesiones_v1.referencia_valida(sesion_ref, 'ses_') IS TRUE
        AND vec_identidad_sesiones_v1.referencia_valida(autenticacion_ref, 'aut_') IS TRUE
        AND vec_identidad_sesiones_v1.referencia_valida(asercion_ref, 'ase_') IS TRUE
        AND vec_identidad_sesiones_v1.referencia_valida(cuenta_ref, 'cta_') IS TRUE
        AND vec_identidad_sesiones_v1.referencia_valida(control_sesion_ref, 'cse_') IS TRUE
        AND vec_identidad_sesiones_v1.texto_tecnico_valido(canal_vinculado_ref, 256) IS TRUE
    ),
    CONSTRAINT consumo_peticion_solicitud CHECK (
        superficie IN ('externa_personal', 'interna_corporativa', 'administracion_privilegiada')
        AND metodo IN ('GET', 'HEAD', 'POST', 'PUT', 'PATCH', 'DELETE')
        AND vec_identidad_sesiones_v1.texto_tecnico_valido(destino, 2048) IS TRUE
        AND cuerpo_sha256 ~ '^[0-9a-f]{64}$'
        AND cuerpo_sha256 <> repeat('0', 64)
        AND emitida_en <= consumida_en + interval '5 minutes'
        AND expira_en > emitida_en
        AND expira_en > consumida_en
    ),
    CONSTRAINT consumo_peticion_control_revision CHECK (
        control_sesion_revision BETWEEN 1 AND 18446744073709551615
    ),
    -- La tabla base no expone una UNIQUE sobre la coordenada de sesión HMAC:
    -- puede conservar sesiones históricas. La función cerrada resuelve la
    -- única sesión activa y guarda sus referencias canónicas antes del INSERT.
    FOREIGN KEY (sesion_ref)
        REFERENCES vec_autorizacion.sesion_autenticacion_v1(sesion_ref),
    FOREIGN KEY (sesion_ref, control_sesion_ref, control_sesion_revision)
        REFERENCES vec_autorizacion.control_sesion_v1(
            sesion_ref, control_sesion_ref, revision
        )
);

CREATE INDEX consumo_asercion_peticion_sesion_v1_sesion_idx
    ON vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1 (
        sesion_ref, consumida_en
    );

CREATE TRIGGER consumo_asercion_peticion_sesion_v1_inmutable
    BEFORE UPDATE OR DELETE ON vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1
    FOR EACH ROW EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();
CREATE TRIGGER consumo_asercion_peticion_sesion_v1_no_truncar
    BEFORE TRUNCATE ON vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1
    FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();
ALTER TABLE vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_propietario_exacto
    ON vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1
    FOR ALL TO vec_identidad_sesiones_v1_propietario
    USING (current_user = 'vec_identidad_sesiones_v1_propietario')
    WITH CHECK (current_user = 'vec_identidad_sesiones_v1_propietario');

CREATE FUNCTION vec_identidad_sesiones_v1.consumir_asercion_peticion_sesion_v1(
    p_esquema_hmac text,
    p_dominio_hmac_ref text,
    p_clave_hmac_id text,
    p_clave_hmac_version bigint,
    p_sesion_id_hmac bytea,
    p_nonce_sha256 text,
    p_canal_vinculado_ref text,
    p_superficie text,
    p_metodo text,
    p_destino text,
    p_cuerpo_sha256 text,
    p_emitida_en timestamptz,
    p_expira_en timestamptz
)
RETURNS TABLE(
    sesion_ref text,
    autenticacion_ref text,
    asercion_ref text,
    cuenta_ref text,
    control_sesion_ref text,
    control_sesion_revision text,
    sesion_valida_hasta timestamptz
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET row_security = on
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    base record;
    revalidada record;
    ahora timestamptz(6);
BEGIN
    IF vec_identidad_sesiones_v1.coordenadas_hmac_validas(
           p_esquema_hmac, p_dominio_hmac_ref, p_clave_hmac_id,
           p_clave_hmac_version
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.huella_hmac_valida(p_sesion_id_hmac) IS NOT TRUE
       OR p_nonce_sha256 !~ '^[0-9a-f]{64}$'
       OR p_nonce_sha256 = repeat('0', 64)
       OR vec_identidad_sesiones_v1.texto_tecnico_valido(p_canal_vinculado_ref, 256) IS NOT TRUE
       OR p_superficie NOT IN ('externa_personal', 'interna_corporativa', 'administracion_privilegiada')
       OR p_metodo NOT IN ('GET', 'HEAD', 'POST', 'PUT', 'PATCH', 'DELETE')
       OR vec_identidad_sesiones_v1.texto_tecnico_valido(p_destino, 2048) IS NOT TRUE
       OR p_cuerpo_sha256 !~ '^[0-9a-f]{64}$'
       OR p_cuerpo_sha256 = repeat('0', 64)
       OR p_emitida_en IS NULL OR p_expira_en IS NULL THEN
        RETURN;
    END IF;

    -- Serializa el nonce sin revelar su valor fuente; el índice único conserva
    -- la denegación definitiva incluso tras reinicio o rotación de sesiones.
    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_identidad_sesiones_v1:nonce-peticion:' || p_nonce_sha256, 0
    ));
    IF EXISTS (
        SELECT 1 FROM vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1
         WHERE nonce_sha256 = p_nonce_sha256
    ) THEN
        RETURN;
    END IF;

    SELECT consumo.autenticacion_ref, consumo.sesion_ref
      INTO STRICT base
      FROM vec_identidad_sesiones_v1.consumo_asercion AS consumo
      JOIN vec_autorizacion.control_sesion_actual_v1 AS actual
        ON actual.sesion_ref = consumo.sesion_ref
      JOIN vec_autorizacion.control_sesion_v1 AS control
        ON control.sesion_ref = actual.sesion_ref
       AND control.control_sesion_ref = actual.control_sesion_ref
       AND control.revision = actual.revision
     WHERE consumo.esquema_hmac = p_esquema_hmac
       AND consumo.dominio_hmac_ref = p_dominio_hmac_ref
       AND consumo.clave_hmac_id = p_clave_hmac_id
       AND consumo.clave_hmac_version = p_clave_hmac_version
       AND consumo.sesion_id_hmac = p_sesion_id_hmac
       AND control.estado = 'activa'
       AND control.sesion_valida_hasta > clock_timestamp();

    SELECT * INTO STRICT revalidada
      FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
          base.autenticacion_ref, base.sesion_ref
      );
    -- El reloj se toma después de todos los bloqueos. statement_timestamp()
    -- conservaría el inicio de la sentencia y admitiría una aserción caducada
    -- mientras esperaba el nonce o la revalidación de sesión.
    ahora := clock_timestamp();
    IF revalidada.superficie <> p_superficie
       OR p_emitida_en > ahora + interval '5 minutes'
       OR p_expira_en <= ahora
       OR p_expira_en <= p_emitida_en
       OR p_expira_en > revalidada.sesion_valida_hasta THEN
        RETURN;
    END IF;

    INSERT INTO vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1 (
        nonce_sha256, esquema_hmac, dominio_hmac_ref, clave_hmac_id,
        clave_hmac_version, sesion_id_hmac, sesion_ref, autenticacion_ref,
        asercion_ref, cuenta_ref, control_sesion_ref, control_sesion_revision,
        superficie, canal_vinculado_ref, metodo,
        destino, cuerpo_sha256, emitida_en, expira_en, consumida_en
    ) VALUES (
        p_nonce_sha256, p_esquema_hmac, p_dominio_hmac_ref, p_clave_hmac_id,
        p_clave_hmac_version, p_sesion_id_hmac, revalidada.sesion_ref,
        revalidada.autenticacion_ref, revalidada.asercion_ref,
        revalidada.cuenta_ref, revalidada.control_sesion_ref,
        revalidada.control_sesion_revision::numeric, p_superficie, p_canal_vinculado_ref, p_metodo,
        p_destino, p_cuerpo_sha256, p_emitida_en, p_expira_en, ahora
    );
    RETURN QUERY SELECT revalidada.sesion_ref, revalidada.autenticacion_ref,
        revalidada.asercion_ref, revalidada.cuenta_ref,
        revalidada.control_sesion_ref, revalidada.control_sesion_revision::text,
        revalidada.sesion_valida_hasta;
EXCEPTION
    WHEN no_data_found OR too_many_rows OR data_exception
        OR invalid_text_representation OR numeric_value_out_of_range
        OR unique_violation OR foreign_key_violation OR check_violation
        OR cardinality_violation THEN
        RETURN;
END
$funcion$;

REVOKE ALL ON TABLE vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1 FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.consumir_asercion_peticion_sesion_v1(
    text, text, text, bigint, bytea, text, text, text, text, text, text,
    timestamptz, timestamptz
) FROM PUBLIC;
REVOKE ALL ON TYPE vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1 FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.consumir_asercion_peticion_sesion_v1(
    text, text, text, bigint, bytea, text, text, text, text, text, text,
    timestamptz, timestamptz
) TO vec_identidad_sesiones_v1_revalidador;
COMMIT;
