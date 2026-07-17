BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
INSERT INTO vec_autorizacion_atestada_v2.revocacion_clave_capacidad(
    clave_id, version, revocada_en, motivo_catalogado_ref, acto_ref
)
SELECT puntero.clave_id, puntero.version,
       date_trunc('microseconds', clock_timestamp() + interval '2 seconds'),
       'seguridad.prueba.revocacion_futura',
       'acto:revocar:clave:actual:durante-espera'
  FROM vec_autorizacion_atestada_v2.puntero_clave_capacidad AS puntero
 ORDER BY puntero.orden DESC LIMIT 1;
COMMIT;
