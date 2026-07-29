\set ON_ERROR_STOP on
-- CT-000044A: datos sintéticos y contrato focal del control causal.

SET search_path = pg_catalog;

SET ROLE vec_contratacion_temporal_propietario;

DO $contrato$
DECLARE
    v_familia record;
    v_estado vec_contratacion_temporal
        .estado_cursor_entrada_cuadro_rrhh_v1;
    v_resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_material vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
    v_salida vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1;
    v_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1;
    v_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_token text;
    v_token_ensayo text;
    v_unicos_token integer;
    v_unicas_familias integer;
BEGIN
    IF (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal
               .control_causal_familia_cursor_rrhh
         WHERE familia_ref LIKE 'familia:cursor:rrhh:000000000000000000000000000000%'
           AND revision = 0
    ) <> 5 OR (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh
         WHERE familia_ref LIKE
               'familia:cursor:rrhh:000000000000000000000000000000%'
    ) <> 5 OR (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal.cursor_cuadro_rrhh
         WHERE familia_ref LIKE
               'familia:cursor:rrhh:000000000000000000000000000000%'
           AND pagina = 2
    ) <> 5 OR (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal.alcance_acceso_rrhh
         WHERE familia_ref LIKE
               'familia:cursor:rrhh:000000000000000000000000000000%'
    ) <> 5 THEN
        RAISE EXCEPTION 'datos causales CT44A incompletos';
    END IF;
    SELECT * INTO STRICT v_familia
      FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh
     WHERE familia_ref =
           'familia:cursor:rrhh:00000000000000000000000000000065';
    v_token := pg_catalog.rtrim(pg_catalog.translate(
        pg_catalog.encode(pg_catalog.decode(
            pg_catalog.repeat('65', 32), 'hex'
        ), 'base64'), '+/', '-_'
    ), E'=\n');
    v_alcance := ROW(
        v_familia.organizacion_ref, v_familia.clase_ambito,
        v_familia.ambito_ref
    )::vec_contratacion_temporal.alcance_consulta_rrhh_v1;
    v_consulta := ROW('', '', '', 2, v_token)::
        vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_estado :=
        vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v1(
            v_alcance, v_consulta,
            v_familia.actor_ref, v_familia.perfil_ref,
            v_familia.perfil_version, v_familia.sesion_ref,
            v_familia.sesion_huella_sha256
        );
    IF NOT v_estado.es_continuacion
       OR v_estado.familia_ref IS DISTINCT FROM v_familia.familia_ref
       OR v_estado.pagina_presentada IS DISTINCT FROM 2 THEN
        RAISE EXCEPTION 'resolución causal CT44A incorrecta';
    END IF;
    BEGIN
        PERFORM
            vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v1(
                v_alcance, v_consulta, 'actor:rrhh:distinto',
                v_familia.perfil_ref, v_familia.perfil_version,
                v_familia.sesion_ref, v_familia.sesion_huella_sha256
            );
        RAISE EXCEPTION 'actor cruzado CT44A aceptado';
    EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
    END;
    BEGIN
        PERFORM
            vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v1(
                v_alcance, v_consulta, v_familia.actor_ref,
                'perfil:rrhh:distinto', v_familia.perfil_version,
                v_familia.sesion_ref, v_familia.sesion_huella_sha256
            );
        RAISE EXCEPTION 'perfil cruzado CT44A aceptado';
    EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
    END;
    BEGIN
        PERFORM
            vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v1(
                v_alcance, v_consulta, v_familia.actor_ref,
                v_familia.perfil_ref, v_familia.perfil_version,
                'sesion:rrhh:distinta', v_familia.sesion_huella_sha256
            );
        RAISE EXCEPTION 'sesión cruzada CT44A aceptada';
    EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
    END;
    BEGIN
        PERFORM
            vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v1(
                ROW(
                    'organizacion:rrhh:distinta', 'organizacion',
                    'organizacion:rrhh:distinta'
                )::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
                v_consulta, v_familia.actor_ref, v_familia.perfil_ref,
                v_familia.perfil_version, v_familia.sesion_ref,
                v_familia.sesion_huella_sha256
            );
        RAISE EXCEPTION 'alcance cruzado CT44A aceptado';
    EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
    END;
    BEGIN
        PERFORM
            vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v1(
                v_alcance,
                ROW('otro', '', '', 2, v_token)::
                    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
                v_familia.actor_ref, v_familia.perfil_ref,
                v_familia.perfil_version, v_familia.sesion_ref,
                v_familia.sesion_huella_sha256
            );
        RAISE EXCEPTION 'filtro cruzado CT44A aceptado';
    EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
    END;
    BEGIN
        PERFORM
            vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v1(
                v_alcance,
                ROW('', '', '', 3, v_token)::
                    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
                v_familia.actor_ref, v_familia.perfil_ref,
                v_familia.perfil_version, v_familia.sesion_ref,
                v_familia.sesion_huella_sha256
            );
        RAISE EXCEPTION 'límite cruzado CT44A aceptado';
    EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
    END;

    SELECT publicada.expediente_ref, publicada.organizacion_ref,
        publicada.numero_visible, publicada.version, publicada.flujo_ref,
        publicada.flujo_version, publicada.flujo_huella_sha256,
        publicada.fase_clave, publicada.estado_clave,
        publicada.centro_ref, publicada.categoria_ref,
        COALESCE(publicada.modalidad_clave, ''),
        COALESCE(publicada.unidad_ref, ''),
        publicada.creado_en, publicada.actualizado_en
      INTO STRICT v_resumen
      FROM vec_contratacion_temporal.publicacion_version_rrhh publicada
     ORDER BY publicada.corte_global
     LIMIT 1;
    v_estado := ROW(
        false, NULL, 1, 0, NULL, NULL, NULL, NULL, NULL, NULL, NULL
    )::vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
    v_material := ROW(
        ARRAY[v_resumen], true, v_resumen.actualizado_en,
        v_resumen.expediente_ref
    )::vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
    v_token_ensayo := pg_catalog.rtrim(pg_catalog.translate(
        pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.uuid_send(pg_catalog.gen_random_uuid())
            || pg_catalog.uuid_send(pg_catalog.gen_random_uuid())
            || pg_catalog.uuid_send(pg_catalog.gen_random_uuid())
        ), 'base64'), '+/', '-_'
    ), E'=\n');
    IF pg_catalog.octet_length(v_token_ensayo) <> 43
       OR v_token_ensayo !~ '^[A-Za-z0-9_-]{43}$' THEN
        RAISE EXCEPTION 'CSPRNG CT44A incompatible';
    END IF;
    SELECT pg_catalog.count(DISTINCT muestra.token_huella),
           pg_catalog.count(DISTINCT muestra.familia_huella)
      INTO STRICT v_unicos_token, v_unicas_familias
      FROM (
          SELECT pg_catalog.sha256(
                     pg_catalog.uuid_send(pg_catalog.gen_random_uuid())
                     || pg_catalog.uuid_send(
                         pg_catalog.gen_random_uuid()
                     )
                     || pg_catalog.uuid_send(
                         pg_catalog.gen_random_uuid()
                     )
                 ) AS token_huella,
                 pg_catalog.substr(pg_catalog.sha256(
                     pg_catalog.uuid_send(pg_catalog.gen_random_uuid())
                     || pg_catalog.uuid_send(
                         pg_catalog.gen_random_uuid()
                     )
                 ), 1, 16) AS familia_huella
            FROM pg_catalog.generate_series(1, 64)
      ) muestra;
    IF v_unicos_token <> 64 OR v_unicas_familias <> 64 THEN
        RAISE EXCEPTION 'unicidad CSPRNG CT44A incompatible';
    END IF;
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario' THEN
        RAISE EXCEPTION 'propietario CT44A incorrecto';
    ELSIF v_estado IS NULL OR v_material IS NULL THEN
        RAISE EXCEPTION 'compuesto inicial CT44A nulo';
    ELSIF pg_catalog.cardinality(v_material.resumenes) <> 1
       OR pg_catalog.array_ndims(v_material.resumenes) <> 1
       OR pg_catalog.array_lower(v_material.resumenes, 1) <> 1
       OR pg_catalog.array_upper(v_material.resumenes, 1) <> 1 THEN
        RAISE EXCEPTION 'array inicial CT44A no canónico';
    ELSIF EXISTS (
        SELECT 1
          FROM pg_catalog.unnest(v_material.resumenes) AS resumen
         WHERE resumen IS NULL
    ) THEN
        RAISE EXCEPTION 'array inicial CT44A contiene nulo';
    ELSIF v_estado.es_continuacion IS DISTINCT FROM false
       OR v_estado.corte_global IS DISTINCT FROM 1
       OR v_estado.pagina_presentada IS DISTINCT FROM 0
       OR v_estado.familia_ref IS NOT NULL
       OR v_estado.token_presentado_huella_sha256 IS NOT NULL
       OR v_estado.acceso_emision_ref IS NOT NULL
       OR v_estado.cursor_emitida_en IS NOT NULL
       OR v_estado.familia_creada_en IS NOT NULL
       OR v_estado.familia_valida_hasta IS NOT NULL
       OR v_estado.ultimo_actualizado_en IS NOT NULL
       OR v_estado.ultimo_expediente_ref IS NOT NULL THEN
        RAISE EXCEPTION 'estado inicial CT44A no canónico';
    ELSIF v_material.hay_mas IS DISTINCT FROM true THEN
        RAISE EXCEPTION 'indicador de continuación CT44A incorrecto';
    ELSIF v_material.ultimo_actualizado_en IS NULL THEN
        RAISE EXCEPTION 'instante final CT44A ausente';
    ELSIF v_material.ultimo_expediente_ref IS NULL THEN
        RAISE EXCEPTION 'expediente final CT44A ausente';
    ELSIF v_resumen.actualizado_en IS DISTINCT FROM
          v_material.ultimo_actualizado_en
       OR v_resumen.expediente_ref IS DISTINCT FROM
          v_material.ultimo_expediente_ref THEN
        RAISE EXCEPTION 'clave final CT44A incoherente';
    END IF;
    v_salida :=
        vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
            v_estado, v_material
        );
    IF NOT v_salida.hay_mas
       OR pg_catalog.octet_length(v_salida.cursor_siguiente) <> 43
       OR pg_catalog.octet_length(v_salida.cursor_huella) <> 32
       OR pg_catalog.encode(v_salida.cursor_huella, 'hex')
          IS DISTINCT FROM v_salida.token_nuevo_huella_sha256
       OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
              v_salida.cursor_siguiente, 'UTF8'
          )), 'hex') IS DISTINCT FROM v_salida.token_nuevo_huella_sha256
       OR v_salida.pagina_nueva IS DISTINCT FROM 2
       OR v_salida.padre_token_huella_sha256 IS NOT NULL THEN
        RAISE EXCEPTION 'salida opaca CT44A incorrecta';
    END IF;
    IF EXISTS (
        SELECT 1 FROM (
            SELECT pg_catalog.to_jsonb(t)::text AS fila
              FROM vec_contratacion_temporal
                   .control_causal_familia_cursor_rrhh t
            UNION ALL SELECT pg_catalog.to_jsonb(t)::text
              FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh t
            UNION ALL SELECT pg_catalog.to_jsonb(t)::text
              FROM vec_contratacion_temporal.cursor_cuadro_rrhh t
            UNION ALL SELECT pg_catalog.to_jsonb(t)::text
              FROM vec_contratacion_temporal.consumo_cursor_cuadro_rrhh t
            UNION ALL SELECT pg_catalog.to_jsonb(t)::text
              FROM vec_contratacion_temporal
                   .revocacion_familia_cursor_rrhh t
        ) historia
        WHERE pg_catalog.strpos(
            historia.fila, v_salida.cursor_siguiente
        ) > 0
    ) THEN
        RAISE EXCEPTION 'token CT44A persistido en claro';
    END IF;

    v_material := ROW(
        ARRAY[v_resumen], false, v_resumen.actualizado_en,
        v_resumen.expediente_ref
    )::vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
    v_salida :=
        vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
            v_estado, v_material
        );
    IF v_salida.hay_mas OR v_salida.cursor_siguiente <> ''
       OR pg_catalog.octet_length(v_salida.cursor_huella) <> 0
       OR v_salida.pagina_nueva <> 0 THEN
        RAISE EXCEPTION 'salida terminal CT44A incorrecta';
    END IF;

    BEGIN
        v_material := ROW(
            pg_catalog.array_fill(v_resumen, ARRAY[101]), true,
            v_resumen.actualizado_en, v_resumen.expediente_ref
        )::vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
        PERFORM
            vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
                v_estado, v_material
            );
        RAISE EXCEPTION 'cardinalidad hostil CT44A aceptada';
    EXCEPTION WHEN SQLSTATE '42501' THEN
        NULL;
    END;
    BEGIN
        v_material := ROW(
            pg_catalog.array_fill(v_resumen, ARRAY[1, 1]), true,
            v_resumen.actualizado_en, v_resumen.expediente_ref
        )::vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
        PERFORM
            vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
                v_estado, v_material
            );
        RAISE EXCEPTION 'array hostil CT44A aceptado';
    EXCEPTION WHEN SQLSTATE '42501' THEN
        NULL;
    END;
    BEGIN
        v_material := ROW(
            pg_catalog.array_fill(v_resumen, ARRAY[1], ARRAY[0]), true,
            v_resumen.actualizado_en, v_resumen.expediente_ref
        )::vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
        PERFORM
            vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
                v_estado, v_material
            );
        RAISE EXCEPTION 'límite inferior hostil CT44A aceptado';
    EXCEPTION WHEN SQLSTATE '42501' THEN
        NULL;
    END;
    BEGIN
        v_material := ROW(
            ARRAY[v_resumen, NULL, v_resumen], true,
            v_resumen.actualizado_en, v_resumen.expediente_ref
        )::vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
        PERFORM
            vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
                v_estado, v_material
            );
        RAISE EXCEPTION 'nulo intermedio CT44A aceptado';
    EXCEPTION WHEN SQLSTATE '42501' THEN
        NULL;
    END;
    BEGIN
        v_material := ROW(
            ARRAY[]::vec_contratacion_temporal
                .resumen_publicacion_rrhh_v1[],
            true, NULL, NULL
        )::vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
        PERFORM
            vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
                v_estado, v_material
            );
        RAISE EXCEPTION 'página vacía con continuación CT44A aceptada';
    EXCEPTION WHEN SQLSTATE '42501' THEN
        NULL;
    END;
    BEGIN
        v_estado := ROW(
            true,
            'familia:cursor:rrhh:00000000000000000000000000000065',
            1, 2, pg_catalog.repeat('a', 64), NULL, NULL, NULL, NULL,
            NULL, NULL
        )::vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
        v_material := ROW(
            ARRAY[v_resumen], true, v_resumen.actualizado_en,
            v_resumen.expediente_ref
        )::vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
        PERFORM
            vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
                v_estado, v_material
            );
        RAISE EXCEPTION 'continuación incompleta CT44A aceptada';
    EXCEPTION WHEN SQLSTATE '42501' THEN
        NULL;
    END;
    BEGIN
        v_estado := ROW(
            true,
            'familia:cursor:rrhh:00000000000000000000000000000065',
            1, 9007199254740991::numeric, pg_catalog.repeat('a', 64),
            'acceso:rrhh:00000000000000000000000000000001',
            pg_catalog.date_trunc(
                'microseconds', pg_catalog.clock_timestamp()
            ),
            pg_catalog.date_trunc(
                'microseconds', pg_catalog.clock_timestamp()
            ),
            pg_catalog.date_trunc(
                'microseconds',
                pg_catalog.clock_timestamp() + interval '1 minute'
            ),
            pg_catalog.date_trunc(
                'microseconds', pg_catalog.clock_timestamp()
            ),
            'expediente:ct44a:limite'
        )::vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
        v_material := ROW(
            ARRAY[v_resumen], true, v_resumen.actualizado_en,
            v_resumen.expediente_ref
        )::vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
        PERFORM
            vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
                v_estado, v_material
            );
        RAISE EXCEPTION 'desbordamiento de página CT44A aceptado';
    EXCEPTION WHEN SQLSTATE '42501' THEN
        NULL;
    END;
END
$contrato$;

RESET ROLE;
