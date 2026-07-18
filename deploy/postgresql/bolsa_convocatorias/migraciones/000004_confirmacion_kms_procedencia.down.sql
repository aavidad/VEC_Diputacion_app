BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
DECLARE
    autoriza_destruccion boolean := COALESCE(
        current_setting(
            'vec.confirmar_destruccion_borradores_convocatorias', true
        ) = 'DESTRUIR_HISTORIA_BORRADORES_CONVOCATORIAS_IRREVERSIBLE',
        false
    );
BEGIN
    IF to_regclass(
           'vec_bolsa_convocatorias.cifrado_kms_borrador'
       ) IS NULL
       OR to_regprocedure(
           'vec_bolsa_convocatorias.confirmar_borrador_v1(text,jsonb,bytea)'
       ) IS NULL
       OR to_regprocedure(
           'vec_bolsa_convocatorias.verificar_recibo_borrador_v1(text,text,text)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para revertir confirmacion KMS';
    END IF;
    IF NOT autoriza_destruccion AND (
        EXISTS (
            SELECT 1 FROM
                vec_bolsa_convocatorias.cifrado_kms_borrador
        ) OR EXISTS (
            SELECT 1 FROM
                vec_bolsa_convocatorias.acreditacion_kms_borrador
        ) OR EXISTS (
            SELECT 1 FROM
                vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador
        )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'reversion KMS destruiria historia de borradores';
    END IF;
END
$prevalidacion$;

DROP TRIGGER borrador_version_exige_acreditacion_kms
    ON vec_bolsa_convocatorias.borrador_convocatoria_version;
DROP TRIGGER auditoria_borrador_exige_acreditacion_kms
    ON vec_bolsa_convocatorias.auditoria_borrador;
DROP TRIGGER outbox_borrador_exige_acreditacion_kms
    ON vec_bolsa_convocatorias.outbox_borrador;
DROP TRIGGER diario_borrador_exige_acreditacion_kms
    ON vec_bolsa_convocatorias.diario_borrador_version;

DROP TRIGGER diario_borrador_version_inmutable
    ON vec_bolsa_convocatorias.diario_borrador_version;
CREATE TRIGGER diario_borrador_version_inmutable
    BEFORE UPDATE OR DELETE
    ON vec_bolsa_convocatorias.diario_borrador_version
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_convocatorias.rechazar_mutacion_inmutable();

DROP FUNCTION vec_bolsa_convocatorias.preparar_confirmacion_borrador_v1(
    jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,
    bytea
);
DROP FUNCTION vec_bolsa_convocatorias.confirmar_borrador_v1(
    text,jsonb,bytea
);
DROP FUNCTION vec_bolsa_convocatorias.verificar_recibo_borrador_v1(
    text,text,text
);

DROP TABLE vec_bolsa_convocatorias.acreditacion_kms_borrador;
DROP TABLE vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador;
DROP TABLE vec_bolsa_convocatorias.cifrado_kms_borrador;

DROP FUNCTION
    vec_bolsa_convocatorias.validar_actualizacion_acreditacion_diario_v1();
DROP FUNCTION
    vec_bolsa_convocatorias.identidad_runtime_verificador_recibo_valida();
DROP FUNCTION
    vec_bolsa_convocatorias.validar_consumo_preparacion_kms_v1();
DROP FUNCTION
    vec_bolsa_convocatorias.exigir_cierre_preparacion_kms_v1();
DROP FUNCTION
    vec_bolsa_convocatorias.exigir_acreditacion_kms_durable_v1();
DROP FUNCTION
    vec_bolsa_convocatorias.evidencia_cifrado_kms_borrador_valida(
        jsonb,jsonb,bytea,bytea,bytea,bytea,bytea
    );
DROP FUNCTION
    vec_bolsa_convocatorias.cuerpo_recibo_canonico_borrador_v1(jsonb);
DROP FUNCTION
    vec_bolsa_convocatorias.revalidacion_kms_preimagen_borrador_v1(jsonb);
DROP FUNCTION
    vec_bolsa_convocatorias.atestacion_kms_preimagen_borrador_v1(jsonb);
DROP FUNCTION
    vec_bolsa_convocatorias.firma_base64url_borrador_v1(text);
DROP FUNCTION
    vec_bolsa_convocatorias.acreditacion_kms_canonica_borrador_v1(jsonb);
DROP FUNCTION
    vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(text);
DROP FUNCTION
    vec_bolsa_convocatorias.aad_canonica_borrador_v1(jsonb);
DROP FUNCTION
    vec_bolsa_convocatorias.huella_sobre_aead_borrador_v1(
        jsonb,jsonb,bytea,bytea
    );
DROP FUNCTION
    vec_bolsa_convocatorias.huella_envoltura_clave_borrador_v1(
        jsonb,jsonb,bytea
    );
DROP FUNCTION
    vec_bolsa_convocatorias.firma_evidencia_borrador_valida(jsonb);
DROP FUNCTION
    vec_bolsa_convocatorias.presupuesto_confirmacion_kms_borrador_v1();
DROP FUNCTION
    vec_bolsa_convocatorias.perfil_cifrado_borrador_valido(jsonb);
DROP FUNCTION
    vec_bolsa_convocatorias.procedencia_borrador_valida(jsonb);
DROP FUNCTION
    vec_bolsa_convocatorias.base64url_sin_relleno_valido(text,integer);

CREATE FUNCTION vec_bolsa_convocatorias.confirmar_borrador_v1(
    p_confirmacion jsonb, p_prueba jsonb, p_decision_canonica bytea,
    p_contexto_recurso_canonico bytea, p_material_canonico bytea,
    p_version_canonica bytea, p_sobre_cifrado bytea
)
RETURNS TABLE (
    resultado text, estado_diario text, revision_diario bigint,
    cercado bigint, transaccion_ref text, accion text,
    estado_principal_ref text, estado_principal_revision bigint,
    estado_principal_huella_sha256 text,
    auditoria_ref text, huella_auditoria_sha256 text,
    evento_outbox_ref text, huella_evento_outbox_sha256 text,
    confirmada_en timestamptz, recibo jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'confirmacion de borrador cerrada: contrato KMS no satisfecho';
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_bolsa_convocatorias.confirmar_borrador_v1(
        jsonb,jsonb,bytea,bytea,bytea,bytea,bytea
    ) FROM PUBLIC,
           vec_bolsa_convocatorias_ejecutor_consulta,
           vec_bolsa_convocatorias_proyector_gobierno,
           vec_bolsa_convocatorias_registrador_atestacion,
           vec_bolsa_convocatorias_verificador_recibo;
COMMENT ON FUNCTION
    vec_bolsa_convocatorias.confirmar_borrador_v1(
        jsonb,jsonb,bytea,bytea,bytea,bytea,bytea
    ) IS
    'NO-GO restaurado por la reversion de 000004: aborta hasta reinstalar el contrato KMS/procedencia.';

COMMIT;
