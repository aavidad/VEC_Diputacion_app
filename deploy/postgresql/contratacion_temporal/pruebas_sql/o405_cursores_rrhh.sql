\set ON_ERROR_STOP on

CREATE FUNCTION pg_temp.prueba_alcance_rrhh(
    p_acceso text,
    p_tipo text,
    p_familia text,
    p_organizacion text,
    p_clase text,
    p_ambito text,
    p_actor text,
    p_perfil text,
    p_perfil_version numeric,
    p_sesion text,
    p_sesion_huella text,
    p_registrado timestamptz
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.convert_to(
               'VEC-CT-ALCANCE-ACCESO-RRHH-V1' || pg_catalog.chr(10),
               'UTF8'
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_acceso)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_tipo)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               COALESCE(p_familia, '')
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_organizacion)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_clase)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_ambito)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_actor)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_perfil)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               p_perfil_version::text
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_sesion)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_sesion_huella)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(p_registrado)
           )
$funcion$;

CREATE FUNCTION pg_temp.prueba_familia_cursor_rrhh(
    p_familia text,
    p_organizacion text,
    p_clase text,
    p_ambito text,
    p_actor text,
    p_perfil text,
    p_perfil_version numeric,
    p_sesion text,
    p_sesion_huella text,
    p_dominio text,
    p_filtros_huella text,
    p_limite smallint,
    p_corte numeric,
    p_creada timestamptz,
    p_valida_hasta timestamptz,
    p_acceso text
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.convert_to(
               'VEC-CT-FAMILIA-CURSOR-CUADRO-RRHH-V1'
                   || pg_catalog.chr(10), 'UTF8'
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_familia)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_organizacion)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_clase)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_ambito)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_actor)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_perfil)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               p_perfil_version::text
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_sesion)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_sesion_huella)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_dominio)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_filtros_huella)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_limite::text)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_corte::text)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(p_creada)
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(p_valida_hasta)
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_acceso)
$funcion$;

CREATE FUNCTION pg_temp.prueba_cursor_rrhh(
    p_token_huella text,
    p_familia text,
    p_padre text,
    p_pagina numeric,
    p_padre_emitida timestamptz,
    p_ultimo_actualizado timestamptz,
    p_ultimo_expediente text,
    p_familia_creada timestamptz,
    p_familia_valida timestamptz,
    p_emitida timestamptz,
    p_acceso text
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.convert_to(
               'VEC-CT-CURSOR-CUADRO-RRHH-V1' || pg_catalog.chr(10),
               'UTF8'
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_token_huella)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_familia)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               COALESCE(p_padre, '')
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_pagina::text)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               CASE
                   WHEN p_padre_emitida IS NULL THEN ''
                   ELSE vec_contratacion_temporal.instante_utc_v1(
                       p_padre_emitida
                   )
               END
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(
                   p_ultimo_actualizado
               )
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               p_ultimo_expediente
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(
                   p_familia_creada
               )
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(
                   p_familia_valida
               )
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(p_emitida)
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_acceso)
$funcion$;

CREATE FUNCTION pg_temp.prueba_consumo_cursor_rrhh(
    p_token_huella text,
    p_familia text,
    p_decision text,
    p_decision_huella text,
    p_consumo_huella text,
    p_acceso_emision text,
    p_acceso_consumo text,
    p_cursor_emitida timestamptz,
    p_familia_valida timestamptz,
    p_consumido timestamptz
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.convert_to(
               'VEC-CT-CONSUMO-CURSOR-CUADRO-RRHH-V1'
                   || pg_catalog.chr(10), 'UTF8'
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_token_huella)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_familia)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_decision)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               p_decision_huella
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               p_consumo_huella
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_acceso_emision)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_acceso_consumo)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(
                   p_cursor_emitida
               )
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(
                   p_familia_valida
               )
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(p_consumido)
           )
$funcion$;

CREATE FUNCTION pg_temp.prueba_revocacion_cursor_rrhh(
    p_familia text,
    p_familia_creada timestamptz,
    p_decision text,
    p_decision_huella text,
    p_auditoria text,
    p_auditoria_huella text,
    p_motivo text,
    p_motivo_version numeric,
    p_motivo_huella text,
    p_revocada timestamptz
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.convert_to(
               'VEC-CT-REVOCACION-FAMILIA-CURSOR-RRHH-V1'
                   || pg_catalog.chr(10), 'UTF8'
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_familia)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(
                   p_familia_creada
               )
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_decision)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               p_decision_huella
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_auditoria)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               p_auditoria_huella
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_motivo)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               p_motivo_version::text
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_motivo_huella)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(p_revocada)
           )
