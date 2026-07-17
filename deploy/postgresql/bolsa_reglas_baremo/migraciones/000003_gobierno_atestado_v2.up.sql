-- Gobierno V2 compuesto con VEC-AD-2. Las funciones permanecen sin GRANT
-- runtime: faltan el adaptador Go, el broker real y el verificador confiable
-- de evidencia de transicion. V1 tampoco se abre.
BEGIN;
SET LOCAL ROLE vec_bolsa_reglas_baremo_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_reglas_baremo:migracion:000003', 0
    )
);

DO $prevalidacion$
BEGIN
    IF to_regprocedure(
           'vec_bolsa_reglas_baremo.confirmar_cambio_atestado_v2(bytea,bytea,bytea,bytea,bytea,bytea,jsonb)'
       ) IS NOT NULL
       OR to_regprocedure(
           'vec_bolsa_reglas_baremo.reconciliar_cambio_atestado_v2(text,text,text,text,text,text)'
       ) IS NOT NULL
       OR to_regclass(
           'vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2'
       ) IS NOT NULL
       OR NOT pg_catalog.has_function_privilege(
           'vec_bolsa_reglas_baremo_propietario',
           'vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada(bytea,bytea,bytea,bytea,bytea,bytea,jsonb)',
           'EXECUTE'
       )
       OR NOT pg_catalog.has_function_privilege(
           'vec_bolsa_reglas_baremo_propietario',
           'vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2(text,text,text,text,text)',
           'EXECUTE'
       )
       OR to_regprocedure(
           'vec_autorizacion_atestada_v2.obtener_vinculo_consumo_modular_v2(text,text,text,text,text,text,text)'
       ) IS NULL
       OR NOT pg_catalog.has_function_privilege(
           'vec_bolsa_reglas_baremo_propietario',
           'vec_autorizacion_atestada_v2.obtener_vinculo_consumo_modular_v2(text,text,text,text,text,text,text)',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar gobierno atestado V2';
    END IF;
END
$prevalidacion$;

-- V2 publica su propio evento. V1 conserva sin cambios su pareja de ruta y
-- esquema; no se aceptan combinaciones cruzadas.
ALTER TABLE vec_bolsa_reglas_baremo.outbox
    DROP CONSTRAINT outbox_valida;
ALTER TABLE vec_bolsa_reglas_baremo.outbox
    ADD CONSTRAINT outbox_valida CHECK (
        vec_bolsa_reglas_baremo.referencia_valida(outbox_ref)
        AND outbox_version = 1
        AND (
            (ruta = 'bolsa.reglas_baremo.estado_confirmado.v1'
             AND esquema_evento =
                 'vec.bolsa.reglas-baremo.estado-confirmado.v1')
            OR
            (ruta = 'bolsa.reglas_baremo.estado_confirmado.v2'
             AND esquema_evento =
                 'vec.bolsa.reglas-baremo.estado-confirmado.v2')
        )
        AND octet_length(evento_canonico) BETWEEN 2 AND 1048576
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_evento_sha256
        )
        AND encode(sha256(evento_canonico), 'hex') =
            huella_evento_sha256
        AND isfinite(creada_en)
    );

CREATE TABLE vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2 (
    tenant_id text NOT NULL,
    recibo_ref text NOT NULL,
    recibo_version numeric(20, 0) NOT NULL DEFAULT 2,
    recibo_canonico bytea NOT NULL,
    huella_recibo_sha256 text NOT NULL,
    plan_canonico bytea NOT NULL,
    huella_plan_sha256 text NOT NULL,
    intencion_ref text NOT NULL,
    intencion_version numeric(20, 0) NOT NULL,
    intencion_huella_sha256 text NOT NULL,
    decision_ref text NOT NULL,
    huella_decision_sha256 text NOT NULL,
    registro_vec_ref text NOT NULL,
    consumo_vec_ref text NOT NULL,
    auditoria_vec_ref text NOT NULL,
    huella_auditoria_vec_sha256 text NOT NULL,
    efecto_ref text NOT NULL,
    huella_efecto_sha256 text NOT NULL,
    huella_nonce_sha256 text NOT NULL,
    contenido_ref text NOT NULL,
    contenido_version numeric(20, 0) NOT NULL,
    huella_contenido_sha256 text NOT NULL,
    resultado_revision numeric(20, 0) NOT NULL,
    resultado_estado text NOT NULL,
    resultado_huella_estado_sha256 text NOT NULL,
    transaccion_ref text NOT NULL,
    transaccion_version numeric(20, 0) NOT NULL,
    huella_transaccion_sha256 text NOT NULL,
    auditoria_local_ref text NOT NULL,
    auditoria_local_version numeric(20, 0) NOT NULL,
    huella_auditoria_local_sha256 text NOT NULL,
    outbox_ref text NOT NULL,
    outbox_version numeric(20, 0) NOT NULL,
    huella_evento_sha256 text NOT NULL,
    confirmada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (tenant_id, recibo_ref),
    UNIQUE (tenant_id, recibo_ref, recibo_version, huella_recibo_sha256),
    UNIQUE (tenant_id, huella_plan_sha256),
    UNIQUE (tenant_id, intencion_ref, intencion_version),
    UNIQUE (tenant_id, decision_ref),
    UNIQUE (tenant_id, consumo_vec_ref),
    FOREIGN KEY (
        registro_vec_ref, decision_ref, huella_decision_sha256
    ) REFERENCES vec_autorizacion_atestada_v2.atestacion_decision_v2(
        registro_ref, decision_ref, huella_decision_sha256
    ),
    FOREIGN KEY (
        consumo_vec_ref, registro_vec_ref, decision_ref,
        huella_decision_sha256, efecto_ref, huella_efecto_sha256
    ) REFERENCES vec_autorizacion_atestada_v2.consumo_decision_v2(
        consumo_ref, registro_ref, decision_ref,
        huella_decision_sha256, efecto_ref, huella_efecto_sha256
    ),
    FOREIGN KEY (
        auditoria_vec_ref, consumo_vec_ref, registro_vec_ref,
        decision_ref, efecto_ref, huella_auditoria_vec_sha256
    ) REFERENCES vec_autorizacion_atestada_v2.auditoria_consumo_v2(
        auditoria_ref, consumo_ref, registro_ref, decision_ref,
        efecto_ref, huella_registro_sha256
    ),
    FOREIGN KEY (
        tenant_id, contenido_ref, contenido_version, resultado_revision,
        resultado_huella_estado_sha256
    ) REFERENCES vec_bolsa_reglas_baremo.version_reglas_baremo(
        tenant_id, contenido_ref, contenido_version, revision,
        huella_estado_sha256
    ),
    FOREIGN KEY (
        tenant_id, decision_ref, consumo_vec_ref
    ) REFERENCES vec_bolsa_reglas_baremo.uso_decision(
        tenant_id, decision_ref, consumo_autorizacion_ref
    ),
    FOREIGN KEY (
        tenant_id, intencion_ref, intencion_version
    ) REFERENCES vec_bolsa_reglas_baremo.intencion_confirmada(
        tenant_id, intencion_ref, intencion_version
    ),
    FOREIGN KEY (
        tenant_id, auditoria_local_ref, auditoria_local_version,
        huella_auditoria_local_sha256
    ) REFERENCES vec_bolsa_reglas_baremo.auditoria(
        tenant_id, auditoria_ref, auditoria_version,
        huella_auditoria_sha256
    ),
    FOREIGN KEY (
        tenant_id, outbox_ref, outbox_version, huella_evento_sha256
    ) REFERENCES vec_bolsa_reglas_baremo.outbox(
        tenant_id, outbox_ref, outbox_version, huella_evento_sha256
    ),
    CONSTRAINT recibo_atestado_v2_identidad CHECK (
        tenant_id = 'diputacion_granada'
        AND recibo_version = 2
        AND recibo_ref =
            'recibo:reglas-baremo:v2:' || huella_recibo_sha256
        AND intencion_version = 2
        AND intencion_ref =
            'intencion:reglas-baremo:v2:' || intencion_huella_sha256
        AND efecto_ref =
            'efecto:reglas-baremo:v2:' || huella_plan_sha256
        AND huella_efecto_sha256 = huella_plan_sha256
        AND contenido_ref ~ '^rgl_[0-9a-f]{32}$'
        AND resultado_estado IN (
            'borrador', 'publicada', 'activa', 'sustituida',
            'retirada', 'descartada'
        )
        AND isfinite(confirmada_en)
    ),
    CONSTRAINT recibo_atestado_v2_huellas CHECK (
        vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_recibo_sha256
        )
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_plan_sha256
        )
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            intencion_huella_sha256
        )
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_decision_sha256
        )
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_auditoria_vec_sha256
        )
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_nonce_sha256
        )
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_contenido_sha256
        )
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            resultado_huella_estado_sha256
        )
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_transaccion_sha256
        )
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_auditoria_local_sha256
        )
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_evento_sha256
        )
    ),
    CONSTRAINT recibo_atestado_v2_bytes CHECK (
        octet_length(plan_canonico) BETWEEN 2 AND 8388608
        AND encode(sha256(plan_canonico), 'hex') = huella_plan_sha256
        AND octet_length(recibo_canonico) BETWEEN 2 AND 1048576
        AND encode(sha256(recibo_canonico), 'hex') = huella_recibo_sha256
    )
);

