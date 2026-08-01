-- Acreditacion privada del material de una capacidad de fuente corporativa.
-- C2 conserva en exclusiva locks de replay, persistencia y autoridad exterior.

CREATE FUNCTION vec_autorizacion_atestada_v3
    .avanzar_checkpoint_puntero_fuente_corporativa_v1()
RETURNS pg_catalog.trigger
LANGUAGE plpgsql
VOLATILE
STRICT
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_revision pg_catalog.numeric;
BEGIN
    IF TG_OP <> 'INSERT' OR TG_WHEN <> 'BEFORE' OR TG_LEVEL <> 'ROW'
       OR TG_TABLE_SCHEMA <> 'vec_autorizacion_atestada_v3'
       OR TG_TABLE_NAME NOT IN (
           'puntero_clave_emision', 'puntero_configuracion_actual'
       ) OR TG_NARGS <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'avance de checkpoint por puntero rechazado';
    END IF;

    -- Toda rotacion toma y avanza la misma fila antes de publicar el puntero.
    UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno
       SET revision = revision + 1,
           actualizada_en = pg_catalog.clock_timestamp()
     WHERE control_id
       AND revision < 9007199254740991::pg_catalog.numeric
     RETURNING revision INTO v_revision;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'checkpoint de puntero no disponible';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER f0_checkpoint_puntero_antes
BEFORE INSERT ON vec_autorizacion_atestada_v3.puntero_clave_emision
FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3
    .avanzar_checkpoint_puntero_fuente_corporativa_v1();
CREATE TRIGGER f0_checkpoint_puntero_antes
BEFORE INSERT ON vec_autorizacion_atestada_v3.puntero_configuracion_actual
FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3
    .avanzar_checkpoint_puntero_fuente_corporativa_v1();

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3
    .avanzar_checkpoint_puntero_fuente_corporativa_v1()
