-- CT-000044B: persistencia privada de los efectos causales del cursor.
--
-- La identidad no entra como argumento. Se relee del alcance durable creado
-- por CT-000043 y se contrasta con el registro de acceso de ese mismo cierre.

CREATE FUNCTION
vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v1(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    p_estado
        vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
    p_salida
        vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1,
    p_consumo
        vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,
    p_decision_canonica bytea,
    p_cierre
        vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
)
RETURNS void
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
    v_alcance
        vec_contratacion_temporal.alcance_acceso_rrhh%ROWTYPE;
    v_registro record;
    v_filtros_huella text;
    v_consulta_huella text;
    v_decision_huella text;
    v_familia_prueba bytea;
    v_cursor_prueba bytea;
    v_consumo_prueba bytea;
    v_valida_hasta timestamptz(6);
    v_padre_emitida_en timestamptz(6);
    v_estado_durable record;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_isolation')
          <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR p_alcance IS NULL
       OR p_consulta IS NULL
       OR p_estado IS NULL
       OR p_salida IS NULL
       OR p_estado.es_continuacion IS NULL
       OR p_salida.hay_mas IS NULL
       OR p_consumo IS NULL
       OR p_decision_canonica IS NULL
       OR p_cierre IS NULL
       OR p_consumo.consumo_nuevo IS DISTINCT FROM true
       OR p_cierre.registrada_en IS NULL
       OR p_cierre.registrada_en <>
          pg_catalog.date_trunc('microseconds', p_cierre.registrada_en)
       OR p_cierre.acceso_ref IS NULL
       OR p_cierre.acceso_ref !~ '^acceso:rrhh:[0-9a-f]{32}$'
       OR p_salida.hay_mas IS DISTINCT FROM
          (p_salida.cursor_siguiente <> '')
       OR (
           NOT p_salida.hay_mas
           AND (
               p_salida.cursor_siguiente IS DISTINCT FROM ''
               OR p_salida.cursor_huella IS DISTINCT FROM ''::bytea
               OR p_salida.familia_ref IS NOT NULL
               OR p_salida.pagina_nueva IS DISTINCT FROM 0
               OR p_salida.token_nuevo_huella_sha256 IS NOT NULL
               OR p_salida.padre_token_huella_sha256 IS NOT NULL
               OR p_salida.ultimo_actualizado_en IS NOT NULL
               OR p_salida.ultimo_expediente_ref IS NOT NULL
           )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'efectos de cursor RRHH rechazados';
    END IF;
    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(p_alcance);
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
        p_consulta
    );

    SELECT alcance.*
      INTO STRICT v_alcance
      FROM vec_contratacion_temporal.alcance_acceso_rrhh alcance
     WHERE alcance.acceso_ref = p_cierre.acceso_ref
       AND alcance.acceso_registrado_en = p_cierre.registrada_en;
    SELECT registro.decision_ref, registro.decision_huella_sha256,
           registro.consulta_huella_sha256,
           registro.consumo_vec_huella_sha256,
           prueba.tipo_consulta AS prueba_tipo_consulta,
           prueba.generada_en AS prueba_generada_en,
           prueba.total AS prueba_total,
           prueba.resumenes AS prueba_resumenes,
           prueba.hay_mas AS prueba_hay_mas,
           COALESCE(
               prueba.cursor_huella_sha256, ''
           ) AS prueba_cursor_huella_sha256
      INTO STRICT v_registro
      FROM vec_contratacion_temporal.registro_acceso_rrhh registro
      JOIN vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 prueba
        ON prueba.acceso_ref = registro.acceso_ref
       AND prueba.registrada_en = registro.registrada_en
     WHERE registro.acceso_ref = p_cierre.acceso_ref
       AND registro.registrada_en = p_cierre.registrada_en
       AND registro.tipo_consulta = 'cuadro';

    v_consulta_huella := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
            p_consulta
        )
    ), 'hex');
    v_filtros_huella := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.canon_familia_cuadro_rrhh_v1(
            p_consulta
        )
    ), 'hex');
    IF p_estado.es_continuacion IS DISTINCT FROM
           (p_consulta.cursor <> '')
       OR (
           p_estado.es_continuacion
           AND p_estado.token_presentado_huella_sha256
               IS DISTINCT FROM pg_catalog.encode(
                   pg_catalog.sha256(pg_catalog.convert_to(
                       p_consulta.cursor, 'UTF8'
                   )), 'hex'
               )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'efectos de cursor RRHH rechazados';
    END IF;
    v_decision_huella := pg_catalog.encode(
        pg_catalog.sha256(p_decision_canonica), 'hex'
    );
    IF v_alcance.tipo_consulta <> 'cuadro'
       OR v_alcance.organizacion_ref IS DISTINCT FROM
          p_alcance.organizacion_ref
       OR v_alcance.clase_ambito IS DISTINCT FROM p_alcance.clase_ambito
       OR v_alcance.ambito_ref IS DISTINCT FROM p_alcance.ambito_ref
       OR v_registro.decision_ref IS DISTINCT FROM p_consumo.decision_ref
       OR v_registro.prueba_tipo_consulta IS DISTINCT FROM 'cuadro'
       OR v_registro.prueba_generada_en IS DISTINCT FROM
          p_cierre.generada_en
       OR v_registro.consulta_huella_sha256 IS DISTINCT FROM
          v_consulta_huella
       OR v_registro.prueba_total IS DISTINCT FROM p_cierre.total
       OR v_registro.prueba_total IS DISTINCT FROM
          pg_catalog.cardinality(v_registro.prueba_resumenes)
       OR v_registro.prueba_hay_mas IS DISTINCT FROM p_salida.hay_mas
       OR v_registro.prueba_cursor_huella_sha256 IS DISTINCT FROM
          p_cierre.cursor_huella_sha256
       OR p_cierre.cursor_huella_sha256 IS DISTINCT FROM
          (CASE WHEN p_salida.hay_mas
                THEN p_salida.token_nuevo_huella_sha256 ELSE '' END)
       OR v_registro.decision_huella_sha256 IS DISTINCT FROM
          v_decision_huella
       OR v_registro.consumo_vec_huella_sha256 IS DISTINCT FROM
          p_consumo.consumo_huella_sha256
       OR p_cierre.consumo_vec_huella_sha256 IS DISTINCT FROM
          p_consumo.consumo_huella_sha256 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'efectos de cursor RRHH rechazados';
    END IF;

    IF p_estado.es_continuacion THEN
        SELECT familia.organizacion_ref, familia.clase_ambito,
               familia.ambito_ref, familia.actor_ref, familia.perfil_ref,
               familia.perfil_version, familia.sesion_ref,
               familia.sesion_huella_sha256, familia.dominio_filtros,
               familia.filtros_huella_sha256, familia.limite,
               familia.corte_global,
               familia.creada_en, familia.valida_hasta,
               cursor.token_huella_sha256, cursor.pagina,
               cursor.emitida_en, cursor.acceso_emision_ref,
               cursor.ultimo_actualizado_en,
               cursor.ultimo_expediente_ref, causal.revision
          INTO STRICT v_estado_durable
          FROM vec_contratacion_temporal
               .familia_cursor_cuadro_rrhh familia
          JOIN vec_contratacion_temporal.cursor_cuadro_rrhh cursor
            USING (familia_ref)
          JOIN vec_contratacion_temporal
               .control_causal_familia_cursor_rrhh causal
            USING (familia_ref)
         WHERE familia.familia_ref = p_estado.familia_ref
           AND cursor.token_huella_sha256 =
               p_estado.token_presentado_huella_sha256
         FOR UPDATE OF causal
         FOR SHARE OF familia, cursor;
        IF v_estado_durable.revision <> 0
           OR EXISTS (
               SELECT 1
                 FROM vec_contratacion_temporal
                      .revocacion_familia_cursor_rrhh revocacion
                WHERE revocacion.familia_ref = p_estado.familia_ref
           )
           OR EXISTS (
               SELECT 1
                 FROM vec_contratacion_temporal
                      .consumo_cursor_cuadro_rrhh consumo
                WHERE consumo.token_huella_sha256 =
                      p_estado.token_presentado_huella_sha256
           )
           OR v_estado_durable.organizacion_ref IS DISTINCT FROM
              v_alcance.organizacion_ref
           OR v_estado_durable.clase_ambito IS DISTINCT FROM
              v_alcance.clase_ambito
           OR v_estado_durable.ambito_ref IS DISTINCT FROM
              v_alcance.ambito_ref
           OR v_estado_durable.actor_ref IS DISTINCT FROM v_alcance.actor_ref
           OR v_estado_durable.perfil_ref IS DISTINCT FROM
              v_alcance.perfil_ref
           OR v_estado_durable.perfil_version IS DISTINCT FROM
              v_alcance.perfil_version
           OR v_estado_durable.sesion_ref IS DISTINCT FROM
              v_alcance.sesion_ref
           OR v_estado_durable.sesion_huella_sha256 IS DISTINCT FROM
              v_alcance.sesion_huella_sha256
           OR v_estado_durable.dominio_filtros IS DISTINCT FROM
              'vec.contratacion_temporal.filtros_rrhh.cuadro.v1'
           OR v_estado_durable.filtros_huella_sha256 IS DISTINCT FROM
              v_filtros_huella
           OR v_estado_durable.limite IS DISTINCT FROM p_consulta.limite
           OR v_estado_durable.corte_global IS DISTINCT FROM
              p_estado.corte_global
           OR v_estado_durable.pagina IS DISTINCT FROM
              p_estado.pagina_presentada
           OR v_estado_durable.emitida_en IS DISTINCT FROM
              p_estado.cursor_emitida_en
           OR v_estado_durable.acceso_emision_ref IS DISTINCT FROM
              p_estado.acceso_emision_ref
           OR v_estado_durable.creada_en IS DISTINCT FROM
              p_estado.familia_creada_en
           OR v_estado_durable.valida_hasta IS DISTINCT FROM
              p_estado.familia_valida_hasta
           OR v_estado_durable.ultimo_actualizado_en IS DISTINCT FROM
              p_estado.ultimo_actualizado_en
           OR v_estado_durable.ultimo_expediente_ref IS DISTINCT FROM
              p_estado.ultimo_expediente_ref THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'efectos de cursor RRHH rechazados';
        END IF;
    ELSE
        SELECT ultimo_corte AS corte_global
          INTO STRICT v_estado_durable
          FROM vec_contratacion_temporal.control_publicacion_rrhh
         WHERE control;
        IF p_estado.familia_ref IS NOT NULL
           OR p_estado.corte_global IS DISTINCT FROM
              v_estado_durable.corte_global
           OR p_estado.pagina_presentada IS DISTINCT FROM 0
           OR p_estado.token_presentado_huella_sha256 IS NOT NULL
           OR p_estado.acceso_emision_ref IS NOT NULL
           OR p_estado.cursor_emitida_en IS NOT NULL
           OR p_estado.familia_creada_en IS NOT NULL
           OR p_estado.familia_valida_hasta IS NOT NULL
           OR p_estado.ultimo_actualizado_en IS NOT NULL
           OR p_estado.ultimo_expediente_ref IS NOT NULL THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'efectos de cursor RRHH rechazados';
        END IF;
    END IF;

    IF p_salida.hay_mas
       AND (
           v_registro.prueba_total = 0
           OR (v_registro.prueba_resumenes[
                   v_registro.prueba_total
               ]).actualizado_en IS DISTINCT FROM
              p_salida.ultimo_actualizado_en
           OR (v_registro.prueba_resumenes[
                   v_registro.prueba_total
               ]).expediente_ref IS DISTINCT FROM
              p_salida.ultimo_expediente_ref
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'efectos de cursor RRHH rechazados';
    END IF;

    IF NOT p_estado.es_continuacion AND NOT p_salida.hay_mas THEN
        IF v_alcance.familia_ref IS NOT NULL
           THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'efectos de cursor RRHH rechazados';
        END IF;
        RETURN;
    END IF;

    IF p_salida.hay_mas
       AND (
           p_salida.cursor_siguiente !~ '^[A-Za-z0-9_-]{43}$'
           OR p_salida.token_nuevo_huella_sha256 !~ '^[0-9a-f]{64}$'
           OR p_salida.cursor_huella IS DISTINCT FROM pg_catalog.decode(
               p_salida.token_nuevo_huella_sha256, 'hex'
           )
           OR p_salida.token_nuevo_huella_sha256 IS DISTINCT FROM
              pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                  p_salida.cursor_siguiente, 'UTF8'
              )), 'hex')
           OR p_salida.ultimo_actualizado_en IS NULL
           OR p_salida.ultimo_expediente_ref IS NULL
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'efectos de cursor RRHH rechazados';
    END IF;

    IF NOT p_estado.es_continuacion THEN
        IF NOT p_salida.hay_mas
           OR v_alcance.familia_ref IS DISTINCT FROM p_salida.familia_ref
           OR p_salida.pagina_nueva IS DISTINCT FROM 2
           OR p_salida.padre_token_huella_sha256 IS NOT NULL
           OR p_estado.corte_global NOT BETWEEN
              1 AND 9007199254740991::numeric THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'efectos de cursor RRHH rechazados';
        END IF;
        v_valida_hasta := p_cierre.registrada_en + interval '5 minutes';
        v_familia_prueba := pg_catalog.convert_to(
            'VEC-CT-FAMILIA-CURSOR-CUADRO-RRHH-V1'
                || pg_catalog.chr(10), 'UTF8'
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_salida.familia_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            v_alcance.organizacion_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            v_alcance.clase_ambito
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            v_alcance.ambito_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(v_alcance.actor_ref)
        || vec_contratacion_temporal.encuadrar_texto_v1(
            v_alcance.perfil_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            v_alcance.perfil_version::text
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            v_alcance.sesion_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            v_alcance.sesion_huella_sha256
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            'vec.contratacion_temporal.filtros_rrhh.cuadro.v1'
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(v_filtros_huella)
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_consulta.limite::text
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_estado.corte_global::text
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(
                p_cierre.registrada_en
            )
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(v_valida_hasta)
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_cierre.acceso_ref
        );
        INSERT INTO
        vec_contratacion_temporal.familia_cursor_cuadro_rrhh (
            familia_ref, organizacion_ref, clase_ambito, ambito_ref,
            actor_ref, perfil_ref, perfil_version, sesion_ref,
            sesion_huella_sha256, dominio_filtros,
            filtros_huella_sha256, limite, corte_global, creada_en,
            valida_hasta, acceso_origen_ref, prueba_canonica,
            prueba_huella_sha256
        ) VALUES (
            p_salida.familia_ref, v_alcance.organizacion_ref,
            v_alcance.clase_ambito, v_alcance.ambito_ref,
            v_alcance.actor_ref, v_alcance.perfil_ref,
            v_alcance.perfil_version, v_alcance.sesion_ref,
            v_alcance.sesion_huella_sha256,
            'vec.contratacion_temporal.filtros_rrhh.cuadro.v1',
            v_filtros_huella, p_consulta.limite,
            p_estado.corte_global, p_cierre.registrada_en,
            v_valida_hasta, p_cierre.acceso_ref, v_familia_prueba,
            pg_catalog.encode(
                pg_catalog.sha256(v_familia_prueba), 'hex'
            )
        );
        INSERT INTO
        vec_contratacion_temporal.control_causal_familia_cursor_rrhh (
            familia_ref, familia_creada_en, revision, actualizada_en
        ) VALUES (
            p_salida.familia_ref, p_cierre.registrada_en, 0,
            p_cierre.registrada_en
        );
        v_padre_emitida_en := NULL;
    ELSE
        IF v_alcance.familia_ref IS DISTINCT FROM p_estado.familia_ref
           OR p_salida.familia_ref IS DISTINCT FROM
              (CASE WHEN p_salida.hay_mas
                    THEN p_estado.familia_ref ELSE NULL END)
           OR p_salida.padre_token_huella_sha256 IS DISTINCT FROM
              (CASE WHEN p_salida.hay_mas
                    THEN p_estado.token_presentado_huella_sha256
                    ELSE NULL END)
           OR p_salida.pagina_nueva IS DISTINCT FROM
              (CASE WHEN p_salida.hay_mas
                    THEN p_estado.pagina_presentada + 1 ELSE 0 END)
           OR p_cierre.registrada_en >= p_estado.familia_valida_hasta THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'efectos de cursor RRHH rechazados';
        END IF;
        v_consumo_prueba := pg_catalog.convert_to(
            'VEC-CT-CONSUMO-CURSOR-CUADRO-RRHH-V1'
                || pg_catalog.chr(10), 'UTF8'
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_estado.token_presentado_huella_sha256
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_estado.familia_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_consumo.decision_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(v_decision_huella)
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_consumo.consumo_huella_sha256
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_estado.acceso_emision_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_cierre.acceso_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(
                p_estado.cursor_emitida_en
            )
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(
                p_estado.familia_valida_hasta
            )
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(
                p_cierre.registrada_en
            )
        );
        INSERT INTO
        vec_contratacion_temporal.consumo_cursor_cuadro_rrhh (
            token_huella_sha256, familia_ref, decision_ref,
            decision_huella_sha256, consumo_vec_huella_sha256,
            acceso_emision_ref, acceso_consumo_ref, cursor_emitida_en,
            familia_valida_hasta, consumido_en, prueba_canonica,
            prueba_huella_sha256
        ) VALUES (
            p_estado.token_presentado_huella_sha256,
            p_estado.familia_ref, p_consumo.decision_ref,
            v_decision_huella, p_consumo.consumo_huella_sha256,
            p_estado.acceso_emision_ref, p_cierre.acceso_ref,
            p_estado.cursor_emitida_en, p_estado.familia_valida_hasta,
            p_cierre.registrada_en, v_consumo_prueba,
            pg_catalog.encode(
                pg_catalog.sha256(v_consumo_prueba), 'hex'
            )
        );
        v_valida_hasta := p_estado.familia_valida_hasta;
        v_padre_emitida_en := p_estado.cursor_emitida_en;
    END IF;

    IF p_salida.hay_mas THEN
        v_cursor_prueba := pg_catalog.convert_to(
            'VEC-CT-CURSOR-CUADRO-RRHH-V1'
                || pg_catalog.chr(10), 'UTF8'
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_salida.token_nuevo_huella_sha256
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_salida.familia_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            COALESCE(p_salida.padre_token_huella_sha256, '')
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_salida.pagina_nueva::text
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            CASE WHEN v_padre_emitida_en IS NULL THEN ''
                 ELSE vec_contratacion_temporal.instante_utc_v1(
                     v_padre_emitida_en
                 ) END
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(
                p_salida.ultimo_actualizado_en
            )
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_salida.ultimo_expediente_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(
                CASE WHEN p_estado.es_continuacion
                     THEN p_estado.familia_creada_en
                     ELSE p_cierre.registrada_en END
            )
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(v_valida_hasta)
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(
                p_cierre.registrada_en
            )
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_cierre.acceso_ref
        );
        INSERT INTO vec_contratacion_temporal.cursor_cuadro_rrhh (
            token_huella_sha256, familia_ref,
            padre_token_huella_sha256, pagina, padre_emitida_en,
            ultimo_actualizado_en, ultimo_expediente_ref,
            familia_creada_en, familia_valida_hasta, emitida_en,
            acceso_emision_ref, prueba_canonica, prueba_huella_sha256
        ) VALUES (
            p_salida.token_nuevo_huella_sha256, p_salida.familia_ref,
            p_salida.padre_token_huella_sha256,
            p_salida.pagina_nueva, v_padre_emitida_en,
            p_salida.ultimo_actualizado_en,
            p_salida.ultimo_expediente_ref,
            CASE WHEN p_estado.es_continuacion
                 THEN p_estado.familia_creada_en
                 ELSE p_cierre.registrada_en END,
            v_valida_hasta, p_cierre.registrada_en,
            p_cierre.acceso_ref, v_cursor_prueba,
            pg_catalog.encode(pg_catalog.sha256(v_cursor_prueba), 'hex')
        );
    END IF;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'efectos de cursor RRHH rechazados';
END;
$funcion$;

ALTER FUNCTION
vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
    vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1,
    vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,
    bytea,
    vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
) OWNER TO vec_contratacion_temporal_propietario;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
    vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1,
    vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,
    bytea,
    vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
) FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_consultor_rrhh,
    vec_contratacion_temporal_lector_resultado_cobertura;

COMMENT ON FUNCTION
vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
    vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1,
    vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,
    bytea,
    vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
) IS
'Persiste familia, consumo y cursor desde el cierre durable, sin identidad libre.';
