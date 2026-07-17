BEGIN;
SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '500ms';
SET LOCAL timezone = 'UTC';
INSERT INTO vec_confianza_atestacion_v2.revocacion_raiz(
    clave_id, version, revocada_en, motivo_catalogado_ref, acto_ref
)
SELECT version.clave_id, version.version, clock_timestamp(),
       'motivo:prueba:revocacion:raiz',
       'acto:prueba:revocar:raiz:atestada'
  FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
  JOIN vec_confianza_atestacion_v2.raiz_confianza_version AS version
    ON version.clave_id = enlace.clave_id AND version.version = enlace.version
  JOIN vec_confianza_atestacion_v2.puntero_configuracion_actual AS puntero
    ON puntero.revision = enlace.configuracion_revision
 ORDER BY puntero.orden DESC, version.clave_id COLLATE "C" LIMIT 1;
ROLLBACK;