FROM PUBLIC, vec_autorizacion_atestada_v3_migrador,
    vec_autorizacion_atestada_v3_emisor,
    vec_autorizacion_atestada_v3_consumidor,
    vec_contratacion_temporal_propietario,
    vec_contexto_actor_v1_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3
    .acreditar_material_fuente_corporativa_contexto_actor_v1(
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
    evento_fuente_emitido_en pg_catalog.timestamptz,
    huella_manifiesto_fuente_sha256 pg_catalog.text,
    operacion_ref pg_catalog.text,
    efecto_ref pg_catalog.text,
    huella_efecto_sha256 pg_catalog.text,
    nonce pg_catalog.text,
    acreditada_en pg_catalog.timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_capacidad pg_catalog.json;
    v_manifiesto pg_catalog.json;
    v_fuente vec_autorizacion_atestada_v3
        .fuente_corporativa_contexto_actor_v1%ROWTYPE;
    v_clave vec_autorizacion_atestada_v3
        .clave_capacidad_version%ROWTYPE;
    v_puntero_clave vec_autorizacion_atestada_v3
        .puntero_clave_emision%ROWTYPE;
    v_configuracion vec_autorizacion_atestada_v3
        .configuracion_confianza_version%ROWTYPE;
    v_puntero_configuracion vec_autorizacion_atestada_v3
        .puntero_configuracion_actual%ROWTYPE;
    v_raiz vec_autorizacion_atestada_v3
        .raiz_confianza_version%ROWTYPE;
    v_configuracion_secuencia_minima pg_catalog.numeric;
    v_raiz_version_minima pg_catalog.numeric;
    v_fuente_version pg_catalog.numeric;
    v_clave_version pg_catalog.numeric;
    v_revision_gobierno pg_catalog.numeric;
    v_configuracion_secuencia pg_catalog.numeric;
    v_raiz_version pg_catalog.numeric;
    v_evento_emitido_en pg_catalog.timestamptz;
    v_emitida_en pg_catalog.timestamptz;
    v_expira_en pg_catalog.timestamptz;
    v_ahora pg_catalog.timestamptz;
    v_timeout_sentencia pg_catalog.numeric;
    v_timeout_transaccion pg_catalog.numeric;
    v_timeout_inactivo pg_catalog.numeric;
BEGIN
    IF CURRENT_USER IS DISTINCT FROM
           'vec_autorizacion_atestada_v3_propietario'
       OR pg_catalog.current_setting('transaction_isolation') IS DISTINCT FROM
          'serializable'
       OR pg_catalog.current_setting('transaction_read_only') IS DISTINCT FROM
          'off'
       OR pg_catalog.current_setting('TimeZone') IS DISTINCT FROM 'UTC'
       OR pg_catalog.pg_is_in_recovery() THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'contexto privado de fuente rechazado';
    END IF;

    SELECT s.setting::pg_catalog.numeric
      INTO v_timeout_sentencia
      FROM pg_catalog.pg_settings AS s
     WHERE s.name = 'statement_timeout';
    SELECT s.setting::pg_catalog.numeric
      INTO v_timeout_transaccion
      FROM pg_catalog.pg_settings AS s
     WHERE s.name = 'transaction_timeout';
    SELECT s.setting::pg_catalog.numeric
      INTO v_timeout_inactivo
      FROM pg_catalog.pg_settings AS s
     WHERE s.name = 'idle_in_transaction_session_timeout';
    IF v_timeout_sentencia NOT BETWEEN 1000 AND 10000
       OR v_timeout_transaccion NOT BETWEEN 1000 AND 15000
       OR v_timeout_inactivo NOT BETWEEN 1000 AND 15000 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'limites privados de fuente rechazados';
    END IF;

    -- Los limites preceden a conversiones, hashes y lectura de gobierno.
    IF vec_autorizacion_atestada_v3
           .capacidad_fuente_bytes_validos(p_capacidad_canonica) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
           .manifiesto_fuente_bytes_validos(
               p_manifiesto_fuente_canonico
           ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
           .sobre_cose_sign1_fuente_bytes_validos(
               p_sobre_cose_sign1
           ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
           .evidencia_verificacion_fuente_bytes_validos(
               p_evidencia_verificacion
           ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
           .raiz_publica_spki_fuente_bytes_validos(
               p_raiz_publica_spki
           ) IS NOT TRUE
       OR pg_catalog.substr(p_raiz_publica_spki, 1, 12) <>
          pg_catalog.decode('302a300506032b6570032100', 'hex')
       OR vec_autorizacion_atestada_v3
           .operacion_ref_fuente_corporativa_valida(
               p_operacion_ref_esperada
           ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
           .referencia_opaca_fuente_corporativa_valida(
               p_efecto_ref_esperada
           ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.huella_sha256_valida(
               p_huella_efecto_sha256_esperada
           ) IS NOT TRUE
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
           ELSE false
           END) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'entrada de fuente invalida';
    END IF;

    IF vec_autorizacion_atestada_v3
           .capacidad_fuente_corporativa_v1_canonica(
               p_capacidad_canonica
           ) IS DISTINCT FROM p_capacidad_canonica
       OR vec_autorizacion_atestada_v3
           .manifiesto_fuente_corporativa_v1_canonico(
               p_manifiesto_fuente_canonico
           ) IS DISTINCT FROM p_manifiesto_fuente_canonico THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'canon de fuente invalido';
    END IF;
    v_capacidad := pg_catalog.convert_from(
        p_capacidad_canonica, 'UTF8'
    )::pg_catalog.json;
    v_manifiesto := pg_catalog.convert_from(
        p_manifiesto_fuente_canonico, 'UTF8'
    )::pg_catalog.json;
    v_fuente_version := (v_capacidad ->> 'fuente_version')::numeric;
    v_clave_version := (v_capacidad ->> 'clave_version')::numeric;
    v_revision_gobierno :=
        (v_capacidad ->> 'revision_gobierno')::numeric;
    v_configuracion_secuencia :=
        (v_capacidad ->> 'configuracion_secuencia')::numeric;
    v_raiz_version := (v_capacidad ->> 'raiz_version')::numeric;
    v_evento_emitido_en :=
        (v_capacidad ->> 'evento_fuente_emitido_en')::timestamptz;
    v_emitida_en := (v_capacidad ->> 'emitida_en')::timestamptz;
    v_expira_en := (v_capacidad ->> 'expira_en')::timestamptz;

    IF v_capacidad ->> 'audiencia_consumo' IS DISTINCT FROM
           p_audiencia_consumo_esperada
       OR v_capacidad ->> 'accion' IS DISTINCT FROM p_accion_esperada
       OR v_capacidad ->> 'tipo_efecto' IS DISTINCT FROM
          p_tipo_efecto_esperado
       OR v_capacidad ->> 'operacion_ref' IS DISTINCT FROM
          p_operacion_ref_esperada
       OR v_capacidad ->> 'efecto_ref' IS DISTINCT FROM
          p_efecto_ref_esperada
       OR v_capacidad ->> 'huella_efecto_sha256' IS DISTINCT FROM
          p_huella_efecto_sha256_esperada
       OR v_manifiesto ->> 'fuente_ref' IS DISTINCT FROM
          v_capacidad ->> 'fuente_ref'
       OR (v_manifiesto ->> 'fuente_version')::numeric IS DISTINCT FROM
          v_fuente_version
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
       OR v_manifiesto ->> 'efecto_ref' IS DISTINCT FROM
          p_efecto_ref_esperada
       OR v_manifiesto ->> 'huella_efecto_sha256' IS DISTINCT FROM
          p_huella_efecto_sha256_esperada
       OR pg_catalog.encode(
              pg_catalog.sha256(p_manifiesto_fuente_canonico), 'hex'
          ) IS DISTINCT FROM
          v_capacidad ->> 'huella_manifiesto_fuente_sha256'
       OR pg_catalog.encode(
              pg_catalog.sha256(p_sobre_cose_sign1), 'hex'
          ) IS DISTINCT FROM
          v_capacidad ->> 'huella_sobre_cose_sign1_sha256'
       OR pg_catalog.encode(
              pg_catalog.sha256(p_evidencia_verificacion), 'hex'
          ) IS DISTINCT FROM
          v_capacidad ->> 'huella_prueba_confianza_sha256'
       OR pg_catalog.encode(
              pg_catalog.sha256(p_raiz_publica_spki), 'hex'
          ) IS DISTINCT FROM
          v_capacidad ->> 'huella_raiz_spki_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'ligadura de fuente rechazada';
    END IF;

    -- El checkpoint precede todas las filas de gobierno y serializa altas y
    -- revocaciones. C2 habra tomado antes sus advisory locks y resuelto replay.
    SELECT cp.configuracion_secuencia_minima, cp.raiz_version_minima
      INTO v_configuracion_secuencia_minima, v_raiz_version_minima
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno AS cp
     WHERE cp.control_id
     FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'gobierno de fuente no disponible';
    END IF;

    SELECT f.* INTO v_fuente
      FROM vec_autorizacion_atestada_v3
               .fuente_corporativa_contexto_actor_v1 AS f
     WHERE f.fuente_ref = v_capacidad ->> 'fuente_ref'
       AND f.fuente_version = v_fuente_version
       AND f.audiencia_consumo = p_audiencia_consumo_esperada
     FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'fuente corporativa rechazada';
    END IF;

    SELECT k.* INTO v_clave
      FROM vec_autorizacion_atestada_v3.clave_capacidad_version AS k
     WHERE k.clave_id = v_capacidad ->> 'clave_id'
       AND k.version = v_clave_version
     FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'clave corporativa rechazada';
    END IF;
    SELECT p.* INTO v_puntero_clave
      FROM vec_autorizacion_atestada_v3.puntero_clave_emision AS p
     WHERE EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v3.clave_capacidad_version AS k
            WHERE k.clave_id = p.clave_id
              AND k.version = p.version
              AND k.audiencia_consumo = p_audiencia_consumo_esperada
       )
       AND p.establecida_en <= v_emitida_en
     ORDER BY p.orden DESC
     LIMIT 1
     FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'puntero de clave corporativa rechazado';
    END IF;

    SELECT c.* INTO v_configuracion
      FROM vec_autorizacion_atestada_v3
               .configuracion_confianza_version AS c
     WHERE c.revision = v_capacidad ->> 'configuracion_revision'
     FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'configuracion corporativa rechazada';
    END IF;
    SELECT p.* INTO v_puntero_configuracion
      FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual AS p
     WHERE p.configuracion_revision = v_configuracion.revision
       AND p.establecida_en <= v_emitida_en
     ORDER BY p.orden DESC
     LIMIT 1
     FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'puntero de configuracion corporativa rechazado';
    END IF;

    SELECT r.* INTO v_raiz
      FROM vec_autorizacion_atestada_v3.raiz_confianza_version AS r
     WHERE r.clave_id = v_capacidad ->> 'raiz_clave_id'
       AND r.version = v_raiz_version
     FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'raiz corporativa rechazada';
    END IF;
    PERFORM 1
      FROM vec_autorizacion_atestada_v3.configuracion_raiz AS cr
     WHERE cr.configuracion_revision = v_configuracion.revision
       AND cr.raiz_clave_id = v_raiz.clave_id
       AND cr.raiz_version = v_raiz.version
     FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'asociacion de raiz corporativa rechazada';
    END IF;

    v_ahora := pg_catalog.clock_timestamp();
    IF v_fuente.accion IS DISTINCT FROM p_accion_esperada
       OR v_fuente.tipo_efecto IS DISTINCT FROM p_tipo_efecto_esperado
       OR v_fuente.clave_id IS DISTINCT FROM v_clave.clave_id
       OR v_fuente.clave_version IS DISTINCT FROM v_clave.version
       OR v_fuente.revision_gobierno IS DISTINCT FROM v_revision_gobierno
       OR v_fuente.huella_gobierno_sha256 IS DISTINCT FROM
          v_capacidad ->> 'huella_gobierno_sha256'
       OR v_fuente.emisor_id IS DISTINCT FROM v_capacidad ->> 'emisor_id'
       OR v_fuente.configuracion_revision IS DISTINCT FROM
          v_configuracion.revision
       OR v_fuente.configuracion_secuencia IS DISTINCT FROM
          v_configuracion_secuencia
       OR v_fuente.huella_configuracion_sha256 IS DISTINCT FROM
          v_capacidad ->> 'huella_configuracion_sha256'
       OR v_fuente.raiz_clave_id IS DISTINCT FROM v_raiz.clave_id
       OR v_fuente.raiz_version IS DISTINCT FROM v_raiz.version
       OR v_fuente.huella_raiz_spki_sha256 IS DISTINCT FROM
          v_capacidad ->> 'huella_raiz_spki_sha256'
       OR v_fuente.audiencia_despliegue IS DISTINCT FROM
          v_capacidad ->> 'audiencia_despliegue'
       OR v_fuente.suite IS DISTINCT FROM v_capacidad ->> 'suite'
       OR v_clave.revision_gobierno IS DISTINCT FROM v_revision_gobierno
       OR v_clave.huella_gobierno_sha256 IS DISTINCT FROM
          v_capacidad ->> 'huella_gobierno_sha256'
       OR v_clave.emisor_id IS DISTINCT FROM v_capacidad ->> 'emisor_id'
       OR v_clave.audiencia_consumo IS DISTINCT FROM
          p_audiencia_consumo_esperada
       OR v_puntero_clave.clave_id IS DISTINCT FROM v_clave.clave_id
       OR v_puntero_clave.version IS DISTINCT FROM v_clave.version
       OR v_configuracion.secuencia IS DISTINCT FROM
          v_configuracion_secuencia
       OR v_configuracion.secuencia < v_configuracion_secuencia_minima
       OR v_configuracion.huella_configuracion_sha256 IS DISTINCT FROM
          v_capacidad ->> 'huella_configuracion_sha256'
       OR v_puntero_configuracion.configuracion_revision IS DISTINCT FROM
          v_configuracion.revision
       OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3
               .puntero_configuracion_actual AS p
            WHERE p.orden > v_puntero_configuracion.orden
              AND p.establecida_en <= v_ahora
       )
       OR v_raiz.version < v_raiz_version_minima
       OR v_raiz.huella_spki_sha256 IS DISTINCT FROM
          v_capacidad ->> 'huella_raiz_spki_sha256'
       OR vec_autorizacion_atestada_v3.bytea_igual_constante(
              v_raiz.clave_publica_spki, p_raiz_publica_spki
          ) IS NOT TRUE
       OR v_raiz.suite IS DISTINCT FROM v_capacidad ->> 'suite'
       OR v_raiz.audiencia_despliegue IS DISTINCT FROM
          v_capacidad ->> 'audiencia_despliegue'
       OR v_emitida_en < v_fuente.valida_desde
       OR v_expira_en > v_fuente.valida_hasta
       OR v_emitida_en < v_clave.valida_desde
       OR v_expira_en > v_clave.valida_hasta
       OR v_emitida_en < v_configuracion.publicada_en
       OR v_expira_en > v_configuracion.expira_en
       OR v_emitida_en < v_raiz.valida_desde
       OR v_expira_en > v_raiz.valida_hasta
       OR v_ahora < v_emitida_en OR v_ahora >= v_expira_en
       OR v_ahora < v_fuente.valida_desde
       OR v_ahora >= v_fuente.valida_hasta
       OR v_ahora < v_clave.valida_desde OR v_ahora >= v_clave.valida_hasta
       OR v_ahora < v_configuracion.publicada_en
       OR v_ahora >= v_configuracion.expira_en
       OR v_ahora < v_raiz.valida_desde OR v_ahora >= v_raiz.valida_hasta
       OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3
               .revocacion_fuente_corporativa_contexto_actor_v1 AS r
            WHERE r.fuente_ref = v_fuente.fuente_ref
              AND r.fuente_version = v_fuente.fuente_version
              AND r.audiencia_consumo = v_fuente.audiencia_consumo
              AND r.revocada_en <= v_ahora
       ) OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3
               .revocacion_clave_capacidad AS r
            WHERE r.clave_id = v_clave.clave_id
              AND r.version = v_clave.version
              AND r.revocada_en <= v_ahora
       ) OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3
               .revocacion_configuracion AS r
            WHERE r.configuracion_revision = v_configuracion.revision
              AND r.revocada_en <= v_ahora
       ) OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3
               .revocacion_raiz AS r
            WHERE r.raiz_clave_id = v_raiz.clave_id
              AND r.raiz_version = v_raiz.version
              AND r.revocada_en <= v_ahora
       ) OR vec_autorizacion_atestada_v3
           .mac_capacidad_fuente_corporativa_v1_valido(
               p_capacidad_canonica, v_clave.secreto_hmac
           ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'material corporativo rechazado';
    END IF;

    -- El segundo reloj y las cuatro revocaciones cierran la ventana hasta C2.
    v_ahora := pg_catalog.clock_timestamp();
    IF v_ahora < v_emitida_en OR v_ahora >= v_expira_en
       OR v_ahora < v_fuente.valida_desde
       OR v_ahora >= v_fuente.valida_hasta
       OR v_ahora < v_clave.valida_desde OR v_ahora >= v_clave.valida_hasta
       OR v_ahora < v_configuracion.publicada_en
       OR v_ahora >= v_configuracion.expira_en
       OR v_ahora < v_raiz.valida_desde OR v_ahora >= v_raiz.valida_hasta
       OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3
               .puntero_configuracion_actual AS p
            WHERE p.orden > v_puntero_configuracion.orden
              AND p.establecida_en <= v_ahora
       )
       OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3
               .revocacion_fuente_corporativa_contexto_actor_v1 AS r
            WHERE r.fuente_ref = v_fuente.fuente_ref
              AND r.fuente_version = v_fuente.fuente_version
              AND r.audiencia_consumo = v_fuente.audiencia_consumo
              AND r.revocada_en <= v_ahora
       ) OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3
               .revocacion_clave_capacidad AS r
            WHERE r.clave_id = v_clave.clave_id
              AND r.version = v_clave.version
              AND r.revocada_en <= v_ahora
       ) OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3
               .revocacion_configuracion AS r
            WHERE r.configuracion_revision = v_configuracion.revision
              AND r.revocada_en <= v_ahora
       ) OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3
               .revocacion_raiz AS r
            WHERE r.raiz_clave_id = v_raiz.clave_id
              AND r.raiz_version = v_raiz.version
              AND r.revocada_en <= v_ahora
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'vigencia corporativa agotada';
    END IF;

    RETURN QUERY SELECT
        'cfc_' || pg_catalog.encode(
            pg_catalog.sha256(p_capacidad_canonica), 'hex'
        ),
        v_fuente.fuente_ref,
        v_fuente.fuente_version,
        v_capacidad ->> 'evento_fuente_ref',
        v_capacidad ->> 'huella_evento_fuente_sha256',
        v_evento_emitido_en,
        v_capacidad ->> 'huella_manifiesto_fuente_sha256',
        v_capacidad ->> 'operacion_ref',
        v_capacidad ->> 'efecto_ref',
        v_capacidad ->> 'huella_efecto_sha256',
        v_capacidad ->> 'nonce',
        v_ahora;
END
$funcion$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3
    .acreditar_material_fuente_corporativa_contexto_actor_v1(
        pg_catalog.text, pg_catalog.text, pg_catalog.text, pg_catalog.text,
        pg_catalog.text, pg_catalog.text, pg_catalog.bytea, pg_catalog.bytea,
        pg_catalog.bytea, pg_catalog.bytea, pg_catalog.bytea
    )
FROM PUBLIC, vec_autorizacion_atestada_v3_migrador,
    vec_autorizacion_atestada_v3_emisor,
    vec_autorizacion_atestada_v3_consumidor,
    vec_contratacion_temporal_propietario,
    vec_contexto_actor_v1_propietario;
