-- CT-000043: cierre interno. Solo el futuro motor propietario podrá llamarlo.
CREATE FUNCTION
vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v2(
    p_contexto
        vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2,
    p_contenido
        vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2,
    p_consumo
        vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,
    p_capacidad_canonica bytea,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_contexto_actor_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_payload_vec_ad_3 bytea,
    p_sobre_cose_sign_1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea
)
RETURNS
    vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '12s'
AS $funcion$
DECLARE
    v_capacidad jsonb;
    v_decision jsonb;
    v_contexto_actor jsonb;
    v_tipo text;
    v_accion text;
    v_finalidad text;
    v_audiencia text;
    v_dominio_consulta text;
    v_tipo_recurso text;
    v_expediente_ref text;
    v_version_expediente numeric(20, 0);
    v_total smallint;
    v_consulta_canonica bytea;
    v_consulta_huella text;
    v_contexto_recurso_canonico bytea;
    v_contexto_recurso_huella text;
    v_contenido_canonico bytea;
    v_contenido_huella text;
    v_cursor_huella text;
    v_resultado_canonico bytea;
    v_resultado_huella text;
    v_material_huella text;
    v_capacidad_huella text;
    v_decision_huella text;
    v_revalidacion record;
    v_registro_peticion jsonb;
    v_registro_salida jsonb;
    v_registro record;
    v_identidad record;
    v_alcance vec_contratacion_temporal.alcance_acceso_rrhh%ROWTYPE;
    v_recibo_evidencia
        vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2;
    v_recibo_canonico bytea;
    v_recibo_sello text;
    v_esquema_constante text :=
        'vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v2';