CREATE TRIGGER impedir_mutacion
    BEFORE UPDATE OR DELETE
    ON vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_reglas_baremo.rechazar_mutacion_inmutable();
CREATE TRIGGER impedir_truncado
    BEFORE TRUNCATE
    ON vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2
    FOR EACH STATEMENT EXECUTE FUNCTION
        vec_bolsa_reglas_baremo.rechazar_mutacion_inmutable();
ALTER TABLE vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2
    FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_propietario_exacto
    ON vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2
    FOR ALL TO vec_bolsa_reglas_baremo_propietario
    USING (current_user = 'vec_bolsa_reglas_baremo_propietario')
    WITH CHECK (current_user = 'vec_bolsa_reglas_baremo_propietario');

CREATE FUNCTION vec_bolsa_reglas_baremo.confirmar_cambio_atestado_v2(
    p_plan_canonico bytea,
    p_decision_canonica bytea,
    p_payload_vec_ad_2 bytea,
    p_sobre_cose_sign1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea,
    p_capacidad jsonb
)
RETURNS TABLE (
    recibo_ref text,
    recibo_version numeric,
    huella_recibo_sha256 text,
    contenido_ref text,
    contenido_version numeric,
    revision numeric,
    estado text,
    huella_estado_sha256 text,
    registro_vec_ref text,
    consumo_vec_ref text,
    auditoria_vec_ref text,
    transaccion_ref text,
    auditoria_local_ref text,
    outbox_ref text,
    confirmada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET timezone = 'UTC'
SET lock_timeout = '5s'
SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '15s'
AS $funcion$
DECLARE
    v_tenant constant text := 'diputacion_granada';
    v_plan jsonb;
    v_version jsonb;
    v_decision jsonb;
    v_motivo jsonb;
    v_version_canonica bytea;
    v_motivo_canonico bytea;
    v_operacion text;
    v_accion text;
    v_estado text;
    v_intencion_ref text;
    v_intencion_version numeric(20, 0);
    v_intencion_huella text;
    v_contenido_ref text;
    v_contenido_version numeric(20, 0);
    v_huella_contenido text;
    v_revision numeric(20, 0);
    v_huella_estado text;
    v_esperado_revision numeric(20, 0);
    v_esperado_huella text;
    v_evidencia_ref text;
    v_evidencia_version numeric(20, 0);
    v_evidencia_huella text;
    v_principal_ref text;
    v_sujeto_seudonimo_hmac text;
    v_recurso_ref text;
    v_correlacion_ref text;
    v_huella_contexto text;
    v_huella_contexto_calculada text;
    v_convocatoria_ref text;
    v_expediente_ref text;
    v_instante_transicion timestamptz(6);
    v_actor_version text;
    v_instante_version timestamptz(6);
    v_motivo_version jsonb;
    v_evidencia_version_json jsonb;
    v_evidencia_valida_hasta timestamptz(6);
    v_huella_plan text;
    v_efecto_ref text;
    v_ahora timestamptz(6);
    v_cotejo_post_vec timestamptz(6);
    v_actual record;
    v_cabeza_auditoria record;
    v_vec record;
    v_vec_reconciliado record;
    v_consumo_local_canonico bytea;
    v_huella_consumo_local text;
    v_consumo_evidencia_ref text;
    v_consumo_evidencia_canonico bytea;
    v_huella_consumo_evidencia text;
    v_transaccion_canonica bytea;
    v_huella_transaccion text;
    v_transaccion_ref text;
    v_auditoria_canonica bytea;
    v_huella_auditoria text;
    v_auditoria_ref text;
    v_evento_canonico bytea;
    v_huella_evento text;
    v_outbox_ref text;
    v_recibo_canonico bytea;
    v_huella_recibo text;
    v_recibo_ref text;
    v_huella_nonce text;
    v_identidades_bloqueo text[];
    v_clave_bloqueo bigint;
BEGIN
    IF current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25001',
            MESSAGE = 'confirmacion atestada V2 requiere SERIALIZABLE';
    END IF;
    IF p_plan_canonico IS NULL
       OR octet_length(p_plan_canonico) NOT BETWEEN 2 AND 8388608
       OR p_decision_canonica IS NULL
       OR p_capacidad IS NULL OR jsonb_typeof(p_capacidad) <> 'object' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'material de cambio atestado V2 invalido';
    END IF;
    BEGIN
        v_plan := convert_from(p_plan_canonico, 'UTF8')::jsonb;
        v_decision := convert_from(p_decision_canonica, 'UTF8')::jsonb;
        v_version_canonica := decode(
            v_plan ->> 'version_resultado_canonica', 'base64'
        );
        v_motivo_canonico := decode(v_plan ->> 'motivo_canonico', 'base64');
        v_version := convert_from(v_version_canonica, 'UTF8')::jsonb;
        v_motivo := convert_from(v_motivo_canonico, 'UTF8')::jsonb;
        v_operacion := v_plan ->> 'operacion';
        v_accion := v_plan ->> 'accion';
        v_intencion_ref := v_plan -> 'intencion' ->> 'referencia';
        v_intencion_version :=
            (v_plan -> 'intencion' ->> 'version')::numeric;
        v_intencion_huella := v_plan -> 'intencion' ->> 'huella_sha256';
        v_contenido_ref :=
            v_plan -> 'vinculo_resultado' -> 'contenido' ->> 'referencia';
        v_contenido_version :=
            (v_plan -> 'vinculo_resultado' -> 'contenido' ->>
             'version')::numeric;
        v_huella_contenido :=
            v_plan -> 'vinculo_resultado' -> 'contenido' ->>
            'huella_sha256';
        v_revision :=
            (v_plan -> 'vinculo_resultado' ->> 'revision')::numeric;
        v_huella_estado :=
            v_plan -> 'vinculo_resultado' ->> 'huella_estado_sha256';
        v_esperado_revision := CASE
            WHEN v_plan -> 'cas_esperado' = 'null'::jsonb THEN NULL
            ELSE (v_plan -> 'cas_esperado' ->> 'revision')::numeric
        END;
        v_esperado_huella :=
            v_plan -> 'cas_esperado' ->> 'huella_estado_sha256';
        v_evidencia_ref :=
            v_plan -> 'vinculo_evidencia' ->> 'referencia';
        v_evidencia_version := CASE
            WHEN v_plan -> 'vinculo_evidencia' = 'null'::jsonb THEN NULL
            ELSE (v_plan -> 'vinculo_evidencia' ->> 'version')::numeric
        END;
        v_evidencia_huella :=
            v_plan -> 'vinculo_evidencia' ->> 'huella_sha256';
        v_principal_ref := v_plan ->> 'principal_ref';
        v_recurso_ref := v_plan ->> 'recurso_ref';
        v_correlacion_ref := v_plan ->> 'correlacion_ref';
        v_huella_contexto :=
            v_plan ->> 'huella_contexto_recurso_sha256';
        v_convocatoria_ref := v_plan ->> 'convocatoria_ref';
        v_expediente_ref := v_plan ->> 'expediente_ref';
        v_instante_transicion :=
            (v_plan ->> 'instante_transicion')::timestamptz;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
            OR numeric_value_out_of_range OR datetime_field_overflow
            OR character_not_in_repertoire OR untranslatable_character THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'plan atestado V2 no interpretable';
    END;

    IF jsonb_typeof(v_plan) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(v_plan)) <> 26
       OR NOT (v_plan ?& ARRAY[
           'esquema', 'operacion', 'intencion', 'cas_esperado',
           'version_resultado_canonica',
           'huella_version_resultado_sha256', 'vinculo_resultado',
           'vinculo_evidencia', 'principal_ref',
           'sujeto_seudonimo_hmac', 'motivo_canonico',
           'huella_motivo_sha256', 'correlacion_ref',
           'instante_transicion', 'accion', 'modulo_id', 'tipo_recurso',
           'perfil_proteccion', 'recurso_ref', 'convocatoria_ref',
           'expediente_ref', 'huella_contexto_recurso_sha256',
           'finalidad', 'campos', 'requisitos_ejecucion', 'componentes'
       ])
       OR v_plan ->> 'esquema' IS DISTINCT FROM
          'vec.bolsa.gobierno-reglas-baremo.plan-cambio.v2'
       OR jsonb_typeof(v_plan -> 'intencion') <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(
              v_plan -> 'intencion'
          )) <> 3
       OR jsonb_typeof(v_plan -> 'vinculo_resultado') <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(
              v_plan -> 'vinculo_resultado'
          )) <> 3
       OR jsonb_typeof(
              v_plan -> 'vinculo_resultado' -> 'contenido'
          ) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(
              v_plan -> 'vinculo_resultado' -> 'contenido'
          )) <> 3
       OR v_plan ->> 'modulo_id' IS DISTINCT FROM 'bolsa'
       OR v_plan ->> 'tipo_recurso' IS DISTINCT FROM
          'version_reglas_baremo_gobernada'
       OR v_plan ->> 'perfil_proteccion' IS DISTINCT FROM 'interno_alto'
       OR jsonb_typeof(
              v_plan -> 'sujeto_seudonimo_hmac'
          ) <> 'string'
       OR v_plan ->> 'finalidad' IS DISTINCT FROM
          'gobierno_reglas_baremo'
       OR v_plan -> 'campos' IS DISTINCT FROM
          '["auditoria","estado_reglas_baremo","salida_eventos"]'::jsonb
       OR v_plan -> 'requisitos_ejecucion' IS DISTINCT FROM
          '["alcance_resuelto_servidor","cotejo_evidencia_verificador_confiable","decision_vec_v2_consumible","consumo_atestado_vec_ad2_mismo_commit","commit_serializable_atomico","reloj_autoritativo_frescura_cotejada","recibo_durable_reconciliable"]'::jsonb
       OR v_plan -> 'componentes' IS DISTINCT FROM
          '["contenido","version","puntero_cas","vinculo_evidencia","vec","auditoria","outbox","recibo"]'::jsonb
       OR v_intencion_version IS DISTINCT FROM 2::numeric
       OR v_intencion_ref IS DISTINCT FROM
          'intencion:reglas-baremo:v2:' || v_intencion_huella
       OR vec_bolsa_reglas_baremo.huella_sha256_valida(
              v_intencion_huella
          ) IS NOT TRUE
       OR (v_contenido_ref ~ '^rgl_[0-9a-f]{32}$') IS NOT TRUE
       OR (v_convocatoria_ref ~ '^con_[0-9a-f]{32}$') IS NOT TRUE
       OR (v_expediente_ref ~ '^exp_[0-9a-f]{32}$') IS NOT TRUE
       OR vec_bolsa_reglas_baremo.version_valida(
              v_contenido_version
          ) IS NOT TRUE
       OR vec_bolsa_reglas_baremo.huella_sha256_valida(
              v_huella_contenido
          ) IS NOT TRUE
       OR vec_bolsa_reglas_baremo.version_valida(v_revision) IS NOT TRUE
       OR vec_bolsa_reglas_baremo.huella_sha256_valida(
              v_huella_estado
          ) IS NOT TRUE
       OR encode(sha256(v_version_canonica), 'hex') IS DISTINCT FROM
          v_plan ->> 'huella_version_resultado_sha256'
       OR v_huella_estado IS DISTINCT FROM
          v_plan ->> 'huella_version_resultado_sha256'
       OR encode(sha256(v_motivo_canonico), 'hex') IS DISTINCT FROM
          v_plan ->> 'huella_motivo_sha256'
       OR v_recurso_ref IS DISTINCT FROM
          'reglas-baremo:' || v_huella_estado
       OR (v_correlacion_ref ~ '^correlacion_[0-9a-f]{32}$') IS NOT TRUE
       OR vec_bolsa_reglas_baremo.huella_sha256_valida(
              v_huella_contexto
          ) IS NOT TRUE
       OR (v_plan ->> 'sujeto_seudonimo_hmac' ~
           '^hmac-sha256:reglas_baremo_v2:[0-9a-f]{64}$') IS NOT TRUE
       OR to_char(v_instante_transicion AT TIME ZONE 'UTC',
              'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM
          v_plan ->> 'instante_transicion' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contrato de plan atestado V2 invalido';
    END IF;

    v_huella_plan := encode(sha256(p_plan_canonico), 'hex');
    v_huella_contexto_calculada := encode(sha256(convert_to(
        '{"ambitos":{"convocatoria_ref":"' || v_convocatoria_ref ||
        '","expediente_ref":"' || v_expediente_ref ||
        '"},"atributos":{}}', 'UTF8'
    )), 'hex');
    v_sujeto_seudonimo_hmac :=
        v_plan ->> 'sujeto_seudonimo_hmac';
    v_efecto_ref := 'efecto:reglas-baremo:v2:' || v_huella_plan;
    IF v_huella_contexto IS DISTINCT FROM v_huella_contexto_calculada
       OR p_capacidad ->> 'efecto_ref' IS DISTINCT FROM v_efecto_ref
       OR p_capacidad ->> 'huella_efecto_sha256' IS DISTINCT FROM
          v_huella_plan
       OR p_capacidad ->> 'principal_id' IS DISTINCT FROM v_principal_ref
       OR p_capacidad ->> 'sujeto_ref' IS DISTINCT FROM
          v_sujeto_seudonimo_hmac
       OR p_capacidad ->> 'accion' IS DISTINCT FROM v_accion
       OR p_capacidad ->> 'finalidad' IS DISTINCT FROM
          'gobierno_reglas_baremo'
       OR p_capacidad ->> 'recurso_ref' IS DISTINCT FROM v_recurso_ref
       OR p_capacidad ->> 'contexto_recurso_huella_sha256'
          IS DISTINCT FROM v_huella_contexto
       OR p_capacidad ->> 'correlacion_ref' IS DISTINCT FROM
          v_correlacion_ref
       OR p_capacidad ->> 'huella_motivo_sha256' IS DISTINCT FROM
          encode(sha256(v_motivo_canonico), 'hex')
       OR v_decision ->> 'decision_ref' IS DISTINCT FROM
          p_capacidad ->> 'decision_ref'
       OR v_decision ->> 'principal_id' IS DISTINCT FROM v_principal_ref
       OR v_decision ->> 'accion' IS DISTINCT FROM v_accion
       OR v_decision ->> 'modulo_id' IS DISTINCT FROM 'bolsa'
       OR v_decision ->> 'tipo_recurso' IS DISTINCT FROM
          'version_reglas_baremo_gobernada'
       OR v_decision ->> 'garantia_minima' IS DISTINCT FROM 'alto'
       OR v_decision -> 'campos_permitidos' IS DISTINCT FROM
          '["auditoria","estado_reglas_baremo","salida_eventos"]'::jsonb
       OR v_decision -> 'obligaciones' IS DISTINCT FROM '[]'::jsonb
       OR v_decision ->> 'finalidad' IS DISTINCT FROM
          'gobierno_reglas_baremo'
       OR v_decision ->> 'recurso_ref' IS DISTINCT FROM v_recurso_ref
       OR v_decision ->> 'contexto_recurso_huella_sha256'
          IS DISTINCT FROM v_huella_contexto
       OR v_decision ->> 'correlacion_ref' IS DISTINCT FROM
          v_correlacion_ref
       OR v_decision ->> 'motivo_huella_sha256' IS DISTINCT FROM
          encode(sha256(v_motivo_canonico), 'hex')
       OR jsonb_typeof(
              v_decision -> 'vinculo_autenticacion_actor'
          ) <> 'object'
       OR v_decision -> 'vinculo_autenticacion_actor' ->>
          'garantia_observada' IS DISTINCT FROM 'alto'
       OR v_decision -> 'vinculo_autenticacion_actor' ->>
          'metodo_observado' IS NULL
       OR v_decision -> 'vinculo_autenticacion_actor' ->>
          'metodo_observado' = 'demo'
       OR NOT COALESCE((
           (v_decision -> 'vinculo_autenticacion_actor' ->> 'superficie' =
                'interna_corporativa'
            AND v_decision -> 'vinculo_autenticacion_actor' ->
                'cuenta_privilegiada' = 'false'::jsonb)
           OR
           (v_decision -> 'vinculo_autenticacion_actor' ->> 'superficie' =
                'administracion_privilegiada'
            AND v_decision -> 'vinculo_autenticacion_actor' ->
                'cuenta_privilegiada' = 'true'::jsonb)
       ), false) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'plan, decision y capacidad VEC no estan ligados';
    END IF;

    IF jsonb_typeof(v_version) IS DISTINCT FROM 'object'
       OR v_version ->> 'esquema' IS DISTINCT FROM
          'vec.bolsa.gobierno-reglas-baremo.v1'
       OR (v_version ->> 'revision')::numeric IS DISTINCT FROM v_revision
       OR v_version -> 'referencia_contenido' ->> 'referencia'
          IS DISTINCT FROM v_contenido_ref
       OR (v_version -> 'referencia_contenido' ->> 'version')::numeric
          IS DISTINCT FROM v_contenido_version
       OR v_version -> 'referencia_contenido' ->> 'huella_sha256'
          IS DISTINCT FROM v_huella_contenido
       OR v_version -> 'contenido' -> 'identidad' ->> 'referencia'
          IS DISTINCT FROM v_contenido_ref
       OR (v_version -> 'contenido' -> 'identidad' ->> 'version')::numeric
          IS DISTINCT FROM v_contenido_version
       OR v_version -> 'contenido' -> 'identidad' ->> 'convocatoria_ref'
          IS DISTINCT FROM v_convocatoria_ref
       OR v_version -> 'contenido' -> 'identidad' ->> 'expediente_ref'
          IS DISTINCT FROM v_expediente_ref THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'version resultado no coincide con su plan';
    END IF;

    CASE v_operacion
    WHEN 'alta_borrador' THEN
        v_estado := 'borrador';
        v_accion := 'bolsa.reglas_baremo.borrador.crear';
        IF v_revision <> 1 OR v_plan -> 'cas_esperado' <> 'null'::jsonb
           OR v_plan -> 'vinculo_evidencia' <> 'null'::jsonb THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'alta V2 invalida';
        END IF;
        v_actor_version := v_version ->> 'creada_por';
        v_instante_version := (v_version ->> 'creada_en')::timestamptz;
        v_motivo_version := v_version -> 'motivo_creacion';
    WHEN 'publicar' THEN
        v_estado := 'publicada';
        v_accion := 'bolsa.reglas_baremo.publicar';
        v_actor_version := v_version -> 'publicacion' ->> 'actor_ref';
        v_instante_version :=
            (v_version -> 'publicacion' ->> 'instante')::timestamptz;
        v_motivo_version := v_version -> 'publicacion' -> 'motivo';
        v_evidencia_version_json :=
            v_version -> 'publicacion' -> 'aprobacion' -> 'atestacion';
        v_evidencia_valida_hasta :=
            (v_version -> 'publicacion' -> 'aprobacion' ->>
             'valida_hasta')::timestamptz;
    WHEN 'activar' THEN
        v_estado := 'activa';
        v_accion := 'bolsa.reglas_baremo.activar';
        v_actor_version := v_version -> 'activacion' ->> 'actor_ref';
        v_instante_version :=
            (v_version -> 'activacion' ->> 'instante')::timestamptz;
        v_motivo_version := v_version -> 'activacion' -> 'motivo';
        v_evidencia_version_json :=
            v_version -> 'activacion' -> 'dependencias' -> 'atestacion';
        v_evidencia_valida_hasta :=
            (v_version -> 'activacion' -> 'dependencias' ->>
             'valida_hasta')::timestamptz;
    WHEN 'sustituir', 'retirar', 'descartar' THEN
        v_estado := CASE v_operacion
            WHEN 'sustituir' THEN 'sustituida'
            WHEN 'retirar' THEN 'retirada'
            ELSE 'descartada'
        END;
        v_accion := 'bolsa.reglas_baremo.' || v_operacion;
        v_actor_version := v_version -> 'terminal' ->> 'actor_ref';
        v_instante_version :=
            (v_version -> 'terminal' ->> 'instante')::timestamptz;
        v_motivo_version := v_version -> 'terminal' -> 'motivo';
        v_evidencia_version_json :=
            v_version -> 'terminal' -> 'autoridad' -> 'atestacion';
        v_evidencia_valida_hasta :=
            (v_version -> 'terminal' -> 'autoridad' ->>
             'valida_hasta')::timestamptz;
    ELSE
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'operacion V2 desconocida';
    END CASE;

    IF v_plan ->> 'accion' IS DISTINCT FROM v_accion
       OR v_version ->> 'estado' IS DISTINCT FROM v_estado
       OR v_actor_version IS DISTINCT FROM v_principal_ref
       OR v_instante_version IS DISTINCT FROM v_instante_transicion
       OR v_motivo ->> 'esquema' IS DISTINCT FROM
          'vec.autorizacion.motivo.v2.referencia-opaca-catalogada'
       OR v_motivo_version -> 'catalogo' ->> 'referencia'
          IS DISTINCT FROM v_motivo -> 'referencia' ->> 'catalogo_id'
       OR (v_motivo_version -> 'catalogo' ->> 'version')::numeric
          IS DISTINCT FROM
          (v_motivo -> 'referencia' ->> 'catalogo_version')::numeric
       OR v_motivo_version -> 'catalogo' ->> 'huella_sha256'
          IS DISTINCT FROM
          v_motivo -> 'referencia' ->> 'catalogo_huella_sha256'
       OR v_motivo_version ->> 'clave' IS DISTINCT FROM
          v_motivo -> 'referencia' ->> 'entrada_clave' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'actor, motivo o instante no incorporados exactamente';
    END IF;

    IF v_operacion <> 'alta_borrador' THEN
        IF jsonb_typeof(v_plan -> 'cas_esperado') <> 'object'
           OR jsonb_typeof(v_plan -> 'vinculo_evidencia') <> 'object'
           OR v_esperado_revision + 1 IS DISTINCT FROM v_revision
           OR v_plan -> 'cas_esperado' -> 'contenido'
              IS DISTINCT FROM
              v_plan -> 'vinculo_resultado' -> 'contenido'
           OR v_evidencia_ref IS DISTINCT FROM
              'atestacion:reglas-baremo:v2:' || v_evidencia_huella
           OR v_evidencia_version IS DISTINCT FROM 2::numeric
           OR vec_bolsa_reglas_baremo.huella_sha256_valida(
                  v_evidencia_huella
              ) IS NOT TRUE
           OR v_evidencia_version_json ->> 'referencia'
              IS DISTINCT FROM v_evidencia_ref
           OR (v_evidencia_version_json ->> 'version')::numeric
              IS DISTINCT FROM v_evidencia_version
           OR v_evidencia_version_json ->> 'huella_sha256'
              IS DISTINCT FROM v_evidencia_huella
           OR v_evidencia_valida_hasta IS NULL
           OR NOT isfinite(v_evidencia_valida_hasta)
           OR v_evidencia_valida_hasta <= v_instante_transicion THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'CAS o evidencia V2 no incorporados exactamente';
        END IF;
    END IF;

    -- La reserva de tablas, siempre en este orden, sucede antes de VEC-AD-2.
    -- Impide que un DDL concurrente introduzca una espera nueva despues del
    -- cotejo atestado; lock_timeout mantiene el rechazo previo acotado.
    LOCK TABLE
        vec_bolsa_reglas_baremo.contenido_reglas_baremo,
        vec_bolsa_reglas_baremo.version_reglas_baremo,
        vec_bolsa_reglas_baremo.estado_actual,
        vec_bolsa_reglas_baremo.uso_decision,
        vec_bolsa_reglas_baremo.uso_prueba_transicion,
        vec_bolsa_reglas_baremo.auditoria,
        vec_bolsa_reglas_baremo.auditoria_actual,
        vec_bolsa_reglas_baremo.outbox,
        vec_bolsa_reglas_baremo.intencion_confirmada,
        vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2
        IN ROW EXCLUSIVE MODE;

    -- Se adquieren tambien todas las identidades potencialmente bloqueantes
    -- antes de invocar VEC-AD-2. Tras su revalidacion solo quedan escrituras
    -- sobre claves ya serializadas, filas ya bloqueadas y tablas reservadas.
    v_identidades_bloqueo := ARRAY[
        'agregado:' || v_tenant || ':' || v_contenido_ref || ':' ||
            v_contenido_version::text,
        'plan:' || v_huella_plan,
        'intencion:' || v_intencion_ref || ':' ||
            v_intencion_version::text,
        'resultado:' || v_huella_estado
    ];
    IF v_evidencia_ref IS NOT NULL THEN
        v_identidades_bloqueo := array_append(
            v_identidades_bloqueo,
            'evidencia:' || v_evidencia_ref || ':' ||
                v_evidencia_version::text || ':' || v_evidencia_huella
        );
    END IF;
    FOR v_clave_bloqueo IN
        SELECT DISTINCT hashtextextended(identidad, 0)
          FROM unnest(v_identidades_bloqueo) AS identidad
         ORDER BY 1
    LOOP
        PERFORM pg_advisory_xact_lock(v_clave_bloqueo);
    END LOOP;
    SELECT ultima_secuencia, ultima_huella_sha256
      INTO STRICT v_cabeza_auditoria
      FROM vec_bolsa_reglas_baremo.auditoria_actual AS cabeza
     WHERE cabeza.tenant_id = v_tenant
     FOR UPDATE;

    IF EXISTS (
        SELECT 1
          FROM vec_bolsa_reglas_baremo.intencion_confirmada AS confirmacion
         WHERE confirmacion.tenant_id = v_tenant
           AND confirmacion.intencion_ref = v_intencion_ref
           AND confirmacion.intencion_version = v_intencion_version
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'intencion V2 ya confirmada; use reconciliacion exacta';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM vec_bolsa_reglas_baremo.version_reglas_baremo
               AS version_existente
         WHERE version_existente.tenant_id = v_tenant
           AND version_existente.huella_estado_sha256 = v_huella_estado
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'resultado V2 ya materializado';
    END IF;
    IF v_evidencia_ref IS NOT NULL AND EXISTS (
        SELECT 1
          FROM vec_bolsa_reglas_baremo.uso_prueba_transicion AS uso_evidencia
         WHERE uso_evidencia.tenant_id = v_tenant
           AND uso_evidencia.prueba_ref = v_evidencia_ref
           AND uso_evidencia.prueba_version = v_evidencia_version
           AND uso_evidencia.prueba_huella_sha256 = v_evidencia_huella
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'evidencia V2 ya consumida';
    END IF;

    IF v_operacion = 'alta_borrador' THEN
        IF EXISTS (
            SELECT 1
              FROM vec_bolsa_reglas_baremo.estado_actual AS estado_existente
             WHERE estado_existente.tenant_id = v_tenant
               AND estado_existente.contenido_ref = v_contenido_ref
               AND estado_existente.contenido_version = v_contenido_version
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'CAS V2: contenido ya existente';
        END IF;
    ELSE
        SELECT actual.*, version.estado
          INTO v_actual
          FROM vec_bolsa_reglas_baremo.estado_actual AS actual
          JOIN vec_bolsa_reglas_baremo.version_reglas_baremo AS version
            ON version.tenant_id = actual.tenant_id
           AND version.contenido_ref = actual.contenido_ref
           AND version.contenido_version = actual.contenido_version
           AND version.revision = actual.revision
           AND version.huella_estado_sha256 = actual.huella_estado_sha256
         WHERE actual.tenant_id = v_tenant
           AND actual.contenido_ref = v_contenido_ref
           AND actual.contenido_version = v_contenido_version
         FOR UPDATE OF actual;
        IF NOT FOUND
           OR v_actual.huella_contenido_sha256 IS DISTINCT FROM
              v_huella_contenido
           OR v_actual.revision IS DISTINCT FROM v_esperado_revision
           OR v_actual.huella_estado_sha256 IS DISTINCT FROM
              v_esperado_huella
           OR (v_operacion = 'publicar' AND
               v_actual.estado IS DISTINCT FROM 'borrador')
           OR (v_operacion = 'activar' AND
               v_actual.estado IS DISTINCT FROM 'publicada')
           OR (v_operacion IN ('sustituir', 'retirar')
               AND v_actual.estado IS DISTINCT FROM 'activa')
           OR (v_operacion = 'descartar'
               AND v_actual.estado IS DISTINCT FROM 'borrador') THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'CAS V2 obsoleto';
        END IF;
    END IF;

    v_ahora := clock_timestamp();
    IF v_ahora < v_instante_transicion
       OR v_ahora - v_instante_transicion > interval '30 seconds'
       OR (v_evidencia_valida_hasta IS NOT NULL
           AND v_ahora >= v_evidencia_valida_hasta) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'reloj o evidencia V2 fuera de vigencia';
    END IF;

    SELECT * INTO v_vec
      FROM vec_autorizacion_atestada_v2.
           registrar_y_consumir_decision_v2_atestada(
               p_decision_canonica, v_motivo_canonico,
               p_payload_vec_ad_2, p_sobre_cose_sign1,
               p_evidencia_verificacion, p_raiz_publica_spki, p_capacidad
           );
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'VEC-AD-2 rechazo el consumo atestado';
    END IF;
    SELECT * INTO v_vec_reconciliado
      FROM vec_autorizacion_atestada_v2.
           obtener_vinculo_consumo_modular_v2(
               v_vec.registro_ref, v_vec.consumo_ref, v_vec.auditoria_ref,
               p_capacidad ->> 'decision_ref',
               p_capacidad ->> 'huella_decision_sha256',
               v_efecto_ref, v_huella_plan
           );
    IF NOT FOUND
       OR v_vec_reconciliado.registro_ref <> v_vec.registro_ref
       OR v_vec_reconciliado.consumo_ref <> v_vec.consumo_ref
       OR v_vec_reconciliado.auditoria_ref <> v_vec.auditoria_ref
       OR v_vec_reconciliado.consumida_en <> v_vec.registrada_en THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'consumo VEC-AD-2 no reconciliable tras registro';
    END IF;
    -- La hora incorporada al recibo procede de VEC-AD-2, pero la vigencia se
    -- vuelve a medir ahora, despues de que la puerta y el helper hayan
    -- terminado. Reutilizar registrada_en dejaria sin observar una caducidad
    -- producida durante la propia llamada.
    v_cotejo_post_vec := date_trunc(
        'microseconds', clock_timestamp()
    );
    IF v_cotejo_post_vec < v_vec.registrada_en
       OR v_cotejo_post_vec < v_instante_transicion
       OR v_cotejo_post_vec - v_instante_transicion > interval '30 seconds'
       OR (v_evidencia_valida_hasta IS NOT NULL
           AND v_cotejo_post_vec >= v_evidencia_valida_hasta) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'reloj o evidencia V2 vencieron durante VEC-AD-2';
    END IF;
    v_ahora := v_vec.registrada_en;

    IF v_operacion = 'alta_borrador' THEN
        INSERT INTO vec_bolsa_reglas_baremo.contenido_reglas_baremo(
            tenant_id, contenido_ref, contenido_version,
            huella_contenido_sha256, creada_en
        ) VALUES (
            v_tenant, v_contenido_ref, v_contenido_version,
            v_huella_contenido, v_ahora
        );
    END IF;
    INSERT INTO vec_bolsa_reglas_baremo.version_reglas_baremo(
        tenant_id, contenido_ref, contenido_version,
        huella_contenido_sha256, revision, estado, version_canonica,
        huella_estado_sha256, operacion_origen, intencion_ref,
        intencion_version, intencion_huella_sha256, registrada_en
    ) VALUES (
        v_tenant, v_contenido_ref, v_contenido_version,
        v_huella_contenido, v_revision, v_estado, v_version_canonica,
        v_huella_estado, v_operacion, v_intencion_ref,
        v_intencion_version, v_intencion_huella, v_ahora
    );
    IF v_operacion = 'alta_borrador' THEN
        INSERT INTO vec_bolsa_reglas_baremo.estado_actual(
            tenant_id, contenido_ref, contenido_version,
            huella_contenido_sha256, revision, huella_estado_sha256,
            actualizada_en
        ) VALUES (
            v_tenant, v_contenido_ref, v_contenido_version,
            v_huella_contenido, v_revision, v_huella_estado, v_ahora
        );
    ELSE
        UPDATE vec_bolsa_reglas_baremo.estado_actual AS actualiza
           SET revision = v_revision,
               huella_estado_sha256 = v_huella_estado,
               actualizada_en = v_ahora
         WHERE actualiza.tenant_id = v_tenant
           AND actualiza.contenido_ref = v_contenido_ref
           AND actualiza.contenido_version = v_contenido_version
           AND actualiza.revision = v_esperado_revision
           AND actualiza.huella_estado_sha256 = v_esperado_huella;
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'CAS V2 cambio tras prebloqueo';
        END IF;
    END IF;

    v_consumo_local_canonico := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.reglas-baremo.consumo-vec-ad2.v2',
        'decision_ref', p_capacidad ->> 'decision_ref',
        'huella_decision_sha256',
            p_capacidad ->> 'huella_decision_sha256',
        'registro_vec_ref', v_vec.registro_ref,
        'consumo_vec_ref', v_vec.consumo_ref,
        'auditoria_vec_ref', v_vec.auditoria_ref,
        'huella_auditoria_vec_sha256',
            v_vec_reconciliado.huella_auditoria_sha256,
        'efecto_ref', v_efecto_ref,
        'huella_efecto_sha256', v_huella_plan,
        'consumida_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_consumo_local := encode(
        sha256(v_consumo_local_canonico), 'hex'
    );
    INSERT INTO vec_bolsa_reglas_baremo.uso_decision(
        tenant_id, decision_ref, consumo_autorizacion_ref,
        consumo_autorizacion_version,
        huella_consumo_autorizacion_sha256, principal_ref, operacion,
        accion, recurso_ref, correlacion_ref, huella_decision_sha256,
        huella_contexto_recurso_sha256, contenido_ref,
        contenido_version, revision, huella_estado_sha256,
        huella_efecto_sha256, consumida_en
    ) VALUES (
        v_tenant, p_capacidad ->> 'decision_ref', v_vec.consumo_ref, 1,
        v_huella_consumo_local, v_principal_ref, v_operacion, v_accion,
        v_recurso_ref, v_correlacion_ref,
        p_capacidad ->> 'huella_decision_sha256', v_huella_contexto,
        v_contenido_ref, v_contenido_version, v_revision,
        v_huella_estado, v_huella_plan, v_ahora
    );

    IF v_evidencia_ref IS NOT NULL THEN
        v_consumo_evidencia_canonico := convert_to(jsonb_build_object(
            'esquema', 'vec.bolsa.reglas-baremo.consumo-evidencia.v2',
            'evidencia_ref', v_evidencia_ref,
            'evidencia_version', v_evidencia_version,
            'evidencia_huella_sha256', v_evidencia_huella,
            'huella_plan_sha256', v_huella_plan,
            'consumida_en', to_char(v_ahora AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        )::text, 'UTF8');
        v_huella_consumo_evidencia := encode(
            sha256(v_consumo_evidencia_canonico), 'hex'
        );
        v_consumo_evidencia_ref :=
            'consumo-evidencia-reglas-v2:' ||
            v_huella_consumo_evidencia;
        INSERT INTO vec_bolsa_reglas_baremo.uso_prueba_transicion(
            tenant_id, prueba_ref, prueba_version, prueba_huella_sha256,
            consumo_prueba_ref, consumo_prueba_version,
            huella_consumo_prueba_sha256, intencion_ref,
            intencion_version, contenido_ref, contenido_version,
            revision, huella_estado_sha256, consumida_en
        ) VALUES (
            v_tenant, v_evidencia_ref, v_evidencia_version,
            v_evidencia_huella, v_consumo_evidencia_ref, 1,
            v_huella_consumo_evidencia, v_intencion_ref,
            v_intencion_version, v_contenido_ref, v_contenido_version,
            v_revision, v_huella_estado, v_ahora
        );
    END IF;

    v_transaccion_canonica := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.reglas-baremo.transaccion.v2',
        'huella_plan_sha256', v_huella_plan,
        'consumo_vec_ref', v_vec.consumo_ref,
        'consumo_evidencia_ref', v_consumo_evidencia_ref,
        'confirmada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_transaccion := encode(sha256(v_transaccion_canonica), 'hex');
    v_transaccion_ref :=
        'transaccion-reglas-v2:' || v_huella_transaccion;

    v_auditoria_ref := 'auditoria-reglas-v2:' || encode(sha256(
        convert_to(v_vec.consumo_ref || ':' ||
            v_cabeza_auditoria.ultima_huella_sha256, 'UTF8')
    ), 'hex');
    v_auditoria_canonica := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.reglas-baremo.auditoria.v2',
        'auditoria_ref', v_auditoria_ref,
        'secuencia', v_cabeza_auditoria.ultima_secuencia + 1,
        'huella_anterior_sha256',
            v_cabeza_auditoria.ultima_huella_sha256,
        'operacion', v_operacion,
        'intencion_ref', v_intencion_ref,
        'decision_ref', p_capacidad ->> 'decision_ref',
        'consumo_vec_ref', v_vec.consumo_ref,
        'auditoria_vec_ref', v_vec.auditoria_ref,
        'huella_plan_sha256', v_huella_plan,
        'registrada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_auditoria := encode(sha256(
        decode(v_cabeza_auditoria.ultima_huella_sha256, 'hex') ||
        v_auditoria_canonica
    ), 'hex');
    INSERT INTO vec_bolsa_reglas_baremo.auditoria(
        tenant_id, secuencia, auditoria_ref, auditoria_version,
        decision_ref, consumo_autorizacion_ref, operacion,
        registro_canonico, huella_anterior_sha256,
        huella_auditoria_sha256, registrada_en
    ) VALUES (
        v_tenant, v_cabeza_auditoria.ultima_secuencia + 1,
        v_auditoria_ref, 1, p_capacidad ->> 'decision_ref',
        v_vec.consumo_ref, v_operacion, v_auditoria_canonica,
        v_cabeza_auditoria.ultima_huella_sha256,
        v_huella_auditoria, v_ahora
    );
    UPDATE vec_bolsa_reglas_baremo.auditoria_actual AS cabeza_actualiza
       SET ultima_secuencia = v_cabeza_auditoria.ultima_secuencia + 1,
           ultima_huella_sha256 = v_huella_auditoria,
           actualizada_en = v_ahora
     WHERE cabeza_actualiza.tenant_id = v_tenant
       AND cabeza_actualiza.ultima_secuencia =
           v_cabeza_auditoria.ultima_secuencia
       AND cabeza_actualiza.ultima_huella_sha256 =
           v_cabeza_auditoria.ultima_huella_sha256;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'cabeza local cambio tras prebloqueo V2';
    END IF;

    v_evento_canonico := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.reglas-baremo.estado-confirmado.v2',
        'operacion', v_operacion,
        'contenido_ref', v_contenido_ref,
        'contenido_version', v_contenido_version,
        'revision', v_revision,
        'estado', v_estado,
        'huella_estado_sha256', v_huella_estado,
        'huella_plan_sha256', v_huella_plan,
        'transaccion_ref', v_transaccion_ref,
        'auditoria_local_ref', v_auditoria_ref,
        'auditoria_vec_ref', v_vec.auditoria_ref,
        'confirmada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_evento := encode(sha256(v_evento_canonico), 'hex');
    v_outbox_ref := 'outbox-reglas-v2:' || v_huella_evento;
    INSERT INTO vec_bolsa_reglas_baremo.outbox(
        tenant_id, outbox_ref, outbox_version, ruta, esquema_evento,
        evento_canonico, huella_evento_sha256, contenido_ref,
        contenido_version, revision, huella_estado_sha256, creada_en
    ) VALUES (
        v_tenant, v_outbox_ref, 1,
        'bolsa.reglas_baremo.estado_confirmado.v2',
        'vec.bolsa.reglas-baremo.estado-confirmado.v2',
        v_evento_canonico, v_huella_evento, v_contenido_ref,
        v_contenido_version, v_revision, v_huella_estado, v_ahora
    );

    INSERT INTO vec_bolsa_reglas_baremo.intencion_confirmada(
        tenant_id, intencion_ref, intencion_version,
        intencion_huella_sha256, operacion, esperado_revision,
        esperado_huella_estado_sha256, contenido_ref,
        contenido_version, huella_contenido_sha256,
        resultado_revision, resultado_estado,
        resultado_huella_estado_sha256, transaccion_ref,
        transaccion_version, huella_transaccion_sha256, decision_ref,
        consumo_autorizacion_ref, prueba_consumo_ref,
        prueba_consumo_version, prueba_consumo_huella_sha256,
        auditoria_ref, outbox_ref, confirmada_en
    ) VALUES (
        v_tenant, v_intencion_ref, v_intencion_version,
        v_intencion_huella, v_operacion, v_esperado_revision,
        v_esperado_huella, v_contenido_ref, v_contenido_version,
        v_huella_contenido, v_revision, v_estado, v_huella_estado,
        v_transaccion_ref, 1, v_huella_transaccion,
        p_capacidad ->> 'decision_ref', v_vec.consumo_ref,
        v_consumo_evidencia_ref,
        CASE WHEN v_consumo_evidencia_ref IS NULL THEN NULL ELSE 1 END,
        v_huella_consumo_evidencia, v_auditoria_ref, v_outbox_ref,
        v_ahora
    );

    v_huella_nonce := encode(sha256(convert_to(
        p_capacidad ->> 'nonce', 'UTF8'
    )), 'hex');
    v_recibo_canonico := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.reglas-baremo.recibo-cambio.v2',
        'huella_plan_sha256', v_huella_plan,
        'intencion_ref', v_intencion_ref,
        'decision_ref', p_capacidad ->> 'decision_ref',
        'huella_decision_sha256',
            p_capacidad ->> 'huella_decision_sha256',
        'registro_vec_ref', v_vec.registro_ref,
        'consumo_vec_ref', v_vec.consumo_ref,
        'auditoria_vec_ref', v_vec.auditoria_ref,
        'huella_auditoria_vec_sha256',
            v_vec_reconciliado.huella_auditoria_sha256,
        'efecto_ref', v_efecto_ref,
        'contenido_ref', v_contenido_ref,
        'contenido_version', v_contenido_version,
        'revision', v_revision,
        'huella_estado_sha256', v_huella_estado,
        'transaccion_ref', v_transaccion_ref,
        'auditoria_local_ref', v_auditoria_ref,
        'outbox_ref', v_outbox_ref,
        'confirmada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_recibo := encode(sha256(v_recibo_canonico), 'hex');
    v_recibo_ref := 'recibo:reglas-baremo:v2:' || v_huella_recibo;
    INSERT INTO vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2(
        tenant_id, recibo_ref, recibo_version, recibo_canonico,
        huella_recibo_sha256, plan_canonico, huella_plan_sha256,
        intencion_ref, intencion_version, intencion_huella_sha256,
        decision_ref, huella_decision_sha256, registro_vec_ref,
        consumo_vec_ref, auditoria_vec_ref,
        huella_auditoria_vec_sha256, efecto_ref,
        huella_efecto_sha256, huella_nonce_sha256, contenido_ref,
        contenido_version, huella_contenido_sha256,
        resultado_revision, resultado_estado,
        resultado_huella_estado_sha256, transaccion_ref,
        transaccion_version, huella_transaccion_sha256,
        auditoria_local_ref, auditoria_local_version,
        huella_auditoria_local_sha256, outbox_ref, outbox_version,
        huella_evento_sha256, confirmada_en
    ) VALUES (
        v_tenant, v_recibo_ref, 2, v_recibo_canonico,
        v_huella_recibo, p_plan_canonico, v_huella_plan,
        v_intencion_ref, v_intencion_version, v_intencion_huella,
        p_capacidad ->> 'decision_ref',
        p_capacidad ->> 'huella_decision_sha256', v_vec.registro_ref,
        v_vec.consumo_ref, v_vec.auditoria_ref,
        v_vec_reconciliado.huella_auditoria_sha256, v_efecto_ref,
        v_huella_plan, v_huella_nonce, v_contenido_ref,
        v_contenido_version, v_huella_contenido, v_revision, v_estado,
        v_huella_estado, v_transaccion_ref, 1, v_huella_transaccion,
        v_auditoria_ref, 1, v_huella_auditoria, v_outbox_ref, 1,
        v_huella_evento, v_ahora
    );

    RETURN QUERY SELECT
        v_recibo_ref, 2::numeric, v_huella_recibo, v_contenido_ref,
        v_contenido_version, v_revision, v_estado, v_huella_estado,
        v_vec.registro_ref, v_vec.consumo_ref, v_vec.auditoria_ref,
        v_transaccion_ref, v_auditoria_ref, v_outbox_ref, v_ahora;
END
$funcion$;

CREATE FUNCTION vec_bolsa_reglas_baremo.reconciliar_cambio_atestado_v2(
    p_decision_ref text,
    p_huella_decision_sha256 text,
    p_efecto_ref text,
    p_huella_efecto_sha256 text,
    p_nonce text,
    p_huella_plan_sha256 text
)
RETURNS TABLE (
    recibo_ref text,
    recibo_version numeric,
    huella_recibo_sha256 text,
    recibo_canonico bytea,
    contenido_ref text,
    contenido_version numeric,
    revision numeric,
    estado text,
    huella_estado_sha256 text,
    registro_vec_ref text,
    consumo_vec_ref text,
    auditoria_vec_ref text,
    confirmada_en timestamptz
)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
SET timezone = 'UTC'
SET statement_timeout = '10s'
AS $funcion$
DECLARE
    v_vec record;
    v_huella_nonce text;
BEGIN
    IF current_setting('transaction_isolation') <> 'serializable'
       OR vec_bolsa_reglas_baremo.referencia_valida(
              p_decision_ref
          ) IS NOT TRUE
       OR vec_bolsa_reglas_baremo.huella_sha256_valida(
              p_huella_decision_sha256
          ) IS NOT TRUE
       OR p_efecto_ref IS DISTINCT FROM
          'efecto:reglas-baremo:v2:' || p_huella_plan_sha256
       OR p_huella_efecto_sha256 IS DISTINCT FROM p_huella_plan_sha256
       OR vec_bolsa_reglas_baremo.huella_sha256_valida(
              p_huella_plan_sha256
          ) IS NOT TRUE
       OR (p_nonce ~ '^[0-9a-f]{64}$') IS NOT TRUE THEN
        RETURN;
    END IF;
    SELECT * INTO v_vec
      FROM vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2(
          p_decision_ref, p_huella_decision_sha256, p_efecto_ref,
          p_huella_efecto_sha256, p_nonce
      );
    IF NOT FOUND THEN
        RETURN;
    END IF;
    v_huella_nonce := encode(
        sha256(convert_to(p_nonce, 'UTF8')), 'hex'
    );
    RETURN QUERY
    SELECT recibo.recibo_ref, recibo.recibo_version,
           recibo.huella_recibo_sha256, recibo.recibo_canonico,
           recibo.contenido_ref, recibo.contenido_version,
           recibo.resultado_revision, recibo.resultado_estado,
           recibo.resultado_huella_estado_sha256,
           recibo.registro_vec_ref, recibo.consumo_vec_ref,
           recibo.auditoria_vec_ref, recibo.confirmada_en
      FROM vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2 AS recibo
     WHERE recibo.tenant_id = 'diputacion_granada'
       AND recibo.decision_ref = p_decision_ref
       AND recibo.huella_decision_sha256 = p_huella_decision_sha256
       AND recibo.efecto_ref = p_efecto_ref
       AND recibo.huella_efecto_sha256 = p_huella_efecto_sha256
       AND recibo.huella_plan_sha256 = p_huella_plan_sha256
       AND recibo.huella_nonce_sha256 = v_huella_nonce
       AND recibo.registro_vec_ref = v_vec.registro_ref
       AND recibo.consumo_vec_ref = v_vec.consumo_ref
       AND recibo.auditoria_vec_ref = v_vec.auditoria_ref
       AND recibo.huella_auditoria_vec_sha256 =
           v_vec.huella_auditoria_sha256
       AND recibo.confirmada_en = v_vec.consumida_en;
END
$funcion$;

REVOKE ALL ON TABLE
    vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2
    FROM PUBLIC,
         vec_bolsa_reglas_baremo_ejecutor_gobierno,
         vec_bolsa_reglas_baremo_ejecutor_consulta,
         vec_bolsa_reglas_baremo_publicador_outbox,
         vec_autorizacion_atestada_v2_consumidor;
REVOKE ALL ON FUNCTION
    vec_bolsa_reglas_baremo.confirmar_cambio_atestado_v2(
        bytea, bytea, bytea, bytea, bytea, bytea, jsonb
    ) FROM PUBLIC,
           vec_bolsa_reglas_baremo_ejecutor_gobierno,
           vec_bolsa_reglas_baremo_ejecutor_consulta,
           vec_bolsa_reglas_baremo_publicador_outbox,
           vec_autorizacion_atestada_v2_consumidor;
REVOKE ALL ON FUNCTION
    vec_bolsa_reglas_baremo.reconciliar_cambio_atestado_v2(
        text, text, text, text, text, text
    ) FROM PUBLIC,
           vec_bolsa_reglas_baremo_ejecutor_gobierno,
           vec_bolsa_reglas_baremo_ejecutor_consulta,
           vec_bolsa_reglas_baremo_publicador_outbox,
           vec_autorizacion_atestada_v2_consumidor;

COMMENT ON FUNCTION
    vec_bolsa_reglas_baremo.confirmar_cambio_atestado_v2(
        bytea, bytea, bytea, bytea, bytea, bytea, jsonb
    ) IS
    'Compone VEC-AD-2, CAS, evidencia, historia, auditoria, outbox y recibo en un COMMIT SERIALIZABLE. Sin GRANT runtime hasta cerrar broker, adaptador y verificador.';
COMMENT ON FUNCTION
    vec_bolsa_reglas_baremo.reconciliar_cambio_atestado_v2(
        text, text, text, text, text, text
    ) IS
    'Recupera solo el recibo exacto de un consumo VEC-AD-2 ya confirmado; no reintenta una decision single-use.';

COMMIT;
