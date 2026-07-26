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
DROP FUNCTION vec_bolsa_importacion_convoca.recuperar_lote_pagina_v1(
    text,integer,integer
);
DROP FUNCTION vec_bolsa_importacion_convoca.consultar_estado_v1(text);
DROP FUNCTION vec_bolsa_importacion_convoca.guardar_lote_v1(
    jsonb,jsonb
);
DROP FUNCTION vec_bolsa_importacion_convoca.publicar_politica_retencion_v1(
    text,bigint,bigint,text
);
DROP FUNCTION vec_bolsa_importacion_convoca.politica_retencion_integra();
DROP FUNCTION vec_bolsa_importacion_convoca.huella_publicacion_retencion(
    text,bigint,bigint,bigint,text,text,timestamp with time zone
);
COMMIT;
