-- CT-000044A: causalidad de revocación y ayudantes privados de cursores.
--
-- Cada familia dispone de su propia fila mutable. Una revocación avanza esa
-- fila antes de insertarse y el lector de continuación la bloquea antes de
-- releer familia, cursor y revocación. Así existe un orden causal por familia
-- sin introducir un cerrojo común entre familias distintas.

CREATE TABLE
vec_contratacion_temporal.control_causal_familia_cursor_rrhh (
    familia_ref text PRIMARY KEY,
    familia_creada_en timestamptz(6) NOT NULL,
    revision numeric(20, 0) NOT NULL DEFAULT 0,
    actualizada_en timestamptz(6) NOT NULL,
    UNIQUE (familia_ref, familia_creada_en),
    FOREIGN KEY (familia_ref, familia_creada_en)
        REFERENCES vec_contratacion_temporal.familia_cursor_cuadro_rrhh(
            familia_ref, creada_en
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (familia_ref ~ '^familia:cursor:rrhh:[0-9a-f]{32}$'),
    -- CT-000038 hace familia_ref clave primaria de la revocación: una
    -- familia solo puede avanzar una vez desde viva a revocada.
    CHECK (revision BETWEEN 0 AND 1),
    CHECK (
        familia_creada_en =
            pg_catalog.date_trunc('microseconds', familia_creada_en)
        AND actualizada_en =
            pg_catalog.date_trunc('microseconds', actualizada_en)
        AND actualizada_en >= familia_creada_en
    )
);

-- El alta admite instalaciones que ya contengan familias de CT-000038.
-- Una familia ya revocada nace en revisión uno; nunca se reescribe historia.
INSERT INTO
vec_contratacion_temporal.control_causal_familia_cursor_rrhh (
    familia_ref, familia_creada_en, revision, actualizada_en
)
SELECT familia.familia_ref, familia.creada_en,
       CASE WHEN revocacion.familia_ref IS NULL THEN 0 ELSE 1 END,
       COALESCE(revocacion.revocada_en, familia.creada_en)
  FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh familia
  LEFT JOIN vec_contratacion_temporal.revocacion_familia_cursor_rrhh revocacion
    USING (familia_ref);

CREATE FUNCTION
vec_contratacion_temporal.validar_avance_control_causal_cursor_rrhh_v1()
RETURNS trigger
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
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR TG_OP <> 'UPDATE'
       OR TG_TABLE_SCHEMA <> 'vec_contratacion_temporal'
       OR TG_TABLE_NAME <> 'control_causal_familia_cursor_rrhh'
       OR NEW.familia_ref IS DISTINCT FROM OLD.familia_ref
       OR NEW.familia_creada_en IS DISTINCT FROM OLD.familia_creada_en
       OR NEW.revision IS DISTINCT FROM OLD.revision + 1
       OR NEW.actualizada_en < OLD.actualizada_en THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'avance causal de familia RRHH rechazado';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.avanzar_control_causal_revocacion_cursor_rrhh_v1()
RETURNS trigger
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
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR TG_OP <> 'INSERT'
       OR TG_TABLE_SCHEMA <> 'vec_contratacion_temporal'
       OR TG_TABLE_NAME <> 'revocacion_familia_cursor_rrhh' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'revocación causal de familia RRHH rechazada';
    END IF;

    UPDATE vec_contratacion_temporal.control_causal_familia_cursor_rrhh
       SET revision = revision + 1,
           actualizada_en = pg_catalog.date_trunc(
               'microseconds', pg_catalog.clock_timestamp()
           )
     WHERE familia_ref = NEW.familia_ref
       AND familia_creada_en = NEW.familia_creada_en
       AND revision = 0;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'punto de control causal de familia RRHH no disponible';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER validar_avance_causal_antes
BEFORE UPDATE
ON vec_contratacion_temporal.control_causal_familia_cursor_rrhh
FOR EACH ROW EXECUTE FUNCTION
vec_contratacion_temporal.validar_avance_control_causal_cursor_rrhh_v1();

CREATE TRIGGER control_causal_familia_cursor_rrhh_no_borrar
BEFORE DELETE
ON vec_contratacion_temporal.control_causal_familia_cursor_rrhh
FOR EACH ROW EXECUTE FUNCTION
vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE TRIGGER control_causal_familia_cursor_rrhh_no_truncar
BEFORE TRUNCATE
ON vec_contratacion_temporal.control_causal_familia_cursor_rrhh
FOR EACH STATEMENT EXECUTE FUNCTION
vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE TRIGGER avanzar_control_causal_revocacion_antes
BEFORE INSERT
ON vec_contratacion_temporal.revocacion_familia_cursor_rrhh
FOR EACH ROW EXECUTE FUNCTION
vec_contratacion_temporal.avanzar_control_causal_revocacion_cursor_rrhh_v1();

ALTER TABLE
    vec_contratacion_temporal.control_causal_familia_cursor_rrhh
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE
    vec_contratacion_temporal.control_causal_familia_cursor_rrhh
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total
ON vec_contratacion_temporal.control_causal_familia_cursor_rrhh
TO vec_contratacion_temporal_propietario
USING (true) WITH CHECK (true);

-- Resuelve el token de forma provisional sobre una tabla inmutable. Solo
-- después bloquea la fila causal de su familia y relee toda la ligadura.
CREATE FUNCTION
vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v1(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    p_actor_ref text,
    p_perfil_ref text,
    p_perfil_version numeric,
    p_sesion_ref text,
    p_sesion_huella_sha256 text
)
RETURNS vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1
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
    v_token_huella text;
    v_familia_ref text;
    v_filtros_huella text;
    v_ahora timestamptz(6);
    v_control record;
    v_ligadura record;
    v_corte numeric(20, 0);
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_isolation')
          <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR p_actor_ref IS NULL
       OR p_actor_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_perfil_ref IS NULL
       OR p_perfil_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_perfil_version IS NULL
       OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_perfil_version <> pg_catalog.trunc(p_perfil_version)
       OR p_sesion_ref IS NULL
       OR p_sesion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_sesion_huella_sha256 IS NULL
       OR p_sesion_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_sesion_huella_sha256 = pg_catalog.repeat('0', 64) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'resolución de cursor RRHH rechazada';
    END IF;
    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(p_alcance);
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
        p_consulta
    );

    IF p_consulta.cursor = '' THEN
        SELECT ultimo_corte
          INTO STRICT v_corte
          FROM vec_contratacion_temporal.control_publicacion_rrhh
         WHERE control;
        RETURN ROW(
            false, NULL, v_corte, 0, NULL, NULL, NULL, NULL, NULL,
            NULL, NULL
        )::vec_contratacion_temporal
             .estado_cursor_entrada_cuadro_rrhh_v1;
    END IF;

    v_token_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(p_consulta.cursor, 'UTF8')
    ), 'hex');
    SELECT cursor.familia_ref
      INTO STRICT v_familia_ref
      FROM vec_contratacion_temporal.cursor_cuadro_rrhh cursor
     WHERE cursor.token_huella_sha256 = v_token_huella;

    SELECT causal.revision, causal.familia_creada_en
      INTO STRICT v_control
      FROM vec_contratacion_temporal
           .control_causal_familia_cursor_rrhh causal
     WHERE causal.familia_ref = v_familia_ref
     FOR UPDATE;

    SELECT familia.*, cursor.token_huella_sha256,
           cursor.pagina, cursor.emitida_en AS cursor_emitida_en,
           cursor.acceso_emision_ref, cursor.ultimo_actualizado_en,
           cursor.ultimo_expediente_ref
      INTO STRICT v_ligadura
      FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh familia
      JOIN vec_contratacion_temporal.cursor_cuadro_rrhh cursor
        USING (familia_ref)
     WHERE familia.familia_ref = v_familia_ref
       AND cursor.token_huella_sha256 = v_token_huella
     FOR SHARE OF familia, cursor;

    v_filtros_huella := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.canon_familia_cuadro_rrhh_v1(p_consulta)
    ), 'hex');
    v_ahora := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    IF v_control.revision <> 0
       OR v_control.familia_creada_en IS DISTINCT FROM v_ligadura.creada_en
       OR v_ligadura.organizacion_ref IS DISTINCT FROM
          p_alcance.organizacion_ref
       OR v_ligadura.clase_ambito IS DISTINCT FROM p_alcance.clase_ambito
       OR v_ligadura.ambito_ref IS DISTINCT FROM p_alcance.ambito_ref
       OR v_ligadura.actor_ref IS DISTINCT FROM p_actor_ref
       OR v_ligadura.perfil_ref IS DISTINCT FROM p_perfil_ref
       OR v_ligadura.perfil_version IS DISTINCT FROM p_perfil_version
       OR v_ligadura.sesion_ref IS DISTINCT FROM p_sesion_ref
       OR v_ligadura.sesion_huella_sha256 IS DISTINCT FROM
          p_sesion_huella_sha256
       OR v_ligadura.filtros_huella_sha256 IS DISTINCT FROM
          v_filtros_huella
       OR v_ligadura.limite IS DISTINCT FROM p_consulta.limite
       OR v_ahora < v_ligadura.creada_en
       OR v_ahora >= v_ligadura.valida_hasta
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .revocacion_familia_cursor_rrhh revocacion
            WHERE revocacion.familia_ref = v_familia_ref
       )
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .consumo_cursor_cuadro_rrhh consumo
            WHERE consumo.token_huella_sha256 = v_token_huella
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'resolución de cursor RRHH rechazada';
    END IF;

    RETURN ROW(
        true, v_ligadura.familia_ref, v_ligadura.corte_global,
        v_ligadura.pagina, v_token_huella,
        v_ligadura.acceso_emision_ref, v_ligadura.cursor_emitida_en,
        v_ligadura.creada_en, v_ligadura.valida_hasta,
        v_ligadura.ultimo_actualizado_en,
        v_ligadura.ultimo_expediente_ref
    )::vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'resolución de cursor RRHH rechazada';