$funcion$;

CREATE FUNCTION pg_temp.exigir_rechazo_cursor_rrhh(
    p_sentencia text,
    p_descripcion text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SET search_path = pg_catalog
AS $funcion$
BEGIN
    BEGIN
        EXECUTE p_sentencia;
        SET CONSTRAINTS ALL IMMEDIATE;
    EXCEPTION
        WHEN check_violation OR foreign_key_violation
          OR unique_violation OR not_null_violation THEN
            SET CONSTRAINTS ALL DEFERRED;
            RETURN;
    END;
    RAISE EXCEPTION 'caso hostil aceptado: %', p_descripcion;
END
$funcion$;

CREATE FUNCTION pg_temp.registro_cuadro_rrhh(p_indice integer)
RETURNS jsonb
LANGUAGE sql
STABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.jsonb_build_object(
        'accion', 'contratacion_temporal.cuadro.consultar',
        'actor_ref', base.actor_ref,
        'ambito_ref', base.ambito_ref,
        'audiencia',
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
        'auditoria_vec_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 80), 64, '8'),
        'auditoria_vec_ref', 'auditoria:cursor:rrhh:' || p_indice::text,
        'capacidad_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 50), 64, '5'),
        'consumo_vec_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 60), 64, '6'),
        'correlacion_ref', 'correlacion:cursor:rrhh:' || p_indice::text,
        'decision_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 10), 64, '1'),
        'decision_ref', 'decision:cursor:rrhh:' || p_indice::text,
        'dominio_huella_consulta',
        'vec.contratacion_temporal.consulta_rrhh.cuadro.v1',
        'expediente_ref', NULL,
        'finalidad', 'gestion_operativa_contratacion_temporal',
        'modulo_id', 'contratacion_temporal',
        'organizacion_ref', base.organizacion_ref,
        'perfil_id', base.perfil_id,
        'perfil_version', base.perfil_version,
        'recurso_ref', base.ambito_ref,
        'recurso_tipo', 'cuadro_rrhh_contratacion_temporal',
        'resultado_generico', 'entregado',
        'resultado_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 70), 64, '7'),
        'sesion_huella_sha256', base.sesion_huella_sha256,
        'sesion_id', base.sesion_id,
        'tipo_consulta', 'cuadro',
        'total', 1,
        'version_expediente', NULL,
        'consulta_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 40), 64, '4')
    )
      FROM vec_contratacion_temporal.registro_acceso_rrhh base
     WHERE base.actor_ref = 'actor:rrhh:1'
     ORDER BY base.registrada_en, base.acceso_ref
     LIMIT 1
$funcion$;

\if :preparar
SELECT vec_contratacion_temporal.registrar_acceso_rrhh_interno_v1(
    pg_temp.registro_cuadro_rrhh(indice)
) FROM (VALUES (3), (4), (5)) AS casos(indice);
\else

DO $estructura$
DECLARE
    v_tabla text;
    v_rol text;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 18
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 2
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_cursores_cuadro_rrhh
         WHERE control AND version_esquema = 1
           AND reloj = pg_catalog.date_trunc('microseconds', reloj)
    ) OR (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal.control_cursores_cuadro_rrhh
    ) <> 1 THEN
        RAISE EXCEPTION 'barreras o singleton de cursores divergentes';
    END IF;

    FOREACH v_tabla IN ARRAY ARRAY[
        'control_cursores_cuadro_rrhh',
        'alcance_acceso_rrhh',
        'familia_cursor_cuadro_rrhh',
        'cursor_cuadro_rrhh',
        'consumo_cursor_cuadro_rrhh',
        'revocacion_familia_cursor_rrhh'
    ]::text[] LOOP
        IF NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_class tabla
              JOIN pg_catalog.pg_namespace esquema
                ON esquema.oid = tabla.relnamespace
              JOIN pg_catalog.pg_roles propietario
                ON propietario.oid = tabla.relowner
             WHERE esquema.nspname = 'vec_contratacion_temporal'
               AND tabla.relname = v_tabla
               AND propietario.rolname =
                   'vec_contratacion_temporal_propietario'
               AND tabla.relrowsecurity
               AND tabla.relforcerowsecurity
        ) OR (
            SELECT pg_catalog.count(*)
              FROM pg_catalog.pg_policies
             WHERE schemaname = 'vec_contratacion_temporal'
               AND tablename = v_tabla
               AND policyname = 'propietario_total'
               AND roles =
                   ARRAY['vec_contratacion_temporal_propietario']::name[]
        ) <> 1 THEN
            RAISE EXCEPTION 'propiedad o RLS incorrecta en %', v_tabla;
        END IF;
    END LOOP;

    FOREACH v_rol IN ARRAY ARRAY[
        'public',
        'vec_contratacion_temporal_migrador',
        'vec_contratacion_temporal_ejecutor',
        'vec_contratacion_temporal_gobernador',
        'vec_contratacion_temporal_confirmador_cobertura',
        'vec_contratacion_temporal_lector_resultado_cobertura',
        'vec_contratacion_temporal_consultor_rrhh'
    ]::text[] LOOP
        FOREACH v_tabla IN ARRAY ARRAY[
            'control_cursores_cuadro_rrhh',
            'alcance_acceso_rrhh',
            'familia_cursor_cuadro_rrhh',
            'cursor_cuadro_rrhh',
            'consumo_cursor_cuadro_rrhh',
            'revocacion_familia_cursor_rrhh'
        ]::text[] LOOP
            IF pg_catalog.has_table_privilege(
                v_rol,
                pg_catalog.format(
                    'vec_contratacion_temporal.%I', v_tabla
                ),
                'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
            ) THEN
                RAISE EXCEPTION 'ACL excesiva para % sobre %',
                    v_rol, v_tabla;
            END IF;
        END LOOP;
    END LOOP;

    IF pg_catalog.has_schema_privilege('public', 'public', 'USAGE')
       OR pg_catalog.has_function_privilege(
           'public', 'public.gen_random_bytes(integer)', 'EXECUTE'
       ) OR NOT pg_catalog.has_schema_privilege(
           'vec_contratacion_temporal_propietario', 'public', 'USAGE'
       ) OR NOT pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_propietario',
           'public.gen_random_bytes(integer)', 'EXECUTE'
       ) OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_attribute columna
             JOIN pg_catalog.pg_class tabla
               ON tabla.oid = columna.attrelid
             JOIN pg_catalog.pg_namespace esquema
               ON esquema.oid = tabla.relnamespace
            WHERE esquema.nspname = 'vec_contratacion_temporal'
              AND tabla.relname IN (
                  'familia_cursor_cuadro_rrhh',
                  'cursor_cuadro_rrhh'
              )
              AND columna.attnum > 0
              AND NOT columna.attisdropped
              AND (
                  columna.atttypid IN (
                      'json'::regtype, 'jsonb'::regtype
                  )
                  OR columna.attname::text IN (
                      'token', 'cursor', 'filtros', 'texto',
                      'estado_clave', 'fase_clave'
                  )
              )
       ) THEN
        RAISE EXCEPTION 'CSPRNG, ACL o minimización de cursor incorrecta';
    END IF;

    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_constraint
         WHERE connamespace =
               'vec_contratacion_temporal'::regnamespace
           AND conrelid IN (
               'vec_contratacion_temporal.alcance_acceso_rrhh'::regclass,
               'vec_contratacion_temporal.familia_cursor_cuadro_rrhh'::regclass,
               'vec_contratacion_temporal.cursor_cuadro_rrhh'::regclass,
               'vec_contratacion_temporal.consumo_cursor_cuadro_rrhh'::regclass,
               'vec_contratacion_temporal.revocacion_familia_cursor_rrhh'::regclass
           )
           AND contype = 'f'
    ) < 8 OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class indice
          JOIN pg_catalog.pg_namespace esquema
            ON esquema.oid = indice.relnamespace
         WHERE esquema.nspname = 'vec_contratacion_temporal'
           AND indice.relname = 'familia_cursor_rrhh_caducidad_idx'
    ) OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_trigger
         WHERE NOT tgisinternal
           AND tgrelid IN (
               'vec_contratacion_temporal.alcance_acceso_rrhh'::regclass,
               'vec_contratacion_temporal.familia_cursor_cuadro_rrhh'::regclass,
               'vec_contratacion_temporal.cursor_cuadro_rrhh'::regclass,
               'vec_contratacion_temporal.consumo_cursor_cuadro_rrhh'::regclass,
               'vec_contratacion_temporal.revocacion_familia_cursor_rrhh'::regclass
           )
    ) <> 10 THEN
        RAISE EXCEPTION 'FK, índices o inmutabilidad incompletos';
    END IF;
