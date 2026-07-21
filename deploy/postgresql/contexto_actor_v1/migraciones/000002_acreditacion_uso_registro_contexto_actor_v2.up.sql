-- Acreditacion cerrada de un recibo ContextoActor V2 para composicion local.
-- No concede privilegios a consumidores: la futura migracion del consumidor
-- debe hacerlo de forma nominal y solo si ambos modulos viven en esta base.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:migracion:acreditacion_uso:v2', 0
    )
);

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'acreditacion de uso ContextoActor V2 requiere migracion superusuario';
    END IF;
    IF pg_catalog.to_regclass(
           'vec_contexto_actor_v1.registros_contexto'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contexto_actor_v1.vinculo_referencia_actual'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta la migracion base de ContextoActor V1';
    END IF;
    IF pg_catalog.to_regclass(
           'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'la acreditacion de uso ContextoActor V2 ya existe';
    END IF;
END
$prevalidacion$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

-- Cerrojo MVCC contra inserciones fantasma. La fila no replica ningun dato de
-- negocio: versiona exclusivamente el conjunto de punteros actuales. Todo DML
-- de punteros toma primero el advisory global en BEFORE STATEMENT, muta sus
-- filas y solo entonces avanza esta generacion en AFTER STATEMENT. TRUNCATE se
-- rechaza en un trigger independiente que nunca espera el advisory: PostgreSQL
-- toma AccessExclusive antes de ejecutar triggers y mezclar ambos mecanismos
-- formaria un ciclo de locks con una acreditacion concurrente. La
-- acreditacion usa el orden simetrico advisory compartido -> filas ->
-- generacion FOR SHARE. La fila es la prueba MVCC; el advisory solo impone un
-- orden libre de interbloqueos entre lectores y escritores multi-sentencia.
CREATE TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    generacion numeric NOT NULL CHECK (
        generacion >= 0 AND pg_catalog.scale(generacion) = 0
    ),
    actualizada_en timestamptz NOT NULL CHECK (
        vec_contexto_actor_v1.instante_valido(actualizada_en)
    )
);
INSERT INTO vec_contexto_actor_v1.control_generacion_punteros_actuales_v2(
    control_id, generacion, actualizada_en
) VALUES (true, 0, pg_catalog.clock_timestamp());

CREATE FUNCTION vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
BEGIN
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_contexto_actor_v1:mutacion_punteros_actuales:v2', 0
        )
    );
    RETURN NULL;
END
$funcion$;

CREATE FUNCTION vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
BEGIN
    UPDATE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
       SET generacion = generacion + 1,
           actualizada_en = pg_catalog.clock_timestamp()
     WHERE control_id = true;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta el cerrojo MVCC de punteros ContextoActor V2';
    END IF;
    RETURN NULL;
END
$funcion$;

DO $triggers_punteros$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'proyeccion_cuenta_actual', 'persona_actual', 'perfil_actual',
        'vinculo_contexto_actual', 'vinculo_referencia_actual'
    ] LOOP
        EXECUTE pg_catalog.format(
          'CREATE TRIGGER puntero_actual_no_truncable_v2 BEFORE TRUNCATE ON vec_contexto_actor_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado()',
          tabla
        );
        EXECUTE pg_catalog.format(
          'CREATE TRIGGER serializar_mutacion_punteros_actuales_v2 BEFORE INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()',
          tabla
        );
        EXECUTE pg_catalog.format(
          'CREATE TRIGGER avanzar_generacion_punteros_actuales_v2 AFTER INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()',
          tabla
        );
    END LOOP;
END
$triggers_punteros$;

REVOKE ALL ON TABLE
    vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
    FROM PUBLIC, vec_contexto_actor_v1_runtime;
REVOKE ALL ON TYPE
    vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
    FROM PUBLIC, vec_contexto_actor_v1_runtime;
REVOKE ALL ON FUNCTION
    vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()
    FROM PUBLIC, vec_contexto_actor_v1_runtime;
REVOKE ALL ON FUNCTION
    vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()
    FROM PUBLIC, vec_contexto_actor_v1_runtime;

