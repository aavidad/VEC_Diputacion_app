BEGIN;
SET LOCAL ROLE vec_bolsa_importacion_convoca_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';
SET LOCAL idle_in_transaction_session_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_importacion_convoca:migraciones', 0
    )
);
DROP FUNCTION vec_bolsa_importacion_convoca.expurgar_staging_vencido_v1(
    text,text,text,bigint,integer
);
DROP FUNCTION vec_bolsa_importacion_convoca.cambiar_bloqueo_retencion_v1(
    text,text,text,text,boolean
);
DROP FUNCTION vec_bolsa_importacion_convoca.conciliar_v1(
    text,text,text,text,text,text
);
DROP FUNCTION vec_bolsa_importacion_convoca.anexar_evento_estado(
    text,text,text,text,text,text,boolean,timestamp with time zone
);
COMMIT;