END
$estructura$;

SET ROLE vec_contratacion_temporal_propietario;

DO $modelo_vacio$
BEGIN
    IF EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.alcance_acceso_rrhh
    ) OR EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh
    ) OR EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.cursor_cuadro_rrhh
    ) OR EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.consumo_cursor_cuadro_rrhh
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.revocacion_familia_cursor_rrhh
    ) THEN
        RAISE EXCEPTION 'el modelo de cursor no comenzó vacío';
    END IF;
END
$modelo_vacio$;

DO $camino_valido$
DECLARE
    v_origen record;
    v_consumo record;
    v_prueba bytea;
    v_familia constant text :=
        'familia:cursor:rrhh:11111111111111111111111111111111';
    v_token constant text := pg_catalog.repeat('a', 64);
    v_valida timestamptz;
BEGIN
    SELECT * INTO STRICT v_origen
      FROM vec_contratacion_temporal.registro_acceso_rrhh
     WHERE actor_ref = 'actor:rrhh:1'
     ORDER BY registrada_en, acceso_ref
     LIMIT 1;
    v_valida := v_origen.registrada_en + interval '5 minutes';

    v_prueba := pg_temp.prueba_alcance_rrhh(
        v_origen.acceso_ref, 'cuadro', v_familia,
        v_origen.organizacion_ref, 'centro', v_origen.ambito_ref,
        v_origen.actor_ref, v_origen.perfil_id, v_origen.perfil_version,
        v_origen.sesion_id, v_origen.sesion_huella_sha256,
        v_origen.registrada_en
    );
    INSERT INTO vec_contratacion_temporal.alcance_acceso_rrhh (
        acceso_ref, tipo_consulta, familia_ref, organizacion_ref,
        clase_ambito, ambito_ref, actor_ref, perfil_ref, perfil_version,
        sesion_ref, sesion_huella_sha256, acceso_registrado_en,
        prueba_canonica, prueba_huella_sha256
    ) VALUES (
        v_origen.acceso_ref, 'cuadro', v_familia,
        v_origen.organizacion_ref, 'centro', v_origen.ambito_ref,
        v_origen.actor_ref, v_origen.perfil_id, v_origen.perfil_version,
        v_origen.sesion_id, v_origen.sesion_huella_sha256,
        v_origen.registrada_en, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
    );

    v_prueba := pg_temp.prueba_familia_cursor_rrhh(
        v_familia, v_origen.organizacion_ref, 'centro',
        v_origen.ambito_ref, v_origen.actor_ref, v_origen.perfil_id,
        v_origen.perfil_version, v_origen.sesion_id,
        v_origen.sesion_huella_sha256,
        'vec.contratacion_temporal.filtros_rrhh.cuadro.v1',
        pg_catalog.repeat('b', 64), 2::smallint, 1,
        v_origen.registrada_en, v_valida, v_origen.acceso_ref
    );
    INSERT INTO vec_contratacion_temporal.familia_cursor_cuadro_rrhh (
        familia_ref, organizacion_ref, clase_ambito, ambito_ref,
        actor_ref, perfil_ref, perfil_version, sesion_ref,
        sesion_huella_sha256, dominio_filtros, filtros_huella_sha256,
        limite, corte_global, creada_en, valida_hasta,
        acceso_origen_ref, prueba_canonica, prueba_huella_sha256
    ) VALUES (
        v_familia, v_origen.organizacion_ref, 'centro',
        v_origen.ambito_ref, v_origen.actor_ref, v_origen.perfil_id,
        v_origen.perfil_version, v_origen.sesion_id,
        v_origen.sesion_huella_sha256,
        'vec.contratacion_temporal.filtros_rrhh.cuadro.v1',
        pg_catalog.repeat('b', 64), 2, 1, v_origen.registrada_en,
        v_valida, v_origen.acceso_ref, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
    );

    v_prueba := pg_temp.prueba_cursor_rrhh(
        v_token, v_familia, NULL, 2, NULL,
        '2026-01-02T00:00:00Z', 'expediente:pub:A',
        v_origen.registrada_en, v_valida, v_origen.registrada_en,
        v_origen.acceso_ref
    );
    INSERT INTO vec_contratacion_temporal.cursor_cuadro_rrhh (
        token_huella_sha256, familia_ref, padre_token_huella_sha256,
        pagina, padre_emitida_en, ultimo_actualizado_en,
        ultimo_expediente_ref, familia_creada_en, familia_valida_hasta,
        emitida_en, acceso_emision_ref, prueba_canonica, prueba_huella_sha256
    ) VALUES (
        v_token, v_familia, NULL, 2, NULL,
        '2026-01-02T00:00:00Z', 'expediente:pub:A',
        v_origen.registrada_en, v_valida, v_origen.registrada_en,
        v_origen.acceso_ref, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
    );

    SELECT * INTO STRICT v_consumo
      FROM vec_contratacion_temporal.registro_acceso_rrhh
     WHERE decision_ref = 'decision:cursor:rrhh:3';
    v_prueba := pg_temp.prueba_alcance_rrhh(
        v_consumo.acceso_ref, 'cuadro', v_familia,
        v_consumo.organizacion_ref, 'centro', v_consumo.ambito_ref,
        v_consumo.actor_ref, v_consumo.perfil_id,
        v_consumo.perfil_version, v_consumo.sesion_id,
        v_consumo.sesion_huella_sha256, v_consumo.registrada_en
    );
    INSERT INTO vec_contratacion_temporal.alcance_acceso_rrhh VALUES (
        v_consumo.acceso_ref, 'cuadro', v_familia,
        v_consumo.organizacion_ref, 'centro', v_consumo.ambito_ref,
        v_consumo.actor_ref, v_consumo.perfil_id,
        v_consumo.perfil_version, v_consumo.sesion_id,
        v_consumo.sesion_huella_sha256, v_consumo.registrada_en,
        v_prueba, pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
    );
    v_prueba := pg_temp.prueba_consumo_cursor_rrhh(
        v_token, v_familia, v_consumo.decision_ref,
        v_consumo.decision_huella_sha256,
        v_consumo.consumo_vec_huella_sha256, v_origen.acceso_ref,
        v_consumo.acceso_ref,
        v_origen.registrada_en, v_valida, v_consumo.registrada_en
    );
    INSERT INTO vec_contratacion_temporal.consumo_cursor_cuadro_rrhh (
        token_huella_sha256, familia_ref, decision_ref,
        decision_huella_sha256, consumo_vec_huella_sha256,
        acceso_emision_ref, acceso_consumo_ref, cursor_emitida_en,
        familia_valida_hasta,
        consumido_en, prueba_canonica, prueba_huella_sha256
    ) VALUES (
        v_token, v_familia, v_consumo.decision_ref,
        v_consumo.decision_huella_sha256,
        v_consumo.consumo_vec_huella_sha256, v_origen.acceso_ref,
        v_consumo.acceso_ref,
        v_origen.registrada_en, v_valida, v_consumo.registrada_en, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
    );

    v_prueba := pg_temp.prueba_revocacion_cursor_rrhh(
        v_familia, v_origen.registrada_en,
        'decision:revocacion:cursor:1',
        pg_catalog.repeat('c', 64), 'auditoria:revocacion:cursor:1',
        pg_catalog.repeat('d', 64), 'motivo:revocacion:cursor:1', 1,
        pg_catalog.repeat('e', 64), v_consumo.registrada_en
    );
    INSERT INTO vec_contratacion_temporal.revocacion_familia_cursor_rrhh (
        familia_ref, familia_creada_en, decision_ref, decision_huella_sha256,
        auditoria_vec_ref, auditoria_vec_huella_sha256,
        motivo_ref, motivo_version, motivo_huella_sha256,
        revocada_en, prueba_canonica, prueba_huella_sha256
    ) VALUES (
        v_familia, v_origen.registrada_en,
        'decision:revocacion:cursor:1',
        pg_catalog.repeat('c', 64), 'auditoria:revocacion:cursor:1',
        pg_catalog.repeat('d', 64), 'motivo:revocacion:cursor:1', 1,
        pg_catalog.repeat('e', 64), v_consumo.registrada_en,
        v_prueba, pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
    );
