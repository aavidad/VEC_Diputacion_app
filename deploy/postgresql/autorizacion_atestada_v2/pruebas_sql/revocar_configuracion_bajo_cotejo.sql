BEGIN;
SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '500ms';
SET LOCAL timezone = 'UTC';
INSERT INTO vec_confianza_atestacion_v2.revocacion_configuracion(
    revision, revocada_en, motivo_catalogado_ref, acto_ref
)
SELECT revision, clock_timestamp() + interval '1 second',
       'motivo:prueba:revocacion:configuracion',
       'acto:prueba:revocar:configuracion:atestada'
  FROM vec_confianza_atestacion_v2.puntero_configuracion_actual
 ORDER BY orden DESC LIMIT 1;
ROLLBACK;
