BEGIN;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE FUNCTION vec_identidad_sesiones_v1.nueva_referencia(
    p_prefijo text
)
RETURNS text
LANGUAGE plpgsql
VOLATILE
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF p_prefijo NOT IN ('aut_', 'ase_', 'ses_', 'cse_', 'cta_', 'ali_') THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'prefijo de referencia no permitido';
    END IF;
    -- Dieciocho bytes CSPRNG: 144 bits antes de codificar en hexadecimal.
    RETURN p_prefijo || encode(public.gen_random_bytes(18), 'hex');
END
$funcion$;

CREATE FUNCTION vec_identidad_sesiones_v1.encuadrar(p_valor text)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT convert_to(
        octet_length(convert_to(p_valor, 'UTF8'))::text || ':' ||
        p_valor || E'\n',
        'UTF8'
    )
$funcion$;

CREATE FUNCTION vec_identidad_sesiones_v1.huella_control_sesion_v1(
    p_control_ref text,
    p_revision numeric,
    p_sesion_ref text,
    p_estado text,
    p_revalidada_en timestamptz,
    p_valida_hasta timestamptz,
    p_acto_ref text
)
RETURNS text
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT encode(public.digest(
        vec_identidad_sesiones_v1.encuadrar(
            'vec.identidad.control-sesion.v1'
        ) ||
        vec_identidad_sesiones_v1.encuadrar(p_control_ref) ||
        vec_identidad_sesiones_v1.encuadrar(p_revision::text) ||
        vec_identidad_sesiones_v1.encuadrar(p_sesion_ref) ||
        vec_identidad_sesiones_v1.encuadrar(p_estado) ||
        vec_identidad_sesiones_v1.encuadrar(to_char(
            p_revalidada_en AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )) ||
        vec_identidad_sesiones_v1.encuadrar(to_char(
            p_valida_hasta AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )) ||
        vec_identidad_sesiones_v1.encuadrar(p_acto_ref),
        'sha256'
    ), 'hex')
$funcion$;

CREATE FUNCTION vec_identidad_sesiones_v1.provisionar_cuenta_v1(
    p_operacion_ref text,
    p_esquema_hmac text,
    p_dominio_hmac_ref text,
    p_clave_hmac_id text,
    p_clave_hmac_version bigint,
    p_cuenta_id_hmac bytea,
    p_sujeto_id_hmac bytea,
    p_cuenta_privilegiada boolean,
    p_cuenta_ordinaria_id_hmac bytea
)
RETURNS TABLE(cuenta_ref text)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    cuenta_nueva_ref text;
    alias_nuevo_ref text;
    cuenta_ordinaria_ref text;
    ahora timestamptz(6);
    existente record;
    ordinaria_existente_ref text;