END
$funcion$;

-- Genera material opaco para el resultado y para los futuros efectos. El
-- token claro nunca se inserta: la persistencia recibirá solo su SHA-256.
CREATE FUNCTION
vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
    p_estado
        vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
    p_materializacion
        vec_contratacion_temporal.materializacion_cuadro_rrhh_v1
)
RETURNS vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1
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
    v_token text;
    v_token_huella text;
    v_familia_ref text;
    v_pagina numeric(20, 0);
    v_padre text;
    v_total integer;
    v_ultima vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
BEGIN
    v_total := pg_catalog.cardinality(p_materializacion.resumenes);
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_estado IS NULL
       OR p_materializacion IS NULL
       OR p_materializacion.resumenes IS NULL
       OR p_materializacion.hay_mas IS NULL
       OR v_total NOT BETWEEN 0 AND 100
       OR (
           v_total = 0
           AND pg_catalog.array_ndims(
               p_materializacion.resumenes
           ) IS NOT NULL
       )
       OR (
           v_total > 0
           AND (
               pg_catalog.array_ndims(
                   p_materializacion.resumenes
               ) IS DISTINCT FROM 1
               OR pg_catalog.array_lower(
                   p_materializacion.resumenes, 1
               ) IS DISTINCT FROM 1
               OR pg_catalog.array_upper(
                   p_materializacion.resumenes, 1
               ) IS DISTINCT FROM v_total
               OR EXISTS (
                   SELECT 1
                     FROM pg_catalog.unnest(
                         p_materializacion.resumenes
                     ) AS resumen
                    WHERE resumen IS NULL
               )
           )
       )
       OR p_estado.es_continuacion IS NULL
       OR p_estado.corte_global IS NULL
       OR p_estado.corte_global NOT BETWEEN 0 AND
          9007199254740991::numeric
       OR p_estado.corte_global <> pg_catalog.trunc(p_estado.corte_global)
       OR (
           NOT p_estado.es_continuacion
           AND (
               p_estado.pagina_presentada IS DISTINCT FROM 0
               OR p_estado.familia_ref IS NOT NULL
               OR p_estado.token_presentado_huella_sha256 IS NOT NULL
               OR p_estado.acceso_emision_ref IS NOT NULL
               OR p_estado.cursor_emitida_en IS NOT NULL
               OR p_estado.familia_creada_en IS NOT NULL
               OR p_estado.familia_valida_hasta IS NOT NULL
               OR p_estado.ultimo_actualizado_en IS NOT NULL
               OR p_estado.ultimo_expediente_ref IS NOT NULL
           )
       )
       OR (
           p_estado.es_continuacion
           AND (
               p_estado.corte_global < 1
               OR p_estado.pagina_presentada NOT BETWEEN 2 AND
                   9007199254740991::numeric
               OR p_estado.pagina_presentada <>
                  pg_catalog.trunc(p_estado.pagina_presentada)
               OR p_estado.familia_ref IS NULL
               OR p_estado.familia_ref !~
                  '^familia:cursor:rrhh:[0-9a-f]{32}$'
               OR p_estado.token_presentado_huella_sha256 IS NULL
               OR p_estado.token_presentado_huella_sha256 !~
                  '^[0-9a-f]{64}$'
               OR p_estado.token_presentado_huella_sha256 =
                  pg_catalog.repeat('0', 64)
               OR p_estado.acceso_emision_ref IS NULL
               OR p_estado.acceso_emision_ref !~
                  '^acceso:rrhh:[0-9a-f]{32}$'
               OR p_estado.cursor_emitida_en IS NULL
               OR p_estado.familia_creada_en IS NULL
               OR p_estado.familia_valida_hasta IS NULL
               OR p_estado.ultimo_actualizado_en IS NULL
               OR p_estado.ultimo_expediente_ref IS NULL
               OR p_estado.ultimo_expediente_ref !~
                  '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
               OR p_estado.cursor_emitida_en <>
                  pg_catalog.date_trunc(
                      'microseconds', p_estado.cursor_emitida_en
                  )
               OR p_estado.familia_creada_en <>
                  pg_catalog.date_trunc(
                      'microseconds', p_estado.familia_creada_en
                  )
               OR p_estado.familia_valida_hasta <>
                  pg_catalog.date_trunc(
                      'microseconds', p_estado.familia_valida_hasta
                  )
               OR p_estado.ultimo_actualizado_en <>
                  pg_catalog.date_trunc(
                      'microseconds', p_estado.ultimo_actualizado_en
                  )
           )
       )
       OR (
           v_total = 0
           AND (
               p_materializacion.hay_mas
               OR p_materializacion.ultimo_actualizado_en IS NOT NULL
               OR p_materializacion.ultimo_expediente_ref IS NOT NULL
           )
       )
       OR (
           v_total > 0
           AND (
               p_materializacion.ultimo_actualizado_en IS NULL
               OR p_materializacion.ultimo_expediente_ref IS NULL
           )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'preparación de cursor RRHH rechazada';
    END IF;

    IF v_total > 0 THEN
        v_ultima := p_materializacion.resumenes[v_total];
        IF v_ultima.actualizado_en IS DISTINCT FROM
               p_materializacion.ultimo_actualizado_en
           OR v_ultima.expediente_ref IS DISTINCT FROM
              p_materializacion.ultimo_expediente_ref THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'preparación de cursor RRHH rechazada';
        END IF;
    END IF;

    IF NOT p_materializacion.hay_mas THEN
        RETURN ROW(
            false, '', ''::bytea, NULL, 0, NULL, NULL, NULL, NULL
        )::vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1;
    END IF;
    IF p_estado.es_continuacion
       AND p_estado.pagina_presentada =
           9007199254740991::numeric THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'preparación de cursor RRHH rechazada';
    END IF;

    -- El SHA-256 de tres UUID independientes conserva 32 bytes de salida y
    -- más de 256 bits de entrada aleatoria aun descontando los bits fijados
    -- por UUIDv4. No exige permisos sobre un esquema de extensión.
    v_token := pg_catalog.rtrim(pg_catalog.translate(
        pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.uuid_send(pg_catalog.gen_random_uuid())
            || pg_catalog.uuid_send(pg_catalog.gen_random_uuid())
            || pg_catalog.uuid_send(pg_catalog.gen_random_uuid())
        ), 'base64'), '+/', '-_'
    ), E'=\n');
    IF pg_catalog.octet_length(v_token) <> 43
       OR v_token !~ '^[A-Za-z0-9_-]{43}$' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'preparación de cursor RRHH rechazada';
    END IF;
    v_token_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(v_token, 'UTF8')
    ), 'hex');

    IF p_estado.es_continuacion THEN
        v_familia_ref := p_estado.familia_ref;
        v_pagina := p_estado.pagina_presentada + 1;
        v_padre := p_estado.token_presentado_huella_sha256;
    ELSE
        v_familia_ref := 'familia:cursor:rrhh:'
            || pg_catalog.substr(pg_catalog.encode(pg_catalog.sha256(
                pg_catalog.uuid_send(pg_catalog.gen_random_uuid())
                || pg_catalog.uuid_send(pg_catalog.gen_random_uuid())
            ), 'hex'), 1, 32);
        v_pagina := 2;
        v_padre := NULL;
    END IF;

    RETURN ROW(
        true, v_token, pg_catalog.decode(v_token_huella, 'hex'),
        v_familia_ref, v_pagina, v_token_huella, v_padre,
        p_materializacion.ultimo_actualizado_en,
        p_materializacion.ultimo_expediente_ref
    )::vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'preparación de cursor RRHH rechazada';