END
$camino_valido$;

DO $rechazos$
DECLARE
    v_acceso record;
    v_prueba bytea;
    v_sql text;
    v_familia constant text :=
        'familia:cursor:rrhh:22222222222222222222222222222222';
    v_dominio constant text :=
        'vec.contratacion_temporal.filtros_rrhh.cuadro.v1';
    v_creada timestamptz;
    v_campos record;
BEGIN
    SELECT * INTO STRICT v_acceso
      FROM vec_contratacion_temporal.registro_acceso_rrhh
     WHERE decision_ref = 'decision:cursor:rrhh:4';
    v_creada := v_acceso.registrada_en;
    v_prueba := pg_temp.prueba_alcance_rrhh(
        v_acceso.acceso_ref, 'cuadro', NULL,
        v_acceso.organizacion_ref, 'centro', v_acceso.ambito_ref,
        v_acceso.actor_ref, v_acceso.perfil_id, v_acceso.perfil_version,
        v_acceso.sesion_id, v_acceso.sesion_huella_sha256,
        v_acceso.registrada_en
    );
    INSERT INTO vec_contratacion_temporal.alcance_acceso_rrhh (
        acceso_ref, tipo_consulta, familia_ref, organizacion_ref,
        clase_ambito, ambito_ref, actor_ref, perfil_ref, perfil_version,
        sesion_ref, sesion_huella_sha256, acceso_registrado_en,
        prueba_canonica, prueba_huella_sha256
    ) VALUES (
        v_acceso.acceso_ref, 'cuadro', NULL,
        v_acceso.organizacion_ref, 'centro', v_acceso.ambito_ref,
        v_acceso.actor_ref, v_acceso.perfil_id, v_acceso.perfil_version,
        v_acceso.sesion_id, v_acceso.sesion_huella_sha256,
        v_acceso.registrada_en, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
    );

    FOR v_campos IN
        SELECT * FROM (VALUES
          ('externo', v_acceso.organizacion_ref, v_acceso.ambito_ref,
           v_acceso.perfil_version, v_acceso.sesion_huella_sha256,
           2::smallint, 1::numeric, v_creada + interval '5 minutes',
           'clase inválida'),
          ('organizacion', 'organizacion:cruzada:rrhh',
           v_acceso.ambito_ref, v_acceso.perfil_version,
           v_acceso.sesion_huella_sha256, 2::smallint, 1::numeric,
           v_creada + interval '5 minutes', 'organización cruzada'),
          ('organizacion', v_acceso.organizacion_ref,
           'ambito:distinto:organizacion', v_acceso.perfil_version,
           v_acceso.sesion_huella_sha256, 2::smallint, 1::numeric,
           v_creada + interval '5 minutes', 'ámbito de organización cruzado'),
          ('organizacion', v_acceso.organizacion_ref,
           v_acceso.ambito_ref, 0::numeric,
           v_acceso.sesion_huella_sha256, 2::smallint, 1::numeric,
           v_creada + interval '5 minutes', 'perfil cero'),
          ('organizacion', v_acceso.organizacion_ref,
           v_acceso.ambito_ref, 9007199254740992::numeric,
           v_acceso.sesion_huella_sha256, 2::smallint, 1::numeric,
           v_creada + interval '5 minutes', 'perfil fuera de rango'),
          ('organizacion', v_acceso.organizacion_ref,
           v_acceso.ambito_ref, v_acceso.perfil_version,
           pg_catalog.repeat('0', 64), 2::smallint, 1::numeric,
           v_creada + interval '5 minutes', 'huella cero'),
          ('organizacion', v_acceso.organizacion_ref,
           v_acceso.ambito_ref, v_acceso.perfil_version,
           v_acceso.sesion_huella_sha256, 0::smallint, 1::numeric,
           v_creada + interval '5 minutes', 'límite cero'),
          ('organizacion', v_acceso.organizacion_ref,
           v_acceso.ambito_ref, v_acceso.perfil_version,
           v_acceso.sesion_huella_sha256, 101::smallint, 1::numeric,
           v_creada + interval '5 minutes', 'límite superior'),
          ('organizacion', v_acceso.organizacion_ref,
           v_acceso.ambito_ref, v_acceso.perfil_version,
           v_acceso.sesion_huella_sha256, 2::smallint, 0::numeric,
           v_creada + interval '5 minutes', 'corte cero'),
          ('organizacion', v_acceso.organizacion_ref,
           v_acceso.ambito_ref, v_acceso.perfil_version,
           v_acceso.sesion_huella_sha256, 2::smallint, 999::numeric,
           v_creada + interval '5 minutes', 'corte ausente'),
          ('organizacion', v_acceso.organizacion_ref,
           v_acceso.ambito_ref, v_acceso.perfil_version,
           v_acceso.sesion_huella_sha256, 2::smallint, 1::numeric,
           v_creada, 'TTL nulo'),
          ('organizacion', v_acceso.organizacion_ref,
           v_acceso.ambito_ref, v_acceso.perfil_version,
           v_acceso.sesion_huella_sha256, 2::smallint, 1::numeric,
           v_creada + interval '5 minutes 1 microsecond', 'TTL excesivo')
        ) AS casos(
            clase, organizacion, ambito, perfil_version,
            sesion_huella, limite, corte, valida_hasta, descripcion
        )
    LOOP
        v_prueba := pg_temp.prueba_familia_cursor_rrhh(
            v_familia, v_campos.organizacion, v_campos.clase,
            v_campos.ambito, v_acceso.actor_ref, v_acceso.perfil_id,
            v_campos.perfil_version, v_acceso.sesion_id,
            v_campos.sesion_huella, v_dominio, pg_catalog.repeat('f', 64),
            v_campos.limite, v_campos.corte, v_creada,
            v_campos.valida_hasta, v_acceso.acceso_ref
        );
        v_sql := pg_catalog.format(
            'INSERT INTO vec_contratacion_temporal.'
            'familia_cursor_cuadro_rrhh VALUES '
            '(%L,%L,%L,%L,%L,%L,%L,%L,%L,%L,%L,%L,%L,%L,%L,%L,%L,%L)',
            v_familia, v_campos.organizacion, v_campos.clase,
            v_campos.ambito, v_acceso.actor_ref, v_acceso.perfil_id,
            v_campos.perfil_version, v_acceso.sesion_id,
            v_campos.sesion_huella, v_dominio, pg_catalog.repeat('f', 64),
            v_campos.limite, v_campos.corte, v_creada,
            v_campos.valida_hasta, v_acceso.acceso_ref, v_prueba,
            pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
        );
        PERFORM pg_temp.exigir_rechazo_cursor_rrhh(
            v_sql, v_campos.descripcion
        );
    END LOOP;

    v_prueba := pg_temp.prueba_familia_cursor_rrhh(
        v_familia, v_acceso.organizacion_ref, 'organizacion',
        v_acceso.ambito_ref, v_acceso.actor_ref, v_acceso.perfil_id,
        v_acceso.perfil_version, v_acceso.sesion_id,
        v_acceso.sesion_huella_sha256, v_dominio,
        pg_catalog.repeat('f', 64), 2::smallint, 1, v_creada,
        v_creada + interval '5 minutes', 'acceso:rrhh:ausente'
    );
    v_sql := pg_catalog.format(
        'INSERT INTO vec_contratacion_temporal.'
        'familia_cursor_cuadro_rrhh VALUES '
        '(%L,%L,%L,%L,%L,%L,%L,%L,%L,%L,%L,2,1,%L,%L,%L,%L,%L)',
        v_familia, v_acceso.organizacion_ref, 'organizacion',
        v_acceso.ambito_ref, v_acceso.actor_ref, v_acceso.perfil_id,
        v_acceso.perfil_version, v_acceso.sesion_id,
        v_acceso.sesion_huella_sha256, v_dominio,
        pg_catalog.repeat('f', 64), v_creada,
        v_creada + interval '5 minutes', 'acceso:rrhh:ausente',
        v_prueba, pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
    );
    PERFORM pg_temp.exigir_rechazo_cursor_rrhh(v_sql, 'alcance ausente');
END
$rechazos$;

\ir o405_cursores_rrhh_inmutabilidad.sql

\endif