BEGIN
    IF CURRENT_USER <>
           'vec_contratacion_temporal_propietario'
       OR SESSION_USER =
          'vec_contratacion_temporal_propietario'
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_isolation')
          <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR p_contexto IS NULL
       OR p_contenido IS NULL
       OR p_consumo IS NULL
       OR p_consumo.consumo_nuevo IS DISTINCT FROM true
       OR p_consumo.consumida_en IS NULL
       OR p_consumo.consumida_en <>
          pg_catalog.date_trunc('microseconds', p_consumo.consumida_en)
       OR p_contenido.generada_en IS NULL
       OR p_contenido.generada_en <
          p_consumo.consumida_en THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'cierre de prueba RRHH rechazado';
    END IF;

    BEGIN
        v_capacidad := pg_catalog.convert_from(
            p_capacidad_canonica, 'UTF8'
        )::jsonb;
        v_decision := pg_catalog.convert_from(
            p_decision_canonica, 'UTF8'
        )::jsonb;
        v_contexto_actor := pg_catalog.convert_from(
            p_contexto_actor_canonico, 'UTF8'
        )::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire
          OR untranslatable_character THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'cierre de prueba RRHH inválido';
    END;

    v_tipo := p_contenido.tipo_consulta;
    IF v_tipo = 'cuadro' THEN
        v_accion := 'contratacion_temporal.cuadro.consultar';
        v_finalidad := 'gestion_operativa_contratacion_temporal';
        v_audiencia :=
            'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1';
        v_dominio_consulta :=
            'vec.contratacion_temporal.consulta_rrhh.cuadro.v1';
        v_tipo_recurso := 'cuadro_rrhh_contratacion_temporal';
        IF p_contexto.consulta_cuadro IS NOT DISTINCT FROM
               NULL::vec_contratacion_temporal.consulta_cuadro_rrhh_v1
           OR p_contexto.consulta_detalle IS DISTINCT FROM
               NULL::vec_contratacion_temporal.consulta_detalle_rrhh_v1
           OR p_contenido.resumenes IS NULL
           OR p_contenido.hay_mas IS NULL
           OR p_contenido.cursor_huella IS NULL
           OR p_contenido.detalle IS DISTINCT FROM
               NULL::vec_contratacion_temporal
                   .entrada_detalle_expediente_rrhh_v1 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'cierre de prueba RRHH inválido';
        END IF;
        v_consulta_canonica :=
            vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
                p_contexto.consulta_cuadro
            );
        v_contenido_canonico :=
            vec_contratacion_temporal.canon_contenido_cuadro_rrhh_v1(
                p_contenido.generada_en, p_contenido.resumenes,
                p_contenido.hay_mas, p_contenido.cursor_huella
            );
        v_total := pg_catalog.cardinality(
            p_contenido.resumenes
        )::smallint;
        v_cursor_huella := CASE
            WHEN p_contenido.hay_mas THEN
                pg_catalog.encode(p_contenido.cursor_huella, 'hex')
            ELSE ''
        END;
    ELSIF v_tipo = 'detalle' THEN
        v_accion := 'contratacion_temporal.expediente.consultar';
        v_finalidad :=
            'tramitacion_expediente_contratacion_temporal';
        v_audiencia :=
            'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1';
        v_dominio_consulta :=
            'vec.contratacion_temporal.consulta_rrhh.detalle.v1';
        v_tipo_recurso := 'expediente_contratacion_temporal';
        IF p_contexto.consulta_cuadro IS DISTINCT FROM
               NULL::vec_contratacion_temporal.consulta_cuadro_rrhh_v1
           OR p_contexto.consulta_detalle IS NOT DISTINCT FROM
               NULL::vec_contratacion_temporal.consulta_detalle_rrhh_v1
           OR p_contenido.detalle IS NOT DISTINCT FROM
               NULL::vec_contratacion_temporal
                   .entrada_detalle_expediente_rrhh_v1
           OR p_contenido.resumenes IS NULL
           OR p_contenido.hay_mas IS DISTINCT FROM false
           OR pg_catalog.cardinality(p_contenido.resumenes) <> 0
           OR pg_catalog.array_ndims(p_contenido.resumenes) IS NOT NULL
           OR p_contenido.cursor_huella IS NULL
           OR pg_catalog.octet_length(p_contenido.cursor_huella) <> 0
           OR p_contexto.familia_ref IS NOT NULL THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'cierre de prueba RRHH inválido';
        END IF;
        v_consulta_canonica :=
            vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
                p_contexto.consulta_detalle
            );
        v_contenido_canonico :=
            vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
                p_contenido.generada_en, p_contenido.detalle
            );
        v_expediente_ref :=
            (p_contenido.detalle).resumen.expediente_ref;
        v_version_expediente :=
            (p_contenido.detalle).resumen.version;
        IF (p_contexto.consulta_detalle).expediente_ref
               <> v_expediente_ref
           OR (p_contexto.consulta_detalle).version_observada
              <> v_version_expediente THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'cierre de prueba RRHH inválido';
        END IF;
        v_total := 1;
        v_cursor_huella := '';
    ELSE
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'cierre de prueba RRHH inválido';
    END IF;

    v_consulta_huella := pg_catalog.encode(
        pg_catalog.sha256(v_consulta_canonica), 'hex'
    );
    v_contexto_recurso_canonico := pg_catalog.convert_to(
        '{"ambitos":{"ambito_ref":"'
        || p_contexto.ambito_ref
        || '","clase_ambito":"' || p_contexto.clase_ambito
        || '","organizacion_ref":"' || p_contexto.organizacion_ref
        || '"},"atributos":{"consulta_dominio":"'
        || v_dominio_consulta
        || '","consulta_huella_sha256":"' || v_consulta_huella
        || '"}}', 'UTF8'
    );
    v_contexto_recurso_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso_canonico), 'hex'
    );

    v_contenido_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contenido_canonico), 'hex'
    );
    v_resultado_canonico :=
        vec_contratacion_temporal
        .canon_resultado_consulta_rrhh_puro_v1(ROW(
            v_tipo, p_contenido.generada_en, v_total,
            v_contenido_huella, v_cursor_huella
        )::vec_contratacion_temporal.evidencia_resultado_rrhh_v1);
    v_resultado_huella := pg_catalog.encode(
        pg_catalog.sha256(v_resultado_canonico), 'hex'
    );
    v_material_huella :=
        vec_contratacion_temporal.huella_material_consumo_rrhh_v3(
            p_capacidad_canonica, p_decision_canonica,
            p_motivo_canonico, p_contexto_actor_canonico,
            p_persona_version, p_perfil_version,
            p_payload_vec_ad_3, p_sobre_cose_sign_1,
            p_evidencia_verificacion, p_raiz_publica_spki
        );
    v_capacidad_huella := pg_catalog.encode(
        pg_catalog.sha256(p_capacidad_canonica), 'hex'
    );
    v_decision_huella := pg_catalog.encode(
        pg_catalog.sha256(p_decision_canonica), 'hex'
    );

    IF p_contexto.organizacion_ref IS NULL
       OR p_contexto.organizacion_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_contexto.clase_ambito NOT IN (
           'organizacion', 'centro', 'unidad_gestion'
       )
       OR p_contexto.ambito_ref IS NULL
       OR p_contexto.ambito_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (
           p_contexto.clase_ambito = 'organizacion'
           AND p_contexto.ambito_ref <> p_contexto.organizacion_ref
       )
       OR (
           p_contexto.familia_ref IS NOT NULL
           AND p_contexto.familia_ref !~
               '^familia:cursor:rrhh:[0-9a-f]{32}$'
       )
       OR v_capacidad ->> 'decision_ref' IS DISTINCT FROM
          p_consumo.decision_ref
       OR v_capacidad ->> 'efecto_ref' IS DISTINCT FROM
          p_consumo.efecto_ref
       OR v_capacidad ->> 'huella_efecto_sha256' IS DISTINCT FROM
          p_consumo.huella_efecto_sha256
       OR v_capacidad ->> 'huella_efecto_sha256' IS DISTINCT FROM
          v_contexto_recurso_huella
       OR v_capacidad ->> 'operacion' IS DISTINCT FROM v_accion
       OR v_capacidad ->> 'audiencia_consumo' IS DISTINCT FROM
          v_audiencia
       OR v_capacidad ->> 'huella_decision_sha256' IS DISTINCT FROM
          v_decision_huella
       OR v_decision ->> 'decision_ref' IS DISTINCT FROM
          p_consumo.decision_ref
       OR v_decision ->> 'recurso_ref' IS DISTINCT FROM
          (CASE WHEN v_tipo = 'cuadro'
                THEN p_contexto.ambito_ref ELSE v_expediente_ref END)
       OR v_decision ->> 'modulo_id' IS DISTINCT FROM
          'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' IS DISTINCT FROM
          v_tipo_recurso
       OR v_decision ->> 'contexto_recurso_huella_sha256'
          IS DISTINCT FROM v_contexto_recurso_huella
       OR v_decision ->> 'accion' IS DISTINCT FROM v_accion
       OR v_decision ->> 'finalidad' IS DISTINCT FROM v_finalidad
       OR v_decision ->> 'principal_id' IS DISTINCT FROM
          v_contexto_actor ->> 'principal_ref'
       OR v_decision ->> 'perfil_activo_ref' IS DISTINCT FROM
          v_contexto_actor ->> 'perfil_activo_ref'
       OR v_contexto_actor ->> 'perfil_version' IS DISTINCT FROM
          p_perfil_version::text
       OR v_contexto_actor ->> 'persona_version' IS DISTINCT FROM
          p_persona_version::text
       OR pg_catalog.encode(
          pg_catalog.sha256(p_contexto_actor_canonico), 'hex'
       ) IS DISTINCT FROM v_capacidad ->> 'huella_contexto_sha256'
       OR p_consumo.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_consumo.auditoria_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_consumo.auditoria_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.registro_acceso_rrhh registro
            WHERE registro.consumo_vec_huella_sha256 =
                  p_consumo.consumo_huella_sha256
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'cierre de prueba RRHH rechazado';
    END IF;

    IF v_tipo = 'cuadro' THEN
        SELECT *
          INTO STRICT v_revalidacion
          FROM vec_autorizacion_atestada_v3
               .revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada(
              p_capacidad_canonica, p_decision_canonica,
              p_motivo_canonico, p_contexto_actor_canonico,
              p_persona_version, p_perfil_version,
              p_payload_vec_ad_3, p_sobre_cose_sign_1,
              p_evidencia_verificacion, p_raiz_publica_spki
          );
    ELSE
        SELECT *
          INTO STRICT v_revalidacion
          FROM vec_autorizacion_atestada_v3
               .revalidar_evidencia_consumo_consulta_detalle_rrhh_v3_atestada(
              p_capacidad_canonica, p_decision_canonica,
              p_motivo_canonico, p_contexto_actor_canonico,
              p_persona_version, p_perfil_version,
              p_payload_vec_ad_3, p_sobre_cose_sign_1,
              p_evidencia_verificacion, p_raiz_publica_spki
          );
    END IF;
    IF v_revalidacion.decision_ref IS DISTINCT FROM
           p_consumo.decision_ref
       OR v_revalidacion.efecto_ref IS DISTINCT FROM
          p_consumo.efecto_ref
       OR v_revalidacion.huella_efecto_sha256 IS DISTINCT FROM
          p_consumo.huella_efecto_sha256
       OR v_revalidacion.consumo_huella_sha256 IS DISTINCT FROM
          p_consumo.consumo_huella_sha256
       OR v_revalidacion.auditoria_ref IS DISTINCT FROM
          p_consumo.auditoria_ref
       OR v_revalidacion.auditoria_huella_sha256 IS DISTINCT FROM
          p_consumo.auditoria_huella_sha256
       OR v_revalidacion.consumida_en IS DISTINCT FROM
          p_consumo.consumida_en
       OR v_revalidacion.revalidada_en IS NULL
       OR v_revalidacion.revalidada_en <
          p_consumo.consumida_en
       OR v_revalidacion.revalidada_en <
          p_contenido.generada_en THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'cierre de prueba RRHH rechazado';
    END IF;

    v_registro_peticion := pg_catalog.jsonb_build_object(
        'registro', pg_catalog.jsonb_build_object(
            'accion', v_accion,
            'actor_ref', v_decision ->> 'principal_id',
            'ambito_ref', p_contexto.ambito_ref,
            'audiencia', v_audiencia,
            'auditoria_vec_huella_sha256',
                p_consumo.auditoria_huella_sha256,
            'auditoria_vec_ref', p_consumo.auditoria_ref,
            'capacidad_huella_sha256', v_capacidad_huella,
            'consulta_huella_sha256', v_consulta_huella,
            'consumo_vec_huella_sha256',
                p_consumo.consumo_huella_sha256,
            'correlacion_ref', v_decision ->> 'correlacion_ref',
            'decision_huella_sha256', v_decision_huella,
            'decision_ref', p_consumo.decision_ref,
            'dominio_huella_consulta', v_dominio_consulta,
            'expediente_ref', v_expediente_ref,
            'finalidad', v_finalidad,
            'modulo_id', 'contratacion_temporal',
            'organizacion_ref', p_contexto.organizacion_ref,
            'perfil_id', v_decision ->> 'perfil_activo_ref',
            'perfil_version', p_perfil_version,
            'recurso_ref', p_consumo.efecto_ref,
            'recurso_tipo', v_tipo_recurso,
            'resultado_generico', 'entregado',
            'resultado_huella_sha256', v_resultado_huella,
            'sesion_huella_sha256',
                v_decision #>>
                '{vinculo_autenticacion_actor,control_sesion_huella_sha256}',
            'sesion_id',
                v_decision #>>
                '{vinculo_autenticacion_actor,sesion_ref}',
            'tipo_consulta', v_tipo,
            'total', v_total,
            'version_expediente', v_version_expediente
        ),
        'alcance', pg_catalog.jsonb_build_object(
            'clase_ambito', p_contexto.clase_ambito,
            'familia_ref', p_contexto.familia_ref
        ),
        'identidad', pg_catalog.jsonb_build_object(
            'actor_ref', v_decision ->> 'principal_id',
            'autenticacion_huella_sha256',
                v_decision #>>
                '{vinculo_autenticacion_actor,autenticacion_huella_sha256}',
            'autenticacion_ref',
                v_decision #>>
                '{vinculo_autenticacion_actor,autenticacion_ref}',
            'control_sesion_huella_sha256',
                v_decision #>>
                '{vinculo_autenticacion_actor,control_sesion_huella_sha256}',
            'control_sesion_ref',
                v_decision #>>
                '{vinculo_autenticacion_actor,control_sesion_ref}',
            'control_sesion_revision',
                v_decision #>>
                '{vinculo_autenticacion_actor,control_sesion_revision}',
            'organizacion_ref', p_contexto.organizacion_ref,
            'perfil_ref', v_decision ->> 'perfil_activo_ref',
            'perfil_version', p_perfil_version,
            'sesion_ref',
                v_decision #>>
                '{vinculo_autenticacion_actor,sesion_ref}'
        )
    );
    v_registro_salida :=
        vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(
            v_registro_peticion
        );

    SELECT *
      INTO STRICT v_registro
      FROM vec_contratacion_temporal.registro_acceso_rrhh
     WHERE acceso_ref = v_registro_salida ->> 'acceso_ref';
    SELECT *
      INTO STRICT v_identidad
      FROM vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
     WHERE acceso_ref = v_registro.acceso_ref;
    IF v_tipo = 'cuadro' THEN
        SELECT *
          INTO STRICT v_alcance
          FROM vec_contratacion_temporal.alcance_acceso_rrhh
         WHERE acceso_ref = v_registro.acceso_ref;
    END IF;

    v_recibo_evidencia := ROW(
        v_esquema_constante, v_registro.acceso_ref,
        v_registro.secuencia, v_registro.anterior_sha256,
        v_registro.huella_sha256,
        v_identidad.prueba_huella_sha256,
        CASE WHEN v_tipo = 'cuadro'
             THEN v_alcance.prueba_huella_sha256 ELSE '' END,
        v_registro.registrada_en, v_registro.auditoria_vec_ref,
        v_registro.auditoria_vec_huella_sha256,
        v_registro.consumo_vec_huella_sha256,
        v_registro.decision_ref, v_registro.decision_huella_sha256,
        v_registro.capacidad_huella_sha256, v_material_huella,
        v_registro.consulta_huella_sha256,
        v_registro.correlacion_ref,
        v_identidad.autenticacion_ref,
        v_identidad.autenticacion_huella_sha256,
        v_identidad.sesion_ref, v_identidad.control_sesion_ref,
        v_identidad.control_sesion_revision,
        v_identidad.control_sesion_huella_sha256,
        v_identidad.actor_ref, v_identidad.perfil_ref,
        v_identidad.perfil_version, v_identidad.organizacion_ref,
        v_identidad.clase_ambito, v_identidad.ambito_ref,
        v_accion, v_finalidad, COALESCE(v_expediente_ref, ''),
        COALESCE(v_version_expediente, 0), v_total,
        v_contenido_huella, v_resultado_huella,
        v_cursor_huella, p_contenido.generada_en
    )::vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2;
    v_recibo_canonico :=
        vec_contratacion_temporal.canon_recibo_lectura_rrhh_v2(
            v_recibo_evidencia
        );
    v_recibo_sello := pg_catalog.encode(
        pg_catalog.sha256(v_recibo_canonico), 'hex'
    );

    INSERT INTO
    vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 (
        acceso_ref, tipo_consulta, expediente_ref,
        version_expediente, total, generada_en,
        resumenes, hay_mas, cursor_material_huella_sha256,
        detalle,
        contenido_canonico, contenido_huella_sha256,
        cursor_huella_sha256, resultado_canonico,
        resultado_huella_sha256, material_huella_sha256,
        revalidada_en, recibo_canonico, recibo_sello_sha256,
        secuencia, anterior_sha256, huella_sha256,
        vinculo_identidad_huella_sha256, alcance_acceso_ref,
        alcance_huella_sha256, registrada_en,
        auditoria_vec_ref, auditoria_vec_huella_sha256,
        consumo_vec_huella_sha256, decision_ref,
        decision_huella_sha256, capacidad_huella_sha256,
        consulta_huella_sha256, correlacion_ref,
        autenticacion_ref, autenticacion_huella_sha256,
        sesion_ref, sesion_huella_sha256, control_sesion_ref,
        control_sesion_revision, control_sesion_huella_sha256,
        actor_ref, perfil_ref, perfil_version, organizacion_ref,
        clase_ambito, ambito_ref, accion, finalidad
    ) VALUES (
        v_registro.acceso_ref, v_tipo, v_expediente_ref,
        v_version_expediente, v_total, p_contenido.generada_en,
        p_contenido.resumenes, p_contenido.hay_mas,
        p_contenido.cursor_huella, p_contenido.detalle,
        v_contenido_canonico, v_contenido_huella,
        NULLIF(v_cursor_huella, ''), v_resultado_canonico,
        v_resultado_huella, v_material_huella,
        v_revalidacion.revalidada_en, v_recibo_canonico,
        v_recibo_sello, v_registro.secuencia,
        v_registro.anterior_sha256, v_registro.huella_sha256,
        v_identidad.prueba_huella_sha256,
        CASE WHEN v_tipo = 'cuadro'
             THEN v_alcance.acceso_ref ELSE NULL END,
        CASE WHEN v_tipo = 'cuadro'
             THEN v_alcance.prueba_huella_sha256 ELSE NULL END,
        v_registro.registrada_en, v_registro.auditoria_vec_ref,
        v_registro.auditoria_vec_huella_sha256,
        v_registro.consumo_vec_huella_sha256,
        v_registro.decision_ref, v_registro.decision_huella_sha256,
        v_registro.capacidad_huella_sha256,
        v_registro.consulta_huella_sha256,
        v_registro.correlacion_ref,
        v_identidad.autenticacion_ref,
        v_identidad.autenticacion_huella_sha256,
        v_identidad.sesion_ref, v_identidad.sesion_huella_sha256,
        v_identidad.control_sesion_ref,
        v_identidad.control_sesion_revision,
        v_identidad.control_sesion_huella_sha256,
        v_identidad.actor_ref, v_identidad.perfil_ref,
        v_identidad.perfil_version, v_identidad.organizacion_ref,
        v_identidad.clase_ambito, v_identidad.ambito_ref,
        v_accion, v_finalidad
    );

    RETURN ROW(
        v_esquema_constante, v_registro.acceso_ref,
        v_registro.secuencia, v_registro.anterior_sha256,
        v_registro.huella_sha256,
        v_identidad.prueba_huella_sha256,
        CASE WHEN v_tipo = 'cuadro'
             THEN v_alcance.prueba_huella_sha256 ELSE '' END,
        v_registro.registrada_en, v_registro.auditoria_vec_ref,
        v_registro.auditoria_vec_huella_sha256,
        v_registro.consumo_vec_huella_sha256,
        v_contenido_huella, v_resultado_huella,
        v_cursor_huella, p_contenido.generada_en,
        COALESCE(v_expediente_ref, ''),
        COALESCE(v_version_expediente, 0), v_total,
        v_recibo_sello
    )::vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'cierre de prueba RRHH rechazado';
END
$funcion$;
