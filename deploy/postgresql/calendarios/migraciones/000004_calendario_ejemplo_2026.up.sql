\set ON_ERROR_STOP on
-- Calendarios 000004: calendario laboral 2026 DE EJEMPLO para que los plazos
-- en días hábiles funcionen con fiestas locales verosímiles (duda 43 de RRHH).
-- Publica versiones sucesoras, sin tocar la historia:
--  * fiestas locales de Granada capital (municipio:ine:18087): 2 de enero y
--    4 de junio de 2026;
--  * fiestas locales de Huéscar (municipio:ine:18098), municipio real de la
--    Residencia «Rodríguez Penalva»: 25 de mayo y 22 de octubre de 2026;
--  * centro-752 pasa a Huéscar y recibe el 24 y el 31 de diciembre como no
--    laborables de la Diputación, como ya tenían los demás centros.
-- Las fechas locales proceden de la Resolución de 6 de octubre de 2025 de la
-- Dirección General de Trabajo, Seguridad y Salud Laboral (BOJA de 14 de
-- octubre de 2025), cotejadas con una reproducción y no con el BOJA original;
-- los no laborables de la Diputación son un supuesto sin acuerdo. Por eso
-- todo consta como sintético y con la marca paquete:ejemplo:vec:v1, que es la
-- de los demás datos de ejemplo. Se retira con 000004.down.sql, que publica
-- versiones sucesoras con el contenido anterior. Las fiestas nacionales y de
-- Andalucía de 000002 ya reproducen BOE-A-2025-21667 y el Decreto 101/2025.
BEGIN;
SET LOCAL ROLE vec_calendarios_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_calendarios:migracion:000004:ejemplo-2026:v1',0));

-- Una aplicación repetida sin retirada intermedia se rechaza: el último
-- estado de algún ámbito ya lleva la marca de ejemplo.
DO $pre$ BEGIN
 IF current_user<>'vec_calendarios_propietario' OR to_regclass('vec_calendarios.version_calendario') IS NULL
    OR NOT EXISTS (SELECT 1 FROM vec_calendarios.version_calendario WHERE id='calendario:centro:centro-752:2026:v1')
    OR NOT EXISTS (SELECT 1 FROM vec_calendarios.version_calendario WHERE id='calendario:local:municipio:ine:18087:2026:v1')
    OR EXISTS (
      SELECT 1 FROM (SELECT DISTINCT ON (v.ambito_tipo, v.ambito_ref) v.procedencia_referencia
                       FROM vec_calendarios.version_calendario v
                      WHERE v.anio=2026 AND (v.ambito_tipo, v.ambito_ref) IN
                            (('local','municipio:ine:18087'),('local','municipio:ine:18098'),('centro','centro-752'))
                      ORDER BY v.ambito_tipo, v.ambito_ref, v.numero DESC) ultima
       WHERE ultima.procedencia_referencia LIKE 'paquete:ejemplo:vec:v1%')
 THEN RAISE EXCEPTION 'Calendarios 000004: preimagen incompatible o ya aplicada' USING ERRCODE='55000'; END IF;
END $pre$;

DO $carga$
DECLARE
  norma_local constant text := 'Resolución de 6 de octubre de 2025, de la Dirección General de Trabajo, Seguridad y Salud Laboral, fiestas locales de Andalucía para 2026 (BOJA de 14 de octubre de 2025); carga de ejemplo cotejada con una reproducción, pendiente de validar por RRHH';
  marca constant text := 'paquete:ejemplo:vec:v1 vec:ejemplo:calendarios:2026';
  previa vec_calendarios.version_calendario%ROWTYPE;
  nuevo_id text;
  nuevo_numero integer;
  ambito record;
  dia record;
BEGIN
  FOR ambito IN
    SELECT * FROM (VALUES
      ('local', 'municipio:ine:18087', 'Fiestas locales de Granada 2026 (ejemplo)', norma_local, date '2025-10-14', NULL::text),
      ('local', 'municipio:ine:18098', 'Fiestas locales de Huéscar 2026 (ejemplo)', norma_local, date '2025-10-14', NULL::text),
      ('centro', 'centro-752', 'Residencia «Rodríguez Penalva» (ejemplo)',
       'Supuesto de trabajo: 24 y 31 de diciembre no laborables de la Diputación y centro situado en Huéscar; sin acuerdo aprobado',
       date '2026-09-25', 'municipio:ine:18098')
    ) AS a(tipo, ref, denominacion, norma, publicada_en, municipio)
  LOOP
    SELECT * INTO previa FROM vec_calendarios.version_calendario v
     WHERE v.anio=2026 AND v.ambito_tipo=ambito.tipo AND v.ambito_ref=ambito.ref
     ORDER BY v.numero DESC LIMIT 1;
    nuevo_numero := COALESCE(previa.numero, 0) + 1;
    nuevo_id := 'calendario:' || ambito.tipo || ':' || ambito.ref || ':2026:v' || nuevo_numero;
    INSERT INTO vec_calendarios.version_calendario
      (id, ambito_tipo, ambito_ref, anio, numero, sustituye_id, denominacion, procedencia_norma, procedencia_referencia,
       procedencia_publicada_en, sintetica, comunidad_ref, municipio_ref)
    VALUES (nuevo_id, ambito.tipo, ambito.ref, 2026, nuevo_numero, previa.id, ambito.denominacion, ambito.norma, marca,
            ambito.publicada_en, true, 'es-an', ambito.municipio);
    FOR dia IN
      SELECT * FROM (VALUES
        ('municipio:ine:18087', date '2026-01-02', 'festivo', 'Día de la Toma de Granada (ejemplo)'),
        ('municipio:ine:18087', date '2026-06-04', 'festivo', 'Jueves de Corpus Christi (ejemplo)'),
        ('municipio:ine:18098', date '2026-05-25', 'festivo', 'Fiesta local de Huéscar (ejemplo)'),
        ('municipio:ine:18098', date '2026-10-22', 'festivo', 'Fiesta local de Huéscar (ejemplo)'),
        ('centro-752', date '2026-12-24', 'no_laborable', 'No laborable de la Diputación (ejemplo)'),
        ('centro-752', date '2026-12-31', 'no_laborable', 'No laborable de la Diputación (ejemplo)')
      ) AS d(ref, fecha, efecto, denominacion) WHERE d.ref=ambito.ref
    LOOP
      INSERT INTO vec_calendarios.dia_calendario (version_id, fecha, efecto, denominacion)
      VALUES (nuevo_id, dia.fecha, dia.efecto, dia.denominacion);
    END LOOP;
  END LOOP;
END $carga$;

DO $post$ BEGIN
 IF (SELECT count(*) FROM (SELECT DISTINCT ON (v.ambito_tipo, v.ambito_ref) v.id, v.procedencia_referencia
                             FROM vec_calendarios.version_calendario v
                            WHERE v.anio=2026 AND (v.ambito_tipo, v.ambito_ref) IN
                                  (('local','municipio:ine:18087'),('local','municipio:ine:18098'),('centro','centro-752'))
                            ORDER BY v.ambito_tipo, v.ambito_ref, v.numero DESC) u
      WHERE u.procedencia_referencia LIKE 'paquete:ejemplo:vec:v1%'
        AND (SELECT count(*) FROM vec_calendarios.dia_calendario d WHERE d.version_id=u.id)=2)<>3
 THEN RAISE EXCEPTION 'Calendarios 000004: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
