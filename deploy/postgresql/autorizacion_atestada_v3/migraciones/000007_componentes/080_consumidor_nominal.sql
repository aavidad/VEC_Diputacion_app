-- Consumidor privado, nominal y de un solo uso de fuente corporativa V1.

CREATE FUNCTION vec_autorizacion_atestada_v3
    .consumir_fuente_corporativa_contexto_actor_v1_atestada(
        p_audiencia_consumo_esperada pg_catalog.text,
        p_accion_esperada pg_catalog.text,
        p_tipo_efecto_esperado pg_catalog.text,
        p_operacion_ref_esperada pg_catalog.text,
        p_efecto_ref_esperada pg_catalog.text,
        p_huella_efecto_sha256_esperada pg_catalog.text,
        p_capacidad_canonica pg_catalog.bytea,
        p_manifiesto_fuente_canonico pg_catalog.bytea,
        p_sobre_cose_sign1 pg_catalog.bytea,
        p_evidencia_verificacion pg_catalog.bytea,
        p_raiz_publica_spki pg_catalog.bytea
    )
RETURNS TABLE (
    capacidad_ref pg_catalog.text,
    fuente_ref pg_catalog.text,
    fuente_version pg_catalog.numeric,
    evento_fuente_ref pg_catalog.text,
    huella_evento_fuente_sha256 pg_catalog.text,
    huella_manifiesto_fuente_sha256 pg_catalog.text,
    operacion_ref pg_catalog.text,
    efecto_ref pg_catalog.text,
    huella_efecto_sha256 pg_catalog.text,
    consumo_huella_sha256 pg_catalog.text,
    consumida_en pg_catalog.timestamptz,
    consumo_nuevo pg_catalog.bool
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_a_capacidad pg_catalog.text[];
    v_a_evento pg_catalog.text[];
    v_a_fuente pg_catalog.text[];
    v_a_version pg_catalog.numeric[];
    v_n_acreditada_en pg_catalog.timestamptz;
    v_n_capacidad pg_catalog.text;
    v_n_efecto pg_catalog.text;
    v_n_evento pg_catalog.text;
    v_n_evento_emitido_en pg_catalog.timestamptz;
    v_n_fuente pg_catalog.text;
    v_n_huella_efecto pg_catalog.text;
    v_n_huella_evento pg_catalog.text;
    v_n_huella_manifiesto pg_catalog.text;
    v_n_nonce pg_catalog.text;
    v_n_operacion pg_catalog.text;
    v_n_version pg_catalog.numeric;
    v_c_canon pg_catalog.bytea[];
    v_c_capacidad pg_catalog.text[];
    v_c_huella pg_catalog.text[];
    v_c_instante pg_catalog.timestamptz[];
    v_c_nonce pg_catalog.text[];
    v_c_operacion pg_catalog.text[];
    v_canon_esperado pg_catalog.bytea[];
    v_cantidad pg_catalog.int8;
    v_cantidad_a_capacidad pg_catalog.int8;
    v_cantidad_a_evento pg_catalog.int8;
    v_cantidad_c_capacidad pg_catalog.int8;
    v_cantidad_c_nonce pg_catalog.int8;
    v_cantidad_c_operacion pg_catalog.int8;
    v_capacidad pg_catalog.json;
    v_capacidad_ref pg_catalog.text;
    v_checkpoint pg_catalog.int8;
    v_consumo pg_catalog.json;
    v_dba pg_catalog.oid;
    v_grupo_despachador pg_catalog.oid;
    v_grupo_esperado pg_catalog.oid;
    v_grupo_esperado_nombre pg_catalog.text;
    v_grupo_publicador pg_catalog.oid;
    v_grupo_revocador pg_catalog.oid;
    v_grupos pg_catalog.oid[];
    v_huella_esperada pg_catalog.text[];
    v_login pg_catalog.oid;
    v_lock pg_catalog.int8;
    v_manifiesto pg_catalog.json;
    v_timeout_inactivo pg_catalog.numeric;
    v_timeout_sentencia pg_catalog.numeric;
    v_timeout_transaccion pg_catalog.numeric;
BEGIN
    IF CURRENT_USER IS DISTINCT FROM
           'vec_autorizacion_atestada_v3_propietario'
       OR pg_catalog.current_setting('role') IS DISTINCT FROM 'none' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'identidad tecnica de fuente rechazada';
    END IF;

    v_grupo_esperado_nombre := CASE p_audiencia_consumo_esperada
        WHEN
          'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1'
        THEN 'vec_contexto_actor_v1_publicador_corporativo'
        WHEN 'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1'
        THEN 'vec_contexto_actor_v1_publicador_corporativo'
        WHEN
          'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1'
        THEN 'vec_contexto_actor_v1_revocador_corporativo'
        WHEN 'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1'
        THEN 'vec_contexto_actor_v1_revocador_corporativo'
        ELSE NULL
    END;

    SELECT
        (pg_catalog.array_agg(r.oid) FILTER (WHERE r.rolname =
            'vec_contexto_actor_v1_publicador_corporativo'))[1],
        (pg_catalog.array_agg(r.oid) FILTER (WHERE r.rolname =
            'vec_contexto_actor_v1_revocador_corporativo'))[1],
        (pg_catalog.array_agg(r.oid) FILTER (WHERE r.rolname =
            'vec_contexto_actor_v1_despachador_corporativo'))[1]
      INTO v_grupo_publicador, v_grupo_revocador, v_grupo_despachador
      FROM pg_catalog.pg_roles AS r
     WHERE r.rolname IN (
        'vec_contexto_actor_v1_publicador_corporativo',
        'vec_contexto_actor_v1_revocador_corporativo',
        'vec_contexto_actor_v1_despachador_corporativo'
     );
    v_grupos := ARRAY[
        v_grupo_publicador, v_grupo_revocador, v_grupo_despachador
    ];
    v_grupo_esperado := CASE v_grupo_esperado_nombre
        WHEN 'vec_contexto_actor_v1_publicador_corporativo'
        THEN v_grupo_publicador
        WHEN 'vec_contexto_actor_v1_revocador_corporativo'
        THEN v_grupo_revocador
        ELSE NULL
    END;

    SELECT d.datdba
      INTO v_dba
      FROM pg_catalog.pg_database AS d
      JOIN pg_catalog.pg_roles AS r
        ON r.oid = d.datdba AND r.rolsuper
     WHERE d.datname = pg_catalog.current_database();
    SELECT r.oid
      INTO v_login
      FROM pg_catalog.pg_roles AS r
     WHERE r.rolname = SESSION_USER
       AND r.rolcanlogin AND r.rolinherit
       AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole
       AND NOT r.rolreplication AND NOT r.rolbypassrls
       AND r.rolconnlimit = -1 AND r.rolvaliduntil IS NULL
       AND r.rolconfig IS NULL;

    IF v_grupo_esperado IS NULL OR v_dba IS NULL OR v_login IS NULL
       OR v_dba = v_login OR v_dba = ANY(v_grupos)
       OR EXISTS (
          SELECT 1 FROM pg_catalog.pg_roles AS r
           WHERE r.oid = ANY(v_grupos)
             AND (r.rolcanlogin OR r.rolinherit OR r.rolsuper
                  OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication
                  OR r.rolbypassrls OR r.rolconnlimit <> -1
                  OR r.rolvaliduntil IS NOT NULL OR r.rolconfig IS NOT NULL)
       )
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_roles AS r
            WHERE r.oid = ANY(v_grupos)) <> 3
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_db_role_setting AS s
                   WHERE s.setrole = ANY(v_grupos) OR s.setrole = v_login)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members AS m
                   WHERE m.member = ANY(v_grupos)
                      OR m.roleid = v_login OR m.grantor = v_login
                      OR m.grantor = ANY(v_grupos))
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members AS m
            WHERE m.member = v_login) <> 1
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members AS m
            WHERE m.member = v_login AND m.roleid = v_grupo_esperado
              AND m.grantor = v_dba AND NOT m.admin_option
              AND m.inherit_option AND NOT m.set_option) <> 1
       OR NOT pg_catalog.pg_has_role(v_login, v_grupo_esperado, 'MEMBER')
       OR NOT pg_catalog.pg_has_role(v_login, v_grupo_esperado, 'USAGE')
       OR pg_catalog.pg_has_role(v_login, v_grupo_esperado, 'SET')
       OR (v_grupo_esperado <> v_grupo_publicador AND
           pg_catalog.pg_has_role(v_login, v_grupo_publicador, 'USAGE'))
       OR (v_grupo_esperado <> v_grupo_revocador AND
           pg_catalog.pg_has_role(v_login, v_grupo_revocador, 'USAGE'))
       OR pg_catalog.pg_has_role(v_login, v_grupo_despachador, 'USAGE') THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'identidad tecnica de fuente rechazada';
    END IF;

    IF pg_catalog.current_setting('transaction_isolation') IS DISTINCT FROM
           'serializable'
       OR pg_catalog.current_setting('transaction_read_only') IS DISTINCT FROM
          'off'
       OR pg_catalog.current_setting('TimeZone') IS DISTINCT FROM 'UTC'
       OR pg_catalog.pg_is_in_recovery() THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'contexto privado de fuente rechazado';
    END IF;
    SELECT s.setting::pg_catalog.numeric INTO v_timeout_sentencia
      FROM pg_catalog.pg_settings AS s WHERE s.name = 'statement_timeout';
    SELECT s.setting::pg_catalog.numeric INTO v_timeout_transaccion
      FROM pg_catalog.pg_settings AS s WHERE s.name = 'transaction_timeout';
    SELECT s.setting::pg_catalog.numeric INTO v_timeout_inactivo
      FROM pg_catalog.pg_settings AS s
     WHERE s.name = 'idle_in_transaction_session_timeout';
    IF v_timeout_sentencia NOT BETWEEN 1000 AND 10000
       OR v_timeout_transaccion NOT BETWEEN 1000 AND 15000
       OR v_timeout_inactivo NOT BETWEEN 1000 AND 15000 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'limites privados de fuente rechazados';
    END IF;

    IF vec_autorizacion_atestada_v3
           .capacidad_fuente_bytes_validos(p_capacidad_canonica) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.manifiesto_fuente_bytes_validos(
              p_manifiesto_fuente_canonico) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.sobre_cose_sign1_fuente_bytes_validos(
              p_sobre_cose_sign1) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
              .evidencia_verificacion_fuente_bytes_validos(
                  p_evidencia_verificacion) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.raiz_publica_spki_fuente_bytes_validos(
              p_raiz_publica_spki) IS NOT TRUE
       OR pg_catalog.substr(p_raiz_publica_spki, 1, 12) <>
          pg_catalog.decode('302a300506032b6570032100', 'hex')
       OR vec_autorizacion_atestada_v3
              .operacion_ref_fuente_corporativa_valida(
                  p_operacion_ref_esperada) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
              .referencia_opaca_fuente_corporativa_valida(
                  p_efecto_ref_esperada) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.huella_sha256_valida(
              p_huella_efecto_sha256_esperada) IS NOT TRUE
       OR (CASE p_audiencia_consumo_esperada
           WHEN
             'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1'
           THEN (p_accion_esperada, p_tipo_efecto_esperado) =
                ('contexto_actor.organizacion_corporativa.publicar',
                 'organizacion_corporativa.alta')
           WHEN
             'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1'
           THEN (p_accion_esperada, p_tipo_efecto_esperado) =
                ('contexto_actor.organizacion_corporativa.revocar',
                 'organizacion_corporativa.revocacion')
           WHEN 'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1'
           THEN (p_accion_esperada, p_tipo_efecto_esperado) =
                ('contexto_actor.vinculo_corporativo.publicar',
                 'vinculo_corporativo.alta')
           WHEN 'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1'
           THEN (p_accion_esperada, p_tipo_efecto_esperado) =
                ('contexto_actor.vinculo_corporativo.revocar',
                 'vinculo_corporativo.revocacion')
           ELSE false END) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'entrada de fuente invalida';
    END IF;
    IF vec_autorizacion_atestada_v3
           .capacidad_fuente_corporativa_v1_canonica(p_capacidad_canonica)
          IS DISTINCT FROM p_capacidad_canonica
       OR vec_autorizacion_atestada_v3
              .manifiesto_fuente_corporativa_v1_canonico(
                  p_manifiesto_fuente_canonico)
          IS DISTINCT FROM p_manifiesto_fuente_canonico THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'canon de fuente invalido';
    END IF;

    v_capacidad := pg_catalog.convert_from(
        p_capacidad_canonica, 'UTF8')::pg_catalog.json;
    v_manifiesto := pg_catalog.convert_from(
        p_manifiesto_fuente_canonico, 'UTF8')::pg_catalog.json;
    v_capacidad_ref := 'cfc_' || pg_catalog.encode(
        pg_catalog.sha256(p_capacidad_canonica), 'hex');
    IF v_capacidad ->> 'audiencia_consumo' IS DISTINCT FROM
           p_audiencia_consumo_esperada
       OR v_capacidad ->> 'accion' IS DISTINCT FROM p_accion_esperada
       OR v_capacidad ->> 'tipo_efecto' IS DISTINCT FROM
          p_tipo_efecto_esperado
       OR v_capacidad ->> 'operacion_ref' IS DISTINCT FROM
          p_operacion_ref_esperada
       OR v_capacidad ->> 'efecto_ref' IS DISTINCT FROM p_efecto_ref_esperada
       OR v_capacidad ->> 'huella_efecto_sha256' IS DISTINCT FROM
          p_huella_efecto_sha256_esperada
       OR v_manifiesto ->> 'fuente_ref' IS DISTINCT FROM
          v_capacidad ->> 'fuente_ref'
       OR (v_manifiesto ->> 'fuente_version')::pg_catalog.numeric
          IS DISTINCT FROM
          (v_capacidad ->> 'fuente_version')::pg_catalog.numeric
       OR v_manifiesto ->> 'evento_fuente_ref' IS DISTINCT FROM
          v_capacidad ->> 'evento_fuente_ref'
       OR v_manifiesto ->> 'huella_evento_fuente_sha256' IS DISTINCT FROM
          v_capacidad ->> 'huella_evento_fuente_sha256'
       OR v_manifiesto ->> 'evento_fuente_emitido_en' IS DISTINCT FROM
          v_capacidad ->> 'evento_fuente_emitido_en'
       OR v_manifiesto ->> 'audiencia_consumo' IS DISTINCT FROM
          p_audiencia_consumo_esperada
       OR v_manifiesto ->> 'accion' IS DISTINCT FROM p_accion_esperada
       OR v_manifiesto ->> 'tipo_efecto' IS DISTINCT FROM
          p_tipo_efecto_esperado
       OR v_manifiesto ->> 'operacion_ref' IS DISTINCT FROM
          p_operacion_ref_esperada
       OR v_manifiesto ->> 'efecto_ref' IS DISTINCT FROM p_efecto_ref_esperada
       OR v_manifiesto ->> 'huella_efecto_sha256' IS DISTINCT FROM
          p_huella_efecto_sha256_esperada
       OR pg_catalog.encode(pg_catalog.sha256(
              p_manifiesto_fuente_canonico), 'hex') IS DISTINCT FROM
          v_capacidad ->> 'huella_manifiesto_fuente_sha256'
       OR pg_catalog.encode(pg_catalog.sha256(p_sobre_cose_sign1), 'hex')
          IS DISTINCT FROM v_capacidad ->> 'huella_sobre_cose_sign1_sha256'
       OR pg_catalog.encode(pg_catalog.sha256(
              p_evidencia_verificacion), 'hex') IS DISTINCT FROM
          v_capacidad ->> 'huella_prueba_confianza_sha256'
       OR pg_catalog.encode(pg_catalog.sha256(p_raiz_publica_spki), 'hex')
          IS DISTINCT FROM v_capacidad ->> 'huella_raiz_spki_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'ligadura de fuente rechazada';
    END IF;

    SELECT pg_catalog.count(*) INTO v_checkpoint
      FROM (
        SELECT cp.control_id
          FROM vec_autorizacion_atestada_v3.checkpoint_gobierno AS cp
         WHERE cp.control_id FOR UPDATE
      ) AS checkpoint_bloqueado;
    IF v_checkpoint <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'gobierno de fuente no disponible';
    END IF;
    FOR v_lock IN
        SELECT DISTINCT bloqueos.clave
          FROM pg_catalog.unnest(ARRAY[
            pg_catalog.hashtextextended(
              'vec_autorizacion_atestada_v3:f0:capacidad:v1:' ||
              v_capacidad_ref, 0),
            pg_catalog.hashtextextended(
              'vec_autorizacion_atestada_v3:f0:operacion:v1:' ||
              (v_capacidad ->> 'operacion_ref'), 0)
          ]) AS bloqueos(clave)
         ORDER BY bloqueos.clave
    LOOP
        PERFORM pg_catalog.pg_advisory_xact_lock(v_lock);
    END LOOP;

    SELECT
      (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
          .atestacion_fuente_corporativa_contexto_actor_v1 AS a
        WHERE a.capacidad_ref = v_capacidad_ref),
      (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
          .atestacion_fuente_corporativa_contexto_actor_v1 AS a
        WHERE a.fuente_ref = v_capacidad ->> 'fuente_ref'
          AND a.evento_fuente_ref = v_capacidad ->> 'evento_fuente_ref'),
      (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
          .consumo_fuente_corporativa_contexto_actor_v1 AS c
        WHERE c.capacidad_ref = v_capacidad_ref),
      (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
          .consumo_fuente_corporativa_contexto_actor_v1 AS c
        WHERE c.nonce = v_capacidad ->> 'nonce'),
      (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
          .consumo_fuente_corporativa_contexto_actor_v1 AS c
        WHERE c.operacion_ref = v_capacidad ->> 'operacion_ref')
      INTO v_cantidad_a_capacidad, v_cantidad_a_evento,
           v_cantidad_c_capacidad, v_cantidad_c_nonce,
           v_cantidad_c_operacion;

    IF v_cantidad_a_capacidad + v_cantidad_a_evento +
       v_cantidad_c_capacidad + v_cantidad_c_nonce +
       v_cantidad_c_operacion > 0 THEN
        IF (v_cantidad_a_capacidad, v_cantidad_a_evento,
            v_cantidad_c_capacidad, v_cantidad_c_nonce,
            v_cantidad_c_operacion) <>
           (1::pg_catalog.int8, 1::pg_catalog.int8, 1::pg_catalog.int8,
            1::pg_catalog.int8, 1::pg_catalog.int8) THEN
            RAISE EXCEPTION USING ERRCODE = '23505',
                MESSAGE = 'colision de historia de fuente';
        END IF;
        SELECT pg_catalog.count(*),
               pg_catalog.array_agg(a.capacidad_ref),
               pg_catalog.array_agg(a.fuente_ref),
               pg_catalog.array_agg(a.fuente_version),
               pg_catalog.array_agg(a.evento_fuente_ref),
               pg_catalog.array_agg(c.capacidad_ref),
               pg_catalog.array_agg(c.nonce),
               pg_catalog.array_agg(c.operacion_ref),
               pg_catalog.array_agg(c.consumo_canonico),
               pg_catalog.array_agg(c.consumo_huella_sha256),
               pg_catalog.array_agg(c.consumida_en)
          INTO v_cantidad, v_a_capacidad, v_a_fuente, v_a_version,
               v_a_evento, v_c_capacidad, v_c_nonce, v_c_operacion,
               v_c_canon, v_c_huella, v_c_instante
          FROM vec_autorizacion_atestada_v3
                   .atestacion_fuente_corporativa_contexto_actor_v1 AS a
          JOIN vec_autorizacion_atestada_v3
                   .consumo_fuente_corporativa_contexto_actor_v1 AS c
            ON c.capacidad_ref = a.capacidad_ref
         WHERE a.capacidad_ref = v_capacidad_ref
            OR (a.fuente_ref = v_capacidad ->> 'fuente_ref' AND
                a.evento_fuente_ref = v_capacidad ->> 'evento_fuente_ref')
            OR c.nonce = v_capacidad ->> 'nonce'
            OR c.operacion_ref = v_capacidad ->> 'operacion_ref';
        IF v_cantidad <> 1 THEN
            RAISE EXCEPTION USING ERRCODE = '23505',
                MESSAGE = 'colision de historia de fuente';
        END IF;
        SELECT pg_catalog.count(*),
               pg_catalog.array_agg(x.consumo_canonico),
               pg_catalog.array_agg(x.consumo_huella_sha256)
          INTO v_cantidad, v_canon_esperado, v_huella_esperada
          FROM vec_autorizacion_atestada_v3
               .canon_y_huella_consumo_fuente_corporativa_v1(
                   p_capacidad_canonica, v_c_instante[1]) AS x;
        IF v_cantidad <> 1
           OR vec_autorizacion_atestada_v3.bytea_igual_constante(
                  v_c_canon[1], v_canon_esperado[1]) IS NOT TRUE
           OR v_c_huella[1] IS DISTINCT FROM v_huella_esperada[1]
           OR v_a_capacidad[1] IS DISTINCT FROM v_capacidad_ref
           OR v_c_capacidad[1] IS DISTINCT FROM v_capacidad_ref
           OR v_a_fuente[1] IS DISTINCT FROM v_capacidad ->> 'fuente_ref'
           OR v_a_version[1] IS DISTINCT FROM
              (v_capacidad ->> 'fuente_version')::pg_catalog.numeric
           OR v_a_evento[1] IS DISTINCT FROM
              v_capacidad ->> 'evento_fuente_ref'
           OR v_c_nonce[1] IS DISTINCT FROM v_capacidad ->> 'nonce'
           OR v_c_operacion[1] IS DISTINCT FROM
              v_capacidad ->> 'operacion_ref' THEN
            RAISE EXCEPTION USING ERRCODE = '23505',
                MESSAGE = 'preimagen historica de fuente rechazada';
        END IF;
        v_consumo := pg_catalog.convert_from(
            v_c_canon[1], 'UTF8')::pg_catalog.json;
        IF v_consumo ->> 'capacidad_ref' IS DISTINCT FROM v_a_capacidad[1]
           OR v_consumo ->> 'fuente_ref' IS DISTINCT FROM v_a_fuente[1]
           OR (v_consumo ->> 'fuente_version')::pg_catalog.numeric
              IS DISTINCT FROM v_a_version[1]
           OR v_consumo ->> 'evento_fuente_ref' IS DISTINCT FROM v_a_evento[1]
           OR v_consumo ->> 'operacion_ref' IS DISTINCT FROM v_c_operacion[1]
           OR v_consumo ->> 'efecto_ref' IS DISTINCT FROM p_efecto_ref_esperada
           OR v_consumo ->> 'huella_efecto_sha256' IS DISTINCT FROM
              p_huella_efecto_sha256_esperada THEN
            RAISE EXCEPTION USING ERRCODE = '23505',
                MESSAGE = 'preimagen historica de fuente rechazada';
        END IF;
        RETURN QUERY SELECT
            v_a_capacidad[1], v_a_fuente[1], v_a_version[1], v_a_evento[1],
            v_consumo ->> 'huella_evento_fuente_sha256',
            v_consumo ->> 'huella_manifiesto_fuente_sha256',
            v_c_operacion[1], v_consumo ->> 'efecto_ref',
            v_consumo ->> 'huella_efecto_sha256', v_c_huella[1],
            v_c_instante[1], false;
        RETURN;
    END IF;

    SELECT pg_catalog.count(*),
           (pg_catalog.array_agg(x.capacidad_ref))[1],
           (pg_catalog.array_agg(x.fuente_ref))[1],
           (pg_catalog.array_agg(x.fuente_version))[1],
           (pg_catalog.array_agg(x.evento_fuente_ref))[1],
           (pg_catalog.array_agg(x.huella_evento_fuente_sha256))[1],
           (pg_catalog.array_agg(x.evento_fuente_emitido_en))[1],
           (pg_catalog.array_agg(x.huella_manifiesto_fuente_sha256))[1],
           (pg_catalog.array_agg(x.operacion_ref))[1],
           (pg_catalog.array_agg(x.efecto_ref))[1],
           (pg_catalog.array_agg(x.huella_efecto_sha256))[1],
           (pg_catalog.array_agg(x.nonce))[1],
           (pg_catalog.array_agg(x.acreditada_en))[1]
      INTO v_cantidad, v_n_capacidad, v_n_fuente, v_n_version,
           v_n_evento, v_n_huella_evento, v_n_evento_emitido_en,
           v_n_huella_manifiesto, v_n_operacion, v_n_efecto,
           v_n_huella_efecto, v_n_nonce, v_n_acreditada_en
      FROM vec_autorizacion_atestada_v3
           .acreditar_material_fuente_corporativa_contexto_actor_v1(
               p_audiencia_consumo_esperada, p_accion_esperada,
               p_tipo_efecto_esperado, p_operacion_ref_esperada,
               p_efecto_ref_esperada, p_huella_efecto_sha256_esperada,
               p_capacidad_canonica, p_manifiesto_fuente_canonico,
               p_sobre_cose_sign1, p_evidencia_verificacion,
               p_raiz_publica_spki) AS x;
    IF v_cantidad <> 1
       OR v_n_capacidad IS DISTINCT FROM v_capacidad_ref
       OR v_n_fuente IS DISTINCT FROM
          v_capacidad ->> 'fuente_ref'
       OR v_n_version IS DISTINCT FROM
          (v_capacidad ->> 'fuente_version')::pg_catalog.numeric
       OR v_n_evento IS DISTINCT FROM
          v_capacidad ->> 'evento_fuente_ref'
       OR v_n_huella_evento IS DISTINCT FROM
          v_capacidad ->> 'huella_evento_fuente_sha256'
       OR v_n_evento_emitido_en IS DISTINCT FROM
          (v_capacidad ->> 'evento_fuente_emitido_en')::pg_catalog.timestamptz
       OR v_n_huella_manifiesto IS DISTINCT FROM
          v_capacidad ->> 'huella_manifiesto_fuente_sha256'
       OR v_n_operacion IS DISTINCT FROM
          v_capacidad ->> 'operacion_ref'
       OR v_n_efecto IS DISTINCT FROM v_capacidad ->> 'efecto_ref'
       OR v_n_huella_efecto IS DISTINCT FROM
          v_capacidad ->> 'huella_efecto_sha256'
       OR v_n_nonce IS DISTINCT FROM v_capacidad ->> 'nonce' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'acreditacion de fuente no singular';
    END IF;
    SELECT pg_catalog.count(*),
           pg_catalog.array_agg(x.consumo_canonico),
           pg_catalog.array_agg(x.consumo_huella_sha256)
      INTO v_cantidad, v_canon_esperado, v_huella_esperada
      FROM vec_autorizacion_atestada_v3
           .canon_y_huella_consumo_fuente_corporativa_v1(
               p_capacidad_canonica, v_n_acreditada_en) AS x;
    IF v_cantidad <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'canon de consumo no singular';
    END IF;

    INSERT INTO vec_autorizacion_atestada_v3
        .atestacion_fuente_corporativa_contexto_actor_v1 (
            capacidad_ref, fuente_ref, fuente_version, evento_fuente_ref)
    VALUES (v_n_capacidad, v_n_fuente, v_n_version, v_n_evento);
    INSERT INTO vec_autorizacion_atestada_v3
        .consumo_fuente_corporativa_contexto_actor_v1 (
            capacidad_ref, nonce, operacion_ref, consumo_canonico,
            consumo_huella_sha256, consumida_en)
    VALUES (v_n_capacidad, v_n_nonce, v_n_operacion, v_canon_esperado[1],
            v_huella_esperada[1], v_n_acreditada_en);

    RETURN QUERY SELECT
        v_n_capacidad, v_n_fuente, v_n_version, v_n_evento,
        v_n_huella_evento, v_n_huella_manifiesto, v_n_operacion, v_n_efecto,
        v_n_huella_efecto, v_huella_esperada[1], v_n_acreditada_en, true;
END
$funcion$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3
    .consumir_fuente_corporativa_contexto_actor_v1_atestada(
        pg_catalog.text, pg_catalog.text, pg_catalog.text, pg_catalog.text,
        pg_catalog.text, pg_catalog.text, pg_catalog.bytea, pg_catalog.bytea,
        pg_catalog.bytea, pg_catalog.bytea, pg_catalog.bytea
    )
FROM PUBLIC, vec_autorizacion_atestada_v3_migrador,
    vec_autorizacion_atestada_v3_emisor,
    vec_autorizacion_atestada_v3_consumidor,
    vec_contratacion_temporal_propietario,
    vec_contexto_actor_v1_propietario;
