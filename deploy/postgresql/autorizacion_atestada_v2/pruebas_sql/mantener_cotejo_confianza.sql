BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
DO $cotejar$
DECLARE
    configuracion record;
    raiz record;
    ahora timestamptz(6) := date_trunc('microseconds', clock_timestamp());
BEGIN
    SELECT version.revision, version.huella_configuracion_sha256,
           version.publicada_en, version.expira_en
      INTO STRICT configuracion
      FROM vec_confianza_atestacion_v2.configuracion_confianza_version AS version
      JOIN vec_confianza_atestacion_v2.puntero_configuracion_actual AS puntero
        ON puntero.revision = version.revision
     ORDER BY puntero.orden DESC LIMIT 1;
    SELECT version.clave_id, version.version,
           version.huella_clave_spki_sha256, version.valida_desde,
           version.valida_hasta, version.suite, version.audiencia_despliegue
      INTO STRICT raiz
      FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
      JOIN vec_confianza_atestacion_v2.raiz_confianza_version AS version
        ON version.clave_id = enlace.clave_id
       AND version.version = enlace.version
     WHERE enlace.configuracion_revision = configuracion.revision
     ORDER BY version.clave_id COLLATE "C" LIMIT 1;
    IF vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(
           configuracion.revision, configuracion.huella_configuracion_sha256,
           configuracion.publicada_en, configuracion.expira_en,
           raiz.clave_id, raiz.version, raiz.huella_clave_spki_sha256,
           raiz.valida_desde, raiz.valida_hasta, raiz.suite,
           raiz.audiencia_despliegue, ahora
       ) IS NOT TRUE THEN
        RAISE EXCEPTION 'el cotejo valido fue rechazado';
    END IF;
END
$cotejar$;
-- El advisory lock compartido adquirido por el cotejo sigue vivo aquí.
SELECT pg_sleep(5);
ROLLBACK;
