-- O4-04B: sincronización cerrada, resolución nominal y barrera TCB actual.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000019_resolucion_gobierno_o4_04b', 0
    )
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_04:migraciones', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_actual'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_retirada'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_material_actuacion(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_publicar(jsonb)'
       ) IS NOT NULL
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_contratacion_temporal_gobernador'
              AND NOT rolcanlogin
              AND NOT rolsuper
              AND NOT rolcreaterole
              AND NOT rolcreatedb
              AND NOT rolreplication
              AND NOT rolbypassrls
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members m
             JOIN pg_catalog.pg_roles miembro ON miembro.oid = m.member
             JOIN pg_catalog.pg_roles grupo ON grupo.oid = m.roleid
            WHERE miembro.rolname = 'vec_contratacion_temporal_gobernador'
              AND grupo.rolname = 'vec_contratacion_temporal_propietario'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para resolver gobierno O4-04B';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_politica_ligada(
    p_catalogo jsonb,
    p_politica jsonb
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_via_politica jsonb;
    v_via_catalogo jsonb;
    v_indice integer;
BEGIN
    IF pg_catalog.jsonb_array_length(p_catalogo -> 'vias') <>
       pg_catalog.jsonb_array_length(p_politica -> 'vias') THEN
        RETURN false;
    END IF;
    FOR v_via_politica IN
        SELECT valor
          FROM pg_catalog.jsonb_array_elements(p_politica -> 'vias')
               AS v(valor)
    LOOP
        SELECT valor
          INTO v_via_catalogo
          FROM pg_catalog.jsonb_array_elements(p_catalogo -> 'vias')
               AS c(valor)
         WHERE valor ->> 'clave' = v_via_politica ->> 'via_clave';
        IF v_via_catalogo IS NULL
           OR pg_catalog.jsonb_array_length(
                  v_via_catalogo -> 'comprobaciones'
              ) <> pg_catalog.jsonb_array_length(
                  v_via_politica -> 'comprobaciones'
              ) THEN
            RETURN false;
        END IF;
        FOR v_indice IN 0..pg_catalog.jsonb_array_length(
            v_via_catalogo -> 'comprobaciones'
        ) - 1 LOOP
            IF v_via_catalogo #>> ARRAY[
                   'comprobaciones', v_indice::text, 'clave'
               ] IS DISTINCT FROM
               v_via_politica #>> ARRAY[
                   'comprobaciones', v_indice::text, 'clave'
               ] THEN
                RETURN false;
            END IF;
        END LOOP;
    END LOOP;
    RETURN true;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_publicar(
    p_publicacion jsonb
)
RETURNS TABLE (
    resultado text,
    evento_ref text,
    huella_evento_sha256 text
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_catalogo jsonb;
    v_politica jsonb;
    v_actuacion jsonb;
    v_secuencia bigint;
    v_evento_ref text;
    v_huella_evento text;
    v_checkpoint record;
    v_anterior record;
    v_ahora timestamptz(6) := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_gobernador', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER'
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_entorno_valido(false)
       OR p_publicacion IS NULL
       OR pg_catalog.octet_length(p_publicacion::text) NOT BETWEEN
          2 AND 3145728
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_publicacion,
           ARRAY[
               'actuacion', 'catalogo', 'esquema', 'evento_ref',
               'politica', 'secuencia'
           ]::text[]
       )
       OR p_publicacion ->> 'esquema' <>
          'vec.contratacion-temporal.gobierno-cobertura.o4-04b.v1'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion -> 'secuencia',
           1,
           9007199254740991::numeric
       )
       OR (p_publicacion ->> 'evento_ref') !~
          '^evento_gobi_o404b_[a-f0-9]{32}$' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'publicación de gobierno O4-04B no autorizada';
    END IF;
    v_catalogo := p_publicacion -> 'catalogo';
    v_politica := p_publicacion -> 'politica';
    v_actuacion := p_publicacion -> 'actuacion';
    v_secuencia := (p_publicacion ->> 'secuencia')::bigint;
    v_evento_ref := p_publicacion ->> 'evento_ref';
    v_huella_evento := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(p_publicacion::text, 'UTF8')
    ), 'hex');
    IF pg_catalog.encode(pg_catalog.sha256(
           vec_contratacion_temporal.gobi_o404b_material_catalogo(v_catalogo)
       ), 'hex') IS DISTINCT FROM v_catalogo ->> 'huella_sha256'
       OR pg_catalog.encode(pg_catalog.sha256(
           vec_contratacion_temporal.gobi_o404b_material_politica(v_politica)
       ), 'hex') IS DISTINCT FROM v_politica ->> 'huella_sha256'
       OR pg_catalog.encode(pg_catalog.sha256(
           vec_contratacion_temporal.gobi_o404b_material_actuacion(v_actuacion)
       ), 'hex') IS DISTINCT FROM v_actuacion ->> 'huella_sha256'
       OR v_politica -> 'catalogo' IS DISTINCT FROM
          pg_catalog.jsonb_build_object(
              'referencia', v_catalogo ->> 'referencia',
              'version', v_catalogo -> 'version',
              'huella_sha256', v_catalogo ->> 'huella_sha256'
          )
       OR v_actuacion -> 'catalogo' IS DISTINCT FROM
          v_politica -> 'catalogo'
       OR v_actuacion -> 'politica' IS DISTINCT FROM
          pg_catalog.jsonb_build_object(
              'referencia', v_politica ->> 'referencia',
              'version', v_politica -> 'version',
              'huella_sha256', v_politica ->> 'huella_sha256'
          )
       OR v_actuacion ->> 'organizacion_ref' IS DISTINCT FROM
          v_politica ->> 'organizacion_ref'
       OR v_actuacion ->> 'finalidad_contratacion_clave'
          IS DISTINCT FROM v_politica ->> 'finalidad_clave'
       OR v_actuacion ->> 'finalidad_contratacion_ref'
          IS DISTINCT FROM v_politica ->> 'finalidad_ref'
       OR NOT vec_contratacion_temporal.gobi_o404b_politica_ligada(
           v_catalogo, v_politica
       )
       OR (v_politica ->> 'publicada_en')::timestamptz <
          (v_catalogo ->> 'publicado_en')::timestamptz
       OR (v_actuacion ->> 'publicada_en')::timestamptz <
          (v_politica ->> 'publicada_en')::timestamptz
       OR (v_politica #>> '{vigencia,desde}')::timestamptz <
          (v_catalogo #>> '{vigencia,desde}')::timestamptz
       OR (
           v_catalogo #>> '{vigencia,hasta}' <>
               '0001-01-01T00:00:00Z'
           AND (v_politica #>> '{vigencia,hasta}')::timestamptz >
               (v_catalogo #>> '{vigencia,hasta}')::timestamptz
       )
       OR (v_actuacion #>> '{vigencia,desde}')::timestamptz <
          (v_politica #>> '{vigencia,desde}')::timestamptz
       OR (v_actuacion #>> '{vigencia,hasta}')::timestamptz >
          (v_politica #>> '{vigencia,hasta}')::timestamptz THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de gobierno O4-04B no confiable';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_contratacion_temporal:o4_04:migraciones', 0
        )
    );
    -- La fila común de C nace en 000020; B serializa aquí su checkpoint propio.
    SELECT * INTO STRICT v_checkpoint
      FROM vec_contratacion_temporal.gobi_o404b_checkpoint
     WHERE control FOR UPDATE;
    IF v_secuencia <= v_checkpoint.ultima_secuencia THEN
        IF EXISTS (
            SELECT 1
              FROM vec_contratacion_temporal.gobi_o404b_evento e
             WHERE e.secuencia = v_secuencia
               AND e.evento_ref = v_evento_ref
               AND e.tipo = 'publicacion'
               AND e.huella_evento_sha256 = v_huella_evento
               AND e.contenido_evento = p_publicacion
        ) THEN
            RETURN QUERY SELECT 'repetida'::text, v_evento_ref,
                                v_huella_evento;
            RETURN;
        END IF;
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'replay divergente de gobierno O4-04B';
    END IF;
    IF v_secuencia <> v_checkpoint.ultima_secuencia + 1 THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'secuencia de gobierno O4-04B no contigua';
    END IF;
    SELECT a.secuencia, a.publicada_en
      INTO v_anterior
      FROM vec_contratacion_temporal.gobi_o404b_actual p
      JOIN vec_contratacion_temporal.gobi_o404b_actuacion a
        ON a.referencia = p.actuacion_ref
       AND a.version = p.actuacion_version
       AND a.huella_sha256 = p.actuacion_huella_sha256
     WHERE p.organizacion_ref = v_actuacion ->> 'organizacion_ref'
       AND p.accion = v_actuacion ->> 'accion'
     FOR UPDATE OF p;
    IF FOUND AND (
        v_anterior.secuencia >= v_secuencia
        OR v_anterior.publicada_en >
           (v_actuacion ->> 'publicada_en')::timestamptz
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'puntero de gobierno O4-04B no monotónico';
    END IF;
    INSERT INTO vec_contratacion_temporal.gobi_o404b_evento VALUES (
        v_secuencia, v_evento_ref, 'publicacion', v_huella_evento,
        p_publicacion, v_ahora
    );
    INSERT INTO vec_contratacion_temporal.gobi_o404b_catalogo (
        referencia, version, huella_sha256, publicacion_json, publicado_en,
        vigente_desde, vigente_hasta, evento_ref, secuencia
    ) VALUES (
        v_catalogo ->> 'referencia', (v_catalogo ->> 'version')::numeric,
        v_catalogo ->> 'huella_sha256', v_catalogo,
        (v_catalogo ->> 'publicado_en')::timestamptz,
        (v_catalogo #>> '{vigencia,desde}')::timestamptz,
        CASE WHEN v_catalogo #>> '{vigencia,hasta}' =
                  '0001-01-01T00:00:00Z' THEN NULL
             ELSE (v_catalogo #>> '{vigencia,hasta}')::timestamptz END,
        v_evento_ref, v_secuencia
    ) ON CONFLICT (referencia, version) DO NOTHING;
    IF NOT EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.gobi_o404b_catalogo c
         WHERE c.referencia = v_catalogo ->> 'referencia'
           AND c.version = (v_catalogo ->> 'version')::numeric
           AND c.huella_sha256 = v_catalogo ->> 'huella_sha256'
           AND c.publicacion_json = v_catalogo
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'colisión de catálogo O4-04B';
    END IF;
    INSERT INTO vec_contratacion_temporal.gobi_o404b_politica (
        referencia, version, huella_sha256, catalogo_ref, catalogo_version,
        catalogo_huella_sha256, organizacion_ref, finalidad_clave,
        finalidad_ref, publicacion_json, publicada_en, vigente_desde,
        vigente_hasta, evento_ref, secuencia
    ) VALUES (
        v_politica ->> 'referencia', (v_politica ->> 'version')::numeric,
        v_politica ->> 'huella_sha256',
        v_politica #>> '{catalogo,referencia}',
        (v_politica #>> '{catalogo,version}')::numeric,
        v_politica #>> '{catalogo,huella_sha256}',
        v_politica ->> 'organizacion_ref',
        v_politica ->> 'finalidad_clave',
        v_politica ->> 'finalidad_ref', v_politica,
        (v_politica ->> 'publicada_en')::timestamptz,
        (v_politica #>> '{vigencia,desde}')::timestamptz,
        (v_politica #>> '{vigencia,hasta}')::timestamptz,
        v_evento_ref, v_secuencia
    ) ON CONFLICT (referencia, version) DO NOTHING;
    IF NOT EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.gobi_o404b_politica p
         WHERE p.referencia = v_politica ->> 'referencia'
           AND p.version = (v_politica ->> 'version')::numeric
           AND p.huella_sha256 = v_politica ->> 'huella_sha256'
           AND p.publicacion_json = v_politica
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'colisión de política O4-04B';
    END IF;
    INSERT INTO vec_contratacion_temporal.gobi_o404b_actuacion (
        referencia, version, huella_sha256, organizacion_ref, accion,
        catalogo_ref, catalogo_version, catalogo_huella_sha256,
        politica_ref, politica_version, politica_huella_sha256,
        publicacion_json, publicada_en, vigente_desde, vigente_hasta,
        evento_ref, secuencia
    ) VALUES (
        v_actuacion ->> 'referencia', (v_actuacion ->> 'version')::numeric,
        v_actuacion ->> 'huella_sha256',
        v_actuacion ->> 'organizacion_ref', v_actuacion ->> 'accion',
        v_actuacion #>> '{catalogo,referencia}',
        (v_actuacion #>> '{catalogo,version}')::numeric,
        v_actuacion #>> '{catalogo,huella_sha256}',
        v_actuacion #>> '{politica,referencia}',
        (v_actuacion #>> '{politica,version}')::numeric,
        v_actuacion #>> '{politica,huella_sha256}', v_actuacion,
        (v_actuacion ->> 'publicada_en')::timestamptz,
        (v_actuacion #>> '{vigencia,desde}')::timestamptz,
        (v_actuacion #>> '{vigencia,hasta}')::timestamptz,
        v_evento_ref, v_secuencia
    ) ON CONFLICT (referencia, version) DO NOTHING;
    IF NOT EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.gobi_o404b_actuacion a
         WHERE a.referencia = v_actuacion ->> 'referencia'
           AND a.version = (v_actuacion ->> 'version')::numeric
           AND a.huella_sha256 = v_actuacion ->> 'huella_sha256'
           AND a.publicacion_json = v_actuacion
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'colisión de actuación O4-04B';
    END IF;
    INSERT INTO vec_contratacion_temporal.gobi_o404b_actual (
        organizacion_ref, accion, actuacion_ref, actuacion_version,
        actuacion_huella_sha256, secuencia, evento_ref, actualizada_en
    ) VALUES (
        v_actuacion ->> 'organizacion_ref', v_actuacion ->> 'accion',
        v_actuacion ->> 'referencia', (v_actuacion ->> 'version')::numeric,
        v_actuacion ->> 'huella_sha256', v_secuencia, v_evento_ref, v_ahora
    ) ON CONFLICT (organizacion_ref, accion) DO UPDATE SET
        actuacion_ref = EXCLUDED.actuacion_ref,
        actuacion_version = EXCLUDED.actuacion_version,
        actuacion_huella_sha256 = EXCLUDED.actuacion_huella_sha256,
        secuencia = EXCLUDED.secuencia,
        evento_ref = EXCLUDED.evento_ref,
        actualizada_en = EXCLUDED.actualizada_en;
    UPDATE vec_contratacion_temporal.gobi_o404b_checkpoint SET
        ultima_secuencia = v_secuencia,
        ultimo_evento_ref = v_evento_ref,
        ultima_huella_evento_sha256 = v_huella_evento,
        actualizado_en = v_ahora
     WHERE control;
    RETURN QUERY SELECT 'publicada'::text, v_evento_ref, v_huella_evento;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_retirar(
    p_retirada jsonb
)
RETURNS TABLE (
    resultado text,
    evento_ref text,
    huella_evento_sha256 text
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_secuencia bigint;
    v_evento_ref text;
    v_huella text;
    v_retirada_en timestamptz(6);
    v_checkpoint record;
    v_ahora timestamptz(6) := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_gobernador', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER'
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_entorno_valido(false)
       OR p_retirada IS NULL
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_retirada,
           ARRAY[
               'accion', 'actuacion_huella_sha256', 'actuacion_ref',
               'actuacion_version', 'esquema', 'evento_ref',
               'organizacion_ref', 'retirada_en', 'secuencia'
           ]::text[]
       )
       OR p_retirada ->> 'esquema' <>
          'vec.contratacion-temporal.retirar-gobierno-cobertura.o4-04b.v1'
       OR (p_retirada ->> 'organizacion_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_retirada ->> 'accion' NOT IN (
          'contratacion_temporal.cobertura.decidir',
          'contratacion_temporal.cobertura.rectificar'
       )
       OR (p_retirada ->> 'actuacion_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_retirada -> 'actuacion_version',
           1,
           9007199254740991::numeric
       )
       OR (p_retirada ->> 'actuacion_huella_sha256') !~
          '^[a-f0-9]{64}$'
       OR p_retirada ->> 'actuacion_huella_sha256' =
          pg_catalog.repeat('0', 64)
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_retirada -> 'secuencia',
           1,
           9007199254740991::numeric
       )
       OR (p_retirada ->> 'evento_ref') !~
          '^evento_gobi_o404b_[a-f0-9]{32}$'
       OR vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
              p_retirada ->> 'retirada_en',
              false
          ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'retirada de gobierno O4-04B no autorizada';
    END IF;
    BEGIN
        v_retirada_en := (p_retirada ->> 'retirada_en')::timestamptz;
    EXCEPTION WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'instante de retirada O4-04B inválido';
    END;
    v_secuencia := (p_retirada ->> 'secuencia')::bigint;
    v_evento_ref := p_retirada ->> 'evento_ref';
    v_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(p_retirada::text, 'UTF8')
    ), 'hex');
    IF pg_catalog.date_trunc('microseconds', v_retirada_en) <>
          v_retirada_en
       OR v_retirada_en > v_ahora THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'instante de retirada O4-04B no confiable';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_contratacion_temporal:o4_04:migraciones', 0
        )
    );
    SELECT * INTO STRICT v_checkpoint
      FROM vec_contratacion_temporal.gobi_o404b_checkpoint
     WHERE control FOR UPDATE;
    IF v_secuencia <= v_checkpoint.ultima_secuencia THEN
        IF EXISTS (
            SELECT 1 FROM vec_contratacion_temporal.gobi_o404b_evento e
             WHERE e.secuencia = v_secuencia
               AND e.evento_ref = v_evento_ref
               AND e.tipo = 'retirada'
               AND e.huella_evento_sha256 = v_huella
               AND e.contenido_evento = p_retirada
        ) THEN
            RETURN QUERY SELECT 'repetida'::text, v_evento_ref, v_huella;
            RETURN;
        END IF;
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'replay divergente de retirada O4-04B';
    END IF;
    IF v_secuencia <> v_checkpoint.ultima_secuencia + 1
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.gobi_o404b_actual x
             JOIN vec_contratacion_temporal.gobi_o404b_actuacion a
               ON a.referencia = x.actuacion_ref
              AND a.version = x.actuacion_version
              AND a.huella_sha256 = x.actuacion_huella_sha256
            WHERE x.organizacion_ref =
                  p_retirada ->> 'organizacion_ref'
              AND x.accion = p_retirada ->> 'accion'
              AND x.actuacion_ref = p_retirada ->> 'actuacion_ref'
              AND x.actuacion_version =
                  (p_retirada ->> 'actuacion_version')::numeric
              AND x.actuacion_huella_sha256 =
                  p_retirada ->> 'actuacion_huella_sha256'
              AND a.publicada_en <= v_retirada_en
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'retirada O4-04B no contigua o no actual';
    END IF;
    INSERT INTO vec_contratacion_temporal.gobi_o404b_evento VALUES (
        v_secuencia, v_evento_ref, 'retirada', v_huella,
        p_retirada, v_ahora
    );
    INSERT INTO vec_contratacion_temporal.gobi_o404b_retirada VALUES (
        p_retirada ->> 'organizacion_ref',
        p_retirada ->> 'accion',
        p_retirada ->> 'actuacion_ref',
        (p_retirada ->> 'actuacion_version')::numeric,
        p_retirada ->> 'actuacion_huella_sha256',
        v_retirada_en, v_secuencia, v_evento_ref
    );
    UPDATE vec_contratacion_temporal.gobi_o404b_checkpoint SET
        ultima_secuencia = v_secuencia,
        ultimo_evento_ref = v_evento_ref,
        ultima_huella_evento_sha256 = v_huella,
        actualizado_en = v_ahora
     WHERE control;
    RETURN QUERY SELECT 'retirada'::text, v_evento_ref, v_huella;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_resolver(
    p_organizacion_ref text,
    p_expediente_ref text,
    p_version_expediente numeric,
    p_accion text,
    p_instante timestamptz
)
RETURNS TABLE (
    catalogo_json text,
    politica_json text,
    actuacion_json text
)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER'
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_entorno_valido(true)
       OR p_organizacion_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_expediente_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_version_expediente NOT BETWEEN
          1 AND 9007199254740990::numeric
       OR p_version_expediente <> pg_catalog.trunc(p_version_expediente)
       OR p_accion NOT IN (
          'contratacion_temporal.cobertura.decidir',
          'contratacion_temporal.cobertura.rectificar'
       )
       OR p_instante IS NULL
       OR pg_catalog.date_trunc('microseconds', p_instante) <> p_instante
       OR p_instante > pg_catalog.clock_timestamp() THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'resolución de gobierno O4-04B no autorizada';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_contratacion_temporal:o4_04:migraciones', 0
        )
    );
    RETURN QUERY
    SELECT c.publicacion_json::text,
           d.publicacion_json::text,
           a.publicacion_json::text
      FROM vec_contratacion_temporal.gobi_o404b_checkpoint k
      JOIN vec_contratacion_temporal.gobi_o404b_actual x
        ON x.secuencia <= k.ultima_secuencia
      JOIN vec_contratacion_temporal.gobi_o404b_actuacion a
        ON a.referencia = x.actuacion_ref
       AND a.version = x.actuacion_version
       AND a.huella_sha256 = x.actuacion_huella_sha256
      JOIN vec_contratacion_temporal.gobi_o404b_politica d
        ON d.referencia = a.politica_ref
       AND d.version = a.politica_version
       AND d.huella_sha256 = a.politica_huella_sha256
      JOIN vec_contratacion_temporal.gobi_o404b_catalogo c
        ON c.referencia = a.catalogo_ref
       AND c.version = a.catalogo_version
       AND c.huella_sha256 = a.catalogo_huella_sha256
     WHERE k.control
       AND x.organizacion_ref = p_organizacion_ref
       AND x.accion = p_accion
       AND a.organizacion_ref = p_organizacion_ref
       AND a.publicada_en <= p_instante
       AND a.vigente_desde <= p_instante
       AND p_instante < a.vigente_hasta
       AND d.publicada_en <= p_instante
       AND d.vigente_desde <= p_instante
       AND p_instante < d.vigente_hasta
       AND c.publicado_en <= p_instante
       AND c.vigente_desde <= p_instante
       AND (c.vigente_hasta IS NULL OR p_instante < c.vigente_hasta)
       AND NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.gobi_o404b_retirada r
            WHERE r.organizacion_ref = x.organizacion_ref
              AND r.accion = x.accion
              AND r.actuacion_ref = x.actuacion_ref
              AND r.actuacion_version = x.actuacion_version
              AND r.actuacion_huella_sha256 =
                  x.actuacion_huella_sha256
              AND r.retirada_en <= p_instante
       );
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_revalidar_actual(
    p_organizacion_ref text,
    p_accion text,
    p_catalogo_ref text,
    p_catalogo_version numeric,
    p_catalogo_huella text,
    p_politica_ref text,
    p_politica_version numeric,
    p_politica_huella text,
    p_actuacion_ref text,
    p_actuacion_version numeric,
    p_actuacion_huella text
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_ahora timestamptz(6);
    v_secuencia bigint;
BEGIN
    IF current_user <>
          'vec_contratacion_temporal_propietario'
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_entorno_valido(false)
       OR p_organizacion_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_accion NOT IN (
          'contratacion_temporal.cobertura.decidir',
          'contratacion_temporal.cobertura.rectificar'
       )
       OR p_catalogo_huella !~ '^[a-f0-9]{64}$'
       OR p_politica_huella !~ '^[a-f0-9]{64}$'
       OR p_actuacion_huella !~ '^[a-f0-9]{64}$' THEN
        RETURN false;
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_contratacion_temporal:o4_04:migraciones', 0
        )
    );
    SELECT ultima_secuencia INTO STRICT v_secuencia
      FROM vec_contratacion_temporal.gobi_o404b_checkpoint
     WHERE control FOR SHARE;
    v_ahora := pg_catalog.clock_timestamp();
    PERFORM 1
      FROM vec_contratacion_temporal.gobi_o404b_actual x
      JOIN vec_contratacion_temporal.gobi_o404b_actuacion a
        ON a.referencia = x.actuacion_ref
       AND a.version = x.actuacion_version
       AND a.huella_sha256 = x.actuacion_huella_sha256
      JOIN vec_contratacion_temporal.gobi_o404b_politica d
        ON d.referencia = a.politica_ref
       AND d.version = a.politica_version
       AND d.huella_sha256 = a.politica_huella_sha256
      JOIN vec_contratacion_temporal.gobi_o404b_catalogo c
        ON c.referencia = a.catalogo_ref
       AND c.version = a.catalogo_version
       AND c.huella_sha256 = a.catalogo_huella_sha256
     WHERE x.organizacion_ref = p_organizacion_ref
       AND x.accion = p_accion
       AND x.secuencia <= v_secuencia
       AND c.referencia = p_catalogo_ref
       AND c.version = p_catalogo_version
       AND c.huella_sha256 = p_catalogo_huella
       AND d.referencia = p_politica_ref
       AND d.version = p_politica_version
       AND d.huella_sha256 = p_politica_huella
       AND a.referencia = p_actuacion_ref
       AND a.version = p_actuacion_version
       AND a.huella_sha256 = p_actuacion_huella
       AND a.publicada_en <= v_ahora
       AND a.vigente_desde <= v_ahora AND v_ahora < a.vigente_hasta
       AND d.publicada_en <= v_ahora
       AND d.vigente_desde <= v_ahora AND v_ahora < d.vigente_hasta
       AND c.publicado_en <= v_ahora
       AND c.vigente_desde <= v_ahora
       AND (c.vigente_hasta IS NULL OR v_ahora < c.vigente_hasta)
       AND NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.gobi_o404b_retirada r
            WHERE r.organizacion_ref = x.organizacion_ref
              AND r.accion = x.accion
              AND r.actuacion_ref = x.actuacion_ref
              AND r.actuacion_version = x.actuacion_version
              AND r.actuacion_huella_sha256 =
                  x.actuacion_huella_sha256
              AND r.retirada_en <= v_ahora
       )
     FOR SHARE OF x, a, d, c;
    RETURN FOUND;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.gobi_o404b_politica_ligada(jsonb, jsonb),
    vec_contratacion_temporal.gobi_o404b_publicar(jsonb),
    vec_contratacion_temporal.gobi_o404b_retirar(jsonb),
    vec_contratacion_temporal.gobi_o404b_resolver(
        text, text, numeric, text, timestamptz
    ),
    vec_contratacion_temporal.gobi_o404b_revalidar_actual(
        text, text, text, numeric, text, text, numeric, text,
        text, numeric, text
    )
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_migrador,
     vec_contratacion_temporal_gobernador;
GRANT USAGE ON SCHEMA vec_contratacion_temporal
TO vec_contratacion_temporal_ejecutor,
   vec_contratacion_temporal_gobernador;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.gobi_o404b_publicar(jsonb)
TO vec_contratacion_temporal_gobernador;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.gobi_o404b_retirar(jsonb)
TO vec_contratacion_temporal_gobernador;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.gobi_o404b_resolver(
        text, text, numeric, text, timestamptz
    )
TO vec_contratacion_temporal_ejecutor;

COMMIT;
