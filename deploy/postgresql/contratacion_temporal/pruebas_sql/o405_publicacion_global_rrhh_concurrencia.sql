\set ON_ERROR_STOP on

SELECT vec_o405_publicacion_prueba.crear_alta(
    :'expediente_ref', :'numero_visible'
);

BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '15s';

WITH datos AS (
    SELECT vec_o405_publicacion_prueba.agregado(
        :'expediente_ref', :'numero_visible', 1, 'base'
    ) AS agregado
), prueba AS (
    SELECT agregado, pg_catalog.convert_to(
        pg_catalog.repeat(
            'prueba:concurrencia:' || :'marca' || ':', 8
        ),
        'UTF8'
    ) AS bytes
    FROM datos
)
INSERT INTO vec_contratacion_temporal.expediente_version_integral
SELECT :'expediente_ref', 1, agregado,
       pg_catalog.encode(pg_catalog.sha256(
           pg_catalog.convert_to(agregado::text, 'UTF8')
       ), 'hex'),
       bytes,
       pg_catalog.encode(pg_catalog.sha256(bytes), 'hex'),
       'flujo:contratacion:publicacion', 3,
       pg_catalog.repeat('a', 64),
       'analisis_rrhh', 'en_curso', 'analisis_o3',
       'operacion:concurrencia:' || :'marca',
       '2026-01-03T00:00:00Z'::timestamptz
FROM prueba;

SELECT pg_catalog.pg_sleep(:'pausa'::numeric);
COMMIT;
