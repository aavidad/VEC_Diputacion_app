-- Reversión probatoria: nunca destruye una publicación, retirada, evento,
-- entrada ni checkpoint avanzado. La evidencia debe exportarse y autorizarse
-- externamente; no se usa CASCADE.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:autorizacion:motivo-cobertura-v1:000002', 0
    )
);

LOCK TABLE vec_autorizacion.motivo_cobertura_v1_checkpoint_origen
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion.motivo_cobertura_v1_evento_origen
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion.motivo_cobertura_v1_catalogo_publicado
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion.motivo_cobertura_v1_entrada
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion.motivo_cobertura_v1_retirada
    IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion.resolver_motivo_cobertura_historico_v1(text,integer,text,text,text,timestamp with time zone)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.resolver_motivo_cobertura_actual_v1(text,integer,text,text,text)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '2BP01',
            MESSAGE = '000002 no se revierte antes de retirar la barrera 000003';
    END IF;
    IF EXISTS (
        SELECT 1 FROM vec_autorizacion.motivo_cobertura_v1_evento_origen
    ) OR EXISTS (
        SELECT 1 FROM vec_autorizacion.motivo_cobertura_v1_catalogo_publicado
    ) OR EXISTS (
        SELECT 1 FROM vec_autorizacion.motivo_cobertura_v1_entrada
    ) OR EXISTS (
        SELECT 1 FROM vec_autorizacion.motivo_cobertura_v1_retirada
    ) OR EXISTS (
        SELECT 1
          FROM vec_autorizacion.motivo_cobertura_v1_checkpoint_origen
         WHERE control_id IS DISTINCT FROM true
            OR ultima_secuencia IS DISTINCT FROM 0
            OR ultimo_evento_ref IS NOT NULL
            OR ultima_huella_evento_sha256 IS NOT NULL
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000002 no se revierte: existen datos o evidencia de cobertura';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION vec_autorizacion.retirar_motivos_cobertura_v1(
    text, bigint, text, text, integer, text, text, text, timestamptz
) FROM vec_autorizacion_motivos_proyector;
REVOKE EXECUTE ON FUNCTION vec_autorizacion.publicar_motivos_cobertura_v1(
    text, bigint, text, text, integer, text, text, timestamptz, jsonb
) FROM vec_autorizacion_motivos_proyector;

DROP FUNCTION vec_autorizacion.retirar_motivos_cobertura_v1(
    text, bigint, text, text, integer, text, text, text, timestamptz
) RESTRICT;
DROP FUNCTION vec_autorizacion.publicar_motivos_cobertura_v1(
    text, bigint, text, text, integer, text, text, timestamptz, jsonb
) RESTRICT;

DROP TABLE vec_autorizacion.motivo_cobertura_v1_retirada RESTRICT;
DROP TABLE vec_autorizacion.motivo_cobertura_v1_entrada RESTRICT;
DROP TABLE vec_autorizacion.motivo_cobertura_v1_catalogo_publicado RESTRICT;
DROP TABLE vec_autorizacion.motivo_cobertura_v1_evento_origen RESTRICT;
DROP TABLE vec_autorizacion.motivo_cobertura_v1_checkpoint_origen RESTRICT;

DROP FUNCTION vec_autorizacion.motivo_cobertura_v1_entradas_validas(
    jsonb
) RESTRICT;
DROP FUNCTION vec_autorizacion.motivo_cobertura_v1_instante_canonico(
    text
) RESTRICT;
DROP FUNCTION vec_autorizacion.motivo_cobertura_v1_validar_checkpoint()
    RESTRICT;
DROP FUNCTION vec_autorizacion.motivo_cobertura_v1_bloquear_inmutable()
    RESTRICT;

-- USAGE puede estar justificado por otras fronteras VEC y no se revoca aquí.
COMMIT;