CREATE FUNCTION vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
    p_registro_contexto_ref text,
    p_contexto_actor_esquema text,
    p_contexto_actor_huella_sha256 text,
    p_manifiesto_procedencia_huella_sha256 text,
    p_autoridad_efectiva text,
    p_cuenta_ref text,
    p_cuenta_version numeric,
    p_persona_ref text,
    p_persona_version numeric,
    p_perfil_ref text,
    p_perfil_version numeric,
    p_contexto_actor_ref text,
    p_contexto_actor_version numeric,
    p_metodo text,
    p_garantia text,
    p_emitida_en timestamptz,
    p_valida_hasta timestamptz
) RETURNS timestamptz
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    registro record;
    cuenta record;
    perfil record;
    persona record;
    contexto record;
    generacion_observada numeric;
    ahora timestamptz;
    coincidencias integer;
    numero_vinculos integer;
    tipos integer;
    referencias integer;
    vinculos_texto text;
    vinculos_procedencia_texto text;
    representacion_texto text;
    representacion_reconstruida bytea;
    manifiesto_texto text;
    manifiesto_reconstruido bytea;
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25000',
            MESSAGE = 'acreditacion de uso ContextoActor V2 requiere SERIALIZABLE de escritura';
    END IF;

    IF vec_contexto_actor_v1.referencia_operacion_valida(
           p_registro_contexto_ref, 'rca_'
       ) IS NOT TRUE
       OR p_contexto_actor_esquema IS DISTINCT FROM
          'vec.contexto-actor.vinculado.v2'
       OR p_contexto_actor_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_manifiesto_procedencia_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_autoridad_efectiva IS DISTINCT FROM
          'autoridad_maestra_acreditada'
       OR vec_contexto_actor_v1.referencia_valida(
           p_cuenta_ref, 'cta_'
       ) IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(
           p_persona_ref, 'per_'
       ) IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(
           p_perfil_ref, 'prf_'
       ) IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(
           p_contexto_actor_ref, 'vca_'
       ) IS NOT TRUE
       OR p_cuenta_version IS NULL OR pg_catalog.scale(p_cuenta_version) <> 0
       OR p_cuenta_version NOT BETWEEN 1 AND 18446744073709551615::numeric
       OR p_persona_version IS NULL OR pg_catalog.scale(p_persona_version) <> 0
       OR p_persona_version NOT BETWEEN 1 AND 18446744073709551615::numeric
       OR p_perfil_version IS NULL OR pg_catalog.scale(p_perfil_version) <> 0
       OR p_perfil_version NOT BETWEEN 1 AND 18446744073709551615::numeric
       OR p_contexto_actor_version IS NULL
       OR pg_catalog.scale(p_contexto_actor_version) <> 0
       OR p_contexto_actor_version NOT BETWEEN
          1 AND 18446744073709551615::numeric
       OR p_metodo NOT IN (
           'certificado', 'dnie', 'sso', 'clave', 'kerberos_ad', 'demo'
       )
       OR p_garantia NOT IN ('bajo', 'sustancial', 'alto')
       OR vec_contexto_actor_v1.instante_valido(p_emitida_en) IS NOT TRUE
       OR vec_contexto_actor_v1.instante_valido(p_valida_hasta) IS NOT TRUE
       OR p_valida_hasta <= p_emitida_en THEN
        RETURN NULL;
    END IF;

    SELECT r.operacion_ref, r.registro_contexto_ref, r.cuenta_ref,
           r.perfil_ref, r.metodo, r.garantia, r.solicitado_en,
           r.resuelto_en, r.representacion_canonica, r.huella_sha256,
           r.manifiesto_procedencia_canonico,
           r.manifiesto_procedencia_huella_sha256,
           r.autoridad_efectiva
      INTO registro
      FROM vec_contexto_actor_v1.registros_contexto AS r
     WHERE r.registro_contexto_ref = p_registro_contexto_ref
     FOR SHARE OF r;
    IF NOT FOUND
       OR registro.cuenta_ref IS DISTINCT FROM p_cuenta_ref
       OR registro.perfil_ref IS DISTINCT FROM p_perfil_ref
       OR registro.metodo IS DISTINCT FROM p_metodo
       OR registro.garantia IS DISTINCT FROM p_garantia
       OR registro.huella_sha256 IS DISTINCT FROM
          p_contexto_actor_huella_sha256
       OR registro.manifiesto_procedencia_huella_sha256 IS DISTINCT FROM
          p_manifiesto_procedencia_huella_sha256
       OR registro.autoridad_efectiva IS DISTINCT FROM p_autoridad_efectiva
       OR pg_catalog.encode(
           pg_catalog.sha256(registro.representacion_canonica), 'hex'
       ) IS DISTINCT FROM registro.huella_sha256
       OR pg_catalog.encode(
           pg_catalog.sha256(registro.manifiesto_procedencia_canonico), 'hex'
       ) IS DISTINCT FROM registro.manifiesto_procedencia_huella_sha256
       OR registro.resuelto_en > p_emitida_en THEN
        RETURN NULL;
    END IF;

    -- El advisory global hace que todo mutador entre por su BEFORE STATEMENT
    -- antes de tocar filas. La seguridad frente a snapshots obsoletos no
    -- depende de este lock: la acredita la fila MVCC leida mas abajo.
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_contexto_actor_v1:mutacion_punteros_actuales:v2', 0
        )
    );

    -- Orden identico al resolutor: cuenta, perfil, candidatos de contexto,
    -- personas y referencias de modulo. Se toma el reloj solo al final.
    SELECT v.version, v.procedencia_ref, v.procedencia_version,
           v.procedencia_huella_sha256, v.procedencia_autoridad,
           v.estado, v.vigente_desde, v.vigente_hasta
      INTO cuenta
      FROM vec_contexto_actor_v1.proyeccion_cuenta_actual AS a
      JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones AS v
        USING (cuenta_ref, version)
     WHERE a.cuenta_ref = p_cuenta_ref
     FOR UPDATE OF a;
    IF NOT FOUND THEN RETURN NULL; END IF;

    SELECT v.version, v.persona_ref, v.procedencia_ref,
           v.procedencia_version, v.procedencia_huella_sha256,
           v.procedencia_autoridad, v.estado, v.vigente_desde,
           v.vigente_hasta
      INTO perfil
      FROM vec_contexto_actor_v1.perfil_actual AS a
      JOIN vec_contexto_actor_v1.perfil_versiones AS v
        USING (perfil_ref, version)
     WHERE a.perfil_ref = p_perfil_ref
     FOR UPDATE OF a;
    IF NOT FOUND THEN RETURN NULL; END IF;

    PERFORM 1
      FROM vec_contexto_actor_v1.vinculo_contexto_actual AS a
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones AS v
        USING (vinculo_ref, version)
     WHERE v.cuenta_ref = p_cuenta_ref AND v.perfil_ref = p_perfil_ref
     ORDER BY a.vinculo_ref
     FOR UPDATE OF a;
    GET DIAGNOSTICS coincidencias = ROW_COUNT;
    IF coincidencias <> 1 THEN RETURN NULL; END IF;

    SELECT v.vinculo_ref, v.version, v.cuenta_ref, v.perfil_ref,
           v.persona_ref, v.procedencia_ref, v.procedencia_version,
           v.procedencia_huella_sha256, v.procedencia_autoridad,
           v.estado, v.vigente_desde, v.vigente_hasta
      INTO contexto
      FROM vec_contexto_actor_v1.vinculo_contexto_actual AS a
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones AS v
        USING (vinculo_ref, version)
     WHERE v.cuenta_ref = p_cuenta_ref AND v.perfil_ref = p_perfil_ref;

    SELECT v.version, v.procedencia_ref, v.procedencia_version,
           v.procedencia_huella_sha256, v.procedencia_autoridad,
           v.estado, v.vigente_desde, v.vigente_hasta
      INTO persona
      FROM vec_contexto_actor_v1.persona_actual AS a
      JOIN vec_contexto_actor_v1.persona_versiones AS v
        USING (persona_ref, version)
     WHERE a.persona_ref = p_persona_ref
     FOR UPDATE OF a;
    IF NOT FOUND THEN RETURN NULL; END IF;

    PERFORM 1
      FROM vec_contexto_actor_v1.vinculo_referencia_actual AS a
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones AS v
        USING (vinculo_ref, version)
     WHERE v.persona_ref = p_persona_ref
     ORDER BY a.vinculo_ref
     FOR UPDATE OF a;

    -- Debe ocurrir despues de bloquear y releer todos los punteros. Si una
    -- mutacion comprometio despues del snapshot SERIALIZABLE, FOR SHARE no
    -- puede bloquear la version nueva invisible y PostgreSQL fuerza 40001.
    -- Si la acreditacion obtuvo primero el advisory, el mutador espera y queda
    -- serializado despues de su COMMIT.
    SELECT generacion
      INTO STRICT generacion_observada
      FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     WHERE control_id = true
     FOR SHARE;

    ahora := pg_catalog.clock_timestamp();

    IF cuenta.version IS DISTINCT FROM p_cuenta_version
       OR perfil.version IS DISTINCT FROM p_perfil_version
       OR perfil.persona_ref IS DISTINCT FROM p_persona_ref
       OR persona.version IS DISTINCT FROM p_persona_version
       OR contexto.vinculo_ref IS DISTINCT FROM p_contexto_actor_ref
       OR contexto.version IS DISTINCT FROM p_contexto_actor_version
       OR contexto.persona_ref IS DISTINCT FROM p_persona_ref
       OR cuenta.estado <> 'activo' OR perfil.estado <> 'activo'
       OR persona.estado <> 'activo' OR contexto.estado <> 'activo'
       OR cuenta.procedencia_autoridad <> p_autoridad_efectiva
       OR perfil.procedencia_autoridad <> p_autoridad_efectiva
       OR persona.procedencia_autoridad <> p_autoridad_efectiva
       OR contexto.procedencia_autoridad <> p_autoridad_efectiva
       OR p_emitida_en < cuenta.vigente_desde
       OR p_valida_hasta > cuenta.vigente_hasta
       OR ahora < cuenta.vigente_desde OR ahora >= cuenta.vigente_hasta
       OR p_emitida_en < perfil.vigente_desde
       OR p_valida_hasta > perfil.vigente_hasta
       OR ahora < perfil.vigente_desde OR ahora >= perfil.vigente_hasta
       OR p_emitida_en < persona.vigente_desde
       OR p_valida_hasta > persona.vigente_hasta
       OR ahora < persona.vigente_desde OR ahora >= persona.vigente_hasta
       OR p_emitida_en < contexto.vigente_desde
       OR p_valida_hasta > contexto.vigente_hasta
       OR ahora < contexto.vigente_desde OR ahora >= contexto.vigente_hasta
       OR ahora < p_emitida_en OR ahora >= p_valida_hasta THEN
        RETURN NULL;
    END IF;

    SELECT pg_catalog.count(*), pg_catalog.count(DISTINCT v.tipo),
           pg_catalog.count(DISTINCT (v.tipo, v.referencia)),
           pg_catalog.string_agg(pg_catalog.format(
             '{"vinculo_ref":%s,"version":%s,"tipo":%s,"referencia":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s}',
             pg_catalog.to_json(v.vinculo_ref)::text, v.version::text,
             pg_catalog.to_json(v.tipo)::text,
             pg_catalog.to_json(v.referencia)::text,
             pg_catalog.to_json(v.estado)::text,
             pg_catalog.to_json(pg_catalog.to_char(
                 v.vigente_desde AT TIME ZONE 'UTC',
                 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
             ))::text,
             pg_catalog.to_json(pg_catalog.to_char(
                 v.vigente_hasta AT TIME ZONE 'UTC',
                 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
             ))::text
           ), ',' ORDER BY v.tipo, v.referencia, v.version, v.vinculo_ref),
           pg_catalog.string_agg(pg_catalog.format(
             '{"vinculo_ref":%s,"version":%s,"tipo":%s,"referencia":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s}',
             pg_catalog.to_json(v.vinculo_ref)::text, v.version::text,
             pg_catalog.to_json(v.tipo)::text,
             pg_catalog.to_json(v.referencia)::text,
             pg_catalog.to_json(v.procedencia_ref)::text,
             v.procedencia_version::text,
             pg_catalog.to_json(v.procedencia_huella_sha256)::text,
             pg_catalog.to_json(v.procedencia_autoridad)::text
           ), ',' ORDER BY v.tipo, v.referencia, v.version, v.vinculo_ref)
      INTO numero_vinculos, tipos, referencias, vinculos_texto,
           vinculos_procedencia_texto
      FROM vec_contexto_actor_v1.vinculo_referencia_actual AS a
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones AS v
        USING (vinculo_ref, version)
     WHERE v.persona_ref = p_persona_ref;

    IF numero_vinculos > 128 OR tipos <> numero_vinculos
       OR referencias <> numero_vinculos
       OR EXISTS (
           SELECT 1
             FROM vec_contexto_actor_v1.vinculo_referencia_actual AS a
             JOIN vec_contexto_actor_v1.vinculo_referencia_versiones AS v
               USING (vinculo_ref, version)
            WHERE v.persona_ref = p_persona_ref
              AND (v.estado <> 'activo'
                   OR v.procedencia_autoridad <> p_autoridad_efectiva
                   OR p_emitida_en < v.vigente_desde
                   OR p_valida_hasta > v.vigente_hasta
                   OR ahora < v.vigente_desde OR ahora >= v.vigente_hasta)
       ) THEN
        RETURN NULL;
    END IF;

    manifiesto_texto := pg_catalog.format(
      '{"esquema":"vec.contexto-actor.procedencia-manifiesto.v1","autoridad_efectiva":"autoridad_maestra_acreditada","cuenta":{"cuenta_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"persona":{"persona_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"perfil":{"perfil_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"contexto":{"vinculo_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"vinculos":[%s]}',
      pg_catalog.to_json(p_cuenta_ref)::text, cuenta.version::text,
      pg_catalog.to_json(cuenta.procedencia_ref)::text,
      cuenta.procedencia_version::text,
      pg_catalog.to_json(cuenta.procedencia_huella_sha256)::text,
      pg_catalog.to_json(cuenta.procedencia_autoridad)::text,
      pg_catalog.to_json(p_persona_ref)::text, persona.version::text,
      pg_catalog.to_json(persona.procedencia_ref)::text,
      persona.procedencia_version::text,
      pg_catalog.to_json(persona.procedencia_huella_sha256)::text,
      pg_catalog.to_json(persona.procedencia_autoridad)::text,
      pg_catalog.to_json(p_perfil_ref)::text, perfil.version::text,
      pg_catalog.to_json(perfil.procedencia_ref)::text,
      perfil.procedencia_version::text,
      pg_catalog.to_json(perfil.procedencia_huella_sha256)::text,
      pg_catalog.to_json(perfil.procedencia_autoridad)::text,
      pg_catalog.to_json(contexto.vinculo_ref)::text,
      contexto.version::text,
      pg_catalog.to_json(contexto.procedencia_ref)::text,
      contexto.procedencia_version::text,
      pg_catalog.to_json(contexto.procedencia_huella_sha256)::text,
      pg_catalog.to_json(contexto.procedencia_autoridad)::text,
      coalesce(vinculos_procedencia_texto, '')
    );
    manifiesto_reconstruido := pg_catalog.convert_to(
        manifiesto_texto, 'UTF8'
    );

    representacion_texto := pg_catalog.format(
      '{"esquema":"vec.contexto-actor.vinculado.v2","principal_ref":%s,"metodo":%s,"garantia":%s,"perfil_activo_ref":%s,"persona_ref":%s,"contexto_actor_ref":%s,"contexto_version":%s,"cuenta_ref":%s,"cuenta_version":%s,"persona_version":%s,"perfil_version":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s,"resuelto_en":%s,"vinculos":[%s]}',
      pg_catalog.to_json(p_persona_ref)::text,
      pg_catalog.to_json(p_metodo)::text,
      pg_catalog.to_json(p_garantia)::text,
      pg_catalog.to_json(p_perfil_ref)::text,
      pg_catalog.to_json(p_persona_ref)::text,
      pg_catalog.to_json(contexto.vinculo_ref)::text,
      contexto.version::text,
      pg_catalog.to_json(p_cuenta_ref)::text,
      cuenta.version::text, persona.version::text, perfil.version::text,
      pg_catalog.to_json(contexto.estado)::text,
      pg_catalog.to_json(pg_catalog.to_char(
          contexto.vigente_desde AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
      ))::text,
      pg_catalog.to_json(pg_catalog.to_char(
          contexto.vigente_hasta AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
      ))::text,
      pg_catalog.to_json(pg_catalog.to_char(
          registro.resuelto_en AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
      ))::text,
      coalesce(vinculos_texto, '')
    );
    representacion_reconstruida := pg_catalog.convert_to(
        representacion_texto, 'UTF8'
    );

    IF representacion_reconstruida IS DISTINCT FROM
          registro.representacion_canonica
       OR manifiesto_reconstruido IS DISTINCT FROM
          registro.manifiesto_procedencia_canonico
       OR pg_catalog.encode(
           pg_catalog.sha256(representacion_reconstruida), 'hex'
       ) IS DISTINCT FROM p_contexto_actor_huella_sha256
       OR pg_catalog.encode(
           pg_catalog.sha256(manifiesto_reconstruido), 'hex'
       ) IS DISTINCT FROM p_manifiesto_procedencia_huella_sha256 THEN
        RETURN NULL;
    END IF;

    RETURN ahora;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN NULL;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
        text, text, text, text, text, text, numeric, text, numeric,
        text, numeric, text, numeric, text, text, timestamptz, timestamptz
    ) FROM PUBLIC, vec_contexto_actor_v1_runtime;

COMMENT ON FUNCTION
    vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
        text, text, text, text, text, text, numeric, text, numeric,
        text, numeric, text, numeric, text, text, timestamptz, timestamptz
    ) IS
    'Acredita un rca_ V2 contra bytes, procedencia y punteros actuales; devuelve solo el instante autoritativo y no concede acceso a tablas.';

COMMIT;
