\set ON_ERROR_STOP on
-- Retirada de Calendarios 000004 sin borrar historia: publica, para cada
-- ámbito cuyo último estado lleva la marca paquete:ejemplo:vec:v1, una
-- versión sucesora con el contenido de la versión a la que sustituyó (o
-- vacía si el ejemplo abrió el ámbito). Si no hay nada que retirar falla con
-- 55000 y no toca nada. Tras la retirada puede volver a aplicarse la subida.
BEGIN;
SET LOCAL ROLE vec_calendarios_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_calendarios:migracion:000004:ejemplo-2026:v1',0));

DO $retirada$
DECLARE
  ultima vec_calendarios.version_calendario%ROWTYPE;
  anterior vec_calendarios.version_calendario%ROWTYPE;
  nuevo_id text;
  ambito record;
  retiradas integer := 0;
BEGIN
  IF current_user<>'vec_calendarios_propietario' OR to_regclass('vec_calendarios.version_calendario') IS NULL THEN
    RAISE EXCEPTION 'Calendarios 000004: preimagen incompatible' USING ERRCODE='55000';
  END IF;
  FOR ambito IN
    SELECT * FROM (VALUES ('local','municipio:ine:18087'), ('local','municipio:ine:18098'), ('centro','centro-752')) AS a(tipo, ref)
  LOOP
    SELECT * INTO ultima FROM vec_calendarios.version_calendario v
     WHERE v.anio=2026 AND v.ambito_tipo=ambito.tipo AND v.ambito_ref=ambito.ref
     ORDER BY v.numero DESC LIMIT 1;
    IF ultima.id IS NULL OR ultima.procedencia_referencia NOT LIKE 'paquete:ejemplo:vec:v1%' THEN
      RAISE EXCEPTION 'Calendarios 000004: no hay calendario de ejemplo que retirar' USING ERRCODE='55000';
    END IF;
    SELECT * INTO anterior FROM vec_calendarios.version_calendario v WHERE v.id=ultima.sustituye_id;
    nuevo_id := 'calendario:' || ambito.tipo || ':' || ambito.ref || ':2026:v' || (ultima.numero + 1);
    INSERT INTO vec_calendarios.version_calendario
      (id, ambito_tipo, ambito_ref, anio, numero, sustituye_id, denominacion, procedencia_norma, procedencia_referencia,
       procedencia_publicada_en, sintetica, comunidad_ref, municipio_ref)
    VALUES (nuevo_id, ambito.tipo, ambito.ref, 2026, ultima.numero + 1, ultima.id,
            COALESCE(anterior.denominacion, 'Fiestas locales retiradas (sin calendario)'),
            COALESCE(anterior.procedencia_norma, 'Retirada del calendario de ejemplo; sin fiestas locales cargadas'),
            COALESCE(anterior.procedencia_referencia, 'vec:retirada:ejemplo:calendarios:2026'),
            COALESCE(anterior.procedencia_publicada_en, date '2026-09-25'),
            COALESCE(anterior.sintetica, true), ultima.comunidad_ref,
            CASE WHEN ambito.tipo='centro' THEN COALESCE(anterior.municipio_ref, ultima.municipio_ref) END);
    IF anterior.id IS NOT NULL THEN
      INSERT INTO vec_calendarios.dia_calendario (version_id, fecha, efecto, denominacion)
      SELECT nuevo_id, d.fecha, d.efecto, d.denominacion FROM vec_calendarios.dia_calendario d WHERE d.version_id=anterior.id;
    END IF;
    retiradas := retiradas + 1;
  END LOOP;
  IF retiradas<>3 THEN
    RAISE EXCEPTION 'Calendarios 000004: retirada incompleta' USING ERRCODE='55000';
  END IF;
END $retirada$;
COMMIT;
