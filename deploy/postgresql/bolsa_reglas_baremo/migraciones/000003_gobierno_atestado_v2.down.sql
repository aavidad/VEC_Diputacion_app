BEGIN;
SET LOCAL ROLE vec_bolsa_reglas_baremo_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_reglas_baremo:migracion:000003', 0
    )
);

LOCK TABLE vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2
    IN ACCESS EXCLUSIVE MODE;
DO $proteger_historia$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada V2 rechazada: conservar recibos atestados',
            HINT = 'una retirada irreversible integral requiere una migracion destructiva especifica; este down nunca deja historia parcial';
    END IF;
END
$proteger_historia$;

DROP FUNCTION
    vec_bolsa_reglas_baremo.reconciliar_cambio_atestado_v2(
        text, text, text, text, text, text
    );
DROP FUNCTION
    vec_bolsa_reglas_baremo.confirmar_cambio_atestado_v2(
        bytea, bytea, bytea, bytea, bytea, bytea, jsonb
    );
DROP TABLE vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2;

ALTER TABLE vec_bolsa_reglas_baremo.outbox
    DROP CONSTRAINT outbox_valida;
ALTER TABLE vec_bolsa_reglas_baremo.outbox
    ADD CONSTRAINT outbox_valida CHECK (
        vec_bolsa_reglas_baremo.referencia_valida(outbox_ref)
        AND outbox_version = 1
        AND ruta = 'bolsa.reglas_baremo.estado_confirmado.v1'
        AND esquema_evento =
            'vec.bolsa.reglas-baremo.estado-confirmado.v1'
        AND octet_length(evento_canonico) BETWEEN 2 AND 1048576
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_evento_sha256
        )
        AND encode(sha256(evento_canonico), 'hex') =
            huella_evento_sha256
        AND isfinite(creada_en)
    );

COMMIT;