END
$funcion$;

ALTER FUNCTION
vec_contratacion_temporal.validar_avance_control_causal_cursor_rrhh_v1()
OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION
vec_contratacion_temporal.avanzar_control_causal_revocacion_cursor_rrhh_v1()
OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION
vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    text, text, numeric, text, text
) OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION
vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
    vec_contratacion_temporal.materializacion_cuadro_rrhh_v1
) OWNER TO vec_contratacion_temporal_propietario;

REVOKE ALL ON TABLE
vec_contratacion_temporal.control_causal_familia_cursor_rrhh
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_consultor_rrhh,
    vec_contratacion_temporal_lector_resultado_cobertura;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.validar_avance_control_causal_cursor_rrhh_v1(),
vec_contratacion_temporal.avanzar_control_causal_revocacion_cursor_rrhh_v1(),
vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    text, text, numeric, text, text
),
vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
    vec_contratacion_temporal.materializacion_cuadro_rrhh_v1
)
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_consultor_rrhh,
    vec_contratacion_temporal_lector_resultado_cobertura;

COMMENT ON TABLE
vec_contratacion_temporal.control_causal_familia_cursor_rrhh IS
'Punto de control causal privado y mutable de una única familia de cursores RRHH.';
COMMENT ON FUNCTION
vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    text, text, numeric, text, text
) IS
'Bloquea una familia, relee su cursor y rechaza consumo, caducidad o revocación.';
COMMENT ON FUNCTION
vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
    vec_contratacion_temporal.materializacion_cuadro_rrhh_v1
) IS
'Prepara token opaco transitorio y huellas; nunca persiste el token en claro.';