BEGIN
    IF vec_identidad_sesiones_v1.referencia_valida(
           p_operacion_ref, 'opr_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.coordenadas_hmac_validas(
           p_esquema_hmac, p_dominio_hmac_ref,
           p_clave_hmac_id, p_clave_hmac_version
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.huella_hmac_valida(
           p_cuenta_id_hmac
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.huella_hmac_valida(
           p_sujeto_id_hmac
       ) IS NOT TRUE
       OR p_cuenta_id_hmac = p_sujeto_id_hmac
       OR p_cuenta_privilegiada IS NULL
       OR (
           p_cuenta_privilegiada
           AND vec_identidad_sesiones_v1.huella_hmac_valida(
               p_cuenta_ordinaria_id_hmac
           ) IS NOT TRUE
       )
       OR (p_cuenta_privilegiada AND p_cuenta_ordinaria_id_hmac IS NULL)
       OR (NOT p_cuenta_privilegiada AND p_cuenta_ordinaria_id_hmac IS NOT NULL)
       OR p_cuenta_id_hmac = p_cuenta_ordinaria_id_hmac THEN
        RETURN;
    END IF;

    SELECT cuenta.cuenta_ref, cuenta.cuenta_privilegiada,
           cuenta.cuenta_ordinaria_ref, alias.esquema_hmac,
           alias.dominio_hmac_ref, alias.clave_hmac_id,
           alias.clave_hmac_version, alias.cuenta_id_hmac,
           alias.sujeto_id_hmac
      INTO existente
      FROM vec_identidad_sesiones_v1.cuenta AS cuenta
      JOIN vec_identidad_sesiones_v1.alias_hmac_cuenta AS alias
        ON alias.cuenta_ref = cuenta.cuenta_ref
     WHERE cuenta.acto_ref = p_operacion_ref
       AND alias.acto_ref = p_operacion_ref;
    IF FOUND THEN
        IF existente.cuenta_privilegiada IS DISTINCT FROM
               p_cuenta_privilegiada
           OR existente.esquema_hmac IS DISTINCT FROM p_esquema_hmac
           OR existente.dominio_hmac_ref IS DISTINCT FROM p_dominio_hmac_ref
           OR existente.clave_hmac_id IS DISTINCT FROM p_clave_hmac_id
           OR existente.clave_hmac_version IS DISTINCT FROM
               p_clave_hmac_version
           OR existente.cuenta_id_hmac IS DISTINCT FROM p_cuenta_id_hmac
           OR existente.sujeto_id_hmac IS DISTINCT FROM p_sujeto_id_hmac THEN
            RETURN;
        END IF;
        IF p_cuenta_privilegiada THEN
            SELECT cuenta.cuenta_ref
              INTO ordinaria_existente_ref
              FROM vec_identidad_sesiones_v1.alias_hmac_cuenta AS alias
              JOIN vec_identidad_sesiones_v1.cuenta AS cuenta
                ON cuenta.cuenta_ref = alias.cuenta_ref
             WHERE alias.esquema_hmac = p_esquema_hmac
               AND alias.dominio_hmac_ref = p_dominio_hmac_ref
               AND alias.clave_hmac_id = p_clave_hmac_id
               AND alias.clave_hmac_version = p_clave_hmac_version
               AND alias.cuenta_id_hmac = p_cuenta_ordinaria_id_hmac
               AND alias.sujeto_id_hmac = p_sujeto_id_hmac
               AND NOT cuenta.cuenta_privilegiada;
            IF NOT FOUND OR existente.cuenta_ordinaria_ref IS DISTINCT FROM
                   ordinaria_existente_ref THEN
                RETURN;
            END IF;
        ELSIF existente.cuenta_ordinaria_ref IS NOT NULL THEN
            RETURN;
        END IF;
        RETURN QUERY SELECT existente.cuenta_ref;
        RETURN;
    END IF;

    IF p_cuenta_privilegiada THEN
        SELECT cuenta.cuenta_ref
          INTO cuenta_ordinaria_ref
          FROM vec_identidad_sesiones_v1.alias_hmac_cuenta AS alias
          JOIN vec_identidad_sesiones_v1.cuenta AS cuenta
            ON cuenta.cuenta_ref = alias.cuenta_ref
          JOIN vec_identidad_sesiones_v1.estado_cuenta_actual AS actual
            ON actual.cuenta_ref = cuenta.cuenta_ref
          JOIN vec_identidad_sesiones_v1.estado_cuenta AS estado
            ON estado.cuenta_ref = actual.cuenta_ref
           AND estado.revision = actual.revision
         WHERE alias.esquema_hmac = p_esquema_hmac
           AND alias.dominio_hmac_ref = p_dominio_hmac_ref
           AND alias.clave_hmac_id = p_clave_hmac_id
           AND alias.clave_hmac_version = p_clave_hmac_version
           AND alias.cuenta_id_hmac = p_cuenta_ordinaria_id_hmac
           AND alias.sujeto_id_hmac = p_sujeto_id_hmac
           AND NOT cuenta.cuenta_privilegiada
           AND estado.estado = 'activa'
         FOR UPDATE OF actual;
        IF NOT FOUND THEN
            RETURN;
        END IF;
    END IF;

    ahora := clock_timestamp();
    cuenta_nueva_ref := vec_identidad_sesiones_v1.nueva_referencia('cta_');
    alias_nuevo_ref := vec_identidad_sesiones_v1.nueva_referencia('ali_');
    INSERT INTO vec_identidad_sesiones_v1.cuenta (
        cuenta_ref, cuenta_privilegiada, cuenta_ordinaria_ref,
        provisionada_en, acto_ref
    ) VALUES (
        cuenta_nueva_ref, p_cuenta_privilegiada, cuenta_ordinaria_ref,
        ahora, p_operacion_ref
    );
    INSERT INTO vec_identidad_sesiones_v1.alias_hmac_cuenta (
        alias_ref, cuenta_ref, esquema_hmac, dominio_hmac_ref,
        clave_hmac_id, clave_hmac_version, cuenta_id_hmac,
        sujeto_id_hmac, registrado_en, acto_ref
    ) VALUES (
        alias_nuevo_ref, cuenta_nueva_ref, p_esquema_hmac,
        p_dominio_hmac_ref, p_clave_hmac_id, p_clave_hmac_version,
        p_cuenta_id_hmac, p_sujeto_id_hmac, ahora, p_operacion_ref
    );
    INSERT INTO vec_identidad_sesiones_v1.estado_cuenta (
        cuenta_ref, revision, estado, registrada_en, acto_ref
    ) VALUES (cuenta_nueva_ref, 1, 'activa', ahora, p_operacion_ref);
    INSERT INTO vec_identidad_sesiones_v1.estado_cuenta_actual (
        cuenta_ref, revision, actualizada_en, acto_ref
    ) VALUES (cuenta_nueva_ref, 1, ahora, p_operacion_ref);
    RETURN QUERY SELECT cuenta_nueva_ref;
EXCEPTION
    WHEN data_exception OR invalid_parameter_value
        OR unique_violation OR foreign_key_violation OR check_violation THEN
        RETURN;
END
$funcion$;

-- Añade una coordenada de HMAC rotada a una cuenta ya provisionada sin
-- cambiar su cta_. La autoridad externa debe demostrar la equivalencia entre
-- claves; PostgreSQL no recibe ni intenta reconstruir el identificador IdP.
CREATE FUNCTION vec_identidad_sesiones_v1.registrar_alias_hmac_cuenta_v1(
    p_operacion_ref text,
    p_cuenta_ref text,
    p_esquema_hmac text,
    p_dominio_hmac_ref text,
    p_clave_hmac_id text,
    p_clave_hmac_version bigint,
    p_cuenta_id_hmac bytea,
    p_sujeto_id_hmac bytea
)
RETURNS text
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    existente record;
    estado_actual text;
    alias_nuevo_ref text;
    ahora timestamptz(6);
BEGIN
    IF vec_identidad_sesiones_v1.referencia_valida(
           p_operacion_ref, 'opr_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           p_cuenta_ref, 'cta_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.coordenadas_hmac_validas(
           p_esquema_hmac, p_dominio_hmac_ref,
           p_clave_hmac_id, p_clave_hmac_version
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.huella_hmac_valida(
           p_cuenta_id_hmac
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.huella_hmac_valida(
           p_sujeto_id_hmac
       ) IS NOT TRUE
       OR p_cuenta_id_hmac = p_sujeto_id_hmac THEN
        RETURN NULL;
    END IF;
    SELECT alias.cuenta_ref, alias.esquema_hmac, alias.dominio_hmac_ref,
           alias.clave_hmac_id, alias.clave_hmac_version,
           alias.cuenta_id_hmac, alias.sujeto_id_hmac
      INTO existente
      FROM vec_identidad_sesiones_v1.alias_hmac_cuenta AS alias
     WHERE alias.acto_ref = p_operacion_ref;
    IF FOUND THEN
        IF existente.cuenta_ref = p_cuenta_ref
           AND existente.esquema_hmac = p_esquema_hmac
           AND existente.dominio_hmac_ref = p_dominio_hmac_ref
           AND existente.clave_hmac_id = p_clave_hmac_id
           AND existente.clave_hmac_version = p_clave_hmac_version
           AND existente.cuenta_id_hmac = p_cuenta_id_hmac
           AND existente.sujeto_id_hmac = p_sujeto_id_hmac THEN
            RETURN p_cuenta_ref;
        END IF;
        RETURN NULL;
    END IF;

    SELECT estado.estado
      INTO STRICT estado_actual
      FROM vec_identidad_sesiones_v1.cuenta AS cuenta
      JOIN vec_identidad_sesiones_v1.estado_cuenta_actual AS actual
        ON actual.cuenta_ref = cuenta.cuenta_ref
      JOIN vec_identidad_sesiones_v1.estado_cuenta AS estado
        ON estado.cuenta_ref = actual.cuenta_ref
       AND estado.revision = actual.revision
     WHERE cuenta.cuenta_ref = p_cuenta_ref
     FOR UPDATE OF actual;
    IF estado_actual <> 'activa' THEN
        RETURN NULL;
    END IF;
    ahora := clock_timestamp();
    alias_nuevo_ref := vec_identidad_sesiones_v1.nueva_referencia('ali_');
    INSERT INTO vec_identidad_sesiones_v1.alias_hmac_cuenta (
        alias_ref, cuenta_ref, esquema_hmac, dominio_hmac_ref,
        clave_hmac_id, clave_hmac_version, cuenta_id_hmac,
        sujeto_id_hmac, registrado_en, acto_ref
    ) VALUES (
        alias_nuevo_ref, p_cuenta_ref, p_esquema_hmac,
        p_dominio_hmac_ref, p_clave_hmac_id, p_clave_hmac_version,
        p_cuenta_id_hmac, p_sujeto_id_hmac, ahora, p_operacion_ref
    );
    RETURN p_cuenta_ref;
EXCEPTION
    WHEN no_data_found OR too_many_rows OR data_exception
        OR unique_violation OR foreign_key_violation OR check_violation THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION vec_identidad_sesiones_v1.registrar_sesion_v1(
    p_operacion_ref text,
    p_esquema_hmac text,
    p_dominio_hmac_ref text,
    p_clave_hmac_id text,
    p_clave_hmac_version bigint,
    p_asercion_id_hmac bytea,
    p_sesion_id_hmac bytea,
    p_sujeto_id_hmac bytea,
    p_cuenta_id_hmac bytea,
    p_cuenta_ordinaria_id_hmac bytea,
    p_cuenta_privilegiada boolean,
    p_superficie text,
    p_metodo_observado text,
    p_garantia_observada text,
    p_autenticacion_huella_sha256 text,
    p_autenticacion_verificada_en timestamptz,
    p_sesion_emitida_en timestamptz,
    p_asercion_expira_en timestamptz,
    p_politica_garantia_ref text,
    p_politica_garantia_huella_sha256 text
)
RETURNS TABLE(
    autenticacion_ref text,
    asercion_ref text,
    sesion_ref text,
    control_sesion_ref text,
    control_sesion_revision_texto text,
    control_sesion_estado text,
    control_sesion_huella_sha256 text,
    cuenta_ref text,
    cuenta_ordinaria_ref text,
    sesion_revalidada_en timestamptz,
    sesion_valida_hasta timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    cuenta_base record;
    cuenta_ordinaria record;
    cuenta_ordinaria_resuelta_ref text;
    cuenta_base_revision numeric;
    cuenta_ordinaria_revision numeric;
    estado_bloqueado record;
    cuentas_activas integer := 0;
    autenticacion_nueva_ref text;
    asercion_nueva_ref text;
    sesion_nueva_ref text;
    control_nuevo_ref text;
    control_huella text;
    ahora timestamptz(6);
BEGIN
    IF vec_identidad_sesiones_v1.referencia_valida(
           p_operacion_ref, 'opr_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.coordenadas_hmac_validas(
           p_esquema_hmac, p_dominio_hmac_ref,
           p_clave_hmac_id, p_clave_hmac_version
       ) IS NOT TRUE
       OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               p_asercion_id_hmac, p_sesion_id_hmac, p_sujeto_id_hmac,
               p_cuenta_id_hmac
           ]) AS huella
           WHERE vec_identidad_sesiones_v1.huella_hmac_valida(
               huella
           ) IS NOT TRUE
       )
       OR p_asercion_id_hmac IN (
           p_sesion_id_hmac, p_sujeto_id_hmac, p_cuenta_id_hmac
       )
       OR p_sesion_id_hmac IN (p_sujeto_id_hmac, p_cuenta_id_hmac)
       OR p_sujeto_id_hmac = p_cuenta_id_hmac
       OR p_cuenta_privilegiada IS NULL
       OR (p_cuenta_privilegiada AND (
           vec_identidad_sesiones_v1.huella_hmac_valida(
               p_cuenta_ordinaria_id_hmac
           ) IS NOT TRUE
           OR p_cuenta_ordinaria_id_hmac IN (
               p_asercion_id_hmac, p_sesion_id_hmac,
               p_sujeto_id_hmac, p_cuenta_id_hmac
           )
       ))
       OR (NOT p_cuenta_privilegiada
           AND p_cuenta_ordinaria_id_hmac IS NOT NULL)
       OR p_superficie IS NULL OR p_superficie NOT IN (
           'externa_personal', 'interna_corporativa',
           'administracion_privilegiada'
       )
       OR p_metodo_observado IS NULL OR p_metodo_observado NOT IN (
           'certificado', 'dnie', 'sso', 'clave', 'kerberos_ad'
       )
       OR p_garantia_observada IS NULL
       OR p_garantia_observada NOT IN ('bajo', 'sustancial', 'alto')
       OR (p_superficie = 'externa_personal'
           AND p_garantia_observada = 'bajo')
       OR (p_superficie IN (
               'interna_corporativa', 'administracion_privilegiada'
           ) AND p_garantia_observada <> 'alto')
       OR p_autenticacion_huella_sha256 IS NULL
       OR p_autenticacion_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_autenticacion_huella_sha256 = repeat('0', 64)
       OR p_politica_garantia_ref IS NULL
       OR p_politica_garantia_ref !~ '^pga_[A-Za-z0-9_-]{22,128}$'
       OR p_politica_garantia_huella_sha256 IS NULL
       OR p_politica_garantia_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_politica_garantia_huella_sha256 = repeat('0', 64)
       OR p_autenticacion_verificada_en IS NULL
       OR p_sesion_emitida_en IS NULL OR p_asercion_expira_en IS NULL
       OR p_autenticacion_verificada_en > p_sesion_emitida_en
       OR p_asercion_expira_en <= p_sesion_emitida_en
       OR p_asercion_expira_en - p_sesion_emitida_en >
           interval '5 minutes'
       OR (p_cuenta_privilegiada AND
           p_superficie <> 'administracion_privilegiada')
       OR (NOT p_cuenta_privilegiada AND
           p_superficie = 'administracion_privilegiada')
       OR EXISTS (
           SELECT 1 FROM vec_identidad_sesiones_v1.consumo_asercion
            WHERE operacion_ref = p_operacion_ref
       ) THEN
        RETURN;
    END IF;

    -- Una sesion IdP puede reemitir aserciones, pero nunca mantiene dos
    -- sesiones VEC simultaneas dentro de una coordenada HMAC. El bloqueo no
    -- contiene datos fuente y una colision solo sobre-serializa de forma segura.
    PERFORM pg_advisory_xact_lock(pg_catalog.hashtextextended(
        'vec_identidad_sesiones_v1:sesion:' || p_dominio_hmac_ref || ':' ||
        p_clave_hmac_id || ':' || p_clave_hmac_version::text || ':' ||
        encode(p_sesion_id_hmac, 'hex'),
        0
    ));

    SELECT cuenta.cuenta_ref, cuenta.cuenta_privilegiada,
           cuenta.cuenta_ordinaria_ref
      INTO cuenta_base
      FROM vec_identidad_sesiones_v1.alias_hmac_cuenta AS alias
      JOIN vec_identidad_sesiones_v1.cuenta AS cuenta
        ON cuenta.cuenta_ref = alias.cuenta_ref
     WHERE alias.esquema_hmac = p_esquema_hmac
       AND alias.dominio_hmac_ref = p_dominio_hmac_ref
       AND alias.clave_hmac_id = p_clave_hmac_id
       AND alias.clave_hmac_version = p_clave_hmac_version
       AND alias.cuenta_id_hmac = p_cuenta_id_hmac
       AND alias.sujeto_id_hmac = p_sujeto_id_hmac;
    IF NOT FOUND OR cuenta_base.cuenta_privilegiada IS DISTINCT FROM
           p_cuenta_privilegiada THEN
        RETURN;
    END IF;

    IF p_cuenta_privilegiada THEN
        SELECT cuenta.cuenta_ref, cuenta.cuenta_privilegiada
          INTO cuenta_ordinaria
          FROM vec_identidad_sesiones_v1.alias_hmac_cuenta AS alias
          JOIN vec_identidad_sesiones_v1.cuenta AS cuenta
            ON cuenta.cuenta_ref = alias.cuenta_ref
         WHERE alias.esquema_hmac = p_esquema_hmac
           AND alias.dominio_hmac_ref = p_dominio_hmac_ref
           AND alias.clave_hmac_id = p_clave_hmac_id
           AND alias.clave_hmac_version = p_clave_hmac_version
           AND alias.cuenta_id_hmac = p_cuenta_ordinaria_id_hmac
           AND alias.sujeto_id_hmac = p_sujeto_id_hmac;
        IF NOT FOUND OR cuenta_ordinaria.cuenta_privilegiada
           OR cuenta_base.cuenta_ordinaria_ref IS DISTINCT FROM
               cuenta_ordinaria.cuenta_ref THEN
            RETURN;
        END IF;
        cuenta_ordinaria_resuelta_ref := cuenta_ordinaria.cuenta_ref;
    ELSE
        cuenta_ordinaria_resuelta_ref := cuenta_base.cuenta_ref;
    END IF;

    -- El orden global evita interbloqueos entre altas privilegiadas y cambios
    -- de estado. Toda mutacion de cuenta bloquea el mismo puntero estable.
    FOR estado_bloqueado IN
        SELECT actual.cuenta_ref, actual.revision, estado.estado
          FROM vec_identidad_sesiones_v1.estado_cuenta_actual AS actual
          JOIN vec_identidad_sesiones_v1.estado_cuenta AS estado
            ON estado.cuenta_ref = actual.cuenta_ref
           AND estado.revision = actual.revision
         WHERE actual.cuenta_ref IN (
             cuenta_base.cuenta_ref, cuenta_ordinaria_resuelta_ref
         )
         ORDER BY actual.cuenta_ref COLLATE "C"
         FOR UPDATE OF actual
    LOOP
        IF estado_bloqueado.estado = 'activa' THEN
            cuentas_activas := cuentas_activas + 1;
            IF estado_bloqueado.cuenta_ref = cuenta_base.cuenta_ref THEN
                cuenta_base_revision := estado_bloqueado.revision;
            END IF;
            IF estado_bloqueado.cuenta_ref =
                   cuenta_ordinaria_resuelta_ref THEN
                cuenta_ordinaria_revision := estado_bloqueado.revision;
            END IF;
        END IF;
    END LOOP;
    IF cuentas_activas <>
           (CASE WHEN p_cuenta_privilegiada THEN 2 ELSE 1 END) THEN
        RETURN;
    END IF;

    ahora := clock_timestamp();
    IF ahora < p_sesion_emitida_en OR ahora >= p_asercion_expira_en
       OR ahora >= p_autenticacion_verificada_en + (
           CASE p_superficie
               WHEN 'externa_personal' THEN interval '12 hours'
               WHEN 'interna_corporativa' THEN interval '15 minutes'
               WHEN 'administracion_privilegiada' THEN interval '5 minutes'
           END
       )
       OR EXISTS (
           SELECT 1
             FROM vec_identidad_sesiones_v1.consumo_asercion AS consumo
             JOIN vec_autorizacion.control_sesion_v1 AS control
               ON control.sesion_ref = consumo.sesion_ref
              AND control.control_sesion_ref = consumo.control_sesion_ref
              AND control.revision = consumo.control_sesion_revision
            WHERE consumo.esquema_hmac = p_esquema_hmac
              AND consumo.dominio_hmac_ref = p_dominio_hmac_ref
              AND consumo.clave_hmac_id = p_clave_hmac_id
              AND consumo.clave_hmac_version = p_clave_hmac_version
              AND consumo.sesion_id_hmac = p_sesion_id_hmac
              AND control.sesion_valida_hasta > ahora
       ) THEN
        RETURN;
    END IF;
    autenticacion_nueva_ref :=
        vec_identidad_sesiones_v1.nueva_referencia('aut_');
    asercion_nueva_ref :=
        vec_identidad_sesiones_v1.nueva_referencia('ase_');
    sesion_nueva_ref :=
        vec_identidad_sesiones_v1.nueva_referencia('ses_');
    control_nuevo_ref :=
        vec_identidad_sesiones_v1.nueva_referencia('cse_');
    control_huella := vec_identidad_sesiones_v1.huella_control_sesion_v1(
        control_nuevo_ref, 1, sesion_nueva_ref, 'activa', ahora,
        p_asercion_expira_en, p_operacion_ref
    );

    INSERT INTO vec_autorizacion.sesion_autenticacion_v1 (
        sesion_ref, autenticacion_ref, autenticacion_huella_sha256,
        asercion_ref, cuenta_ref, cuenta_ordinaria_ref,
        cuenta_privilegiada, superficie, metodo_observado,
        garantia_observada, politica_garantia_ref,
        politica_garantia_huella_sha256, autenticacion_verificada_en,
        sesion_emitida_en
    ) VALUES (
        sesion_nueva_ref, autenticacion_nueva_ref,
        p_autenticacion_huella_sha256, asercion_nueva_ref,
        cuenta_base.cuenta_ref, cuenta_ordinaria_resuelta_ref,
        p_cuenta_privilegiada, p_superficie, p_metodo_observado,
        p_garantia_observada, p_politica_garantia_ref,
        p_politica_garantia_huella_sha256,
        p_autenticacion_verificada_en, p_sesion_emitida_en
    );
    INSERT INTO vec_autorizacion.control_sesion_v1 (
        control_sesion_ref, revision, sesion_ref, estado, huella_sha256,
        sesion_revalidada_en, sesion_valida_hasta
    ) VALUES (
        control_nuevo_ref, 1, sesion_nueva_ref, 'activa', control_huella,
        ahora, p_asercion_expira_en
    );
    INSERT INTO vec_autorizacion.control_sesion_actual_v1 (
        sesion_ref, control_sesion_ref, revision, actualizada_en, acto_ref
    ) VALUES (
        sesion_nueva_ref, control_nuevo_ref, 1, ahora, p_operacion_ref
    );
    INSERT INTO vec_identidad_sesiones_v1.consumo_asercion (
        operacion_ref, esquema_hmac, dominio_hmac_ref, clave_hmac_id,
        clave_hmac_version, asercion_id_hmac, sesion_id_hmac,
        sujeto_id_hmac, cuenta_id_hmac, cuenta_ordinaria_id_hmac,
        autenticacion_ref, autenticacion_huella_sha256,
        asercion_ref, sesion_ref, control_sesion_ref,
        control_sesion_revision, cuenta_ref, cuenta_revision,
        cuenta_ordinaria_ref, cuenta_ordinaria_revision, consumida_en
    ) VALUES (
        p_operacion_ref, p_esquema_hmac, p_dominio_hmac_ref,
        p_clave_hmac_id, p_clave_hmac_version, p_asercion_id_hmac,
        p_sesion_id_hmac, p_sujeto_id_hmac, p_cuenta_id_hmac,
        p_cuenta_ordinaria_id_hmac, autenticacion_nueva_ref,
        p_autenticacion_huella_sha256, asercion_nueva_ref,
        sesion_nueva_ref, control_nuevo_ref, 1,
        cuenta_base.cuenta_ref, cuenta_base_revision,
        cuenta_ordinaria_resuelta_ref, cuenta_ordinaria_revision, ahora
    );

    RETURN QUERY SELECT
        autenticacion_nueva_ref, asercion_nueva_ref, sesion_nueva_ref,
        control_nuevo_ref, '1'::text, 'activa'::text, control_huella,
        cuenta_base.cuenta_ref, cuenta_ordinaria_resuelta_ref,
        ahora, p_asercion_expira_en;
EXCEPTION
    WHEN data_exception OR invalid_parameter_value OR unique_violation
        OR foreign_key_violation OR check_violation OR cardinality_violation THEN
        RETURN;
END
$funcion$;

CREATE FUNCTION vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(
    p_operacion_ref text,
    p_esquema_hmac text,
    p_dominio_hmac_ref text,
    p_clave_hmac_id text,
    p_clave_hmac_version bigint,
    p_asercion_id_hmac bytea,
    p_sesion_id_hmac bytea,
    p_sujeto_id_hmac bytea,
    p_cuenta_id_hmac bytea,
    p_cuenta_ordinaria_id_hmac bytea,
    p_cuenta_privilegiada boolean,
    p_superficie text,
    p_metodo_observado text,
    p_garantia_observada text,
    p_autenticacion_huella_sha256 text,
    p_autenticacion_verificada_en timestamptz,
    p_sesion_emitida_en timestamptz,
    p_asercion_expira_en timestamptz,
    p_politica_garantia_ref text,
    p_politica_garantia_huella_sha256 text
)
RETURNS TABLE(
    autenticacion_ref text,
    asercion_ref text,
    sesion_ref text,
    control_sesion_ref text,
    control_sesion_revision_texto text,
    control_sesion_estado text,
    control_sesion_huella_sha256 text,
    cuenta_ref text,
    cuenta_ordinaria_ref text,
    sesion_revalidada_en timestamptz,
    sesion_valida_hasta timestamptz
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT sesion.autenticacion_ref, sesion.asercion_ref, sesion.sesion_ref,
           control.control_sesion_ref, control.revision::text,
           control.estado, control.huella_sha256, sesion.cuenta_ref,
           sesion.cuenta_ordinaria_ref, control.sesion_revalidada_en,
           control.sesion_valida_hasta
      FROM vec_identidad_sesiones_v1.consumo_asercion AS consumo
      JOIN vec_autorizacion.sesion_autenticacion_v1 AS sesion
        ON sesion.sesion_ref = consumo.sesion_ref
       AND sesion.autenticacion_ref = consumo.autenticacion_ref
       AND sesion.autenticacion_huella_sha256 =
           consumo.autenticacion_huella_sha256
       AND sesion.cuenta_ref = consumo.cuenta_ref
       AND sesion.cuenta_ordinaria_ref = consumo.cuenta_ordinaria_ref
      JOIN vec_autorizacion.control_sesion_v1 AS control
        ON control.sesion_ref = consumo.sesion_ref
       AND control.control_sesion_ref = consumo.control_sesion_ref
       AND control.revision = consumo.control_sesion_revision
     WHERE consumo.operacion_ref = p_operacion_ref
       AND consumo.esquema_hmac = p_esquema_hmac
       AND consumo.dominio_hmac_ref = p_dominio_hmac_ref
       AND consumo.clave_hmac_id = p_clave_hmac_id
       AND consumo.clave_hmac_version = p_clave_hmac_version
       AND consumo.asercion_id_hmac = p_asercion_id_hmac
       AND consumo.sesion_id_hmac = p_sesion_id_hmac
       AND consumo.sujeto_id_hmac = p_sujeto_id_hmac
       AND consumo.cuenta_id_hmac = p_cuenta_id_hmac
       AND consumo.cuenta_ordinaria_id_hmac IS NOT DISTINCT FROM
           p_cuenta_ordinaria_id_hmac
       AND sesion.cuenta_privilegiada = p_cuenta_privilegiada
       AND sesion.superficie = p_superficie
       AND sesion.metodo_observado = p_metodo_observado
       AND sesion.garantia_observada = p_garantia_observada
       AND sesion.autenticacion_huella_sha256 =
           p_autenticacion_huella_sha256
       AND sesion.autenticacion_verificada_en =
           p_autenticacion_verificada_en
       AND sesion.sesion_emitida_en = p_sesion_emitida_en
       AND control.sesion_valida_hasta = p_asercion_expira_en
       AND sesion.politica_garantia_ref = p_politica_garantia_ref
       AND sesion.politica_garantia_huella_sha256 =
           p_politica_garantia_huella_sha256
$funcion$;

CREATE FUNCTION vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(
    p_autenticacion_ref text,
    p_autenticacion_huella_sha256 text,
    p_asercion_ref text,
    p_sesion_ref text,
    p_cuenta_ref text,
    p_cuenta_ordinaria_ref text,
    p_cuenta_privilegiada boolean,
    p_superficie text,
    p_metodo_observado text,
    p_garantia_observada text,
    p_politica_garantia_ref text,
    p_politica_garantia_huella_sha256 text,
    p_autenticacion_verificada_en timestamptz,
    p_sesion_emitida_en timestamptz,
    p_control_sesion_ref text,
    p_control_sesion_revision_texto text,
    p_control_sesion_estado text,
    p_control_sesion_huella_sha256 text,
    p_sesion_revalidada_en timestamptz,
    p_sesion_valida_hasta timestamptz
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    revision_esperada numeric;
    sesion record;
    estado_bloqueado record;
    cuentas_activas integer := 0;
    ahora timestamptz(6);
BEGIN
    IF p_control_sesion_revision_texto !~ '^[1-9][0-9]{0,19}$'
       OR p_control_sesion_estado <> 'activa' THEN
        RETURN false;
    END IF;
    revision_esperada := p_control_sesion_revision_texto::numeric;
    IF revision_esperada > 18446744073709551615 THEN
        RETURN false;
    END IF;

    SELECT base.sesion_ref, base.cuenta_ref, base.cuenta_ordinaria_ref,
           base.cuenta_privilegiada, base.superficie,
           base.autenticacion_verificada_en, control.sesion_valida_hasta,
           consumo.cuenta_revision, consumo.cuenta_ordinaria_revision
      INTO sesion
      FROM vec_autorizacion.sesion_autenticacion_v1 AS base
      JOIN vec_identidad_sesiones_v1.consumo_asercion AS consumo
        ON consumo.sesion_ref = base.sesion_ref
       AND consumo.autenticacion_ref = base.autenticacion_ref
       AND consumo.autenticacion_huella_sha256 =
           base.autenticacion_huella_sha256
       AND consumo.cuenta_ref = base.cuenta_ref
       AND consumo.cuenta_ordinaria_ref = base.cuenta_ordinaria_ref
      JOIN vec_autorizacion.control_sesion_actual_v1 AS actual
        ON actual.sesion_ref = base.sesion_ref
      JOIN vec_autorizacion.control_sesion_v1 AS control
        ON control.sesion_ref = actual.sesion_ref
       AND control.control_sesion_ref = actual.control_sesion_ref
       AND control.revision = actual.revision
     WHERE base.autenticacion_ref = p_autenticacion_ref
       AND base.autenticacion_huella_sha256 =
           p_autenticacion_huella_sha256
       AND base.asercion_ref = p_asercion_ref
       AND base.sesion_ref = p_sesion_ref
       AND base.cuenta_ref = p_cuenta_ref
       AND base.cuenta_ordinaria_ref = p_cuenta_ordinaria_ref
       AND base.cuenta_privilegiada = p_cuenta_privilegiada
       AND base.superficie = p_superficie
       AND base.metodo_observado = p_metodo_observado
       AND base.garantia_observada = p_garantia_observada
       AND base.politica_garantia_ref = p_politica_garantia_ref
       AND base.politica_garantia_huella_sha256 =
           p_politica_garantia_huella_sha256
       AND base.autenticacion_verificada_en =
           p_autenticacion_verificada_en
       AND base.sesion_emitida_en = p_sesion_emitida_en
       AND control.control_sesion_ref = p_control_sesion_ref
       AND control.revision = revision_esperada
       AND control.estado = p_control_sesion_estado
       AND control.huella_sha256 = p_control_sesion_huella_sha256
       AND control.sesion_revalidada_en = p_sesion_revalidada_en
       AND control.sesion_valida_hasta = p_sesion_valida_hasta
       AND consumo.control_sesion_ref = control.control_sesion_ref
       AND consumo.control_sesion_revision = control.revision
     FOR UPDATE OF actual;
    IF NOT FOUND THEN
        RETURN false;
    END IF;

    FOR estado_bloqueado IN
        SELECT actual.cuenta_ref, actual.revision, estado.estado
          FROM vec_identidad_sesiones_v1.estado_cuenta_actual AS actual
          JOIN vec_identidad_sesiones_v1.estado_cuenta AS estado
            ON estado.cuenta_ref = actual.cuenta_ref
           AND estado.revision = actual.revision
         WHERE actual.cuenta_ref IN (
             sesion.cuenta_ref, sesion.cuenta_ordinaria_ref
         )
         ORDER BY actual.cuenta_ref COLLATE "C"
         FOR UPDATE OF actual
    LOOP
        IF estado_bloqueado.estado = 'activa'
           AND (
               (estado_bloqueado.cuenta_ref = sesion.cuenta_ref
                AND estado_bloqueado.revision = sesion.cuenta_revision)
               OR
               (estado_bloqueado.cuenta_ref = sesion.cuenta_ordinaria_ref
                AND estado_bloqueado.revision =
                    sesion.cuenta_ordinaria_revision)
           ) THEN
            cuentas_activas := cuentas_activas + 1;
        END IF;
    END LOOP;
    IF cuentas_activas <>
           (CASE WHEN sesion.cuenta_privilegiada THEN 2 ELSE 1 END) THEN
        RETURN false;
    END IF;

    ahora := clock_timestamp();
    RETURN ahora >= p_sesion_revalidada_en
       AND ahora < sesion.sesion_valida_hasta
       AND ahora < sesion.autenticacion_verificada_en + (
           CASE sesion.superficie
               WHEN 'externa_personal' THEN interval '12 hours'
               WHEN 'interna_corporativa' THEN interval '15 minutes'
               WHEN 'administracion_privilegiada' THEN interval '5 minutes'
               ELSE interval '0 seconds'
           END
       );
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR numeric_value_out_of_range OR cardinality_violation THEN
        RETURN false;
END
$funcion$;

CREATE FUNCTION vec_identidad_sesiones_v1.cambiar_estado_cuenta_v1(
    p_cuenta_ref text,
    p_revision_esperada_texto text,
    p_estado_nuevo text,
    p_operacion_ref text
)
RETURNS text
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    revision_esperada numeric;
    revision_nueva numeric;
    estado_anterior text;
    ahora timestamptz(6);
BEGIN
    IF vec_identidad_sesiones_v1.referencia_valida(
           p_cuenta_ref, 'cta_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           p_operacion_ref, 'opr_'
       ) IS NOT TRUE
       OR p_revision_esperada_texto !~ '^[1-9][0-9]{0,19}$'
       OR p_estado_nuevo NOT IN ('activa', 'inactiva') THEN
        RETURN NULL;
    END IF;
    revision_esperada := p_revision_esperada_texto::numeric;
    SELECT actual.revision, estado.estado
      INTO STRICT revision_nueva, estado_anterior
      FROM vec_identidad_sesiones_v1.estado_cuenta_actual AS actual
      JOIN vec_identidad_sesiones_v1.estado_cuenta AS estado
        ON estado.cuenta_ref = actual.cuenta_ref
       AND estado.revision = actual.revision
     WHERE actual.cuenta_ref = p_cuenta_ref
     FOR UPDATE OF actual;
    IF revision_nueva <> revision_esperada
       OR estado_anterior = p_estado_nuevo
       OR revision_nueva >= 18446744073709551615 THEN
        RETURN NULL;
    END IF;
    revision_nueva := revision_nueva + 1;
    ahora := clock_timestamp();
    INSERT INTO vec_identidad_sesiones_v1.estado_cuenta (
        cuenta_ref, revision, estado, registrada_en, acto_ref
    ) VALUES (
        p_cuenta_ref, revision_nueva, p_estado_nuevo, ahora, p_operacion_ref
    );
    UPDATE vec_identidad_sesiones_v1.estado_cuenta_actual
       SET revision = revision_nueva,
           actualizada_en = ahora,
           acto_ref = p_operacion_ref
     WHERE cuenta_ref = p_cuenta_ref;
    RETURN revision_nueva::text;
EXCEPTION
    WHEN no_data_found OR too_many_rows OR data_exception
        OR invalid_text_representation OR numeric_value_out_of_range
        OR unique_violation OR check_violation THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION vec_identidad_sesiones_v1.revocar_sesion_v1(
    p_sesion_ref text,
    p_control_sesion_ref text,
    p_revision_esperada_texto text,
    p_operacion_ref text
)
RETURNS text
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    actual record;
    revision_esperada numeric;
    revision_nueva numeric;
    ahora timestamptz(6);
    huella text;
BEGIN
    IF vec_identidad_sesiones_v1.referencia_valida(
           p_sesion_ref, 'ses_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           p_control_sesion_ref, 'cse_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           p_operacion_ref, 'opr_'
       ) IS NOT TRUE
       OR p_revision_esperada_texto !~ '^[1-9][0-9]{0,19}$' THEN
        RETURN NULL;
    END IF;
    revision_esperada := p_revision_esperada_texto::numeric;
    SELECT puntero.revision, control.estado,
           control.sesion_revalidada_en, control.sesion_valida_hasta
      INTO STRICT actual
      FROM vec_autorizacion.control_sesion_actual_v1 AS puntero
      JOIN vec_autorizacion.control_sesion_v1 AS control
        ON control.sesion_ref = puntero.sesion_ref
       AND control.control_sesion_ref = puntero.control_sesion_ref
       AND control.revision = puntero.revision
      JOIN vec_identidad_sesiones_v1.consumo_asercion AS consumo
        ON consumo.sesion_ref = puntero.sesion_ref
     WHERE puntero.sesion_ref = p_sesion_ref
       AND puntero.control_sesion_ref = p_control_sesion_ref
     FOR UPDATE OF puntero;
    ahora := clock_timestamp();
    IF actual.revision <> revision_esperada OR actual.estado <> 'activa'
       OR actual.revision >= 18446744073709551615
       OR ahora >= actual.sesion_valida_hasta THEN
        RETURN NULL;
    END IF;
    revision_nueva := actual.revision + 1;
    huella := vec_identidad_sesiones_v1.huella_control_sesion_v1(
        p_control_sesion_ref, revision_nueva, p_sesion_ref, 'revocada',
        ahora, actual.sesion_valida_hasta, p_operacion_ref
    );
    INSERT INTO vec_autorizacion.control_sesion_v1 (
        control_sesion_ref, revision, sesion_ref, estado, huella_sha256,
        sesion_revalidada_en, sesion_valida_hasta
    ) VALUES (
        p_control_sesion_ref, revision_nueva, p_sesion_ref, 'revocada',
        huella, ahora, actual.sesion_valida_hasta
    );
    UPDATE vec_autorizacion.control_sesion_actual_v1
       SET revision = revision_nueva,
           actualizada_en = ahora,
           acto_ref = p_operacion_ref
     WHERE sesion_ref = p_sesion_ref;
    RETURN revision_nueva::text;
EXCEPTION
    WHEN no_data_found OR too_many_rows OR data_exception
        OR invalid_text_representation OR numeric_value_out_of_range
        OR unique_violation OR check_violation THEN
        RETURN NULL;
END
$funcion$;

REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_identidad_sesiones_v1 FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1
    TO vec_identidad_sesiones_v1_provisionador,
       vec_identidad_sesiones_v1_registrador,
       vec_identidad_sesiones_v1_revalidador,
       vec_identidad_sesiones_v1_revocador;
GRANT EXECUTE ON FUNCTION
    vec_identidad_sesiones_v1.provisionar_cuenta_v1(
        text, text, text, text, bigint, bytea, bytea, boolean, bytea
    ) TO vec_identidad_sesiones_v1_provisionador;
GRANT EXECUTE ON FUNCTION
    vec_identidad_sesiones_v1.registrar_alias_hmac_cuenta_v1(
        text, text, text, text, text, bigint, bytea, bytea
    ) TO vec_identidad_sesiones_v1_provisionador;
GRANT EXECUTE ON FUNCTION
    vec_identidad_sesiones_v1.registrar_sesion_v1(
        text, text, text, text, bigint, bytea, bytea, bytea, bytea,
        bytea, boolean, text, text, text, text, timestamptz,
        timestamptz, timestamptz, text, text
    ) TO vec_identidad_sesiones_v1_registrador;
GRANT EXECUTE ON FUNCTION
    vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(
        text, text, text, text, bigint, bytea, bytea, bytea, bytea,
        bytea, boolean, text, text, text, text, timestamptz,
        timestamptz, timestamptz, text, text
    ) TO vec_identidad_sesiones_v1_registrador;
GRANT EXECUTE ON FUNCTION
    vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(
        text, text, text, text, text, text, boolean, text, text, text,
        text, text, timestamptz, timestamptz, text, text, text, text,
        timestamptz, timestamptz
    ) TO vec_identidad_sesiones_v1_revalidador;
GRANT EXECUTE ON FUNCTION
    vec_identidad_sesiones_v1.cambiar_estado_cuenta_v1(
        text, text, text, text
    ) TO vec_identidad_sesiones_v1_revocador;
GRANT EXECUTE ON FUNCTION
    vec_identidad_sesiones_v1.revocar_sesion_v1(
        text, text, text, text
    ) TO vec_identidad_sesiones_v1_revocador;
COMMIT;
