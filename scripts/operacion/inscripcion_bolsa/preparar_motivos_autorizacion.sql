-- Datos sintéticos del catálogo de motivos V2 para la copia de inscripción.
-- psql -X -v ON_ERROR_STOP=1 -1 -f preparar_motivos_autorizacion.sql
-- No es una migración ni modifica funciones, roles o permisos.
SET LOCAL search_path = pg_catalog;
SET LOCAL TimeZone = 'UTC';
SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:inscripcion:motivos:20261013', 0));

DO $motivos$
DECLARE
  catalogo constant text := 'motivos_inscripcion_bolsa_desarrollo';
  version constant integer := 1;
  huella constant text := '37714f8ee837cd7f72a68c1fde2b3a4cfa1d5bcf440cb7f4d79500b8cb9260d1';
  desde constant timestamptz := '2026-10-01T00:00:00Z';
  claves constant text[] := ARRAY[
    'motivo_1791fdf6f06603948eacd5534d2c1517',
    'motivo_add9e62706d8575fa22b5529f3370177',
    'motivo_e99601da335957e930423d711ed03c4c'];
  entradas jsonb;
  secuencia bigint;
  evento text;
  evento_huella text;
  n integer;
BEGIN
  IF pg_catalog.to_regprocedure('vec_autorizacion.publicar_motivos_autorizacion_v2(text,bigint,text,text,integer,text,timestamp with time zone,jsonb)') IS NULL
    OR pg_catalog.to_regclass('vec_autorizacion.motivo_v2_catalogo_publicado') IS NULL THEN
    RAISE EXCEPTION 'Falta la proyección V2 de motivos';
  END IF;
  SELECT count(*) INTO n FROM vec_autorizacion.motivo_v2_catalogo_publicado
    WHERE catalogo_id = catalogo AND catalogo_version = version;
  IF n = 0 THEN
    SELECT ultima_secuencia + 1 INTO STRICT secuencia
      FROM vec_autorizacion.motivo_v2_checkpoint_origen WHERE control_id = true FOR UPDATE;
    entradas := (SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
      'clave', clave, 'vigente_desde', '2026-10-01T00:00:00.000000Z',
      'vigente_hasta', NULL) ORDER BY clave)
      FROM pg_catalog.unnest(claves) AS clave);
    evento_huella := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
      'vec:inscripcion:motivos:20261013:' || secuencia::text, 'UTF8')), 'hex');
    evento := 'evento_' || pg_catalog.substr(evento_huella, 1, 32);
    IF vec_autorizacion.publicar_motivos_autorizacion_v2(
      evento, secuencia, evento_huella, catalogo, version, huella, desde, entradas) IS NOT TRUE THEN
      RAISE EXCEPTION 'Publicación V2 de motivos rechazada';
    END IF;
  END IF;
  IF (SELECT catalogo_huella_publicada_sha256 FROM vec_autorizacion.motivo_v2_catalogo_publicado
      WHERE catalogo_id = catalogo AND catalogo_version = version) IS DISTINCT FROM huella
    OR (SELECT count(*) FROM vec_autorizacion.motivo_v2_entrada
      WHERE catalogo_id = catalogo AND catalogo_version = version
        AND entrada_clave = ANY(claves) AND vigente_desde <= pg_catalog.clock_timestamp()
        AND (vigente_hasta IS NULL OR vigente_hasta > pg_catalog.clock_timestamp())) <> 3
    OR (SELECT count(*) FROM vec_autorizacion.motivo_v2_entrada
      WHERE catalogo_id = catalogo AND catalogo_version = version) <> 3
    OR EXISTS (SELECT 1 FROM vec_autorizacion.motivo_v2_retirada
      WHERE catalogo_id = catalogo AND catalogo_version = version) THEN
    RAISE EXCEPTION 'El catálogo V2 existente no coincide';
  END IF;
END
$motivos$;
