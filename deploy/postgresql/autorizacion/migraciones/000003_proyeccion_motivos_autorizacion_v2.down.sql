-- Reversion estructural. Falla cerrado si la proyeccion contiene cualquier
-- publicacion, entrada, retirada o evento de origen. La evidencia se conserva
-- o exporta bajo procedimiento aprobado; este guion nunca la borra ni usa
-- CASCADE.
BEGIN;

SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_autorizacion:migracion:motivos_v2:000003', 0)
);

SET LOCAL ROLE vec_autorizacion_propietario;

-- Todos los flujos runtime toman primero el checkpoint. Mantener el mismo
-- orden evita interbloqueos: un escritor previo termina y se vuelve visible
-- antes del preflight; uno posterior queda bloqueado sin haber mutado ninguna
-- tabla y falla cuando la retirada estructural confirma.
LOCK TABLE vec_autorizacion.motivo_v2_checkpoint_origen
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion.motivo_v2_evento_origen
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion.motivo_v2_catalogo_publicado
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion.motivo_v2_entrada
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion.motivo_v2_retirada
    IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    tiene_evidencia boolean;
BEGIN
    SELECT
        EXISTS (SELECT 1 FROM vec_autorizacion.motivo_v2_evento_origen)
        OR EXISTS (SELECT 1 FROM vec_autorizacion.motivo_v2_catalogo_publicado)
        OR EXISTS (SELECT 1 FROM vec_autorizacion.motivo_v2_entrada)
        OR EXISTS (SELECT 1 FROM vec_autorizacion.motivo_v2_retirada)
        OR EXISTS (
            SELECT 1
              FROM vec_autorizacion.motivo_v2_checkpoint_origen
             WHERE control_id IS DISTINCT FROM true
                OR ultima_secuencia IS DISTINCT FROM 0
                OR ultimo_evento_ref IS NOT NULL
                OR ultima_huella_evento_sha256 IS NOT NULL
        )
      INTO tiene_evidencia;
    IF tiene_evidencia THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000003 no se revierte: existen datos o evidencia de motivos V2';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
    text, integer, text, text, timestamptz
) FROM vec_autorizacion_motivos_evaluador;
REVOKE EXECUTE ON FUNCTION vec_autorizacion.retirar_motivos_autorizacion_v2(
    text, bigint, text, text, integer, text, text, timestamptz
) FROM vec_autorizacion_motivos_proyector;
REVOKE EXECUTE ON FUNCTION vec_autorizacion.publicar_motivos_autorizacion_v2(
    text, bigint, text, text, integer, text, timestamptz, jsonb
) FROM vec_autorizacion_motivos_proyector;
REVOKE USAGE ON SCHEMA vec_autorizacion
    FROM vec_autorizacion_motivos_evaluador,
         vec_autorizacion_motivos_proyector;

DROP FUNCTION vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
    text, integer, text, text
);
DROP FUNCTION vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
    text, integer, text, text, timestamptz
);
DROP FUNCTION vec_autorizacion.retirar_motivos_autorizacion_v2(
    text, bigint, text, text, integer, text, text, timestamptz
);
DROP FUNCTION vec_autorizacion.publicar_motivos_autorizacion_v2(
    text, bigint, text, text, integer, text, timestamptz, jsonb
);

DROP TABLE vec_autorizacion.motivo_v2_retirada;
DROP TABLE vec_autorizacion.motivo_v2_entrada;
DROP TABLE vec_autorizacion.motivo_v2_catalogo_publicado;
DROP TABLE vec_autorizacion.motivo_v2_evento_origen;
DROP TABLE vec_autorizacion.motivo_v2_checkpoint_origen;

DROP FUNCTION vec_autorizacion.motivo_v2_entradas_validas(jsonb);
DROP FUNCTION vec_autorizacion.motivo_v2_instante_canonico_valido(text);
DROP FUNCTION vec_autorizacion.motivo_v2_validar_avance_checkpoint();
DROP FUNCTION vec_autorizacion.motivo_v2_bloquear_mutacion_inmutable();

COMMIT;
